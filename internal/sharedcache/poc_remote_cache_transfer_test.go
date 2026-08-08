package sharedcache

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/edgecache"
	"github.com/tonyredondo/buildopt/internal/localauthority"
)

const (
	pocRemoteCacheTransferLatency   = 337 * time.Millisecond
	pocRemoteCacheTransferBandwidth = int64(6994831)
	pocRemoteCacheTransferOutputs   = 4062
	pocRemoteCacheTransferMinHits   = 4
)

var pocRemoteCacheTransferOutputRoots = []string{
	"clients/build/classes/java/main",
	"clients/build/classes/java/test",
	"clients/build/resources/main",
	"clients/build/resources/test",
}

type pocRemoteCacheTransferObservation struct {
	Pair                       int    `json:"pair"`
	Order                      string `json:"order"`
	ControlDurationMs          int64  `json:"controlDurationMs"`
	CandidateDurationMs        int64  `json:"candidateDurationMs"`
	SavedMs                    int64  `json:"savedMs"`
	InterArmIdleGapMs          int64  `json:"interArmIdleGapMs"`
	ControlOriginRequests      int    `json:"controlOriginRequests"`
	ControlOriginBytes         int64  `json:"controlOriginBytes"`
	CandidateOriginRequests    int    `json:"candidateOriginRequests"`
	CandidateOriginBytes       int64  `json:"candidateOriginBytes"`
	ControlRemoteCacheHits     int    `json:"controlRemoteCacheHits"`
	CandidateRemoteCacheHits   int    `json:"candidateRemoteCacheHits"`
	RequiredOutputCount        int    `json:"requiredOutputCount"`
	RequiredOutputBytes        int64  `json:"requiredOutputBytes"`
	RequiredOutputSHA256       string `json:"requiredOutputSha256"`
	TaskOutcomeSetSHA256       string `json:"taskOutcomeSetSha256"`
	OutputsIdentical           bool   `json:"outputsIdentical"`
	TaskOutcomesIdentical      bool   `json:"taskOutcomesIdentical"`
	ProductAttributableFailure bool   `json:"productAttributableFailure"`
}

type pocRemoteCacheTransferRun struct {
	Duration    int64
	Started     time.Time
	Finished    time.Time
	OutputCount int
	OutputBytes int64
	OutputHash  string
	TaskHash    string
	CacheHits   int
	Output      string
}

