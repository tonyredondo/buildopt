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
	pocRemoteCacheLatency   = 80 * time.Millisecond
	pocRemoteCacheBandwidth = int64(20 << 20)
	pocRemoteCacheTasks     = 8
	pocRemoteCacheBytes     = int64(32 << 20)
	pocRemoteCacheUsername  = "buildopt-poc"
	pocRemoteCachePassword  = "remote-cache-value"
)

type pocRemoteCacheMetrics struct {
	mutex    sync.Mutex
	requests int
	bytes    int64
}

type pocRemoteCacheMetricSnapshot struct {
	Requests int   `json:"requests"`
	Bytes    int64 `json:"bytes"`
}

func (metrics *pocRemoteCacheMetrics) snapshot() pocRemoteCacheMetricSnapshot {
	metrics.mutex.Lock()
	defer metrics.mutex.Unlock()
	return pocRemoteCacheMetricSnapshot{Requests: metrics.requests, Bytes: metrics.bytes}
}

func (metrics *pocRemoteCacheMetrics) reset() {
	metrics.mutex.Lock()
	metrics.requests = 0
	metrics.bytes = 0
	metrics.mutex.Unlock()
}

type pocRemoteCacheObservation struct {
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
	RequiredOutputCount        int    `json:"requiredOutputCount"`
	RequiredOutputBytes        int64  `json:"requiredOutputBytes"`
	RequiredOutputSHA256       string `json:"requiredOutputSha256"`
	TaskOutcomeSetSHA256       string `json:"taskOutcomeSetSha256"`
	OutputsIdentical           bool   `json:"outputsIdentical"`
	TaskOutcomesIdentical      bool   `json:"taskOutcomesIdentical"`
	ProductAttributableFailure bool   `json:"productAttributableFailure"`
}

type pocRemoteCacheRun struct {
	Duration    int64
	Started     time.Time
	Finished    time.Time
	OutputCount int
	OutputBytes int64
	OutputHash  string
	TaskHash    string
}

