package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type environmentEvidence struct {
	Boot              string    `json:"bootId"`
	Started           float64   `json:"startedBootSeconds"`
	Completed         float64   `json:"completedBootSeconds"`
	QuiescenceSeconds int       `json:"quiescenceSeconds"`
	Samples           []float64 `json:"samplesMilliseconds"`
}
type reviewEvidence struct {
	BootSeconds float64 `json:"bootSeconds"`
	Decision    string  `json:"decision"`
	Note        string  `json:"note"`
}

func verifyHost(ctx context.Context) error {
	profile, err := exec.CommandContext(ctx, "powerprofilesctl", "get").Output()
	if err != nil || strings.TrimSpace(string(profile)) != "performance" {
		return errors.New("performance power profile unavailable")
	}
	driver, err := os.ReadFile("/sys/devices/system/cpu/cpu0/cpufreq/scaling_driver")
	if err != nil || strings.TrimSpace(string(driver)) != "intel_pstate" {
		return errors.New("unexpected CPU scaling driver")
	}
	for cpu := 0; cpu < 4; cpu++ {
		preference, err := os.ReadFile(fmt.Sprintf("/sys/devices/system/cpu/cpu%d/cpufreq/energy_performance_preference", cpu))
		if err != nil || strings.TrimSpace(string(preference)) != "performance" {
			return errors.New("CPU EPP is not performance")
		}
	}
	supplies, err := os.ReadDir("/sys/class/power_supply")
	if err != nil {
		return err
	}
	mains := false
	for _, supply := range supplies {
		root := filepath.Join("/sys/class/power_supply", supply.Name())
		kind, err := os.ReadFile(filepath.Join(root, "type"))
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(kind)) != "Mains" {
			continue
		}
		online, err := os.ReadFile(filepath.Join(root, "online"))
		if err == nil && strings.TrimSpace(string(online)) == "1" {
			mains = true
		}
	}
	if !mains {
		return errors.New("mains power is not confirmed")
	}
	return nil
}

func stabilize(ctx context.Context, root string, state campaign) error {
	start, err := bootClock()
	if err != nil {
		return err
	}
	left, err := remaining(state, start)
	if err != nil {
		return err
	}
	if left <= 120 {
		return errors.New("insufficient remaining window for quiescence")
	}
	if err = verifyHost(ctx); err != nil {
		return err
	}
	if err = exec.CommandContext(ctx, "/usr/bin/taskset", "--all-tasks", "--pid", "--cpu-list", "0-3", strconv.Itoa(os.Getpid())).Run(); err != nil {
		return errors.New("cannot bind stability probe to CPUs 0-3")
	}
	timer := time.NewTimer(120 * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}
	evidence := environmentEvidence{Boot: start.Boot, Started: start.Seconds, QuiescenceSeconds: 120, Samples: []float64{}}
	buffer := make([]byte, 128<<20)
	for sample := 0; sample < 7; sample++ {
		if err = ctx.Err(); err != nil {
			return err
		}
		begin := time.Now()
		sum := sha256.Sum256(buffer)
		if sum == [32]byte{} {
			return errors.New("invalid stability digest")
		}
		evidence.Samples = append(evidence.Samples, float64(time.Since(begin).Nanoseconds())/1e6)
	}
	end, err := bootClock()
	if err != nil {
		return err
	}
	evidence.Completed = end.Seconds
	if err = writeNewJSON(filepath.Join(root, "environment.json"), evidence); err != nil {
		return err
	}
	return checkEnvironment(state, evidence)
}

func checkEnvironment(state campaign, evidence environmentEvidence) error {
	if evidence.Boot != state.Boot || evidence.Started < state.Started || evidence.Completed-evidence.Started < 120 || evidence.Completed >= state.Deadline || evidence.QuiescenceSeconds != 120 || len(evidence.Samples) != 7 {
		return errors.New("invalid environment evidence")
	}
	minimum, maximum := evidence.Samples[0], evidence.Samples[0]
	for _, sample := range evidence.Samples {
		if sample <= 0 {
			return errors.New("invalid stability sample")
		}
		minimum = min(minimum, sample)
		maximum = max(maximum, sample)
	}
	if maximum/minimum > 1.15 {
		return errors.New("INCOMPLETE_PERFORMANCE_ENVIRONMENT")
	}
	return nil
}

