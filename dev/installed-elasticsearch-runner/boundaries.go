package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/wcncpobserve"
)

type boundaryProcess struct {
	StartNS  int64  `json:"startBootNanoseconds"`
	EndNS    int64  `json:"endBootNanoseconds"`
	PID      int    `json:"pid"`
	ExitCode int    `json:"exitCode"`
	Cgroup   string `json:"cgroup"`
}

type boundaryObservation struct {
	AtNS  int64                           `json:"receivedBootNanoseconds"`
	Facts []wcncpobserve.ObservationFacts `json:"facts"`
}

func validateInstalledBoundary(outer, child boundaryProcess, observation *boundaryObservation, expectedExit int) error {
	if outer.StartNS <= 0 || outer.StartNS > child.StartNS || child.StartNS >= child.EndNS || child.EndNS >= outer.EndNS || outer.ExitCode != expectedExit || child.ExitCode != expectedExit || child.PID <= 0 || outer.PID <= 0 || child.Cgroup != outer.Cgroup || !strings.HasPrefix(outer.Cgroup, "0::/") || !strings.HasSuffix(outer.Cgroup, ".service") {
		return errors.New("invalid installed/native process boundaries")
	}
	if observation == nil {
		return nil
	}
	if len(observation.Facts) != 1 {
		return errors.New("one current observation required")
	}
	f := observation.Facts[0]
	if err := f.Validate(); err != nil {
		return err
	}
	if f.Duration.ValueMs == nil || *f.Duration.ValueMs <= 0 || *f.Duration.ValueMs > 120000 || f.Duration.Classification != "CONTROLLED_VALUE_INPUT" || f.Completeness != "COMPLETE" {
		return errors.New("missing measured recorder duration")
	}
	duration := *f.Duration.ValueMs * int64(time.Millisecond)
	if duration+int64(time.Millisecond) < child.EndNS-child.StartNS || duration > outer.EndNS-outer.StartNS || observation.AtNS < child.EndNS || observation.AtNS > outer.EndNS {
		return errors.New("recorder duration or upload outside actual process boundaries")
	}
	want := "SUCCESS"
	if expectedExit != 0 {
		want = "FAILED"
	}
	if f.Child.Outcome != want || f.Child.ExitCode == nil || *f.Child.ExitCode != expectedExit {
		return errors.New("native outcome lost by recorder")
	}
	return nil
}

type boundaryReceipt struct {
	Schema string `json:"schemaVersion"`
	Class  string `json:"environmentClass"`
	Files  map[string]struct {
		SHA256 string `json:"sha256"`
		Size   int64  `json:"size"`
		Mode   uint32 `json:"mode"`
	} `json:"files"`
}

type boundarySummary struct {
	Class                string `json:"environmentClass"`
	Starts               int    `json:"gradleStarts"`
	Observations         int    `json:"observations"`
	Failures             int    `json:"expectedFailures"`
	Verified             bool   `json:"verified"`
	PerformanceAuthority bool   `json:"performanceAuthority"`
}

