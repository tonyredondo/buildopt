//go:build linux && amd64

package main

import (
	"context"
	"errors"
	"time"
)

// Keep the original polling cadence and a single size scan in flight, while
// allowing ownership/free-space/process checks to run during a slow traversal.
// Both observers are joined; cancellation never discards a real disk error.
func watchOwnedRequest(done <-chan struct{}, parent func() error, disk func(<-chan struct{}) error, processes func() error, cancel func()) error {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	diskResult := make(chan error, 1)
	go func() {
		err := pollOwnedRequest(ctx.Done(), done, func() error { return disk(ctx.Done()) })
		if err != nil {
			cancel()
			stop()
		}
		diskResult <- err
	}()
	err := pollOwnedRequest(ctx.Done(), done, func() error {
		if e := parent(); e != nil {
			return e
		}
		return processes()
	})
	if err != nil {
		cancel()
	}
	stop()
	return errors.Join(err, <-diskResult)
}

func pollOwnedRequest(stop, done <-chan struct{}, check func() error) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return nil
		case <-done:
			return nil
		case <-ticker.C:
			if e := check(); e != nil {
				if errors.Is(e, errDiskObservationStopped) {
					return nil
				}
				return e
			}
		}
	}
}
