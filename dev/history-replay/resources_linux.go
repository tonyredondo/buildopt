//go:build linux && amd64

package main

import (
	"os"
	"path/filepath"
	"strings"
)

// Diagnostic snapshots describe the host and the whole owned service, including
// reused JVMs. CLI rusage remains separately labelled in ProcessReceipt.
type ResourceObservation struct {
	At                  Stamp             `json:"at"`
	HostLoadAverage     string            `json:"hostLoadAverage"`
	HostMemoryAvailable string            `json:"hostMemoryAvailable"`
	CPUAffinity         string            `json:"cpuAffinity"`
	CgroupFiles         map[string]string `json:"cgroupFiles"`
}

func observeResources(group string) (ResourceObservation, error) {
	r := ResourceObservation{At: stamp(), CgroupFiles: map[string]string{}}
	raw, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return r, err
	}
	r.HostLoadAverage = strings.TrimSpace(string(raw))
	raw, err = os.ReadFile("/proc/meminfo")
	if err != nil {
		return r, err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "MemAvailable:") {
			r.HostMemoryAvailable = strings.TrimSpace(strings.TrimPrefix(line, "MemAvailable:"))
		}
	}
	raw, err = os.ReadFile("/proc/self/status")
	if err != nil {
		return r, err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "Cpus_allowed_list:") {
			r.CPUAffinity = strings.TrimSpace(strings.TrimPrefix(line, "Cpus_allowed_list:"))
		}
	}
	for _, name := range []string{"cpu.stat", "io.stat", "memory.current", "memory.peak", "pids.current"} {
		raw, err = os.ReadFile(filepath.Join("/sys/fs/cgroup", group, name))
		if os.IsNotExist(err) && name == "io.stat" {
			r.CgroupFiles[name] = "UNAVAILABLE: io controller is not delegated to this user service"
			continue
		}
		if err != nil {
			return r, err
		}
		r.CgroupFiles[name] = string(raw)
	}
	return r, nil
}
