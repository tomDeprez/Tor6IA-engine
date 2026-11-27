//go:build darwin

package main

import (
	"os/exec"
	"strconv"
	"strings"
)

// freeMemory returns the memory programs can still use, in bytes, 0 when unknown.
// shortcut: macOS does not give a simple figure, so this is 60 % of the installed memory; read vm_stat if the advice proves wrong on Macs.
func freeMemory() int64 {
	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 0
	}
	total, _ := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	return total * 6 / 10
}
