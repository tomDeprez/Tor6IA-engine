//go:build linux

package main

import (
	"os"
	"strconv"
	"strings"
)

// freeMemory returns the memory programs can still use, in bytes, 0 when unknown.
func freeMemory() int64 {
	raw, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if f := strings.Fields(line); len(f) >= 2 && f[0] == "MemAvailable:" {
			kb, _ := strconv.ParseInt(f[1], 10, 64)
			return kb << 10
		}
	}
	return 0
}
