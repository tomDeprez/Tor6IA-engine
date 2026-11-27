// tor6ia-engine runs one language model across several machines: the model is cut into
// blocks of layers and each machine holds the blocks its memory allows.
package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"strings"
)

var version = "dev"

const unlockedHelp = "run block servers with every right of your account instead of locked in: only for machines you trust, or to debug"

const deviceHelp = `what computes on a block server: empty for llama.cpp's choice, CPU for the processor (use it if a block server stops with "unsupported op")`

const usage = `tor6ia-engine: one model, several machines

  tor6ia-engine demo  [--model gemma4:e2b] [--machines 3]      try it on this machine alone
  tor6ia-engine plan  --model M                                 the blocks of a model, and how many this machine can hold
  tor6ia-engine block --model M --blocks 4-9 [--listen ADDR]    hold some blocks of a model for a head
  tor6ia-engine head  --model M --blocks LINE,LINE              run a model across the machines that hold its blocks
                                                                (a LINE is what "block" prints: address=blocks#secret)
  tor6ia-engine relay [--listen ADDR]                           join machines that cannot reach each other (for a server)
  tor6ia-engine clean [--all]                                   empty the block cache, or remove everything the engine installed
  tor6ia-engine version

A model is a .gguf file, or the name of a model Ollama already downloaded.`

