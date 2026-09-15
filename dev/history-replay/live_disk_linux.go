//go:build linux && amd64

package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

// liveDiskGuard caches logical sizes only while every regular inode and
// directory is watched. File watches also cover writes through an outside
// hardlink. Lost events, moved directories, unsupported filesystems
// and observation errors revert this request to the original full scanner.
// It is single-owner state: Check and Close are called by the disk observer.
// Independent preflight/postflight full scans and the polling cadence remain.
// See https://man7.org/linux/man-pages/man7/inotify.7.html for notification gaps.
type liveDiskGuard struct {
	root                            string
	limits                          Limits
	fd                              int
	initialized                     bool
	fallback                        string
	rootInfo                        os.FileInfo
	mounts                          []byte
	dirs                            map[string]*diskDirectory
	watches                         map[int]*diskWatch
	total                           int64
	checks, fullScans, cachedChecks int
}

type diskFile struct {
	size     int64
	wd       int
	dev, ino uint64
}

type diskDirectory struct {
	path  string
	info  os.FileInfo
	wd    int
	files map[string]diskFile
}

type diskWatch struct {
	directory string
	paths     map[string]bool
	dev, ino  uint64
	dead      bool
}

func newLiveDiskGuard(root string, limits Limits) *liveDiskGuard {
	return &liveDiskGuard{root: filepath.Clean(root), limits: limits, fd: -1}
}

func diskObservationCancelled(stop <-chan struct{}) error {
	select {
	case <-stop:
		return errDiskObservationStopped
	default:
		return nil
	}
}

func (g *liveDiskGuard) Close() error {
	if g.fd < 0 {
		return nil
	}
	fd := g.fd
	g.fd = -1
	g.dirs, g.watches = nil, nil
	return unix.Close(fd)
}

func (g *liveDiskGuard) disable(reason string) error {
	g.fallback = reason
	return g.Close()
}

func (g *liveDiskGuard) Status() map[string]interface{} {
	return map[string]interface{}{"checks": g.checks, "fullScans": g.fullScans, "cachedChecks": g.cachedChecks, "fallbackReason": g.fallback, "watchedInodes": len(g.watches)}
}

func (g *liveDiskGuard) Check(stop <-chan struct{}) error {
	g.checks++
	if err := diskObservationCancelled(stop); err != nil {
		return err
	}
	if err := freeDiskGuard(g.root, g.limits.MinimumFreeBytes); err != nil {
		return err
	}
	if !g.initialized {
		g.initialized = true
		if err := g.initialize(stop); err != nil {
			if errors.Is(err, errDiskObservationStopped) {
				return err
			}
			if closeErr := g.disable(err.Error()); closeErr != nil {
				return closeErr
			}
		}
	}
	if g.fallback == "" {
		if err := g.update(stop); err != nil {
			if errors.Is(err, errDiskObservationStopped) {
				return err
			}
			if closeErr := g.disable(err.Error()); closeErr != nil {
				return closeErr
			}
		} else {
			g.cachedChecks++
			if g.total > g.limits.MaxBytes {
				return errors.New("NOT_RUN_LIMIT: artifact size ceiling")
			}
			return nil
		}
	}
	g.fullScans++
	return diskGuardUntil(g.root, g.limits, stop)
}

func (g *liveDiskGuard) initialize(stop <-chan struct{}) error {
	var fs unix.Statfs_t
	if err := unix.Statfs(g.root, &fs); err != nil {
		return err
	}
	switch fs.Type {
	case unix.BTRFS_SUPER_MAGIC, unix.EXT4_SUPER_MAGIC, unix.XFS_SUPER_MAGIC, unix.TMPFS_MAGIC:
	default:
		return fmt.Errorf("full scan required for filesystem %x", fs.Type)
	}
	var err error
	if g.mounts, err = os.ReadFile("/proc/self/mountinfo"); err != nil {
		return err
	}
	unescape := strings.NewReplacer("\\040", " ", "\\011", "\t", "\\012", "\n", "\\134", "\\")
	for _, line := range strings.Split(string(g.mounts), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 4 {
			mount := unescape.Replace(fields[4])
			if mount != g.root && inside(g.root, mount) {
				return errors.New("full scan required for nested mount")
			}
		}
	}
	if g.rootInfo, err = os.Lstat(g.root); err != nil {
		return err
	}
	if !g.rootInfo.IsDir() {
		return errors.New("full scan required for non-directory root")
	}
	if g.fd, err = unix.InotifyInit1(unix.IN_NONBLOCK | unix.IN_CLOEXEC); err != nil {
		return err
	}
	g.dirs = map[string]*diskDirectory{}
	g.watches = map[int]*diskWatch{}
	return g.addTree(g.root, stop)
}

