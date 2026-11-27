package main

import (
	"crypto/ed25519"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// A block server has no login and no encryption, so it listens on its own machine only. What
// other machines reach is the engine, through a link: TLS 1.3 where each side proves a key the
// other expects. Nobody in between can read or change what passes.
//
// For a group set up by hand, both keys come from the secret at the end of the line a holder
// gives to the head. Whoever has the line may use the blocks, nobody else can.

// linkSecret returns the secret of this machine's lines. It is made once and kept: a line given
// to a head still works after a restart. Deleting the file makes every line given so far useless.
func linkSecret() ([]byte, error) {
	home, err := homeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, "link.secret")
	if raw, err := os.ReadFile(path); err == nil {
		if secret, err := hex.DecodeString(strings.TrimSpace(string(raw))); err == nil && len(secret) == 16 {
			return secret, nil
		}
	}
	secret := make([]byte, 16)
	rand.Read(secret)
	return secret, os.WriteFile(path, []byte(hex.EncodeToString(secret)), 0o600)
}

// linkKey derives the key of one side of a group from the group's secret. role is "block" or "head".
func linkKey(secret []byte, role string) ed25519.PrivateKey {
	seed, err := hkdf.Key(sha256.New, secret, nil, "tor6ia-engine link "+role, ed25519.SeedSize)
	if err != nil {
		panic(err) // only for an output longer than the hash allows
	}
	return ed25519.NewKeyFromSeed(seed)
}

// linkConfig is the TLS setting of one side: it shows mine and accepts peer, nothing else.
func linkConfig(mine ed25519.PrivateKey, peer ed25519.PublicKey) (*tls.Config, error) {
	template := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: time.Unix(0, 0), NotAfter: time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)}
	der, err := x509.CreateCertificate(rand.Reader, template, template, mine.Public(), mine)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: mine}},
		MinVersion:   tls.VersionTLS13,
		ClientAuth:   tls.RequireAnyClientCert,
		NextProtos:   []string{linkProto}, // both sides know it: tensors the holder has are named, not sent
		// No authority signs these certificates: the check below compares the key itself.
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(raw [][]byte, _ [][]*x509.Certificate) error {
			if len(raw) > 0 {
				if cert, err := x509.ParseCertificate(raw[0]); err == nil {
					if key, ok := cert.PublicKey.(ed25519.PublicKey); ok && key.Equal(peer) {
						return nil
					}
				}
			}
			return errors.New("the other side does not hold the key of this group")
		},
	}, nil
}

// A holder is reached by one or several routes, tried in order and separated by "|" in its line:
//
//	192.168.1.20:50052        directly, when the head can reach that address
//	relay:relay.host:8444     through a relay both machines reach, when it cannot
const relayPrefix = "relay:"

// blockLink is the holder's side of every link to one block server.
type blockLink struct {
	cfg   *tls.Config
	key   ed25519.PrivateKey
	inner func() string // where the block server listens now
}

func newBlockLink(secret []byte, inner func() string) (*blockLink, error) {
	key := linkKey(secret, "block")
	cfg, err := linkConfig(key, linkKey(secret, "head").Public().(ed25519.PublicKey))
	return &blockLink{cfg, key, inner}, err
}

// serve takes one incoming connection, direct or through a relay. A stranger fails the
// handshake: nothing it sends reaches the block server.
func (b *blockLink) serve(raw net.Conn) {
	c := tls.Server(raw, b.cfg)
	c.SetDeadline(time.Now().Add(15 * time.Second))
	if c.Handshake() != nil {
		c.Close()
		return
	}
	c.SetDeadline(time.Time{})
	in, err := net.DialTimeout("tcp", b.inner(), 5*time.Second)
	if err != nil {
		c.Close()
		return
	}
	if c.ConnectionState().NegotiatedProtocol != linkProto {
		pipe(c, in)
		return
	}
	// Say which tensors are on this disk, then put back the ones the head names instead of sending.
	dir, err := blockCache()
	if err == nil {
		err = sendKnown(c, tensorsIn(dir))
	}
	if err == nil {
		go func() {
			io.Copy(c, answersOf(in)) // the block server's answers go back as they are
			c.Close()
			in.Close()
		}()
		holderFromHead(in, c, dir)
	}
	c.Close()
	in.Close()
}