func main() {
	if len(os.Args) < 2 {
		fmt.Println(usage)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "demo":
		fs := flag.NewFlagSet("demo", flag.ExitOnError)
		model := fs.String("model", "gemma4:e2b", "GGUF file, or a model Ollama has")
		machines := fs.Int("machines", 3, "how many block servers to start")
		prompt := fs.String("prompt", "Explain in one sentence what a peer-to-peer network is.", "question to ask")
		device := fs.String("device", "", deviceHelp)
		unlocked := fs.Bool("unlocked", false, unlockedHelp)
		fs.Parse(os.Args[2:])
		if *machines < 1 || *machines > 16 {
			err = fmt.Errorf("--machines must be between 1 and 16")
			break
		}
		err = demo(*model, *machines, *prompt, *device, !*unlocked)
	case "block":
		err = runBlock(os.Args[2:])
	case "head":
		err = runHead(os.Args[2:])
	case "plan":
		err = runPlan(os.Args[2:])
	case "fetch":
		err = runFetch(os.Args[2:])
	case "clean":
		err = runClean(os.Args[2:])
	case "relay":
		err = runRelay(os.Args[2:])
	case "jail-test":
		err = jailTest()
	case "abuse-test":
		fs := flag.NewFlagSet("abuse-test", flag.ExitOnError)
		device := fs.String("device", "", deviceHelp)
		fs.Parse(os.Args[2:])
		err = abuseTest(*device)
	case "jail-exec":
		err = runJailExec(os.Args[2:])
	case "bridge":
		err = runBridge(os.Args[2:])
	case "probe":
		err = runProbe(os.Args[2:])
	case "version":
		fmt.Println("tor6ia-engine", version, "with llama.cpp", llamaBuild)
	default:
		fmt.Println(usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func runBlock(args []string) error {
	fs := flag.NewFlagSet("block", flag.ExitOnError)
	listen := fs.String("listen", "127.0.0.1:50052", "address other machines reach this one on")
	threads := fs.Int("threads", max(1, runtime.NumCPU()/2), "processors to use")
	device := fs.String("device", "", deviceHelp)
	unlocked := fs.Bool("unlocked", false, unlockedHelp)
	model := fs.String("model", "", "model whose blocks this machine holds")
	blockList := fs.String("blocks", "", "blocks to hold, like 4-9 (see the plan command)")
	asJSON := fs.Bool("json", false, "print progress and the ready line as JSON, for a program that drives the engine")
	withStdin := fs.Bool("exit-with-stdin", false, "stop when standard input closes: the program that started the engine has gone")
	shared := fs.Bool("shared-cache", false, "keep the blocks of other holders in the cache: for several holders on one machine")
	relay := fs.String("relay", "", "host:port of a relay, for heads that cannot reach this machine directly")
	fs.Parse(args)
	dir, err := ensureLlama()
	if err != nil {
		return err
	}
	// With a model, the machine first gets the blocks it announced, and only those.
	memoryMB, announce := 0, ""
	if *model != "" {
		src, err := openSource(*model, false)
		if err != nil {
			return err
		}
		plan, err := src.Plan()
		if err != nil {
			src.Close()
			return err
		}
		blocks := plan.Blocks()
		first, last, err := blockRange(*blockList, len(blocks))
		if err != nil {
			src.Close()
			return err
		}
		var size int64
		for _, b := range blocks[first-1 : last] {
			size += b.Size
		}
		shown := -1
		_, files, err := fetchLayers(src, plan, blocks[first-1].First, blocks[last-1].Last, false, func(done, total int64) {
			if pct := int(100 * done / max(total, 1)); pct != shown {
				shown = pct
				if *asJSON {
					fmt.Printf("{\"progress\":%d}\n", pct)
				} else {
					fmt.Printf("\rGetting blocks %d to %d of %s: %d %%", first, last, *model, pct)
				}
			}
		})
		src.Close()
		if err != nil {
			return err
		}
		if !*asJSON {
			fmt.Println()
		}
		// The cache holds the blocks this machine holds, and nothing it held before.
		if !*shared {
			if _, err := pruneCache(files); err != nil {
				return err
			}
		}
		memoryMB = memoryCap(size)
		announce = fmt.Sprintf("=%d-%d", first, last)
	}
	// The block server listens on this machine only: it has no login. Other machines reach the
	// engine, through a link that asks for the secret of the line printed below.
	// A head can make it stop, with a wrong order or a size it cannot hold: it is started again.
	block, err := keepBlock(func(addr string) (*jailProc, error) {
		return startBlock(dir, addr, *threads, *device, !*unlocked, memoryMB, *model != "")
	}, func(said string) {
		if *asJSON {
			json.NewEncoder(os.Stdout).Encode(map[string]any{"restarted": true, "because": said})
		} else {
			fmt.Printf("The block server stopped and was started again. Its last words:\n%s\n", said)
		}
	})
	if err != nil {
		return err
	}
	defer block.Stop()
	secret, err := linkSecret()
	if err != nil {
		return err
	}
	link, err := newBlockLink(secret, block.Addr)
	if err != nil {
		return err
	}
	front, err := link.listen(*listen)
	if err != nil {
		return err
	}
	defer front.Close()
	addr, err := reachable(front.Addr().String())
	if err != nil {
		addr = front.Addr().String()
	}
	// With a relay, a head that cannot reach this machine directly still finds it.
	if *relay != "" {
		left := make(chan struct{})
		defer close(left)
		go link.holdAtRelay(*relay, left)
		addr += "|" + relayPrefix + *relay
	}
	line := addr + announce + "#" + hex.EncodeToString(secret)
	if *asJSON {
		json.NewEncoder(os.Stdout).Encode(map[string]any{"ready": true, "addr": addr, "announce": line, "locked": !*unlocked})
	} else {
		how := "locked in: it cannot read your files, connect out or start programs"
		if *unlocked {
			how = "NOT locked in: it has every right of your account"
		}
		fmt.Printf("Give this line to the head of the group:  %s\n", line)
		fmt.Printf("Block server behind %s, %d processors, %s.\n", front.Addr(), *threads, how)
		fmt.Println("Only a machine that has this line can give it work, and what passes between them is encrypted.")
		fmt.Println("Stop it with Ctrl+C.")
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	if *withStdin {
		go func() {
			io.Copy(io.Discard, os.Stdin)
			stop <- os.Interrupt
		}()
	}
	select {
	case <-stop:
	case err := <-block.Gone:
		return err
	}
	return nil
}

func runHead(args []string) error {
	fs := flag.NewFlagSet("head", flag.ExitOnError)
	model := fs.String("model", "", "GGUF file, or a model Ollama has")
	blocks := fs.String("blocks", "", "the lines the holders gave, separated by commas: host:port=1-3#secret,host:port=4-9#secret")
	port := fs.Int("port", 8090, "local port of the OpenAI-compatible API")
	threads := fs.Int("threads", max(1, runtime.NumCPU()/2), "processors to use")
	context := fs.Int("ctx", defaultContext, "tokens a conversation may hold; every block server reserves memory for it")
	shares := fs.String("shares", "", "part of the model each block server takes, in order, like 1,3,2 (default: by free memory)")
	asJSON := fs.Bool("json", false, "print the address and the blocks as one line of JSON, for a program that drives the engine")
	withStdin := fs.Bool("exit-with-stdin", false, "stop when standard input closes: the program that started the engine has gone")
	fs.Parse(args)
	if *model == "" || *blocks == "" {
		return fmt.Errorf("--model and --blocks are required")
	}
	dir, err := ensureLlama()
	if err != nil {
		return err
	}
	file, err := resolveModel(*model)
	if err != nil {
		return err
	}
	// Each item is the line a holder gave: its address, the blocks it holds, the secret of its link.
	secrets := map[string][]byte{}
	var addrs, announced []string
	for _, item := range strings.Split(*blocks, ",") {
		rest, secretHex, _ := strings.Cut(strings.TrimSpace(item), "#")
		secret, err := hex.DecodeString(secretHex)
		if err != nil || len(secret) != 16 {
			return fmt.Errorf("%q: the secret at the end is missing or cut. Copy the whole line the holder shows", rest)
		}
		addr, _, hasBlocks := strings.Cut(rest, "=")
		secrets[addr] = secret
		addrs = append(addrs, addr)
		if hasBlocks {
			announced = append(announced, rest)
		}
	}
	// Machines that announced their blocks get exactly those layers. The group must make the whole model.
	var holders []holder
	var modelBlocks []ModelBlock
	if len(announced) > 0 {
		if len(announced) != len(addrs) {
			return fmt.Errorf("some machines announce their blocks and others do not: every line must be of the same kind")
		}
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		st, _ := f.Stat()
		plan, err := readPlan(f, st.Size())
		f.Close()
		if err != nil {
			return err
		}
		modelBlocks = plan.Blocks()
		if holders, err = parseHolders(strings.Join(announced, ","), len(modelBlocks)); err != nil {
			return err
		}
		if err := cover(holders, len(modelBlocks)); err != nil {
			return err
		}
		addrs = addrs[:0]
		for _, h := range holders {
			addrs = append(addrs, h.Addr)
		}
		*shares = sharesFor(holders, modelBlocks)
	}
	// llama.cpp connects to ports of this machine: each one leads to a holder through its link.
	local, holderOf, routes := make([]string, len(addrs)), map[string]string{}, map[string]string{}
	for i, addr := range addrs {
		l, route, err := dialLink(addr, secrets[addr])
		if err != nil {
			return fmt.Errorf("the holder does not answer, or its line is not the right one: %w", err)
		}
		defer l.Close()
		local[i] = l.Addr().String()
		holderOf[local[i]] = addr
		routes[addr] = route
	}
	fmt.Fprintf(os.Stderr, "Loading %s across %s...\n", *model, strings.Join(addrs, ", "))
	head, err := startHead(dir, file, local, *shares, *context, *port, *threads)
	if err != nil {
		return err
	}
	defer head.Stop()
	placed := head.Blocks()
	for i := range placed {
		if addr, ok := holderOf[placed[i].Machine]; ok {
			placed[i].Machine = addr
		}
	}
	if gone := absent(placed, addrs); len(gone) > 0 {
		return fmt.Errorf("these block servers did not answer, the model was not split: %s", strings.Join(gone, ", "))
	}
	if *asJSON {
		total := 0
		for _, b := range placed {
			total += b.Layers()
		}
		type block struct {
			Machine string `json:"machine"`
			First   int    `json:"first"`
			Last    int    `json:"last"`
			Share   int    `json:"share"`            // percent of the model's layers
			Blocks  string `json:"blocks,omitempty"` // the blocks it announced, like 4-9
			Route   string `json:"route,omitempty"`  // how it is reached: its address, or relay:host:port
		}
		out := struct {
			URL    string  `json:"url"`
			Blocks []block `json:"blocks"`
			Of     int     `json:"model_blocks,omitempty"` // how many blocks the model has
		}{URL: head.URL, Of: len(modelBlocks)}
		for _, b := range placed {
			announced := ""
			for _, h := range holders {
				if h.Addr == b.Machine {
					announced = fmt.Sprintf("%d-%d", h.First, h.Last)
				}
			}
			out.Blocks = append(out.Blocks, block{b.Machine, b.First, b.Last, 100 * b.Layers() / max(total, 1), announced, routes[b.Machine]})
		}
		json.NewEncoder(os.Stdout).Encode(out)
	} else {
		fmt.Println()
		printBlocks(os.Stdout, placed)
		fmt.Printf("\nReady. OpenAI-compatible API: %s/v1   Stop with Ctrl+C.\n", head.URL)
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	if *withStdin {
		go func() {
			io.Copy(io.Discard, os.Stdin)
			stop <- os.Interrupt
		}()
	}
	select {
	case <-stop:
		return nil
	case <-head.exited:
		return fmt.Errorf("the model stopped: a block server may have left\n%s", head.tail())
	}
}
