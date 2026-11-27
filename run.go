package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// command prepares one llama.cpp program. Its libraries sit next to it.
func command(dir, name string, args ...string) *exec.Cmd {
	cmd := exec.Command(filepath.Join(dir, exe(name)), args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "LD_LIBRARY_PATH="+dir, "DYLD_LIBRARY_PATH="+dir)
	return cmd
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// startBlock starts one block server: it lends this machine's memory and processor to a head.
// listen is where the head reaches it. device chooses what computes: empty lets llama.cpp decide
// (the graphics card when there is one), "CPU" forces the processor.
//
// locked runs it inside the lock described in jail.go, which is the only safe way to serve
// machines that are not trusted. Unlocked, it has every right of the user who started it.
// memoryMB is the most memory it may use, 0 for no limit (the limit is enforced on Windows).
func startBlock(dir, listen string, threads int, device string, locked bool, memoryMB int, cache bool) (*jailProc, error) {
	host, portText, err := net.SplitHostPort(listen)
	port, perr := strconv.Atoi(portText)
	if err != nil || perr != nil {
		return nil, fmt.Errorf("bad address %q: want host:port", listen)
	}
	// -c makes it load from the block cache the layers it finds there, instead of over the network.
	// It is for a machine that fetched its blocks first: without them, the cache would only grow.
	args := func(host string, port int) []string {
		a := []string{"-H", host, "-p", strconv.Itoa(port), "-t", strconv.Itoa(threads)}
		if cache {
			a = append(a, "-c")
		}
		if device != "" {
			a = append(a, "-d", device)
		}
		return a
	}
	program := filepath.Join(dir, exe("ggml-rpc-server"))
	if locked {
		_, writeDir, err := jailHome()
		if err != nil {
			return nil, err
		}
		inner := port
		if runtime.GOOS == "windows" {
			// Its container cannot be reached from outside: it listens on a port of its own, and
			// the engine carries the connections from listen through the bridge.
			if inner, err = freePort(); err != nil {
				return nil, err
			}
			host = "127.0.0.1"
		}
		return startJailed(Jail{Run: []string{dir}, Write: writeDir, Port: inner, MemoryMB: memoryMB}, listen, program, args(host, inner))
	}
	logFile, err := os.CreateTemp("", "tor6ia-engine-block-*.log")
	if err != nil {
		return nil, err
	}
	cmd := command(dir, "ggml-rpc-server", args(host, port)...)
	// Locked in or not, block servers keep their tensors in the engine's own folder.
	if _, writeDir, err := jailHome(); err == nil {
		cmd.Env = append(cmd.Env, "LLAMA_CACHE="+writeDir)
	}
	cmd.Stdout, cmd.Stderr = logFile, logFile
	if err := launch(cmd); err != nil {
		return nil, fmt.Errorf("block server: %w", err)
	}
	p := &jailProc{Addr: listen, Log: logFile.Name(), Done: make(chan struct{}), kill: func() { cmd.Process.Kill() }}
	go func() { cmd.Wait(); close(p.Done) }()
	if err := waitPort(dialable(listen), p.Done); err != nil {
		p.Stop()
		return nil, fmt.Errorf("block server: %w\n%s", err, tailFile(p.Log, 8))
	}
	return p, nil
}

// dialable turns a listening address into one that can be dialled from this machine.
func dialable(listen string) string {
	host, port, _ := net.SplitHostPort(listen)
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, port)
}

// defaultContext is the conversation length the head reserves memory for, in tokens.
const defaultContext = 8192

// Head is the machine that holds the conversation: it runs the model across the block servers
// and serves an OpenAI-compatible API.
type Head struct {
	cmd    *exec.Cmd
	URL    string
	log    string
	exited chan error // receives once, when llama-server stops by itself
}

