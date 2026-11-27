package main

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// header builds a small GGUF header by hand: two settings, one of them a list of strings like the
// token list of a real model, and two tensors.
func header() []byte {
	var b bytes.Buffer
	le := binary.LittleEndian
	str := func(s string) { binary.Write(&b, le, uint64(len(s))); b.WriteString(s) }
	b.WriteString("GGUF")
	binary.Write(&b, le, uint32(3)) // version
	binary.Write(&b, le, uint64(2)) // tensors
	binary.Write(&b, le, uint64(2)) // settings

	str("general.alignment")
	binary.Write(&b, le, uint32(4)) // a 32-bit number
	binary.Write(&b, le, uint32(64))
	str("tokenizer.tokens")
	binary.Write(&b, le, uint32(9)) // an array
	binary.Write(&b, le, uint32(8)) // of strings
	binary.Write(&b, le, uint64(2))
	str("hello")
	str("world")

	str("blk.3.attn.weight")
	binary.Write(&b, le, uint32(2))
	binary.Write(&b, le, uint64(256))
	binary.Write(&b, le, uint64(4))
	binary.Write(&b, le, uint32(12)) // Q4_K: 144 bytes for 256 values
	binary.Write(&b, le, uint64(0))
	str("output.weight")
	binary.Write(&b, le, uint32(1))
	binary.Write(&b, le, uint64(10))
	binary.Write(&b, le, uint32(0)) // 32-bit floats
	binary.Write(&b, le, uint64(576))
	return b.Bytes()
}

func TestReadPlan(t *testing.T) {
	raw := header()
	p, err := readPlan(bytes.NewReader(raw), 5000)
	if err != nil {
		t.Fatal(err)
	}
	data := (int64(len(raw)) + 63) / 64 * 64 // tensors start at the next multiple of the alignment
	want := []Tensor{
		{Name: "blk.3.attn.weight", Layer: 3, Offset: data, Size: 4 * 144, Type: 12},
		{Name: "output.weight", Layer: -1, Offset: data + 576, Size: 40, Type: 0},
	}
	if p.Layers != 4 || p.HeaderSize != int64(len(raw)) || len(p.Tensors) != 2 || p.Tensors[0] != want[0] || p.Tensors[1] != want[1] {
		t.Fatalf("got %+v", p)
	}
	if p.LayerSize(3) != 576 || p.LayerSize(-1) != 40 || p.LayerSize(0) != 0 {
		t.Fatalf("layer sizes: %d %d %d", p.LayerSize(3), p.LayerSize(-1), p.LayerSize(0))
	}
	if _, err := readPlan(bytes.NewReader([]byte("NOPE....")), 8); err == nil {
		t.Fatal("a file that is not GGUF was accepted")
	}
	if _, err := readPlan(bytes.NewReader(raw[:40]), 5000); err == nil {
		t.Fatal("a cut header was accepted")
	}
}

func TestParseLayers(t *testing.T) {
	for _, c := range []struct {
		text        string
		first, last int
		ok          bool
	}{
		{"all", 0, 47, true}, {"3-5", 3, 5, true}, {"7", 7, 7, true}, {"0-47", 0, 47, true},
		{"5-3", 0, 0, false}, {"40-48", 0, 0, false}, {"x", 0, 0, false}, {"-1", 0, 0, false},
	} {
		first, last, err := parseLayers(c.text, 48)
		if (err == nil) != c.ok || (c.ok && (first != c.first || last != c.last)) {
			t.Errorf("parseLayers(%q) = %d, %d, %v", c.text, first, last, err)
		}
	}
}
