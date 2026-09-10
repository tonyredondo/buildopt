//go:build linux && amd64

package main

import (
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func parseCPUSet(value string) (unix.CPUSet, error) {
	var mask unix.CPUSet
	for _, token := range strings.Split(value, ",") {
		ends := strings.Split(token, "-")
		if len(ends) > 2 {
			return mask, errors.New("invalid CPU range")
		}
		first, err := strconv.Atoi(ends[0])
		if err != nil {
			return mask, err
		}
		last := first
		if len(ends) == 2 {
			last, err = strconv.Atoi(ends[1])
			if err != nil {
				return mask, err
			}
		}
		if first < 0 || last < first || last >= 1024 {
			return mask, errors.New("CPU range outside supported mask")
		}
		for cpu := first; cpu <= last; cpu++ {
			mask.Set(cpu)
		}
	}
	if mask.Count() == 0 {
		return mask, errors.New("empty CPU mask")
	}
	return mask, nil
}

func threadAffinities(pid int) (map[string]string, error) {
	entries, err := os.ReadDir(fmt.Sprintf("/proc/%d/task", pid))
	if err != nil {
		return nil, err
	}
	result := map[string]string{}
	for _, entry := range entries {
		tid, err := strconv.Atoi(entry.Name())
		if err != nil {
			return nil, err
		}
		var mask unix.CPUSet
		if err := unix.SchedGetaffinity(tid, &mask); err != nil {
			if errors.Is(err, unix.ESRCH) {
				continue
			}
			return nil, err
		}
		cpus := []string{}
		for cpu := 0; cpu < 1024; cpu++ {
			if mask.IsSet(cpu) {
				cpus = append(cpus, strconv.Itoa(cpu))
			}
		}
		result[filepath.Base(entry.Name())] = strings.Join(cpus, ",")
	}
	return result, nil
}

type affinityScope struct {
	original, native unix.CPUSet
	safe             bool
	set              func(int, *unix.CPUSet) error
}

func newAffinityScope(native, observer string) (*affinityScope, error) {
	n, err := parseCPUSet(native)
	if err != nil {
		return nil, err
	}
	o, err := parseCPUSet(observer)
	if err != nil {
		return nil, err
	}
	// LockOSThread creates Go's template thread before inheritable state changes.
	// Keep this thread alive until child wait: Pdeathsig is tied to its lifetime.
	runtime.LockOSThread()
	s := &affinityScope{native: n, safe: true, set: unix.SchedSetaffinity}
	if err := unix.SchedGetaffinity(0, &s.original); err != nil {
		runtime.UnlockOSThread()
		return nil, err
	}
	if s.original != o {
		runtime.UnlockOSThread()
		return nil, errors.New("observer affinity differs from bound configuration")
	}
	return s, nil
}

func (s *affinityScope) exact(mask *unix.CPUSet) error {
	if err := s.set(0, mask); err != nil {
		return err
	}
	var actual unix.CPUSet
	if err := unix.SchedGetaffinity(0, &actual); err != nil {
		return err
	}
	if actual != *mask {
		return errors.New("kernel restricted requested CPU mask")
	}
	return nil
}

func (s *affinityScope) restore() error {
	err := s.exact(&s.original)
	s.safe = err == nil
	return err
}

func (s *affinityScope) start(c *exec.Cmd) error {
	s.safe = false
	if err := s.exact(&s.native); err != nil {
		return errors.Join(err, s.restore())
	}
	err := c.Start()
	restoreErr := s.restore()
	if restoreErr != nil && c.Process != nil {
		// A restoration failure must not leave a native child running or allow
		// another request. The worker exits; this locked thread is never pooled.
		var killErr error
		if c.Cancel != nil {
			killErr = c.Cancel()
		} else {
			killErr = c.Process.Kill()
		}
		waitErr := c.Wait()
		return errors.Join(err, fmt.Errorf("restore observer affinity: %w", restoreErr), killErr, waitErr)
	}
	return errors.Join(err, restoreErr)
}

func (s *affinityScope) release() {
	if s.safe {
		runtime.UnlockOSThread()
	}
}
