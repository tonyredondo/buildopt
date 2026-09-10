package main

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
)

type correctnessRuntime struct {
	Root         string       `json:"root"`
	Runner       fileBinding  `json:"runner"`
	Package      fileBinding  `json:"package"`
	BackendURL   string       `json:"backendUrl"`
	CA           fileBinding  `json:"ca"`
	Credential   fileBinding  `json:"credential"`
	Installed    []entry      `json:"installedFiles"`
	CommonGit    string       `json:"commonGit"`
	BootID       string       `json:"bootId"`
	StartNS      int64        `json:"startBootNanoseconds"`
	DeadlineNS   int64        `json:"deadlineBootNanoseconds"`
	EvidenceRoot string       `json:"evidenceRoot,omitempty"`
	Metadata     *fileBinding `json:"metadataRuntime,omitempty"`
	metadata     *metadataSession
	Provenance   *fileBinding `json:"provenanceRuntime,omitempty"`
	provenance   *provenanceSession
}

// A JVM property survives the installed launcher's private-environment scrub.
// Keep the previously frozen native collector unchanged for receipt replay.
var correctnessDiagnosticScript = strings.Replace(nativeDiagnosticScript, "System.getenv('BUILDOPT_TASK_GRAPH_OUTPUT')", "System.getProperty('buildopt.eic.taskGraphOutput')", 1)

func correctnessRequestFor(f correctnessRuntime, step correctnessStep) (nativeRequest, error) {
	matched := false
	for _, expected := range makeCorrectnessPlan().Steps {
		if reflect.DeepEqual(step, expected) && expected.Row.Block != "M" {
			matched = true
			break
		}
	}
	if !matched {
		return nativeRequest{}, errors.New("only exact C/T requests use the native campaign adapter")
	}
	var preparation row
	for _, r := range makeProtocol().Rows[:3] {
		if r.Arm == step.Row.Arm {
			preparation = r
		}
	}
	r, err := makeNativeRequest(f.Root, f.Runner.Path, preparation)
	if err != nil {
		return r, err
	}
	r.Runner = f.Runner
	r.Row = step.Row
	evidenceRoot := f.Root
	if f.EvidenceRoot != "" {
		evidenceRoot = f.EvidenceRoot
	}
	attempt := filepath.Join(evidenceRoot, "attempts", r.Row.ID)
	r.Diagnostic = &fileBinding{filepath.Join(f.Root, "runtime", "correctness-diagnostic.init.gradle"), digest([]byte(correctnessDiagnosticScript))}
	r.Arguments = append([]string{"/usr/bin/taskset", "--cpu-list", "0-7"}, step.Row.Arguments...)
	r.Arguments = append(r.Arguments, "--init-script", r.Diagnostic.Path, "-Dbuildopt.eic.taskGraphOutput="+filepath.Join(attempt, "task-graph.jsonl"), "-Dorg.gradle.internal.operations.trace="+filepath.Join(attempt, "operations"))
	if r.Row.Arm == "W1" {
		state := filepath.Join(f.Root, "state", "W1")
		workflow, _ := json.Marshal(r.Arguments[4:])
		r.Environment = append(r.Environment,
			"BUILDOPT_EIC_NATIVE_TRACE="+filepath.Join(attempt, "native-supervision.json"),
			"BUILDOPT_TEAM_TOKEN=@"+f.Credential.Path,
			"BUILDOPT_WRAPPER_CACHE_HOME="+filepath.Join(state, "wrapper"),
			"BUILDOPT_STICKY_OBSERVATION=wcncp", "BUILDOPT_STICKY_CA_FILE="+f.CA.Path, "CURL_CA_BUNDLE="+f.CA.Path,
			"WCNCP_OUTBOX_DIR="+filepath.Join(state, "outbox"), "WCNCP_ENVIRONMENT_CLASS=LOCAL_FUNCTIONAL",
			"WCNCP_REPOSITORY_REVISION="+r.Row.Revision, "WCNCP_PACKAGE_SHA256="+f.Package.SHA256,
			"WCNCP_GRADLE_VERSION=9.7.1", "WCNCP_WORKFLOW_SHA256="+digest(workflow))
	}
	return r, nil
}
