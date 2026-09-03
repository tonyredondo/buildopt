//go:build !windows

package main

import (
	"os"
	"os/signal"
	"syscall"
)

func exitWithSignal(number int) {
	value := syscall.Signal(number)
	signal.Reset(value)
	_ = syscall.Kill(os.Getpid(), value)
	os.Exit(128 + number)
}
