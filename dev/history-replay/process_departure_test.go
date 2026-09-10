//go:build linux && amd64

package main

import (
	"errors"
	"os"
	"syscall"
	"testing"
)

func TestCgroupEnumerationProcessDeparture(t *testing.T) {
	for _, cause := range []error{syscall.ENOENT, syscall.ESRCH} {
		t.Run(cause.Error(), func(t *testing.T) {
			got, err := identifyCgroupProcesses("/owned", map[int]bool{1: true, 2: true}, func(pid int) (ProcessIdentity, error) {
				if pid == 1 {
					return ProcessIdentity{}, &os.PathError{Op: "read", Path: "/proc/1/cgroup", Err: cause}
				}
				return ProcessIdentity{PID: pid, StartTicks: 42, Cgroup: "/owned/child"}, nil
			})
			if err != nil || len(got) != 1 || got[0].PID != 2 || got[0].StartTicks != 42 {
				t.Fatalf("surviving owned process lost: %+v, %v", got, err)
			}
		})
	}
}

func TestCgroupEnumerationRejectsUnknownOwnership(t *testing.T) {
	for _, cause := range []error{syscall.EACCES, errors.New("bad process stat")} {
		_, err := identifyCgroupProcesses("/owned", map[int]bool{1: true}, func(int) (ProcessIdentity, error) { return ProcessIdentity{}, cause })
		if !errors.Is(err, cause) {
			t.Fatalf("ownership failure hidden: %v", err)
		}
	}
	for _, group := range []string{"/other", "/owned-elsewhere", ""} {
		_, err := identifyCgroupProcesses("/owned", map[int]bool{1: true}, func(int) (ProcessIdentity, error) { return ProcessIdentity{PID: 1, Cgroup: group}, nil })
		if err == nil {
			t.Fatalf("accepted escaped process in %q", group)
		}
	}
}
