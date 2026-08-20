package launcher

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

func TestCentralOptimizeMaterializesChunkedOutputPackExactly(t *testing.T) {
	repository := t.TempDir()
	payload := []byte("verified-unaffected-output\n")
	payloadDigest := sha256.Sum256(payload)
	sha := hex.EncodeToString(payloadDigest[:])
	entry := optimizePortfolioEntry{
		RepositoryID: "example/materialized", FamilySHA256: optimizeDigest("family"),
		TargetRevision:   "0123456789abcdef0123456789abcdef01234567",
		RequiredOutputs:  []string{"changed/build/**", "unchanged/build/**"},
		CandidateOutputs: []string{"changed/build/**"},
		Materialization: &optimizePortfolioMaterialization{
			PackSHA256: sha, PackSize: int64(len(payload)), ChunkSHA256: []string{sha},
			MaterializedProjects: []string{":unchanged"},
		},
	}
	manifest := optimizeOutputMaterializationManifest{
		SchemaVersion: optimizeMaterializationSchema, RepositoryID: entry.RepositoryID,
		TargetRevision: entry.TargetRevision, RequiredOutputs: entry.RequiredOutputs,
		CandidateOutputs: entry.CandidateOutputs,
		PackFile:         ".buildopt/optimize/v1/materialization/payload.pack",
		PackSHA256:       sha, PackSize: int64(len(payload)),
		Entries: []optimizeOutputMaterializationEntry{{
			Path: "unchanged/build/unchanged.jar", SHA256: sha,
			Size: int64(len(payload)), Mode: 0o644,
		}},
	}
	manifestRaw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestDigest := sha256.Sum256(manifestRaw)
	entry.Materialization.ManifestSHA256 = hex.EncodeToString(manifestDigest[:])
	bundleManifest := filepath.ToSlash(filepath.Join(
		"portfolio", "profiles", entry.FamilySHA256, "materialization-manifest.json",
	))
	integration := &centralOptimizeIntegration{portfolio: &centralRemoteSnapshot{
		manifestSHA256: optimizeDigest("portfolio"),
		objects:        map[string][]byte{sha: payload},
		bundle:         centralStateBundle{Kind: sharedcache.StateKindPortfolio},
	}}
	invocation := optimizeInvocation{
		repositoryRoot: repository, stateRelative: optimizeDefaultStateDir,
		connectionRelative: centralConnectionDir,
		discovery:          optimizeDiscoveryContext{TargetRevision: "89abcdef0123456789abcdef0123456789abcdef"},
	}
	materializedRoot := filepath.ToSlash(filepath.Join(
		centralConnectionDir, "materialized", optimizeDigest("target"),
	))
	materialization, err := integration.materializeOutputPack(
		invocation, entry, map[string][]byte{bundleManifest: manifestRaw}, materializedRoot,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !validOptimizeOutputMaterializationShape(materialization, true) {
		t.Fatalf("remote materialization shape = %+v", materialization)
	}
	run := &optimizeRun{
		invocation: invocation,
		centralReplay: &centralOptimizeReplay{discovery: optimizeDiscoveryResult{
			Status: optimizeDiscoveryRemoteRevalidated, RepositoryID: entry.RepositoryID,
			TargetRevision:  invocation.discovery.TargetRevision,
			RequiredOutputs: entry.RequiredOutputs, CandidateOutputs: entry.CandidateOutputs,
			Materialization: materialization,
		}},
	}
	if err := run.materializeCandidateOutputs(); err != nil {
		t.Fatal(err)
	}
	restored, err := os.ReadFile(filepath.Join(repository, "unchanged", "build", "unchanged.jar"))
	if err != nil || string(restored) != string(payload) {
		t.Fatalf("restored output = %q, %v", restored, err)
	}
}

func TestCentralOptimizeInvalidatesWhenMaterializedProducerChanges(t *testing.T) {
	snapshot := buildimpact.DiscoverySnapshot{
		SchemaVersion: buildimpact.DiscoverySchemaVersion,
		Complete:      true,
		Projects: []buildimpact.DiscoveredProject{
			{Path: ":app", SourcePaths: []string{"app/src/**"}},
			{Path: ":packaging", SourcePaths: []string{"packaging/src/**"}},
		},
	}
	materialized := []string{":packaging"}
	if !centralOptimizeMaterializedProducerChanged(
		snapshot, []string{"packaging/src/main/Archive.java"}, materialized,
	) {
		t.Fatal("materialized producer change did not invalidate the profile")
	}
	if centralOptimizeMaterializedProducerChanged(
		snapshot, []string{"app/src/main/App.java", "README.md"}, materialized,
	) {
		t.Fatal("unrelated source and documentation changes invalidated the profile")
	}
}