func checkInstalledBoundaries(binding fileBinding) (boundarySummary, error) {
	out := boundarySummary{Class: "LOCAL_INSTALLED_BOUNDARY_QUALIFICATION"}
	if err := checkBinding(binding); err != nil {
		return out, err
	}
	var receipt boundaryReceipt
	if err := readJSON(binding.Path, &receipt); err != nil {
		return out, err
	}
	if receipt.Schema != "buildopt.eic/installed-boundary-receipt/v1" || receipt.Class != out.Class || len(receipt.Files) < 20 || len(receipt.Files) > 100 {
		return out, errors.New("invalid boundary receipt")
	}
	root := filepath.Dir(binding.Path)
	for name, expected := range receipt.Files {
		if !filepath.IsLocal(name) || filepath.Clean(name) != name {
			return out, errors.New("unsafe boundary artifact")
		}
		path := filepath.Join(root, name)
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Size() != expected.Size || uint32(info.Mode().Perm()) != expected.Mode {
			return out, errors.New("boundary artifact type/size/mode drift")
		}
		if err = checkBinding(fileBinding{path, expected.SHA256}); err != nil {
			return out, err
		}
	}
	for _, required := range []string{"freeze.json", "verified.json"} {
		if _, ok := receipt.Files[required]; !ok {
			return out, errors.New("missing boundary completion artifact")
		}
	}
	var freeze map[string]json.RawMessage
	if err := readJSON(filepath.Join(root, "freeze.json"), &freeze); err != nil {
		return out, err
	}
	var class, packageSHA, helperSHA string
	var starts, limit int
	var performance bool
	for key, target := range map[string]any{"environmentClass": &class, "maximumGradleStarts": &starts, "maximumServiceSeconds": &limit, "packageSha256": &packageSHA, "helperSha256": &helperSHA, "performanceAuthority": &performance} {
		if json.Unmarshal(freeze[key], target) != nil {
			return out, errors.New("missing frozen boundary input")
		}
	}
	if class != out.Class || starts != 4 || limit != 120 || performance || packageSHA != "3eb84a331caf2b78346ff1001053f23b5243b05943c298e5380a569a893ae456" {
		return out, errors.New("boundary allocation or package drift")
	}
	dirs, err := os.ReadDir(filepath.Join(root, "rows"))
	if err != nil || len(dirs) != 4 {
		return out, errors.New("missing or extra boundary rows")
	}
	last := int64(0)
	seenObservations := map[string]bool{}
	for ordinal := 1; ordinal <= 4; ordinal++ {
		rowName := fmt.Sprintf("QW%03d", ordinal)
		dir := filepath.Join(root, "rows", rowName)
		entries, err := os.ReadDir(dir)
		if err != nil {
			return out, err
		}
		for _, entry := range entries {
			if _, ok := receipt.Files[filepath.Join("rows", rowName, entry.Name())]; !ok || entry.IsDir() {
				return out, errors.New("unbound file in boundary row")
			}
		}
		for _, name := range []string{"outer.json", "child.json", "service-request.json", "service.log", "verified.json"} {
			if _, ok := receipt.Files[filepath.Join("rows", rowName, name)]; !ok {
				return out, errors.New("missing raw boundary artifact")
			}
		}
		var outer, child boundaryProcess
		if err := readJSON(filepath.Join(dir, "outer.json"), &outer); err != nil {
			return out, err
		}
		if err := readJSON(filepath.Join(dir, "child.json"), &child); err != nil {
			return out, err
		}
		if outer.StartNS < last || outer.EndNS-outer.StartNS > int64(120*time.Second) {
			return out, errors.New("overlapping or excessive boundary duration")
		}
		last = outer.EndNS
		var service []string
		if err := readJSON(filepath.Join(dir, "service-request.json"), &service); err != nil {
			return out, err
		}
		i := -1
		for index, value := range service {
			if value == "--" {
				i = index
				break
			}
		}
		if i < 0 || len(service) < i+11 || service[i+1] != "/usr/bin/taskset" || service[i+2] != "--cpu-list" || service[i+3] != "0-3" || service[i+5] != "-test.run=^TestStickyWCNCPBoundaryHelper$" || service[i+7] != "--eic-owned-boundary" || service[i+8] != filepath.Join(dir, "outer.json") {
			return out, errors.New("installed command provenance drift")
		}
		if err := checkBinding(fileBinding{service[i+4], helperSHA}); err != nil {
			return out, err
		}
		unit := filepath.Base(strings.TrimPrefix(outer.Cgroup, "0::"))
		for _, required := range []string{"--unit=" + unit, "--property=KillMode=control-group", "--property=RuntimeMaxSec=120s", "--property=Restart=no"} {
			if !slices.Contains(service, required) {
				return out, errors.New("lost service ownership/bound")
			}
		}
		wantExit := 0
		if ordinal == 4 {
			wantExit = 1
			out.Failures++
		}
		var observation *boundaryObservation
		if ordinal > 1 {
			if _, ok := receipt.Files[filepath.Join("rows", rowName, "observation.json")]; !ok {
				return out, errors.New("unbound recorder evidence")
			}
			observation = &boundaryObservation{}
			if err := readJSON(filepath.Join(dir, "observation.json"), observation); err != nil {
				return out, err
			}
		}
		if err := validateInstalledBoundary(outer, child, observation, wantExit); err != nil {
			return out, err
		}
		if observation != nil {
			fact := observation.Facts[0]
			if seenObservations[fact.ObservationID] {
				return out, errors.New("duplicate recorder observation")
			}
			seenObservations[fact.ObservationID] = true
			arguments, _ := json.Marshal(service[i+10:])
			if fact.Bindings.BuildOptPackageSHA256 != packageSHA || fact.Bindings.WorkflowSHA256 != digest(arguments) {
				return out, errors.New("observation does not bind actual invocation")
			}
			out.Observations++
		}
		out.Starts++
	}
	out.Verified = true
	return out, nil
}
