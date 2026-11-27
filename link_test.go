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
	link, err := newBlockLink(secret, func() string { return inner.Addr().String() })
	if err != nil {
		t.Fatal(err)
	}
	front, err := link.listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer front.Close()

	// What goes in must come back.
	echo := func(routes string) string {
		t.Helper()
		local, route, err := dialLink(routes, secret)
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
			t.Fatalf("through %s: %q, %v", route, line, err)
		}
		return route
	}
	echo(front.Addr().String())

	// Through a relay: the holder keeps a connection to it, the head asks it for the holder.
	relay, err := startRelay("127.0.0.1:0", relayLimits{PairRate: 1 << 30, TotalRate: 1 << 30, Daily: 1 << 40})
	if err != nil {
		t.Fatal(err)
	}
	defer relay.Close()
	stop := make(chan struct{})
	defer close(stop)
	go link.holdAtRelay(relay.Addr().String(), stop)
	viaRelay := relayPrefix + relay.Addr().String()
	for i := 0; relay.holders() == 0; i++ {
		if i > 100 {
			t.Fatal("the holder did not reach the relay")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got := echo(viaRelay); got != viaRelay {
		t.Fatalf("route %q, want the relay", got)
	}
	if relay.relayed() == 0 {
		t.Fatal("the relay counted nothing")
	}
	// A direct route that works is preferred, and then nothing passes by the relay.
	for relay.pairs.Load() > 0 { // the last bytes of the link that just closed are still counted
		time.Sleep(10 * time.Millisecond)
	}
	before := relay.relayed()
	if got := echo(front.Addr().String() + "|" + viaRelay); got != front.Addr().String() {
		t.Fatalf("route %q, want the direct one", got)
	}
	if relay.relayed() != before {
		t.Fatal("a direct link passed by the relay")
	}
	// A direct route out of reach falls back to the relay.
	if got := echo("127.0.0.1:1|" + viaRelay); got != viaRelay {
		t.Fatalf("route %q, want the relay after a dead address", got)
	}
	for len(reached) > 0 {
		<-reached
	}

	// A wrong secret is refused, directly and through the relay (where nobody holds that key).
	if _, _, err := dialLink(front.Addr().String()+"|"+viaRelay, []byte("another secret!!")); err == nil {
		t.Fatal("a wrong secret opened the link")
	}
	// A stranger who speaks without TLS gets nothing.
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
