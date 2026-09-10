//go:build linux && amd64

package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type bufferedConfig struct {
	PendingRecords int    `json:"pendingRecords"`
	PendingBytes   int64  `json:"pendingBytes"`
	FrameBytes     int    `json:"frameBytes"`
	TotalBytes     int64  `json:"totalBytes"`
	DrainNS        int64  `json:"drainNS"`
	Fault          string `json:"fault"` // Controlled fixture fault, empty in owner dispatch.
}

func defaultBufferedConfig() bufferedConfig {
	return bufferedConfig{128, 16 << 20, 512 << 10, 512 << 20, int64(5 * time.Second), ""}
}
func (c bufferedConfig) validate() error {
	if c.PendingRecords < 1 || c.PendingRecords > 128 || c.PendingBytes < 1 || c.PendingBytes > 16<<20 || c.FrameBytes < 1 || c.FrameBytes > 512<<10 || c.TotalBytes < 1 || c.TotalBytes > 512<<20 || c.DrainNS <= 0 || c.DrainNS > int64(5*time.Second) {
		return errors.New("invalid buffered writer bounds")
	}
	switch c.Fault {
	case "", "delay", "error", "short", "stall", "exit":
		return nil
	default:
		return errors.New("unknown writer fixture fault")
	}
}

type writeAck struct {
	Index         int           `json:"index"`
	Bytes         int           `json:"bytes"`
	PayloadSHA256 string        `json:"payloadSHA256"`
	WrittenSHA256 string        `json:"writtenSHA256"`
	Begin         observerStamp `json:"begin"`
	End           observerStamp `json:"end"`
	Error         string        `json:"error"`
}
type bufferedSummary struct {
	Config             bufferedConfig  `json:"config"`
	Accepted           int             `json:"accepted"`
	Acknowledged       int             `json:"acknowledged"`
	AcceptedBytes      int64           `json:"acceptedBytes"`
	PersistedBytes     int64           `json:"persistedBytes"`
	PendingRecords     int             `json:"pendingRecords"`
	PendingBytes       int64           `json:"pendingBytes"`
	PeakPendingRecords int             `json:"peakPendingRecords"`
	PeakPendingBytes   int64           `json:"peakPendingBytes"`
	Writes             []writeAck      `json:"writes"`
	Writer             ProcessIdentity `json:"writer"`
	WriterExit         int             `json:"writerExit"`
	WriterCPUNS        int64           `json:"writerCPUNS"`
	Error              string          `json:"error"`
	DrainBegin         observerStamp   `json:"drainBegin"`
	DrainEnd           observerStamp   `json:"drainEnd"`
}
type payloadFrame struct {
	index  int
	data   []byte
	digest string
}

// Exactly one pump owns transport I/O; admission never waits for that I/O.
// A record remains pending until its successful, exact acknowledgement.
type bufferedWriter struct {
	mu                         sync.Mutex
	summary                    bufferedSummary
	err                        error
	accepting                  bool
	queue                      chan payloadFrame
	failed                     chan struct{}
	pumped                     chan struct{}
	cmd                        *exec.Cmd
	input, acks, lifetime, log *os.File
}