// noisy spoils what a block server answers: the results of its computations come back wrong, as
// from a machine that is broken or that cheats. The start of each read is left alone so that
// the sizes still read right and the head receives noise, not an error.
type noisy struct{ io.Reader }

func (n noisy) Read(p []byte) (int, error) {
	k, err := n.Reader.Read(p)
	for i := 64; i < k; i++ {
		p[i] ^= 0x55
	}
	return k, err
}

// answersOf is what goes back to the head. T6E_TEST_NOISE=1 makes this holder answer noise: it
// exists to test that a network notices such a machine and leaves it out.
func answersOf(blockServer io.Reader) io.Reader {
	if os.Getenv("T6E_TEST_NOISE") != "" {
		return noisy{blockServer}
	}
	return blockServer
}

// listen accepts direct links on an address of this machine.
func (b *blockLink) listen(addr string) (net.Listener, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go b.serve(c)
		}
	}()
	return l, nil
}

// dialLink opens a port on this machine that leads to the block server of a holder: llama.cpp on
// the head connects to it as to a block server next to it. A holder that is absent, or a wrong
// secret, is an error now. It also returns the route that reached the holder.
func dialLink(routes string, secret []byte) (net.Listener, string, error) {
	block := linkKey(secret, "block").Public().(ed25519.PublicKey)
	cfg, err := linkConfig(linkKey(secret, "head"), block)
	if err != nil {
		return nil, "", err
	}
	// open returns the link, and the tensors the holder said it has when the link names them.
	open := func(route string) (net.Conn, map[uint64]bool, error) {
		wait := 10 * time.Second
		if strings.Contains(routes, "|") {
			wait = 3 * time.Second // another route follows: do not linger on an address out of reach
		}
		var raw net.Conn
		var err error
		if relay, ok := strings.CutPrefix(route, relayPrefix); ok {
			raw, err = relayDial(relay, hex.EncodeToString(block))
		} else {
			raw, err = net.DialTimeout("tcp", route, wait)
		}
		if err != nil {
			return nil, nil, err
		}
		c := tls.Client(raw, cfg)
		c.SetDeadline(time.Now().Add(15 * time.Second))
		if err := c.Handshake(); err != nil {
			c.Close()
			return nil, nil, err
		}
		var have map[uint64]bool
		if c.ConnectionState().NegotiatedProtocol == linkProto {
			if have, err = readKnown(c); err != nil {
				c.Close()
				return nil, nil, err
			}
		}
		c.SetDeadline(time.Time{})
		return c, have, nil
	}
	// The first route that works is the one this group keeps.
	var route string
	var failures []string
	for _, r := range strings.Split(routes, "|") {
		c, _, err := open(r)
		if err == nil {
			c.Close()
			route = r
			break
		}
		failures = append(failures, r+": "+err.Error())
	}
	if route == "" {
		return nil, "", errors.New(strings.Join(failures, "; "))
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", err
	}
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go func() {
				out, have, err := open(route)
				if err != nil {
					c.Close()
					return
				}
				if have == nil {
					pipe(c, out)
					return
				}
				go func() {
					io.Copy(c, out) // answers come back as they are
					c.Close()
					out.Close()
				}()
				headToHolder(out, c, have)
				c.Close()
				out.Close()
			}()
		}
	}()
	return l, route, nil
}

// pipe carries bytes both ways until one side closes.
func pipe(a, b net.Conn) {
	go func() {
		io.Copy(a, b)
		a.Close()
		b.Close()
	}()
	io.Copy(b, a)
	a.Close()
	b.Close()
}
