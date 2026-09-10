//go:build linux && amd64

package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Hash historical blobs in one Git process. Reuse immutable blob IDs across
// recorded states; per-file Git subprocesses make a large owner impractical.
func historicalHashes(common string, files []sourceFile, cache map[string]string) error {
	missing := []string{}
	seen := map[string]bool{}
	for _, f := range files {
		if cache[f.blob] == "" && !seen[f.blob] {
			missing = append(missing, f.blob)
			seen[f.blob] = true
		}
	}
	if len(missing) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, "git", "-C", common, "cat-file", "--batch")
	c.Stdin = strings.NewReader(strings.Join(missing, "\n") + "\n")
	stdout, err := c.StdoutPipe()
	if err != nil {
		return err
	}
	if err = c.Start(); err != nil {
		return err
	}
	b := bufio.NewReader(stdout)
	for _, id := range missing {
		line, e := b.ReadString('\n')
		if e != nil {
			cancel()
			c.Wait()
			return e
		}
		fields := strings.Fields(line)
		if len(fields) != 3 || fields[0] != id || fields[1] != "blob" {
			cancel()
			c.Wait()
			return errors.New("historical blob identity differs")
		}
		size, e := strconv.ParseInt(fields[2], 10, 64)
		if e != nil || size < 0 || size > 1<<30 {
			cancel()
			c.Wait()
			return errors.New("historical blob exceeds bound")
		}
		h := sha256.New()
		if _, e = io.CopyN(h, b, size); e != nil {
			cancel()
			c.Wait()
			return e
		}
		ch, e := b.ReadByte()
		if e != nil || ch != '\n' {
			cancel()
			c.Wait()
			return errors.New("historical blob framing differs")
		}
		cache[id] = hex.EncodeToString(h.Sum(nil))
	}
	return c.Wait()
}

func checkRecordedSource(m Manifest, s StatePin, cache map[string]string) error {
	files, err := sourceTree(m.CommonGit, s.Revision)
	if err != nil {
		return err
	}
	if err = historicalHashes(m.CommonGit, files, cache); err != nil {
		return err
	}
	entries := map[string]Entry{}
	for _, e := range s.Entries {
		if strings.HasPrefix(e.Path, "repo/") {
			e.Path = strings.TrimPrefix(e.Path, "repo/")
			entries[e.Path] = e
		}
	}
	owned := map[string]FilePatch{}
	for _, p := range patchesFor(m, s) {
		for _, f := range p.Files {
			owned[f.Path] = f
		}
	}
	tracked := map[string]bool{}
	for _, f := range files {
		tracked[f.path] = true
		if _, ok := owned[f.path]; ok {
			continue
		}
		e, ok := entries[f.path]
		if !ok || e.SHA256 != cache[f.blob] {
			return fmt.Errorf("recorded source bytes differ from Git: %s", f.path)
		}
		mode := "100644"
		if e.Kind == "symlink" {
			mode = "120000"
		} else if e.Kind != "file" {
			return errors.New("recorded source kind differs")
		} else if e.Mode&0111 != 0 {
			mode = "100755"
		}
		if mode != f.mode {
			return errors.New("recorded source mode differs")
		}
	}
	for path, f := range owned {
		e, ok := entries[path]
		if !ok || e.Kind != "file" || e.Mode != f.AfterMode || e.SHA256 != f.After.SHA256 {
			return errors.New("recorded patch postimage differs")
		}
	}
	for path, e := range entries {
		if path == ".git" || e.Kind == "directory" || tracked[path] {
			continue
		}
		if _, ok := owned[path]; ok {
			continue
		}
		allowed := false
		for _, pattern := range m.GeneratedPaths {
			if pathPattern(pattern, path) {
				allowed = true
			}
		}
		if !allowed {
			return fmt.Errorf("recorded untracked source drift: %s", path)
		}
	}
	return nil
}

func safePatchPath(repo, path string) error {
	if !relativePath(path) {
		return errors.New("invalid patch path")
	}
	parts := strings.Split(path, "/")
	current := repo
	for _, part := range parts[:len(parts)-1] {
		current = filepath.Join(current, part)
		s, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if !s.IsDir() {
			return errors.New("patch parent is not an owned directory")
		}
	}
	return nil
}

func gitAt(root string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
	c.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0")
	b, e := c.Output()
	if e != nil {
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), e)
	}
	return b, nil
}