func TestPOCRemoteCacheValueExperiment(t *testing.T) {
	resultPath := os.Getenv("BUILDOPT_POC_REMOTE_CACHE_RESULT")
	if resultPath == "" {
		t.Skip("controlled remote-cache measurement is not requested")
	}
	repoRoot := requiredPOCRemoteCacheEnv(t, "BUILDOPT_POC_REPO_ROOT")
	gradleBin := requiredPOCRemoteCacheEnv(t, "BUILDOPT_POC_GRADLE_BIN")
	revision := requiredPOCRemoteCacheEnv(t, "BUILDOPT_POC_REVISION")
	runnerSHA := requiredPOCRemoteCacheEnv(t, "BUILDOPT_POC_RUNNER_SHA")
	specSHA := requiredPOCRemoteCacheEnv(t, "BUILDOPT_POC_SPEC_SHA")
	fixtureSHA := requiredPOCRemoteCacheEnv(t, "BUILDOPT_POC_FIXTURE_SHA")
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" || runtime.NumCPU() != 12 {
		t.Fatalf("remote-cache POC runner = %s/%s/%d, want linux/amd64/12", runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
	}

	root := t.TempDir()
	fixture := filepath.Join(repoRoot, "fixtures", "remote-cache-value")
	seedProject := filepath.Join(root, "seed-project")
	controlProject := filepath.Join(root, "control-project")
	candidateProject := filepath.Join(root, "candidate-project")
	copyPOCRemoteCacheTree(t, fixture, seedProject)
	copyPOCRemoteCacheTree(t, fixture, controlProject)
	copyPOCRemoteCacheTree(t, fixture, candidateProject)

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
	seedRun := runPOCRemoteCacheGradle(t, gradleBin, seedProject, filepath.Join(root, "seed-home"), seedServer.URL, true)
	seedServer.Close()
	if seedRun.OutputCount != pocRemoteCacheTasks || seedRun.OutputBytes != pocRemoteCacheBytes {
		t.Fatalf("seed outputs = %d/%d", seedRun.OutputCount, seedRun.OutputBytes)
	}
	recordedMutex.Lock()
	seeded := make(map[string][]byte, len(recorded))
	var seededBytes int64
	for key, payload := range recorded {
		seeded[key] = bytes.Clone(payload)
		seededBytes += int64(len(payload))
	}
	recordedMutex.Unlock()
	if len(seeded) < pocRemoteCacheTasks || seededBytes < pocRemoteCacheBytes {
		t.Fatalf("seeded remote entries = %d/%d", len(seeded), seededBytes)
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
			t.Fatalf("seed Shared %s: %v", key, putErr)
		}
		objects = append(objects, pending.Object)
	}
	status, err := storage.AttemptStatus(ctx, binding.AttemptID)
	if err != nil {
		t.Fatal(err)
	}
	request := StartAttemptRequest{
		AttemptID:                 status.AttemptID,
		AuthorityDigest:           status.AuthorityDigest,
		Repository:                status.Repository,
		NamespaceGeneration:       status.NamespaceGeneration,
		SourceRevision:            status.SourceRevision,
		SourceStateDigest:         status.SourceStateDigest,
		PolicyDigest:              status.PolicyDigest,
		ConfigurationPolicyDigest: status.ConfigurationPolicyDigest,
		CacheContractDigest:       status.CacheContractDigest,
		OwnerID:                   status.OwnerID,
		LeaseID:                   status.LeaseID,
		LeaseExpiresAt:            status.LeaseExpiresAt,
	}
	canonical := signLifecycleDecision(t, privateKey, request, "poc-remote-cache-decision", objects, testRevocationEpoch, now)
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
	shaped := pocRemoteCacheShape(metrics, pocRemoteCacheUpstreamAuth(sharedHandler, token.Token, binding.AuthorityDigest))
	sharedServer := httptest.NewServer(shaped)
	defer sharedServer.Close()

	edgeConfig := edgecache.Config{
		Shared: edgecache.Shared{BaseURL: sharedServer.URL, AllowInsecureLoopback: true},
		Storage: edgecache.Storage{
			StateDirectory:       filepath.Join(root, "edge"),
			FilesystemPolicy:     edgecache.FilesystemPolicy,
			CapacityBytes:        edgecache.MinimumCapacityBytes,
			MaximumObjectBytes:   edgecache.MaximumObjectBytes,
			StableTTLSeconds:     int64(edgecache.MaximumStableTTL / time.Second),
			PendingTTLSeconds:    int64(edgecache.MaximumPendingTTL / time.Second),
			HighWatermarkPercent: edgecache.HighWatermarkPercent,
			LowWatermarkPercent:  edgecache.LowWatermarkPercent,
			ProtectedPercent:     edgecache.ProtectedPercent,
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
			t.Fatalf("warm Edge %s: %v", key, readErr)
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

	controlHome := filepath.Join(root, "control-home")
	candidateHome := filepath.Join(root, "candidate-home")
	metrics.reset()
	_ = runPOCRemoteCacheGradle(t, gradleBin, controlProject, controlHome, controlServer.URL, false)
	_ = runPOCRemoteCacheGradle(t, gradleBin, candidateProject, candidateHome, edgeServer.URL, false)
	metrics.reset()

	orders := []string{"CONTROL_FIRST", "CANDIDATE_FIRST", "CONTROL_FIRST", "CANDIDATE_FIRST"}
	observations := make([]pocRemoteCacheObservation, 0, len(orders))
	for index, order := range orders {
		var control, candidate pocRemoteCacheRun
		var controlTraffic, candidateTraffic pocRemoteCacheMetricSnapshot
		if order == "CONTROL_FIRST" {
			metrics.reset()
			control = runPOCRemoteCacheGradle(t, gradleBin, controlProject, controlHome, controlServer.URL, false)
			controlTraffic = metrics.snapshot()
			metrics.reset()
			candidate = runPOCRemoteCacheGradle(t, gradleBin, candidateProject, candidateHome, edgeServer.URL, false)
			candidateTraffic = metrics.snapshot()
		} else {
			metrics.reset()
			candidate = runPOCRemoteCacheGradle(t, gradleBin, candidateProject, candidateHome, edgeServer.URL, false)
			candidateTraffic = metrics.snapshot()
			metrics.reset()
			control = runPOCRemoteCacheGradle(t, gradleBin, controlProject, controlHome, controlServer.URL, false)
			controlTraffic = metrics.snapshot()
		}
		gap := candidate.Started.Sub(control.Finished)
		if order == "CANDIDATE_FIRST" {
			gap = control.Started.Sub(candidate.Finished)
		}
		if gap < 0 || gap > 15*time.Second {
			t.Fatalf("pair %d inter-arm gap = %v", index+1, gap)
		}
		if control.OutputHash != candidate.OutputHash || control.TaskHash != candidate.TaskHash {
			t.Fatalf("pair %d outputs/tasks differ", index+1)
		}
		if controlTraffic.Requests < pocRemoteCacheTasks || controlTraffic.Bytes < pocRemoteCacheBytes {
			t.Fatalf("pair %d control traffic = %+v", index+1, controlTraffic)
		}
		if candidateTraffic.Requests != 0 || candidateTraffic.Bytes != 0 {
			t.Fatalf("pair %d candidate contacted Shared = %+v", index+1, candidateTraffic)
		}
		observations = append(observations, pocRemoteCacheObservation{
			Pair: index + 1, Order: order,
			ControlDurationMs: control.Duration, CandidateDurationMs: candidate.Duration,
			SavedMs:               control.Duration - candidate.Duration,
			InterArmIdleGapMs:     gap.Milliseconds(),
			ControlOriginRequests: controlTraffic.Requests, ControlOriginBytes: controlTraffic.Bytes,
			CandidateOriginRequests: candidateTraffic.Requests, CandidateOriginBytes: candidateTraffic.Bytes,
			RequiredOutputCount: control.OutputCount, RequiredOutputBytes: control.OutputBytes,
			RequiredOutputSHA256: control.OutputHash, TaskOutcomeSetSHA256: control.TaskHash,
			OutputsIdentical: true, TaskOutcomesIdentical: true,
			ProductAttributableFailure: false,
		})
		t.Logf("remote-cache pair %d/4: native=%dms edge=%dms saved=%dms order=%s", index+1, control.Duration, candidate.Duration, control.Duration-candidate.Duration, order)
	}

	controlMean, candidateMean, savedMean, ratio, interval, positive, qualified := summarizePOCRemoteCache(observations)
	decisionName := "RETAIN_NATIVE_REMOTE_CACHE"
	if qualified {
		decisionName = "QUALIFY_EDGE_LOCALITY_FOR_CONTROLLED_REMOTE_CACHE_POC"
	}
	result := map[string]any{
		"schemaVersion":      "buildopt.evidence/poc-remote-cache-value/v1",
		"workItem":           "POC-REMOTE-CACHE-VALUE-001",
		"capturedAt":         time.Now().UTC().Truncate(time.Second).Format(time.RFC3339),
		"buildoptRevision":   revision,
		"runnerScriptSha256": runnerSHA,
		"specSha256":         specSHA,
		"fixtureSha256":      fixtureSHA,
		"runner":             map[string]any{"id": "linux-amd64-12c-16659865600b-v1", "cpuCount": 12, "memoryBytes": 16659865600, "jdk": "21", "gradle": "9.6.1"},
		"hypothesis":         map[string]any{"mechanism": "EDGE_COMMITTED_READ_LOCALITY", "control": "GRADLE_HTTP_BUILD_CACHE_DIRECT_TO_SHARED", "candidate": "GRADLE_HTTP_BUILD_CACHE_TO_PREWARMED_BUILDOPT_EDGE", "singleChangedInput": "CACHE_READ_LOCATION"},
		"network":            map[string]any{"model": "LOOPBACK_SHAPED_WAN", "latencyPerResponseMs": 80, "bandwidthBytesPerSecond": 20971520, "packetLossRatio": 0},
		"preparation":        map[string]any{"measured": false, "seededObjectCount": len(seeded), "seededObjectBytes": seededBytes, "edgeWarmupOriginRequests": edgeWarmup.Requests, "edgeWarmupOriginBytes": edgeWarmup.Bytes},
		"observations":       observations,
		"result":             map[string]any{"pairs": 4, "controlMeanMs": controlMean, "candidateMeanMs": candidateMean, "meanSavedMs": savedMean, "reductionRatio": ratio, "interval95SavedMs": interval, "positivePairs": positive, "qualified": qualified, "decision": decisionName},
		"boundaries":         map[string]any{"sameSharedOrigin": true, "sameGradleHttpBuildCacheClient": true, "sameCommittedObjectBytes": true, "sameWorkloadAndOutputs": true, "sameAuthenticationCheck": true, "onlyReadLocalityChanges": true, "safeCacheChanged": false, "runtimeTuningChanged": false, "buildImpactChanged": false, "testOptimizationModified": false, "proofOfConcept": true, "productionReadinessClaimed": false, "soakRequired": false, "designPartnerRequired": false},
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(resultPath, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestPOCRemoteCacheEndpoint(t *testing.T) {
	for _, baseURL := range []string{"http://127.0.0.1:1234", "http://127.0.0.1:1234/"} {
		if endpoint := pocRemoteCacheEndpoint(baseURL); endpoint != "http://127.0.0.1:1234/cache/" {
			t.Fatalf("pocRemoteCacheEndpoint(%q) = %q", baseURL, endpoint)
		}
	}
}

func requiredPOCRemoteCacheEnv(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Fatalf("%s is required", name)
	}
	return value
}

func pocRemoteCacheBasicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		username, password, ok := request.BasicAuth()
		if !ok || username != pocRemoteCacheUsername || password != pocRemoteCachePassword {
			response.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(response, request)
	})
}

func pocRemoteCacheUpstreamAuth(next http.Handler, token, authorityDigest string) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer "+token || request.Header.Get(AuthorityDigestHeader) != authorityDigest {
			response.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(response, request)
	})
}

