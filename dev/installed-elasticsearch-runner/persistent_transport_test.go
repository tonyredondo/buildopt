package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
	"github.com/tonyredondo/buildopt/internal/sharedcache"
	"github.com/tonyredondo/buildopt/internal/wcncpobserve"
	"golang.org/x/sys/unix"
)

// This opt-in measures fresh durable writes under the production deadline.
// It starts no Gradle process and grants no value authority to its local facts.
func TestPersistentObservationTransport(t *testing.T) {
	root := os.Getenv("EIC_PERSISTENT_TRANSPORT_ROOT")
	if root == "" {
		t.Skip("requires a separately allocated persistent transport probe")
	}
	if !filepath.IsAbs(root) {
		t.Fatal("absolute task-owned root required")
	}
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	var fs unix.Statfs_t
	if err := unix.Statfs(root, &fs); err != nil {
		t.Fatal(err)
	}
	if fs.Type != unix.BTRFS_SUPER_MAGIC {
		t.Fatal("campaign Btrfs storage required")
	}
	pkg := os.Getenv("EIC_PERSISTENT_TRANSPORT_PACKAGE")
	pin, err := hashFile(pkg)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(filepath.Join(root, "freeze.json"), map[string]any{"schemaVersion": "buildopt.eic/persistent-transport/v1", "maximumRequests": 20, "deadlineMilliseconds": 100, "maximumSeconds": 120, "maximumBytes": 1 << 28, "gradleStarts": 0, "package": fileBinding{pkg, pin}, "filesystemType": fs.Type, "performanceAuthority": false}); err != nil {
		t.Fatal(err)
	}
	b, err := startCorrectnessBackend(context.Background(), filepath.Join(root, "backend"), fileBinding{pkg, pin})
	if err != nil {
		t.Fatal(err)
	}
	defer b.close()
	var credential struct {
		Token string `json:"token"`
	}
	raw, err := os.ReadFile(b.Credential.Path)
	if err != nil || json.Unmarshal(raw, &credential) != nil {
		t.Fatal("cannot read owned credential")
	}
	endpoint, err := wcncpobserve.EndpointForScope(b.URL, correctnessScopeDigest(), "WCNCP_OBSERVATION/batch")
	if err != nil {
		t.Fatal(err)
	}
	type measured struct {
		Ordinal   int                        `json:"ordinal"`
		ElapsedNS int64                      `json:"elapsedNanoseconds"`
		Fact      fileBinding                `json:"fact"`
		Outcome   wcncpobserve.UploadOutcome `json:"outcome"`
		Persisted bool                       `json:"persistedAfterReopen"`
	}
	results := []measured{}
	for i := 1; i <= 20; i++ {
		code := 0
		facts := wcncpobserve.BuildObservation(correctnessScope, strings.Repeat("a", 64), func(key string) string {
			switch key {
			case "WCNCP_REPOSITORY_REVISION":
				return makeProtocol().Rows[0].Revision
			case "WCNCP_ENVIRONMENT_CLASS":
				return "LOCAL_FUNCTIONAL"
			default:
				return ""
			}
		}, root, nil, []string{}, wcncpobserve.PassthroughResult{Child: wcncpobserve.ChildResult{Outcome: "SUCCESS", ExitCode: &code}})
		if err := facts.Validate(); err != nil {
			t.Fatal(err)
		}
		if facts.Authority.ProspectiveGateInput {
			t.Fatal("local transport facts gained value authority")
		}
		raw, err := json.Marshal(facts)
		if err != nil {
			t.Fatal(err)
		}
		raw, err = contractcrypto.CanonicalizeJCS(raw)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, facts.ObservationID+".json")
		if err := writeNew(path, raw); err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		outcome := wcncpobserve.UploadBatch(context.Background(), b.client, endpoint, credential.Token, []json.RawMessage{raw}, 100*time.Millisecond)
		results = append(results, measured{Ordinal: i, ElapsedNS: time.Since(start).Nanoseconds(), Fact: fileBinding{path, digest(raw)}, Outcome: outcome})
	}
	if err := b.close(); err != nil {
		t.Fatal(err)
	}
	storage, err := sharedcache.Open(context.Background(), filepath.Join(root, "backend/state"))
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	for i := range results {
		f, err := storage.OpenWCNCPObject(context.Background(), correctnessScopeDigest(), sharedcache.StateKind("WCNCP_OBSERVATION"), results[i].Fact.SHA256)
		if err == nil {
			data, e := io.ReadAll(f)
			closeErr := f.Close()
			results[i].Persisted = e == nil && closeErr == nil && digest(data) == results[i].Fact.SHA256
		}
		if results[i].Outcome.Uploaded == 1 && !results[i].Persisted {
			t.Fatal("acknowledged observation lost after storage reopen")
		}
	}
	if err := storage.Close(); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(filepath.Join(root, "results.json"), results); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(filepath.Join(root, "closed.json"), map[string]any{"backendClosed": true, "reopenedStorageClosed": true, "requests": 20, "gradleStarts": 0}); err != nil {
		t.Fatal(err)
	}
}