func payloadDigest(data []byte) string { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }
func startBufferedWriter(path string, config bufferedConfig) (*bufferedWriter, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	if err = observerSave(path+".writer-config.json", config); err != nil {
		return nil, err
	}
	files := []*os.File{}
	pipe := func() (*os.File, *os.File, error) {
		r, w, e := os.Pipe()
		if e == nil {
			files = append(files, r, w)
		}
		return r, w, e
	}
	inRead, inWrite, err := pipe()
	if err != nil {
		return nil, err
	}
	success := false
	defer func() {
		if !success {
			for _, f := range files {
				f.Close()
			}
		}
	}()
	ackRead, ackWrite, err := pipe()
	if err != nil {
		return nil, err
	}
	lifeRead, lifeWrite, err := pipe()
	if err != nil {
		return nil, err
	}
	readyRead, readyWrite, err := pipe()
	if err != nil {
		return nil, err
	}
	log, err := os.OpenFile(path+".writer.log", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	files = append(files, log)
	cmd := exec.Command(executable, "observer-writer", path)
	cmd.ExtraFiles = []*os.File{inRead, ackWrite, lifeRead, readyWrite}
	cmd.Stdout = log
	cmd.Stderr = log
	if err = cmd.Start(); err != nil {
		return nil, err
	}
	inRead.Close()
	ackWrite.Close()
	lifeRead.Close()
	readyWrite.Close()
	if err = readyRead.SetReadDeadline(time.Now().Add(5 * time.Second)); err == nil {
		var b [1]byte
		_, err = io.ReadFull(readyRead, b[:])
		if err == nil && b[0] != 1 {
			err = errors.New("invalid writer ready receipt")
		}
	}
	readyRead.Close()
	if err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		return nil, fmt.Errorf("writer startup: %w", err)
	}
	var identity ProcessIdentity
	if err = readJSON(path+".writer-identity.json", &identity); err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		return nil, err
	}
	if identity.PID != cmd.Process.Pid || !sameProcess(identity) {
		cmd.Process.Kill()
		cmd.Wait()
		return nil, errors.New("writer identity mismatch")
	}
	w := &bufferedWriter{summary: bufferedSummary{Config: config, Writes: make([]writeAck, 0, 10000), Writer: identity}, accepting: true, queue: make(chan payloadFrame, config.PendingRecords), failed: make(chan struct{}), pumped: make(chan struct{}), cmd: cmd, input: inWrite, acks: ackRead, lifetime: lifeWrite, log: log}
	success = true
	go w.pump()
	return w, nil
}
func (w *bufferedWriter) failure(err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.err == nil {
		w.err = err
		close(w.failed)
	}
}
func (w *bufferedWriter) Error() error { w.mu.Lock(); defer w.mu.Unlock(); return w.err }
func (w *bufferedWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	s := &w.summary
	c := s.Config
	if w.err != nil {
		return 0, w.err
	}
	if !w.accepting {
		return 0, errors.New("writer admission closed")
	}
	if len(data) > c.FrameBytes {
		return 0, errors.New("writer frame byte limit")
	}
	if int64(len(data)) > c.TotalBytes-s.AcceptedBytes {
		return 0, errors.New("writer total byte limit")
	}
	if s.PendingRecords == c.PendingRecords {
		return 0, errors.New("writer pending record limit")
	}
	if int64(len(data)) > c.PendingBytes-s.PendingBytes {
		return 0, errors.New("writer pending byte limit")
	}
	frame := payloadFrame{s.Accepted, append([]byte(nil), data...), payloadDigest(data)}
	// PendingRecords includes the in-flight frame, so a queue slot must exist.
	select {
	case w.queue <- frame:
	default:
		return 0, errors.New("writer queue invariant violated")
	}
	s.Accepted++
	s.AcceptedBytes += int64(len(data))
	s.PendingRecords++
	s.PendingBytes += int64(len(data))
	if s.PendingRecords > s.PeakPendingRecords {
		s.PeakPendingRecords = s.PendingRecords
	}
	if s.PendingBytes > s.PeakPendingBytes {
		s.PeakPendingBytes = s.PendingBytes
	}
	return len(data), nil
}
func (w *bufferedWriter) pump() {
	defer close(w.pumped)
	decoder := json.NewDecoder(w.acks)
	for frame := range w.queue {
		var header [4]byte
		binary.BigEndian.PutUint32(header[:], uint32(len(frame.data)))
		if _, err := w.input.Write(header[:]); err != nil {
			w.failure(fmt.Errorf("writer transport: %w", err))
			return
		}
		if n, err := w.input.Write(frame.data); err != nil || n != len(frame.data) {
			w.failure(fmt.Errorf("writer transport: %v (%d/%d)", err, n, len(frame.data)))
			return
		}
		var ack writeAck
		if err := decoder.Decode(&ack); err != nil {
			w.failure(fmt.Errorf("writer acknowledgement: %w", err))
			return
		}
		if ack.Index != frame.index || ack.PayloadSHA256 != frame.digest || ack.Bytes < 0 || ack.Bytes > len(frame.data) || ack.End.MonoNS < ack.Begin.MonoNS || ack.WrittenSHA256 != payloadDigest(frame.data[:ack.Bytes]) {
			w.failure(errors.New("writer acknowledgement mismatch"))
			return
		}
		w.mu.Lock()
		w.summary.Writes = append(w.summary.Writes, ack)
		w.summary.PersistedBytes += int64(ack.Bytes)
		if ack.Error == "" && ack.Bytes == len(frame.data) {
			w.summary.Acknowledged++
			w.summary.PendingRecords--
			w.summary.PendingBytes -= int64(len(frame.data))
		}
		w.mu.Unlock()
		if ack.Error != "" {
			w.failure(errors.New(ack.Error))
			return
		}
		if ack.Bytes != len(frame.data) {
			w.failure(io.ErrShortWrite)
			return
		}
	}
	if err := w.input.Close(); err != nil {
		w.failure(err)
	}
}

