//go:build !linux

package main

import "errors"

// runJailExec exists on Linux only, where a program locks itself in before it becomes the block server.
func runJailExec([]string) error { return errors.New("jail-exec is for Linux") }
