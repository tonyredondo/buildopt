package main

import (
	"context"
	"github.com/tonyredondo/buildopt/internal/stickywrapper"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestCorrectnessBackendUsesScopedCredentialsAndBoundPackage(t *testing.T) {
	root := t.TempDir()
	pkg := filepath.Join(root, "package.tar.gz")
	if err := os.WriteFile(pkg, []byte("frozen package bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	binding := fileBinding{pkg, digest([]byte("frozen package bytes"))}
	backend, err := startCorrectnessBackend(context.Background(), filepath.Join(root, "backend"), binding)
	if err != nil {
		t.Fatal(err)
	}
	defer backend.close()
	response, err := backend.client.Get(backend.URL + "/0.0.1/linux-amd64")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || string(raw) != "frozen package bytes" {
		t.Fatal("package download not bound")
	}
	work := filepath.Join(root, "work")
	if err := os.Mkdir(work, 0700); err != nil {
		t.Fatal(err)
	}
	files, err := backend.install(context.Background(), work)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 4 {
		t.Fatal("installed wrapper file set differs", len(files))
	}
	snapshot, err := (stickywrapper.Generator{Root: work}).Check()
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Config.ServerURL != backend.URL || snapshot.Config.ProjectScope != "elastic/elasticsearch" || snapshot.Config.Mode != "observe" {
		t.Fatal("installed scope drift", snapshot.Config)
	}
	if snapshot.Release.Distributions["linux-amd64"].SHA256 != binding.SHA256 {
		t.Fatal("installed package digest drift")
	}
	if err := checkBinding(backend.Credential); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(backend.Credential.Path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("ephemeral credential is not private")
	}
	if err := os.WriteFile(pkg, []byte("drift"), 0600); err != nil {
		t.Fatal(err)
	}
	response, err = backend.client.Get(backend.URL + "/0.0.1/linux-amd64")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 409 {
		t.Fatal("package drift served")
	}
}

func TestCorrectnessBackendRefusesNonLoopbackListener(t *testing.T) {
	root := filepath.Join(t.TempDir(), "backend")
	if _, err := startCorrectnessBackendAt(context.Background(), root, fileBinding{}, "0.0.0.0:0"); err == nil {
		t.Fatal("non-loopback listener accepted")
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("refused endpoint created state")
	}
}
