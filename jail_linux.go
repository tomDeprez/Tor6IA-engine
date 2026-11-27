//go:build linux

package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

// keeper is one thread that never ends. A child started with Pdeathsig is killed when the thread
// that started it ends, so children are started from this one and die only when the engine does.
var keeper = make(chan func())

func init() {
	go func() {
		runtime.LockOSThread()
		for f := range keeper {
			f()
		}
	}()
}

// launch starts a program that stops with the engine, even when the engine is killed by force.
func launch(cmd *exec.Cmd) error {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Pdeathsig = syscall.SIGKILL
	done := make(chan error)
	keeper <- func() { done <- cmd.Start() }
	return <-done
}

// startJailed starts a program through "jail-exec", which locks itself in and then becomes the program.
func startJailed(j Jail, front, program string, args []string) (*jailProc, error) {
	runDir, _, err := jailHome()
	if err != nil {
		return nil, err
	}
	self, err := selfCopy(runDir)
	if err != nil {
		return nil, err
	}
	a := []string{"jail-exec", "--write", j.Write, "--port", strconv.Itoa(j.Port)}
	for _, dir := range append([]string{runDir}, j.Run...) {
		a = append(a, "--run", dir)
	}
	cmd := exec.Command(self, append(append(a, program), args...)...)
	logPath := filepath.Join(j.Write, "block-"+strconv.Itoa(j.Port)+".log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, err
	}
	defer logFile.Close()
	cmd.Stdout, cmd.Stderr = logFile, logFile
	cmd.Dir = filepath.Dir(program)
	cmd.Env = append(os.Environ(), "LLAMA_CACHE="+j.Write, "TMPDIR="+j.Write, "LD_LIBRARY_PATH="+filepath.Dir(program))
	if err := launch(cmd); err != nil {
		return nil, err
	}
	p := &jailProc{Addr: front, Log: logPath, Done: make(chan struct{}), kill: func() { cmd.Process.Kill() }}
	go func() { cmd.Wait(); close(p.Done) }()
	if j.Port != 0 {
		if err := waitPort(dialable(front), p.Done); err != nil {
			p.Stop()
			said := tailFile(logPath, 8)
			if strings.Contains(said, "libgomp") {
				said += "\nThis system lacks the OpenMP library llama.cpp needs. On Debian and Ubuntu: sudo apt install libgomp1"
			}
			return nil, fmt.Errorf("%w\n%s", err, said)
		}
	}
	return p, nil
}

type dirList []string

func (d *dirList) String() string     { return fmt.Sprint(*d) }
func (d *dirList) Set(v string) error { *d = append(*d, v); return nil }

// runJailExec locks this process in, then replaces it with the program. Both locks are kept across
// the replacement and by anything the program starts.
func runJailExec(args []string) error {
	fs := flag.NewFlagSet("jail-exec", flag.ExitOnError)
	var run dirList
	fs.Var(&run, "run", "folder the program may read and execute")
	write := fs.String("write", "", "folder the program may write")
	port := fs.Int("port", 0, "port the program may listen on")
	fs.Parse(args)
	if fs.NArg() == 0 {
		return errors.New("jail-exec: no program")
	}
	runtime.LockOSThread() // the locks belong to this thread, which becomes the program
	if _, _, e := syscall.Syscall6(syscall.SYS_PRCTL, 38 /* PR_SET_NO_NEW_PRIVS */, 1, 0, 0, 0, 0); e != 0 {
		return fmt.Errorf("no_new_privs: %v", e)
	}
	if err := lockFiles(run, *write, *port); err != nil {
		return err
	}
	if err := lockSockets(); err != nil {
		return err
	}
	return syscall.Exec(fs.Arg(0), fs.Args(), os.Environ())
}

// Landlock access rights for files (linux/landlock.h).
const (
	llExecute  = 1 << 0
	llWrite    = 1 << 1
	llReadFile = 1 << 2
	llReadDir  = 1 << 3
	llTruncate = 1 << 14
	llIoctlDev = 1 << 15
	llOnFiles  = llExecute | llWrite | llReadFile | llTruncate | llIoctlDev // the rights that make sense on a file

	sysLandlockCreateRuleset = 444
	sysLandlockAddRule       = 445
	sysLandlockRestrictSelf  = 446
	oPath                    = 0x200000
)

