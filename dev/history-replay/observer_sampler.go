//go:build linux && amd64

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"runtime/metrics"
	"strconv"
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
	maxRows           int
	maxBytes          int64
	stopAfter         int // Fixed fixture completion only; zero for actual owner.
	writer            func(io.Writer) io.Writer
	buffered          *bufferedConfig
	heartbeatAffinity string
	ready             func() error
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
	cpuNS   int64
}

func startExternalObserver(path, affinity string) (*externalObserver, error) {
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
	cmd := exec.Command("/usr/bin/taskset", "-c", affinity, executable, "observer-heartbeat", path, affinity)
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
	return &externalObserver{cmd: cmd, control: control, log: log}, nil
}
func (h *externalObserver) stop() error {
	closeErr := h.control.Close()
	done := make(chan error, 1)
	go func() {
		err := h.cmd.Wait()
		if h.cmd.ProcessState != nil {
			h.cpuNS = (h.cmd.ProcessState.UserTime() + h.cmd.ProcessState.SystemTime()).Nanoseconds()
		}
		done <- err
	}()
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

func serveObserverHeartbeat(path, requestedAffinity string) error {
	if path == "" {
		return errors.New("explicit heartbeat output required")
	}
	identity, err := processIdentity(os.Getpid())
	if err != nil {
		return err
	}
	mask, err := threadAffinities(os.Getpid())
	if err != nil {
		return err
	}
	// taskset restricts the whole Go runtime before it starts.
	expected, err := parseCPUSet(requestedAffinity)
	if err != nil {
		return err
	}
	var affinity unix.CPUSet
	if err = unix.SchedGetaffinity(0, &affinity); err != nil || affinity != expected {
		return errors.Join(errors.New("heartbeat CPU differs"), err)
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
		return errors.Join(err, errors.New(beats.Error))
	}
	return nil
}

// All test and owner cases consume the same live collector, encoder and file.
// Diagnostic records are buffered with hard limits, then saved after sampling.
func observeSampler(worker workerSession, path string, stop <-chan struct{}, observe func(string) ([]ProcessIdentity, error), options samplerOptions) (resultErr error) {
	var f *os.File
	var queued *bufferedWriter
	var sink io.Writer
	var err error
	if options.buffered != nil {
		queued, err = startBufferedWriter(path, *options.buffered)
		sink = queued
	} else {
		f, err = os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		sink = f
	}
	if err != nil {
		return err
	}
	external, err := startExternalObserver(path+".external.json", options.heartbeatAffinity)
	if err != nil {
		if queued != nil {
			_, closeErr := queued.Close()
			return errors.Join(err, closeErr)
		}
		return errors.Join(err, f.Close())
	}
	heartStop := make(chan struct{})
	heartDone := make(chan observerBeats, 1)
	go func() { heartDone <- observeBeats(heartStop) }()
	rows := make([]observerPhase, 0, options.maxRows)
	pausesBefore := observerPauses()
	begin := observerClock()
	if options.writer != nil {
		sink = options.writer(sink)
	}
	writer := &observerWriter{sink: sink, max: options.maxBytes}
	encoder := json.NewEncoder(writer)
	defer func() {
		end := observerClock()
		pausesAfter := observerPauses()
		var persistence *bufferedSummary
		if queued != nil {
			summary, closeErr := queued.Close()
			persistence = &summary
			resultErr = errors.Join(resultErr, closeErr)
		} else {
			resultErr = errors.Join(resultErr, f.Close())
		}
		close(heartStop)
		beats := <-heartDone
		resultErr = errors.Join(resultErr, external.stop())
		if beats.Error != "" {
			resultErr = errors.Join(resultErr, errors.New(beats.Error))
		}
		message := ""
		if resultErr != nil {
			message = resultErr.Error()
		}
		schema, boundary := "buildopt.live-observer/phases/v1", "synchronous-file-write"
		if queued != nil {
			schema, boundary = "buildopt.buffered-observer/phases/v1", "queue-admission"
		}
		resultErr = errors.Join(resultErr, observerSave(path+".phases.json", map[string]interface{}{
			"schema": schema, "writeBoundary": boundary, "pid": os.Getpid(), "begin": begin, "end": end, "closeEnd": observerClock(),
			"rows": rows, "heartbeat": beats, "runtimePausesBefore": pausesBefore, "runtimePausesAfter": pausesAfter,
			"bytes": writer.total, "error": message, "maxRows": options.maxRows, "maxBytes": options.maxBytes, "persistence": persistence, "externalHeartbeatCPUNS": external.cpuNS,
		}))
	}()
	if options.ready != nil {
		if err := options.ready(); err != nil {
			return err
		}
	}
	var failed <-chan struct{}
	if queued != nil {
		failed = queued.failed
	}

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
		case <-failed:
			return queued.Error()
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
