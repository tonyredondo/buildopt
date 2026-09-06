package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// A transient user service keeps all campaign children in one delegated cgroup.
// Native daemons may survive successful requests, but not campaign expiry or
// cancellation. No installed unit or host policy is changed.
type ownership struct {
	Unit         string  `json:"unit"`
	InvocationID string  `json:"invocationId"`
	Workers      string  `json:"workers"`
	Boot         string  `json:"bootId"`
	Deadline     float64 `json:"deadlineBootSeconds"`
}

type ownershipLease struct {
	PID          int     `json:"pid"`
	ProcessStart string  `json:"processStartTicks"`
	Deadline     float64 `json:"deadlineBootSeconds"`
}

func processStartTicks(pid int) (string, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return "", err
	}
	end := strings.LastIndexByte(string(data), ')')
	if end < 0 {
		return "", errors.New("invalid process identity")
	}
	fields := strings.Fields(string(data[end+1:]))
	if len(fields) < 20 {
		return "", errors.New("incomplete process identity")
	}
	return fields[19], nil
}

func checkLeases(root string, now clockReading) error {
	entries, err := os.ReadDir(filepath.Join(root, "attempts"))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || !validSlot(entry.Name()) {
			return errors.New("unknown attempt during supervision")
		}
		attempt := filepath.Join(root, "attempts", entry.Name())
		if info, err := os.Lstat(filepath.Join(attempt, "result.json")); err == nil && info.Mode().IsRegular() {
			continue
		}
		var lease ownershipLease
		if err := readJSON(filepath.Join(attempt, "lease.json"), &lease); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}
		start, err := processStartTicks(lease.PID)
		if err != nil || start != lease.ProcessStart {
			return errors.New("capture owner disappeared; reservation requires reconciliation")
		}
		if now.Seconds >= lease.Deadline {
			return errors.New("attempt lease expired")
		}
	}
	return nil
}

func ownershipUnit(root string, state campaign) string {
	sum := sha256.Sum256([]byte(root + state.PackageSHA256 + strconv.FormatFloat(state.Started, 'f', -1, 64)))
	return fmt.Sprintf("buildopt-cnc-%x.service", sum[:12])
}

