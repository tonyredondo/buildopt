//go:build linux

package launcher

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestStickyWCNCPNativeTracePreservesNativeFailureAndScrubsItsControl(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "native.json")
	t.Setenv("BUILDOPT_EIC_NATIVE_TRACE", path)
	var out, stderr bytes.Buffer
	code := runStickyWCNCP(root, []string{"/bin/sh", "-c", `test -z "$BUILDOPT_EIC_NATIVE_TRACE" || exit 99; printf '%s' "$$"; exit 7`}, nil, &out, &stderr)
	if code != 7 || stderr.Len() != 0 {
		t.Fatalf("native behavior changed: %d %s", code, stderr.String())
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var trace struct {
		Schema     string `json:"schemaVersion"`
		PID        int    `json:"pid"`
		Started    bool   `json:"started"`
		Start      int64  `json:"startBootNanoseconds"`
		End        int64  `json:"endBootNanoseconds"`
		Duration   int64  `json:"nativeDurationNanoseconds"`
		Exit       int    `json:"exitCode"`
		Signal     int    `json:"signal"`
		Cgroup     string `json:"cgroup"`
		Diagnostic bool   `json:"diagnosticOnly"`
	}
	if err = json.Unmarshal(raw, &trace); err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(out.String())
	if err != nil {
		t.Fatal(err)
	}
	if trace.Schema != "buildopt.eic/native-supervision/v1" || !trace.Started || trace.PID != pid || trace.Start <= 0 || trace.End <= trace.Start || trace.Duration <= 0 || trace.Duration > trace.End-trace.Start || trace.Exit != 7 || trace.Signal != 0 || !trace.Diagnostic || !strings.HasPrefix(trace.Cgroup, "0::/") {
		t.Fatalf("invalid native supervision trace: %+v", trace)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatal("trace is not private")
	}
}

func TestStickyWCNCPNativeTraceFailureNeverReplacesNativeResult(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "existing.json")
	if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BUILDOPT_EIC_NATIVE_TRACE", path)
	var out, stderr bytes.Buffer
	if code := runStickyWCNCP(root, []string{"/bin/sh", "-c", "exit 3"}, nil, &out, &stderr); code != 3 {
		t.Fatal("trace failure replaced native result", code)
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != "keep" || out.Len() != 0 || stderr.Len() != 0 {
		t.Fatal("trace overwrote prior data or changed streams")
	}
}

func TestStickyWCNCPNativeTraceRetainsNativeSignal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "signal.json")
	t.Setenv("BUILDOPT_EIC_NATIVE_TRACE", path)
	var out, stderr bytes.Buffer
	if code := runStickyWCNCP(filepath.Dir(path), []string{"/bin/sh", "-c", "kill -TERM $$"}, nil, &out, &stderr); code != 143 {
		t.Fatal("native signal changed", code)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var trace struct {
		Signal int `json:"signal"`
		Exit   int `json:"exitCode"`
	}
	if err = json.Unmarshal(raw, &trace); err != nil {
		t.Fatal(err)
	}
	if trace.Signal != 15 || trace.Exit != -1 {
		t.Fatalf("lost signal: %+v", trace)
	}
}
