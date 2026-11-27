package main

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Two machines behind home routers cannot reach each other, but both can reach a relay. A holder
// keeps a connection open towards it. When a head asks for that holder, the relay tells the
// holder, which opens a second connection, and the relay joins the two.
//
// What passes is the link's own TLS, between the head and the holder: the relay knows who talks
// to whom and how much, never what. It keeps nothing.
//
// One line each way, then raw bytes:
//
//	holder:  HOLD <key>   relay: NONCE <n>   holder: SIG <signature of n>   relay: OK, then CALL <t> at each call
//	head:    DIAL <key>   relay: OK, once the holder answered
//	holder:  ANSWER <t>   relay: OK

// relayLimits protect the server the relay runs on.
type relayLimits struct {
	PairRate  int64 // bytes per second for one head and one holder, both ways together
	TotalRate int64 // bytes per second for the whole relay
	Daily     int64 // bytes per day for the whole relay
}

// bucket slows a flow down to a rate: take waits until n bytes may pass.
type bucket struct {
	mu    sync.Mutex
	rate  float64
	level float64
	at    time.Time
}

func newBucket(rate int64) *bucket {
	return &bucket{rate: float64(rate), level: float64(rate), at: time.Now()}
}

func (b *bucket) take(n int) {
	b.mu.Lock()
	now := time.Now()
	b.level = min(b.rate, b.level+now.Sub(b.at).Seconds()*b.rate) - float64(n)
	b.at = now
	wait := time.Duration(0)
	if b.level < 0 {
		wait = time.Duration(-b.level / b.rate * float64(time.Second))
	}
	b.mu.Unlock()
	time.Sleep(wait)
}

type relay struct {
	net.Listener
	limits relayLimits
	total  *bucket
	mu     sync.Mutex
	held   map[string]net.Conn      // holder's key → its waiting connection
	calls  map[string]chan net.Conn // call → where the holder's answer goes
	bytes  atomic.Int64             // relayed since the start
	today  atomic.Int64             // relayed this day
	day    atomic.Int64
	pairs  atomic.Int64 // joined now
}

func startRelay(listen string, limits relayLimits) (*relay, error) {
	l, err := net.Listen("tcp", listen)
	if err != nil {
		return nil, err
	}
	r := &relay{Listener: l, limits: limits, total: newBucket(limits.TotalRate),
		held: map[string]net.Conn{}, calls: map[string]chan net.Conn{}}
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go r.serve(c)
		}
	}()
	return r, nil
}

func (r *relay) holders() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.held)
}

func (r *relay) relayed() int64 { return r.bytes.Load() }

// line reads one short line: the relay never buffers past it, the bytes after it belong to the link.
func readLine(c net.Conn) (string, error) {
	var line []byte
	one := make([]byte, 1)
	for len(line) < 512 {
		if _, err := io.ReadFull(c, one); err != nil {
			return "", err
		}
		if one[0] == '\n' {
			return string(line), nil
		}
		line = append(line, one[0])
	}
	return "", errors.New("line too long")
}

func token() string {
	raw := make([]byte, 16)
	rand.Read(raw)
	return hex.EncodeToString(raw)
}

func (r *relay) serve(c net.Conn) {
	c.SetDeadline(time.Now().Add(20 * time.Second))
	first, err := readLine(c)
	verb, arg, _ := strings.Cut(first, " ")
	switch {
	case err != nil:
		c.Close()
	case verb == "HOLD":
		r.hold(c, arg)
	case verb == "DIAL":
		r.dial(c, arg)
	case verb == "ANSWER":
		r.mu.Lock()
		waiting := r.calls[arg]
		delete(r.calls, arg)
		r.mu.Unlock()
		if waiting == nil {
			c.Close()
			return
		}
		waiting <- c
	case verb == "STATS":
		json.NewEncoder(c).Encode(map[string]int64{"holders": int64(r.holders()), "pairs": r.pairs.Load(),
			"bytes": r.bytes.Load(), "bytes_today": r.today.Load()})
		c.Close()
	default:
		c.Close()
	}
}

// hold keeps a holder's connection after it proved it owns the key it announces.
func (r *relay) hold(c net.Conn, keyHex string) {
	defer c.Close()
	key, err := hex.DecodeString(keyHex)
	if err != nil || len(key) != ed25519.PublicKeySize {
		return
	}
	nonce := token()
	fmt.Fprintf(c, "NONCE %s\n", nonce)
	answer, err := readLine(c)
	sig, _ := hex.DecodeString(strings.TrimPrefix(answer, "SIG "))
	if err != nil || !ed25519.Verify(key, []byte("tor6ia relay hold "+nonce), sig) {
		return
	}
	r.mu.Lock()
	if old := r.held[keyHex]; old != nil {
		old.Close() // the same holder came back: its earlier connection is dead
	}
	r.held[keyHex] = c
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		if r.held[keyHex] == c {
			delete(r.held, keyHex)
		}
		r.mu.Unlock()
	}()
	fmt.Fprint(c, "OK\n")
	// The holder says PING now and then: silence for a minute means it has gone.
	for {
		c.SetReadDeadline(time.Now().Add(60 * time.Second))
		if _, err := readLine(c); err != nil {
			return
		}
	}
}

