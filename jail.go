package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// A block server obeys the computing orders of another machine, in a C++ program with no login.
// The lock is what makes that safe for strangers: the program can read and run the llama.cpp
// folder, write its block cache, listen on one port of this machine, and nothing else. No
// personal file, no outgoing connection, no other program.
//
//	Windows: an AppContainer with no capability and a job limited to two processes and in memory.
//	Linux:   Landlock for files and ports, seccomp for sockets, a limit on the memory it may ask for.
//	macOS:   a sandbox-exec profile, and a guard that ends it when it holds too much memory.
type Jail struct {
	Run      []string // folders it may read and execute
	Write    string   // the one folder it may write
	Port     int      // the one port it may listen on
	MemoryMB int      // most memory it may use, 0 for no limit
}

// jailProc is a locked-in program that is running.
type jailProc struct {
	Addr string        // where to reach it
	Log  string        // what it printed
	Done chan struct{} // closed when it has stopped
	kill func()
}

func (p *jailProc) Stop() {
	p.kill()
	select {
	case <-p.Done:
	case <-time.After(5 * time.Second):
	}
}

// jailHome returns the two folders of the lock: the one that holds a copy of this program, which
// the locked side may run, and the one it may write.
func jailHome() (run, write string, err error) {
	home, err := homeDir()
	if err != nil {
		return "", "", err
	}
	run, write = filepath.Join(home, "jail"), filepath.Join(home, "blocks")
	for _, d := range []string{run, write} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return "", "", err
		}
	}
	return run, write, nil
}

// selfCopy puts a copy of this program in the lock's folder and returns its path. The locked side
// runs that copy: it never needs to see the folder this program was started from.
func selfCopy(runDir string) (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	dst := filepath.Join(runDir, exe("tor6ia-engine"))
	if a, err1 := os.Stat(self); err1 == nil {
		if b, err2 := os.Stat(dst); err2 == nil && a.Size() == b.Size() && !a.ModTime().After(b.ModTime()) {
			return dst, nil
		}
	}
	in, err := os.Open(self)
	if err != nil {
		return "", err
	}
	defer in.Close()
	tmp := dst + ".new"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return "", err
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	return dst, os.Rename(tmp, dst)
}

// waitPort waits until something accepts connections at addr, or until the program stops.
func waitPort(addr string, done <-chan struct{}) error {
	for deadline := time.Now().Add(60 * time.Second); time.Now().Before(deadline); {
		if c, err := net.DialTimeout("tcp", addr, time.Second); err == nil {
			c.Close()
			return nil
		}
		select {
		case <-done:
			return errors.New("the program stopped before it listened")
		case <-time.After(150 * time.Millisecond):
		}
	}
	return errors.New("the program did not listen within a minute")
}

// runBridge is the small relay that runs inside the lock on Windows: it starts the program next
// to it, then carries the engine's connections to the program's port over its own input and output.
func runBridge(args []string) error {
	fs := flag.NewFlagSet("bridge", flag.ExitOnError)
	port := fs.Int("port", 0, "port the program listens on, 0 when it does not listen")
	logPath := fs.String("log", "", "file that receives what the program prints")
	fs.Parse(args)
	if fs.NArg() == 0 {
		return errors.New("bridge: no program")
	}
	logFile, err := os.OpenFile(*logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	cmd := exec.Command(fs.Arg(0), fs.Args()[1:]...)
	cmd.Dir = filepath.Dir(fs.Arg(0))
	cmd.Stdout, cmd.Stderr = logFile, logFile
	// An empty input of our own: by default Go opens the NUL device for it, which a container
	// is refused on Windows Server.
	cmd.Stdin = strings.NewReader("")
	quiet(cmd)
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(logFile, "bridge: cannot start the program:", err)
		return err
	}
	stopped := make(chan struct{})
	go func() { cmd.Wait(); close(stopped) }()

	m := newMux(os.Stdout)
	addr := "127.0.0.1:" + strconv.Itoa(*port)
	if *port != 0 {
		if err := waitPort(addr, stopped); err != nil {
			cmd.Process.Kill()
			return err
		}
		m.send(0, frameReady, nil)
	}
	go func() {
		// The engine closed our input: it has gone, or it wants us gone.
		m.run(os.Stdin, func() (net.Conn, error) { return net.DialTimeout("tcp", addr, 5*time.Second) })
		cmd.Process.Kill()
	}()
	<-stopped
	return nil
}

// memoryAttempt is the one attempt the lock answers by stopping the program instead of refusing.
const memoryAttempt = "use 1 GB of memory"