func pocRemoteCacheShape(metrics *pocRemoteCacheMetrics, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		recorder := httptest.NewRecorder()
		next.ServeHTTP(recorder, request)
		for name, values := range recorder.Header() {
			for _, value := range values {
				response.Header().Add(name, value)
			}
		}
		payload := recorder.Body.Bytes()
		if request.Method == http.MethodGet && recorder.Code == http.StatusOK {
			metrics.mutex.Lock()
			metrics.requests++
			metrics.bytes += int64(len(payload))
			metrics.mutex.Unlock()
			time.Sleep(pocRemoteCacheLatency + time.Duration(int64(time.Second)*int64(len(payload))/pocRemoteCacheBandwidth))
		}
		response.WriteHeader(recorder.Code)
		_, _ = response.Write(payload)
	})
}

func pocRemoteCacheKey(path string) (string, bool) {
	const prefix = "/cache/"
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	key := strings.TrimPrefix(path, prefix)
	return key, key != "" && !strings.Contains(key, "/")
}

func copyPOCRemoteCacheTree(t *testing.T, source, target string) {
	t.Helper()
	if err := filepath.Walk(source, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		if info.IsDir() {
			return os.MkdirAll(destination, 0o700)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, content, 0o600)
	}); err != nil {
		t.Fatal(err)
	}
}

