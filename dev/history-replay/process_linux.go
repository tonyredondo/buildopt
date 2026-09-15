//go:build linux && amd64

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

type Stamp struct {
	NS   int64  `json:"ns"`
	UTC  string `json:"utc"`
	Boot string `json:"boot"`
}

func stamp() Stamp {
	var ts unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_BOOTTIME, &ts); err != nil {
		panic(err)
	}
	boot, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		panic(err)
	}
	return Stamp{ts.Nano(), time.Now().UTC().Format(time.RFC3339Nano), strings.TrimSpace(string(boot))}
}

type ProcessIdentity struct {
	PID        int      `json:"pid"`
	StartTicks uint64   `json:"startTicks"`
	Cgroup     string   `json:"cgroup"`
	Command    []string `json:"command"`
}

func processIdentity(pid int) (ProcessIdentity, error) {
	p := ProcessIdentity{PID: pid, Command: []string{}}
	prefix := fmt.Sprintf("/proc/%d/", pid)
	b, err := os.ReadFile(prefix + "stat")
	if err != nil {
		return p, err
	}
	end := strings.LastIndexByte(string(b), ')')
	if end < 0 {
		return p, errors.New("bad process stat")
	}
	fields := strings.Fields(string(b[end+1:]))
	if len(fields) < 20 {
		return p, errors.New("short process stat")
	}
	p.StartTicks, err = strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		return p, err
	}
	b, err = os.ReadFile(prefix + "cgroup")
	if err != nil {
		return p, err
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "0::") {
			p.Cgroup = strings.TrimPrefix(line, "0::")
		}
	}
	b, err = os.ReadFile(prefix + "cmdline")
	if err != nil {
		return p, err
	}
	for _, s := range strings.Split(string(b), "\x00") {
		if s != "" {
			p.Command = append(p.Command, s)
		}
	}
	return p, nil
}