func (g *liveDiskGuard) addTree(root string, stop <-chan struct{}) error {
	queue := []string{root}
	for len(queue) > 0 {
		if err := diskObservationCancelled(stop); err != nil {
			return err
		}
		path := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		info, e := os.Lstat(path)
		if e != nil {
			return e
		}
		if !info.IsDir() {
			return errors.New("directory changed during watch setup")
		}
		wd, e := unix.InotifyAddWatch(g.fd, path, unix.IN_ONLYDIR|unix.IN_DONT_FOLLOW|unix.IN_CREATE|unix.IN_DELETE|unix.IN_MOVED_FROM|unix.IN_MOVED_TO|unix.IN_ATTRIB|unix.IN_MOVE_SELF|unix.IN_DELETE_SELF)
		if e != nil {
			return e
		}
		if _, exists := g.watches[wd]; exists {
			return errors.New("aliased or recycled directory watch")
		}
		dir := &diskDirectory{path: path, info: info, wd: wd, files: map[string]diskFile{}}
		g.dirs[path] = dir
		g.watches[wd] = &diskWatch{directory: path}
		children, e := g.scanDirectory(dir, stop, true)
		if e != nil {
			return e
		}
		queue = append(queue, children...)
	}
	return nil
}

func (g *liveDiskGuard) scanDirectory(dir *diskDirectory, stop <-chan struct{}, initial bool) ([]string, error) {
	file, err := os.OpenFile(dir.path, os.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(info, dir.info) {
		return nil, errors.New("watched directory identity changed")
	}
	next := map[string]diskFile{}
	children := []string{}
	for {
		if err = diskObservationCancelled(stop); err != nil {
			return nil, err
		}
		names, readErr := file.Readdirnames(256)
		for _, name := range names {
			if err = diskObservationCancelled(stop); err != nil {
				return nil, err
			}
			var st unix.Stat_t
			if err = unix.Fstatat(int(file.Fd()), name, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
				return nil, err
			}
			path := filepath.Join(dir.path, name)
			switch st.Mode & unix.S_IFMT {
			case unix.S_IFDIR:
				if initial {
					children = append(children, path)
				} else if g.dirs[path] == nil {
					if err := g.addTree(path, stop); err != nil {
						return nil, err
					}
				}
			case unix.S_IFREG:
				item, exists := dir.files[name]
				if !exists || item.dev != uint64(st.Dev) || item.ino != st.Ino || g.watches[item.wd] == nil || g.watches[item.wd].dead {
					wd, e := unix.InotifyAddWatch(g.fd, path, unix.IN_DONT_FOLLOW|unix.IN_MODIFY|unix.IN_ATTRIB|unix.IN_CLOSE_WRITE|unix.IN_MOVE_SELF|unix.IN_DELETE_SELF)
					if e != nil {
						return nil, e
					}
					// A path replaced while a watch is installed cannot qualify this count.
					var verify unix.Stat_t
					if e = unix.Fstatat(int(file.Fd()), name, &verify, unix.AT_SYMLINK_NOFOLLOW); e != nil {
						return nil, e
					}
					if verify.Dev != st.Dev || verify.Ino != st.Ino || verify.Mode&unix.S_IFMT != unix.S_IFREG {
						return nil, errors.New("file changed during watch setup")
					}
					st = verify
					watch := g.watches[wd]
					if watch == nil {
						watch = &diskWatch{paths: map[string]bool{}, dev: uint64(st.Dev), ino: st.Ino}
						g.watches[wd] = watch
					}
					if watch.dead || watch.directory != "" || watch.dev != uint64(st.Dev) || watch.ino != st.Ino {
						return nil, errors.New("watch descriptor identity changed")
					}
					watch.paths[path] = true
					item = diskFile{wd: wd, dev: uint64(st.Dev), ino: st.Ino}
				}
				item.size = st.Size
				next[name] = item
			}
		}
		if readErr != nil {
			if !errors.Is(readErr, io.EOF) {
				return nil, readErr
			}
			break
		}
	}
	for name, old := range dir.files {
		g.total -= old.size
		if replacement, exists := next[name]; !exists || replacement.wd != old.wd {
			delete(g.watches[old.wd].paths, filepath.Join(dir.path, name))
		}
	}
	for _, item := range next {
		g.total += item.size
	}
	dir.files = next
	return children, nil
}

func (g *liveDiskGuard) events(stop <-chan struct{}) (map[string]bool, error) {
	dirty := map[string]bool{}
	var buf [65536]byte
	// Bounded work under a continuous writer. Queue loss or excessive churn
	// returns to the original scanner instead of retaining an incomplete count.
	for reads := 0; reads < 64; reads++ {
		if err := diskObservationCancelled(stop); err != nil {
			return nil, err
		}
		n, err := unix.Read(g.fd, buf[:])
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if errors.Is(err, unix.EAGAIN) {
			return dirty, nil
		}
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, errors.New("notification stream closed")
		}
		for at := 0; at < n; {
			if n-at < unix.SizeofInotifyEvent {
				return nil, errors.New("truncated notification")
			}
			wd := int(int32(binary.LittleEndian.Uint32(buf[at:])))
			mask := binary.LittleEndian.Uint32(buf[at+4:])
			length := int(binary.LittleEndian.Uint32(buf[at+12:]))
			if length > n-at-unix.SizeofInotifyEvent {
				return nil, errors.New("truncated notification name")
			}
			at += unix.SizeofInotifyEvent + length
			if mask&(unix.IN_Q_OVERFLOW|unix.IN_UNMOUNT) != 0 {
				return nil, errors.New("filesystem notifications lost")
			}
			watch := g.watches[wd]
			if watch == nil {
				return nil, errors.New("unknown filesystem watch")
			}
			if watch.directory != "" {
				if mask&unix.IN_MOVE_SELF != 0 || mask&unix.IN_ISDIR != 0 && mask&(unix.IN_MOVED_FROM|unix.IN_MOVED_TO) != 0 {
					return nil, errors.New("directory topology changed")
				}
				if mask&(unix.IN_DELETE_SELF|unix.IN_IGNORED) != 0 {
					watch.dead = true
				}
				dirty[watch.directory] = true
			} else {
				for path := range watch.paths {
					dirty[filepath.Dir(path)] = true
				}
				if mask&unix.IN_IGNORED != 0 {
					watch.dead = true
				}
			}
		}
	}
	return nil, errors.New("filesystem notification churn exceeded observation budget")
}