func runPOCRemoteCacheGradle(t *testing.T, gradleBin, project, home, remoteURL string, push bool) pocRemoteCacheRun {
	t.Helper()
	if err := os.RemoveAll(filepath.Join(project, "build")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(project, ".gradle")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	command := exec.Command(gradleBin, "--no-daemon", "--build-cache", "--no-configuration-cache", "--console=plain", "--max-workers=4", "remoteCacheFixture")
	command.Dir = project
	pushValue := "0"
	if push {
		pushValue = "1"
	}
	cacheURL := pocRemoteCacheEndpoint(remoteURL)
	command.Env = append(os.Environ(), "GRADLE_USER_HOME="+home, "BUILDOPT_POC_REMOTE_CACHE_URL="+cacheURL, "BUILDOPT_POC_REMOTE_CACHE_PUSH="+pushValue)
	output, err := command.CombinedOutput()
	finished := time.Now()
	if err != nil {
		t.Fatalf("Gradle remote-cache arm failed: %v\n%s", err, output)
	}
	if !bytes.Contains(output, []byte("BUILD SUCCESSFUL")) {
		t.Fatalf("Gradle remote-cache arm lacks success marker:\n%s", output)
	}
	fromCache := make([]string, 0, pocRemoteCacheTasks)
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "> Task :generateBlob") && strings.HasSuffix(line, " FROM-CACHE") {
			fromCache = append(fromCache, line)
		}
	}
	if !push && len(fromCache) != pocRemoteCacheTasks {
		t.Fatalf("Gradle remote-cache hits = %d, want %d\n%s", len(fromCache), pocRemoteCacheTasks, output)
	}
	sort.Strings(fromCache)
	taskDigest := sha256.Sum256([]byte(strings.Join(fromCache, "\n") + "\n"))
	outputRoot := filepath.Join(project, "build", "remote-cache")
	entries, err := os.ReadDir(outputRoot)
	if err != nil {
		t.Fatal(err)
	}
	hasher := sha256.New()
	var outputBytes int64
	var outputCount int
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		content, readErr := os.ReadFile(filepath.Join(outputRoot, entry.Name()))
		if readErr != nil {
			t.Fatal(readErr)
		}
		_, _ = hasher.Write([]byte(entry.Name()))
		_, _ = hasher.Write([]byte{0})
		_, _ = hasher.Write(content)
		outputCount++
		outputBytes += int64(len(content))
	}
	if outputCount != pocRemoteCacheTasks || outputBytes != pocRemoteCacheBytes {
		t.Fatalf("Gradle outputs = %d/%d", outputCount, outputBytes)
	}
	return pocRemoteCacheRun{Duration: finished.Sub(started).Milliseconds(), Started: started, Finished: finished, OutputCount: outputCount, OutputBytes: outputBytes, OutputHash: hex.EncodeToString(hasher.Sum(nil)), TaskHash: hex.EncodeToString(taskDigest[:])}
}

