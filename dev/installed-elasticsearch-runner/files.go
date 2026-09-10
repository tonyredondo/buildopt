package main

import (
	"bytes"
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

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
)

type entry struct {
	Path   string `json:"path"`
	Type   string `json:"type"`
	Mode   uint32 `json:"mode"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

func hashFile(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("expected regular file")
	}
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

func readJSON(path string, target any) error {
	return readJSONLimit(path, target, 4<<20)
}

// Large native inventories have a separate bounded reader; ordinary protocol
// records retain their original 4 MiB limit.
func readJSONLimit(path string, target any, limit int64) error {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	if resolved != filepath.Clean(path) {
		return errors.New("symlink in evidence path")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return err
	}
	if int64(len(b)) > limit {
		return errors.New("JSON exceeds record size limit")
	}
	if _, err = contractcrypto.CanonicalizeJCS(b); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err = d.Decode(target); err != nil {
		return err
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return errors.New("trailing JSON")
	}
	return nil
}

func writeJSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeNew(path, append(b, '\n'))
}

func writeNew(path string, b []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	if _, err = f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// inventory covers complete declared subtrees, including directory modes and
// explicit absence. It never normalizes bytes or follows links. Overlapping
// selectors cannot hide an entry or assign it two meanings.
func inventory(root string, selectors []string) ([]entry, error) {
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	if !filepath.IsAbs(root) || canonical != root {
		return nil, errors.New("root must be canonical and absolute")
	}
	sorted := append([]string{}, selectors...)
	sort.Strings(sorted)
	if len(sorted) == 0 {
		return nil, errors.New("empty output contract")
	}
	for i, s := range sorted {
		if !filepath.IsLocal(s) || filepath.Clean(s) != s || s == "." || strings.Contains(s, "\\") {
			return nil, errors.New("unsafe selector")
		}
		for _, prior := range sorted[:i] {
			if s == prior || strings.HasPrefix(s, prior+"/") {
				return nil, errors.New("overlapping selectors")
			}
		}
	}
	entries := []entry{}
	for _, selector := range sorted {
		path := root
		for _, part := range strings.Split(selector, "/") {
			path = filepath.Join(path, part)
			info, e := os.Lstat(path)
			if os.IsNotExist(e) {
				break
			}
			if e != nil {
				return nil, e
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return nil, errors.New("symlink in selector")
			}
		}
		err = filepath.WalkDir(filepath.Join(root, selector), func(path string, d fs.DirEntry, e error) error {
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			if os.IsNotExist(e) && path == filepath.Join(root, selector) {
				entries = append(entries, entry{Path: rel, Type: "absent"})
				return nil
			}
			if e != nil {
				return e
			}
			info, e := d.Info()
			if e != nil {
				return e
			}
			item := entry{Path: rel, Mode: uint32(info.Mode().Perm())}
			switch {
			case info.IsDir():
				item.Type = "directory"
			case info.Mode().IsRegular():
				item.Type = "file"
				item.Size = info.Size()
				item.SHA256, e = hashFile(path)
				if e != nil {
					return e
				}
			default:
				return fmt.Errorf("unsupported output type: %s", rel)
			}
			if info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
				return errors.New("special permission bits")
			}
			entries = append(entries, item)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}

func copyTree(source, target string) error {
	type directoryMode struct {
		path string
		mode fs.FileMode
	}
	directories := []directoryMode{}
	err := filepath.WalkDir(source, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		dest := filepath.Join(target, rel)
		if info.IsDir() {
			if err := os.Mkdir(dest, 0700); err != nil {
				return err
			}
			directories = append(directories, directoryMode{dest, info.Mode().Perm()})
			return nil
		}
		if !info.Mode().IsRegular() {
			return errors.New("cannot retain symlink or special file")
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
		if err != nil {
			return err
		}
		if _, err = io.Copy(out, in); err != nil {
			out.Close()
			return err
		}
		if err = out.Chmod(info.Mode().Perm()); err != nil {
			out.Close()
			return err
		}
		return out.Close()
	})
	if err != nil {
		return err
	}
	// Restore exact directory modes only after their children are copied. This
	// preserves both group permissions and read-only trees despite the umask.
	for i := len(directories) - 1; i >= 0; i-- {
		if err = os.Chmod(directories[i].path, directories[i].mode); err != nil {
			return err
		}
	}
	return nil
}
