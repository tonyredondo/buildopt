package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	command.Env = append(os.Environ(), "GIT_AUTHOR_NAME=CNC Fixture", "GIT_AUTHOR_EMAIL=fixture@example.invalid", "GIT_COMMITTER_NAME=CNC Fixture", "GIT_COMMITTER_EMAIL=fixture@example.invalid")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("fixture git %v: %v %s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

func TestRegisteredSharedWorktreeAndSourceDrift(t *testing.T) {
	root := t.TempDir()
	repository := filepath.Join(root, "repository")
	if err := os.Mkdir(repository, 0700); err != nil {
		t.Fatal(err)
	}
	fixtureGit(t, repository, "init", "--quiet")
	content := []byte("fixture source\n")
	file := filepath.Join(repository, "input.txt")
	if err := os.WriteFile(file, content, 0600); err != nil {
		t.Fatal(err)
	}
	fixtureGit(t, repository, "add", "input.txt")
	tree := fixtureGit(t, repository, "write-tree")
	revision := fixtureGit(t, repository, "commit-tree", tree, "-m", "isolated fixture identity")
	// Synthetic fixture plumbing never changes BuildOpt's index/commits/refs.
	fixtureGit(t, repository, "update-ref", "refs/heads/fixture", revision)
	worktree := filepath.Join(root, "worktree")
	fixtureGit(t, repository, "worktree", "add", "--detach", worktree, revision)
	archive := exec.Command("git", "-C", worktree, "archive", "--format=tar", revision)
	output, err := archive.Output()
	if err != nil {
		t.Fatal(err)
	}
	binding := sourceBinding{Revision: revision, ArchiveSHA256: fmt.Sprintf("%x", sha256.Sum256(output)), Files: []fileBinding{{"input.txt", fmt.Sprintf("%x", sha256.Sum256(content))}}}
	common := filepath.Join(repository, ".git")
	if err := verifyWorktree(context.Background(), worktree, common, binding); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(worktree, ".git"))
	if err != nil || !info.Mode().IsRegular() {
		t.Fatal("fixture did not use a Git-file worktree")
	}
	if err := verifyWorktree(context.Background(), worktree, filepath.Join(root, "wrong"), binding); err == nil {
		t.Fatal("wrong shared Git identity accepted")
	}
	changed := binding
	changed.ArchiveSHA256 = strings.Repeat("0", 64)
	if err := verifyWorktree(context.Background(), worktree, common, changed); err == nil {
		t.Fatal("archive drift accepted")
	}
	changed = binding
	changed.Revision = strings.Repeat("0", 40)
	if err := verifyWorktree(context.Background(), worktree, common, changed); err == nil {
		t.Fatal("revision drift accepted")
	}
	if err := os.WriteFile(filepath.Join(worktree, "input.txt"), []byte("drift"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyWorktree(context.Background(), worktree, common, binding); err == nil {
		t.Fatal("dirty source accepted")
	}
	// t.TempDir owns the entire synthetic repository and its registered worktree;
	// no external worktree is removed or pruned by this test.
}

func runtimeFixture(t *testing.T) (string, string, runtimeBinding) {
	t.Helper()
	root := t.TempDir()
	installed := filepath.Join(root, "jdk")
	if err := os.MkdirAll(filepath.Join(installed, "bin"), 0700); err != nil {
		t.Fatal(err)
	}
	var compressed bytes.Buffer
	zipped := gzip.NewWriter(&compressed)
	writer := tar.NewWriter(zipped)
	for _, dir := range []string{"fixture/", "fixture/bin/"} {
		if err := writer.WriteHeader(&tar.Header{Name: dir, Mode: 0755, Typeflag: tar.TypeDir}); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range []struct {
		path, body string
		mode       int64
	}{{"bin/java", "verified fixture executable\n", 0755}, {"release", "JAVA_VERSION=fixture\n", 0644}} {
		data := []byte(file.body)
		if err := writer.WriteHeader(&tar.Header{Name: "fixture/" + file.path, Mode: file.mode, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(data); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(installed, file.path), data, os.FileMode(file.mode)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zipped.Close(); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(root, "runtime.tar.gz")
	if err := os.WriteFile(archive, compressed.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return archive, installed, runtimeBinding{Version: "fixture", SHA256: fmt.Sprintf("%x", sha256.Sum256(compressed.Bytes()))}
}

func TestRuntimeArchiveAndBinaryDrift(t *testing.T) {
	archive, root, binding := runtimeFixture(t)
	if err := verifyRuntime(archive, root, binding); err != nil {
		t.Fatal(err)
	}
	wrong := binding
	wrong.SHA256 = strings.Repeat("0", 64)
	if err := verifyRuntime(archive, root, wrong); err == nil {
		t.Fatal("archive drift accepted")
	}
	if err := os.WriteFile(filepath.Join(root, "bin/java"), []byte("same release, different executable"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := verifyRuntime(archive, root, binding); err == nil {
		t.Fatal("binary drift accepted")
	}
	archive, root, binding = runtimeFixture(t)
	if err := os.WriteFile(filepath.Join(root, "injected"), []byte("extra"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyRuntime(archive, root, binding); err == nil {
		t.Fatal("unexpected runtime member accepted")
	}
}

func TestPackageFreezeAndDrift(t *testing.T) {
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := makePackage(repo)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "package.json")
	if err := writeNewJSON(path, manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyPackage(repo, path); err != nil {
		t.Fatal(err)
	}
	if err := writeNewJSON(path, manifest); err == nil {
		t.Fatal("occupied package replaced")
	}
	manifest.Files[0].SHA256 = strings.Repeat("0", 64)
	bad := filepath.Join(filepath.Dir(path), "changed.json")
	if err := writeNewJSON(bad, manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyPackage(repo, bad); err == nil {
		t.Fatal("package drift accepted")
	}
}