func reconstructHistory(common, end string, count int) ([]Revision, error) {
	if !gitPattern.MatchString(end) || count < 1 || count > 101 {
		return nil, errors.New("invalid endpoint/count")
	}
	raw, err := gitAt(common, "rev-list", "--first-parent", "--max-count="+strconv.Itoa(count), "--reverse", end)
	if err != nil {
		return nil, err
	}
	commits := strings.Fields(string(raw))
	if len(commits) != count {
		return nil, errors.New("incomplete first-parent history")
	}
	rows := []Revision{}
	for i, c := range commits {
		b, e := gitAt(common, "show", "-s", "--format=%H%n%T%n%P%n%cI", c)
		if e != nil {
			return nil, e
		}
		f := strings.Split(strings.TrimSpace(string(b)), "\n")
		if len(f) != 4 {
			return nil, errors.New("malformed Git metadata")
		}
		parents := strings.Fields(f[2])
		parent := ""
		if len(parents) > 0 {
			parent = parents[0]
		}
		ts, e := time.Parse(time.RFC3339, f[3])
		if e != nil {
			return nil, e
		}
		args := []string{"diff-tree", "--no-commit-id", "--no-renames", "--name-status", "-r", "-z"}
		if parent == "" {
			args = append(args, "--root", c)
		} else {
			args = append(args, parent, c)
		}
		changed, e := gitAt(common, args...)
		if e != nil {
			return nil, e
		}
		rows = append(rows, Revision{i, c, parent, f[1], ts.UTC().Format(time.RFC3339), digest(changed)})
	}
	return rows, nil
}

func validateHistory(common string, rows []Revision) error {
	if len(rows) == 0 {
		return errors.New("missing history")
	}
	r, err := gitAt(common, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return err
	}
	real, err := filepath.EvalSymlinks(strings.TrimSpace(string(r)))
	if err != nil {
		return err
	}
	expected, err := filepath.EvalSymlinks(common)
	if err != nil || expected != real {
		return errors.New("shared Git identity differs")
	}
	seen := map[string]bool{}
	for i, r := range rows {
		if r.Ordinal != i || seen[r.Commit] || !gitPattern.MatchString(r.Commit) || !gitPattern.MatchString(r.Tree) || !shaPattern.MatchString(r.ChangedPathsSHA256) {
			return errors.New("missing/duplicate ordinal, revision or digest")
		}
		seen[r.Commit] = true
		if i > 0 && r.Parent != rows[i-1].Commit {
			return errors.New("non-consecutive first-parent edge")
		}
	}
	actual, err := reconstructHistory(common, rows[len(rows)-1].Commit, len(rows))
	if err != nil {
		return err
	}
	if objectDigest(actual) != objectDigest(rows) {
		return errors.New("history/tree/date/path-digest drift")
	}
	return nil
}

type sourceFile struct {
	mode string
	blob string
	path string
}

func sourceTree(common, revision string) ([]sourceFile, error) {
	b, err := gitAt(common, "ls-tree", "-r", "-z", revision)
	if err != nil {
		return nil, err
	}
	files := []sourceFile{}
	for _, line := range bytes.Split(b, []byte{0}) {
		if len(line) == 0 {
			continue
		}
		parts := bytes.SplitN(line, []byte{'\t'}, 2)
		if len(parts) != 2 {
			return nil, errors.New("invalid ls-tree")
		}
		meta := strings.Fields(string(parts[0]))
		if len(meta) != 3 || meta[1] != "blob" {
			return nil, errors.New("submodule or unsupported Git entry")
		}
		files = append(files, sourceFile{meta[0], meta[2], string(parts[1])})
	}
	return files, nil
}

func gitBlob(path string, symlink bool) (string, error) {
	h := sha1.New()
	if symlink {
		b, err := os.Readlink(path)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "blob %d\x00", len(b))
		h.Write([]byte(b))
	} else {
		s, err := os.Stat(path)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "blob %d\x00", s.Size())
		f, err := os.Open(path)
		if err != nil {
			return "", err
		}
		_, err = io.Copy(h, f)
		f.Close()
		if err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func verifySource(common, repo, revision string, patches []Patch, generated []string) (string, error) {
	head, err := gitAt(repo, "rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(string(head)) != revision {
		return "", errors.New("source HEAD drift")
	}
	gd, err := gitAt(repo, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", err
	}
	if filepath.Clean(strings.TrimSpace(string(gd))) != filepath.Clean(common) {
		return "", errors.New("source common Git drift")
	}
	files, err := sourceTree(common, revision)
	if err != nil {
		return "", err
	}
	owned := map[string]FilePatch{}
	for _, p := range patches {
		for _, f := range p.Files {
			owned[f.Path] = f
		}
	}
	tracked := map[string]bool{}
	for _, f := range files {
		tracked[f.path] = true
		if _, ok := owned[f.path]; ok {
			continue
		}
		path := filepath.Join(repo, f.path)
		s, err := os.Lstat(path)
		if err != nil {
			return "", err
		}
		mode := "100644"
		if s.Mode()&os.ModeSymlink != 0 {
			mode = "120000"
		} else if !s.Mode().IsRegular() {
			return "", fmt.Errorf("source kind drift %s", f.path)
		} else if s.Mode().Perm()&0111 != 0 {
			mode = "100755"
		}
		if mode != f.mode {
			return "", fmt.Errorf("source mode drift %s", f.path)
		}
		h, err := gitBlob(path, mode == "120000")
		if err != nil {
			return "", err
		}
		if h != f.blob {
			return "", fmt.Errorf("source byte drift %s", f.path)
		}
	}
	for path, f := range owned {
		e, err := entryAt(repo, filepath.Join(repo, path))
		if err != nil {
			return "", err
		}
		if e.Kind != "file" || e.SHA256 != f.After.SHA256 || e.Mode != f.AfterMode {
			return "", fmt.Errorf("owned patch drift %s", path)
		}
	}
	// Enumerate ignored files too. Generated paths never exempt tracked files.
	err = filepath.WalkDir(repo, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		rel, _ := filepath.Rel(repo, path)
		if rel == "." {
			return nil
		}
		if rel == ".git" {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if tracked[rel] {
			return nil
		}
		if _, ok := owned[rel]; ok {
			return nil
		}
		for _, pattern := range generated {
			if pathPattern(pattern, rel) {
				return nil
			}
		}
		return fmt.Errorf("unexpected untracked source %s", rel)
	})
	if err != nil {
		return "", err
	}
	return objectDigest(struct {
		Revision string
		Patches  []Patch
	}{revision, patches}), nil
}

