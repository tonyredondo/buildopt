//go:build linux && amd64

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"golang.org/x/sys/unix"
)

type Entry struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Mode   uint32 `json:"mode"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
	Target string `json:"target"`
}

func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func jsonBytes(v any) []byte {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	return append(b, '\n')
}
func objectDigest(v any) string { return digest(jsonBytes(v)) }
func fileDigest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func inside(root, path string) bool {
	r, e := filepath.Rel(root, path)
	return e == nil && (r == "." || relativePath(r))
}

func entryAt(root, path string) (Entry, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return Entry{}, err
	}
	e := Entry{Path: rel, Kind: "absent"}
	s, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return e, nil
	}
	if err != nil {
		return e, err
	}
	e.Mode = uint32(s.Mode().Perm())
	e.Size = s.Size()
	switch {
	case s.Mode().IsRegular():
		e.Kind = "file"
		e.SHA256, err = fileDigest(path)
	case s.IsDir():
		e.Kind = "directory"
		e.Size = 0
	case s.Mode()&os.ModeSymlink != 0:
		e.Kind = "symlink"
		e.Target, err = os.Readlink(path)
		e.SHA256 = digest([]byte(e.Target))
	default:
		return e, fmt.Errorf("unsupported special file %s", path)
	}
	return e, err
}

func inventory(root string) ([]Entry, error) {
	entries := []Entry{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		e, err := entryAt(root, path)
		if err == nil {
			entries = append(entries, e)
		}
		return err
	})
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, err
}

func bindingDigest(path string) (string, error) {
	s, e := os.Lstat(path)
	if e != nil {
		return "", e
	}
	if s.Mode().IsRegular() {
		return fileDigest(path)
	}
	if s.IsDir() {
		entries, e := inventory(path)
		if e != nil {
			return "", e
		}
		return objectDigest(entries), nil
	}
	return "", errors.New("bound input must be a regular file or directory")
}
func checkBinding(b Binding) error {
	h, e := bindingDigest(b.Path)
	if e != nil {
		return e
	}
	if h != b.SHA256 {
		return fmt.Errorf("binding drift: %s", b.Path)
	}
	return nil
}

func syncDir(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}
func writeExclusive(path string, data []byte, mode fs.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	_, w := f.Write(data)
	s := f.Sync()
	c := f.Close()
	if w != nil {
		return w
	}
	if s != nil {
		return s
	}
	if c != nil {
		return c
	}
	return syncDir(filepath.Dir(path))
}
func atomicJSON(path string, v any) error {
	return atomicBytes(path, jsonBytes(v))
}
func atomicBytes(path string, data []byte) error {
	tmp := path + ".writing"
	if err := writeExclusive(tmp, data, 0600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	return syncDir(filepath.Dir(path))
}

func copyTree(src, dst string) error {
	if _, err := os.Lstat(dst); !os.IsNotExist(err) {
		return fmt.Errorf("copy destination exists: %s", dst)
	}
	s, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if s.IsDir() {
		// Populate even read-only source directories, then restore their exact
		// mode. Mkdir/OpenFile alone apply the caller's umask and change state.
		if err = os.Mkdir(dst, s.Mode().Perm()|0700); err != nil {
			return err
		}
		children, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, c := range children {
			if err = copyTree(filepath.Join(src, c.Name()), filepath.Join(dst, c.Name())); err != nil {
				return err
			}
		}
		return os.Chmod(dst, s.Mode().Perm())
	}
	if s.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(target, dst)
	}
	if !s.Mode().IsRegular() {
		return fmt.Errorf("cannot snapshot special file %s", src)
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, s.Mode().Perm())
	if err != nil {
		return err
	}
	// Reflink retains an independent writable inode; fall back to copying bytes.
	err = unix.IoctlFileClone(int(out.Fd()), int(in.Fd()))
	if err != nil {
		_, err = io.Copy(out, in)
	}
	if err == nil {
		err = out.Chmod(s.Mode().Perm())
	}
	if err == nil {
		err = out.Sync()
	}
	closeErr := out.Close()
	if err != nil {
		return err
	}
	return closeErr
}

type fileCopy struct{ Source, Destination string }

// Retained evidence has independent files and a durability barrier per file.
// Bound the outstanding I/O so the filesystem can commit those barriers
// together. This runs after native completion, outside customer timing.
func copyEvidenceFiles(files []fileCopy) error {
	return copyEvidenceFilesWith(files, func(job fileCopy) error { return copyTree(job.Source, job.Destination) })
}

func copyEvidenceFilesWith(files []fileCopy, copyFile func(fileCopy) error) error {
	jobs := make(chan fileCopy)
	var workers sync.WaitGroup
	var lock sync.Mutex
	var firstErr error
	for worker := 0; worker < 8; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for job := range jobs {
				lock.Lock()
				failed := firstErr != nil
				lock.Unlock()
				if failed {
					continue
				}
				if err := copyFile(job); err != nil {
					lock.Lock()
					if firstErr == nil {
						firstErr = err
					}
					lock.Unlock()
				}
			}
		}()
	}
	for _, file := range files {
		jobs <- file
	}
	close(jobs)
	workers.Wait()
	return firstErr
}

// Only retained evidence shares inodes. Native outputs remain independent and
// writable. Verify every reused object; never repair a corrupt evidence object
// silently. All object bytes are durable before publishing attempt output names.
func retainOutputFiles(files []fileCopy, store string) error {
	if err := os.MkdirAll(store, 0700); err != nil {
		return err
	}
	err := copyEvidenceFilesWith(files, func(job fileCopy) error {
		source, err := os.Lstat(job.Source)
		if err != nil {
			return err
		}
		if !source.Mode().IsRegular() {
			return fmt.Errorf("nonregular retained source: %s", job.Source)
		}
		hash, err := fileDigest(job.Source)
		if err != nil {
			return err
		}
		object := filepath.Join(store, fmt.Sprintf("%s-%04o", hash, source.Mode().Perm()))
		verify := func() error {
			info, err := os.Lstat(object)
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() || info.Size() != source.Size() || info.Mode().Perm() != source.Mode().Perm() || os.SameFile(source, info) {
				return fmt.Errorf("invalid retained object: %s", object)
			}
			actual, err := fileDigest(object)
			if err != nil {
				return err
			}
			if actual != hash {
				return fmt.Errorf("corrupt retained object: %s", object)
			}
			return nil
		}
		if _, err := os.Lstat(object); os.IsNotExist(err) {
			tmp, err := os.MkdirTemp(store, ".populate-")
			if err != nil {
				return err
			}
			defer os.RemoveAll(tmp)
			copied := filepath.Join(tmp, "bytes")
			if err := copyTree(job.Source, copied); err != nil {
				return err
			}
			actual, err := fileDigest(copied)
			if err != nil {
				return err
			}
			if actual != hash {
				return fmt.Errorf("source changed during retention: %s", job.Source)
			}
			// An identical concurrent publisher may win; its bytes still need verification.
			if err := os.Link(copied, object); err != nil && !os.IsExist(err) {
				return err
			}
		} else if err != nil {
			return err
		}
		if err := verify(); err != nil {
			return err
		}
		return os.Link(object, job.Destination)
	})
	if err != nil {
		return err
	}
	if err := syncDir(store); err != nil {
		return err
	}
	return syncDir(filepath.Dir(store))
}

var errDiskObservationStopped = errors.New("disk observation stopped after child exit")

func treeBytes(root string) (int64, error) { return treeBytesUntil(root, nil) }

// A live resource sample needs neither sorted names nor content hashes. Read
// bounded directory batches so cancellation does not wait for all retained
// evidence, including a flat directory with tens of thousands of output files.
func treeBytesUntil(root string, stop <-chan struct{}) (int64, error) {
	info, err := os.Lstat(root)
	if err != nil {
		return 0, err
	}
	if !info.IsDir() {
		if info.Mode().IsRegular() {
			return info.Size(), nil
		}
		return 0, nil
	}
	var total int64
	queue := []string{root}
	readDirectory := func(path string) error {
		directory, err := os.OpenFile(path, os.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
		if err != nil {
			if path != root && os.IsNotExist(err) {
				return nil
			}
			return err
		}
		defer directory.Close()
		fd := int(directory.Fd())
		for {
			select {
			case <-stop:
				return errDiskObservationStopped
			default:
			}
			names, readErr := directory.Readdirnames(256)
			var info unix.Stat_t
			for _, name := range names {
				select {
				case <-stop:
					return errDiskObservationStopped
				default:
				}
				// Resolve metadata relative to the open directory, avoiding a
				// full-path lookup and FileInfo allocation for every entry.
				err := unix.Fstatat(fd, name, &info, unix.AT_SYMLINK_NOFOLLOW)
				if err != nil {
					// Native temporary descendants may disappear during a sample.
					if os.IsNotExist(err) {
						continue
					}
					return err
				}
				if info.Mode&unix.S_IFMT == unix.S_IFDIR {
					queue = append(queue, filepath.Join(path, name))
				} else if info.Mode&unix.S_IFMT == unix.S_IFREG {
					total += info.Size
				}
			}
			if readErr == io.EOF {
				return nil
			}
			if readErr != nil {
				if path != root && os.IsNotExist(readErr) {
					return nil
				}
				return readErr
			}
		}
	}
	for len(queue) > 0 {
		select {
		case <-stop:
			return total, errDiskObservationStopped
		default:
		}
		path := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		if err := readDirectory(path); err != nil {
			return total, err
		}
	}
	return total, nil
}
func diskGuard(root string, l Limits) error {
	return diskGuardUntil(root, l, nil)
}
func diskGuardUntil(root string, l Limits, stop <-chan struct{}) error {
	if err := freeDiskGuard(root, l.MinimumFreeBytes); err != nil {
		return err
	}
	n, err := treeBytesUntil(root, stop)
	if err != nil {
		return err
	}
	if n > l.MaxBytes {
		return errors.New("NOT_RUN_LIMIT: artifact size ceiling")
	}
	return nil
}

func freeDiskGuard(root string, minimumFreeBytes uint64) error {
	var s unix.Statfs_t
	if err := unix.Statfs(root, &s); err != nil {
		return err
	}
	if s.Bavail*uint64(s.Bsize) < minimumFreeBytes {
		return errors.New("NOT_RUN_LIMIT: free disk floor")
	}
	return nil
}

// Generated path patterns have only whole-component ** and shell * / ?.
func pathPattern(pattern, path string) bool {
	p := strings.Split(pattern, "/")
	s := strings.Split(path, "/")
	var match func(int, int) bool
	match = func(i, j int) bool {
		if i == len(p) {
			return j == len(s)
		}
		if p[i] == "**" {
			return match(i+1, j) || (j < len(s) && match(i, j+1))
		}
		if j == len(s) {
			return false
		}
		ok, e := filepath.Match(p[i], s[j])
		return e == nil && ok && match(i+1, j+1)
	}
	return match(0, 0)
}