func (w *bufferedWriter) Close() (bufferedSummary, error) {
	w.mu.Lock()
	w.accepting = false
	w.summary.DrainBegin = observerClock()
	close(w.queue)
	w.mu.Unlock()
	timer := time.NewTimer(time.Duration(w.summary.Config.DrainNS))
	defer timer.Stop()
	timedOut := false
	select {
	case <-w.pumped:
	case <-timer.C:
		w.failure(errors.New("writer drain deadline"))
		timedOut = true
	}
	if w.Error() != nil {
		// Closing pollable pipes also releases a pump blocked in transport I/O.
		w.cmd.Process.Kill()
		w.input.Close()
		w.acks.Close()
	}
	<-w.pumped
	wait := make(chan error, 1)
	go func() { wait <- w.cmd.Wait() }()
	var waitErr error
	// Remaining lifetime shares the original drain deadline.
	if timedOut {
		waitErr = <-wait
	} else {
		select {
		case waitErr = <-wait:
		case <-timer.C:
			w.failure(errors.New("writer exit deadline"))
			w.cmd.Process.Kill()
			waitErr = <-wait
		}
	}
	w.input.Close()
	w.acks.Close()
	w.lifetime.Close()
	logErr := w.log.Close()
	if waitErr != nil && w.Error() == nil {
		w.failure(fmt.Errorf("writer exit: %w", waitErr))
	}
	if logErr != nil {
		w.failure(logErr)
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cmd.ProcessState != nil {
		w.summary.WriterExit = w.cmd.ProcessState.ExitCode()
		w.summary.WriterCPUNS = (w.cmd.ProcessState.UserTime() + w.cmd.ProcessState.SystemTime()).Nanoseconds()
	}
	if w.err == nil && (w.summary.Accepted != w.summary.Acknowledged || w.summary.PendingRecords != 0 || w.summary.PendingBytes != 0) {
		w.err = errors.New("writer closed with unacknowledged data")
	}
	if w.err != nil {
		w.summary.Error = w.err.Error()
	}
	w.summary.DrainEnd = observerClock()
	// Release owned payload buffers after the pump has stopped; failure counts remain.
	for range w.queue {
	}
	return w.summary, w.err
}

func serveBufferedWriter(path string) error {
	if !filepath.IsAbs(path) {
		return errors.New("explicit writer output required")
	}
	var config bufferedConfig
	if err := readJSON(path+".writer-config.json", &config); err != nil {
		return err
	}
	if err := config.validate(); err != nil {
		return err
	}
	input := os.NewFile(3, "payloads")
	acks := os.NewFile(4, "acknowledgements")
	life := os.NewFile(5, "parent lifetime")
	ready := os.NewFile(6, "ready")
	// Independent of a slow file write. EOF is a failed parent lifetime, not drain.
	go func() { io.Copy(io.Discard, life); fmt.Fprintln(os.Stderr, "WRITER_PARENT_EOF"); os.Exit(88) }()
	identity, err := processIdentity(os.Getpid())
	if err != nil {
		return err
	}
	if err = observerSave(path+".writer-identity.json", identity); err != nil {
		return err
	}
	output, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(acks)
	if _, err = ready.Write([]byte{1}); err != nil {
		return err
	}
	ready.Close()
	begin := observerClock()
	count := 0
	total := int64(0)
	message := ""
	for {
		var header [4]byte
		_, err = io.ReadFull(input, header[:])
		if err == io.EOF {
			break
		}
		if err != nil {
			message = err.Error()
			break
		}
		length := int(binary.BigEndian.Uint32(header[:]))
		if length < 1 || length > config.FrameBytes || int64(length) > config.TotalBytes-total || count == 10000 {
			message = "writer protocol bound"
			break
		}
		data := make([]byte, length)
		if _, err = io.ReadFull(input, data); err != nil {
			message = err.Error()
			break
		}
		ack := writeAck{Index: count, PayloadSHA256: payloadDigest(data), Begin: observerClock()}
		if count == 2 {
			switch config.Fault {
			case "delay":
				time.Sleep(2104 * time.Millisecond)
			case "stall":
				if err = observerSave(path+".stall-ready.json", map[string]interface{}{"identity": identity, "at": observerClock()}); err != nil {
					return err
				}
				time.Sleep(20 * time.Second)
			case "exit":
				os.Exit(89)
			}
		}
		if count == 2 && config.Fault == "error" {
			err = errors.New("deliberate writer error")
		} else if count == 2 && config.Fault == "short" {
			ack.Bytes, err = output.Write(data[:len(data)/2])
			if err == nil {
				err = io.ErrShortWrite
			}
		} else {
			ack.Bytes, err = output.Write(data)
		}
		ack.End = observerClock()
		ack.WrittenSHA256 = payloadDigest(data[:ack.Bytes])
		total += int64(ack.Bytes)
		if err != nil {
			ack.Error = err.Error()
		}
		if e := encoder.Encode(ack); e != nil {
			message = e.Error()
			break
		}
		count++
		if err != nil {
			message = err.Error()
			break
		}
	}
	closeErr := output.Close()
	if closeErr != nil {
		message = closeErr.Error()
	}
	end := observerClock()
	if err = observerSave(path+".writer-result.json", map[string]interface{}{"begin": begin, "end": end, "recordsAttempted": count, "bytes": total, "error": message}); err != nil {
		return err
	}
	if message != "" {
		return errors.New(message)
	}
	return nil
}