func patchApplicable(repo string, p Patch) (bool, error) {
	for path, hash := range p.Prerequisites {
		if err := safePatchPath(repo, path); err != nil {
			return false, err
		}
		e, err := entryAt(repo, filepath.Join(repo, path))
		if err != nil {
			return false, err
		}
		if e.Kind != "file" || e.SHA256 != hash {
			return false, nil
		}
	}
	for _, f := range p.Files {
		if err := safePatchPath(repo, f.Path); err != nil {
			return false, err
		}
		if err := checkBinding(f.After); err != nil {
			return false, err
		}
		e, err := entryAt(repo, filepath.Join(repo, f.Path))
		if err != nil {
			return false, err
		}
		if f.BeforeSHA256 == "" {
			if e.Kind != "absent" {
				return false, nil
			}
		} else if e.Kind != "file" || e.SHA256 != f.BeforeSHA256 || e.Mode != f.BeforeMode {
			return false, nil
		}
	}
	return true, nil
}

func applyPatch(repo string, p Patch) error {
	already := true
	for _, f := range p.Files {
		if err := safePatchPath(repo, f.Path); err != nil {
			return err
		}
		e, err := entryAt(repo, filepath.Join(repo, f.Path))
		if err != nil {
			return err
		}
		if e.Kind != "file" || e.SHA256 != f.After.SHA256 || e.Mode != f.AfterMode {
			already = false
		}
	}
	if already {
		return nil
	}
	ok, err := patchApplicable(repo, p)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("patch preconditions changed")
	}
	for _, f := range p.Files {
		path := filepath.Join(repo, f.Path)
		if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		raw, e := os.ReadFile(f.After.Path)
		if e != nil {
			return e
		}
		if f.BeforeSHA256 == "" {
			// Match the qualified candidate's git-apply source-write semantics.
			// Durable research receipts must not impose per-source-file fsyncs.
			var output *os.File
			output, e = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, os.FileMode(f.AfterMode))
			if e == nil {
				_, e = output.Write(raw)
				closeErr := output.Close()
				if e == nil {
					e = closeErr
				}
			}
		} else {
			e = os.WriteFile(path, raw, os.FileMode(f.AfterMode))
		}
		if e != nil {
			return e
		}
		if e = os.Chmod(path, os.FileMode(f.AfterMode)); e != nil {
			return e
		}
	}
	for _, f := range p.Files {
		e, err := entryAt(repo, filepath.Join(repo, f.Path))
		if err != nil {
			return err
		}
		if e.Kind != "file" || e.SHA256 != f.After.SHA256 || e.Mode != f.AfterMode {
			return errors.New("source postimage verification failed")
		}
	}
	return nil
}

func reversePatch(common, repo, revision string, p Patch) error {
	// Validate all postimages before any mutation.
	for _, f := range p.Files {
		e, err := entryAt(repo, filepath.Join(repo, f.Path))
		if err != nil {
			return err
		}
		if e.Kind != "file" || e.SHA256 != f.After.SHA256 || e.Mode != f.AfterMode {
			return fmt.Errorf("inverse refuses drift: %s", f.Path)
		}
	}
	for _, f := range p.Files {
		path := filepath.Join(repo, f.Path)
		if f.BeforeSHA256 == "" {
			if err := os.Remove(path); err != nil {
				return err
			}
			continue
		}
		raw, err := gitAt(common, "show", revision+":"+f.Path)
		if err != nil {
			return err
		}
		if digest(raw) != f.BeforeSHA256 {
			return errors.New("inverse preimage is not historical source")
		}
		if err = os.WriteFile(path, raw, os.FileMode(f.BeforeMode)); err != nil {
			return err
		}
		if err = os.Chmod(path, os.FileMode(f.BeforeMode)); err != nil {
			return err
		}
	}
	return nil
}
