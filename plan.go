package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// llama.cpp's block server keeps on disk the tensors it received, one file each, named by the
// FNV-1a 64-bit hash of its bytes. Before sending a tensor above this size the head sends only its
// hash; when the block server has that file it loads it from disk. So a machine that puts the
// tensors of its layers in that folder beforehand receives almost nothing from the head.
const cacheThreshold = 10 << 20

// blockCache returns the folder where block servers started by this engine keep their tensors.
func blockCache() (string, error) {
	_, write, err := jailHome()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(write, "rpc")
	return dir, os.MkdirAll(dir, 0o755)
}

// parseLayers reads "3-5", "7" or "all".
func parseLayers(text string, total int) (first, last int, err error) {
	if text == "all" || text == "" {
		return 0, total - 1, nil
	}
	a, b, isRange := strings.Cut(text, "-")
	if !isRange {
		b = a
	}
	first, err1 := strconv.Atoi(a)
	last, err2 := strconv.Atoi(b)
	if err1 != nil || err2 != nil || first < 0 || last < first || last >= total {
		return 0, 0, fmt.Errorf("layers %q: want a range like 3-5, between 0 and %d", text, total-1)
	}
	return first, last, nil
}

func runPlan(args []string) error {
	fs := flag.NewFlagSet("plan", flag.ExitOnError)
	model := fs.String("model", "", "model name, or a GGUF file")
	remote := fs.Bool("remote", false, "read the plan from the public registry even when the model is on this disk")
	asJSON := fs.Bool("json", false, "print the plan as JSON")
	fs.Parse(args)
	if *model == "" {
		return errors.New("--model is required")
	}
	src, err := openSource(*model, *remote)
	if err != nil {
		return err
	}
	defer src.Close()
	plan, err := src.Plan()
	if err != nil {
		return err
	}
	sizes := make([]int64, plan.Layers)
	for i := range sizes {
		sizes[i] = plan.LayerSize(i)
	}
	if *asJSON {
		return json.NewEncoder(os.Stdout).Encode(map[string]any{
			"model": *model, "size": plan.Size, "header_read": plan.HeaderSize, "layers": sizes, "outside_layers": plan.LayerSize(-1)})
	}
	where := "read from this disk"
	if !src.Local {
		where = "read from the public registry, model not downloaded"
	}
	lo, hi := sizes[0], sizes[0]
	for _, s := range sizes {
		lo, hi = min(lo, s), max(hi, s)
	}
	fmt.Printf("%s: %.2f GB, %d layers of %d to %d MB, and %d MB outside the layers\n",
		*model, float64(plan.Size)/1e9, plan.Layers, lo>>20, hi>>20, plan.LayerSize(-1)>>20)
	fmt.Printf("Plan %s: %.1f MB read, %.2f %% of the file.\n", where, float64(plan.HeaderSize)/1e6, 100*float64(plan.HeaderSize)/float64(plan.Size))
	return nil
}

// fetchLayers puts the large tensors of some layers in the block cache, where a block server
// started by this engine finds them. It returns how many bytes it had to read from the source.
// outside adds the tensors that belong to no layer: llama.cpp places them with the last layers,
// so the machine that holds the end of the model needs them too.
func fetchLayers(src *source, plan *Plan, first, last int, outside bool, progress func(done, total int64)) (int64, error) {
	dir, err := blockCache()
	if err != nil {
		return 0, err
	}
	var todo []Tensor
	var total int64
	for _, t := range plan.Tensors {
		wanted := (t.Layer >= first && t.Layer <= last) || (outside && t.Layer < 0)
		if wanted && t.Size > cacheThreshold {
			todo = append(todo, t)
			total += t.Size
		}
	}
	var done atomic.Int64
	var firstErr error
	var mu sync.Mutex
	work := make(chan Tensor)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range work {
				buf := make([]byte, t.Size)
				if _, err := src.ReadAt(buf, t.Offset); err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("%s: %w", t.Name, err)
					}
					mu.Unlock()
					continue
				}
				h := fnv.New64a()
				h.Write(buf)
				name := filepath.Join(dir, fmt.Sprintf("%016x", h.Sum64()))
				if err := os.WriteFile(name+".part", buf, 0o644); err == nil {
					err = os.Rename(name+".part", name)
				}
				if err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
				}
				progress(done.Add(t.Size), total)
			}
		}()
	}
	for _, t := range todo {
		work <- t
	}
	close(work)
	wg.Wait()
	return done.Load(), firstErr
}

func runFetch(args []string) error {
	fs := flag.NewFlagSet("fetch", flag.ExitOnError)
	model := fs.String("model", "", "model name, or a GGUF file")
	layers := fs.String("layers", "all", "layers to fetch, like 3-5")
	remote := fs.Bool("remote", false, "fetch from the public registry even when the model is on this disk")
	outside := fs.Bool("outside", false, "also fetch what belongs to no layer: needed by the machine that holds the last layers")
	fs.Parse(args)
	if *model == "" {
		return errors.New("--model is required")
	}
	src, err := openSource(*model, *remote)
	if err != nil {
		return err
	}
	defer src.Close()
	plan, err := src.Plan()
	if err != nil {
		return err
	}
	first, last, err := parseLayers(*layers, plan.Layers)
	if err != nil {
		return err
	}
	start, shown := time.Now(), time.Now()
	n, err := fetchLayers(src, plan, first, last, *outside, func(done, total int64) {
		if time.Since(shown) > time.Second {
			shown = time.Now()
			fmt.Printf("\r%d %%", 100*done/max(total, 1))
		}
	})
	if err != nil {
		return err
	}
	from := "this disk"
	if !src.Local {
		from = "the public registry"
	}
	fmt.Printf("\rLayers %d to %d of %s: %.2f GB taken from %s in %.0f s, %.0f %% of the model.\n",
		first, last, *model, float64(n)/1e9, from, time.Since(start).Seconds(), 100*float64(n)/float64(plan.Size))
	return nil
}
