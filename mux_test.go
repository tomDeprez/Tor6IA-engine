package main

import (
	"bytes"
	"crypto/rand"
	"io"
	"net"
	"sync"
	"testing"
)

// Several connections cross one pair of pipes at the same time, each keeping its own bytes in order.
func TestBridgeCarriesManyConnections(t *testing.T) {
	// The "block server": echoes what it receives.
	target, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	go func() {
		for {
			c, err := target.Accept()
			if err != nil {
				return
			}
			go func() { io.Copy(c, c); c.Close() }()
		}
	}()

	toInside, fromOutside := io.Pipe()
	toOutside, fromInside := io.Pipe()
	outside, inside := newMux(fromOutside), newMux(fromInside)
	go inside.run(toInside, func() (net.Conn, error) { return net.Dial("tcp", target.Addr().String()) })
	go outside.run(toOutside, nil)

	front, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer front.Close()
	go outside.serveOutside(front)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, err := net.Dial("tcp", front.Addr().String())
			if err != nil {
				t.Error(err)
				return
			}
			defer c.Close()
			sent := make([]byte, 300<<10) // larger than one frame
			rand.Read(sent)
			go c.Write(sent)
			got := make([]byte, len(sent))
			if _, err := io.ReadFull(c, got); err != nil || !bytes.Equal(sent, got) {
				t.Errorf("connection came back different: %v", err)
			}
		}()
	}
	wg.Wait()
}
