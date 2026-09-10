package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTestKitCountCLIReconstructsPinnedLogs(t *testing.T) {
	root := t.TempDir()
	log := filepath.Join(root, "daemon-123.out.log")
	raw := []byte(daemonFixture("24932983-0128-44b8-a167-83431604c33c"))
	if err := writeNew(log, raw); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(root, "logs.json")
	binding := fileBinding{log, digest(raw)}
	if err := writeJSON(manifest, testKitLogManifest{"buildopt.eic/testkit-logs/v1", []fileBinding{binding}}); err != nil {
		t.Fatal(err)
	}
	pin, err := hashFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err = runCLI([]string{"testkit-count", "--input", manifest, "--sha256", pin}, &out); err != nil {
		t.Fatal(err)
	}
	var result testKitCount
	if err = json.Unmarshal(out.Bytes(), &result); err != nil || result.Starts != 1 || result.PerformanceAuthority {
		t.Fatalf("%+v %v", result, err)
	}
	if err = os.WriteFile(log, append(raw, ' '), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = countTestKitBuilds(fileBinding{manifest, pin}); err == nil {
		t.Fatal("changed log accepted")
	}
	if err = os.WriteFile(log, raw, 0600); err != nil {
		t.Fatal(err)
	}
	duplicate := filepath.Join(root, "duplicate.json")
	if err = writeJSON(duplicate, testKitLogManifest{"buildopt.eic/testkit-logs/v1", []fileBinding{binding, binding}}); err != nil {
		t.Fatal(err)
	}
	pin, err = hashFile(duplicate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = countTestKitBuilds(fileBinding{duplicate, pin}); err == nil {
		t.Fatal("duplicate log accepted")
	}
}

func daemonFixture(id string) string {
	prefix := "2026-09-07T14:47:02.133+0000 [DEBUG] [org.gradle.launcher.daemon.server."
	build := "Build{id=" + id + ", currentDir=/fixture}"
	return prefix + "DefaultIncomingConnectionHandler] Received command: " + build + ".\n" +
		prefix + "DefaultIncomingConnectionHandler] Starting executing command: " + build + " with connection: socket connection.\n" +
		prefix + "exec.ExecuteBuild] The daemon has started executing the build.\n" +
		prefix + "exec.ExecuteBuild] Executing build with daemon context: DefaultDaemonContext[uid=11111111-1111-1111-1111-111111111111,javaHome=/jdk,javaVersion=21,javaVendor=Eclipse Adoptium,daemonRegistryDir=/fixture,pid=123,idleTimeout=120000]\n" +
		prefix + "exec.ExecuteBuild] The daemon has finished executing the build.\n" +
		prefix + "DefaultIncomingConnectionHandler] Finishing executing command: " + build + "\n"
}

func TestTestKitDaemonCountsRequestsInReusedDaemon(t *testing.T) {
	raw := daemonFixture("24932983-0128-44b8-a167-83431604c33c") + daemonFixture("4ce3f78c-78a4-433e-a2f5-ec0a98916309")
	path := filepath.Join(t.TempDir(), "daemon-123.out.log")
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	builds, err := readDaemonBuilds(fileBinding{path, digest([]byte(raw))})
	if err != nil || len(builds) != 2 || builds[0].PID != builds[1].PID || builds[0].ID == builds[1].ID {
		t.Fatalf("%+v %v", builds, err)
	}
	for _, bad := range []string{
		strings.Replace(raw, "The daemon has finished executing the build.", "missing finish", 1),
		strings.Replace(raw, "pid=123", "pid=999", 1),
		strings.Replace(raw, "Starting executing command:", "missing start:", 1),
		strings.ReplaceAll(raw, "4ce3f78c-78a4-433e-a2f5-ec0a98916309", "24932983-0128-44b8-a167-83431604c33c"),
		raw[:strings.LastIndex(raw, "Finishing executing command:")],
	} {
		if err := os.WriteFile(path, []byte(bad), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := readDaemonBuilds(fileBinding{path, digest([]byte(bad))}); err == nil {
			t.Fatal("incomplete or ambiguous daemon evidence accepted")
		}
	}
}
