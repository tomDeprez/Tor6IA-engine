package main

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
)

// A locked-in block server cannot be reached through the network on every system: on Windows its
// container refuses any connection from outside. So the engine talks to a small bridge that runs
// inside the lock, over the bridge's standard input and output, and the bridge talks to the block
// server next to it. This file carries many connections over that one pair of pipes.
//
// A frame is: connection number (4 bytes), kind (1 byte), payload length (4 bytes), payload.

const (
	frameOpen  = 1 // the outside accepted a connection: open one to the block server
	frameData  = 2
	frameClose = 3
	frameReady = 4 // the bridge tells the outside that the block server listens
	maxFrame   = 1 << 20
)

type mux struct {
	w     io.Writer
	wmu   sync.Mutex
	mu    sync.Mutex
	conns map[uint32]net.Conn
	ready chan struct{} // closed when the other side says it is ready
	once  sync.Once
}

func newMux(w io.Writer) *mux {
	return &mux{w: w, conns: map[uint32]net.Conn{}, ready: make(chan struct{})}
}

func (m *mux) send(id uint32, kind byte, payload []byte) error {
	head := make([]byte, 9, 9+len(payload))
	binary.BigEndian.PutUint32(head, id)
	head[4] = kind
	binary.BigEndian.PutUint32(head[5:], uint32(len(payload)))
	m.wmu.Lock()
	defer m.wmu.Unlock()
	_, err := m.w.Write(append(head, payload...))
	return err
}

// attach carries one connection: what it receives leaves as frames, until it ends.
func (m *mux) attach(id uint32, c net.Conn) {
	m.mu.Lock()
	m.conns[id] = c
	m.mu.Unlock()
	go func() {
		buf := make([]byte, 64<<10)
		for {
			n, err := c.Read(buf)
			if n > 0 && m.send(id, frameData, buf[:n]) != nil {
				break
			}
			if err != nil {
				break
			}
		}
		m.drop(id, true)
	}()
}

func (m *mux) drop(id uint32, tell bool) {
	m.mu.Lock()
	c := m.conns[id]
	delete(m.conns, id)
	m.mu.Unlock()
	if c != nil {
		c.Close()
		if tell {
			m.send(id, frameClose, nil)
		}
	}
}

// run reads frames until the pipe closes. open is called for each connection the other side
// starts; it is nil on the side that only starts connections.
func (m *mux) run(r io.Reader, open func() (net.Conn, error)) error {
	head := make([]byte, 9)
	for {
		if _, err := io.ReadFull(r, head); err != nil {
			m.closeAll()
			return err
		}
		id, kind, size := binary.BigEndian.Uint32(head), head[4], binary.BigEndian.Uint32(head[5:])
		if size > maxFrame {
			m.closeAll()
			return errors.New("bridge: frame too large")
		}
		payload := make([]byte, size)
		if _, err := io.ReadFull(r, payload); err != nil {
			m.closeAll()
			return err
		}
		switch kind {
		case frameOpen:
			if open == nil {
				continue
			}
			c, err := open()
			if err != nil {
				m.send(id, frameClose, nil)
				continue
			}
			m.attach(id, c)
		case frameData:
			m.mu.Lock()
			c := m.conns[id]
			m.mu.Unlock()
			if c != nil {
				if _, err := c.Write(payload); err != nil {
					m.drop(id, true)
				}
			}
		case frameClose:
			m.drop(id, false)
		case frameReady:
			m.once.Do(func() { close(m.ready) })
		}
	}
}

func (m *mux) closeAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, c := range m.conns {
		c.Close()
		delete(m.conns, id)
	}
}

// serveOutside accepts connections on l and carries each one through the bridge.
func (m *mux) serveOutside(l net.Listener) {
	var next uint32
	for {
		c, err := l.Accept()
		if err != nil {
			return
		}
		next++
		if m.send(next, frameOpen, nil) != nil {
			c.Close()
			return
		}
		m.attach(next, c)
	}
}
