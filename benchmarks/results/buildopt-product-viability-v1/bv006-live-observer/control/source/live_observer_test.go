//go:build linux && amd64 && replay_integration && replay_gradle

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/metrics"
	"strconv"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

type observerStamp struct {
	BootNS int64 `json:"bootNS"`
	MonoNS int64 `json:"monoNS"`
	CPUNS  int64 `json:"cpuNS"`
}

func observerClock() observerStamp {
	var boot, mono unix.Timespec
	var cpu unix.Rusage
	if err := unix.ClockGettime(unix.CLOCK_BOOTTIME, &boot); err != nil {
		panic(err)
	}
	if err := unix.ClockGettime(unix.CLOCK_MONOTONIC, &mono); err != nil {
		panic(err)
	}
	if err := unix.Getrusage(unix.RUSAGE_SELF, &cpu); err != nil {
		panic(err)
	}
	return observerStamp{boot.Nano(), mono.Nano(), cpu.Utime.Nano() + cpu.Stime.Nano()}
}

type observerBeat struct {
	At     observerStamp `json:"at"`
	LateNS int64         `json:"lateNS"`
}
type observerBeats struct {
	Rows  []observerBeat `json:"rows"`
	Error string         `json:"error"`
}

func observeBeats(stop <-chan struct{}) observerBeats {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	result := observerBeats{Rows: make([]observerBeat, 0, 100000)}
	for {
		select {
		case <-stop:
			return result
		case tick := <-ticker.C:
			if len(result.Rows) == cap(result.Rows) {
				result.Error = "heartbeat record limit"
				return result
			}
			result.Rows = append(result.Rows, observerBeat{observerClock(), time.Since(tick).Nanoseconds()})
		}
	}
}

// Histogram edges include infinity, which JSON cannot encode as a number.
// Preserve their exact numeric spellings rather than substitute finite edges.
func observerPauses() map[string]interface{} {
	samples := []metrics.Sample{{Name: "/sched/pauses/total/gc:seconds"}, {Name: "/sched/pauses/total/other:seconds"}}
	metrics.Read(samples)
	result := map[string]interface{}{}
	for _, sample := range samples {
		h := sample.Value.Float64Histogram()
		edges := make([]string, len(h.Buckets))
		for i, v := range h.Buckets {
			edges[i] = strconv.FormatFloat(v, 'g', -1, 64)
		}
		result[sample.Name] = map[string]interface{}{"buckets": edges, "counts": append([]uint64{}, h.Counts...)}
	}
	return result
}

type observerPhase struct {
	Index         int           `json:"index"`
	Wake          observerStamp `json:"wake"`
	WakeLateNS    int64         `json:"wakeLateNS"`
	ScanBegin     observerStamp `json:"scanBegin"`
	RecordReady   observerStamp `json:"recordReady"`
	EncodeBegin   observerStamp `json:"encodeBegin"`
	WriteBegin    observerStamp `json:"writeBegin"`
	WriteEnd      observerStamp `json:"writeEnd"`
	EncodeEnd     observerStamp `json:"encodeEnd"`
	Bytes         int           `json:"bytes"`
	PayloadSHA256 string        `json:"payloadSHA256"`
	WriteCalls    int           `json:"writeCalls"`
	Error         string        `json:"error"`
}

type observerWriter struct {
	sink       io.Writer
	row        *observerPhase
	total, max int64
}

func (w *observerWriter) Write(data []byte) (int, error) {
	w.row.WriteBegin = observerClock()
	w.row.WriteCalls++
	var n int
	var err error
	if int64(len(data)) > w.max-w.total {
		err = errors.New("sampler byte limit")
	} else {
		n, err = w.sink.Write(data)
	}
	w.row.WriteEnd = observerClock()
	w.row.Bytes = n
	w.total += int64(n)
	digest := sha256.Sum256(data)
	w.row.PayloadSHA256 = hex.EncodeToString(digest[:])
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	return n, err
}

type samplerOptions struct {
	maxRows   int
	maxBytes  int64
	stopAfter int // Fixed fixture completion only; zero for actual owner.
	writer    func(io.Writer) io.Writer
}

func observerSave(path string, value interface{}) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	return errors.Join(json.NewEncoder(f).Encode(value), f.Close())
}

type externalObserver struct {
	cmd     *exec.Cmd
	control *os.File
	log     *os.File
}

