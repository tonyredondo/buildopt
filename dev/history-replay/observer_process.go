//go:build linux && amd64

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type ObserverLaunch struct {
	Schema         string          `json:"schema"`
	Policy         Binding         `json:"policy"`
	Parent         ProcessIdentity `json:"parent"`
	Cgroup         string          `json:"cgroup"`
	Path           string          `json:"path"`
	NativeAffinity string          `json:"nativeAffinity"`
}

type ObserverAbort struct {
	Parent ProcessIdentity `json:"parent"`
	At     Stamp           `json:"at"`
	Reason string          `json:"reason"`
}

func checkObserverAbort(evidence string, parent ProcessIdentity, start Stamp) error {
	var abort ObserverAbort
	err := readJSON(filepath.Join(evidence, "observer-abort.json"), &abort)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	// A queued abort may precede native startup. It still belongs to this exact
	// request directory and authenticated driver and must stop that request.
	if !equalJSON(abort.Parent, parent) || abort.At.Boot != start.Boot || abort.At.NS <= 0 || abort.Reason == "" {
		return errors.New("invalid observer abort ownership")
	}
	return fmt.Errorf("observer requested cancellation: %s", abort.Reason)
}

type ObservationReceipt struct {
	Schema         string          `json:"schema"`
	Policy         Binding         `json:"policy"`
	Launch         Binding         `json:"launch"`
	Start          Stamp           `json:"start"`
	End            Stamp           `json:"end"`
	Sampler        ProcessIdentity `json:"sampler"`
	SamplerCPUNS   int64           `json:"samplerCPUNS"`
	WriterCPUNS    int64           `json:"writerCPUNS"`
	HeartbeatCPUNS int64           `json:"heartbeatCPUNS"`
	ExitCode       int             `json:"exitCode"`
	Error          string          `json:"error"`
	Artifacts      []Binding       `json:"artifacts"`
}

type processObserver struct {
	cmd          *exec.Cmd
	control, log *os.File
	done         chan struct{}
	stopOnce     sync.Once
	err          error
	receipt      ObservationReceipt
	path         string
}

