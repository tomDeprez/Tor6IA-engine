//go:build windows

// winjail is trial E1 for Windows: can a process be locked in an AppContainer without
// administrator rights, and still be reached by the program that started it?
//
//	winjail            starts itself locked in, and prints what the locked copy could and could not do
//
// The locked copy tries to read a personal file, to connect to the Internet and to start another
// program: all three must fail. It then checks the two ways it could be reached: a named pipe
// opened by its parent, and a TCP port on the machine itself.
package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	userenv  = syscall.NewLazyDLL("userenv.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")
)

const (
	pipeName      = `\\.\pipe\tor6ia-winjail`
	containerName = "tor6ia.engine.trial"
)

func u16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }

// containerSID creates the AppContainer profile, or finds the one that exists, and returns its SID.
func containerSID() (unsafe.Pointer, string, error) {
	var sid unsafe.Pointer
	hr, _, _ := userenv.NewProc("CreateAppContainerProfile").Call(
		uintptr(unsafe.Pointer(u16(containerName))), uintptr(unsafe.Pointer(u16(containerName))),
		uintptr(unsafe.Pointer(u16("Tor6IA engine trial"))), 0, 0, uintptr(unsafe.Pointer(&sid)))
	if hr != 0 {
		hr, _, _ = userenv.NewProc("DeriveAppContainerSidFromAppContainerName").Call(
			uintptr(unsafe.Pointer(u16(containerName))), uintptr(unsafe.Pointer(&sid)))
		if hr != 0 {
			return nil, "", fmt.Errorf("no AppContainer: %#x", hr)
		}
	}
	var text *uint16
	if ok, _, err := advapi32.NewProc("ConvertSidToStringSidW").Call(uintptr(sid), uintptr(unsafe.Pointer(&text))); ok == 0 {
		return nil, "", err
	}
	return sid, syscall.UTF16ToString((*[256]uint16)(unsafe.Pointer(text))[:]), nil
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

// startLocked starts a program inside the AppContainer, with no capability at all.
func startLocked(sid unsafe.Pointer, command, dir string) (syscall.Handle, error) {
	const securityCapabilitiesAttribute = 0x00020009
	const extendedStartupInfo, noWindow = 0x00080000, 0x08000000
	var size uintptr
	initList := kernel32.NewProc("InitializeProcThreadAttributeList")
	initList.Call(0, 1, 0, uintptr(unsafe.Pointer(&size)))
	list := make([]byte, size)
	if ok, _, err := initList.Call(uintptr(unsafe.Pointer(&list[0])), 1, 0, uintptr(unsafe.Pointer(&size))); ok == 0 {
		return 0, err
	}
	caps := securityCapabilities{AppContainerSid: sid}
	if ok, _, err := kernel32.NewProc("UpdateProcThreadAttribute").Call(uintptr(unsafe.Pointer(&list[0])), 0,
		securityCapabilitiesAttribute, uintptr(unsafe.Pointer(&caps)), unsafe.Sizeof(caps), 0, 0); ok == 0 {
		return 0, err
	}
	var si startupInfoEx
	si.Cb = uint32(unsafe.Sizeof(si))
	si.AttributeList = unsafe.Pointer(&list[0])
	var pi syscall.ProcessInformation
	if ok, _, err := kernel32.NewProc("CreateProcessW").Call(0, uintptr(unsafe.Pointer(u16(command))), 0, 0, 0,
		extendedStartupInfo|noWindow, 0, uintptr(unsafe.Pointer(u16(dir))), uintptr(unsafe.Pointer(&si)), uintptr(unsafe.Pointer(&pi))); ok == 0 {
		return 0, err
	}
	syscall.CloseHandle(pi.Thread)
	return pi.Process, nil
}

// openPipe creates the named pipe the locked copy reports through. Only this user and the container may open it.
func openPipe(sidText string) (syscall.Handle, error) {
	var sd unsafe.Pointer
	sddl := "D:(A;;GA;;;OW)(A;;GA;;;" + sidText + ")"
	if ok, _, err := advapi32.NewProc("ConvertStringSecurityDescriptorToSecurityDescriptorW").Call(
		uintptr(unsafe.Pointer(u16(sddl))), 1, uintptr(unsafe.Pointer(&sd)), 0); ok == 0 {
		return 0, err
	}
	sa := syscall.SecurityAttributes{SecurityDescriptor: uintptr(sd)}
	sa.Length = uint32(unsafe.Sizeof(sa))
	const inbound, byteMode = 0x1, 0x0
	h, _, err := kernel32.NewProc("CreateNamedPipeW").Call(uintptr(unsafe.Pointer(u16(pipeName))), inbound, byteMode, 1, 65536, 65536, 0, uintptr(unsafe.Pointer(&sa)))
	if syscall.Handle(h) == syscall.InvalidHandle {
		return 0, err
	}
	return syscall.Handle(h), nil
}

func outcome(err error) string {
	if err != nil {
		return "refused (" + strings.TrimSpace(err.Error()) + ")"
	}
	return "ALLOWED"
}

// locked is what runs inside the container.
func locked(secret string) {
	var report []string
	say := func(what string, err error) { report = append(report, fmt.Sprintf("%-46s %s", what, outcome(err))) }

	_, err := os.ReadFile(secret)
	say("read a personal file", err)
	conn, err := net.DialTimeout("tcp", "1.1.1.1:443", 4*time.Second)
	if err == nil {
		conn.Close()
	}
	say("connect to the Internet", err)
	say("start another program (cmd.exe)", exec.Command("cmd", "/c", "exit 0").Run())

	// A port on the machine itself, reached by a second locked process: what the engine would need.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	say("listen on the machine itself", err)
	if err == nil {
		go func() {
			for {
				c, err := ln.Accept()
				if err != nil {
					return
				}
				c.Close()
			}
		}()
		self, _ := os.Executable()
		say("port reached by a second locked process", exec.Command(self, "dial", ln.Addr().String()).Run())
		report = append(report, "PORT "+ln.Addr().String())
	}
	pipe, err := os.OpenFile(pipeName, os.O_WRONLY, 0)
	if err != nil {
		os.Exit(3) // nobody would see the report
	}
	fmt.Fprintln(pipe, strings.Join(report, "\n"))
	time.Sleep(4 * time.Second) // let the parent try the port from outside
	pipe.Close()
}

func main() {
	if len(os.Args) > 2 && os.Args[1] == "locked" {
		locked(os.Args[2])
		return
	}
	if len(os.Args) > 2 && os.Args[1] == "dial" {
		c, err := net.DialTimeout("tcp", os.Args[2], 3*time.Second)
		if err != nil {
			os.Exit(1)
		}
		c.Close()
		return
	}

	sid, sidText, err := containerSID()
	if err != nil {
		fmt.Println("cannot create an AppContainer:", err)
		os.Exit(1)
	}
	self, _ := os.Executable()
	dir := filepath.Dir(self)
	// The container sees nothing by default: let it read and run this folder only.
	if out, err := exec.Command("icacls", dir, "/grant", "*"+sidText+":(OI)(CI)(RX)", "/T", "/Q").CombinedOutput(); err != nil {
		fmt.Println("icacls:", err, string(out))
		os.Exit(1)
	}
	home, _ := os.UserHomeDir()
	secret := filepath.Join(home, "tor6ia-winjail-secret.txt")
	os.WriteFile(secret, []byte("personal"), 0o600)
	defer os.Remove(secret)

	pipe, err := openPipe(sidText)
	if err != nil {
		fmt.Println("pipe:", err)
		os.Exit(1)
	}
	proc, err := startLocked(sid, fmt.Sprintf(`"%s" locked "%s"`, self, secret), dir)
	if err != nil {
		fmt.Println("cannot start the locked copy:", err)
		os.Exit(1)
	}
	defer syscall.TerminateProcess(proc, 0)
	kernel32.NewProc("ConnectNamedPipe").Call(uintptr(pipe), 0)
	buf := make([]byte, 8192)
	var n uint32
	syscall.ReadFile(pipe, buf, &n, nil)
	fmt.Println("Without administrator rights, inside an AppContainer with no capability:")
	for _, line := range strings.Split(strings.TrimSpace(string(buf[:n])), "\n") {
		if addr, ok := strings.CutPrefix(line, "PORT "); ok {
			c, err := net.DialTimeout("tcp", strings.TrimSpace(addr), 3*time.Second)
			if err == nil {
				c.Close()
			}
			fmt.Printf("  %-46s %s\n", "port reached from outside the container", outcome(err))
			continue
		}
		fmt.Println(" ", line)
	}
	fmt.Printf("  %-46s %s\n", "report received through the named pipe", "ALLOWED")
}