func sameProcess(p ProcessIdentity) bool {
	now, err := processIdentity(p.PID)
	return err == nil && now.StartTicks == p.StartTicks && now.Cgroup == p.Cgroup
}
func cgroupProcesses(group string) ([]ProcessIdentity, error) {
	if group == "" || group == "/" {
		return nil, errors.New("unowned cgroup")
	}
	root := filepath.Join("/sys/fs/cgroup", group)
	ids := map[int]bool{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.Name() != "cgroup.procs" {
			return nil
		}
		b, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		for _, s := range strings.Fields(string(b)) {
			pid, e := strconv.Atoi(s)
			if e != nil {
				return e
			}
			ids[pid] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return identifyCgroupProcesses(group, ids, processIdentity)
}

// A process may exit after cgroup.procs is read, including between proc reads.
func identifyCgroupProcesses(group string, ids map[int]bool, identify func(int) (ProcessIdentity, error)) ([]ProcessIdentity, error) {
	out := []ProcessIdentity{}
	for pid := range ids {
		p, e := identify(pid)
		if os.IsNotExist(e) || errors.Is(e, syscall.ESRCH) {
			continue
		}
		if e != nil {
			return nil, e
		}
		if p.Cgroup == group || strings.HasPrefix(p.Cgroup, group+"/") {
			out = append(out, p)
		} else {
			return nil, errors.New("process escaped owned cgroup")
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PID < out[j].PID })
	return out, nil
}

type WorkerConfig struct {
	Schema       string          `json:"schema"`
	Manifest     Binding         `json:"manifest"`
	Executable   Binding         `json:"executable"`
	Parent       ProcessIdentity `json:"parent"`
	Unit         string          `json:"unit"`
	Socket       string          `json:"socket"`
	ArmRoot      string          `json:"armRoot"`
	EvidenceRoot string          `json:"evidenceRoot"`
	RunRoot      string          `json:"runRoot"`
	Policy       string          `json:"policy"`
	Limits       Limits          `json:"limits"`
}

type ProcessRequest struct {
	Schema      string            `json:"schema"`
	Attempt     string            `json:"attempt"`
	Directory   string            `json:"directory"`
	Command     []string          `json:"command"`
	Environment map[string]string `json:"environment"`
	Evidence    string            `json:"evidence"`
	TimeoutNS   int64             `json:"timeoutNS"`
}

type ProcessReceipt struct {
	Schema          string              `json:"schema"`
	Attempt         string              `json:"attempt"`
	Unit            string              `json:"unit"`
	InvocationID    string              `json:"invocationID"`
	Supervisor      ProcessIdentity     `json:"supervisor"`
	Process         ProcessIdentity     `json:"process"`
	Observed        []ProcessIdentity   `json:"observed"`
	Start           Stamp               `json:"start"`
	End             Stamp               `json:"end"`
	ExitCode        int                 `json:"exitCode"`
	Signal          int                 `json:"signal"`
	Outcome         string              `json:"outcome"`
	StdoutSHA256    string              `json:"stdoutSHA256"`
	StderrSHA256    string              `json:"stderrSHA256"`
	UserNS          int64               `json:"userNS"`
	SystemNS        int64               `json:"systemNS"`
	MaxRSSBytes     int64               `json:"maxRSSBytes"`
	Error           string              `json:"error"`
	ResourcesBefore ResourceObservation `json:"resourcesBefore"`
	ResourcesAfter  ResourceObservation `json:"resourcesAfter"`
}

type boundedOutput struct {
	mu        sync.Mutex
	file      *os.File
	remaining int64
	cancel    context.CancelFunc
}

func (w *boundedOutput) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if int64(len(b)) > w.remaining {
		w.cancel()
		return 0, errors.New("output allocation exceeded")
	}
	n, e := w.file.Write(b)
	w.remaining -= int64(n)
	return n, e
}

func executeOwned(cfg WorkerConfig, q ProcessRequest, onExit func(ProcessReceipt) error) (ProcessReceipt, error) {
	r := ProcessReceipt{Schema: recordSchema, Attempt: q.Attempt, Unit: cfg.Unit, InvocationID: os.Getenv("INVOCATION_ID"), Observed: []ProcessIdentity{}, Process: ProcessIdentity{Command: []string{}}, ExitCode: -1, Outcome: "START_FAILURE"}
	r.ResourcesBefore.CgroupFiles = map[string]string{}
	r.ResourcesAfter.CgroupFiles = map[string]string{}
	var err error
	r.Supervisor, err = processIdentity(os.Getpid())
	if err != nil {
		return r, err
	}
	if !strings.HasSuffix(r.Supervisor.Cgroup, "/"+cfg.Unit) || r.InvocationID == "" {
		return r, errors.New("supervisor lacks owned systemd containment")
	}
	if q.Schema != recordSchema || !namePattern.MatchString(q.Attempt) || q.TimeoutNS <= 0 || q.TimeoutNS > cfg.Limits.MaxRequestNS || !inside(cfg.ArmRoot, q.Directory) || !inside(cfg.EvidenceRoot, q.Evidence) || len(q.Command) == 0 {
		return r, errors.New("invalid worker request")
	}
	var manifest Manifest
	if err = readJSON(cfg.Manifest.Path, &manifest); err != nil {
		return r, err
	}
	var affinity *affinityScope
	if !absentBinding(manifest.Observer) {
		policy, e := readObserverPolicy(manifest)
		if e != nil {
			return r, e
		}
		affinity, e = newAffinityScope(manifest.Affinity, policy.SamplerAffinity)
		if e != nil {
			return r, e
		}
		defer affinity.release()
	}
	out, err := os.OpenFile(filepath.Join(q.Evidence, "stdout.log"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return r, err
	}
	defer out.Close()
	errout, err := os.OpenFile(filepath.Join(q.Evidence, "stderr.log"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return r, err
	}
	defer errout.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(q.TimeoutNS))
	defer cancel()
	c := exec.CommandContext(ctx, q.Command[0], q.Command[1:]...)
	c.Dir = q.Directory
	c.Env = []string{}
	keys := []string{}
	for k := range q.Environment {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		c.Env = append(c.Env, k+"="+q.Environment[k])
	}
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
	c.Cancel = func() error {
		if c.Process != nil {
			return syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
		}
		return nil
	}
	c.WaitDelay = 2 * time.Second
	c.Stdout = &boundedOutput{file: out, remaining: 64 << 20, cancel: cancel}
	c.Stderr = &boundedOutput{file: errout, remaining: 64 << 20, cancel: cancel}
	r.ResourcesBefore, err = observeResources(r.Supervisor.Cgroup)
	if err != nil {
		return r, err
	}
	r.Start = stamp()
	if affinity == nil {
		err = c.Start()
	} else {
		err = affinity.start(c)
	}
	if err == nil {
		r.Process, err = processIdentity(c.Process.Pid)
		if err != nil {
			cancel()
		}
		seen := map[string]ProcessIdentity{}
		var watchErr error
		disk := newLiveDiskGuard(cfg.RunRoot, cfg.Limits)
		diskReceipt := DiskObserverReceipt{Schema: "buildopt.disk-observer/v1", Attempt: q.Attempt, Root: cfg.RunRoot, MaxBytes: cfg.Limits.MaxBytes, MinimumFreeBytes: cfg.Limits.MinimumFreeBytes, Begin: stamp()}
		done := make(chan struct{})
		finished := make(chan struct{})
		go func() {
			defer close(finished)
			watchErr = watchOwnedRequest(done, func() error {
				if !sameProcess(cfg.Parent) {
					return errors.New("driver disappeared")
				}
				if !absentBinding(manifest.Observer) {
					if err := checkObserverAbort(q.Evidence, cfg.Parent, r.Start); err != nil {
						return err
					}
				}
				return freeDiskGuard(cfg.RunRoot, cfg.Limits.MinimumFreeBytes)
			}, func(stop <-chan struct{}) error {
				return disk.Check(stop)
			}, func() error {
				ps, e := cgroupProcesses(r.Supervisor.Cgroup)
				if e != nil {
					return e
				}
				for _, p := range ps {
					seen[fmt.Sprintf("%d/%d", p.PID, p.StartTicks)] = p
				}
				return nil
			}, cancel)
			diskReceipt.Status = disk.Status()
			watchErr = errors.Join(watchErr, disk.Close())
			diskReceipt.End = stamp()
		}()
		waitErr := c.Wait()
		r.End = stamp()
		close(done)
		<-finished
		diskReceipt.End = stamp()
		if e := writeExclusive(filepath.Join(q.Evidence, "disk-observer.json"), jsonBytes(diskReceipt), 0600); e != nil {
			return r, e
		}
		for _, p := range seen {
			r.Observed = append(r.Observed, p)
		}
		sort.Slice(r.Observed, func(i, j int) bool { return r.Observed[i].PID < r.Observed[j].PID })
		if err == nil {
			err = waitErr
		}
		if watchErr != nil {
			err = watchErr
		}
		if c.ProcessState != nil {
			r.ExitCode = c.ProcessState.ExitCode()
			r.UserNS = c.ProcessState.UserTime().Nanoseconds()
			r.SystemNS = c.ProcessState.SystemTime().Nanoseconds()
			if u, ok := c.ProcessState.SysUsage().(*syscall.Rusage); ok {
				r.MaxRSSBytes = u.Maxrss * 1024
			}
			if s, ok := c.ProcessState.Sys().(syscall.WaitStatus); ok && s.Signaled() {
				r.Signal = int(s.Signal())
			}
		}
		r.Outcome = "EXITED"
		if ctx.Err() != nil {
			r.Outcome = "CANCELLED"
		}
		if watchErr != nil {
			r.Outcome = "OWNERSHIP_OR_LIMIT_FAILURE"
		}
	}
	if r.End.NS == 0 {
		r.End = stamp()
	}
	if err != nil {
		r.Error = err.Error()
	}
	// Report actual child completion before durable research receipt writes.
	// Even a disconnected driver must not prevent retaining native evidence.
	var notifyErr error
	if onExit != nil {
		notifyErr = onExit(r)
	}
	r.ResourcesAfter, err = observeResources(r.Supervisor.Cgroup)
	if err != nil {
		return r, err
	}
	if err = writeExclusive(filepath.Join(q.Evidence, "native-start.json"), jsonBytes(struct {
		Request    ProcessRequest  `json:"request"`
		Supervisor ProcessIdentity `json:"supervisor"`
		At         Stamp           `json:"at"`
	}{q, r.Supervisor, r.Start}), 0600); err != nil {
		return r, err
	}
	if r.Process.PID > 0 {
		if err = writeExclusive(filepath.Join(q.Evidence, "native-pid.json"), jsonBytes(r.Process), 0600); err != nil {
			return r, err
		}
	}
	if e := out.Sync(); e != nil {
		return r, e
	}
	if e := errout.Sync(); e != nil {
		return r, e
	}
	r.StdoutSHA256, err = fileDigest(out.Name())
	if err != nil {
		return r, err
	}
	r.StderrSHA256, err = fileDigest(errout.Name())
	if err != nil {
		return r, err
	}
	if err = writeExclusive(filepath.Join(q.Evidence, "native-finish.json"), jsonBytes(r), 0600); err != nil {
		return r, err
	}
	return r, notifyErr
}

func serveWorker(configPath, expected string) error {
	if err := checkBinding(Binding{configPath, expected}); err != nil {
		return err
	}
	var cfg WorkerConfig
	if err := readJSON(configPath, &cfg); err != nil {
		return err
	}
	if cfg.Schema != recordSchema || !sameProcess(cfg.Parent) {
		return errors.New("invalid worker parent")
	}
	if err := checkBinding(cfg.Manifest); err != nil {
		return err
	}
	if err := checkBinding(cfg.Executable); err != nil {
		return err
	}
	self, err := os.Executable()
	if err != nil || self != cfg.Executable.Path {
		return errors.New("worker executable identity differs")
	}
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: cfg.Socket, Net: "unix"})
	if err != nil {
		return err
	}
	defer l.Close()
	for {
		if !sameProcess(cfg.Parent) {
			return errors.New("driver ended")
		}
		if err = l.SetDeadline(time.Now().Add(200 * time.Millisecond)); err != nil {
			return err
		}
		conn, err := l.AcceptUnix()
		if err != nil {
			if e, ok := err.(net.Error); ok && e.Timeout() {
				continue
			}
			return err
		}
		// Abstract Linux sockets have no filesystem permissions. Require the
		// exact frozen driver process, in addition to immutable request evidence.
		raw, err := conn.SyscallConn()
		if err != nil {
			conn.Close()
			return err
		}
		var cred *unix.Ucred
		var credentialError error
		if err = raw.Control(func(fd uintptr) {
			cred, credentialError = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
		}); err != nil {
			conn.Close()
			return err
		}
		if credentialError != nil || cred == nil || int(cred.Pid) != cfg.Parent.PID || cred.Uid != uint32(os.Getuid()) {
			conn.Close()
			return errors.New("unexpected worker client")
		}
		if err = conn.SetDeadline(time.Now().Add(time.Duration(cfg.Limits.MaxRequestNS) + 10*time.Second)); err != nil {
			conn.Close()
			return err
		}
		b, err := io.ReadAll(io.LimitReader(conn, 4<<20))
		if err != nil {
			conn.Close()
			return err
		}
		var q ProcessRequest
		if err = decodeStrict(b, &q); err != nil {
			conn.Close()
			return err
		}
		r, runErr := executeOwned(cfg, q, func(early ProcessReceipt) error { return json.NewEncoder(conn).Encode(early) })
		if runErr != nil {
			conn.Close()
			return runErr
		}
		err = json.NewEncoder(conn).Encode(r)
		conn.Close()
		if err != nil {
			return err
		}
		if cfg.Policy == "REQUEST" || r.Outcome != "EXITED" {
			return nil
		}
	}
}

