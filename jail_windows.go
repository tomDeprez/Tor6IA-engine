//go:build windows

package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

var (
	userenv  = syscall.NewLazyDLL("userenv.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")
)

const containerName = "tor6ia.engine.block"

func u16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }

// containerSID creates the AppContainer profile, or finds the one that exists.
func containerSID() (unsafe.Pointer, string, error) {
	var sid unsafe.Pointer
	hr, _, _ := userenv.NewProc("CreateAppContainerProfile").Call(
		uintptr(unsafe.Pointer(u16(containerName))), uintptr(unsafe.Pointer(u16(containerName))),
		uintptr(unsafe.Pointer(u16("Tor6IA block server"))), 0, 0, uintptr(unsafe.Pointer(&sid)))
	if hr != 0 {
		hr, _, _ = userenv.NewProc("DeriveAppContainerSidFromAppContainerName").Call(
			uintptr(unsafe.Pointer(u16(containerName))), uintptr(unsafe.Pointer(&sid)))
		if hr != 0 {
			return nil, "", fmt.Errorf("Windows refused to create an AppContainer (%#x)", hr)
		}
	}
	var text *uint16
	if ok, _, err := advapi32.NewProc("ConvertSidToStringSidW").Call(uintptr(sid), uintptr(unsafe.Pointer(&text))); ok == 0 {
		return nil, "", err
	}
	return sid, syscall.UTF16ToString((*[256]uint16)(unsafe.Pointer(text))[:]), nil
}

// grant lets the container use one folder. Without it the container sees nothing of the user's files.
// The right is inherited by what is created in the folder later; /T also gives it to what is there
// already. A file that disappears while icacls walks the folder (a download being renamed) makes
// it fail, so that walk is allowed to fail as long as the folder itself took the right.
func grant(sidText, dir, rights string) error {
	rule := "*" + sidText + ":(OI)(CI)" + rights
	if exec.Command("icacls", dir, "/grant", rule, "/T", "/C", "/Q").Run() == nil {
		return nil
	}
	out, err := exec.Command("icacls", dir, "/grant", rule, "/Q").CombinedOutput()
	if err != nil {
		return fmt.Errorf("icacls %s: %v %s", dir, err, strings.TrimSpace(string(out)))
	}
	return nil
}

type securityCapabilities struct {
	AppContainerSid unsafe.Pointer
	Capabilities    unsafe.Pointer
	CapabilityCount uint32
	Reserved        uint32
}

type startupInfoEx struct {
	syscall.StartupInfo
	AttributeList unsafe.Pointer
}

// jobLimits is JOBOBJECT_EXTENDED_LIMIT_INFORMATION.
type jobLimits struct {
	PerProcessUserTimeLimit, PerJobUserTimeLimit int64
	LimitFlags                                   uint32
	MinimumWorkingSetSize, MaximumWorkingSetSize uintptr
	ActiveProcessLimit                           uint32
	Affinity                                     uintptr
	PriorityClass, SchedulingClass               uint32
	IoInfo                                       [6]uint64
	ProcessMemoryLimit, JobMemoryLimit           uintptr
	PeakProcessMemoryUsed, PeakJobMemoryUsed     uintptr
}

const (
	jobKillOnClose   = 0x2000
	jobActiveProcess = 0x8
	jobMemory        = 0x200
)

// newJob creates a job that ends its processes when this program ends, however it ends.
func newJob(processes uint32, memoryMB int) (syscall.Handle, error) {
	h, _, err := kernel32.NewProc("CreateJobObjectW").Call(0, 0)
	if h == 0 {
		return 0, err
	}
	info := jobLimits{LimitFlags: jobKillOnClose}
	if processes > 0 {
		info.LimitFlags |= jobActiveProcess
		info.ActiveProcessLimit = processes
	}
	if memoryMB > 0 {
		info.LimitFlags |= jobMemory
		info.JobMemoryLimit = uintptr(memoryMB) << 20
	}
	const extendedLimitInformation = 9
	if ok, _, err := kernel32.NewProc("SetInformationJobObject").Call(h, extendedLimitInformation,
		uintptr(unsafe.Pointer(&info)), unsafe.Sizeof(info)); ok == 0 {
		return 0, err
	}
	return syscall.Handle(h), nil
}

var ownJob syscall.Handle

// launch starts a program that stops with the engine, even when the engine is killed by force.
func launch(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	if ownJob == 0 {
		if j, err := newJob(0, 0); err == nil {
			ownJob = j
		}
	}
	const setQuota, terminate = 0x0100, 0x0001
	if h, err := syscall.OpenProcess(setQuota|terminate, false, uint32(cmd.Process.Pid)); err == nil {
		kernel32.NewProc("AssignProcessToJobObject").Call(uintptr(ownJob), uintptr(h))
		syscall.CloseHandle(h)
	}
	return nil
}

