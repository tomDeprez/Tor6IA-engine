// tor6ia-engine runs one language model across several machines: the model is cut into
// blocks of layers and each machine holds the blocks its memory allows.
package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strings"
)

var version = "dev"

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
		fs.Parse(os.Args[2:])
		if *machines < 1 || *machines > 16 {
			err = fmt.Errorf("--machines must be between 1 and 16")
			break
		}
		err = demo(*model, *machines, *prompt)
	case "block":
		err = runBlock(os.Args[2:])
	case "head":
		err = runHead(os.Args[2:])
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
	fs.Parse(args)
	dir, err := ensureLlama()
	if err != nil {
		return err
	}
	cmd, err := blockCommand(dir, *listen, *threads)
	if err != nil {
		return err
	}
	if host, _, _ := net.SplitHostPort(*listen); host != "127.0.0.1" && host != "localhost" && host != "::1" {
		fmt.Println("Warning: a block server lets any machine that can reach this port run computations inside this process.")
		fmt.Println("It has no login and no encryption. Open it only on a private network or a VPN, to machines you trust.")
	}
	fmt.Printf("Block server on %s, %d processors. Stop it with Ctrl+C.\n", *listen, *threads)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

func runHead(args []string) error {
	fs := flag.NewFlagSet("head", flag.ExitOnError)
	model := fs.String("model", "", "GGUF file, or a model Ollama has")
	blocks := fs.String("blocks", "", "block servers, as host:port,host:port")
	port := fs.Int("port", 8090, "local port of the OpenAI-compatible API")
	threads := fs.Int("threads", max(1, runtime.NumCPU()/2), "processors to use")
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
	fmt.Printf("Loading %s across %s...\n", *model, *blocks)
	head, err := startHead(dir, file, strings.Split(*blocks, ","), *port, *threads)
	if err != nil {
		return err
	}
	defer head.Stop()
	fmt.Println()
	printBlocks(os.Stdout, head.Blocks())
	fmt.Printf("\nReady. OpenAI-compatible API: %s/v1   Stop with Ctrl+C.\n", head.URL)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	return nil
}