func startExternalObserver(path string) (*externalObserver, error) {
	executable, err := os.Executable()
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
	log, err := os.OpenFile(path+".log", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		input.Close()
		control.Close()
		ready.Close()
		notify.Close()
		return nil, err
	}
	cmd := exec.Command("/usr/bin/taskset", "-c", "10", executable, "-test.run=^TestLiveObserverHeartbeat$", "-test.timeout=1005s")
	cmd.Env = append(os.Environ(), "BUILDOPT_HEARTBEAT_OUTPUT="+path)
	cmd.ExtraFiles = []*os.File{input, notify}
	cmd.Stdout = log
	cmd.Stderr = log
	err = cmd.Start()
	input.Close()
	notify.Close()
	if err != nil {
		control.Close()
		ready.Close()
		log.Close()
		return nil, err
	}
	// The pipe is inherited only by the heartbeat. Parent death closes it too.
	ready.SetReadDeadline(time.Now().Add(5 * time.Second))
	b := make([]byte, 1)
	_, err = io.ReadFull(ready, b)
	ready.Close()
	if err != nil || b[0] != 1 {
		control.Close()
		cmd.Process.Kill()
		cmd.Wait()
		log.Close()
		return nil, errors.New("heartbeat startup failed")
	}
	return &externalObserver{cmd, control, log}, nil
}
func (h *externalObserver) stop() error {
	closeErr := h.control.Close()
	done := make(chan error, 1)
	go func() { done <- h.cmd.Wait() }()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case err := <-done:
		return errors.Join(closeErr, err, h.log.Close())
	case <-timer.C:
		killErr := h.cmd.Process.Kill()
		waitErr := <-done
		return errors.Join(errors.New("heartbeat shutdown deadline"), closeErr, killErr, waitErr, h.log.Close())
	}
}