type workerSession struct {
	Config     WorkerConfig
	ConfigPath string
	Cgroup     string
}

type SessionClosure struct {
	Schema    string            `json:"schema"`
	Unit      string            `json:"unit"`
	Cgroup    string            `json:"cgroup"`
	At        Stamp             `json:"at"`
	Remaining []ProcessIdentity `json:"remaining"`
}

func ownedUnitCgroup(unit string) (string, error) {
	if !strings.HasPrefix(unit, "buildopt-replay-") || !strings.HasSuffix(unit, ".service") || strings.ContainsAny(unit, "/\x00\r\n") {
		return "", errors.New("invalid owned service name")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	b, err := exec.CommandContext(ctx, "systemctl", "--user", "show", unit, "--property=ControlGroup", "--value").Output()
	if err != nil {
		return "", err
	}
	group := strings.TrimSpace(string(b))
	if group != "" && !strings.HasSuffix(group, "/"+unit) {
		return "", errors.New("service cgroup identity differs")
	}
	return group, nil
}

func startWorker(m Manifest, manifest Binding, rep int, arm string, generation int, evidence string) (workerSession, error) {
	s := workerSession{}
	parent, err := processIdentity(os.Getpid())
	if err != nil {
		return s, err
	}
	identity := digest([]byte(m.RunRoot))[:16]
	unit := fmt.Sprintf("buildopt-replay-%s-r%d-%s-%d.service", identity, rep, arm, generation)
	s.Config = WorkerConfig{recordSchema, manifest, m.Executable, parent, unit, "@" + strings.TrimSuffix(unit, ".service"), filepath.Join(m.RunRoot, fmt.Sprintf("r%d", rep), arm), filepath.Join(m.RunRoot, "attempts"), m.RunRoot, m.DaemonPolicy, m.Limits}
	s.ConfigPath = filepath.Join(evidence, "worker-config.json")
	if err = writeExclusive(s.ConfigPath, jsonBytes(s.Config), 0600); err != nil {
		return s, err
	}
	h, err := fileDigest(s.ConfigPath)
	if err != nil {
		return s, err
	}
	workerAffinity := m.Affinity
	if !absentBinding(m.Observer) {
		policy, err := readObserverPolicy(m)
		if err != nil {
			return s, err
		}
		workerAffinity = policy.SamplerAffinity
	}
	args := []string{"--user", "--quiet", "--collect", "--service-type=exec", "--unit=" + unit, "--property=KillMode=control-group", "--property=TimeoutStopSec=2s", "--property=Restart=no", "--property=RuntimeMaxSec=" + strconv.FormatInt((m.Limits.MaxRunNS+int64(time.Second)-1)/int64(time.Second), 10) + "s", "--property=CPUAffinity=" + workerAffinity, "--property=StandardOutput=append:" + filepath.Join(evidence, "worker.log"), "--property=StandardError=append:" + filepath.Join(evidence, "worker.log"), "--", m.Executable.Path, "worker", s.ConfigPath, h}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	b, err := exec.CommandContext(ctx, "systemd-run", args...).CombinedOutput()
	if err != nil {
		return s, fmt.Errorf("start owned service: %w: %s", err, b)
	}
	// Checking active state does not connect to the authenticated command socket.
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		b, e := exec.CommandContext(ctx, "systemctl", "--user", "show", unit, "--property=ActiveState,ControlGroup", "--no-pager").Output()
		if e != nil {
			return s, e
		}
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "ControlGroup=") {
				s.Cgroup = strings.TrimPrefix(line, "ControlGroup=")
			}
		}
		if strings.Contains(string(b), "ActiveState=active") && strings.HasSuffix(s.Cgroup, "/"+unit) {
			return s, nil
		}
		if strings.Contains(string(b), "ActiveState=failed") {
			return s, errors.New("owned worker failed during startup")
		}
		time.Sleep(20 * time.Millisecond)
	}
	return s, errors.New("owned worker did not start")
}

