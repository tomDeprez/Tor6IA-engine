// tor6ia-engine runs one language model across several machines: the model is cut into
// blocks of layers and each machine holds the blocks its memory allows.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strings"
)

var version = "dev"

const unlockedHelp = "run block servers with every right of your account instead of locked in: only for machines you trust, or to debug"

const deviceHelp = `what computes on a block server: empty for llama.cpp's choice, CPU for the processor (use it if a block server stops with "unsupported op")`

const usage = `tor6ia-engine: one model, several machines

  tor6ia-engine demo  [--model gemma4:e2b] [--machines 3]    try it on this machine alone
  tor6ia-engine block [--listen 127.0.0.1:50052]              lend this machine to a head
  tor6ia-engine head  --model M --blocks host:port,host:port  run a model across block servers
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
	case "jail-test":
		err = jailTest()
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
	listen := fs.String("listen", "127.0.0.1:50052", "address the block server listens on")
	threads := fs.Int("threads", max(1, runtime.NumCPU()/2), "processors to use")
	device := fs.String("device", "", deviceHelp)
	unlocked := fs.Bool("unlocked", false, unlockedHelp)
	fs.Parse(args)
	dir, err := ensureLlama()
	if err != nil {
		return err
	}
	block, err := startBlock(dir, *listen, *threads, *device, !*unlocked)
	if err != nil {
		return err
	}
	defer block.Stop()
	how := "locked in: it cannot read your files, connect out or start programs"
	if *unlocked {
		how = "NOT locked in: it has every right of your account"
	}
	fmt.Printf("Block server on %s, %d processors, %s.\n", block.Addr, *threads, how)
	if host, _, _ := net.SplitHostPort(*listen); host != "127.0.0.1" && host != "localhost" && host != "::1" {
		fmt.Println("It has no login and no encryption: any machine that can reach this port can give it work.")
	}
	fmt.Println("Stop it with Ctrl+C.")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	select {
	case <-stop:
	case <-block.Done:
		return fmt.Errorf("the block server stopped by itself\n%s", tailFile(block.Log, 12))
	}
	return nil
}

func runHead(args []string) error {
	fs := flag.NewFlagSet("head", flag.ExitOnError)
	model := fs.String("model", "", "GGUF file, or a model Ollama has")
	blocks := fs.String("blocks", "", "block servers, as host:port,host:port")
	port := fs.Int("port", 8090, "local port of the OpenAI-compatible API")
	threads := fs.Int("threads", max(1, runtime.NumCPU()/2), "processors to use")
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
	fmt.Fprintf(os.Stderr, "Loading %s across %s...\n", *model, *blocks)
	head, err := startHead(dir, file, strings.Split(*blocks, ","), *port, *threads)
	if err != nil {
		return err
	}
	defer head.Stop()
	placed := head.Blocks()
	if gone := absent(placed, strings.Split(*blocks, ",")); len(gone) > 0 {
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
			Share   int    `json:"share"` // percent of the model's layers
		}
		out := struct {
			URL    string  `json:"url"`
			Blocks []block `json:"blocks"`
		}{URL: head.URL}
		for _, b := range placed {
			out.Blocks = append(out.Blocks, block{b.Machine, b.First, b.Last, 100 * b.Layers() / max(total, 1)})
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
