//go:build linux && amd64

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type observerReport struct {
	Schema                 string          `json:"schema"`
	WriteBoundary          string          `json:"writeBoundary"`
	PID                    int             `json:"pid"`
	Begin                  observerStamp   `json:"begin"`
	End                    observerStamp   `json:"end"`
	CloseEnd               observerStamp   `json:"closeEnd"`
	Rows                   []observerPhase `json:"rows"`
	Heartbeat              observerBeats   `json:"heartbeat"`
	RuntimePausesBefore    map[string]any  `json:"runtimePausesBefore"`
	RuntimePausesAfter     map[string]any  `json:"runtimePausesAfter"`
	Bytes                  int64           `json:"bytes"`
	Error                  string          `json:"error"`
	MaxRows                int             `json:"maxRows"`
	MaxBytes               int64           `json:"maxBytes"`
	Persistence            bufferedSummary `json:"persistence"`
	ExternalHeartbeatCPUNS int64           `json:"externalHeartbeatCPUNS"`
}

type observedProcess struct {
	Identity         ProcessIdentity   `json:"identity"`
	ThreadAffinities map[string]string `json:"threadAffinities"`
	AffinityError    string            `json:"affinityError"`
	Stat             string            `json:"stat"`
}

func checkObservation(m Manifest, dir string, native ProcessReceipt) error {
	policy, err := readObserverPolicy(m)
	if err != nil {
		return err
	}
	var receipt ObservationReceipt
	if err = readJSON(filepath.Join(dir, "observation.json"), &receipt); err != nil {
		return fmt.Errorf("missing/invalid observation receipt: %w", err)
	}
	if receipt.Schema != "buildopt.history-replay/observation/v1" || receipt.Policy != m.Observer || receipt.Error != "" || receipt.ExitCode != 0 || receipt.SamplerCPUNS < 0 || receipt.WriterCPUNS < 0 || receipt.HeartbeatCPUNS < 0 {
		return errors.New("failed or unbound observation")
	}
	if err = checkStampInterval(receipt.Start, receipt.End); err != nil {
		return err
	}
	if receipt.Start.Boot != native.Start.Boot || receipt.Start.NS > native.Start.NS || receipt.End.NS < native.End.NS {
		return errors.New("observer lifetime does not cover native request")
	}
	if err = checkBinding(receipt.Launch); err != nil {
		return err
	}
	var launch ObserverLaunch
	if err = readJSON(receipt.Launch.Path, &launch); err != nil {
		return err
	}
	path := filepath.Join(dir, "proc-samples.jsonl")
	if launch.Policy != m.Observer || launch.Path != path || launch.Cgroup != native.Supervisor.Cgroup || launch.NativeAffinity != m.Affinity || receipt.Launch.Path != filepath.Join(dir, "observer-launch.json") {
		return errors.New("observer launch belongs to another request/policy")
	}
	paths, err := filepath.Glob(path + "*")
	if err != nil {
		return err
	}
	if len(paths) != len(receipt.Artifacts) {
		return errors.New("observer artifact coverage differs")
	}
	for i, b := range receipt.Artifacts {
		if b.Path != paths[i] {
			return errors.New("observer artifact order/root differs")
		}
		if err = checkBinding(b); err != nil {
			return err
		}
	}
	var report observerReport
	if err = readJSON(path+".phases.json", &report); err != nil {
		return err
	}
	s := report.Persistence
	if report.Schema != "buildopt.buffered-observer/phases/v1" || report.WriteBoundary != "queue-admission" || report.PID != receipt.Sampler.PID || report.Error != "" || s.Error != "" || s.WriterExit != 0 || !equalJSON(s.Config, policy.writer()) || report.MaxRows != policy.MaxRows || report.MaxBytes != policy.TotalBytes {
		return errors.New("observation phase/configuration differs or failed")
	}
	if receipt.WriterCPUNS != s.WriterCPUNS || receipt.HeartbeatCPUNS != report.ExternalHeartbeatCPUNS || receipt.SamplerCPUNS < report.CloseEnd.CPUNS {
		return errors.New("observer CPU cost omitted or altered")
	}
	if report.Begin.BootNS < native.Start.NS && report.End.BootNS >= native.End.NS {
		// The entire capture brackets the native request, including both endpoints.
	} else {
		return errors.New("sample capture does not bracket native execution")
	}
	if report.Begin.BootNS > report.End.BootNS || report.End.BootNS > s.DrainBegin.BootNS || s.DrainBegin.BootNS > s.DrainEnd.BootNS || s.DrainEnd.BootNS > report.CloseEnd.BootNS || report.CloseEnd.BootNS > receipt.End.NS {
		return errors.New("observer capture/drain chronology differs")
	}
	if s.Accepted != len(report.Rows) || s.Acknowledged != s.Accepted || len(s.Writes) != s.Accepted || s.PendingRecords != 0 || s.PendingBytes != 0 || s.PeakPendingRecords < 0 || s.PeakPendingRecords > policy.PendingRecords || s.PeakPendingBytes < 0 || s.PeakPendingBytes > policy.PendingBytes || s.AcceptedBytes > policy.TotalBytes || s.AcceptedBytes != s.PersistedBytes || report.Bytes != s.AcceptedBytes {
		return errors.New("unacknowledged or over-limit observation payload")
	}
	if len(report.Rows) > policy.MaxRows {
		return errors.New("observation row limit")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if int64(len(raw)) != s.PersistedBytes {
		return errors.New("observer persisted byte count differs")
	}
	nativeMask, err := parseCPUSet(m.Affinity)
	if err != nil {
		return err
	}
	supervisorMask, err := parseCPUSet(policy.SamplerAffinity)
	if err != nil {
		return err
	}
	for _, value := range []string{native.ResourcesBefore.CPUAffinity, native.ResourcesAfter.CPUAffinity} {
		actual, err := parseCPUSet(value)
		if err != nil || actual != supervisorMask {
			return errors.New("supervisor/native isolation or restoration differs")
		}
	}
	points := []int64{native.Start.NS}
	cursor := 0
	lastWake := int64(0)
	for i, row := range report.Rows {
		ack := s.Writes[i]
		if row.Index != i || ack.Index != i || row.Error != "" || ack.Error != "" || row.WriteCalls != 1 || row.Bytes <= 0 || row.Bytes > policy.FrameBytes || ack.Bytes != row.Bytes || ack.PayloadSHA256 != row.PayloadSHA256 || ack.WrittenSHA256 != row.PayloadSHA256 || cursor+row.Bytes > len(raw) {
			return errors.New("observer order/hash/count differs")
		}
		data := raw[cursor : cursor+row.Bytes]
		cursor += row.Bytes
		if payloadDigest(data) != row.PayloadSHA256 || !bytes.HasSuffix(data, []byte{'\n'}) {
			return errors.New("corrupt/partial observer payload")
		}
		stamps := []observerStamp{row.Wake, row.ScanBegin, row.RecordReady, row.EncodeBegin, row.WriteBegin, row.WriteEnd, row.EncodeEnd}
		for j, at := range stamps {
			if at.BootNS <= 0 || at.MonoNS <= 0 || at.CPUNS < 0 {
				return errors.New("invalid observer clock")
			}
			if j > 0 && (at.BootNS < stamps[j-1].BootNS || at.MonoNS < stamps[j-1].MonoNS || at.CPUNS < stamps[j-1].CPUNS) {
				return errors.New("observer phase clock reversed")
			}
		}
		if row.Wake.BootNS <= lastWake || row.Wake.BootNS < report.Begin.BootNS || row.EncodeEnd.BootNS > report.End.BootNS || ack.Begin.BootNS < row.EncodeBegin.BootNS || ack.End.BootNS < ack.Begin.BootNS || ack.End.BootNS > s.DrainEnd.BootNS {
			return errors.New("observer sample/write chronology differs")
		}
		lastWake = row.Wake.BootNS
		var sample struct {
			Begin     Stamp             `json:"begin"`
			End       Stamp             `json:"end"`
			Processes []observedProcess `json:"processes"`
			Gap       string            `json:"gap"`
			Cgroup    map[string]string `json:"cgroup"`
		}
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		if _, err = parseJSON(decoder); err != nil {
			return err
		}
		if err = json.Unmarshal(data, &sample); err != nil {
			return err
		}
		if sample.Begin.NS < row.ScanBegin.BootNS || sample.End.NS > row.RecordReady.BootNS || sample.End.NS < sample.Begin.NS || sample.Begin.Boot != native.Start.Boot {
			return errors.New("sample clock differs from collector phase")
		}
		if sample.Gap != "" {
			if sample.Gap != "PROCESS_EXITED_DURING_SNAPSHOT" {
				return errors.New("unrecognized process observation gap")
			}
			continue
		}
		if len(sample.Cgroup) != 4 || len(sample.Processes) == 0 {
			return errors.New("missing owned process/cgroup sample")
		}
		for _, key := range []string{"cpu.stat", "cpu.pressure", "io.pressure", "memory.pressure"} {
			if _, ok := sample.Cgroup[key]; !ok {
				return errors.New("missing cgroup metric")
			}
		}
		for _, process := range sample.Processes {
			if process.Identity.Cgroup != launch.Cgroup && !strings.HasPrefix(process.Identity.Cgroup, launch.Cgroup+"/") {
				return errors.New("sample outside owned cgroup")
			}
			if process.AffinityError != "" || len(process.ThreadAffinities) == 0 || process.Stat == "" {
				continue
			} // A vanishing process cannot supply coverage.
			want := nativeMask
			if process.Identity.PID == native.Supervisor.PID && process.Identity.StartTicks == native.Supervisor.StartTicks {
				want, err = parseCPUSet(policy.SamplerAffinity)
				if err != nil {
					return err
				}
			}
			for _, value := range process.ThreadAffinities {
				mask, err := parseCPUSet(value)
				if err != nil || mask != want {
					return errors.New("native thread CPU mask differs")
				}
			}
			covers := process.Identity.PID == native.Process.PID && process.Identity.StartTicks == native.Process.StartTicks
			if m.Driver == "GRADLE" {
				covers = false
				for _, arg := range process.Identity.Command {
					if arg == "org.gradle.launcher.daemon.bootstrap.GradleDaemon" {
						covers = true
					}
				}
			}
			if covers && sample.Begin.NS >= native.Start.NS && sample.Begin.NS <= native.End.NS {
				points = append(points, sample.Begin.NS)
			}
		}
	}
	if cursor != len(raw) {
		return errors.New("unacknowledged trailing payload")
	}
	if m.Driver == "GRADLE" && len(points) == 1 {
		return errors.New("missing Gradle daemon snapshots; CLI coverage cannot substitute")
	}
	points = append(points, native.End.NS)
	sort.Slice(points, func(i, j int) bool { return points[i] < points[j] })
	for i := 1; i < len(points); i++ {
		if points[i]-points[i-1] > policy.MaxGapNS {
			return fmt.Errorf("observation gap exceeds 500 ms including native endpoints: %d ns", points[i]-points[i-1])
		}
	}
	if report.Heartbeat.Error != "" {
		return errors.New("internal heartbeat failed")
	}
	var external struct {
		Schema           string            `json:"schema"`
		Identity         ProcessIdentity   `json:"identity"`
		ThreadAffinities map[string]string `json:"threadAffinities"`
		Begin            observerStamp     `json:"begin"`
		End              observerStamp     `json:"end"`
		Beats            observerBeats     `json:"beats"`
	}
	if err = readJSON(path+".external.json", &external); err != nil {
		return err
	}
	if external.Beats.Error != "" || external.Schema != "buildopt.live-observer/heartbeat/v1" {
		return errors.New("external heartbeat failed")
	}
	for _, beats := range []observerBeats{report.Heartbeat, external.Beats} {
		previous := report.Begin.BootNS
		for _, beat := range beats.Rows {
			if beat.At.BootNS < report.Begin.BootNS || beat.At.BootNS > report.End.BootNS {
				continue
			}
			if beat.At.BootNS < previous || beat.At.BootNS-previous > policy.MaxGapNS {
				return errors.New("heartbeat observation gap")
			}
			previous = beat.At.BootNS
		}
		if report.End.BootNS-previous > policy.MaxGapNS {
			return errors.New("heartbeat endpoint gap")
		}
	}
	var sampler struct {
		Identity         ProcessIdentity   `json:"identity"`
		ThreadAffinities map[string]string `json:"threadAffinities"`
	}
	if err = readJSON(path+".sampler.json", &sampler); err != nil {
		return err
	}
	if !equalJSON(sampler.Identity, receipt.Sampler) {
		return errors.New("sampler process identity differs")
	}
	for _, row := range []struct {
		masks    map[string]string
		expected string
	}{{sampler.ThreadAffinities, policy.SamplerAffinity}, {external.ThreadAffinities, policy.HeartbeatAffinity}} {
		want, err := parseCPUSet(row.expected)
		if err != nil {
			return err
		}
		if len(row.masks) == 0 {
			return errors.New("missing helper affinity")
		}
		for _, value := range row.masks {
			actual, err := parseCPUSet(value)
			if err != nil || actual != want {
				return errors.New("helper affinity differs")
			}
		}
	}
	for _, identity := range []ProcessIdentity{receipt.Sampler, s.Writer, external.Identity} {
		if identity.PID <= 0 || identity.StartTicks == 0 || sameProcess(identity) {
			return errors.New("missing helper identity or helper remains alive")
		}
	}
	return nil
}
