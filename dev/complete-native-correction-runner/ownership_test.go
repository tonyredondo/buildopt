package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "guard-child" {
		if err := run(os.Args[1:]); err != nil {
			os.Stderr.WriteString(err.Error() + "\n")
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// This is a separate, explicit host integration gate. Ordinary hosted CI runs
// generic fixtures; it must not pretend to provide user-systemd delegation.
func TestSystemdOwnership(t *testing.T) {
	if os.Getenv("CNC_SYSTEMD_TESTS") != "1" {
		t.Skip("requires explicit transient user-unit integration gate")
	}
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"cancellation", "expiry", "owner-loss", "native"} {
		t.Run(mode, func(t *testing.T) {
			base := t.TempDir()
			packagePath := filepath.Join(base, "package.json")
			manifest, err := makePackage(repo)
			if err != nil {
				t.Fatal(err)
			}
			if err = writeNewJSON(packagePath, manifest); err != nil {
				t.Fatal(err)
			}
			digest, err := fileDigest(packagePath)
			if err != nil {
				t.Fatal(err)
			}
			now, err := bootClock()
			if err != nil {
				t.Fatal(err)
			}
			if mode == "expiry" {
				now.Seconds -= 7175
			}
			root := filepath.Join(base, "state")
			if err = initialize(root, digest, now); err != nil {
				t.Fatal(err)
			}
			var state campaign
			if err = readJSON(filepath.Join(root, "state.json"), &state); err != nil {
				t.Fatal(err)
			}
			unit := ownershipUnit(root, state)
			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				// The exact test-owned transient unit; no global stop/reset operation.
				_ = exec.CommandContext(ctx, "systemctl", "--user", "stop", unit).Run()
			})
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()
			if err = startGuardian(ctx, repo, root, packagePath, state); err != nil {
				log, _ := os.ReadFile(filepath.Join(root, "guard.log"))
				t.Fatalf("guardian: %v\n%s", err, log)
			}
			var owner ownership
			if err = readJSON(filepath.Join(root, "ownership.json"), &owner); err != nil {
				t.Fatal(err)
			}
			owned, err := openOwnership(ctx, root, state, owner)
			if err != nil {
				t.Fatal(err)
			}
			defer owned.Close()
			// A detached daemon analogue survives its launcher and a second request.
			idle := exec.Command("/bin/sleep", "30")
			applyOwnership(idle, owned)
			idle.SysProcAttr.Setpgid = false
			idle.SysProcAttr.Setsid = true
			if err = idle.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = idle.Process.Kill(); _ = idle.Wait() })
			probe := exec.Command("/bin/true")
			applyOwnership(probe, owned)
			if err = probe.Run(); err != nil {
				t.Fatal(err)
			}
			if err = idle.Process.Signal(syscall.Signal(0)); err != nil {
				t.Fatal("native idle daemon lost between requests")
			}
			unrelated := exec.Command("/bin/sleep", "30")
			if err = unrelated.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = unrelated.Process.Kill(); _ = unrelated.Wait() })
			if mode == "cancellation" {
				r := childRequest(t, root, "P01", "wait")
				r.Ownership = &owner
				r.TimeoutSeconds = 2
				got, err := capture(ctx, root, state, r, bootClock, nil)
				if err != nil || got.Outcome != "TIME_LIMIT" || !got.Started {
					t.Fatalf("%+v %v", got, err)
				}
			}
			if mode == "native" {
				testNativeConsumerMatrix(t, repo, root, state, owner)
				if err := killWorkers(owner.Workers); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "owner-loss" {
				binary, err := os.Executable()
				if err != nil {
					t.Fatal(err)
				}
				client := exec.Command(binary, "-test.run=^TestOwnershipClient$")
				client.Env = append(os.Environ(), "CNC_CLIENT_ROOT="+root)
				if err = client.Start(); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = client.Process.Kill() })
				for {
					if _, err := os.Stat(filepath.Join(root, "client-child-started")); err == nil {
						break
					}
					select {
					case <-ctx.Done():
						t.Fatal("client child never started")
					case <-time.After(10 * time.Millisecond):
					}
				}
				if err = client.Process.Kill(); err != nil {
					t.Fatal(err)
				}
				_ = client.Wait()
			}
			for {
				data, readErr := os.ReadFile(filepath.Join(owner.Workers, "cgroup.events"))
				if os.IsNotExist(readErr) || (readErr == nil && strings.Contains(string(data), "populated 0")) {
					break
				}
				select {
				case <-ctx.Done():
					t.Fatal("owned descendants survived stop/expiry")
				case <-time.After(20 * time.Millisecond):
				}
			}
			if err = unrelated.Process.Signal(syscall.Signal(0)); err != nil {
				t.Fatal("unrelated process touched")
			}
			if mode == "expiry" {
				if _, err = openOwnership(ctx, root, state, owner); err == nil {
					t.Fatal("expired guardian accepted")
				}
			}
			if mode == "owner-loss" {
				if _, err := inspect(root, state); err == nil {
					t.Fatal("lost owner accepted as completed")
				}
				r := childRequest(t, root, "P01", "success")
				r.Ownership = &owner
				if _, err := capture(ctx, root, state, r, bootClock, nil); err == nil {
					t.Fatal("lost attempt replayed")
				}
			}
		})
	}
}

func TestOwnershipClient(t *testing.T) {
	root := os.Getenv("CNC_CLIENT_ROOT")
	if root == "" {
		return
	}
	var state campaign
	if err := readJSON(filepath.Join(root, "state.json"), &state); err != nil {
		t.Fatal(err)
	}
	var owner ownership
	if err := readJSON(filepath.Join(root, "ownership.json"), &owner); err != nil {
		t.Fatal(err)
	}
	r := childRequest(t, root, "P01", "wait")
	r.Ownership = &owner
	r.TimeoutSeconds = 10
	r.Environment = append(r.Environment, "CNC_TEST_HANDSHAKE="+filepath.Join(root, "client-child-started"))
	if _, err := capture(context.Background(), root, state, r, bootClock, nil); err != nil {
		t.Fatal(err)
	}
}
