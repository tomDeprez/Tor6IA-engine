//go:build !windows

package main

import "os/exec"

// quiet is for Windows, where a program can bring a console process with it.
func quiet(*exec.Cmd) {}
