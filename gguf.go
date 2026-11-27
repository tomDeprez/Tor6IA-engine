package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// A GGUF file starts with a header that lists every tensor of the model: its name, its shape and
// where its bytes are. That header is a fraction of a percent of the file, so the plan of a model
// (how many layers, how big each one is) can be read without downloading the model.

// Tensor is one weight array of a model.
type Tensor struct {
	Name   string
	Layer  int    // layer it belongs to, -1 for what is outside the layers (embeddings, output)
	Offset int64  // where its bytes start in the file
	Size   int64  // how many bytes
	Type   uint32 // ggml type
}

// Plan is what the header says about a model.
type Plan struct {
	Size       int64 // size of the whole file
	HeaderSize int64 // bytes read to build this plan
	Layers     int
	Tensors    []Tensor
}

// LayerSize returns the bytes of one layer, or of everything outside the layers for -1.
func (p *Plan) LayerSize(layer int) int64 {
	var n int64
	for _, t := range p.Tensors {
		if t.Layer == layer {
			n += t.Size
		}
	}
	return n
}

// ggml types: bytes per block and elements per block.
var ggmlTypes = map[uint32][2]int64{
	0: {4, 1}, 1: {2, 1}, 2: {18, 32}, 3: {20, 32}, 6: {22, 32}, 7: {24, 32}, 8: {34, 32}, 9: {36, 32},
	10: {84, 256}, 11: {110, 256}, 12: {144, 256}, 13: {176, 256}, 14: {210, 256}, 15: {292, 256},
	16: {66, 256}, 17: {74, 256}, 18: {98, 256}, 19: {50, 256}, 20: {18, 32}, 21: {110, 256}, 22: {82, 256},
	23: {136, 256}, 24: {1, 1}, 25: {2, 1}, 26: {4, 1}, 27: {8, 1}, 28: {8, 1}, 29: {56, 256}, 30: {2, 1},
	34: {54, 256}, 35: {66, 256},
}

var layerName = regexp.MustCompile(`^blk\.(\d+)\.`)

type counter struct {
	r io.Reader
	n int64
}

func (c *counter) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

// readPlan parses a GGUF header from the start of r. size is the size of the whole file.
func readPlan(r io.Reader, size int64) (*Plan, error) {
	c := &counter{r: r}
	le := binary.LittleEndian
	u32 := func() (v uint32, err error) { err = binary.Read(c, le, &v); return }
	u64 := func() (v uint64, err error) { err = binary.Read(c, le, &v); return }
	skip := func(n uint64) error { _, err := io.CopyN(io.Discard, c, int64(n)); return err }
	str := func() (string, error) {
		n, err := u64()
		if err != nil || n > 1<<20 {
			return "", errors.New("not a GGUF header")
		}
		b := make([]byte, n)
		_, err = io.ReadFull(c, b)
		return string(b), err
	}
	scalar := map[uint32]uint64{0: 1, 1: 1, 2: 2, 3: 2, 4: 4, 5: 4, 6: 4, 7: 1, 10: 8, 11: 8, 12: 8}

	magic := make([]byte, 4)
	if _, err := io.ReadFull(c, magic); err != nil || string(magic) != "GGUF" {
		return nil, errors.New("this is not a GGUF model file")
	}
	if _, err := u32(); err != nil { // version
		return nil, err
	}
	tensors, err := u64()
	if err != nil {
		return nil, err
	}
	pairs, err := u64()
	if err != nil || tensors > 1<<20 || pairs > 1<<20 {
		return nil, errors.New("not a GGUF header")
	}
	alignment := int64(32)
	for i := uint64(0); i < pairs; i++ {
		key, err := str()
		if err != nil {
			return nil, err
		}
		kind, err := u32()
		if err != nil {
			return nil, err
		}
		switch {
		case key == "general.alignment" && kind == 4:
			v, _ := u32()
			alignment = int64(v)
		case kind == 8:
			_, err = str()
		case kind == 9: // an array: the token list is by far the largest part of a header
			elem, _ := u32()
			n, _ := u64()
			if w, ok := scalar[elem]; ok {
				err = skip(n * w)
			} else if elem == 8 {
				for j := uint64(0); j < n && err == nil; j++ {
					var l uint64
					if l, err = u64(); err == nil {
						err = skip(l)
					}
				}
			} else {
				err = fmt.Errorf("GGUF header: array of type %d is not supported", elem)
			}
		default:
			w, ok := scalar[kind]
			if !ok {
				return nil, fmt.Errorf("GGUF header: value of type %d is not supported", kind)
			}
			err = skip(w)
		}
		if err != nil {
			return nil, err
		}
	}
	p := &Plan{Size: size}
	for i := uint64(0); i < tensors; i++ {
		name, err := str()
		if err != nil {
			return nil, err
		}
		dims, err := u32()
		if err != nil || dims > 8 {
			return nil, errors.New("not a GGUF header")
		}
		elements := int64(1)
		for d := uint32(0); d < dims; d++ {
			n, _ := u64()
			elements *= int64(n)
		}
		kind, _ := u32()
		offset, err := u64()
		if err != nil {
			return nil, err
		}
		t := Tensor{Name: name, Layer: -1, Offset: int64(offset), Type: kind}
		if blk, ok := ggmlTypes[kind]; ok {
			t.Size = elements / blk[1] * blk[0]
		}
		if m := layerName.FindStringSubmatch(name); m != nil {
			fmt.Sscan(m[1], &t.Layer)
			p.Layers = max(p.Layers, t.Layer+1)
		}
		p.Tensors = append(p.Tensors, t)
	}
	p.HeaderSize = c.n
	// Tensor offsets count from the first aligned byte after the header.
	data := (p.HeaderSize + alignment - 1) / alignment * alignment
	for i := range p.Tensors {
		p.Tensors[i].Offset += data
	}
	return p, nil
}

