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
	"fmt"
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

// serveLink listens on listen and carries each link to the block server, which inner locates.
// A stranger fails the handshake: nothing it sends reaches the block server.
func serveLink(listen string, secret []byte, inner func() string) (net.Listener, error) {
	cfg, err := linkConfig(linkKey(secret, "block"), linkKey(secret, "head").Public().(ed25519.PublicKey))
	if err != nil {
		return nil, err
	}
	l, err := tls.Listen("tcp", listen, cfg)
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go func() {
				c.SetDeadline(time.Now().Add(15 * time.Second))
				if c.(*tls.Conn).Handshake() != nil {
					c.Close()
					return
				}
				c.SetDeadline(time.Time{})
				in, err := net.DialTimeout("tcp", inner(), 5*time.Second)
				if err != nil {
					c.Close()
					return
				}
				pipe(c, in)
			}()
		}
	}()
	return l, nil
}

// dialLink opens a port on this machine that leads to the block server of a holder: llama.cpp on
// the head connects to it as to a block server next to it. A holder that is absent, or a wrong
// secret, is an error now.
func dialLink(addr string, secret []byte) (net.Listener, error) {
	cfg, err := linkConfig(linkKey(secret, "head"), linkKey(secret, "block").Public().(ed25519.PublicKey))
	if err != nil {
		return nil, err
	}
	dial := func() (net.Conn, error) {
		return tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, cfg)
	}
	first, err := dial()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", addr, err)
	}
	first.Close()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go func() {
				out, err := dial()
				if err != nil {
					c.Close()
					return
				}
				pipe(c, out)
			}()
		}
	}()
	return l, nil
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
