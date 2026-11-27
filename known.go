package main

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
)

// When it loads a model, llama.cpp on the head sends every tensor to the block server that gets
// it. For the large ones it first sends a hash, and a holder that fetched its blocks answers "I
// have it". Tensors under 10 MB are always sent whole: a fifth of the model, which is slow and
// costly through a relay.
//
// The link closes that gap. A holder has every tensor of its blocks on its disk, in files named
// by the hash of their content. When a link opens, it says which ones. The head then watches what
// llama.cpp sends: a tensor the holder has leaves as its hash, and the holder puts the bytes back
// from its disk before they reach its block server. llama.cpp on both sides sees nothing of it.

// linkProto is the TLS protocol name of links that work this way. A side that does not know it
// gets the plain link.
const linkProto = "t6-known-tensors"

// What the head looks at in llama.cpp's stream (ggml-rpc.cpp of the pinned build): an order is
// one byte, the size of what follows on 8 bytes, then that many bytes. For "set tensor" what
// follows is a description of fixed size, then the tensor's bytes.
const (
	rpcOrders       = 24      // orders that exist: a larger number means the stream is not what we think
	tensorHeader    = 305     // the description: rpc_tensor (296 bytes), one flag, the offset on 8 bytes
	tensorNameAt    = 228     // where the tensor's name is in it, on 64 bytes
	knownMin        = 4 << 10 // smaller than this, a hash saves nothing worth the detour
	knownMax        = 64 << 20
	recordRaw       = 0
	recordKnown     = 1
	maxRecordHeader = 4 << 10
)

func fnvOf(data []byte) uint64 {
	h := fnv.New64a()
	h.Write(data)
	return h.Sum64()
}

// tensorsIn lists the tensors a holder has: the files of its cache, named by their hash.
func tensorsIn(dir string) []uint64 {
	var have []uint64
	files, _ := os.ReadDir(dir)
	for _, f := range files {
		if raw, err := hex.DecodeString(f.Name()); err == nil && len(raw) == 8 {
			have = append(have, binary.BigEndian.Uint64(raw))
		}
	}
	return have
}

// sendKnown is the first thing a holder writes on a link: how many tensors it has, then their hashes.
func sendKnown(link io.Writer, have []uint64) error {
	msg := binary.LittleEndian.AppendUint32(nil, uint32(len(have)))
	for _, h := range have {
		msg = binary.LittleEndian.AppendUint64(msg, h)
	}
	_, err := link.Write(msg)
	return err
}

func readKnown(link io.Reader) (map[uint64]bool, error) {
	var count uint32
	if err := binary.Read(link, binary.LittleEndian, &count); err != nil {
		return nil, err
	}
	if count > 1<<20 {
		return nil, errors.New("the holder announces more tensors than a model has")
	}
	raw := make([]byte, 8*int(count))
	if _, err := io.ReadFull(link, raw); err != nil {
		return nil, err
	}
	have := make(map[uint64]bool, count)
	for i := 0; i < len(raw); i += 8 {
		have[binary.LittleEndian.Uint64(raw[i:])] = true
	}
	return have, nil
}

func record(kind byte, payload []byte) []byte {
	msg := append([]byte{kind}, binary.LittleEndian.AppendUint32(nil, uint32(len(payload)))...)
	return append(msg, payload...)
}

