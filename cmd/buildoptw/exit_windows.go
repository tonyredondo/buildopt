//go:build windows

package main

import "os"

func exitWithSignal(number int) { os.Exit(128 + number) }