// runProbe is the escape test. Started inside a lock in place of a block server, it tries the
// things the lock must forbid and prints one line for each.
func runProbe(args []string) error {
	if len(args) < 1 {
		return errors.New("probe: no file to read")
	}
	say := func(what string, err error) {
		verdict := "ALLOWED"
		if err != nil {
			verdict = "refused"
		}
		fmt.Printf("%s: %s\n", what, verdict)
	}
	_, err := os.ReadFile(args[0])
	say("read a personal file", err)
	conn, err := net.DialTimeout("tcp", "1.1.1.1:443", 4*time.Second)
	if err == nil {
		conn.Close()
	}
	say("connect to the Internet", err)
	other := exec.Command("/bin/sh", "-c", "true")
	if runtime.GOOS == "windows" {
		other = exec.Command("cmd", "/c", "exit 0")
	}
	other.Stdin = strings.NewReader("") // so that a refusal comes from the lock, not from a missing NUL device
	say("start another program", other.Run())
	// Last, because a lock stops the program here.
	hog := make([]byte, 1<<30)
	for i := 0; i < len(hog); i += 4096 {
		hog[i] = 1
	}
	time.Sleep(4 * time.Second) // a lock that watches from outside needs a moment
	say(memoryAttempt, nil)
	runtime.KeepAlive(hog)
	return nil
}

// blockServers counts the block servers running on this machine.
func blockServers() int {
	cmd := exec.Command("pgrep", "-x", "ggml-rpc-server")
	if runtime.GOOS == "windows" {
		cmd = exec.Command("tasklist", "/FI", "IMAGENAME eq ggml-rpc-server.exe", "/FO", "CSV", "/NH")
	}
	out, _ := cmd.Output()
	n := 0
	for _, line := range strings.Split(string(out), "\n") {
		if runtime.GOOS != "windows" && strings.TrimSpace(line) != "" || strings.Contains(line, "ggml-rpc-server") {
			n++
		}
	}
	return n
}

// killTest starts an engine with a block server, kills the engine by force, and checks that the
// block server stopped with it.
func killTest() error {
	if _, err := ensureLlama(); err != nil {
		return err
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	port, err := freePort()
	if err != nil {
		return err
	}
	before := blockServers()
	engine := exec.Command(self, "block", "--listen", "127.0.0.1:"+strconv.Itoa(port))
	if err := engine.Start(); err != nil {
		return err
	}
	stopped := make(chan struct{})
	go func() { engine.Wait(); close(stopped) }()
	if err := waitPort("127.0.0.1:"+strconv.Itoa(port), stopped); err != nil {
		engine.Process.Kill()
		return fmt.Errorf("the engine to kill did not start: %w", err)
	}
	if blockServers() != before+1 {
		engine.Process.Kill()
		return errors.New("cannot count the block servers on this system")
	}
	engine.Process.Kill() // by force: the engine gets no chance to stop anything itself
	<-stopped
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(500 * time.Millisecond) {
		if blockServers() == before {
			return nil
		}
	}
	return errors.New("the block server outlived its engine")
}

// jailTest runs the probe twice: free, to show it can do everything here, then locked in with
// 256 MB of memory. It fails when the lock lets one attempt through. Then it kills an engine by
// force and checks that its block server stopped too.
func jailTest() error {
	runDir, writeDir, err := jailHome()
	if err != nil {
		return err
	}
	self, err := selfCopy(runDir)
	if err != nil {
		return err
	}
	home, _ := os.UserHomeDir()
	secret := filepath.Join(home, ".tor6ia-engine-secret-test")
	if err := os.WriteFile(secret, []byte("personal"), 0o600); err != nil {
		return err
	}
	defer os.Remove(secret)

	free, _ := exec.Command(self, "probe", secret).Output()
	p, err := startJailed(Jail{Run: []string{runDir}, Write: writeDir, MemoryMB: 256}, "", self, []string{"probe", secret})
	if err != nil {
		return fmt.Errorf("cannot lock a program in on this system: %w", err)
	}
	select {
	case <-p.Done:
	case <-time.After(60 * time.Second):
		p.Stop()
		return errors.New("the locked probe did not finish")
	}
	raw, _ := os.ReadFile(p.Log)
	if !strings.Contains(string(raw), "read a personal file: ") {
		// The probe did not answer: show what was written instead, and what the lock itself reported.
		fmt.Printf("the locked probe did not answer. What was written in its place:\n%s\n", raw)
		if said, err := os.ReadFile(p.Log + ".bridge"); err == nil {
			fmt.Printf("The lock reported:\n%s\n", said)
		}
	}
	locked := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		if what, verdict, ok := strings.Cut(strings.TrimSpace(line), ": "); ok {
			locked[what] = verdict
		}
	}
	failed := false
	fmt.Printf("%-28s %-10s %s\n", "ATTEMPT", "FREE", "LOCKED IN")
	for _, line := range strings.Split(strings.TrimSpace(string(free)), "\n") {
		what, verdict, _ := strings.Cut(strings.TrimSpace(line), ": ")
		got := locked[what]
		switch {
		case got == "" && what == memoryAttempt && len(locked) > 0:
			got = "stopped" // the other attempts answered, then the lock ended the program
		case got == "":
			got = "no answer"
		}
		fmt.Printf("%-28s %-10s %s\n", what, verdict, got)
		failed = failed || got != "refused" && got != "stopped"
	}
	if failed || len(locked) == 0 {
		return errors.New("the lock let something through")
	}
	err = killTest()
	verdict := "stopped with it"
	if err != nil {
		verdict = "FAILED"
	}
	fmt.Printf("%-28s %-10s %s\n", "kill the engine by force", "", "block server "+verdict)
	return err
}
