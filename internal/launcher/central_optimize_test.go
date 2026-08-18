package launcher

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

func TestCentralOptimizeReusesQualifiedProfileAcrossSourceCommitAndRejectsStructuralDrift(t *testing.T) {
	const (
		repositoryID     = "apache/kafka"
		evidenceRevision = "929868cbdee1fbdc9cb60701e2ca17e8a66cd2ae"
		changedPath      = "clients/src/main/java/org/apache/kafka/clients/Metadata.java"
	)
	fixture := filepath.Join(
		"..", "..", "benchmarks", "results", "poc-generic-change-breadth-v1",
		"apache-kafka-shadow-clients-source", "capture-1",
	)
	fixtureFiles := map[string][]byte{}
	for _, name := range []string{
		"profile.json", "buildopt-impact-manifest.json", "buildopt-impact-graph.generated.json",
		"buildopt-impact.generated.json", "buildopt-changes.txt", "evidence.json",
	} {
		raw, err := os.ReadFile(filepath.Join(fixture, name))
		if err != nil {
			t.Fatal(err)
		}
		fixtureFiles[name] = raw
	}
	var profile qualifiedPOCProfile
	if err := json.Unmarshal(fixtureFiles["profile.json"], &profile); err != nil {
		t.Fatal(err)
	}
	// Automatic optimize portfolios carry the three structural inputs. The
	// public benchmark profile adds output-equivalence as a fourth external
	// precondition, so trim only that independent publication binding here.
	profile.Preconditions = append([]qualifiedPOCPrecondition(nil), profile.Preconditions[:3]...)
	fixtureFiles["profile.json"], _ = json.Marshal(profile)

	repository := t.TempDir()
	centralOptimizeGit(t, repository, "init", "-q")
	centralOptimizeGit(t, repository, "config", "user.email", "buildopt@example.invalid")
	centralOptimizeGit(t, repository, "config", "user.name", "BuildOpt fixture")
	centralOptimizeGit(t, repository, "config", "commit.gpgsign", "false")
	writeGradleWrapperProperties(t, repository, "distributionUrl=https\\://services.gradle.org/distributions/gradle-9.6.1-bin.zip\n")
	writeCentralOptimizeFile(t, repository, changedPath, "class Metadata {}\n")
	centralOptimizeGit(t, repository, "add", ".")
	centralOptimizeGit(t, repository, "commit", "-qm", "qualified base")
	baseRevision := strings.TrimSpace(centralOptimizeGit(t, repository, "rev-parse", "HEAD"))
	centralOptimizeGit(t, repository, "update-ref", "refs/replace/"+evidenceRevision, baseRevision)

	writeCentralOptimizeFile(t, repository, changedPath, "class Metadata { int revision = 2; }\n")
	centralOptimizeGit(t, repository, "add", changedPath)
	tree := strings.TrimSpace(centralOptimizeGit(t, repository, "write-tree"))
	currentRevision := strings.TrimSpace(centralOptimizeGitInput(t, repository, "source change\n", "commit-tree", tree, "-p", evidenceRevision))
	centralOptimizeGit(t, repository, "update-ref", "refs/heads/main", currentRevision)
	centralOptimizeGit(t, repository, "reset", "-q", "--hard", currentRevision)

	eventPath := filepath.Join(repository, ".buildopt", "github-event.json")
	eventRaw, _ := json.Marshal(map[string]string{"before": evidenceRevision})
	if err := os.MkdirAll(filepath.Dir(eventPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(eventPath, eventRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_EVENT_PATH", eventPath)
	t.Setenv("GITHUB_REPOSITORY", repositoryID)
	t.Chdir(repository)
	invocation, err := prepareOptimizeInvocation(append([]string{
		"--calibration-pairs", "8", "--",
	}, append(append([]string(nil), profile.GradleOptions...), "shadowJar")...), true)
	if err != nil {
		t.Fatal(err)
	}
	if !invocation.discovery.Ready || invocation.discovery.TargetRevision != currentRevision ||
		!equalOptimizeStrings(invocation.discovery.changedPaths, []string{changedPath}) {
		t.Fatalf("current discovery was not bound to the source-only commit: %+v", invocation.discovery)
	}

	integration := centralOptimizeFixtureIntegration(t, invocation, fixtureFiles, profile, evidenceRevision)
	run, err := beginOptimizeRun(invocation)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := integration.materializeReplay(run); err != nil {
		t.Fatalf("materialize compatible remote profile: %v", err)
	}
	impact := integration.prepareAutomaticReplay(run)
	if impact == nil || run.centralReplay == nil || !run.selection.Selected ||
		run.selection.Source != optimizeSelectionSourceCentral ||
		run.selection.EvidenceRevision != evidenceRevision ||
		run.selection.RevalidatedRevision != currentRevision ||
		!impact.plan.CandidateSelected || !equalOptimizeStrings(impact.plan.Entrypoints, []string{":clients:shadowJar"}) ||
		integration.result.NativeFallback {
		t.Fatalf("remote profile was not selected before Gradle: impact=%+v selection=%+v central=%+v", impact, run.selection, integration.result)
	}
	if run.centralReplay.calibration.Status != optimizeCalibrationRemoteQualified ||
		run.centralReplay.calibration.MeanSavedMS != 30106.5 ||
		run.centralReplay.discovery.Status != optimizeDiscoveryRemoteRevalidated {
		t.Fatalf("remote evidence was not preserved and revalidated: %+v / %+v", run.centralReplay.calibration, run.centralReplay.discovery)
	}
	selectedProfileSHA, err := optimizeFileSHA256(filepath.Join(repository, filepath.FromSlash(run.selection.ProfileFile)), false)
	if err != nil || selectedProfileSHA != run.selection.ProfileSHA256 {
		t.Fatalf("materialized profile binding = %s/%v, want %s", selectedProfileSHA, err, run.selection.ProfileSHA256)
	}
	run.central = integration
	run.childStarted = true
	if err := run.finish(0, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !validOptimizeState(run.state) {
		t.Fatalf("completed remote replay state is invalid: %+v", run.state)
	}
	var completed optimizeResult
	completedRaw, err := os.ReadFile(run.resultPath)
	if err != nil || json.Unmarshal(completedRaw, &completed) != nil ||
		completed.Execution.Mode != "SELECTIVE_PROFILE" ||
		completed.Central.Status != optimizeCentralSelected || completed.Central.NativeFallback {
		t.Fatalf("completed remote result did not preserve central selection: err=%v result=%+v", err, completed)
	}

	writeCentralOptimizeFile(t, repository, "build.gradle.kts", "plugins { base }\n")
	centralOptimizeGit(t, repository, "add", "build.gradle.kts")
	centralOptimizeGit(t, repository, "commit", "-qm", "structural drift")
	driftRevision := strings.TrimSpace(centralOptimizeGit(t, repository, "rev-parse", "HEAD"))
	invocation, err = prepareOptimizeInvocation(append([]string{
		"--calibration-pairs", "8", "--",
	}, append(append([]string(nil), profile.GradleOptions...), "shadowJar")...), true)
	if err != nil {
		t.Fatal(err)
	}
	if invocation.discovery.TargetRevision != driftRevision {
		t.Fatalf("drift target revision = %s, want %s", invocation.discovery.TargetRevision, driftRevision)
	}
	integration = centralOptimizeFixtureIntegration(t, invocation, fixtureFiles, profile, evidenceRevision)
	run, err = beginOptimizeRun(invocation)
	if err != nil {
		t.Fatal(err)
	}
	if impact = integration.prepareAutomaticReplay(run); impact != nil || run.selection.Selected ||
		!integration.result.NativeFallback || integration.result.Reason != optimizeCentralReasonStructural {
		t.Fatalf("build-logic drift did not retain native Gradle: impact=%+v selection=%+v central=%+v", impact, run.selection, integration.result)
	}
}

func TestCentralOptimizeAutomaticallyPublishesAndUsesVerifiedOfflineSnapshots(t *testing.T) {
	const repositoryID = "example/central-optimize"
	now := time.Now().UTC().Truncate(time.Second)
	repositoryScope := optimizePortfolioRepositoryScope(repositoryID)
	storage, err := sharedcache.Open(context.Background(), filepath.Join(t.TempDir(), "server-state"))
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	issued, err := storage.IssueCentralToken(
		context.Background(),
		sharedcache.CentralTokenIssueRequest{
			Scope: sharedcache.CentralTokenScope{
				RepositoryScopeSHA256: repositoryScope,
				Tenant:                "owner-poc", Repository: repositoryID, TrustDomain: "owner-poc",
				Namespace: "gradle-9.6.1/linux-amd64/jdk-21/project", NamespaceGeneration: 1,
			},
			Capabilities: []sharedcache.CentralCapability{
				sharedcache.CentralStateRead, sharedcache.CentralStateWrite,
			},
			ExpiresAt: now.Add(time.Hour),
		},
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := sharedcache.NewCentralHTTPSHandler(storage)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(handler)
	server.TLS = &tls.Config{MinVersion: tls.VersionTLS13}
	server.StartTLS()
	certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	if len(certificate) == 0 {
		t.Fatal("test server certificate was not encoded")
	}

	newRepository := func(withState bool, variant string, updatedAt time.Time) (string, string, string) {
		t.Helper()
		repository := t.TempDir()
		writeGradleWrapperProperties(t, repository, "distributionUrl=gradle-9.6.1-bin.zip\n")
		if withState {
			writeCentralSyncLocalState(t, repository, repositoryID, updatedAt, variant)
		}
		tokenPath := filepath.Join(repository, "central-token.json")
		writeCentralSyncTokenDocument(t, tokenPath, issued)
		caPath := filepath.Join(repository, "central-ca.pem")
		if err := os.WriteFile(caPath, certificate, 0o600); err != nil {
			t.Fatal(err)
		}
		return repository, tokenPath, caPath
	}
	invocationFor := func(repository string) optimizeInvocation {
		return optimizeInvocation{
			repositoryRoot:      repository,
			stateDirectory:      filepath.Join(repository, filepath.FromSlash(optimizeDefaultStateDir)),
			stateRelative:       optimizeDefaultStateDir,
			connectionDirectory: filepath.Join(repository, filepath.FromSlash(centralConnectionDir)),
			connectionRelative:  centralConnectionDir,
			discovery:           optimizeDiscoveryContext{RepositoryID: repositoryID},
		}
	}

	t.Setenv("GITHUB_REPOSITORY", repositoryID)
	producer, producerToken, producerCA := newRepository(true, "first", now.Add(-time.Hour))
	if result, code := runCentralSyncCommand(t, producer, []string{
		"connect", server.URL, "--token-file", producerToken, "--ca-file", producerCA,
	}); code != 0 || !allCentralKindStatuses(result, "PUSHED") {
		t.Fatalf("initial publication = code %d, %+v", code, result)
	}
	consumer, consumerToken, consumerCA := newRepository(false, "unused", now)
	if result, code := runCentralSyncCommand(t, consumer, []string{
		"connect", server.URL, "--token-file", consumerToken, "--ca-file", consumerCA,
	}); code != 0 || !allCentralKindStatuses(result, "PULLED") {
		t.Fatalf("initial lookup = code %d, %+v", code, result)
	}

	producerInvocation := invocationFor(producer)
	var diagnostics bytes.Buffer
	integration := prepareCentralOptimizeIntegration(producerInvocation, &diagnostics)
	if integration.client == nil || !integration.result.PreSyncOnline || !integration.result.SnapshotsVerified {
		t.Fatalf("automatic pre-sync did not load verified state: %+v; %s", integration.result, diagnostics.String())
	}
	writeCentralSyncLocalState(t, producer, repositoryID, now, "second")
	integration.publish(&optimizeRun{invocation: producerInvocation}, &diagnostics)
	if integration.result.PostSyncStatus != optimizeCentralPublished || !integration.result.PostSyncOnline {
		t.Fatalf("automatic post-sync did not publish the new state: %+v; %s", integration.result, diagnostics.String())
	}
	if result, code := runCentralSyncCommand(t, consumer, []string{"sync"}); code != 0 ||
		!hasCentralKindStatus(result, "PULLED") {
		t.Fatalf("consumer did not observe automatic publication = code %d, %+v", code, result)
	}
	assertCentralSnapshotsEqual(t, producer, consumer)

	server.Close()
	diagnostics.Reset()
	offline := prepareCentralOptimizeIntegration(invocationFor(consumer), &diagnostics)
	if offline.result.PreSyncOnline || !offline.result.SnapshotsVerified ||
		offline.result.Status != optimizeCentralAvailable || offline.portfolio == nil || offline.evidence == nil {
		t.Fatalf("verified offline snapshots were not available to optimize: %+v; %s", offline.result, diagnostics.String())
	}
}

func centralOptimizeFixtureIntegration(
	t *testing.T,
	invocation optimizeInvocation,
	fixtureFiles map[string][]byte,
	profile qualifiedPOCProfile,
	evidenceRevision string,
) *centralOptimizeIntegration {
	t.Helper()
	manifest, err := buildimpact.LoadRepositoryManifest(
		invocation.repositoryRoot, centralOptimizeStageFixtureFile(t, invocation.repositoryRoot, ".buildopt/fixture-manifest.json", fixtureFiles["buildopt-impact-manifest.json"]),
		profile.RepositoryID, profile.PipelineClass,
	)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := buildimpact.ParseDeclaredGraph(fixtureFiles["buildopt-impact-graph.generated.json"], manifest)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := centralOptimizeDiscoverySnapshot(graph.Graph)
	evidenceChanges := strings.Fields(string(fixtureFiles["buildopt-changes.txt"]))
	owners, err := buildimpact.ResolveProjectOwners(snapshot, evidenceChanges)
	if err != nil {
		t.Fatal(err)
	}
	family := optimizeChangeFamily(snapshot, evidenceChanges, owners)
	entrypoints := []string{"shadowJar"}
	candidate := []string{":clients:shadowJar"}
	outputs := []string{"clients/build/libs/kafka-clients-*.jar"}
	familySHA := optimizePortfolioFamilyDigest(profile.RepositoryID, family, owners, entrypoints, candidate, outputs)
	directory := filepath.ToSlash(filepath.Join(optimizeDefaultStateDir, "portfolio", "profiles", familySHA))
	pathFor := func(name string) string { return filepath.ToSlash(filepath.Join(directory, name)) }
	bundlePathFor := func(name string) string {
		return filepath.ToSlash(filepath.Join("portfolio", "profiles", familySHA, name))
	}
	digest := func(raw []byte) string {
		sum := sha256.Sum256(raw)
		return hex.EncodeToString(sum[:])
	}
	entry := optimizePortfolioEntry{
		Family: family, FamilySHA256: familySHA, ChangedProjects: owners,
		RepositoryID: profile.RepositoryID, Entrypoints: entrypoints, CandidateEntrypoints: candidate,
		RequiredOutputs: outputs, TargetRevision: evidenceRevision,
		WrapperSHA256: invocation.wrapperSHA256, ExecutableSHA256: invocation.executableSHA256,
		ManifestSHA256:  digest(fixtureFiles["buildopt-impact-manifest.json"]),
		GraphSHA256:     digest(fixtureFiles["buildopt-impact-graph.generated.json"]),
		GeneratedSHA256: digest(fixtureFiles["buildopt-impact.generated.json"]),
		EvidenceSHA256:  digest(fixtureFiles["evidence.json"]),
		ProfileSHA256:   digest(fixtureFiles["profile.json"]), ProfilePath: pathFor("profile.json"),
		State: "QUALIFIED",
	}
	portfolio := optimizeProfilePortfolio{
		SchemaVersion: optimizePortfolioSchemaVersion, Generation: 1,
		RepositoryScopeSHA256: optimizePortfolioRepositoryScope(profile.RepositoryID),
		Profiles:              []optimizePortfolioEntry{entry}, UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	portfolioRaw, err := json.Marshal(portfolio)
	if err != nil {
		t.Fatal(err)
	}
	portfolioFiles := map[string][]byte{
		filepath.ToSlash(filepath.Join("portfolio", optimizePortfolioIndexFile)): portfolioRaw,
		bundlePathFor("profile.json"):                                            fixtureFiles["profile.json"],
		bundlePathFor("manifest.json"):                                           fixtureFiles["buildopt-impact-manifest.json"],
		bundlePathFor("graph.json"):                                              fixtureFiles["buildopt-impact-graph.generated.json"],
		bundlePathFor("generated-manifest.json"):                                 fixtureFiles["buildopt-impact.generated.json"],
		bundlePathFor("evidence.json"):                                           fixtureFiles["evidence.json"],
	}
	evidenceFiles := map[string][]byte{bundlePathFor("evidence.json"): fixtureFiles["evidence.json"]}
	portfolioManifest := optimizeDigest("central-optimize-test-portfolio-v1", invocation.discovery.TargetRevision)
	evidenceManifest := optimizeDigest("central-optimize-test-evidence-v1", evidenceRevision)
	return &centralOptimizeIntegration{
		invocation: invocation, startedAt: time.Now(),
		connection: centralConnection{
			RepositoryID:          profile.RepositoryID,
			RepositoryScopeSHA256: optimizePortfolioRepositoryScope(profile.RepositoryID),
		},
		result: optimizeCentralResult{
			SchemaVersion: optimizeCentralSchemaVersion, Status: optimizeCentralAvailable,
			Reason: "VERIFIED_REMOTE_STATE_AVAILABLE", Connected: true, SnapshotsVerified: true,
			PortfolioManifestSHA256: portfolioManifest, EvidenceManifestSHA256: evidenceManifest,
			ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
		},
		portfolio: &centralRemoteSnapshot{manifestSHA256: portfolioManifest, bundle: centralOptimizeTestBundle(sharedcache.StateKindPortfolio, portfolioFiles)},
		evidence:  &centralRemoteSnapshot{manifestSHA256: evidenceManifest, bundle: centralOptimizeTestBundle(sharedcache.StateKindEvidence, evidenceFiles)},
	}
}

func centralOptimizeTestBundle(kind sharedcache.StateKind, files map[string][]byte) centralStateBundle {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	bundle := centralStateBundle{Kind: kind, Files: make([]centralStateBundleFile, 0, len(paths))}
	for _, path := range paths {
		raw := files[path]
		sum := sha256.Sum256(raw)
		bundle.Files = append(bundle.Files, centralStateBundleFile{
			Path: path, SHA256: hex.EncodeToString(sum[:]), SizeBytes: int64(len(raw)),
			ContentBase64: base64.RawStdEncoding.EncodeToString(raw),
		})
	}
	return bundle
}

func centralOptimizeStageFixtureFile(t *testing.T, root, relative string, raw []byte) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, relative), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return relative
}

func writeCentralOptimizeFile(t *testing.T, root, relative, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func centralOptimizeGit(t *testing.T, root string, arguments ...string) string {
	t.Helper()
	return centralOptimizeGitInput(t, root, "", arguments...)
}

func centralOptimizeGitInput(t *testing.T, root, stdin string, arguments ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, arguments...)...)
	command.Stdin = strings.NewReader(stdin)
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Run(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(arguments, " "), err, output.String())
	}
	return output.String()
}