func (s workerSession) stop() error {
	if !strings.HasSuffix(s.Cgroup, "/"+s.Config.Unit) || !strings.HasPrefix(s.Config.Unit, "buildopt-replay-") {
		return errors.New("refuse to stop unowned service")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	_ = exec.CommandContext(ctx, "systemctl", "--user", "stop", s.Config.Unit).Run()
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
		ps, e := cgroupProcesses(s.Cgroup)
		if os.IsNotExist(e) || e == nil && len(ps) == 0 {
			path := filepath.Join(filepath.Dir(s.ConfigPath), "closed.json")
			if _, err := os.Stat(path); err == nil {
				return nil
			}
			return writeExclusive(path, jsonBytes(SessionClosure{recordSchema, s.Config.Unit, s.Cgroup, stamp(), []ProcessIdentity{}}), 0600)
		}
		if e != nil {
			return e
		}
		time.Sleep(20 * time.Millisecond)
	}
	return errors.New("owned service still has processes")
}

func (s workerSession) invoke(q ProcessRequest, onExit func(ProcessReceipt)) (ProcessReceipt, error) {
	var receipt ProcessReceipt
	var conn *net.UnixConn
	var err error
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		conn, err = net.DialUnix("unix", nil, &net.UnixAddr{Name: s.Config.Socket, Net: "unix"})
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		return receipt, err
	}
	defer conn.Close()
	if err = conn.SetDeadline(time.Now().Add(time.Duration(q.TimeoutNS) + 8*time.Second)); err != nil {
		return receipt, err
	}
	if _, err = conn.Write(jsonBytes(q)); err != nil {
		return receipt, err
	}
	if err = conn.CloseWrite(); err != nil {
		return receipt, err
	}
	d := json.NewDecoder(io.LimitReader(conn, 8<<20))
	var raw json.RawMessage
	if err = d.Decode(&raw); err != nil {
		return receipt, err
	}
	var early ProcessReceipt
	if err = decodeStrict(raw, &early); err != nil {
		return receipt, err
	}
	if early.Schema != recordSchema || early.Attempt != q.Attempt {
		return receipt, errors.New("unexpected native completion frame")
	}
	if onExit != nil {
		onExit(early)
	}
	if err = d.Decode(&raw); err != nil {
		return receipt, err
	}
	if err = decodeStrict(raw, &receipt); err != nil {
		return receipt, err
	}
	withoutHashes := receipt
	withoutHashes.ResourcesAfter = early.ResourcesAfter
	withoutHashes.StdoutSHA256 = ""
	withoutHashes.StderrSHA256 = ""
	if !equalJSON(withoutHashes, early) {
		return receipt, errors.New("native completion changed while persisting evidence")
	}
	if err = d.Decode(&raw); err != io.EOF {
		return receipt, errors.New("extra native response frame")
	}
	return receipt, nil
}
