//go:build !windows

package main

import "time"

func nativeTestTimeout(value time.Duration) time.Duration { return value }
