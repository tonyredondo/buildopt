package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestNativeServiceHelper(t *testing.T) {
	for i, arg := range os.Args {
		if arg == "--eic-service-helper" {
			exe, err := os.Executable()
			if err != nil {
				os.Exit(65)
			}
			child := exec.Command(exe, "-test.run=^TestFixtureChildProcess$", "--", "--eic-child", "sleep")
			child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
			if err = child.Start(); err != nil {
				os.Exit(65)
			}
			if err = writeNew(os.Args[i+1], []byte(strconv.Itoa(child.Process.Pid))); err != nil {
				child.Process.Kill()
				os.Exit(65)
			}
			_ = child.Wait()
			os.Exit(0)
		}
	}
}

func TestNativeServiceCancellationKillsEscapedDescendant(t *testing.T) {
	check, checkCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer checkCancel()
	if err := exec.CommandContext(check, "/usr/bin/systemctl", "--user", "show", "--property=Version").Run(); err != nil {
		if os.Getenv("EIC_REQUIRE_SYSTEMD") == "1" {
			t.Fatal("required user systemd unavailable", err)
		}
		t.Skip("supplemental Linux user-systemd proof requires EIC_REQUIRE_SYSTEMD=1")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	pidPath := filepath.Join(dir, "pid")
	unit := "buildopt-eic-test-" + digest([]byte(dir))[:20] + ".service"
	args := nativeServiceArguments(unit, exe, "unused", "unused")
	args = append(args[:len(args)-4], "-test.run=^TestNativeServiceHelper$", "--", "--eic-service-helper", pidPath)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := runNativeService(ctx, args, unit); done <- err }()
	deadline := time.Now().Add(5 * time.Second)
	pid := 0
	for time.Now().Before(deadline) {
		b, e := os.ReadFile(pidPath)
		if e == nil {
			pid, _ = strconv.Atoi(string(b))
			if pid > 0 {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if pid <= 0 {
		cancel()
		t.Fatal("owned service fixture did not start")
	}
	t.Cleanup(func() { _ = exec.Command("/usr/bin/systemctl", "--user", "stop", unit).Run() })
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("service cancellation did not finish")
	}
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		b, e := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
		if os.IsNotExist(e) || (e == nil && strings.Contains(string(b), ") Z ")) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("escaped descendant survived its owned service cancellation")
}

func TestNativePreparationCommandsAndIsolation(t *testing.T) {
	root := t.TempDir()
	for i, arm := range []string{"N0", "N1", "W1"} {
		r, err := makeNativeRequest(root, "/runner", makeProtocol().Rows[i])
		if err != nil {
			t.Fatal(err)
		}
		want := append([]string{"/usr/bin/taskset", "--cpu-list", "0-7"}, makeProtocol().Rows[i].Arguments...)
		if !reflect.DeepEqual(r.Arguments, want) || r.Directory != filepath.Join(root, "arms", arm) || r.TimeoutSeconds != 600 {
			t.Fatalf("%+v", r)
		}
		for _, item := range r.Environment {
			if strings.HasPrefix(item, "CI=") || strings.HasPrefix(item, "GRADLE_OPTS=") || strings.HasPrefix(item, "HOME=") {
				t.Fatalf("inherited/repurposed environment: %s", item)
			}
		}
		if !containsNativeEnv(r.Environment, "GRADLE_USER_HOME="+filepath.Join(root, "state", arm, "gradle")) {
			t.Fatal("shared Gradle home")
		}
	}
	if _, err := makeNativeRequest(root, "/runner", makeProtocol().Rows[5]); err == nil {
		t.Fatal("candidate before native preparation gates")
	}
}

func containsNativeEnv(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestNativeServiceHasHardDeadlineAndOwnsDescendants(t *testing.T) {
	args := nativeServiceArguments("buildopt-eic-test.service", "/runner", "/request", "sha")
	for _, want := range []string{"--user", "--wait", "--pipe", "--collect", "--property=KillMode=control-group", "--property=RuntimeMaxSec=610s", "--property=TimeoutStopSec=2s", "--property=Restart=no"} {
		if !containsNativeEnv(args, want) {
			t.Fatalf("missing %s", want)
		}
	}
	if !reflect.DeepEqual(args[len(args)-5:], []string{"/runner", "native-child", "/request", "sha", "buildopt-eic-test.service"}) {
		t.Fatalf("%v", args)
	}
}

func TestNativePreparationCannotSkipOrRepeatSlots(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "attempts"), 0700)
	if err := validateNativeSequence(root, "P001"); err != nil {
		t.Fatal(err)
	}
	if err := validateNativeSequence(root, "P002"); err == nil {
		t.Fatal("missing predecessor accepted")
	}
	dir := filepath.Join(root, "attempts", "P001")
	os.Mkdir(dir, 0700)
	writeJSON(filepath.Join(dir, "native-result.json"), nativeResult{RowID: "P001", Process: processRecord{Started: true, ExitCode: 1, Outcome: "FAILURE"}})
	if err := validateNativeSequence(root, "P002"); err == nil {
		t.Fatal("failed preparation bypassed")
	}
	if err := validateNativeSequence(root, "P001"); err == nil {
		t.Fatal("attempt overwritten")
	}
}