func reviewDue(root string, state campaign, now clockReading) error {
	if now.Seconds < state.ReviewAt {
		return nil
	}
	var review reviewEvidence
	if err := readJSON(filepath.Join(root, "review.json"), &review); err != nil {
		return errors.New("30-minute review required before another start")
	}
	if review.BootSeconds < state.ReviewAt || review.BootSeconds > now.Seconds || review.Decision != "continue" || strings.TrimSpace(review.Note) == "" {
		return errors.New("review stops execution or is invalid")
	}
	return nil
}

func seedDependencies(root string) error {
	source := filepath.Join(root, "homes", "prefetch-b")
	destination := filepath.Join(root, "dependency-seed")
	if err := os.Mkdir(destination, 0700); err != nil {
		return err
	}
	files := []fileBinding{}
	for _, relative := range []string{"caches/modules-2", "wrapper/dists"} {
		base := filepath.Join(source, relative)
		if _, err := os.Stat(base); err != nil {
			return err
		}
		err := filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			if !entry.Type().IsRegular() {
				return errors.New("non-regular dependency member")
			}
			if strings.HasSuffix(entry.Name(), ".lock") || strings.HasSuffix(entry.Name(), ".lck") || entry.Name() == "gc.properties" {
				return nil
			}
			rel, err := filepath.Rel(source, path)
			if err != nil {
				return err
			}
			target := filepath.Join(destination, rel)
			if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
				return err
			}
			if err = copyNewFile(path, target); err != nil {
				return err
			}
			hash, err := fileDigest(target)
			if err != nil {
				return err
			}
			files = append(files, fileBinding{filepath.ToSlash(rel), hash})
			return nil
		})
		if err != nil {
			return err
		}
	}
	return writeNewJSON(filepath.Join(destination, "inventory.json"), files)
}

func copyNewFile(source, target string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil {
		return err
	}
	output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err = io.Copy(output, input); err != nil {
		output.Close()
		return err
	}
	if err = output.Sync(); err != nil {
		output.Close()
		return err
	}
	return output.Close()
}

func prepareHome(root, home string, prefetch, reuse bool) error {
	if reuse {
		if _, err := os.Stat(home); err != nil {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(home), 0700); err != nil {
		return err
	}
	if err := os.Mkdir(home, 0700); err != nil {
		return err
	}
	if prefetch {
		return nil
	}
	seed := filepath.Join(root, "dependency-seed")
	var files []fileBinding
	if err := readJSON(filepath.Join(seed, "inventory.json"), &files); err != nil {
		return err
	}
	for _, file := range files {
		source := filepath.Join(seed, file.Path)
		target := filepath.Join(home, file.Path)
		if err := contained(seed, source); err != nil {
			return err
		}
		if err := contained(home, target); err != nil {
			return err
		}
		hash, err := fileDigest(source)
		if err != nil || hash != file.SHA256 {
			return errors.New("dependency seed drift")
		}
		if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		if err = copyNewFile(source, target); err != nil {
			return err
		}
	}
	return nil
}

func expectedPaths(root, slot string) (string, string) {
	n, _ := strconv.Atoi(slot[1:])
	role := "native-a"
	if n == 2 && !strings.HasPrefix(slot, "M") {
		role = "native-b"
	}
	home := map[string]string{"P01": "prefetch-a", "P02": "prefetch-b", "D01": "diagnostic-a", "D02": "diagnostic-b", "M01": "materiality", "M02": "materiality"}[slot]
	return filepath.Join(root, "worktrees", role), filepath.Join(root, "homes", home)
}