// source is where a model's bytes come from: the file Ollama keeps, or the public registry.
type source struct {
	Name  string
	Size  int64
	file  *os.File // set when the model is on this disk
	url   string   // otherwise
	Local bool
}

// openSource finds a model. A model already on disk is read there: nothing is downloaded twice.
// remote forces the registry, to prove a model can be planned and fetched without having it.
func openSource(model string, remote bool) (*source, error) {
	if !remote {
		if path, err := resolveModel(model); err == nil {
			f, err := os.Open(path)
			if err != nil {
				return nil, err
			}
			st, _ := f.Stat()
			return &source{Name: model, Size: st.Size(), file: f, Local: true}, nil
		}
	}
	family, tag, ok := strings.Cut(model, ":")
	if !ok {
		tag = "latest"
	}
	base := "https://registry.ollama.ai/v2/library/" + family
	req, _ := http.NewRequest("GET", base+"/manifests/"+tag, nil)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("the public registry does not know the model %q (%s)", model, resp.Status)
	}
	var manifest struct {
		Layers []struct {
			MediaType string `json:"mediaType"`
			Digest    string `json:"digest"`
			Size      int64  `json:"size"`
		} `json:"layers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, err
	}
	for _, l := range manifest.Layers {
		if strings.HasSuffix(l.MediaType, "image.model") {
			return &source{Name: model, Size: l.Size, url: base + "/blobs/" + l.Digest}, nil
		}
	}
	return nil, fmt.Errorf("the registry's copy of %q has no model file", model)
}

func (s *source) Close() {
	if s.file != nil {
		s.file.Close()
	}
}

// ReadAt reads one piece of the model, from the disk or with an HTTP range request.
func (s *source) ReadAt(p []byte, off int64) (int, error) {
	if s.file != nil {
		return s.file.ReadAt(p, off)
	}
	// A connection that stalls or drops is retried from the byte it reached: a piece of several
	// hundred megabytes must survive a bad minute.
	done := 0
	var last error
	for attempt := 0; attempt < 8 && done < len(p); attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
		n, err := s.readRange(p[done:], off+int64(done))
		done += n
		if last = err; err == nil {
			break
		}
		if n > 0 {
			attempt = 0 // it advanced: the link works, keep going
		}
	}
	if done < len(p) {
		return done, fmt.Errorf("the registry stopped answering: %w", last)
	}
	return done, nil
}

// readRange asks for one piece. It gives up when nothing arrives for a while, and reports what it got.
func (s *source) readRange(p []byte, off int64) (int, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	idle := time.AfterFunc(30*time.Second, cancel)
	defer idle.Stop()
	req, _ := http.NewRequestWithContext(ctx, "GET", s.url, nil)
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", off, off+int64(len(p))-1))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPartialContent {
		return 0, fmt.Errorf("the registry did not serve a piece of the file (%s)", resp.Status)
	}
	done := 0
	for done < len(p) {
		n, err := resp.Body.Read(p[done:])
		done += n
		if n > 0 {
			idle.Reset(30 * time.Second)
		}
		if err != nil {
			if done == len(p) {
				return done, nil
			}
			return done, err
		}
	}
	return done, nil
}

// Plan reads the model's header. From the registry it is fetched a few megabytes at a time.
func (s *source) Plan() (*Plan, error) {
	return readPlan(&chunked{src: s, step: 8 << 20}, s.Size)
}

// chunked reads a source from its start, one large piece at a time.
type chunked struct {
	src  *source
	step int64
	pos  int64
	buf  []byte
}

func (c *chunked) Read(p []byte) (int, error) {
	if len(c.buf) == 0 {
		if c.pos >= c.src.Size {
			return 0, io.EOF
		}
		c.buf = make([]byte, min(c.step, c.src.Size-c.pos))
		n, err := c.src.ReadAt(c.buf, c.pos)
		if n == 0 && err != nil {
			return 0, err
		}
		c.buf = c.buf[:n]
		c.pos += int64(n)
	}
	n := copy(p, c.buf)
	c.buf = c.buf[n:]
	return n, nil
}