// startJailed starts a program in an AppContainer with no capability. Such a container cannot be
// reached through the network, even from this machine, so a bridge runs inside it next to the
// program, and front is a port of ours whose connections are carried through that bridge.
func startJailed(j Jail, front, program string, args []string) (*jailProc, error) {
	sid, sidText, err := containerSID()
	if err != nil {
		return nil, err
	}
	runDir, _, err := jailHome()
	if err != nil {
		return nil, err
	}
	bridge, err := selfCopy(runDir)
	if err != nil {
		return nil, err
	}
	for _, dir := range append([]string{runDir}, j.Run...) {
		if err := grant(sidText, dir, "RX"); err != nil {
			return nil, err
		}
	}
	if err := grant(sidText, j.Write, "M"); err != nil {
		return nil, err
	}
	logPath := filepath.Join(j.Write, "block-"+strconv.Itoa(j.Port)+".log")

	parts := append([]string{bridge, "bridge", "--port", strconv.Itoa(j.Port), "--log", logPath, program}, args...)
	for i, p := range parts {
		parts[i] = syscall.EscapeArg(p)
	}
	// The locked side gets this program's environment, with its cache and temporary files in the folder it may write.
	var env []uint16
	for _, kv := range append(os.Environ(), "LLAMA_CACHE="+j.Write, "TEMP="+j.Write, "TMP="+j.Write) {
		env = append(env, syscall.StringToUTF16(kv)...)
	}
	env = append(env, 0)

	toChild, fromUs, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	fromChild, toUs, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	// What the bridge itself prints goes to a file we open here: the bridge could not create it.
	null, err := os.Create(logPath + ".bridge")
	if err != nil {
		return nil, err
	}
	defer null.Close()
	const inherit = 1
	for _, f := range []*os.File{toChild, toUs, null} {
		syscall.SetHandleInformation(syscall.Handle(f.Fd()), inherit, inherit)
	}

	const securityCapabilitiesAttribute = 0x00020009
	var size uintptr
	initList := kernel32.NewProc("InitializeProcThreadAttributeList")
	initList.Call(0, 1, 0, uintptr(unsafe.Pointer(&size)))
	list := make([]byte, size)
	if ok, _, err := initList.Call(uintptr(unsafe.Pointer(&list[0])), 1, 0, uintptr(unsafe.Pointer(&size))); ok == 0 {
		return nil, err
	}
	caps := securityCapabilities{AppContainerSid: sid}
	if ok, _, err := kernel32.NewProc("UpdateProcThreadAttribute").Call(uintptr(unsafe.Pointer(&list[0])), 0,
		securityCapabilitiesAttribute, uintptr(unsafe.Pointer(&caps)), unsafe.Sizeof(caps), 0, 0); ok == 0 {
		return nil, err
	}
	const useStdHandles = 0x100
	var si startupInfoEx
	si.Cb = uint32(unsafe.Sizeof(si))
	si.Flags = useStdHandles
	si.StdInput, si.StdOutput, si.StdErr = syscall.Handle(toChild.Fd()), syscall.Handle(toUs.Fd()), syscall.Handle(null.Fd())
	si.AttributeList = unsafe.Pointer(&list[0])

	// The job is in place before the first instruction runs: two processes at most, the bridge and the program.
	job, err := newJob(2, j.MemoryMB)
	if err != nil {
		return nil, err
	}
	const extendedStartupInfo, noWindow, suspended, unicodeEnv = 0x00080000, 0x08000000, 0x4, 0x400
	var pi syscall.ProcessInformation
	ok, _, cerr := kernel32.NewProc("CreateProcessW").Call(0, uintptr(unsafe.Pointer(u16(strings.Join(parts, " ")))), 0, 0, 1,
		extendedStartupInfo|noWindow|suspended|unicodeEnv, uintptr(unsafe.Pointer(&env[0])),
		uintptr(unsafe.Pointer(u16(runDir))), uintptr(unsafe.Pointer(&si)), uintptr(unsafe.Pointer(&pi)))
	toChild.Close()
	toUs.Close()
	if ok == 0 {
		syscall.CloseHandle(job)
		return nil, fmt.Errorf("cannot start the locked program: %v", cerr)
	}
	if ok, _, err := kernel32.NewProc("AssignProcessToJobObject").Call(uintptr(job), uintptr(pi.Process)); ok == 0 {
		syscall.TerminateProcess(pi.Process, 1)
		return nil, fmt.Errorf("cannot limit the locked program: %v", err)
	}
	kernel32.NewProc("ResumeThread").Call(uintptr(pi.Thread))
	syscall.CloseHandle(pi.Thread)

	p := &jailProc{Log: logPath, Done: make(chan struct{})}
	p.kill = func() {
		fromUs.Close()
		syscall.CloseHandle(job) // ends everything in the job
	}
	go func() {
		syscall.WaitForSingleObject(pi.Process, syscall.INFINITE)
		var code uint32
		syscall.GetExitCodeProcess(pi.Process, &code)
		if code != 0 {
			if f, err := os.OpenFile(logPath+".bridge", os.O_APPEND|os.O_WRONLY, 0); err == nil {
				fmt.Fprintf(f, "the locked side ended with code %#x\n", code)
				f.Close()
			}
		}
		syscall.CloseHandle(pi.Process)
		close(p.Done)
	}()

	m := newMux(fromUs)
	go m.run(fromChild, nil)
	if j.Port == 0 {
		return p, nil // a program that does not listen: the caller waits for it to end
	}
	select {
	case <-m.ready:
	case <-p.Done:
		return nil, errors.New("the locked program stopped at once, see " + logPath)
	}
	l, err := net.Listen("tcp", front)
	if err != nil {
		p.Stop()
		return nil, err
	}
	p.Addr = l.Addr().String()
	stop := p.kill
	p.kill = func() { l.Close(); stop() }
	go m.serveOutside(l)
	return p, nil
}

// hog takes n bytes of memory from the system. Committed memory counts at once on Windows.
func hog(n int) error {
	const commit, reserve, readWrite = 0x1000, 0x2000, 0x04
	if mem, _, err := kernel32.NewProc("VirtualAlloc").Call(0, uintptr(n), commit|reserve, readWrite); mem == 0 {
		return err
	}
	return nil
}

// quiet starts a program without a console. A console is one more process, and the job of a
// locked program allows two: the bridge and the program.
func quiet(cmd *exec.Cmd) {
	const noWindow = 0x08000000
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: noWindow}
}
