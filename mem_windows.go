//go:build windows

package main

import "unsafe"

// freeMemory returns the memory programs can still use, in bytes, 0 when unknown.
func freeMemory() int64 {
	var m struct {
		Length, MemoryLoad                                                                            uint32
		TotalPhys, AvailPhys, TotalPageFile, AvailPageFile, TotalVirtual, AvailVirtual, AvailExtended uint64
	}
	m.Length = uint32(unsafe.Sizeof(m))
	if ok, _, _ := kernel32.NewProc("GlobalMemoryStatusEx").Call(uintptr(unsafe.Pointer(&m))); ok == 0 {
		return 0
	}
	return int64(m.AvailPhys)
}
