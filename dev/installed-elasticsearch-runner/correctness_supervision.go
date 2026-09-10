package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type correctnessSupervision struct {
	Schema     string `json:"schemaVersion"`
	Started    bool   `json:"started"`
	PID        int    `json:"pid"`
	Start      int64  `json:"startBootNanoseconds"`
	End        int64  `json:"endBootNanoseconds"`
	Duration   int64  `json:"nativeDurationNanoseconds"`
	Exit       int    `json:"exitCode"`
	Signal     int    `json:"signal"`
	Cgroup     string `json:"cgroup"`
	Diagnostic bool   `json:"diagnosticOnly"`
}

func verifyCorrectnessSupervision(dir string, p processRecord) error {
	var v correctnessSupervision
	if err := readJSON(filepath.Join(dir, "native-supervision.json"), &v); err != nil {
		return err
	}
	if v.Schema != "buildopt.eic/native-supervision/v1" || !v.Started || v.PID <= 0 || !v.Diagnostic || v.Start < p.StartNS || v.End > p.EndNS || v.End <= v.Start || v.Duration <= 0 || v.Duration > v.End-v.Start || v.Signal != p.Signal || v.Exit != p.ExitCode || v.Cgroup == "" {
		return errors.New("installed native supervision does not fit the owned outer result")
	}
	return nil
}

// Requests retain only a hash-bound reference. The token exists solely in the
// installed child environment and is scrubbed again before native Gradle.
func correctnessPrivateEnvironment(f correctnessRuntime, frozen []string) ([]string, error) {
	env := append([]string(nil), frozen...)
	for i, value := range env {
		if !strings.HasPrefix(value, "BUILDOPT_TEAM_TOKEN=") {
			continue
		}
		if value != "BUILDOPT_TEAM_TOKEN=@"+f.Credential.Path {
			return nil, errors.New("credential reference drift")
		}
		if err := checkBinding(f.Credential); err != nil {
			return nil, err
		}
		info, err := os.Lstat(f.Credential.Path)
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() > 16<<10 {
			return nil, errors.New("private bounded credential file required")
		}
		raw, err := os.ReadFile(f.Credential.Path)
		if err != nil {
			return nil, err
		}
		env[i] = "BUILDOPT_TEAM_TOKEN=" + string(raw)
	}
	return env, nil
}