func pocRemoteCacheEndpoint(baseURL string) string {
	return strings.TrimSuffix(baseURL, "/") + "/cache/"
}

func summarizePOCRemoteCache(observations []pocRemoteCacheObservation) (float64, float64, float64, float64, [2]float64, int, bool) {
	var controlTotal, candidateTotal, savedTotal int64
	positive := 0
	for _, observation := range observations {
		controlTotal += observation.ControlDurationMs
		candidateTotal += observation.CandidateDurationMs
		savedTotal += observation.SavedMs
		if observation.SavedMs > 0 {
			positive++
		}
	}
	controlMean := float64(controlTotal) / 4
	candidateMean := float64(candidateTotal) / 4
	savedMean := float64(savedTotal) / 4
	ratio := float64(savedTotal) / float64(controlTotal)
	bootstrap := make([]float64, 4096)
	for sample := range bootstrap {
		state := uint32(2654435761 * uint64(sample+1))
		var sum int64
		for draw := 0; draw < 4; draw++ {
			state = 1664525*state + 1013904223
			index := int(state / 1073741824)
			sum += observations[index].SavedMs
		}
		bootstrap[sample] = float64(sum) / 4
	}
	sort.Float64s(bootstrap)
	interval := [2]float64{bootstrap[102], bootstrap[3993]}
	qualified := savedMean >= 500 && ratio >= 0.02 && interval[0] > 0 && positive == 4
	return controlMean, candidateMean, savedMean, ratio, interval, positive, qualified
}
