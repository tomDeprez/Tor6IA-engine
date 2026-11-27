package main

import "testing"

// plan with n layers of the given size each.
func evenPlan(layers int, size int64) *Plan {
	p := &Plan{Layers: layers}
	for i := 0; i < layers; i++ {
		p.Tensors = append(p.Tensors, Tensor{Layer: i, Size: size})
	}
	return p
}

func TestBlocksCoverEveryLayerOnce(t *testing.T) {
	for _, c := range []struct {
		layers int
		size   int64
		want   int
	}{
		{48, 136 << 20, 6},  // Gemma 4 12B: 6.4 GB of layers
		{60, 290 << 20, 17}, // Gemma 4 31B: 17 GB
		{35, 20 << 20, 1},   // a small model is one block
		{3, 2 << 30, 3},     // never more blocks than layers
	} {
		blocks := evenPlan(c.layers, c.size).Blocks()
		if len(blocks) != c.want {
			t.Errorf("%d layers of %d MB: %d blocks, want %d", c.layers, c.size>>20, len(blocks), c.want)
		}
		next := 0
		var total int64
		for _, b := range blocks {
			if b.First != next || b.Last < b.First {
				t.Fatalf("%d layers: blocks do not follow each other: %+v", c.layers, blocks)
			}
			next = b.Last + 1
			total += b.Size
		}
		if next != c.layers || total != int64(c.layers)*c.size {
			t.Errorf("%d layers: blocks cover %d layers and %d bytes", c.layers, next, total)
		}
	}
}

func TestAdvise(t *testing.T) {
	blocks := evenPlan(60, 290<<20).Blocks() // 17 blocks of about 1 GB
	for _, c := range []struct {
		freeGB int64
		want   int
	}{{1, 0}, {4, 1}, {8, 4}, {16, 10}, {64, 17}} {
		if got := advise(blocks, c.freeGB<<30); got != c.want {
			t.Errorf("%d GB free: advised %d blocks, want %d", c.freeGB, got, c.want)
		}
	}
}

func TestHoldersMustCoverTheModel(t *testing.T) {
	blocks := evenPlan(48, 136<<20).Blocks() // 6 blocks of 8 layers
	hs, err := parseHolders("b:1=4-6, a:1=1-3", len(blocks))
	if err != nil || len(hs) != 2 || hs[0].Addr != "a:1" {
		t.Fatalf("holders not read in block order: %+v %v", hs, err)
	}
	if err := cover(hs, 6); err != nil {
		t.Errorf("a full cover was refused: %v", err)
	}
	// 24 layers each, and the step after the last layer goes to the last machine.
	if got := sharesFor(hs, blocks); got != "24,25" {
		t.Errorf("shares %q, want 24,25", got)
	}
	for text, problem := range map[string]string{
		"a:1=1-2,b:1=4-6": "blocks 3 to 3 of 6 are held by no machine",
		"a:1=1-3,b:1=3-6": "block 3 of 6 is announced by two machines",
		"a:1=1-4":         "blocks 5 to 6 of 6 are held by no machine",
		"a:1=2-6":         "blocks 1 to 1 of 6 are held by no machine",
	} {
		hs, err := parseHolders(text, 6)
		if err != nil {
			t.Fatal(err)
		}
		if err := cover(hs, 6); err == nil || err.Error() != problem {
			t.Errorf("%s: got %v, want %q", text, err, problem)
		}
	}
	if _, err := parseHolders("a:1=5-9", 6); err == nil {
		t.Error("blocks beyond the model were accepted")
	}
}
