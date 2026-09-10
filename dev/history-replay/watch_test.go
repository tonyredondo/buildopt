//go:build linux && amd64

package main

import (
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestOwnedWatchObservesProcessesDuringSlowDiskScan(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	result := make(chan error, 1)
	var visits atomic.Int32
	go func() {
		result <- watchOwnedRequest(done, func() error { return nil }, func(stop <-chan struct{}) error {
			select {
			case <-entered:
			default:
				close(entered)
			}
			select {
			case <-release:
				return nil
			case <-stop:
				return errDiskObservationStopped
			}
		}, func() error { visits.Add(1); return nil }, func() {})
	}()
	defer func() {
		close(done)
		close(release)
		if err := <-result; err != nil {
			t.Error(err)
		}
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("disk scan did not start")
	}
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) && visits.Load() < 3 {
		time.Sleep(5 * time.Millisecond)
	}
	if visits.Load() < 3 {
		t.Fatal("slow disk scan blocks live process observation")
	}
}

func TestOwnedWatchRetainsDiskFailureAtChildExit(t *testing.T) {
	entered := make(chan struct{})
	done := make(chan struct{})
	result := make(chan error, 1)
	want := errors.New("disk I/O failure")
	var cancels atomic.Int32
	go func() {
		result <- watchOwnedRequest(done, func() error { return nil }, func(stop <-chan struct{}) error {
			close(entered)
			<-stop
			return want
		}, func() error { return nil }, func() { cancels.Add(1) })
	}()
	<-entered
	close(done)
	select {
	case err := <-result:
		if !errors.Is(err, want) || cancels.Load() != 1 {
			t.Fatalf("disk failure was discarded at exit: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("observer was not joined")
	}
}

func TestOwnedWatchEnforcesLiveDiskLimits(t *testing.T) {
	for _, mode := range []string{"growth", "free-space", "missing-root", "process-error"} {
		t.Run(mode, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "owned")
			if err := os.Mkdir(root, 0700); err != nil {
				t.Fatal(err)
			}
			file := filepath.Join(root, "growing")
			if err := os.WriteFile(file, nil, 0600); err != nil {
				t.Fatal(err)
			}
			entered, done := make(chan struct{}), make(chan struct{})
			result := make(chan error, 1)
			var trigger atomic.Bool
			var cancels atomic.Int32
			go func() {
				result <- watchOwnedRequest(done, func() error {
					floor := uint64(0)
					if mode == "free-space" && trigger.Load() {
						floor = ^uint64(0)
					}
					return freeDiskGuard(root, floor)
				}, func(stop <-chan struct{}) error {
					select {
					case <-entered:
					default:
						close(entered)
					}
					if mode != "growth" {
						<-stop
						return errDiskObservationStopped
					}
					return diskGuardUntil(root, Limits{MaxBytes: 1}, stop)
				}, func() error {
					if mode == "process-error" && trigger.Load() {
						return os.ErrPermission
					}
					return nil
				}, func() { cancels.Add(1) })
			}()
			defer close(done)
			<-entered
			if mode == "growth" {
				if err := os.Truncate(file, 2); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "missing-root" {
				if err := os.Remove(file); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(root); err != nil {
					t.Fatal(err)
				}
			}
			trigger.Store(true)
			select {
			case err := <-result:
				if err == nil || cancels.Load() == 0 {
					t.Fatalf("live failure lost: %v", err)
				}
			case <-time.After(time.Second):
				t.Fatal("live guard did not cancel")
			}
		})
	}
}

func TestOwnedWatchCancelsBlockedDiskAfterDriverLoss(t *testing.T) {
	entered := make(chan struct{})
	done := make(chan struct{})
	result := make(chan error, 1)
	lost := errors.New("driver disappeared")
	var calls, cancels atomic.Int32
	go func() {
		result <- watchOwnedRequest(done, func() error {
			if calls.Add(1) > 1 {
				return lost
			}
			return nil
		}, func(stop <-chan struct{}) error {
			select {
			case <-entered:
			default:
				close(entered)
			}
			<-stop
			return errDiskObservationStopped
		}, func() error { return nil }, func() { cancels.Add(1) })
	}()
	defer close(done)
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("scan not entered")
	}
	select {
	case err := <-result:
		if !errors.Is(err, lost) || cancels.Load() != 1 {
			t.Fatalf("lost ownership: %v %d", err, cancels.Load())
		}
	case <-time.After(time.Second):
		t.Fatal("driver loss cannot cancel blocked disk observation")
	}
}
