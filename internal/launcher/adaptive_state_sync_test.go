package launcher

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/adaptivefragment"
	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

func TestAdaptiveStateSyncReusesExactStateAcrossMachinesAndFallsBackSafely(t *testing.T) {
	bundle := readAdaptiveStateFixture(t)
	producerRoot := filepath.Join(t.TempDir(), "producer-state")
	producer, err := adaptivefragment.SaveLocalState(producerRoot, bundle, "")
	if err != nil {
		t.Fatal(err)
	}
	storage, err := sharedcache.Open(context.Background(), filepath.Join(t.TempDir(), "central-state"))
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	now := time.Now().UTC().Truncate(time.Second)
	issued, err := storage.IssueCentralToken(
		context.Background(),
		sharedcache.CentralTokenIssueRequest{
			Scope: sharedcache.CentralTokenScope{
				RepositoryScopeSHA256: bundle.Fragment.RepositoryScopeSHA256,
				Tenant:                "owner-poc", Repository: "example/adaptive-state",
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
	recorder := &adaptiveStatePathRecorder{next: handler}
	server := httptest.NewUnstartedServer(recorder)
	server.TLS = &tls.Config{MinVersion: tls.VersionTLS13}
	server.StartTLS()
	certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	connection := centralConnection{
		SchemaVersion: centralConnectionSchema, ServerURL: server.URL,
		RepositoryID:          "example/adaptive-state",
		RepositoryScopeSHA256: bundle.Fragment.RepositoryScopeSHA256,
		ConnectedAt:           now.Format(time.RFC3339Nano), TestOptimization: "OUT_OF_SCOPE",
	}
	newClient := func() *centralStateClient {
		t.Helper()
		client, clientErr := newCentralStateClient(server.URL, issued.Token, certificate)
		if clientErr != nil {
			t.Fatal(clientErr)
		}
		return client
	}
	origin := sharedcache.StateOrigin{
		BaseRevision: bundle.Fragment.RevisionID, TargetRevision: bundle.Fragment.RevisionID,
		BuildOptExecutableSHA256: strings.Repeat("b", 64),
		WrapperSHA256:            bundle.Fragment.Bindings[adaptivefragment.BindingWrapper],
		GradleVersion:            "9.6.1",
	}
	producerConnection := filepath.Join(t.TempDir(), "producer-connection")
	result, err := synchronizeAdaptiveState(
		context.Background(), "SYNC", producerRoot, producerConnection,
		connection, newClient(), origin,
	)
	if err != nil || result.NativeFallback || !result.UsedVerifiedLocal ||
		centralKindStatus(result.Central, sharedcache.StateKindEvidence) != "PUSHED" ||
		centralKindStatus(result.Central, sharedcache.StateKindPortfolio) != "PUSHED" {
		t.Fatalf("producer adaptive sync = %+v, %v", result, err)
	}

	consumerRoot := filepath.Join(t.TempDir(), "consumer-state")
	consumerConnection := filepath.Join(t.TempDir(), "consumer-connection")
	result, err = synchronizeAdaptiveState(
		context.Background(), "SYNC", consumerRoot, consumerConnection,
		connection, newClient(), origin,
	)
	if err != nil || !result.RestoredFromCentral || result.NativeFallback ||
		centralKindStatus(result.Central, sharedcache.StateKindEvidence) != "PULLED" ||
		centralKindStatus(result.Central, sharedcache.StateKindPortfolio) != "PULLED" {
		t.Fatalf("consumer adaptive sync = %+v, %v", result, err)
	}
	consumer, err := adaptivefragment.LoadLocalState(consumerRoot)
	if err != nil || consumer.HeadSHA256 != producer.HeadSHA256 ||
		!equalAdaptiveFiles(producer.Files, consumer.Files) {
		t.Fatalf("second machine did not restore exact state: %v", err)
	}
	if recorder.cacheRequests != 0 || recorder.stateRequests == 0 {
		t.Fatalf("adaptive transport used wrong protocol: cache=%d state=%d", recorder.cacheRequests, recorder.stateRequests)
	}

	server.Close()
	result, err = synchronizeAdaptiveState(
		context.Background(), "SYNC", consumerRoot, consumerConnection,
		connection, newClient(), origin,
	)
	if err == nil || result.NativeFallback || !result.UsedVerifiedLocal ||
		result.LocalHeadSHA256 != producer.HeadSHA256 {
		t.Fatalf("verified offline state was not retained: %+v, %v", result, err)
	}
	cleanRoot := filepath.Join(t.TempDir(), "clean-state")
	cleanConnection := filepath.Join(t.TempDir(), "clean-connection")
	result, err = synchronizeAdaptiveState(
		context.Background(), "SYNC", cleanRoot, cleanConnection,
		connection, newClient(), origin,
	)
	if err == nil || !result.NativeFallback || result.UsedVerifiedLocal ||
		result.Reason != "NATIVE_FALLBACK_NO_VERIFIED_STATE" {
		t.Fatalf("clean offline machine did not retain native: %+v, %v", result, err)
	}

	headPath := filepath.Join(consumerRoot, "head.json")
	if err := os.WriteFile(headPath, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err = synchronizeAdaptiveState(
		context.Background(), "SYNC", consumerRoot, consumerConnection,
		connection, newClient(), origin,
	)
	if !errors.Is(err, adaptivefragment.ErrLocalStateCorrupt) ||
		!result.NativeFallback || result.LocalStatus != "CORRUPT" {
		t.Fatalf("corrupt adaptive state was not rejected: %+v, %v", result, err)
	}
}

func TestAdaptiveStateSyncPreservesOptimisticConcurrency(t *testing.T) {
	bundle := readAdaptiveStateFixture(t)
	firstRoot := filepath.Join(t.TempDir(), "first")
	secondRoot := filepath.Join(t.TempDir(), "second")
	first, err := adaptivefragment.SaveLocalState(firstRoot, bundle, "")
	if err != nil {
		t.Fatal(err)
	}
	variant := bundle
	variant.Ledger.Entries = append([]adaptivefragment.LedgerEntry(nil), bundle.Ledger.Entries...)
	variant.Ledger.Entries[0].GrossSavedMs++
	variant.Ledger.Entries[0].CumulativeNetMs++
	second, err := adaptivefragment.SaveLocalState(secondRoot, variant, "")
	if err != nil {
		t.Fatal(err)
	}
	if first.HeadSHA256 == second.HeadSHA256 {
		t.Fatal("concurrent variants unexpectedly have one head")
	}

	storage, err := sharedcache.Open(context.Background(), filepath.Join(t.TempDir(), "central-state"))
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	now := time.Now().UTC().Truncate(time.Second)
	issued, err := storage.IssueCentralToken(context.Background(), sharedcache.CentralTokenIssueRequest{
		Scope: sharedcache.CentralTokenScope{
			RepositoryScopeSHA256: bundle.Fragment.RepositoryScopeSHA256,
			Tenant:                "owner-poc", Repository: "example/adaptive-race", TrustDomain: "owner-poc",
			Namespace: "gradle-9.6.1/linux-amd64/jdk-21/project", NamespaceGeneration: 1,
		},
		Capabilities: []sharedcache.CentralCapability{sharedcache.CentralStateRead, sharedcache.CentralStateWrite},
		ExpiresAt:    now.Add(time.Hour),
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := sharedcache.NewCentralHTTPSHandler(storage)
	if err != nil {
		t.Fatal(err)
	}
	controlled := &centralSyncControlledHandler{next: handler}
	controlled.barrierNextEvidenceObject()
	server := httptest.NewUnstartedServer(controlled)
	server.TLS = &tls.Config{MinVersion: tls.VersionTLS13}
	server.StartTLS()
	defer server.Close()
	certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	connection := centralConnection{
		SchemaVersion: centralConnectionSchema, ServerURL: server.URL,
		RepositoryID: "example/adaptive-race", RepositoryScopeSHA256: bundle.Fragment.RepositoryScopeSHA256,
		ConnectedAt: now.Format(time.RFC3339Nano), TestOptimization: "OUT_OF_SCOPE",
	}
	origin := sharedcache.StateOrigin{
		BaseRevision: bundle.Fragment.RevisionID, TargetRevision: bundle.Fragment.RevisionID,
		BuildOptExecutableSHA256: strings.Repeat("c", 64),
		WrapperSHA256:            bundle.Fragment.Bindings[adaptivefragment.BindingWrapper], GradleVersion: "9.6.1",
	}
	type outcome struct {
		result adaptiveStateSyncResult
		err    error
	}
	outcomes := make(chan outcome, 2)
	connectionRoots := []string{
		filepath.Join(t.TempDir(), "first-connection"),
		filepath.Join(t.TempDir(), "second-connection"),
	}
	for index, root := range []string{firstRoot, secondRoot} {
		index, root := index, root
		go func() {
			client, clientErr := newCentralStateClient(server.URL, issued.Token, certificate)
			if clientErr != nil {
				outcomes <- outcome{err: clientErr}
				return
			}
			result, syncErr := synchronizeAdaptiveState(
				context.Background(), "SYNC", root,
				connectionRoots[index],
				connection, client, origin,
			)
			outcomes <- outcome{result: result, err: syncErr}
		}()
	}
	items := []outcome{<-outcomes, <-outcomes}
	winnerObserved := false
	for _, item := range items {
		if item.err != nil {
			t.Fatalf("adaptive concurrent sync failed: %+v, %v", item.result, item.err)
		}
		winnerObserved = winnerObserved || hasCentralKindStatus(item.result.Central, "CONCURRENT_REMOTE_WON")
	}
	if !winnerObserved {
		t.Fatalf("adaptive CAS did not retain an explicit winner: %+v", items)
	}
}

func TestAdaptiveStateSyncRejectsUnknownTransportedFile(t *testing.T) {
	bundle := readAdaptiveStateFixture(t)
	snapshot, err := adaptivefragment.PrepareLocalState(bundle)
	if err != nil {
		t.Fatal(err)
	}
	origin := sharedcache.StateOrigin{
		BaseRevision: bundle.Fragment.RevisionID, TargetRevision: bundle.Fragment.RevisionID,
		BuildOptExecutableSHA256: strings.Repeat("d", 64),
		WrapperSHA256:            bundle.Fragment.Bindings[adaptivefragment.BindingWrapper], GradleVersion: "9.6.1",
	}
	publications, err := collectAdaptiveStatePublications(snapshot, origin)
	if err != nil {
		t.Fatal(err)
	}
	evidence := publications[sharedcache.StateKindEvidence].bundle
	portfolio := publications[sharedcache.StateKindPortfolio].bundle
	evidence.Files = append(evidence.Files, centralStateBundleFile{
		Path: "generations/unknown.json", SHA256: strings.Repeat("e", 64),
		SizeBytes: 2, PayloadSchemaVersion: "buildopt.adaptive/unknown/v1",
		ContentBase64: base64.RawStdEncoding.EncodeToString([]byte("{}")),
	})
	if _, _, err := decodeAdaptiveStateSnapshots(evidence, portfolio); err == nil ||
		!strings.Contains(err.Error(), "unknown file") {
		t.Fatalf("unknown transported adaptive file = %v", err)
	}
}

type adaptiveStatePathRecorder struct {
	next http.Handler

	mutex         sync.Mutex
	stateRequests int
	cacheRequests int
}

func (recorder *adaptiveStatePathRecorder) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	recorder.mutex.Lock()
	if strings.Contains(request.URL.Path, "/state/") {
		recorder.stateRequests++
	}
	if strings.Contains(request.URL.Path, "/cache/") {
		recorder.cacheRequests++
	}
	recorder.mutex.Unlock()
	recorder.next.ServeHTTP(response, request)
}

func readAdaptiveStateFixture(t *testing.T) adaptivefragment.StateBundle {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(
		"..", "..", "contracts", "jsonschema", "testdata",
		"adaptive-fragment-state.v1", "valid", "active-lifecycle.json",
	))
	if err != nil {
		t.Fatal(err)
	}
	var bundle adaptivefragment.StateBundle
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&bundle); err != nil {
		t.Fatal(err)
	}
	return bundle
}

func equalAdaptiveFiles(left, right map[string][]byte) bool {
	if len(left) != len(right) {
		return false
	}
	for path, raw := range left {
		if !bytes.Equal(raw, right[path]) {
			return false
		}
	}
	return true
}