// shares, when given, says which part of the model each block server takes, in the order of
// blocks: 1,3,2 gives the second one half of the layers. Empty lets llama.cpp divide by free memory.
// context is how many tokens a conversation may hold. Every block server reserves memory for it.
func startHead(dir, model string, blocks []string, shares string, context, port, threads int) (*Head, error) {
	logFile, err := os.CreateTemp("", "tor6ia-engine-head-*.log")
	if err != nil {
		return nil, err
	}
	args := []string{"-m", model, "--host", "127.0.0.1", "--port", strconv.Itoa(port), "-t", strconv.Itoa(threads),
		// Without this llama.cpp reserves the largest context the model was trained for: 262 144 tokens
		// for Gemma 4, gigabytes of memory on every block server for conversations nobody has.
		"-c", strconv.Itoa(context),
		"--no-webui", "-v", // -v prints which machine receives each layer
		"--reasoning", "off"} // answer directly: thinking text would cross every machine for nothing
	if len(blocks) > 0 {
		// Every layer goes to the block servers. The word table stays here: it is what turns the
		// last state into one score per word, and a block that held it would send back a megabyte
		// per token for a 262 000-word vocabulary. Kept here, a token costs a few hundred kilobytes
		// (measured on Gemma 4 12B: 1 300 KB before, 350 KB after, with two blocks).
		args = append(args, "--rpc", strings.Join(blocks, ","), "-ngl", "999",
			"--override-tensor", `^token_embd\.weight$=CPU,^output\.weight$=CPU`)
		if shares != "" {
			args = append(args, "--tensor-split", shares)
		}
	}
	cmd := command(dir, "llama-server", args...)
	cmd.Stdout, cmd.Stderr = logFile, logFile
	if err := launch(cmd); err != nil { // the model stops with the engine, even when the engine is killed by force
		return nil, err
	}
	exited := make(chan error, 1)
	h := &Head{cmd: cmd, URL: "http://127.0.0.1:" + strconv.Itoa(port), log: logFile.Name(), exited: exited}
	go func() { exited <- cmd.Wait() }()
	for deadline := time.Now().Add(30 * time.Minute); time.Now().Before(deadline); {
		select {
		case <-exited:
			return nil, fmt.Errorf("the head stopped while loading the model, see %s\n%s", h.log, h.tail())
		case <-time.After(500 * time.Millisecond):
		}
		if resp, err := http.Get(h.URL + "/health"); err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return h, nil
			}
		}
	}
	h.Stop()
	return nil, errors.New("the model did not finish loading in 30 minutes")
}

func (h *Head) tail() string { return tailFile(h.log, 8) }

func tailFile(path string, n int) string {
	raw, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	return strings.Join(lines[max(0, len(lines)-n):], "\n")
}

func (h *Head) Blocks() []Block {
	raw, _ := os.ReadFile(h.log)
	return parseBlocks(string(raw))
}

func (h *Head) Stop() {
	h.cmd.Process.Kill()
	os.Remove(h.log)
}

// Ask sends one question and returns the answer with the generation speed in tokens per second.
func (h *Head) Ask(prompt string, maxTokens int) (string, float64, error) {
	body, _ := json.Marshal(map[string]any{
		"messages": []map[string]string{{"role": "user", "content": prompt}}, "max_tokens": maxTokens})
	resp, err := http.Post(h.URL+"/v1/chat/completions", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Timings struct {
			PerSecond float64 `json:"predicted_per_second"`
		} `json:"timings"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", 0, err
	}
	if len(out.Choices) == 0 {
		return "", 0, errors.New("the model did not answer: " + out.Error.Message)
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), out.Timings.PerSecond, nil
}

// demo shows the whole idea on one machine: it starts several block servers, splits the model
// across them, prints who holds which layers and asks one question.
func demo(model string, machines int, prompt, device string, locked bool) error {
	dir, err := ensureLlama()
	if err != nil {
		return err
	}
	file, err := resolveModel(model)
	if err != nil {
		return err
	}
	threads := max(1, runtime.NumCPU()/2) // one block computes at a time: each may use half of the processors
	// Any block server may receive most of the model: each is capped as if it held it all.
	memoryMB := 0
	if st, err := os.Stat(file); err == nil {
		memoryMB = memoryCap(st.Size())
	}

	var addrs, blockLogs []string
	for i := 0; i < machines; i++ {
		port, err := freePort()
		if err != nil {
			return err
		}
		block, err := startBlock(dir, "127.0.0.1:"+strconv.Itoa(port), threads, device, locked, memoryMB, false)
		if err != nil {
			return err
		}
		defer block.Stop()
		addrs = append(addrs, block.Addr)
		blockLogs = append(blockLogs, block.Log)
	}
	how := "locked in"
	if !locked {
		how = "not locked in"
	}
	fmt.Printf("%d block servers started, %s. Loading %s across them...\n", machines, how, model)

	port, err := freePort()
	if err != nil {
		return err
	}
	head, err := startHead(dir, file, addrs, "", defaultContext, port, threads)
	if err != nil {
		// The head usually fails because a block server did: show what each one last said.
		for i, path := range blockLogs {
			fmt.Fprintf(os.Stderr, "--- block server %d (%s)\n%s\n", i+1, addrs[i], tailFile(path, 12))
		}
		return err
	}
	defer head.Stop()

	blocks := head.Blocks()
	if gone := absent(blocks, addrs); len(gone) > 0 {
		return fmt.Errorf("these block servers hold no layer of the model: %s", strings.Join(gone, ", "))
	}
	fmt.Println()
	printBlocks(os.Stdout, blocks)
	fmt.Printf("\nQuestion: %s\n", prompt)
	answer, speed, err := head.Ask(prompt, 96)
	if err != nil {
		return err
	}
	fmt.Printf("Answer:   %s\n\n%.1f tokens per second, with every layer computed by a block server.\n", answer, speed)
	return nil
}