// lockFiles applies Landlock: the program reads the system's libraries, reads and runs its own
// folders, writes one folder, and sees nothing else. With kernel 6.7 or newer it also may listen
// on one port only and connect nowhere.
func lockFiles(run []string, write string, port int) error {
	abi, _, e := syscall.Syscall(sysLandlockCreateRuleset, 0, 0, 1 /* ask for the version */)
	if e != 0 {
		return fmt.Errorf("this Linux kernel cannot lock a program in: Landlock is missing (kernel 5.13 or newer is needed): %v", e)
	}
	all := uint64(1<<13 - 1)
	if abi >= 2 {
		all |= 1 << 13
	}
	if abi >= 3 {
		all |= llTruncate
	}
	if abi >= 5 {
		all |= llIoctlDev
	}
	ruleset := struct{ files, network uint64 }{files: all}
	size := uintptr(8)
	if abi >= 4 {
		ruleset.network, size = 3, 16 // bind and connect, TCP
	}
	fd, _, e := syscall.Syscall(sysLandlockCreateRuleset, uintptr(unsafe.Pointer(&ruleset)), size, 0)
	if e != 0 {
		return fmt.Errorf("landlock: %v", e)
	}
	allow := func(path string, rights uint64) {
		f, err := syscall.Open(path, oPath|syscall.O_CLOEXEC, 0)
		if err != nil {
			return // not on this system
		}
		defer syscall.Close(f)
		var st syscall.Stat_t
		if syscall.Fstat(f, &st) == nil && st.Mode&syscall.S_IFMT != syscall.S_IFDIR {
			rights &= llOnFiles
		}
		var rule [12]byte // struct landlock_path_beneath_attr, packed
		binary.LittleEndian.PutUint64(rule[:], rights&all)
		binary.LittleEndian.PutUint32(rule[8:], uint32(f))
		syscall.Syscall6(sysLandlockAddRule, fd, 1, uintptr(unsafe.Pointer(&rule)), 0, 0, 0)
	}
	const read = llReadFile | llReadDir
	for _, dir := range []string{"/usr", "/lib", "/lib64", "/bin", "/sbin", "/etc", "/proc", "/sys/devices/system/cpu", "/dev/urandom", "/dev/zero"} {
		allow(dir, read)
	}
	allow("/dev/null", read|llWrite)
	// The dynamic loader is run by the kernel when the program starts: it is the only system file that may be executed.
	for _, loader := range []string{"/lib64/ld-linux-x86-64.so.2", "/lib/ld-linux-aarch64.so.1", "/lib/ld-musl-x86_64.so.1", "/lib/ld-musl-aarch64.so.1"} {
		allow(loader, read|llExecute)
	}
	for _, dir := range run {
		allow(dir, read|llExecute)
	}
	allow(write, all)
	if abi >= 4 && port != 0 {
		rule := struct{ access, port uint64 }{1 /* bind */, uint64(port)}
		syscall.Syscall6(sysLandlockAddRule, fd, 2, uintptr(unsafe.Pointer(&rule)), 0, 0, 0)
	}
	if _, _, e := syscall.Syscall(sysLandlockRestrictSelf, fd, 0, 0); e != 0 {
		return fmt.Errorf("landlock: %v", e)
	}
	return nil
}

type sockFilter struct {
	Code   uint16
	Jt, Jf uint8
	K      uint32
}

// lockSockets applies a seccomp filter: no outgoing connection, and no socket other than TCP-like
// ones, so nothing leaves by UDP either. It works on kernels too old for Landlock's network rules.
func lockSockets() error {
	const eperm = 0x00050001 // SECCOMP_RET_ERRNO | EPERM
	filter := []sockFilter{
		{0x20, 0, 0, 4},              // A = architecture
		{0x15, 1, 0, auditArch},      // ours: go on
		{0x06, 0, 0, 0x80000000},     // another one: kill, the numbers below would mean something else
		{0x20, 0, 0, 0},              // A = system call number
		{0x15, 0, 1, sysConnectCall}, // connect:
		{0x06, 0, 0, eperm},          //   refused
		{0x15, 0, 4, sysSocketCall},  // socket:
		{0x20, 0, 0, 24},             //   A = its type argument
		{0x54, 0, 0, 0xf},            //   without the flags
		{0x15, 1, 0, 1},              //   a stream socket: allowed
		{0x06, 0, 0, eperm},          //   anything else: refused
		{0x06, 0, 0, 0x7fff0000},     // allowed
	}
	prog := struct {
		Len    uint16
		Filter *sockFilter
	}{uint16(len(filter)), &filter[0]}
	if _, _, e := syscall.Syscall6(syscall.SYS_PRCTL, 22 /* PR_SET_SECCOMP */, 2 /* filter */, uintptr(unsafe.Pointer(&prog)), 0, 0, 0); e != 0 {
		return fmt.Errorf("seccomp: %v", e)
	}
	return nil
}
