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

// blockCommand is one block server: it lends this machine's memory and processor to a head.
// device chooses what computes: empty lets llama.cpp decide (the graphics card when there is one),
// "CPU" forces the processor.
func blockCommand(dir, listen string, threads int, device string) (*exec.Cmd, error) {
	host, port, err := net.SplitHostPort(listen)
	if err != nil {
		return nil, fmt.Errorf("bad address %q: want host:port", listen)
	}
	// -c keeps the layers it received on disk: the next start loads them locally instead of over the network.
	args := []string{"-H", host, "-p", port, "-t", strconv.Itoa(threads), "-c"}
	if device != "" {
		args = append(args, "-d", device)
	}
	return command(dir, "ggml-rpc-server", args...), nil
}

// Head is the machine that holds the conversation: it runs the model across the block servers
// and serves an OpenAI-compatible API.
type Head struct {
	cmd *exec.Cmd
	URL string
	log string
}

func startHead(dir, model string, blocks []string, port, threads int) (*Head, error) {
	logFile, err := os.CreateTemp("", "tor6ia-engine-head-*.log")
	if err != nil {
		return nil, err
	}
	args := []string{"-m", model, "--host", "127.0.0.1", "--port", strconv.Itoa(port), "-t", strconv.Itoa(threads),
		"--no-webui", "-v", // -v prints which machine receives each layer
		"--reasoning-budget", "0"} // answer directly: thinking text would cross every machine for nothing
	if len(blocks) > 0 {
		args = append(args, "--rpc", strings.Join(blocks, ","), "-ngl", "999") // every layer goes to the block servers
	}
	cmd := command(dir, "llama-server", args...)
	cmd.Stdout, cmd.Stderr = logFile, logFile
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	h := &Head{cmd: cmd, URL: "http://127.0.0.1:" + strconv.Itoa(port), log: logFile.Name()}
	exited := make(chan error, 1)
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
func demo(model string, machines int, prompt, device string) error {
	dir, err := ensureLlama()
	if err != nil {
		return err
	}
	file, err := resolveModel(model)
	if err != nil {
		return err
	}
	threads := max(1, runtime.NumCPU()/2) // one block computes at a time: each may use half of the processors

	var addrs, blockLogs []string
	for i := 0; i < machines; i++ {
		port, err := freePort()
		if err != nil {
			return err
		}
		addr := "127.0.0.1:" + strconv.Itoa(port)
		cmd, err := blockCommand(dir, addr, threads, device)
		if err != nil {
			return err
		}
		logFile, err := os.CreateTemp("", "tor6ia-engine-block-*.log")
		if err != nil {
			return err
		}
		defer os.Remove(logFile.Name())
		cmd.Stdout, cmd.Stderr = logFile, logFile
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("block server: %w", err)
		}
		defer cmd.Process.Kill()
		addrs = append(addrs, addr)
		blockLogs = append(blockLogs, logFile.Name())
	}
	fmt.Printf("%d block servers started. Loading %s across them...\n", machines, model)
	time.Sleep(time.Second) // let the block servers open their ports

	port, err := freePort()
	if err != nil {
		return err
	}
	head, err := startHead(dir, file, addrs, port, threads)
	if err != nil {
		// The head usually fails because a block server did: show what each one last said.
		for i, path := range blockLogs {
			fmt.Fprintf(os.Stderr, "--- block server %d (%s)\n%s\n", i+1, addrs[i], tailFile(path, 12))
		}
		return err
	}
	defer head.Stop()

	blocks := head.Blocks()
	if len(blocks) == 0 {
		return errors.New("the head loaded the model but reported no layer placement")
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
