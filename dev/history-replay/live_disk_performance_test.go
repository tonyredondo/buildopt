//go:build linux && amd64 && replay_disk_perf

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// Explicit allocation: two sizes, old/new, two repetitions, twenty checks per
// case. This is not part of regular CI and does not launch native workflows.
func TestLiveDiskAllocatedPerformance(t *testing.T) {
	output := os.Getenv("BUILDOPT_DISK_PERF_OUTPUT")
	if output == "" {
		t.Fatal("BUILDOPT_DISK_PERF_OUTPUT must name a new receipt")
	}
	f, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	check := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	cpu := func() int64 {
		var u unix.Rusage
		check(unix.Getrusage(unix.RUSAGE_SELF, &u))
		return u.Utime.Nano() + u.Stime.Nano()
	}
	rows := []map[string]interface{}{}
	for _, count := range []int{16, 32768} {
		root := t.TempDir()
		for i := 0; i < count; i++ {
			dir := filepath.Join(root, "retained", fmt.Sprintf("%04d", i/256))
			if i%256 == 0 {
				check(os.MkdirAll(dir, 0700))
			}
			check(os.WriteFile(filepath.Join(dir, fmt.Sprintf("%06d", i)), []byte("retained output"), 0600))
		}
		active := filepath.Join(root, "active")
		check(os.Mkdir(active, 0700))
		file := filepath.Join(active, "growing")
		check(os.WriteFile(file, nil, 0600))
		byVersion := map[string][]int64{}
		for repeat := 0; repeat < 2; repeat++ {
			order := []string{"full", "notified"}
			if repeat == 1 {
				order = []string{"notified", "full"}
			}
			for _, version := range order {
				check(os.Truncate(file, 0))
				runtime.GC()
				g := newLiveDiskGuard(root, Limits{MaxBytes: 1 << 30})
				beginCPU, begin := cpu(), time.Now()
				var firstCPU, firstWall int64
				for i := 0; i < 20; i++ {
					check(os.Truncate(file, int64(i+1)))
					tmp := filepath.Join(active, "temporary")
					if i%2 == 0 {
						check(os.Mkdir(tmp, 0700))
						check(os.WriteFile(filepath.Join(tmp, "output"), []byte("temporary"), 0600))
					} else {
						check(os.RemoveAll(tmp))
					}
					if version == "full" {
						check(diskGuard(root, g.limits))
					} else {
						check(g.Check(nil))
					}
					if i == 0 {
						firstCPU = cpu() - beginCPU
						firstWall = time.Since(begin).Nanoseconds()
					}
				}
				status := g.Status()
				check(g.Close())
				elapsed, work := time.Since(begin).Nanoseconds(), cpu()-beginCPU
				row := map[string]interface{}{"files": count, "version": version, "repeat": repeat + 1, "checks": 20,
					"elapsedNS": elapsed, "cpuNS": work, "firstCheckElapsedNS": firstWall, "firstCheckCPUNS": firstCPU, "observer": status}
				rows = append(rows, row)
				check(json.NewEncoder(f).Encode(row))
				check(f.Sync())
				t.Logf("files=%d version=%s repetition=%d CPU=%.6fs elapsed=%.6fs", count, version, repeat+1, float64(work)/1e9, float64(elapsed)/1e9)
				byVersion[version] = append(byVersion[version], work)
				if version == "notified" && (g.fallback != "" || g.fullScans != 0) {
					t.Fatalf("fixture lost incremental accounting: %+v", status)
				}
			}
		}
		if count == 32768 {
			for i := 0; i < 2; i++ {
				if byVersion["notified"][i] >= byVersion["full"][i] {
					t.Fatalf("large fixture did not reduce total CPU: %+v", byVersion)
				}
			}
		}
	}
	if len(rows) != 8 {
		t.Fatal("measurement allocation differs")
	}
}
