//go:build darwin

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// launch starts a program. macOS has no way to end a child when its parent is killed by force:
// locked-in programs get a guard for that (see startJailed).
func launch(cmd *exec.Cmd) error { return cmd.Start() }

// startJailed starts a program under sandbox-exec. The profile allows what a program normally
// needs to run, then takes away the three things that matter: the user's files, outgoing
// connections, and starting any program outside its own folders.
func startJailed(j Jail, front, program string, args []string) (*jailProc, error) {
	runDir, _, err := jailHome()
	if err != nil {
		return nil, err
	}
	quote := func(path string) (string, error) {
		real, err := filepath.EvalSymlinks(path)
		if err != nil {
			return "", err
		}
		if strings.ContainsAny(real, "\"\\") {
			return "", errors.New("this folder name cannot be used in a sandbox profile: " + real)
		}
		return `(subpath "` + real + `")`, nil
	}
	var runnable []string
	for _, dir := range append([]string{runDir}, j.Run...) {
		q, err := quote(dir)
		if err != nil {
			return nil, err
		}
		runnable = append(runnable, q)
	}
	writable, err := quote(j.Write)
	if err != nil {
		return nil, err
	}
	profile := `(version 1)
(allow default)
(deny network-outbound)
(deny file-read* file-write* (subpath "/Users") (subpath "/Volumes") (subpath "/private/var/root"))
(allow file-read-metadata)
(allow file-read* ` + strings.Join(runnable, " ") + ` ` + writable + `)
(allow file-write* ` + writable + `)
(deny process-exec*)
(allow process-exec* ` + strings.Join(runnable, " ") + `)`

	cmd := exec.Command("/usr/bin/sandbox-exec", append([]string{"-p", profile, program}, args...)...)
	logPath := filepath.Join(j.Write, "block-"+strconv.Itoa(j.Port)+".log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, err
	}
	defer logFile.Close()
	cmd.Stdout, cmd.Stderr = logFile, logFile
	cmd.Dir = filepath.Dir(program)
	cmd.Env = append(os.Environ(), "LLAMA_CACHE="+j.Write, "TMPDIR="+j.Write, "DYLD_LIBRARY_PATH="+filepath.Dir(program))
	if err := launch(cmd); err != nil {
		return nil, err
	}
	// macOS can neither end a program when the one that started it is killed by force, nor cap
	// its memory. This small loop does both: it ends the program when the engine has gone, or
	// when the program holds more memory than it was given.
	guard := exec.Command("/bin/sh", "-c",
		`while kill -0 $1 2>/dev/null && kill -0 $2 2>/dev/null; do [ $3 -gt 0 ] && [ "$(ps -o rss= -p $2 | tr -d ' ')" -gt $3 ] 2>/dev/null && break; sleep 1; done; kill -9 $2 2>/dev/null`,
		"guard", strconv.Itoa(os.Getpid()), strconv.Itoa(cmd.Process.Pid), strconv.Itoa(j.MemoryMB<<10))
	if guard.Start() == nil {
		go guard.Wait()
	}
	p := &jailProc{Addr: front, Log: logPath, Done: make(chan struct{}), kill: func() { cmd.Process.Kill() }}
	go func() { cmd.Wait(); close(p.Done) }()
	if j.Port != 0 {
		if err := waitPort(dialable(front), p.Done); err != nil {
			p.Stop()
			return nil, fmt.Errorf("%w\n%s", err, tailFile(logPath, 8))
		}
	}
	return p, nil
}
