package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCorrectnessRequestsKeepNativeCommandsAndSurvivePrivateEnvironmentScrubbing(t *testing.T) {
	root := t.TempDir()
	f := correctnessRuntime{Root: root, Runner: fileBinding{"/runner", strings.Repeat("a", 64)}, Package: fileBinding{"/package", strings.Repeat("b", 64)}, BackendURL: "https://127.0.0.1:32123", CA: fileBinding{filepath.Join(root, "backend/ca.pem"), strings.Repeat("c", 64)}, Credential: fileBinding{filepath.Join(root, "backend/credential.json"), strings.Repeat("d", 64)}}
	for _, id := range []string{"C001", "C005", "C009", "T001", "T003"} {
		step := correctnessStepForTest(t, id)
		r, err := correctnessRequestFor(f, step)
		if err != nil {
			t.Fatal(err)
		}
		executable := "./gradlew"
		if id == "C005" {
			executable = "./buildoptw"
		}
		if len(r.Arguments) < 8 || r.Arguments[3] != executable || r.Arguments[0] != "/usr/bin/taskset" || r.Arguments[2] != "0-7" {
			t.Fatalf("entrypoint/affinity drift: %v", r.Arguments)
		}
		foundGraph := false
		for _, arg := range r.Arguments {
			if arg == "-Dbuildopt.eic.taskGraphOutput="+filepath.Join(root, "attempts", id, "task-graph.jsonl") {
				foundGraph = true
			}
			if arg == "--rerun-tasks" || arg == "--no-configuration-cache" || arg == "--scan" {
				t.Fatal("native mode changed", arg)
			}
		}
		if !foundGraph {
			t.Fatal("graph still depends on scrubbed environment")
		}
		for _, env := range r.Environment {
			if strings.HasPrefix(env, "BUILDOPT_TASK_GRAPH_OUTPUT=") || strings.HasPrefix(env, "CI=") {
				t.Fatal("invalid forwarded environment", env)
			}
		}
		if id == "C005" {
			if !containsNativeEnv(r.Environment, "BUILDOPT_EIC_NATIVE_TRACE="+filepath.Join(root, "attempts", id, "native-supervision.json")) {
				t.Fatal("installed native boundary missing")
			}
			if !containsNativeEnv(r.Environment, "BUILDOPT_TEAM_TOKEN=@"+f.Credential.Path) {
				t.Fatal("credential must remain a private file reference in the request")
			}
		} else {
			for _, env := range r.Environment {
				if strings.HasPrefix(env, "BUILDOPT_TEAM_TOKEN=") {
					t.Fatal("native control received installed credentials")
				}
			}
		}
	}
	if _, err := correctnessRequestFor(f, correctnessStepForTest(t, "M001")); err == nil {
		t.Fatal("standalone mutation bypassed its qualified adapter")
	}
}