func startGuardian(ctx context.Context, repo, root, packagePath string, state campaign) error {
	now, err := bootClock()
	if err != nil {
		return err
	}
	left, err := remaining(state, now)
	if err != nil {
		return err
	}
	if left < 15 {
		return errors.New("insufficient window to establish process ownership")
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	unit := ownershipUnit(root, state)
	if err := writeNewJSON(filepath.Join(root, "guard-request.json"), map[string]string{"unit": unit}); err != nil {
		return err
	}
	// The kernel/service limit is deliberately earlier than the campaign deadline.
	// The guardian also checks the immutable boot clock, including idle/human waits.
	args := []string{"--user", "--quiet", "--collect", "--service-type=exec", "--unit=" + unit,
		"--property=Delegate=yes", "--property=KillMode=control-group", "--property=TimeoutStopSec=1s",
		"--property=RuntimeMaxSec=" + strconv.Itoa(int(left)-10) + "s", "--property=Restart=no",
		"--property=StandardOutput=null", "--property=StandardError=append:" + filepath.Join(root, "guard.log"),
		"--", executable, "guard-child", "--repo", repo, "--state", root, "--package", packagePath}
	startCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if data, err := exec.CommandContext(startCtx, "systemd-run", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("start campaign guardian: %w: %s", err, data)
	}
	for {
		var owner ownership
		if err := readJSON(filepath.Join(root, "ownership.json"), &owner); err == nil {
			file, err := openOwnership(startCtx, root, state, owner)
			if err != nil {
				return err
			}
			return file.Close()
		}
		select {
		case <-startCtx.Done():
			return errors.New("guardian readiness missing; reserved unit must be reconciled, not restarted")
		case <-time.After(20 * time.Millisecond):
		}
	}
}

func guardian(ctx context.Context, root string, state campaign) error {
	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return err
	}
	entry := strings.TrimSpace(string(data))
	unit := ownershipUnit(root, state)
	if !strings.HasPrefix(entry, "0::/") || !strings.HasSuffix(entry, "/"+unit) || strings.Contains(entry, "\n") {
		return errors.New("guardian is not inside its exact transient unit")
	}
	base := filepath.Join("/sys/fs/cgroup", strings.TrimPrefix(entry, "0::"))
	workers := filepath.Join(base, "workers")
	if err := os.Mkdir(workers, 0700); err != nil {
		return err
	}
	properties, err := unitProperties(ctx, unit)
	if err != nil {
		return err
	}
	owner := ownership{unit, properties["InvocationID"], workers, state.Boot, state.Deadline}
	if err := validateOwnership(root, state, owner, properties); err != nil {
		return err
	}
	if err := writeNewJSON(filepath.Join(root, "ownership.json"), owner); err != nil {
		return err
	}
	defer killWorkers(workers)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		now, err := bootClock()
		if err != nil {
			return err
		}
		if _, err := remaining(state, now); err != nil {
			return err
		}
		if err := checkLeases(root, now); err != nil {
			_ = writeNewJSON(filepath.Join(root, "ownership-stop.json"), map[string]any{"reason": err.Error(), "bootSeconds": now.Seconds})
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func unitProperties(ctx context.Context, unit string) (map[string]string, error) {
	data, err := exec.CommandContext(ctx, "systemctl", "--user", "show", unit,
		"--property=InvocationID,ControlGroup,ActiveState,KillMode,Delegate,Restart,RuntimeMaxUSec,TimeoutStopUSec").Output()
	if err != nil {
		return nil, err
	}
	values := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || values[key] != "" {
			return nil, errors.New("ambiguous unit properties")
		}
		values[key] = value
	}
	return values, nil
}

func validateOwnership(root string, state campaign, owner ownership, values map[string]string) error {
	if owner.Unit != ownershipUnit(root, state) || owner.Boot != state.Boot || owner.Deadline != state.Deadline || len(owner.InvocationID) != 32 || owner.InvocationID != values["InvocationID"] || values["ActiveState"] != "active" || values["KillMode"] != "control-group" || values["Delegate"] != "yes" || values["Restart"] != "no" {
		return errors.New("campaign ownership identity/policy drift")
	}
	if !strings.HasSuffix(values["ControlGroup"], "/"+owner.Unit) || owner.Workers != filepath.Join("/sys/fs/cgroup", values["ControlGroup"], "workers") {
		return errors.New("campaign cgroup binding drift")
	}
	lifetime, err := time.ParseDuration(strings.ReplaceAll(strings.ReplaceAll(values["RuntimeMaxUSec"], "min", "m"), " ", ""))
	if err != nil || lifetime <= 0 || lifetime > 7200*time.Second || values["TimeoutStopUSec"] != "1s" {
		return errors.New("unbounded campaign service")
	}
	return nil
}

func openOwnership(ctx context.Context, root string, state campaign, owner ownership) (*os.File, error) {
	values, err := unitProperties(ctx, owner.Unit)
	if err != nil {
		return nil, err
	}
	if err = validateOwnership(root, state, owner, values); err != nil {
		return nil, err
	}
	fd, err := unix.Open(owner.Workers, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), owner.Workers)
	var fs unix.Statfs_t
	if err = unix.Fstatfs(fd, &fs); err != nil || fs.Type != unix.CGROUP2_SUPER_MAGIC {
		file.Close()
		return nil, errors.New("ownership is not cgroup v2")
	}
	return file, nil
}

func killWorkers(path string) error {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if errors.Is(err, unix.ENOENT) {
		return nil
	}
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	return killWorkerFD(fd)
}

func killWorkerFD(group int) error {
	// An open cgroup descriptor cannot be redirected to a later reused unit path.
	fd, err := unix.Openat(group, "cgroup.kill", unix.O_WRONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if errors.Is(err, unix.ENOENT) || errors.Is(err, unix.ENODEV) {
		return nil
	}
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	_, err = unix.Write(fd, []byte("1"))
	if errors.Is(err, unix.ENODEV) {
		return nil
	}
	return err
}

func applyOwnership(command *exec.Cmd, file *os.File) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, UseCgroupFD: true, CgroupFD: int(file.Fd())}
}
