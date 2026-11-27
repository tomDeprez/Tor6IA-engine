//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

// hog takes n bytes of memory from the system and writes to all of it.
func hog(n int) error {
	mem, err := syscall.Mmap(-1, 0, n, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE)
	if err != nil {
		return err
	}
	for i := 0; i < n; i += 4096 {
		mem[i] = 1
	}
	return nil
}

// quiet is for Windows, where a program can bring a console process with it.
func quiet(*exec.Cmd) {}
