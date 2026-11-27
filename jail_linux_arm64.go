//go:build linux

package main

// System call numbers and architecture tag for the seccomp filter.
const (
	auditArch      = 0xC00000B7
	sysConnectCall = 203
	sysSocketCall  = 198
)
