//go:build !windows

package main

import "time"

func nativeTestTimeout(value time.Duration) time.Duration {
	// Server startup can contend with SQLite and filesystem-heavy packages when
	// the full suite runs under the race detector on an ordinary CI runner.
	// Keep the behavioral assertions strict while allowing that initialization
	// enough wall time to complete.
	if value < 10*time.Second {
		return 10 * time.Second
	}
	return value
}