// headToHolder carries what llama.cpp on the head sends, replacing the tensors the holder has by
// their hash. It returns when one side closes.
func headToHolder(link io.Writer, local io.Reader, have map[uint64]bool) error {
	in := bufio.NewReaderSize(local, 256<<10)
	raw := func(n uint64, first []byte) error { // first, then n more bytes, as they come
		if n <= 256<<10 {
			// A small order leaves in one piece: one packet on the network, not two.
			msg := make([]byte, len(first)+int(n))
			copy(msg, first)
			if _, err := io.ReadFull(in, msg[len(first):]); err != nil {
				return err
			}
			_, err := link.Write(record(recordRaw, msg))
			return err
		}
		if _, err := link.Write(record(recordRaw, first)); err != nil {
			return err
		}
		buf := make([]byte, 64<<10)
		for n > 0 {
			k, err := in.Read(buf[:min(n, uint64(len(buf)))])
			if k > 0 {
				if _, werr := link.Write(record(recordRaw, buf[:k])); werr != nil {
					return werr
				}
				n -= uint64(k)
			}
			if err != nil {
				return err
			}
		}
		return nil
	}
	for {
		head := make([]byte, 9)
		if _, err := io.ReadFull(in, head[:1]); err != nil {
			return err
		}
		if head[0] >= rpcOrders {
			// Not the stream we know: carry the rest untouched.
			if _, err := link.Write(record(recordRaw, head[:1])); err != nil {
				return err
			}
			buf := make([]byte, 64<<10)
			for {
				k, err := in.Read(buf)
				if k > 0 {
					if _, werr := link.Write(record(recordRaw, buf[:k])); werr != nil {
						return werr
					}
				}
				if err != nil {
					return err
				}
			}
		}
		if _, err := io.ReadFull(in, head[1:]); err != nil {
			return err
		}
		order, size := head[0], binary.LittleEndian.Uint64(head[1:])
		if order != rpcSetTensor || size < tensorHeader+knownMin || size > knownMax {
			if err := raw(size, head); err != nil {
				return err
			}
			continue
		}
		body := make([]byte, size)
		if _, err := io.ReadFull(in, body); err != nil {
			return err
		}
		desc, data := body[:tensorHeader], body[tensorHeader:]
		hash := fnvOf(data)
		if !have[hash] || !nameAt(desc) {
			if _, err := link.Write(record(recordRaw, append(head, body...))); err != nil {
				return err
			}
			continue
		}
		// The order and the description go as they are. The bytes are named, not sent.
		known := append(append([]byte{}, head...), desc...)
		known = binary.LittleEndian.AppendUint64(known, hash)
		known = binary.LittleEndian.AppendUint64(known, uint64(len(data)))
		if _, err := link.Write(record(recordKnown, known)); err != nil {
			return err
		}
	}
}

// nameAt checks that the description holds a tensor name where the pinned llama.cpp puts it:
// some printable text, then zeros. Another layout must not be cut at the wrong place.
func nameAt(desc []byte) bool {
	name := desc[tensorNameAt : tensorNameAt+64]
	n := 0
	for n < len(name) && name[n] != 0 {
		if name[n] < 0x20 || name[n] > 0x7e {
			return false
		}
		n++
	}
	return n > 0 && n < len(name)
}

// holderFromHead is the other end: it gives the block server what the head's llama.cpp meant to
// send, taking from dir the tensors that came as a hash.
func holderFromHead(inner io.Writer, link io.Reader, dir string) error {
	in := bufio.NewReaderSize(link, 256<<10)
	for {
		head := make([]byte, 5)
		if _, err := io.ReadFull(in, head); err != nil {
			return err
		}
		size := binary.LittleEndian.Uint32(head[1:])
		switch head[0] {
		case recordRaw:
			if _, err := io.CopyN(inner, in, int64(size)); err != nil {
				return err
			}
		case recordKnown:
			if size < 16 || size > maxRecordHeader {
				return errors.New("bad record")
			}
			msg := make([]byte, size)
			if _, err := io.ReadFull(in, msg); err != nil {
				return err
			}
			hash, length := binary.LittleEndian.Uint64(msg[size-16:]), binary.LittleEndian.Uint64(msg[size-8:])
			name := make([]byte, 8)
			binary.BigEndian.PutUint64(name, hash)
			f, err := os.Open(filepath.Join(dir, hex.EncodeToString(name)))
			if err != nil {
				return err
			}
			st, err := f.Stat()
			if err == nil && uint64(st.Size()) != length {
				err = errors.New("a tensor on this disk does not have the size the head expects")
			}
			if err == nil {
				_, err = inner.Write(msg[:size-16])
			}
			if err == nil {
				_, err = io.Copy(inner, f)
			}
			f.Close()
			if err != nil {
				return err
			}
		default:
			return errors.New("bad record")
		}
	}
}
