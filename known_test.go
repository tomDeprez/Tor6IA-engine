package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestKnownTensors(t *testing.T) {
	// A tensor message as llama.cpp sends it: order, size, description with the name in place, bytes.
	tensor := func(name string, data []byte) []byte {
		desc := make([]byte, tensorHeader)
		copy(desc[tensorNameAt:], name)
		return append(order(rpcSetTensor, uint64(len(desc)+len(data)), desc), data...)
	}
	fill := func(n int, seed byte) []byte {
		b := make([]byte, n)
		for i := range b {
			b[i] = seed + byte(i*7)
		}
		return b
	}
	onDisk, notOnDisk, tiny := fill(300<<10, 1), fill(200<<10, 2), fill(100, 3)

	// The holder has one of the tensors, in a file named by its hash.
	dir := t.TempDir()
	name := make([]byte, 8)
	binary.BigEndian.PutUint64(name, fnvOf(onDisk))
	if err := os.WriteFile(filepath.Join(dir, hex.EncodeToString(name)), onDisk, 0o644); err != nil {
		t.Fatal(err)
	}
	var said bytes.Buffer
	sendKnown(&said, tensorsIn(dir))
	have, err := readKnown(&said)
	if err != nil || len(have) != 1 || !have[fnvOf(onDisk)] {
		t.Fatalf("what the holder has: %v %v", have, err)
	}

	// What the head's llama.cpp sends, in order.
	var sent bytes.Buffer
	sent.Write(order(rpcHello, 4, []byte{1, 2, 3, 4}))
	sent.Write(tensor("blk.0.attn_q.weight", onDisk))
	sent.Write(order(rpcGraphCompute, 6, []byte("abcdef")))
	sent.Write(tensor("blk.0.attn_k.weight", notOnDisk))
	sent.Write(tensor("blk.0.norm.weight", tiny))
	sent.Write(tensor("blk.1.attn_q.weight", onDisk))
	want := bytes.Clone(sent.Bytes())

	var link, got bytes.Buffer
	headToHolder(&link, &sent, have)
	// The tensor the holder has went by twice, and its bytes not once.
	if link.Len() > len(want)-2*len(onDisk)+1024 {
		t.Fatalf("%d bytes on the link for %d sent by llama.cpp, with %d known twice", link.Len(), len(want), len(onDisk))
	}
	holderFromHead(&got, &link, dir)
	if !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("the block server received %d bytes, llama.cpp sent %d: not the same stream", got.Len(), len(want))
	}

	// A stream that is not llama.cpp's is carried as it is.
	strange := append([]byte{200, 1, 2, 3, 4, 5, 6, 7, 8}, fill(5000, 9)...)
	link.Reset()
	got.Reset()
	headToHolder(&link, bytes.NewReader(strange), have)
	holderFromHead(&got, &link, dir)
	if !bytes.Equal(got.Bytes(), strange) {
		t.Fatal("an unknown stream was changed on the way")
	}
	// A description with no name where it should be is not cut: the tensor goes whole.
	odd := order(rpcSetTensor, uint64(tensorHeader+len(onDisk)), append(make([]byte, tensorHeader), onDisk...))
	link.Reset()
	got.Reset()
	headToHolder(&link, bytes.NewReader(odd), have)
	if link.Len() < len(onDisk) {
		t.Fatal("a tensor with an unknown description was replaced by its hash")
	}
	holderFromHead(&got, &link, dir)
	if !bytes.Equal(got.Bytes(), odd) {
		t.Fatal("a tensor with an unknown description was changed on the way")
	}
}
