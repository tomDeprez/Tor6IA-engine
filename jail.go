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
//	Windows: an AppContainer with no capability and a job limited to two processes.
//	Linux:   Landlock for files and ports, seccomp for sockets.
//	macOS:   a sandbox-exec profile.
type Jail struct {
	Run      []string // folders it may read and execute
	Write    string   // the one folder it may write
	Port     int      // the one port it may listen on
	MemoryMB int      // most memory it may use, 0 for no limit (enforced on Windows)
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

// runProbe is the escape test. Started inside a lock in place of a block server, it tries the three
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
	return nil
}

// jailTest runs the probe twice: free, to show it can do the three things here, then locked in.
// It fails when the lock lets one of them through.
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
	p, err := startJailed(Jail{Run: []string{runDir}, Write: writeDir}, "", self, []string{"probe", secret})
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
		if got == "" {
			got = "no answer"
		}
		fmt.Printf("%-28s %-10s %s\n", what, verdict, got)
		failed = failed || got != "refused"
	}
	if failed || len(locked) == 0 {
		return errors.New("the lock let something through")
	}
	return nil
}
