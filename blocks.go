package main

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"text/tabwriter"
)

// Block is the run of consecutive layers one machine holds.
type Block struct {
	Machine string // address of the block server, or "this machine" for the head
	First   int
	Last    int
	FreeMiB int // memory the machine announced, 0 when unknown
}

func (b Block) Layers() int { return b.Last - b.First + 1 }

var (
	deviceLine = regexp.MustCompile(`using device (\S+) \(([^)]+)\).*?- (\d+) MiB free`)
	layerLine  = regexp.MustCompile(`layer\s+(\d+) assigned to device ([^,\s]+)`)
)

// parseBlocks reads the head's load log and returns who holds which layers, in layer order.
// llama.cpp fills the machines one after the other, in proportion to the memory each announced.
func parseBlocks(log string) []Block {
	type device struct {
		addr string
		free int
	}
	devices := map[string]device{}
	for _, m := range deviceLine.FindAllStringSubmatch(log, -1) {
		free, _ := strconv.Atoi(m[3])
		devices[m[1]] = device{m[2], free}
	}
	var blocks []Block
	last, previous := "", -1
	for _, m := range layerLine.FindAllStringSubmatch(log, -1) {
		layer, _ := strconv.Atoi(m[1])
		if layer <= previous {
			// llama.cpp lists the layers once to try the fit, then again for the real load: keep the last list.
			blocks, last = nil, ""
		}
		previous = layer
		if m[2] != last {
			machine, free := "this machine", 0
			if d, ok := devices[m[2]]; ok {
				machine, free = d.addr, d.free
			}
			blocks = append(blocks, Block{Machine: machine, First: layer, FreeMiB: free})
			last = m[2]
		}
		blocks[len(blocks)-1].Last = layer
	}
	return blocks
}

func printBlocks(w io.Writer, blocks []Block) {
	total := 0
	for _, b := range blocks {
		total += b.Layers()
	}
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "BLOCK\tMACHINE\tLAYERS\tSHARE OF THE MODEL")
	for i, b := range blocks {
		fmt.Fprintf(tw, "%d\t%s\t%d to %d\t%d %%\n", i+1, b.Machine, b.First, b.Last, 100*b.Layers()/total)
	}
	tw.Flush()
}