func TestPOCRemoteCacheTransferExperiment(t *testing.T) {
	resultPath := os.Getenv("BUILDOPT_POC_REMOTE_CACHE_TRANSFER_RESULT")
	if resultPath == "" {
		t.Skip("real-repository remote-cache transfer is not requested")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" || runtime.NumCPU() != 12 {
		t.Fatalf("remote-cache transfer runner = %s/%s/%d, want linux/amd64/12", runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
	}

	seedProject := requiredPOCRemoteCacheEnv(t, "BUILDOPT_POC_REMOTE_CACHE_TRANSFER_SEED_PROJECT")
	controlProject := requiredPOCRemoteCacheEnv(t, "BUILDOPT_POC_REMOTE_CACHE_TRANSFER_CONTROL_PROJECT")
	candidateProject := requiredPOCRemoteCacheEnv(t, "BUILDOPT_POC_REMOTE_CACHE_TRANSFER_CANDIDATE_PROJECT")
	seedHome := requiredPOCRemoteCacheEnv(t, "BUILDOPT_POC_REMOTE_CACHE_TRANSFER_SEED_HOME")
	controlHome := requiredPOCRemoteCacheEnv(t, "BUILDOPT_POC_REMOTE_CACHE_TRANSFER_CONTROL_HOME")
	candidateHome := requiredPOCRemoteCacheEnv(t, "BUILDOPT_POC_REMOTE_CACHE_TRANSFER_CANDIDATE_HOME")
	javaHome := requiredPOCRemoteCacheEnv(t, "BUILDOPT_POC_REMOTE_CACHE_TRANSFER_JAVA_HOME")
	revision := requiredPOCRemoteCacheEnv(t, "BUILDOPT_POC_REMOTE_CACHE_TRANSFER_REVISION")
	runnerSHA := requiredPOCRemoteCacheEnv(t, "BUILDOPT_POC_REMOTE_CACHE_TRANSFER_RUNNER_SHA")
	specSHA := requiredPOCRemoteCacheEnv(t, "BUILDOPT_POC_REMOTE_CACHE_TRANSFER_SPEC_SHA")
	harnessSHA := requiredPOCRemoteCacheEnv(t, "BUILDOPT_POC_REMOTE_CACHE_TRANSFER_HARNESS_SHA")
	sourceSHA := requiredPOCRemoteCacheEnv(t, "BUILDOPT_POC_REMOTE_CACHE_TRANSFER_SOURCE_SHA")
	dependencySHA := requiredPOCRemoteCacheEnv(t, "BUILDOPT_POC_REMOTE_CACHE_TRANSFER_DEPENDENCY_SHA")
	initScript := writePOCRemoteCacheTransferInit(t)

	recorded := make(map[string][]byte)
	var recordedMutex sync.Mutex
	seedServer := httptest.NewServer(pocRemoteCacheBasicAuth(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		key, ok := pocRemoteCacheKey(request.URL.Path)
		if !ok {
			response.WriteHeader(http.StatusNotFound)
			return
		}
		switch request.Method {
		case http.MethodGet:
			recordedMutex.Lock()
			payload, found := recorded[key]
			recordedMutex.Unlock()
			if !found {
				response.WriteHeader(http.StatusNotFound)
				return
			}
			response.Header().Set("Content-Length", fmt.Sprint(len(payload)))
			response.WriteHeader(http.StatusOK)
			_, _ = response.Write(payload)
		case http.MethodPut:
			payload, err := io.ReadAll(request.Body)
			if err != nil {
				response.WriteHeader(http.StatusBadRequest)
				return
			}
			recordedMutex.Lock()
			recorded[key] = bytes.Clone(payload)
			recordedMutex.Unlock()
			response.WriteHeader(http.StatusOK)
		default:
			response.WriteHeader(http.StatusMethodNotAllowed)
		}
	})))
	seedRun := runPOCRemoteCacheTransferGradle(t, seedProject, seedHome, javaHome, initScript, seedServer.URL, true)
	seedServer.Close()
	if seedRun.OutputCount != pocRemoteCacheTransferOutputs {
		t.Fatalf("Kafka seed outputs = %d, want %d", seedRun.OutputCount, pocRemoteCacheTransferOutputs)
	}

	recordedMutex.Lock()
	seeded := make(map[string][]byte, len(recorded))
	var seededBytes int64
	for key, payload := range recorded {
		seeded[key] = bytes.Clone(payload)
		seededBytes += int64(len(payload))
	}
	recordedMutex.Unlock()
	if len(seeded) < pocRemoteCacheTransferMinHits || seededBytes < 1<<20 {
		t.Fatalf("Kafka committed remote entries = %d/%d\n%s", len(seeded), seededBytes, seedRun.Output)
	}

	ctx := context.Background()
	now := sharedAuthorityNow
	storage := openLifecycleTestStorage(t)
	storage.clock = func() time.Time { return now }
	verified, credential, privateKey, _ := sharedAuthorityFixture(t, func(document *localauthority.Document) {
		document.Attempt.AllowRead = true
		document.Attempt.AllowWrite = true
	})
	binding, _, err := storage.InstallLocalAuthority(ctx, verified, credential, now)
	if err != nil {
		t.Fatal(err)
	}
	keys := make([]string, 0, len(seeded))
	for key := range seeded {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	objects := make([]CommitObject, 0, len(keys))
	for _, key := range keys {
		pending, putErr := storage.PutPending(ctx, binding.AttemptID, key, bytes.NewReader(seeded[key]))
		if putErr != nil {
			t.Fatalf("seed Kafka Shared %s: %v", key, putErr)
		}
		objects = append(objects, pending.Object)
	}
	status, err := storage.AttemptStatus(ctx, binding.AttemptID)
	if err != nil {
		t.Fatal(err)
	}
	request := StartAttemptRequest{
		AttemptID: status.AttemptID, AuthorityDigest: status.AuthorityDigest,
		Repository: status.Repository, NamespaceGeneration: status.NamespaceGeneration,
		SourceRevision: status.SourceRevision, SourceStateDigest: status.SourceStateDigest,
		PolicyDigest: status.PolicyDigest, ConfigurationPolicyDigest: status.ConfigurationPolicyDigest,
		CacheContractDigest: status.CacheContractDigest, OwnerID: status.OwnerID,
		LeaseID: status.LeaseID, LeaseExpiresAt: status.LeaseExpiresAt,
	}
	canonical := signLifecycleDecision(t, privateKey, request, "poc-remote-cache-transfer-decision", objects, testRevocationEpoch, now)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	decision, err := VerifyCommitDecision(ctx, canonical, map[string]ed25519.PublicKey{testDecisionKeyID: publicKey}, testRevocationEpoch, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := storage.CommitAttempt(ctx, status.StateVersion, testRevocationEpoch, decision); err != nil {
		t.Fatal(err)
	}
	sharedHandler, err := NewBetaTokenHTTPHandler(storage, binding, BetaTokenPlaneStable)
	if err != nil {
		t.Fatal(err)
	}
	token := issueBetaTokenForTest(t, storage, betaScopeForBinding(binding, BetaTokenPlaneStable), BetaTokenRead, now, now.Add(30*time.Minute))
	metrics := &pocRemoteCacheMetrics{}
	shaped := pocRemoteCacheShapeWithProfile(metrics, pocRemoteCacheUpstreamAuth(sharedHandler, token.Token, binding.AuthorityDigest), pocRemoteCacheTransferLatency, pocRemoteCacheTransferBandwidth)
	sharedServer := httptest.NewServer(shaped)
	defer sharedServer.Close()

	edgeConfig := edgecache.Config{
		Shared: edgecache.Shared{BaseURL: sharedServer.URL, AllowInsecureLoopback: true},
		Storage: edgecache.Storage{
			StateDirectory: filepath.Join(t.TempDir(), "edge"), FilesystemPolicy: edgecache.FilesystemPolicy,
			CapacityBytes: edgecache.MinimumCapacityBytes, MaximumObjectBytes: edgecache.MaximumObjectBytes,
			StableTTLSeconds: int64(edgecache.MaximumStableTTL / time.Second), PendingTTLSeconds: int64(edgecache.MaximumPendingTTL / time.Second),
			HighWatermarkPercent: edgecache.HighWatermarkPercent, LowWatermarkPercent: edgecache.LowWatermarkPercent,
			ProtectedPercent: edgecache.ProtectedPercent,
		},
	}
	edgeStore, err := edgecache.OpenStore(edgeConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer edgeStore.Close()
	readAuthority, err := edgecache.NewReadAuthority(verified, now)
	if err != nil {
		t.Fatal(err)
	}
	edgeClient, err := edgecache.NewSharedClient(edgeConfig.Shared, []byte(token.Token), sharedServer.Client())
	if err != nil {
		t.Fatal(err)
	}
	defer edgeClient.Close()
	for _, key := range keys {
		file, readErr := edgeStore.ReadThrough(ctx, readAuthority, edgeClient, key, now)
		if readErr != nil {
			t.Fatalf("warm Kafka Edge %s: %v", key, readErr)
		}
		_ = file.Close()
	}
	edgeWarmup := metrics.snapshot()
	proxy, err := edgecache.NewProxy(edgeStore, edgeClient, readAuthority, nil, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	edgeServer := httptest.NewServer(pocRemoteCacheBasicAuth(proxy.Handler()))
	defer edgeServer.Close()
	controlServer := httptest.NewServer(pocRemoteCacheBasicAuth(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		clone := request.Clone(request.Context())
		clone.Header = request.Header.Clone()
		clone.Header.Set("Authorization", "Bearer "+token.Token)
		clone.Header.Set(AuthorityDigestHeader, binding.AuthorityDigest)
		shaped.ServeHTTP(response, clone)
	})))
	defer controlServer.Close()

	metrics.reset()
	controlWarmup := runPOCRemoteCacheTransferGradle(t, controlProject, controlHome, javaHome, initScript, controlServer.URL, false)
	candidateWarmup := runPOCRemoteCacheTransferGradle(t, candidateProject, candidateHome, javaHome, initScript, edgeServer.URL, false)
	if controlWarmup.OutputHash != candidateWarmup.OutputHash || controlWarmup.TaskHash != candidateWarmup.TaskHash {
		t.Fatal("Kafka warm-up outputs or task outcomes differ")
	}
	metrics.reset()

	orders := []string{"CONTROL_FIRST", "CANDIDATE_FIRST", "CONTROL_FIRST", "CANDIDATE_FIRST"}
	observations := make([]pocRemoteCacheTransferObservation, 0, len(orders))
	for index, order := range orders {
		var control, candidate pocRemoteCacheTransferRun
		var controlTraffic, candidateTraffic pocRemoteCacheMetricSnapshot
		if order == "CONTROL_FIRST" {
			metrics.reset()
			control = runPOCRemoteCacheTransferGradle(t, controlProject, controlHome, javaHome, initScript, controlServer.URL, false)
			controlTraffic = metrics.snapshot()
			metrics.reset()
			candidate = runPOCRemoteCacheTransferGradle(t, candidateProject, candidateHome, javaHome, initScript, edgeServer.URL, false)
			candidateTraffic = metrics.snapshot()
		} else {
			metrics.reset()
			candidate = runPOCRemoteCacheTransferGradle(t, candidateProject, candidateHome, javaHome, initScript, edgeServer.URL, false)
			candidateTraffic = metrics.snapshot()
			metrics.reset()
			control = runPOCRemoteCacheTransferGradle(t, controlProject, controlHome, javaHome, initScript, controlServer.URL, false)
			controlTraffic = metrics.snapshot()
		}
		gap := candidate.Started.Sub(control.Finished)
		if order == "CANDIDATE_FIRST" {
			gap = control.Started.Sub(candidate.Finished)
		}
		if gap < 0 || gap > 15*time.Second {
			t.Fatalf("Kafka pair %d inter-arm gap = %v", index+1, gap)
		}
		if control.OutputHash != candidate.OutputHash || control.TaskHash != candidate.TaskHash {
			t.Fatalf("Kafka pair %d outputs or tasks differ", index+1)
		}
		if control.CacheHits < pocRemoteCacheTransferMinHits || controlTraffic.Requests < control.CacheHits {
			t.Fatalf("Kafka pair %d control hits/traffic = %d/%+v", index+1, control.CacheHits, controlTraffic)
		}
		if candidate.CacheHits != control.CacheHits {
			t.Fatalf("Kafka pair %d cache hits differ: %d/%d", index+1, control.CacheHits, candidate.CacheHits)
		}
		if candidateTraffic.Requests != 0 || candidateTraffic.Bytes != 0 {
			t.Fatalf("Kafka pair %d candidate contacted Shared = %+v", index+1, candidateTraffic)
		}
		observations = append(observations, pocRemoteCacheTransferObservation{
			Pair: index + 1, Order: order,
			ControlDurationMs: control.Duration, CandidateDurationMs: candidate.Duration,
			SavedMs: control.Duration - candidate.Duration, InterArmIdleGapMs: gap.Milliseconds(),
			ControlOriginRequests: controlTraffic.Requests, ControlOriginBytes: controlTraffic.Bytes,
			CandidateOriginRequests: candidateTraffic.Requests, CandidateOriginBytes: candidateTraffic.Bytes,
			ControlRemoteCacheHits: control.CacheHits, CandidateRemoteCacheHits: candidate.CacheHits,
			RequiredOutputCount: control.OutputCount, RequiredOutputBytes: control.OutputBytes,
			RequiredOutputSHA256: control.OutputHash, TaskOutcomeSetSHA256: control.TaskHash,
			OutputsIdentical: true, TaskOutcomesIdentical: true, ProductAttributableFailure: false,
		})
		t.Logf("Kafka remote-cache pair %d/4: native=%dms edge=%dms saved=%dms hits=%d order=%s", index+1, control.Duration, candidate.Duration, control.Duration-candidate.Duration, control.CacheHits, order)
	}

	summaryInput := make([]pocRemoteCacheObservation, len(observations))
	for index, observation := range observations {
		summaryInput[index] = pocRemoteCacheObservation{ControlDurationMs: observation.ControlDurationMs, CandidateDurationMs: observation.CandidateDurationMs, SavedMs: observation.SavedMs}
	}
	controlMean, candidateMean, savedMean, ratio, interval, positive, qualified := summarizePOCRemoteCache(summaryInput)
	decisionName := "RETAIN_NATIVE_REMOTE_CACHE_FOR_KAFKA_PROFILE"
	if qualified {
		decisionName = "QUALIFY_EDGE_LOCALITY_TRANSFER_ON_KAFKA_PROFILE"
	}
	result := map[string]any{
		"schemaVersion": "buildopt.evidence/poc-remote-cache-transfer/v1", "workItem": "POC-REMOTE-CACHE-TRANSFER-001",
		"capturedAt": time.Now().UTC().Truncate(time.Second).Format(time.RFC3339), "buildoptRevision": revision,
		"runnerScriptSha256": runnerSHA, "specSha256": specSHA, "harnessSha256": harnessSHA,
		"sourceArchiveSha256": sourceSHA, "dependencyCacheSha256": dependencySHA,
		"repository":   map[string]any{"nameWithOwner": "apache/kafka", "releaseTag": "4.3.1", "revision": "26b251a451ce941d3d7a55e6487bcb7f16b5ad48", "gradleVersion": "9.2.1", "jdk": "temurin-25.0.3+9"},
		"runner":       map[string]any{"id": "linux-amd64-12c-16659865600b-v1", "cpuCount": 12, "memoryBytes": 16659865600, "maxWorkers": 12},
		"hypothesis":   map[string]any{"mechanism": "EDGE_COMMITTED_READ_LOCALITY", "control": "GRADLE_HTTP_BUILD_CACHE_DIRECT_TO_SHARED", "candidate": "GRADLE_HTTP_BUILD_CACHE_TO_PREWARMED_BUILDOPT_EDGE", "singleChangedInput": "CACHE_READ_LOCATION"},
		"network":      map[string]any{"model": "INDEPENDENT_SOURCE_ARCHIVE_DERIVED_LOOPBACK_WAN", "latencyPerResponseMs": 337, "bandwidthBytesPerSecond": 6994831, "packetLossRatio": 0, "derivationMethod": "MEDIAN_OF_THREE_SEQUENTIAL_FIXED_SOURCE_ARCHIVE_DOWNLOADS"},
		"preparation":  map[string]any{"measured": false, "dependenciesResolvedBeforeMeasurement": true, "nativeRemoteCacheOnlineMode": true, "externalDependencyNetworkBlocked": true, "seededObjectCount": len(seeded), "seededObjectBytes": seededBytes, "edgeWarmupOriginRequests": edgeWarmup.Requests, "edgeWarmupOriginBytes": edgeWarmup.Bytes},
		"observations": observations,
		"result":       map[string]any{"pairs": 4, "controlMeanMs": controlMean, "candidateMeanMs": candidateMean, "meanSavedMs": savedMean, "reductionRatio": ratio, "interval95SavedMs": interval, "positivePairs": positive, "qualified": qualified, "decision": decisionName},
		"boundaries":   map[string]any{"sameSharedOrigin": true, "sameGradleHttpBuildCacheClient": true, "sameCommittedObjectBytes": true, "sameWorkloadAndOutputs": true, "sameAuthenticationCheck": true, "onlyReadLocalityChanges": true, "edgeImplementationChanged": false, "safeCacheChanged": false, "runtimeTuningChanged": false, "buildImpactChanged": false, "testSelectionChanged": false, "testExecutionChanged": false, "testOptimizationModified": false, "proofOfConcept": true, "productionReadinessClaimed": false, "soakRequired": false, "designPartnerRequired": false, "universalSavingsClaimed": false},
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(resultPath, append(encoded, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writePOCRemoteCacheTransferInit(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "remote-cache.init.gradle")
	content := `
import org.gradle.caching.http.HttpBuildCache

settingsEvaluated { settings ->
    settings.buildCache {
        local { enabled = false }
        remote(HttpBuildCache) {
            url = uri(System.getenv('BUILDOPT_POC_REMOTE_CACHE_URL'))
            push = System.getenv('BUILDOPT_POC_REMOTE_CACHE_PUSH') == '1'
            allowInsecureProtocol = true
            credentials {
                username = 'buildopt-poc'
                password = 'remote-cache-value'
            }
        }
    }
}
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func runPOCRemoteCacheTransferGradle(t *testing.T, project, home, javaHome, initScript, remoteURL string, push bool) pocRemoteCacheTransferRun {
	t.Helper()
	removePOCRemoteCacheTransferOutputs(t, project)
	started := time.Now()
	command := exec.Command(filepath.Join(project, "gradlew"), "--daemon", "--build-cache", "--no-configuration-cache", "--parallel", "--console=plain", "--max-workers=12", "--no-scan", "--init-script", initScript, ":clients:testClasses")
	command.Dir = project
	pushValue := "0"
	if push {
		pushValue = "1"
	}
	command.Env = append(filteredPOCRemoteCacheTransferEnv(os.Environ()), "JAVA_HOME="+javaHome, "GRADLE_USER_HOME="+home, "GRADLE_OPTS=-Dhttp.proxyHost=127.0.0.1 -Dhttp.proxyPort=1 -Dhttps.proxyHost=127.0.0.1 -Dhttps.proxyPort=1 -Dhttp.nonProxyHosts=127.0.0.1|localhost", "BUILDOPT_POC_REMOTE_CACHE_URL="+pocRemoteCacheEndpoint(remoteURL), "BUILDOPT_POC_REMOTE_CACHE_PUSH="+pushValue)
	output, err := command.CombinedOutput()
	finished := time.Now()
	if err != nil {
		t.Fatalf("Kafka remote-cache arm failed: %v\n%s", err, output)
	}
	if !bytes.Contains(output, []byte("BUILD SUCCESSFUL")) {
		t.Fatalf("Kafka remote-cache arm lacks success marker:\n%s", output)
	}
	lines := make([]string, 0)
	cacheHits := 0
	for _, line := range strings.Split(string(output), "\n") {
		if !strings.HasPrefix(line, "> Task ") {
			continue
		}
		if strings.Contains(line, ":test ") || strings.HasSuffix(line, ":test") {
			t.Fatalf("Kafka remote-cache arm executed a Gradle Test task: %s", line)
		}
		lines = append(lines, line)
		if strings.HasSuffix(line, " FROM-CACHE") {
			cacheHits++
		}
	}
	if !push && cacheHits < pocRemoteCacheTransferMinHits {
		t.Fatalf("Kafka remote-cache hits = %d, want at least %d\n%s", cacheHits, pocRemoteCacheTransferMinHits, output)
	}
	sort.Strings(lines)
	taskDigest := sha256.Sum256([]byte(strings.Join(lines, "\n") + "\n"))
	outputCount, outputBytes, outputHash := hashPOCRemoteCacheTransferOutputs(t, project)
	return pocRemoteCacheTransferRun{Duration: finished.Sub(started).Milliseconds(), Started: started, Finished: finished, OutputCount: outputCount, OutputBytes: outputBytes, OutputHash: outputHash, TaskHash: hex.EncodeToString(taskDigest[:]), CacheHits: cacheHits, Output: string(output)}
}

func filteredPOCRemoteCacheTransferEnv(values []string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if strings.HasPrefix(value, "CI=") || strings.HasPrefix(value, "BUILDOPT_POC_REMOTE_CACHE_") {
			continue
		}
		filtered = append(filtered, value)
	}
	return filtered
}

func removePOCRemoteCacheTransferOutputs(t *testing.T, project string) {
	t.Helper()
	paths := make([]string, 0)
	if err := filepath.Walk(project, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() && (info.Name() == "build" || info.Name() == ".gradle") {
			paths = append(paths, path)
			return filepath.SkipDir
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	sort.Slice(paths, func(i, j int) bool { return len(paths[i]) > len(paths[j]) })
	for _, path := range paths {
		if err := os.RemoveAll(path); err != nil {
			t.Fatal(err)
		}
	}
}

func hashPOCRemoteCacheTransferOutputs(t *testing.T, project string) (int, int64, string) {
	t.Helper()
	lines := make([]string, 0, pocRemoteCacheTransferOutputs)
	var outputBytes int64
	for _, relativeRoot := range pocRemoteCacheTransferOutputRoots {
		root := filepath.Join(project, filepath.FromSlash(relativeRoot))
		if info, err := os.Stat(root); err != nil || !info.IsDir() {
			t.Fatalf("Kafka required output root is unavailable: %s", relativeRoot)
		}
		if err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			digest := sha256.Sum256(content)
			relative, err := filepath.Rel(project, path)
			if err != nil {
				return err
			}
			lines = append(lines, hex.EncodeToString(digest[:])+"  "+filepath.ToSlash(relative))
			outputBytes += int64(len(content))
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	sort.Strings(lines)
	manifest := strings.Join(lines, "\n") + "\n"
	digest := sha256.Sum256([]byte(manifest))
	if len(lines) != pocRemoteCacheTransferOutputs {
		t.Fatalf("Kafka required output count = %d, want %d", len(lines), pocRemoteCacheTransferOutputs)
	}
	const expected = "0a1ba415bbdeec51775d720beed627b95b11f6073a2fef450ccfd269df57ea23"
	if actual := hex.EncodeToString(digest[:]); actual != expected {
		t.Fatalf("Kafka required output digest = %s, want %s", actual, expected)
	}
	return len(lines), outputBytes, hex.EncodeToString(digest[:])
}
