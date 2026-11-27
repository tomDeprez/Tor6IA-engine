package main

import (
	"errors"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
)

// A block is the unit people choose: a run of consecutive layers of about one gigabyte. Every
// machine cuts a model the same way, from its header alone, so "blocks 4 to 9 of Gemma 4 31B"
// means the same layers everywhere without anyone having to agree first.
const blockTarget = 1 << 30

// A block server needs this many times the size of its layers in memory, and the system keeps
// memoryReserved for itself. Measured on Gemma 4 12B with a context of 8192 tokens: 1.4 GB for
// one block of 1.07 GB, 2.6 GB for two, 3.9 GB for three.
const (
	memoryPerByte  = 1.3
	memoryReserved = 2 << 30
)

type ModelBlock struct {
	First, Last int   // layers
	Size        int64 // bytes of those layers
}

// Blocks cuts the layers of a model into blocks of about blockTarget, as even as possible.
func (p *Plan) Blocks() []ModelBlock {
	var total int64
	sizes := make([]int64, p.Layers)
	for i := range sizes {
		sizes[i] = p.LayerSize(i)
		total += sizes[i]
	}
	count := int((total + blockTarget/2) / blockTarget)
	count = max(1, min(count, p.Layers))
	var blocks []ModelBlock
	var done, inBlock int64
	first := 0
	for layer, size := range sizes {
		done += size
		inBlock += size
		toMake := count - len(blocks) - 1 // blocks still to make after this one
		after := p.Layers - layer - 1     // layers after this one
		share := int64(len(blocks)+1) * total / int64(count)
		// A block ends when the layers so far reach its share of the model, or when each layer
		// that is left is needed to give the remaining blocks one layer each. The last block takes the rest.
		if toMake > 0 && (done >= share || after == toMake) {
			blocks = append(blocks, ModelBlock{First: first, Last: layer, Size: inBlock})
			first, inBlock = layer+1, 0
		}
	}
	return append(blocks, ModelBlock{First: first, Last: p.Layers - 1, Size: inBlock})
}

// blockRange reads "3-5" or "7", block numbers starting at 1.
func blockRange(text string, total int) (first, last int, err error) {
	a, b, isRange := strings.Cut(text, "-")
	if !isRange {
		b = a
	}
	first, err1 := strconv.Atoi(strings.TrimSpace(a))
	last, err2 := strconv.Atoi(strings.TrimSpace(b))
	if err1 != nil || err2 != nil || first < 1 || last < first || last > total {
		return 0, 0, fmt.Errorf("blocks %q: want a range like 3-5, between 1 and %d", text, total)
	}
	return first, last, nil
}

// advise says how many blocks a machine with this much free memory can hold without slowing down.
func advise(blocks []ModelBlock, freeBytes int64) int {
	if len(blocks) == 0 {
		return 0
	}
	var total int64
	for _, b := range blocks {
		total += b.Size
	}
	average := float64(total) / float64(len(blocks))
	n := int(float64(freeBytes-memoryReserved) / (memoryPerByte * average))
	return max(0, min(n, len(blocks)))
}

// holder is one machine of a group and the blocks it announced.
type holder struct {
	Addr        string
	First, Last int // blocks, starting at 1
}

// parseHolders reads "host:port=1-3,host:port=4-9".
func parseHolders(text string, total int) ([]holder, error) {
	var hs []holder
	for _, item := range strings.Split(text, ",") {
		addr, rng, ok := strings.Cut(strings.TrimSpace(item), "=")
		if !ok {
			return nil, fmt.Errorf("%q: want address=blocks, like 192.168.1.20:50052=4-9", item)
		}
		first, last, err := blockRange(rng, total)
		if err != nil {
			return nil, err
		}
		hs = append(hs, holder{addr, first, last})
	}
	sort.Slice(hs, func(i, j int) bool { return hs[i].First < hs[j].First })
	return hs, nil
}

// cover checks that the announced blocks make the whole model, each block once, and says what is wrong otherwise.
func cover(hs []holder, total int) error {
	next := 1
	for _, h := range hs {
		switch {
		case h.First > next:
			return fmt.Errorf("blocks %d to %d of %d are held by no machine", next, h.First-1, total)
		case h.First < next:
			return fmt.Errorf("block %d of %d is announced by two machines", h.First, total)
		}
		next = h.Last + 1
	}
	if next <= total {
		return fmt.Errorf("blocks %d to %d of %d are held by no machine", next, total, total)
	}
	return nil
}

// sharesFor gives each machine as many parts as it holds layers, which makes llama.cpp place the
// layers exactly on the machines that announced them. The step after the last layer goes with it.
func sharesFor(hs []holder, blocks []ModelBlock) string {
	parts := make([]string, len(hs))
	for i, h := range hs {
		layers := blocks[h.Last-1].Last - blocks[h.First-1].First + 1
		if i == len(hs)-1 {
			layers++
		}
		parts[i] = strconv.Itoa(layers)
	}
	return strings.Join(parts, ",")
}

// reachable turns the address a block server listens on into the one to give to the head.
func reachable(listen string) (string, error) {
	host, port, err := net.SplitHostPort(listen)
	if err != nil {
		return "", err
	}
	if host != "" && host != "0.0.0.0" && host != "::" {
		return listen, nil
	}
	// The address this machine uses to reach the outside is the one others can reach. Asking for a
	// route sends nothing. The first interface in the list is often a dead one (169.254.x.x).
	if c, err := net.Dial("udp4", "192.0.2.1:9"); err == nil {
		ip := c.LocalAddr().(*net.UDPAddr).IP
		c.Close()
		if !ip.IsLoopback() && !ip.IsLinkLocalUnicast() {
			return net.JoinHostPort(ip.String(), port), nil
		}
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && !ipnet.IP.IsLinkLocalUnicast() && ipnet.IP.To4() != nil {
			return net.JoinHostPort(ipnet.IP.String(), port), nil
		}
	}
	return "", errors.New("this machine has no network address to give to the head")
}
