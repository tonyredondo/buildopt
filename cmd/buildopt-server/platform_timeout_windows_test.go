//go:build windows

package main

import "time"

func nativeTestTimeout(value time.Duration) time.Duration {
	if value < 15*time.Second {
		return 15 * time.Second
	}
	return value
}
