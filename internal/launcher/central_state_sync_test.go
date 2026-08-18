package launcher

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

func TestCentralStateSyncConnectLifecycleConcurrencyAndOfflineSnapshot(t *testing.T) {
	const repositoryID = "example/central-state-sync"
	now := time.Now().UTC().Truncate(time.Second)
	repositoryScope := optimizePortfolioRepositoryScope(repositoryID)
	storage, err := sharedcache.Open(context.Background(), filepath.Join(t.TempDir(), "server-state"))
	if err != nil {
		t.Fatal(err)
	}
	readOnlyIssued, err := storage.IssueCentralToken(
		context.Background(),
		sharedcache.CentralTokenIssueRequest{
			Scope: sharedcache.CentralTokenScope{
				RepositoryScopeSHA256: repositoryScope,
				Tenant:                "owner-poc", Repository: repositoryID,
				TrustDomain: "owner-poc", Namespace: "gradle-9.6.1/linux-amd64/jdk-21/project",
				NamespaceGeneration: 1,
			},
			Capabilities: []sharedcache.CentralCapability{sharedcache.CentralStateRead},
			ExpiresAt:    now.Add(time.Hour),
		},
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	issued, err := storage.IssueCentralToken(
		context.Background(),
		sharedcache.CentralTokenIssueRequest{
			Scope: sharedcache.CentralTokenScope{
				RepositoryScopeSHA256: repositoryScope,
				Tenant:                "owner-poc", Repository: repositoryID,
				TrustDomain: "owner-poc", Namespace: "gradle-9.6.1/linux-amd64/jdk-21/project",
				NamespaceGeneration: 1,
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
	controlled := &centralSyncControlledHandler{next: handler}
	server := httptest.NewUnstartedServer(controlled)
	server.TLS = &tls.Config{MinVersion: tls.VersionTLS13}
	server.StartTLS()
	defer server.Close()
	certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	if len(certificate) == 0 {
		t.Fatal("test server certificate was not encoded")
	}

	newRepository := func(updatedAt time.Time, variant string, withState bool) (string, string, string) {
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

	t.Setenv("GITHUB_REPOSITORY", repositoryID)
	producer, producerToken, producerCA := newRepository(now.Add(-3*time.Hour), "producer-one", true)
	producerResult, code := runCentralSyncCommand(t, producer, []string{
		"connect", server.URL, "--token-file", producerToken, "--ca-file", producerCA,
	})
	if code != 0 || producerResult.LocalStateStatus != "VALID" ||
		!allCentralKindStatuses(producerResult, "PUSHED") {
		t.Fatalf("first producer sync = code %d, %+v", code, producerResult)
	}
	producerResult, code = runCentralSyncCommand(t, producer, []string{"sync"})
	if code != 0 || !allCentralKindStatuses(producerResult, "NO_CHANGE") {
		t.Fatalf("no-change sync = code %d, %+v", code, producerResult)
	}

	consumer, consumerToken, consumerCA := newRepository(now, "unused", false)
	writeCentralSyncTokenDocument(t, consumerToken, readOnlyIssued)
	consumerResult, code := runCentralSyncCommand(t, consumer, []string{
		"connect", server.URL, "--token-file", consumerToken, "--ca-file", consumerCA,
	})
	if code != 0 || consumerResult.LocalStateStatus != "ABSENT" ||
		!allCentralKindStatuses(consumerResult, "PULLED") {
		t.Fatalf("clean consumer sync = code %d, %+v", code, consumerResult)
	}
	assertCentralSnapshotsEqual(t, producer, consumer)

	incompatible, incompatibleToken, incompatibleCA := newRepository(now.Add(-4*time.Hour), "older-incompatible", true)
	incompatibleResult, code := runCentralSyncCommand(t, incompatible, []string{
		"connect", server.URL, "--token-file", incompatibleToken, "--ca-file", incompatibleCA,
	})
	if code != 0 || !incompatibleResult.NativeFallback ||
		!hasCentralKindStatus(incompatibleResult, "INCOMPATIBLE_REMOTE_RETAINED") {
		t.Fatalf("incompatible fallback = code %d, %+v", code, incompatibleResult)
	}

	interrupted, interruptedToken, interruptedCA := newRepository(now.Add(-2*time.Hour), "interrupted", true)
	controlled.failNextCAS()
	interruptedResult, code := runCentralSyncCommand(t, interrupted, []string{
		"connect", server.URL, "--token-file", interruptedToken, "--ca-file", interruptedCA,
	})
	if code == 0 || interruptedResult.Online ||
		centralKindStatus(interruptedResult, sharedcache.StateKindEvidence) != "OFFLINE_NO_SNAPSHOT" {
		t.Fatalf("interrupted publication = code %d, %+v", code, interruptedResult)
	}
	interruptedResult, code = runCentralSyncCommand(t, interrupted, []string{"sync"})
	if code != 0 || !allCentralKindStatuses(interruptedResult, "PUSHED") {
		t.Fatalf("exact interrupted resume = code %d, %+v", code, interruptedResult)
	}

	left, _, _ := newRepository(now.Add(-time.Hour), "concurrent-left", true)
	right, _, _ := newRepository(now.Add(-time.Hour), "concurrent-right", true)
	for _, repository := range []string{left, right} {
		connectionDirectory := filepath.Join(repository, filepath.FromSlash(centralConnectionDir))
		if err := ensurePrivateDirectory(connectionDirectory, true); err != nil {
			t.Fatal(err)
		}
	}
	clientFor := func() *centralStateClient {
		t.Helper()
		client, err := newCentralStateClient(server.URL, issued.Token, certificate)
		if err != nil {
			t.Fatal(err)
		}
		return client
	}
	connection := centralConnection{
		SchemaVersion: centralConnectionSchema, ServerURL: server.URL,
		RepositoryID: repositoryID, RepositoryScopeSHA256: repositoryScope,
		StateDirectory: optimizeDefaultStateDir, TokenFile: centralTokenFile,
		ConnectedAt: now.Format(time.RFC3339Nano), TestOptimization: "OUT_OF_SCOPE",
	}
	controlled.barrierNextEvidenceObject()
	type concurrentResult struct {
		result centralSyncResult
		err    error
	}
	results := make(chan concurrentResult, 2)
	for _, repository := range []string{left, right} {
		repository := repository
		go func() {
			result, err := synchronizeCentralState(
				context.Background(), "SYNC", repository,
				filepath.Join(repository, filepath.FromSlash(optimizeDefaultStateDir)),
				filepath.Join(repository, filepath.FromSlash(centralConnectionDir)),
				connection, clientFor(),
			)
			results <- concurrentResult{result: result, err: err}
		}()
	}
	concurrent := []concurrentResult{<-results, <-results}
	concurrentWinnerObserved := false
	for _, item := range concurrent {
		if item.err != nil {
			t.Fatalf("concurrent sync failed: %v, %+v", item.err, item.result)
		}
		if hasCentralKindStatus(item.result, "CONCURRENT_REMOTE_WON") {
			concurrentWinnerObserved = true
		}
	}
	if !concurrentWinnerObserved {
		t.Fatalf("concurrent CAS did not expose one remote winner: %+v", concurrent)
	}

	server.Close()
	consumerResult, code = runCentralSyncCommand(t, consumer, []string{"sync"})
	if code != 0 || consumerResult.Online || !consumerResult.UsedOfflineSnapshot ||
		!allCentralKindStatuses(consumerResult, "OFFLINE_SNAPSHOT") {
		t.Fatalf("verified offline snapshot = code %d, %+v", code, consumerResult)
	}
	tampered := filepath.Join(
		consumer, filepath.FromSlash(centralConnectionDir), centralSnapshotDir,
		"evidence", "bundle.json",
	)
	if err := os.WriteFile(tampered, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	consumerResult, code = runCentralSyncCommand(t, consumer, []string{"sync"})
	if code != 0 || !consumerResult.UsedOfflineSnapshot ||
		centralKindStatus(consumerResult, sharedcache.StateKindEvidence) != "OFFLINE_NO_SNAPSHOT" ||
		centralKindStatus(consumerResult, sharedcache.StateKindPortfolio) != "OFFLINE_NO_SNAPSHOT" ||
		centralKindStatus(consumerResult, sharedcache.StateKindCheckpoint) != "OFFLINE_SNAPSHOT" {
		t.Fatalf("corrupt offline evidence was accepted = code %d, %+v", code, consumerResult)
	}
}

func TestCentralStateSyncRejectsUnsafeConnectionInputs(t *testing.T) {
	repository := t.TempDir()
	t.Chdir(repository)
	t.Setenv("GITHUB_REPOSITORY", "example/repository")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"connect", "http://central.example", "--token-file", "token"}, strings.NewReader(""), &stdout, &stderr); code != exitConfiguration ||
		!strings.Contains(stderr.String(), "canonical HTTPS origin") {
		t.Fatalf("plaintext connect = %d/%q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"sync"}, strings.NewReader(""), &stdout, &stderr); code != exitConfiguration ||
		!strings.Contains(stderr.String(), "central") {
		t.Fatalf("sync without connection = %d/%q", code, stderr.String())
	}
}

type centralSyncControlledHandler struct {
	next http.Handler

	mutex                 sync.Mutex
	failCAS               int
	barrierEvidenceObject bool
	barrierArrivals       int
	barrierRelease        chan struct{}
}

func (handler *centralSyncControlledHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	isCAS := strings.HasSuffix(request.URL.Path, "/head:cas")
	isEvidence := strings.Contains(request.URL.Path, "/state/evidence/")
	isObjectPut := request.Method == http.MethodPut && strings.Contains(request.URL.Path, "/objects/")
	handler.mutex.Lock()
	if isCAS && handler.failCAS > 0 {
		handler.failCAS--
		handler.mutex.Unlock()
		response.Header().Set("Cache-Control", "no-store")
		response.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	var release chan struct{}
	if isObjectPut && isEvidence && handler.barrierEvidenceObject {
		handler.barrierArrivals++
		release = handler.barrierRelease
		if handler.barrierArrivals == 2 {
			handler.barrierEvidenceObject = false
			close(release)
		}
	}
	handler.mutex.Unlock()
	if release != nil {
		<-release
	}
	handler.next.ServeHTTP(response, request)
}

func (handler *centralSyncControlledHandler) failNextCAS() {
	handler.mutex.Lock()
	defer handler.mutex.Unlock()
	handler.failCAS = 20
}

func (handler *centralSyncControlledHandler) barrierNextEvidenceObject() {
	handler.mutex.Lock()
	defer handler.mutex.Unlock()
	handler.barrierEvidenceObject = true
	handler.barrierArrivals = 0
	handler.barrierRelease = make(chan struct{})
}

func runCentralSyncCommand(t *testing.T, repository string, arguments []string) (centralSyncResult, int) {
	t.Helper()
	t.Chdir(repository)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(arguments, strings.NewReader(""), &stdout, &stderr)
	var result centralSyncResult
	if stdout.Len() > 0 {
		if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &result); err != nil {
			t.Fatalf("decode central sync result: %v; stdout=%q stderr=%q", err, stdout.String(), stderr.String())
		}
	}
	if code != 0 {
		t.Logf("central sync stderr: %s", strings.TrimSpace(stderr.String()))
	}
	return result, code
}

func writeCentralSyncLocalState(
	t *testing.T,
	repository string,
	repositoryID string,
	updatedAt time.Time,
	variant string,
) {
	t.Helper()
	stateDirectory := filepath.Join(repository, filepath.FromSlash(optimizeDefaultStateDir))
	if err := os.MkdirAll(stateDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	digest := func(label string) string { return optimizeDigest("central-sync-test-v1", label, variant) }
	baseRevision := strings.Repeat("a", 40)
	targetRevision := strings.Repeat("b", 40)
	state := optimizeState{
		SchemaVersion: optimizeStateSchemaVersion, Generation: 1, Attempt: 1,
		Phase: "NATIVE_RETAINED", LastOutcome: optimizeOutcomeNative,
		LastReason: "CENTRAL_SYNC_TEST_STATE", BuildStarted: true,
		Bindings: optimizeBindings{
			SHA256: digest("bindings"), Completeness: optimizeBindingContractOnly,
			ExecutableSHA256: digest("executable"), WrapperSHA256: digest("wrapper"),
			InvocationSHA256: digest("invocation"), RepositoryScopeSHA256: digest("repository"),
			DiscoveryContextSHA256: digest("discovery-context"),
		},
		Budget: optimizeBudget{WallTimeSeconds: 1800, Pairs: 8, MaxBreakEvenBuilds: 30},
		Resume: optimizeResume{Mode: optimizeResumeAuto, Reason: optimizeResumeNone},
		Discovery: optimizeDiscoveryResult{
			Status: optimizeDiscoveryRetained, Reason: "CENTRAL_SYNC_TEST_DISCOVERY",
			RepositoryID: repositoryID, BaseRevision: baseRevision, TargetRevision: targetRevision,
			GeneratedFiles: []string{}, RequiredOutputs: []string{}, CandidateEntrypoints: []string{},
			ChangedProjects: []string{}, ReviewRequired: true, TestOptimization: "OUT_OF_SCOPE",
		},
		Calibration: optimizeCalibrationResult{
			Status: optimizeCalibrationRetained, Reason: "CENTRAL_SYNC_TEST_CALIBRATION",
			Interval95SavedMS: []float64{}, GeneratedFiles: []string{},
			DiscoverySHA256: digest("discovery-evidence"), TestOptimization: "OUT_OF_SCOPE",
		},
		Portfolio: emptyOptimizePortfolio("CENTRAL_SYNC_TEST_PORTFOLIO"),
		Selection: emptyOptimizeSelection(optimizeSelectionSkipped, optimizeSelectionReasonNone, false),
		UpdatedAt: updatedAt.UTC().Format(time.RFC3339Nano),
	}
	state.Selection.DurationNS = 1
	if !validOptimizeState(state) {
		t.Fatalf("central sync test state is not valid: %+v", state)
	}
	if err := writeCanonicalPrivateJSON(filepath.Join(stateDirectory, optimizeStateFile), state); err != nil {
		t.Fatal(err)
	}

	familySHA := digest("family")
	profileDirectoryRelative := filepath.ToSlash(filepath.Join(optimizeDefaultStateDir, "portfolio", "profiles", familySHA))
	profileDirectory := filepath.Join(repository, filepath.FromSlash(profileDirectoryRelative))
	if err := os.MkdirAll(profileDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{
		"manifest.json":           centralSyncTestJSON("buildopt.poc/impact-manifest/v1", variant+"-manifest"),
		"graph.json":              centralSyncTestJSON("buildopt.poc/impact-graph/v1", variant+"-graph"),
		"generated-manifest.json": centralSyncTestJSON("buildopt.poc/generated-manifest/v1", variant+"-generated"),
		"evidence.json":           centralSyncTestJSON("buildopt.poc/structural-measurement/v1", variant+"-evidence"),
		"profile.json":            centralSyncTestJSON("buildopt.poc/qualified-profile/v1", variant+"-profile"),
	}
	for name, raw := range files {
		if err := writePrivateAtomicFile(filepath.Join(profileDirectory, name), raw); err != nil {
			t.Fatal(err)
		}
	}
	fileDigest := func(name string) string {
		sum := sha256.Sum256(files[name])
		return hex.EncodeToString(sum[:])
	}
	entry := optimizePortfolioEntry{
		Family: optimizeFamilyLeaf, FamilySHA256: familySHA,
		ChangedProjects: []string{":app"}, RepositoryID: repositoryID,
		Entrypoints: []string{"assemble"}, CandidateEntrypoints: []string{":app:assemble"},
		RequiredOutputs: []string{"app/build/libs/app.jar"}, TargetRevision: targetRevision,
		WrapperSHA256: state.Bindings.WrapperSHA256, ExecutableSHA256: state.Bindings.ExecutableSHA256,
		ManifestSHA256: fileDigest("manifest.json"), GraphSHA256: fileDigest("graph.json"),
		GeneratedSHA256: fileDigest("generated-manifest.json"), EvidenceSHA256: fileDigest("evidence.json"),
		ProfileSHA256: fileDigest("profile.json"),
		ProfilePath:   filepath.ToSlash(filepath.Join(profileDirectoryRelative, "profile.json")),
		State:         "QUALIFIED",
	}
	portfolio := optimizeProfilePortfolio{
		SchemaVersion: optimizePortfolioSchemaVersion, Generation: 1,
		RepositoryScopeSHA256: optimizePortfolioRepositoryScope(repositoryID),
		Profiles:              []optimizePortfolioEntry{entry},
		UpdatedAt:             updatedAt.UTC().Format(time.RFC3339Nano),
	}
	portfolioDirectory := filepath.Join(stateDirectory, "portfolio")
	if err := os.MkdirAll(portfolioDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeCanonicalPrivateJSON(filepath.Join(portfolioDirectory, optimizePortfolioIndexFile), portfolio); err != nil {
		t.Fatal(err)
	}
}

func centralSyncTestJSON(schema, value string) []byte {
	raw, _ := json.Marshal(struct {
		SchemaVersion string `json:"schemaVersion"`
		Value         string `json:"value"`
	}{SchemaVersion: schema, Value: value})
	canonical, _ := contractcrypto.CanonicalizeJCS(raw)
	return canonical
}

func writeCentralSyncTokenDocument(t *testing.T, path string, issued sharedcache.IssuedCentralToken) {
	t.Helper()
	document := centralIssuedTokenDocument{
		SchemaVersion: "buildopt.central/access-token/v1",
		TokenID:       issued.TokenID, Token: issued.Token,
		RepositoryScopeSHA256: issued.Scope.RepositoryScopeSHA256,
		Tenant:                issued.Scope.Tenant, Repository: issued.Scope.Repository,
		TrustDomain: issued.Scope.TrustDomain, Namespace: issued.Scope.Namespace,
		NamespaceGeneration: issued.Scope.NamespaceGeneration,
		Capabilities:        issued.Capabilities,
		IssuedAt:            issued.IssuedAt.Format(time.RFC3339Nano),
		ExpiresAt:           issued.ExpiresAt.Format(time.RFC3339Nano),
	}
	raw, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func allCentralKindStatuses(result centralSyncResult, status string) bool {
	if len(result.Kinds) != 3 {
		return false
	}
	for _, kind := range result.Kinds {
		if kind.Status != status || !kind.SnapshotVerified {
			return false
		}
	}
	return true
}

func hasCentralKindStatus(result centralSyncResult, status string) bool {
	for _, kind := range result.Kinds {
		if kind.Status == status {
			return true
		}
	}
	return false
}

func centralKindStatus(result centralSyncResult, wanted sharedcache.StateKind) string {
	for _, kind := range result.Kinds {
		if kind.Kind == wanted {
			return kind.Status
		}
	}
	return ""
}

func assertCentralSnapshotsEqual(t *testing.T, left, right string) {
	t.Helper()
	for _, kind := range []string{"evidence", "portfolio", "checkpoint"} {
		for _, name := range []string{"head.json", "manifest.json", "bundle.json"} {
			leftRaw, err := os.ReadFile(filepath.Join(left, filepath.FromSlash(centralConnectionDir), centralSnapshotDir, kind, name))
			if err != nil {
				t.Fatal(err)
			}
			rightRaw, err := os.ReadFile(filepath.Join(right, filepath.FromSlash(centralConnectionDir), centralSnapshotDir, kind, name))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(leftRaw, rightRaw) {
				t.Fatalf("%s/%s differs between producer and consumer", kind, name)
			}
		}
	}
}