func TestLiveObserverHeartbeat(t *testing.T) {
	path := os.Getenv("BUILDOPT_HEARTBEAT_OUTPUT")
	if path == "" {
		t.Fatal("explicit heartbeat output required")
	}
	identity, err := processIdentity(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	mask, err := threadAffinities(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	// taskset restricts the whole Go runtime before it starts.
	var affinity unix.CPUSet
	if err = unix.SchedGetaffinity(0, &affinity); err != nil || affinity.Count() != 1 || !affinity.IsSet(10) {
		t.Fatal("heartbeat CPU differs", err)
	}
	control := os.NewFile(3, "control")
	ready := os.NewFile(4, "ready")
	stop := make(chan struct{})
	go func() { io.Copy(io.Discard, control); close(stop) }()
	ready.Write([]byte{1})
	ready.Close()
	begin := observerClock()
	beats := observeBeats(stop)
	end := observerClock()
	control.Close()
	err = observerSave(path, map[string]interface{}{"schema": "buildopt.live-observer/heartbeat/v1", "identity": identity, "threadAffinities": mask, "begin": begin, "end": end, "beats": beats})
	if err != nil || beats.Error != "" {
		t.Fatal(err, beats.Error)
	}
}

// All test and owner cases consume the same live collector, encoder and file.
// Diagnostic records are buffered with hard limits, then saved after sampling.
func observeSampler(worker workerSession, path string, stop <-chan struct{}, observe func(string) ([]ProcessIdentity, error), options samplerOptions) (resultErr error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	external, err := startExternalObserver(path + ".external.json")
	if err != nil {
		f.Close()
		return err
	}
	heartStop := make(chan struct{})
	heartDone := make(chan observerBeats, 1)
	go func() { heartDone <- observeBeats(heartStop) }()
	rows := make([]observerPhase, 0, options.maxRows)
	pausesBefore := observerPauses()
	begin := observerClock()
	var sink io.Writer = f
	if options.writer != nil {
		sink = options.writer(sink)
	}
	writer := &observerWriter{sink: sink, max: options.maxBytes}
	encoder := json.NewEncoder(writer)
	defer func() {
		end := observerClock()
		pausesAfter := observerPauses()
		close(heartStop)
		beats := <-heartDone
		resultErr = errors.Join(resultErr, f.Close(), external.stop())
		if beats.Error != "" {
			resultErr = errors.Join(resultErr, errors.New(beats.Error))
		}
		message := ""
		if resultErr != nil {
			message = resultErr.Error()
		}
		resultErr = errors.Join(resultErr, observerSave(path+".phases.json", map[string]interface{}{"schema": "buildopt.live-observer/phases/v1", "pid": os.Getpid(), "begin": begin, "end": end, "rows": rows, "heartbeat": beats, "runtimePausesBefore": pausesBefore, "runtimePausesAfter": pausesAfter, "bytes": writer.total, "error": message, "maxRows": options.maxRows, "maxBytes": options.maxBytes}))
	}()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		// A cancellation already observed must not consume an arbitrary ready tick.
		select {
		case <-stop:
			return nil
		default:
		}
		select {
		case <-stop:
			return nil
		case tick := <-ticker.C:
			if len(rows) == options.maxRows {
				return errors.New("sampler record limit")
			}
			row := observerPhase{Index: len(rows), Wake: observerClock(), WakeLateNS: time.Since(tick).Nanoseconds()}
			row.ScanBegin = observerClock()
			record, err := collectOwnedSample(worker, observe)
			row.RecordReady = observerClock()
			if err == nil {
				writer.row = &row
				row.EncodeBegin = observerClock()
				err = encoder.Encode(record)
				row.EncodeEnd = observerClock()
			}
			if err != nil {
				row.Error = err.Error()
			}
			rows = append(rows, row)
			if err != nil {
				return err
			}
			if options.stopAfter > 0 && len(rows) == options.stopAfter {
				return nil
			}
		}
	}
}

type deliberateWriter struct {
	sink   io.Writer
	mode   string
	writes int
	stop   chan struct{}
}

func (w *deliberateWriter) Write(data []byte) (int, error) {
	w.writes++
	if w.mode == "writer-failure" && w.writes == 3 {
		return 0, errors.New("deliberate writer failure")
	}
	if w.mode == "writer-delay" && w.writes == 3 {
		time.Sleep(2104 * time.Millisecond)
	}
	if w.mode == "cancel-write" && w.writes == 1 {
		close(w.stop)
		time.Sleep(150 * time.Millisecond)
	}
	return w.sink.Write(data)
}

func TestLiveObserverCase(t *testing.T) {
	name := os.Getenv("BUILDOPT_LIVE_CASE")
	root := os.Getenv("BUILDOPT_LIVE_CASE_ROOT")
	if !filepath.IsAbs(root) {
		t.Fatal("explicit fixture root required")
	}
	self, err := processIdentity(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if err = observerSave(filepath.Join(root, "fixture-identity.json"), self); err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	options := samplerOptions{maxRows: 10000, maxBytes: 512 << 20, stopAfter: 40}
	observe := cgroupProcesses
	expected := ""
	switch name {
	case "normal", "process-pause":
	case "writer-delay", "writer-failure", "cancel-write":
		options.writer = func(sink io.Writer) io.Writer { return &deliberateWriter{sink: sink, mode: name, stop: stop} }
		if name == "writer-failure" {
			expected = "deliberate writer failure"
		}
	case "cancel-before":
		close(stop)
	case "scan-failure":
		expected = "deliberate scan failure"
		observe = func(string) ([]ProcessIdentity, error) { return nil, errors.New(expected) }
	case "process-exit":
		calls := 0
		options.stopAfter = 3
		observe = func(group string) ([]ProcessIdentity, error) {
			calls++
			if calls == 1 {
				return nil, &os.PathError{Op: "open", Path: "/proc/999999999/stat", Err: unix.ENOENT}
			}
			return cgroupProcesses(group)
		}
	case "record-limit":
		options.maxRows = 2
		options.stopAfter = 0
		expected = "sampler record limit"
	default:
		t.Fatal("unregistered case", name)
	}
	err = observeSampler(workerSession{Cgroup: self.Cgroup}, filepath.Join(root, "proc-samples.jsonl"), stop, observe, options)
	if (expected == "" && err != nil) || (expected != "" && (err == nil || err.Error() != expected)) {
		t.Fatalf("expected %q, got %v", expected, err)
	}
}

func TestLiveObserverOwner(t *testing.T) {
	var m Manifest
	if err := readJSON(os.Getenv("BUILDOPT_DIAGNOSTIC_MANIFEST"), &m); err != nil {
		t.Fatal(err)
	}
	if m.Subject != "elasticsearch-cpu-isolation-control" || m.Phase != "QUALIFICATION" || len(m.History) != 1 || m.ExecutionEnd != 0 || m.Limits.MaxWorkflowStarts != 2 || m.Limits.MaxGradleStarts != 2 || m.Limits.MaxRequestNS != int64(900*time.Second) || len(m.Candidate.Files) != 0 {
		t.Fatal("two-start diagnostic scope differs")
	}
	if err := checkBinding(m.Overhead); err != nil {
		t.Fatal(err)
	}
	fmt.Println("LIVE_OBSERVER owner diagnostic; two cold builds only; no timing qualification")
	runDaemonDiagnostic(t, m, true)
}
