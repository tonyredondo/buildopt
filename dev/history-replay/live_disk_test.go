//go:build linux && amd64

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestLiveDiskDoesNotEnumerateUnchangedRetainedDirectory(t *testing.T) {
	root := t.TempDir()
	retained := filepath.Join(root, "retained")
	if err := os.Mkdir(retained, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(retained, "result"), []byte("result"), 0600); err != nil {
		t.Fatal(err)
	}
	guard := newLiveDiskGuard(root, Limits{MaxBytes: 100})
	defer guard.Close()
	if err := guard.Check(nil); err != nil {
		t.Fatal(err)
	}
	// Observe actual directory enumeration, rather than assert a timing on a
	// shared host. Attach after the initial complete observation.
	fd, err := unix.InotifyInit1(unix.IN_NONBLOCK | unix.IN_CLOEXEC)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fd)
	if _, err = unix.InotifyAddWatch(fd, retained, unix.IN_OPEN|unix.IN_ONLYDIR); err != nil {
		t.Fatal(err)
	}
	if err = guard.Check(nil); err != nil {
		t.Fatal(err)
	}
	var events [4096]byte
	n, err := unix.Read(fd, events[:])
	if !errors.Is(err, unix.EAGAIN) || n > 0 {
		t.Fatalf("unchanged retained directory was enumerated again: bytes=%d error=%v", n, err)
	}
}