func (g *liveDiskGuard) removeTree(root string) {
	for path, dir := range g.dirs {
		if path != root && !inside(root, path) {
			continue
		}
		for name, item := range dir.files {
			g.total -= item.size
			delete(g.watches[item.wd].paths, filepath.Join(path, name))
		}
		// Keep descriptor tombstones. Recycled or aliased watches cannot
		// silently reinterpret queued events from a different directory.
		g.watches[dir.wd].dead = true
		delete(g.dirs, path)
	}
}

func (g *liveDiskGuard) update(stop <-chan struct{}) error {
	info, err := os.Lstat(g.root)
	if err != nil {
		return err
	}
	if !os.SameFile(info, g.rootInfo) {
		return errors.New("root identity changed")
	}
	mounts, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return err
	}
	if !bytes.Equal(g.mounts, mounts) {
		return errors.New("mount topology changed")
	}
	dirty, err := g.events(stop)
	if err != nil {
		return err
	}
	// Deleted directories must be removed before their parents are rescanned.
	// A still-present directory with a lost watch instead requires a full scan.
	for path := range dirty {
		dir := g.dirs[path]
		if dir == nil || !g.watches[dir.wd].dead {
			continue
		}
		if _, err := os.Lstat(path); os.IsNotExist(err) && path != g.root {
			g.removeTree(path)
		} else {
			return errors.New("directory watch lost")
		}
	}
	for path := range dirty {
		dir := g.dirs[path]
		if dir == nil {
			continue // A deleted subtree was removed above.
		}
		if _, err := g.scanDirectory(dir, stop, false); err != nil {
			return err
		}
	}
	// Notifications generated during a scan are deliberately retained for the
	// next poll. As with the full scanner, this is a live observation, not an
	// atomic filesystem snapshot. A complete postflight scan remains mandatory.
	return nil
}
