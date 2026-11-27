package main

import (
	"bufio"
	"io"
	"net"
	"testing"
	"time"
)

func TestLink(t *testing.T) {
	// The "block server": it echoes, and counts who reaches it.
	inner, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer inner.Close()
	reached := make(chan struct{}, 8)
	go func() {
		for {
			c, err := inner.Accept()
			if err != nil {
				return
			}
			reached <- struct{}{}
			go io.Copy(c, c)
		}
	}()
	secret := []byte("0123456789abcdef")
	front, err := serveLink("127.0.0.1:0", secret, func() string { return inner.Addr().String() })
	if err != nil {
		t.Fatal(err)
	}
	defer front.Close()

	// The right secret: what goes in comes back.
	local, err := dialLink(front.Addr().String(), secret)
	if err != nil {
		t.Fatal(err)
	}
	defer local.Close()
	c, err := net.Dial("tcp", local.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.Write([]byte("hello\n"))
	c.SetReadDeadline(time.Now().Add(5 * time.Second))
	if line, err := bufio.NewReader(c).ReadString('\n'); err != nil || line != "hello\n" {
		t.Fatalf("through the link: %q, %v", line, err)
	}
	for len(reached) > 0 {
		<-reached
	}

	// A wrong secret is refused, and a stranger who speaks without TLS gets nothing.
	if _, err := dialLink(front.Addr().String(), []byte("another secret!!")); err == nil {
		t.Fatal("a wrong secret opened the link")
	}
	raw, err := net.Dial("tcp", front.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	raw.Write(make([]byte, 64))
	raw.SetReadDeadline(time.Now().Add(3 * time.Second))
	io.Copy(io.Discard, raw)
	if len(reached) != 0 {
		t.Fatal("a stranger reached the block server")
	}
}
