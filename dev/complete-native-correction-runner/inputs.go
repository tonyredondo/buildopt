package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type fileBinding struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type sourceBinding struct {
	Revision      string        `json:"revision"`
	ArchiveSHA256 string        `json:"archiveSha256"`
	Files         []fileBinding `json:"files"`
	Outputs       outputPolicy  `json:"outputs"`
}
type runtimeBinding struct {
	Role    string `json:"role"`
	Version string `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
}
type subjectsDocument struct {
	Subjects []sourceBinding  `json:"subjects"`
	Runtimes []runtimeBinding `json:"runtimes"`
}
type protocol struct {
	Commands       map[string][]string `json:"commands"`
	ExecutionOrder []string            `json:"executionOrder"`
	Rows           []struct {
		ID       string `json:"id"`
		Category string `json:"category"`
		Command  string `json:"command"`
	} `json:"rows"`
}

func loadInputs(repo string) (subjectsDocument, protocol, error) {
	var subjects subjectsDocument
	var contract protocol
	data, err := os.ReadFile(filepath.Join(repo, "specs/poc-complete-native-correction-v1.subjects.json"))
	if err != nil {
		return subjects, contract, err
	}
	if err = json.Unmarshal(data, &subjects); err != nil {
		return subjects, contract, err
	}
	data, err = os.ReadFile(filepath.Join(repo, "specs/poc-complete-native-correction-v1.json"))
	if err != nil {
		return subjects, contract, err
	}
	if err = json.Unmarshal(data, &contract); err != nil {
		return subjects, contract, err
	}
	if len(subjects.Subjects) != 1 || len(subjects.Runtimes) != 2 || len(contract.ExecutionOrder) != 58 {
		return subjects, contract, errors.New("invalid CNC inputs")
	}
	return subjects, contract, nil
}

func gitOutput(ctx context.Context, root string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
	command.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	data, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(data)), nil
}

func verifyWorktree(ctx context.Context, root, common string, binding sourceBinding) error {
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	if canonical != root || !filepath.IsAbs(root) {
		return errors.New("source root must be canonical and absolute")
	}
	top, err := gitOutput(ctx, root, "rev-parse", "--show-toplevel")
	if err != nil || top != root {
		return errors.New("source is not the worktree root")
	}
	actualCommon, err := gitOutput(ctx, root, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return err
	}
	canonicalCommon, err := filepath.EvalSymlinks(common)
	if err != nil || canonicalCommon != actualCommon {
		return errors.New("shared Git directory mismatch")
	}
	registry, err := gitOutput(ctx, root, "worktree", "list", "--porcelain", "-z")
	if err != nil {
		return err
	}
	registered := false
	for _, record := range strings.Split(registry, "\x00\x00") {
		if strings.HasPrefix(record, "worktree "+root+"\x00") {
			registered = true
		}
	}
	if !registered {
		return errors.New("source worktree is not registered")
	}
	head, err := gitOutput(ctx, root, "rev-parse", "HEAD")
	if err != nil || head != binding.Revision {
		return errors.New("source revision drift")
	}
	branch, branchErr := gitOutput(ctx, root, "symbolic-ref", "-q", "HEAD")
	var detachedExit *exec.ExitError
	if !errors.As(branchErr, &detachedExit) || detachedExit.ExitCode() != 1 || branch != "" {
		return errors.New("source must be detached at the frozen revision")
	}
	status, err := gitOutput(ctx, root, "status", "--porcelain", "--untracked-files=all")
	if err != nil || status != "" {
		return errors.New("source is dirty")
	}
	command := exec.CommandContext(ctx, "git", "-C", root, "archive", "--format=tar", binding.Revision)
	hash := sha256.New()
	command.Stdout = hash
	if err = command.Run(); err != nil {
		return err
	}
	if fmt.Sprintf("%x", hash.Sum(nil)) != binding.ArchiveSHA256 {
		return errors.New("source archive drift")
	}
	for _, file := range binding.Files {
		path := filepath.Join(root, file.Path)
		if err = contained(root, path); err != nil {
			return err
		}
		got, err := fileDigest(path)
		if err != nil || got != file.SHA256 {
			return fmt.Errorf("source file drift: %s", file.Path)
		}
	}
	return nil
}

// Compare the installed tree to the pinned archive, including executable bytes
// and link targets. A matching release string alone cannot prove JDK identity.
func verifyRuntime(archive, root string, binding runtimeBinding) error {
	digest, err := fileDigest(archive)
	if err != nil || digest != binding.SHA256 {
		return errors.New("runtime archive drift")
	}
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer file.Close()
	zipped, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer zipped.Close()
	reader := tar.NewReader(zipped)
	seen := map[string]bool{}
	implicitDirectories := map[string]bool{}
	prefix := ""
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		parts := strings.SplitN(strings.TrimSuffix(header.Name, "/"), "/", 2)
		if prefix == "" {
			prefix = parts[0]
		}
		if prefix != parts[0] {
			return errors.New("multiple runtime archive roots")
		}
		if len(parts) == 1 {
			continue
		}
		relative := parts[1]
		if !filepath.IsLocal(relative) || filepath.Clean(relative) != relative || seen[relative] {
			return errors.New("unsafe or duplicate runtime archive member")
		}
		seen[relative] = true
		// Archives may omit directory headers for required parents (Corretto's
		// man directory does this). Admit only ancestors of verified members,
		// never arbitrary extra directories, files or symlinks in the install.
		for parent := filepath.Dir(relative); parent != "."; parent = filepath.Dir(parent) {
			implicitDirectories[parent] = true
		}
		path := filepath.Join(root, relative)
		// Ancestor symlinks are forbidden; an archived leaf symlink is checked below.
		if err = contained(root, filepath.Dir(path)); err != nil && filepath.Dir(path) != root {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if !info.IsDir() {
				return errors.New("runtime directory drift")
			}
		case tar.TypeReg, tar.TypeRegA:
			if !info.Mode().IsRegular() || info.Mode().Perm()&0111 != os.FileMode(header.Mode)&0111 {
				return errors.New("runtime mode drift")
			}
			hash := sha256.New()
			if _, err = io.Copy(hash, reader); err != nil {
				return err
			}
			got, err := fileDigest(path)
			if err != nil || got != fmt.Sprintf("%x", hash.Sum(nil)) {
				return errors.New("runtime binary drift")
			}
		case tar.TypeSymlink:
			target, err := os.Readlink(path)
			if err != nil || target != header.Linkname {
				return errors.New("runtime link drift")
			}
			if err = contained(root, filepath.Clean(filepath.Join(filepath.Dir(path), target))); err != nil {
				return errors.New("runtime link escapes root")
			}
		default:
			return errors.New("unsupported runtime archive member")
		}
	}
	if !seen["bin/java"] || !seen["release"] {
		return errors.New("runtime archive missing java or release identity")
	}
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if !seen[filepath.ToSlash(relative)] && !(entry.IsDir() && implicitDirectories[relative]) {
			return fmt.Errorf("unexpected installed runtime member: %s", filepath.ToSlash(relative))
		}
		return nil
	})
}

type packageManifest struct {
	Schema           string        `json:"schemaVersion"`
	Files            []fileBinding `json:"files"`
	ExecutableSHA256 string        `json:"executableSha256"`
}

var packageFiles = []string{
	"go.mod", "go.sum", "dev/run", "dev/toolchains.lock.yaml",
	"dev/run-complete-native-correction", "dev/check-complete-native-correction",
	"dev/complete-native-correction-runner/main.go", "dev/complete-native-correction-runner/state.go",
	"dev/complete-native-correction-runner/inputs.go", "dev/complete-native-correction-runner/capture.go",
	"dev/complete-native-correction-runner/preparation.go",
	"dev/complete-native-correction-runner/ownership.go", "dev/complete-native-correction-runner/ownership_test.go",
	"dev/complete-native-correction-runner/native.go", "dev/complete-native-correction-runner/native_test.go",
	"dev/complete-native-correction-runner/outputs.go", "dev/complete-native-correction-runner/outputs_test.go",
	"internal/contractcrypto/jcs.go", "dev/complete-native-correction.init.gradle",
	"dev/complete-native-correction-runner/capture_test.go", "dev/complete-native-correction-runner/inputs_test.go",
	"dev/complete-native-correction-runner/preparation_test.go",
	"dev/complete-native-correction-validator/main.go",
	"dev/complete-native-correction-validator/main_test.go",
	"internal/strictdiagnostic/report.go", "internal/strictdiagnostic/report_v2.go",
	"dev/gradle-critical-path.init.gradle",
	"docs/reference/complete-native-correction-capture.md",
	"specs/poc-complete-native-correction-v1.md", "specs/poc-complete-native-correction-v1.json",
	"specs/poc-complete-native-correction-v1.subjects.json",
}

func makePackage(repo string) (packageManifest, error) {
	manifest := packageManifest{Schema: "buildopt.cnc/capture-package/v1", Files: []fileBinding{}}
	executable, err := os.Executable()
	if err != nil {
		return manifest, err
	}
	manifest.ExecutableSHA256, err = fileDigest(executable)
	if err != nil {
		return manifest, err
	}
	for _, relative := range packageFiles {
		path := filepath.Join(repo, relative)
		if err := contained(repo, path); err != nil {
			return manifest, err
		}
		digest, err := fileDigest(path)
		if err != nil {
			return manifest, err
		}
		manifest.Files = append(manifest.Files, fileBinding{relative, digest})
	}
	return manifest, nil
}
func verifyPackage(repo, path string) (string, error) {
	var retained packageManifest
	if err := readJSON(path, &retained); err != nil {
		return "", err
	}
	actual, err := makePackage(repo)
	if err != nil {
		return "", err
	}
	a, _ := json.Marshal(actual)
	b, _ := json.Marshal(retained)
	if !bytes.Equal(a, b) {
		return "", errors.New("capture package drift")
	}
	return fileDigest(path)
}

func verifyCommittedPackage(ctx context.Context, repo string) error {
	for _, path := range packageFiles {
		command := exec.CommandContext(ctx, "git", "-C", repo, "show", "HEAD:"+path)
		hash := sha256.New()
		command.Stdout = hash
		if err := command.Run(); err != nil {
			return fmt.Errorf("package file is not committed: %s", path)
		}
		actual, err := fileDigest(filepath.Join(repo, path))
		if err != nil {
			return err
		}
		if actual != fmt.Sprintf("%x", hash.Sum(nil)) {
			return fmt.Errorf("package file differs from committed source: %s", path)
		}
	}
	return nil
}
