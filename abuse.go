package main

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"
)

// A block server obeys the head that holds its line, and that head can be hostile: a wrong
// order, a size it cannot hold, a connection cut in the middle. The lock keeps the damage inside
// the block server. The keeper below starts it again, so that the next head finds it.

// blockKeeper keeps one block server running.
type blockKeeper struct {
	start    func(addr string) (*jailProc, error)
	mu       sync.Mutex
	block    *jailProc
	restarts int
	stopped  bool
	Gone     chan error // receives once, when the block server cannot be kept running
}

// keepBlock starts a block server on a port of this machine and starts it again, on a new port,
// each time it stops by itself. report is told why it stopped.
func keepBlock(start func(addr string) (*jailProc, error), report func(said string)) (*blockKeeper, error) {
	k := &blockKeeper{start: start, Gone: make(chan error, 1)}
	b, err := k.fresh()
	if err != nil {
		return nil, err
	}
	k.block = b
	go func() {
		var recent []time.Time
		for {
			<-b.Done
			k.mu.Lock()
			stopped := k.stopped
			k.mu.Unlock()
			if stopped {
				return
			}
			said, now := tailFile(b.Log, 3), time.Now()
			recent = append(recent, now)
			for now.Sub(recent[0]) > time.Minute {
				recent = recent[1:]
			}
			if len(recent) > 20 {
				k.Gone <- fmt.Errorf("the block server stopped %d times in a minute, the last time with:\n%s", len(recent), said)
				return
			}
			report(said)
			if b, err = k.fresh(); err != nil {
				k.Gone <- err
				return
			}
			k.mu.Lock()
			if k.stopped {
				k.mu.Unlock()
				b.Stop()
				return
			}
			k.block = b
			k.restarts++
			k.mu.Unlock()
		}
	}()
	return k, nil
}

func (k *blockKeeper) fresh() (*jailProc, error) {
	port, err := freePort()
	if err != nil {
		return nil, err
	}
	return k.start("127.0.0.1:" + strconv.Itoa(port))
}

// Addr is where the block server listens now.
func (k *blockKeeper) Addr() string {
	k.mu.Lock()
	defer k.mu.Unlock()
	return dialable(k.block.Addr)
}

func (k *blockKeeper) Restarts() int {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.restarts
}

func (k *blockKeeper) Stop() {
	k.mu.Lock()
	k.stopped = true
	b := k.block
	k.mu.Unlock()
	b.Stop()
}

// The orders of llama.cpp's block server used below (ggml-rpc.cpp). An order is one byte, the
// size of what follows on 8 bytes, then that many bytes.
const (
	rpcAllocBuffer  = 0
	rpcSetTensor    = 6
	rpcGraphCompute = 10
	rpcHello        = 14
)

func order(cmd byte, size uint64, payload []byte) []byte {
	msg := append([]byte{cmd}, binary.LittleEndian.AppendUint64(nil, size)...)
	return append(msg, payload...)
}

// greet opens a connection and says hello the way a head does. helloSize is the size of the
// hello message of this llama.cpp build.
func greet(addr string, helloSize int) (net.Conn, error) {
	c, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return nil, err
	}
	c.SetDeadline(time.Now().Add(20 * time.Second))
	c.Write(order(rpcHello, uint64(helloSize), make([]byte, helloSize)))
	var size uint64
	if err := binary.Read(c, binary.LittleEndian, &size); err != nil || size > 1024 {
		c.Close()
		return nil, errors.New("no hello back")
	}
	if _, err := io.CopyN(io.Discard, c, int64(size)); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

