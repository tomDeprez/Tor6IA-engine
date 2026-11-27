package main

import (
	"bytes"
	"strings"
	"testing"
)

// Lines copied from a real llama-server load, shortened. The first two layer lines are the
// trial placement llama.cpp prints before the real one.
const loadLog = `0.00.320.731 I llama_prepare_model_devices: using device RPC0 (127.0.0.1:50052) (unknown id) - 18650 MiB free
0.00.322.145 I llama_prepare_model_devices: using device RPC1 (192.168.1.20:50052) (unknown id) - 9000 MiB free
0.00.500.001 D load_tensors: layer   0 assigned to device RPC0, is_swa = 1
0.00.500.002 D load_tensors: layer   1 assigned to device RPC1, is_swa = 1
0.00.670.821 D load_tensors: layer   0 assigned to device RPC0, is_swa = 1
0.00.670.828 D load_tensors: layer   1 assigned to device RPC0, is_swa = 1
0.00.670.829 D load_tensors: layer   2 assigned to device RPC0, is_swa = 0
0.00.670.834 D load_tensors: layer   3 assigned to device RPC1, is_swa = 1
0.00.670.835 D load_tensors: layer   4 assigned to device CPU, is_swa = 1
0.00.670.836 D load_tensors: layer   5 assigned to device CPU, is_swa = 1
`

func TestParseBlocks(t *testing.T) {
	got := parseBlocks(loadLog)
	want := []Block{
		{Machine: "127.0.0.1:50052", First: 0, Last: 2, FreeMiB: 18650},
		{Machine: "192.168.1.20:50052", First: 3, Last: 3, FreeMiB: 9000},
		{Machine: "this machine", First: 4, Last: 5},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d blocks, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("block %d: got %+v, want %+v", i+1, got[i], want[i])
		}
	}
	var table bytes.Buffer
	printBlocks(&table, got)
	if !strings.Contains(table.String(), "0 to 2") || !strings.Contains(table.String(), "50 %") {
		t.Errorf("table does not show the first block as layers 0 to 2, half of the model:\n%s", table.String())
	}
	if gone := absent(got, []string{"127.0.0.1:50052", "10.0.0.9:50052"}); len(gone) != 1 || gone[0] != "10.0.0.9:50052" {
		t.Errorf("absent machines: got %v, want the one that holds no layer", gone)
	}
	if len(parseBlocks("nothing useful")) != 0 {
		t.Error("blocks found in a log that has none")
	}
}

func TestArchiveEntriesStayInside(t *testing.T) {
	root := t.TempDir()
	for _, bad := range []string{"../evil", "a/../../evil"} {
		if _, err := inside(root, bad); err == nil {
			t.Errorf("entry %q accepted", bad)
		}
	}
	if _, err := inside(root, "llama-b1/llama-server"); err != nil {
		t.Errorf("normal entry refused: %v", err)
	}
}
