//go:build !windows

package launcher

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStickyConnectionCredentialNeverReachesGradle(t *testing.T) {
	root := writeStickyConnectionRepository(
		t,
		"https://buildopt.example.com",
		"example/private-project",
		"BUILDOPT_TEAM_TOKEN",
	)
	configPath := filepath.Join(root, ".buildopt", "config.toml")
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), `mode = "auto"`, `mode = "off"`, 1))
	if err := os.WriteFile(configPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	gradle := filepath.Join(root, "gradlew")
	if err := os.WriteFile(
		gradle,
		[]byte("#!/bin/sh\nprintf 'token=<%s> root=<%s> ordinary=<%s>\\n' \"${BUILDOPT_TEAM_TOKEN-}\" \"${BUILDOPT_STICKY_WRAPPER_ROOT-}\" \"${STICKY_ORDINARY-}\"\n"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}
	t.Setenv(stickyWrapperRootEnvironment, root)
	t.Setenv("BUILDOPT_TEAM_TOKEN", "must-not-reach-gradle")
	t.Setenv("STICKY_ORDINARY", "preserved")
	t.Setenv(bypassEnvironment, "1")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"run", "--", gradle}, nil, &stdout, &stderr); code != 0 {
		t.Fatalf("run exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stdout.String() != "token=<> root=<> ordinary=<preserved>\n" || stderr.Len() != 0 {
		t.Fatalf("child output stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestStickyConnectionCredentialOverrideCannotReintroduceSecret(t *testing.T) {
	environment := replaceEnvironmentWithReserved(
		[]string{"BUILDOPT_TEAM_TOKEN=parent", "STICKY_ORDINARY=preserved"},
		map[string]string{"BUILDOPT_TEAM_TOKEN": "override"},
		[]string{"BUILDOPT_TEAM_TOKEN"},
	)
	if environmentValue(environment, "BUILDOPT_TEAM_TOKEN") != "" {
		t.Fatalf("credential override reached child environment: %v", environment)
	}
	if environmentValue(environment, "STICKY_ORDINARY") != "preserved" {
		t.Fatalf("ordinary environment was not preserved: %v", environment)
	}
}
