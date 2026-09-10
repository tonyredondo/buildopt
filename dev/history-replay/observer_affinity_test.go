//go:build linux && amd64 && replay_integration

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestAffinityChild(t *testing.T) {
	if os.Getenv("ISOLATION_CHILD") != "1" {
		return
	}
	var work sync.WaitGroup
	for i := 0; i < 16; i++ {
		work.Add(1)
		go func() {
			defer work.Done()
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			time.Sleep(30 * time.Millisecond)
		}()
	}
	work.Wait()
	masks, err := threadAffinities(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	for tid, mask := range masks {
		if mask != "0,1,2,3,4,5,6,7" {
			t.Fatalf("child thread %s mask %s", tid, mask)
		}
	}
	fmt.Printf("CHILD_MASK_VERIFIED threads=%d\n", len(masks))
	if os.Getenv("ISOLATION_SLEEP") == "1" {
		time.Sleep(10 * time.Second)
	}
}

func TestAffinityScope(t *testing.T) {
	for _, mode := range []string{"success", "launch-failure", "set-failure", "restore-failure", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			// The actual failure path must dispose of its locked thread. Exercise
			// it on a secondary OS thread and observe its removal afterward.
			// Go parks m0 instead of removing it (runtime.mexit); /proc presence
			// alone would incorrectly classify that retired main thread as reused.
			result := make(chan error, 1)
			tidResult := make(chan int, 1)
			var exercise func()
			exercise = func() {
				runtime.LockOSThread()
				if unix.Gettid() == os.Getpid() {
					defer runtime.UnlockOSThread()
					done := make(chan struct{})
					go func() { exercise(); close(done) }()
					<-done
					return
				}
				var initial unix.CPUSet
				if err := unix.SchedGetaffinity(0, &initial); err != nil {
					result <- err
					return
				}
				observer, _ := parseCPUSet("8")
				if err := unix.SchedSetaffinity(0, &observer); err != nil {
					result <- err
					return
				}
				scope, err := newAffinityScope("0-7", "8")
				if err != nil {
					result <- err
					return
				}
				t.Logf("affinity mode=%s tid=%d pid=%d", mode, unix.Gettid(), os.Getpid())
				tidResult <- unix.Gettid()
				defer func() {
					scope.release()
					if scope.safe {
						_ = unix.SchedSetaffinity(0, &initial)
						runtime.UnlockOSThread()
					}
				}()
				calls := 0
				if mode == "set-failure" || mode == "restore-failure" {
					scope.set = func(pid int, mask *unix.CPUSet) error {
						calls++
						if (mode == "set-failure" && calls == 1) || (mode == "restore-failure" && calls == 2) {
							return unix.EPERM
						}
						return unix.SchedSetaffinity(pid, mask)
					}
				}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				self, _ := os.Executable()
				c := exec.CommandContext(ctx, self, "-test.run=^TestAffinityChild$")
				c.Env = append(os.Environ(), "ISOLATION_CHILD=1")
				if mode == "cancel" || mode == "restore-failure" {
					c.Env = append(c.Env, "ISOLATION_SLEEP=1")
				}
				c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
				c.Cancel = func() error { return syscall.Kill(-c.Process.Pid, syscall.SIGKILL) }
				var output bytes.Buffer
				c.Stdout = &output
				c.Stderr = &output
				if mode == "launch-failure" {
					c.Path = "/buildopt-isolation-missing-executable"
				}
				err = scope.start(c)
				if mode == "restore-failure" {
					if err == nil || scope.safe || c.ProcessState == nil || c.Process == nil {
						result <- fmt.Errorf("restoration failure leaked child: %v", err)
						return
					}
					if e := c.Process.Signal(syscall.Signal(0)); !errors.Is(e, os.ErrProcessDone) {
						result <- fmt.Errorf("failed child not reaped: %v", e)
						return
					}
					result <- nil
					return
				}
				var after unix.CPUSet
				if e := unix.SchedGetaffinity(0, &after); e != nil || after != observer || !scope.safe {
					result <- fmt.Errorf("observer mask not restored: %v", e)
					return
				}
				if mode == "launch-failure" || mode == "set-failure" {
					if err == nil || c.Process != nil {
						result <- fmt.Errorf("failed setup launched child: %v", err)
						return
					}
					result <- nil
					return
				}
				if err != nil {
					result <- err
					return
				}
				if mode == "cancel" {
					time.Sleep(100 * time.Millisecond)
					cancel()
				}
				err = c.Wait()
				if mode == "cancel" {
					if err == nil {
						result <- errors.New("cancelled child succeeded")
						return
					}
				} else if err != nil || !strings.Contains(output.String(), "CHILD_MASK_VERIFIED") {
					result <- fmt.Errorf("native inheritance: %v %s", err, output.String())
					return
				}
				result <- nil
			}
			go exercise()
			select {
			case err := <-result:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("affinity scope did not close")
			}
			if mode == "restore-failure" {
				tid := <-tidResult
				for limit := time.Now().Add(time.Second); time.Now().Before(limit); {
					if _, err := os.Stat(fmt.Sprintf("/proc/self/task/%d", tid)); os.IsNotExist(err) {
						return
					}
					time.Sleep(time.Millisecond)
				}
				status, _ := os.ReadFile(fmt.Sprintf("/proc/self/task/%d/status", tid))
				wchan, _ := os.ReadFile(fmt.Sprintf("/proc/self/task/%d/wchan", tid))
				t.Fatalf("unrestored thread remains tid=%d pid=%d wchan=%s status=%s", tid, os.Getpid(), wchan, status)
			}
		})
	}
}