func liveDiskFixture(t *testing.T) (string, *liveDiskGuard) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "owned")
	for _, name := range []string{"a", "b"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "a", "file"), []byte("five!"), 0600); err != nil {
		t.Fatal(err)
	}
	g := newLiveDiskGuard(root, Limits{MaxBytes: 1 << 40})
	t.Cleanup(func() {
		if err := g.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := g.Check(nil); err != nil {
		t.Fatal(err)
	}
	if g.fallback != "" {
		t.Fatalf("fixture cannot exercise notifications: %s", g.fallback)
	}
	return root, g
}

func assertLiveDiskTotal(t *testing.T, g *liveDiskGuard, fallback bool) {
	t.Helper()
	want, err := treeBytes(g.root)
	if err != nil {
		t.Fatal(err)
	}
	g.limits.MaxBytes = want
	if err = g.Check(nil); err != nil {
		t.Fatalf("exact ceiling %d: %v", want, err)
	}
	if (g.fallback != "") != fallback {
		t.Fatalf("fallback=%q, want fallback=%v", g.fallback, fallback)
	}
	if !fallback && g.total != want {
		t.Fatalf("cached bytes=%d, full scan=%d", g.total, want)
	}
	if want > 0 {
		g.limits.MaxBytes = want - 1
		if err = g.Check(nil); err == nil || !strings.Contains(err.Error(), "artifact size ceiling") {
			t.Fatalf("under-sized ceiling accepted: %v", err)
		}
	}
}

func TestLiveDiskMutationAccounting(t *testing.T) {
	for _, kind := range []string{"grow", "shrink", "create", "delete", "rename", "cross-directory-rename", "atomic-replace", "temporary", "temporary-directory", "symlink", "symlink-replacement", "internal-hardlink", "external-hardlink", "sparse", "fallocate"} {
		t.Run(kind, func(t *testing.T) {
			root, g := liveDiskFixture(t)
			file := filepath.Join(root, "a", "file")
			check := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			switch kind {
			case "grow":
				check(os.Truncate(file, 100))
			case "shrink":
				check(os.Truncate(file, 1))
			case "create":
				check(os.WriteFile(filepath.Join(root, "b", "new"), []byte("created"), 0600))
			case "delete":
				check(os.Remove(file))
			case "rename":
				check(os.Rename(file, filepath.Join(root, "a", "renamed")))
			case "cross-directory-rename":
				check(os.Rename(file, filepath.Join(root, "b", "renamed")))
			case "atomic-replace":
				tmp := filepath.Join(root, "a", "temporary")
				check(os.WriteFile(tmp, []byte("replacement bytes"), 0600))
				check(os.Rename(tmp, file))
			case "temporary":
				tmp := filepath.Join(root, "a", "temporary")
				check(os.WriteFile(tmp, []byte("temporary bytes"), 0600))
				assertLiveDiskTotal(t, g, false)
				check(os.Remove(tmp))
			case "temporary-directory":
				tmp := filepath.Join(root, "b", "tmp", "nested")
				check(os.MkdirAll(tmp, 0700))
				check(os.WriteFile(filepath.Join(tmp, "file"), []byte("temporary bytes"), 0600))
				assertLiveDiskTotal(t, g, false)
				check(os.RemoveAll(filepath.Dir(tmp)))
			case "symlink":
				check(os.Symlink(filepath.Dir(root), filepath.Join(root, "b", "link")))
			case "symlink-replacement":
				check(os.Remove(file))
				check(os.Symlink(filepath.Dir(root), file))
			case "internal-hardlink":
				link := filepath.Join(root, "b", "alias")
				check(os.Link(file, link))
				assertLiveDiskTotal(t, g, false)
				check(os.Truncate(link, 99))
			case "external-hardlink":
				// After creation has been consumed, only a watch on the regular
				// inode can report growth through this outside directory entry.
				link := filepath.Join(filepath.Dir(root), "outside")
				check(os.Link(file, link))
				assertLiveDiskTotal(t, g, false)
				check(os.Truncate(link, 999))
			case "sparse":
				check(os.Truncate(file, 1<<30))
			case "fallocate":
				f, err := os.OpenFile(file, os.O_WRONLY, 0)
				check(err)
				check(unix.Fallocate(int(f.Fd()), 0, 0, 8192))
				check(f.Close())
			}
			assertLiveDiskTotal(t, g, false)
		})
	}
}

func TestLiveDiskInvalidation(t *testing.T) {
	for _, kind := range []string{"new-directory", "directory-rename", "directory-removal", "root-replaced", "watch-lost", "mount-change"} {
		t.Run(kind, func(t *testing.T) {
			root, g := liveDiskFixture(t)
			var err error
			switch kind {
			case "new-directory":
				err = os.Mkdir(filepath.Join(root, "new"), 0700)
				if err == nil {
					err = os.WriteFile(filepath.Join(root, "new", "file"), []byte("included"), 0600)
				}
			case "directory-rename":
				err = os.Rename(filepath.Join(root, "a"), filepath.Join(root, "renamed"))
			case "directory-removal":
				err = os.Remove(filepath.Join(root, "b"))
			case "root-replaced":
				err = os.Rename(root, root+"-previous")
				if err == nil {
					err = os.Mkdir(root, 0700)
				}
				if err == nil {
					err = os.WriteFile(filepath.Join(root, "new"), []byte("new tree"), 0600)
				}
			case "watch-lost":
				_, err = unix.InotifyRmWatch(g.fd, uint32(g.dirs[root].wd))
			case "mount-change":
				g.mounts = []byte("previous mount observation")
			}
			if err != nil {
				t.Fatal(err)
			}
			fallback := kind != "new-directory" && kind != "directory-removal"
			assertLiveDiskTotal(t, g, fallback)
			if fallback && g.fd != -1 {
				t.Fatal("invalid observer leaked its descriptor")
			}
		})
	}
}

func TestLiveDiskQueueOverflowFallsBack(t *testing.T) {
	root, g := liveDiskFixture(t)
	limitRaw, err := os.ReadFile("/proc/sys/fs/inotify/max_queued_events")
	if err != nil {
		t.Fatal(err)
	}
	limit, err := strconv.Atoi(strings.TrimSpace(string(limitRaw)))
	if err != nil || limit > 65536 {
		t.Fatalf("bounded overflow fixture requires queue <=65536: %s %v", limitRaw, err)
	}
	// Alternating watched inodes prevents identical adjacent events coalescing.
	a := filepath.Join(root, "a", "file")
	b := filepath.Join(root, "b", "other")
	if err := os.WriteFile(b, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := g.Check(nil); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < limit+2; i++ {
		path := a
		if i%2 == 0 {
			path = b
		}
		if err := os.Truncate(path, int64(i+1)); err != nil {
			t.Fatal(err)
		}
	}
	assertLiveDiskTotal(t, g, true)
	if !strings.Contains(g.fallback, "notifications lost") {
		t.Fatalf("overflow reason: %s", g.fallback)
	}
}

func TestLiveDiskRefusesMissingObservationsAndFreeSpace(t *testing.T) {
	for _, kind := range []string{"missing-root", "unreadable-directory", "low-free-space"} {
		t.Run(kind, func(t *testing.T) {
			root, g := liveDiskFixture(t)
			var err error
			switch kind {
			case "missing-root":
				err = os.Rename(root, root+"-moved")
			case "unreadable-directory":
				if os.Geteuid() == 0 {
					t.Skip("permission denial needs an unprivileged test process")
				}
				path := filepath.Join(root, "a")
				err = os.Chmod(path, 0000)
				defer os.Chmod(path, 0700)
			case "low-free-space":
				g.limits.MinimumFreeBytes = ^uint64(0)
			}
			if err != nil {
				t.Fatal(err)
			}
			if err = g.Check(nil); err == nil {
				t.Fatal("failed observation or floor was accepted")
			}
		})
	}
}

func TestLiveDiskCancellationJoinsSetupAndCloses(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 8192; i++ {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("%06d", i)), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	stop := make(chan struct{})
	result := make(chan error, 1)
	g := newLiveDiskGuard(root, Limits{MaxBytes: 1 << 40})
	go func() { result <- g.Check(stop) }()
	time.Sleep(time.Millisecond)
	begin := time.Now()
	close(stop)
	select {
	case err := <-result:
		if !errors.Is(err, errDiskObservationStopped) {
			t.Fatalf("setup cancellation: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("setup failed to join on cancellation")
	}
	fd := g.fd
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
	if fd >= 0 {
		if _, err := unix.FcntlInt(uintptr(fd), unix.F_GETFD, 0); !errors.Is(err, unix.EBADF) {
			t.Fatalf("watch descriptor remains open: %v", err)
		}
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
	t.Logf("cancellation and close: %s", time.Since(begin))
}
