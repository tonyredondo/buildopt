//go:build linux && amd64

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// Only a vanished process identity is a recoverable snapshot gap. A missing
// owned cgroup, permission error or unrelated I/O error still stops observation.
func diagnosticExitedProcess(err error) bool {
	var pathError *os.PathError
	if !errors.As(err, &pathError) || !(os.IsNotExist(err) || errors.Is(err, syscall.ESRCH)) {
		return false
	}
	parts := strings.Split(pathError.Path, "/")
	if len(parts) != 4 || parts[0] != "" || parts[1] != "proc" {
		return false
	}
	pid, e := strconv.Atoi(parts[2])
	if e != nil || pid <= 0 {
		return false
	}
	return parts[3] == "stat" || parts[3] == "cgroup" || parts[3] == "cmdline"
}

// Preserve the original object construction and process-exit error policy.
func collectOwnedSample(worker workerSession, observe func(string) ([]ProcessIdentity, error)) (map[string]interface{}, error) {
	begin := stamp()
	ps, err := observe(worker.Cgroup)
	if err != nil {
		if diagnosticExitedProcess(err) {
			return map[string]interface{}{"begin": begin, "end": stamp(), "gap": "PROCESS_EXITED_DURING_SNAPSHOT", "error": err.Error()}, nil
		}
		return nil, err
	}
	rows := []map[string]interface{}{}
	for _, p := range ps {
		row := map[string]interface{}{"identity": p}
		masks, affinityErr := threadAffinities(p.PID)
		if affinityErr != nil {
			row["affinityError"] = affinityErr.Error()
		} else {
			row["threadAffinities"] = masks
		}

		for _, name := range []string{"stat", "schedstat", "io"} {
			raw, e := os.ReadFile(fmt.Sprintf("/proc/%d/%s", p.PID, name))
			if e != nil {
				row[name+"Error"] = e.Error()
			} else {
				row[name] = string(raw)
			}
		}
		rows = append(rows, row)
	}
	group := map[string]string{}
	for _, name := range []string{"cpu.stat", "cpu.pressure", "io.pressure", "memory.pressure"} {
		raw, e := os.ReadFile(filepath.Join("/sys/fs/cgroup", worker.Cgroup, name))
		if e != nil {
			group[name] = "UNAVAILABLE: " + e.Error()
		} else {
			group[name] = string(raw)
		}
	}
	return map[string]interface{}{"begin": begin, "end": stamp(), "processes": rows, "cgroup": group}, nil
}