func startProcessObserver(m Manifest, worker workerSession, evidence string) (*processObserver, error) {
	policy, err := readObserverPolicy(m)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(evidence, "proc-samples.jsonl")
	cfg := ObserverLaunch{"buildopt.history-replay/observer-launch/v1", m.Observer, mustIdentity(), worker.Cgroup, path, m.Affinity}
	b, err := saveBound(filepath.Join(evidence, "observer-launch.json"), cfg)
	if err != nil {
		return nil, err
	}
	input, control, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	ready, notify, err := os.Pipe()
	if err != nil {
		input.Close()
		control.Close()
		return nil, err
	}
	log, err := os.OpenFile(filepath.Join(evidence, "observer.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		input.Close()
		control.Close()
		ready.Close()
		notify.Close()
		return nil, err
	}
	cmd := exec.Command("/usr/bin/taskset", "-c", policy.SamplerAffinity, m.Executable.Path, "observer", b.Path, b.SHA256)
	cmd.ExtraFiles = []*os.File{input, notify}
	cmd.Stdout = log
	cmd.Stderr = log
	o := &processObserver{cmd: cmd, control: control, log: log, done: make(chan struct{}), path: path, receipt: ObservationReceipt{Schema: "buildopt.history-replay/observation/v1", Policy: m.Observer, Launch: b, Start: stamp(), Artifacts: []Binding{}}}
	err = cmd.Start()
	input.Close()
	notify.Close()
	if err != nil {
		control.Close()
		ready.Close()
		log.Close()
		return nil, err
	}
	o.receipt.Sampler, err = processIdentity(cmd.Process.Pid)
	go func() { o.err = cmd.Wait(); close(o.done) }()
	if err == nil {
		err = ready.SetReadDeadline(time.Now().Add(8 * time.Second))
	}
	if err == nil {
		var token [1]byte
		_, err = io.ReadFull(ready, token[:])
		if err == nil && token[0] != 1 {
			err = errors.New("invalid observer ready signal")
		}
	}
	ready.Close()
	if err != nil {
		o.stop()
		_, closeErr := o.finish()
		return nil, errors.Join(err, closeErr)
	}
	// Read after exec/readiness: taskset may still be the command immediately
	// after Start, even though the PID and start ticks are already stable.
	o.receipt.Sampler, err = processIdentity(cmd.Process.Pid)
	if err != nil {
		o.stop()
		_, closeErr := o.finish()
		return nil, errors.Join(err, closeErr)
	}
	return o, nil
}

func (o *processObserver) stop() { o.stopOnce.Do(func() { o.control.Close() }) }

func (o *processObserver) finish() (ObservationReceipt, error) {
	o.stop()
	select {
	case <-o.done:
	case <-time.After(12 * time.Second):
		o.cmd.Process.Kill()
		<-o.done
		o.err = errors.Join(o.err, errors.New("observer shutdown deadline"))
	}
	o.err = errors.Join(o.err, o.log.Close())
	o.receipt.End = stamp()
	if state := o.cmd.ProcessState; state != nil {
		o.receipt.ExitCode = state.ExitCode()
		o.receipt.SamplerCPUNS = (state.UserTime() + state.SystemTime()).Nanoseconds()
	}
	var report observerReport
	if err := readJSON(o.path+".phases.json", &report); err != nil {
		o.err = errors.Join(o.err, err)
	} else {
		o.receipt.WriterCPUNS = report.Persistence.WriterCPUNS
		o.receipt.HeartbeatCPUNS = report.ExternalHeartbeatCPUNS
		if report.Error != "" {
			o.err = errors.Join(o.err, errors.New(report.Error))
		}
	}
	paths, err := filepath.Glob(o.path + "*")
	o.err = errors.Join(o.err, err)
	for _, path := range paths {
		b, err := bind(path)
		if err != nil {
			o.err = errors.Join(o.err, err)
		} else {
			o.receipt.Artifacts = append(o.receipt.Artifacts, b)
		}
	}
	if o.err != nil {
		o.receipt.Error = o.err.Error()
	}
	_, saveErr := saveBound(filepath.Join(filepath.Dir(o.path), "observation.json"), o.receipt)
	return o.receipt, errors.Join(o.err, saveErr)
}

func serveProcessObserver(path, hash string) error {
	if err := checkBinding(Binding{path, hash}); err != nil {
		return err
	}
	var cfg ObserverLaunch
	if err := readJSON(path, &cfg); err != nil {
		return err
	}
	if cfg.Schema != "buildopt.history-replay/observer-launch/v1" || !sameProcess(cfg.Parent) || !filepath.IsAbs(cfg.Path) || filepath.Dir(cfg.Path) != filepath.Dir(path) {
		return errors.New("invalid observer ownership")
	}
	p, err := readObserverPolicy(Manifest{Schema: observedManifestSchema, Observer: cfg.Policy, Affinity: cfg.NativeAffinity})
	if err != nil {
		return err
	}
	return runProcessObserver(cfg, p, samplerOptions{maxRows: p.MaxRows, maxBytes: p.TotalBytes, heartbeatAffinity: p.HeartbeatAffinity})
}

func runProcessObserver(cfg ObserverLaunch, p ObserverPolicy, options samplerOptions) error {
	identity, err := processIdentity(os.Getpid())
	if err != nil {
		return err
	}
	masks, err := threadAffinities(identity.PID)
	if err != nil {
		return err
	}
	expected, err := parseCPUSet(p.SamplerAffinity)
	if err != nil {
		return err
	}
	for _, mask := range masks {
		actual, err := parseCPUSet(mask)
		if err != nil || actual != expected {
			return errors.New("sampler affinity differs")
		}
	}
	if err = observerSave(cfg.Path+".sampler.json", map[string]any{"identity": identity, "threadAffinities": masks}); err != nil {
		return err
	}
	control := os.NewFile(3, "observer lifetime")
	ready := os.NewFile(4, "observer ready")
	defer control.Close()
	defer ready.Close()
	stop := make(chan struct{})
	go func() { io.Copy(io.Discard, control); close(stop) }()
	config := p.writer()
	if options.buffered == nil {
		options.buffered = &config
	}
	options.ready = func() error { _, err := ready.Write([]byte{1}); return errors.Join(err, ready.Close()) }
	return observeSampler(workerSession{Cgroup: cfg.Cgroup}, cfg.Path, stop, cgroupProcesses, options)
}

// A failed observer closes the owned workflow and cannot advance the pair.
func invokeObserved(worker workerSession, q ProcessRequest, o *processObserver, onExit func(ProcessReceipt)) (ProcessReceipt, error) {
	type result struct {
		receipt ProcessReceipt
		err     error
	}
	finished := make(chan result, 1)
	go func() {
		receipt, err := worker.invoke(q, func(r ProcessReceipt) { onExit(r); o.stop() })
		finished <- result{receipt, err}
	}()
	select {
	case result := <-finished:
		return result.receipt, result.err
	case <-o.done:
		if o.err != nil {
			// Keep the supervisor alive long enough to stamp and persist native
			// completion. Stopping its unit here would erase launch accounting.
			_, stopErr := saveBound(filepath.Join(q.Evidence, "observer-abort.json"), ObserverAbort{worker.Config.Parent, stamp(), o.err.Error()})
			if stopErr != nil {
				stopErr = errors.Join(stopErr, worker.stop())
			}
			result := <-finished
			return result.receipt, errors.Join(fmt.Errorf("observer failed: %w", o.err), stopErr, result.err)
		}
		result := <-finished
		return result.receipt, result.err
	}
}