// abuseTest starts a locked-in block server with 512 MB of memory and plays a hostile head. After
// each attempt the block server must answer a polite head again: it refused, or it stopped and
// was started again. Anything else is a failure.
func abuseTest(device string) error {
	dir, err := ensureLlama()
	if err != nil {
		return err
	}
	threads := 2
	k, err := keepBlock(func(addr string) (*jailProc, error) {
		return startBlock(dir, addr, threads, device, true, 512, false)
	}, func(string) {})
	if err != nil {
		return err
	}
	defer k.Stop()

	// The size of the hello message is not part of what the engine knows: find the one it accepts.
	helloSize := -1
	for size := 0; size <= 64 && helloSize < 0; size++ {
		if c, err := greet(k.Addr(), size); err == nil {
			c.Close()
			helloSize = size
		}
	}
	if helloSize < 0 {
		return errors.New("the block server does not answer a hello")
	}
	alive := func() bool {
		for deadline := time.Now().Add(40 * time.Second); time.Now().Before(deadline); time.Sleep(300 * time.Millisecond) {
			select {
			case <-k.Gone:
				return false
			default:
			}
			if c, err := greet(k.Addr(), helloSize); err == nil {
				c.Close()
				return true
			}
		}
		return false
	}
	noise := make([]byte, 64<<10)
	rand.Read(noise)
	alloc := func(size uint64) []byte {
		req := binary.LittleEndian.AppendUint64(binary.LittleEndian.AppendUint32(nil, 0), size)
		return order(rpcAllocBuffer, uint64(len(req)), req)
	}
	attempts := []struct {
		what  string
		hello bool // say hello first, as a head would
		send  [][]byte
	}{
		{"noise instead of a hello", false, [][]byte{noise}},
		{"a hello of an absurd size", false, [][]byte{order(rpcHello, 1<<63, noise)}},
		{"an order that does not exist", true, [][]byte{order(200, 8, noise[:8])}},
		{"a computation of an absurd size", true, [][]byte{order(rpcGraphCompute, 1<<62, noise)}},
		{"a computation made of noise", true, [][]byte{order(rpcGraphCompute, uint64(len(noise)), noise)}},
		{"a computation larger than its memory", true, [][]byte{order(rpcGraphCompute, 2<<30, noise)}},
		{"one giant block of memory", true, [][]byte{alloc(1 << 44)}},
		{"more memory than it was given", true, [][]byte{alloc(300 << 20), alloc(300 << 20), alloc(300 << 20), alloc(300 << 20)}},
		{"a tensor cut in the middle", true, [][]byte{order(rpcSetTensor, 8<<20, noise)}},
	}
	failed := false
	fmt.Printf("%-40s %s\n", "A HOSTILE HEAD SENDS", "THE BLOCK SERVER")
	for _, a := range attempts {
		before := k.Restarts()
		var c net.Conn
		var err error
		if a.hello {
			c, err = greet(k.Addr(), helloSize)
		} else {
			c, err = net.DialTimeout("tcp", k.Addr(), 3*time.Second)
		}
		if err == nil {
			c.SetDeadline(time.Now().Add(6 * time.Second))
			for _, msg := range a.send {
				c.Write(msg)
			}
			io.Copy(io.Discard, io.LimitReader(c, 1<<20)) // until it answers, closes or stops
			c.Close()
		}
		verdict := "refused, still running"
		switch {
		case !alive():
			verdict, failed = "DID NOT COME BACK", true
		case k.Restarts() > before:
			verdict = "stopped, started again"
		}
		fmt.Printf("%-40s %s\n", a.what, verdict)
		if failed {
			break
		}
	}
	if failed {
		return errors.New("a hostile head left the block server out of service")
	}
	// Whatever a head finds to make it stop, the result is this: the process ends, and comes back.
	before := k.Restarts()
	k.mu.Lock()
	b := k.block
	k.mu.Unlock()
	b.kill()
	if !alive() || k.Restarts() != before+1 {
		fmt.Printf("%-40s %s\n", "(its process is ended)", "DID NOT COME BACK")
		return errors.New("a block server that stopped was not started again")
	}
	fmt.Printf("%-40s %s\n", "(its process is ended)", "stopped, started again")
	return nil
}
