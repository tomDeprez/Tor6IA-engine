//go:build linux

package main

// System call numbers and architecture tag for the seccomp filter.
const (
	auditArch      = 0xC000003E
	sysConnectCall = 42
	sysSocketCall  = 41
)