// dial joins a head to a holder.
func (r *relay) dial(head net.Conn, keyHex string) {
	defer head.Close()
	if day := time.Now().Unix() / 86400; r.day.Swap(day) != day {
		r.today.Store(0)
	}
	if r.today.Load() > r.limits.Daily {
		fmt.Fprint(head, "ERR the relay has carried all it may carry today\n")
		return
	}
	call, answer := token(), make(chan net.Conn, 1)
	r.mu.Lock()
	holder := r.held[keyHex]
	if holder != nil {
		r.calls[call] = answer
	}
	r.mu.Unlock()
	if holder == nil {
		fmt.Fprint(head, "ERR this holder is not at the relay\n")
		return
	}
	holder.SetWriteDeadline(time.Now().Add(5 * time.Second))
	fmt.Fprintf(holder, "CALL %s\n", call)
	var other net.Conn
	select {
	case other = <-answer:
	case <-time.After(15 * time.Second):
		r.mu.Lock()
		delete(r.calls, call)
		r.mu.Unlock()
		fmt.Fprint(head, "ERR the holder did not answer\n")
		return
	}
	defer other.Close()
	fmt.Fprint(other, "OK\n")
	fmt.Fprint(head, "OK\n")
	head.SetDeadline(time.Time{})
	other.SetDeadline(time.Time{})
	r.pairs.Add(1)
	defer r.pairs.Add(-1)
	pair := newBucket(r.limits.PairRate)
	carry := func(dst, src net.Conn) {
		buf := make([]byte, 32<<10)
		for {
			n, err := src.Read(buf)
			if n > 0 {
				pair.take(n)
				r.total.take(n)
				r.bytes.Add(int64(n))
				r.today.Add(int64(n))
				if _, werr := dst.Write(buf[:n]); werr != nil {
					break
				}
			}
			if err != nil {
				break
			}
		}
		dst.Close()
		src.Close()
	}
	go carry(other, head)
	carry(head, other)
}

// relayDial asks a relay for a holder and returns the connection once the relay joined them.
func relayDial(relay, keyHex string) (net.Conn, error) {
	c, err := net.DialTimeout("tcp", relay, 10*time.Second)
	if err != nil {
		return nil, err
	}
	c.SetDeadline(time.Now().Add(25 * time.Second))
	fmt.Fprintf(c, "DIAL %s\n", keyHex)
	answer, err := readLine(c)
	if err == nil && answer != "OK" {
		err = errors.New(strings.TrimPrefix(answer, "ERR "))
	}
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("relay %s: %w", relay, err)
	}
	c.SetDeadline(time.Time{})
	return c, nil
}

// holdAtRelay keeps this holder reachable through a relay until stop closes. A lost connection
// is opened again.
func (b *blockLink) holdAtRelay(relay string, stop <-chan struct{}) {
	for {
		b.atRelay(relay, stop)
		select {
		case <-stop:
			return
		case <-time.After(5 * time.Second):
		}
	}
}

func (b *blockLink) atRelay(relay string, stop <-chan struct{}) {
	c, err := net.DialTimeout("tcp", relay, 10*time.Second)
	if err != nil {
		return
	}
	defer c.Close()
	gone := make(chan struct{})
	defer close(gone)
	go func() {
		select {
		case <-stop:
			c.Close()
		case <-gone:
		}
	}()
	c.SetDeadline(time.Now().Add(20 * time.Second))
	fmt.Fprintf(c, "HOLD %s\n", hex.EncodeToString(b.key.Public().(ed25519.PublicKey)))
	nonce, err := readLine(c)
	if err != nil || !strings.HasPrefix(nonce, "NONCE ") {
		return
	}
	sig := ed25519.Sign(b.key, []byte("tor6ia relay hold "+strings.TrimPrefix(nonce, "NONCE ")))
	fmt.Fprintf(c, "SIG %s\n", hex.EncodeToString(sig))
	if ok, err := readLine(c); err != nil || ok != "OK" {
		return
	}
	c.SetDeadline(time.Time{})
	go func() {
		for {
			select {
			case <-gone:
				return
			case <-time.After(25 * time.Second):
				c.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if _, err := fmt.Fprint(c, "PING\n"); err != nil {
					c.Close()
					return
				}
			}
		}
	}()
	lines := bufio.NewScanner(c)
	for lines.Scan() {
		call, ok := strings.CutPrefix(lines.Text(), "CALL ")
		if !ok {
			continue
		}
		go func() {
			a, err := net.DialTimeout("tcp", relay, 10*time.Second)
			if err != nil {
				return
			}
			a.SetDeadline(time.Now().Add(20 * time.Second))
			fmt.Fprintf(a, "ANSWER %s\n", call)
			if ok, err := readLine(a); err != nil || ok != "OK" {
				a.Close()
				return
			}
			a.SetDeadline(time.Time{})
			b.serve(a)
		}()
	}
}

func runRelay(args []string) error {
	fs := flag.NewFlagSet("relay", flag.ExitOnError)
	listen := fs.String("listen", "0.0.0.0:8444", "address the relay listens on")
	pair := fs.Float64("pair-mbps", 16, "megabits per second for one head and one holder")
	total := fs.Float64("total-mbps", 40, "megabits per second for the whole relay")
	daily := fs.Float64("daily-gb", 40, "gigabytes per day for the whole relay")
	fs.Parse(args)
	r, err := startRelay(*listen, relayLimits{int64(*pair * 125000), int64(*total * 125000), int64(*daily * (1 << 30))})
	if err != nil {
		return err
	}
	log.Printf("relay on %s: %.0f Mbit/s per pair, %.0f Mbit/s in all, %.0f GB per day", r.Addr(), *pair, *total, *daily)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	return nil
}
