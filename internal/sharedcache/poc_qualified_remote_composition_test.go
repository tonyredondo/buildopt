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

const pocQualifiedRemoteJarSHA = "894d8e71aa10070f7c73212cfcce0489401a00c3217a75f2f3d55c8e6da4738a"

type pocQualifiedRemoteObservation struct {
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
	ControlFullGraph           bool   `json:"controlFullGraph"`
	CandidatePlanSelected      bool   `json:"candidatePlanSelected"`
	CandidateAlternative       string `json:"candidateAlternative"`
	CandidateJarFromCache      bool   `json:"candidateJarFromCache"`
	RequiredOutputCount        int    `json:"requiredOutputCount"`
	RequiredOutputBytes        int64  `json:"requiredOutputBytes"`
	RequiredOutputSHA256       string `json:"requiredOutputSha256"`
	ControlTaskOutcomeSHA256   string `json:"controlTaskOutcomeSha256"`
	CandidateTaskOutcomeSHA256 string `json:"candidateTaskOutcomeSha256"`
	OutputsIdentical           bool   `json:"outputsIdentical"`
	ProductAttributableFailure bool   `json:"productAttributableFailure"`
}

type pocQualifiedRemoteRun struct {
	Duration          int64
	Started           time.Time
	Finished          time.Time
	OutputBytes       int64
	OutputHash        string
	TaskHash          string
	CacheHits         int
	FullGraph         bool
	PlanSelected      bool
	Alternative       string
	JarFromCache      bool
	OutsideTask       bool
	SelectionReason   string
	Output            string
}

func TestPOCQualifiedRemoteCompositionExperiment(t *testing.T) {
	resultPath := os.Getenv("BUILDOPT_POC_QUALIFIED_REMOTE_RESULT")
	if resultPath == "" {
		t.Skip("qualified remote composition is not requested")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" || runtime.NumCPU() != 12 {
		t.Fatalf("qualified composition runner = %s/%s/%d, want linux/amd64/12", runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
	}

	seedProject := requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_QUALIFIED_REMOTE_SEED_PROJECT")
	controlProject := requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_QUALIFIED_REMOTE_CONTROL_PROJECT")
	candidateProject := requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_QUALIFIED_REMOTE_CANDIDATE_PROJECT")
	seedHome := requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_QUALIFIED_REMOTE_SEED_HOME")
	controlHome := requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_QUALIFIED_REMOTE_CONTROL_HOME")
	candidateHome := requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_QUALIFIED_REMOTE_CANDIDATE_HOME")
	javaHome := requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_QUALIFIED_REMOTE_JAVA_HOME")
	buildoptBin := requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_QUALIFIED_REMOTE_BUILDOPT_BIN")
	packagedInit := requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_QUALIFIED_REMOTE_INIT_SCRIPT")
	pluginJar := requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_QUALIFIED_REMOTE_PLUGIN_JAR")
	logs := requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_QUALIFIED_REMOTE_LOGS")
	controlInit, candidateInit := writePOCQualifiedRemoteInitScripts(t, packagedInit)

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
	seedRun := runPOCQualifiedRemoteCandidate(t, "seed", seedProject, seedHome, javaHome, buildoptBin, candidateInit, pluginJar, seedServer.URL, true, false, logs)
	seedServer.Close()
	if !seedRun.PlanSelected || seedRun.Alternative != "kafka-clients-jar" || seedRun.OutputHash != pocQualifiedRemoteJarSHA {
		t.Fatalf("qualified seed did not exercise the fixed candidate:\n%s", seedRun.Output)
	}

	recordedMutex.Lock()
	seeded := make(map[string][]byte, len(recorded))
	var seededBytes int64
	for key, payload := range recorded {
		seeded[key] = bytes.Clone(payload)
		seededBytes += int64(len(payload))
	}
	recordedMutex.Unlock()
	if len(seeded) == 0 || seededBytes < 1<<20 {
		t.Fatalf("qualified composition committed entries = %d/%d", len(seeded), seededBytes)
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
			t.Fatalf("seed composition Shared %s: %v", key, putErr)
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
	canonical := signLifecycleDecision(t, privateKey, request, "poc-qualified-remote-composition-decision", objects, testRevocationEpoch, now)
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
	token := issueBetaTokenForTest(t, storage, betaScopeForBinding(binding, BetaTokenPlaneStable), BetaTokenRead, now, now.Add(45*time.Minute))
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
			t.Fatalf("warm composition Edge %s: %v", key, readErr)
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
	controlWarmup := runPOCQualifiedRemoteControl(t, "warmup-control", controlProject, controlHome, javaHome, controlInit, controlServer.URL, logs)
	candidateWarmup := runPOCQualifiedRemoteCandidate(t, "warmup-candidate", candidateProject, candidateHome, javaHome, buildoptBin, candidateInit, pluginJar, edgeServer.URL, false, true, logs)
	assertPOCQualifiedRemotePair(t, 0, controlWarmup, candidateWarmup)
	metrics.reset()

	orders := []string{"CONTROL_FIRST", "CANDIDATE_FIRST", "CONTROL_FIRST", "CANDIDATE_FIRST"}
	observations := make([]pocQualifiedRemoteObservation, 0, len(orders))
	for index, order := range orders {
		var control, candidate pocQualifiedRemoteRun
		var controlTraffic, candidateTraffic pocRemoteCacheMetricSnapshot
		if order == "CONTROL_FIRST" {
			metrics.reset()
			control = runPOCQualifiedRemoteControl(t, fmt.Sprintf("pair-%d-control", index+1), controlProject, controlHome, javaHome, controlInit, controlServer.URL, logs)
			controlTraffic = metrics.snapshot()
			metrics.reset()
			candidate = runPOCQualifiedRemoteCandidate(t, fmt.Sprintf("pair-%d-candidate", index+1), candidateProject, candidateHome, javaHome, buildoptBin, candidateInit, pluginJar, edgeServer.URL, false, true, logs)
			candidateTraffic = metrics.snapshot()
		} else {
			metrics.reset()
			candidate = runPOCQualifiedRemoteCandidate(t, fmt.Sprintf("pair-%d-candidate", index+1), candidateProject, candidateHome, javaHome, buildoptBin, candidateInit, pluginJar, edgeServer.URL, false, true, logs)
			candidateTraffic = metrics.snapshot()
			metrics.reset()
			control = runPOCQualifiedRemoteControl(t, fmt.Sprintf("pair-%d-control", index+1), controlProject, controlHome, javaHome, controlInit, controlServer.URL, logs)
			controlTraffic = metrics.snapshot()
		}
		gap := candidate.Started.Sub(control.Finished)
		if order == "CANDIDATE_FIRST" {
			gap = control.Started.Sub(candidate.Finished)
		}
		if gap < 0 || gap > 15*time.Second {
			t.Fatalf("qualified composition pair %d gap = %v", index+1, gap)
		}
		assertPOCQualifiedRemotePair(t, index+1, control, candidate)
		if controlTraffic.Requests == 0 || controlTraffic.Bytes == 0 {
			t.Fatalf("qualified composition pair %d control Shared traffic = %+v", index+1, controlTraffic)
		}
		if candidateTraffic.Requests != 0 || candidateTraffic.Bytes != 0 {
			t.Fatalf("qualified composition pair %d Edge fetched from Shared = %+v", index+1, candidateTraffic)
		}
		observations = append(observations, pocQualifiedRemoteObservation{
			Pair: index + 1, Order: order,
			ControlDurationMs: control.Duration, CandidateDurationMs: candidate.Duration,
			SavedMs: control.Duration - candidate.Duration, InterArmIdleGapMs: gap.Milliseconds(),
			ControlOriginRequests: controlTraffic.Requests, ControlOriginBytes: controlTraffic.Bytes,
			CandidateOriginRequests: candidateTraffic.Requests, CandidateOriginBytes: candidateTraffic.Bytes,
			ControlRemoteCacheHits: control.CacheHits, CandidateRemoteCacheHits: candidate.CacheHits,
			ControlFullGraph: control.FullGraph, CandidatePlanSelected: candidate.PlanSelected,
			CandidateAlternative: candidate.Alternative, CandidateJarFromCache: candidate.JarFromCache,
			RequiredOutputCount: 1, RequiredOutputBytes: control.OutputBytes,
			RequiredOutputSHA256: control.OutputHash,
			ControlTaskOutcomeSHA256: control.TaskHash, CandidateTaskOutcomeSHA256: candidate.TaskHash,
			OutputsIdentical: true, ProductAttributableFailure: false,
		})
		t.Logf("Kafka composition pair %d/4: native=%dms candidate=%dms saved=%dms order=%s", index+1, control.Duration, candidate.Duration, control.Duration-candidate.Duration, order)
	}

	if err := os.WriteFile(filepath.Join(candidateProject, ".buildopt-changes"), []byte("gradle.properties\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	impactFallback := runPOCQualifiedRemoteCandidate(t, "fallback-impact", candidateProject, candidateHome, javaHome, buildoptBin, candidateInit, pluginJar, edgeServer.URL, false, false, logs)
	impactFallbackPassed := impactFallback.FullGraph && impactFallback.SelectionReason == "IMPACT_GLOBAL_CHANGE" && impactFallback.OutsideTask && impactFallback.OutputHash == pocQualifiedRemoteJarSHA
	if !impactFallbackPassed {
		t.Fatalf("qualified composition impact fallback failed:\n%s", impactFallback.Output)
	}
	changePath := "clients/src/main/java/org/apache/kafka/clients/Metadata.java"
	if err := os.WriteFile(filepath.Join(candidateProject, ".buildopt-changes"), []byte(changePath+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	failingEdge := httptest.NewServer(pocRemoteCacheBasicAuth(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusServiceUnavailable)
	})))
	edgeFailure := runPOCQualifiedRemoteCandidate(t, "fallback-edge", candidateProject, candidateHome, javaHome, buildoptBin, candidateInit, pluginJar, failingEdge.URL, false, false, logs)
	failingEdge.Close()
	edgeFailureFallbackPassed := edgeFailure.PlanSelected && edgeFailure.OutputHash == pocQualifiedRemoteJarSHA
	if !edgeFailureFallbackPassed {
		t.Fatalf("qualified composition Edge failure fallback failed:\n%s", edgeFailure.Output)
	}

	summary := make([]pocRemoteCacheObservation, len(observations))
	for index, observation := range observations {
		summary[index] = pocRemoteCacheObservation{ControlDurationMs: observation.ControlDurationMs, CandidateDurationMs: observation.CandidateDurationMs, SavedMs: observation.SavedMs}
	}
	controlMean, candidateMean, savedMean, ratio, interval, positive, qualified := summarizePOCRemoteCache(summary)
	qualified = qualified && impactFallbackPassed && edgeFailureFallbackPassed
	decisionName := "RETAIN_SEPARATE_KAFKA_MECHANISMS"
	if qualified {
		decisionName = "QUALIFY_KAFKA_REMOTE_COMPOSITION"
	}
	result := map[string]any{
		"schemaVersion": "buildopt.evidence/poc-qualified-remote-composition/v1",
		"workItem": "POC-QUALIFIED-REMOTE-COMPOSITION-001",
		"capturedAt": time.Now().UTC().Truncate(time.Second).Format(time.RFC3339),
		"buildoptRevision": requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_QUALIFIED_REMOTE_REVISION"),
		"runnerScriptSha256": requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_QUALIFIED_REMOTE_RUNNER_SHA"),
		"specSha256": requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_QUALIFIED_REMOTE_SPEC_SHA"),
		"harnessSha256": requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_QUALIFIED_REMOTE_HARNESS_SHA"),
		"sourceArchiveSha256": requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_QUALIFIED_REMOTE_SOURCE_SHA"),
		"dependencyCacheSha256": requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_QUALIFIED_REMOTE_DEPENDENCY_SHA"),
		"installedAssets": map[string]any{
			"binarySha256": requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_QUALIFIED_REMOTE_BINARY_SHA"),
			"initScriptSha256": requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_QUALIFIED_REMOTE_INIT_SHA"),
			"pluginJarSha256": requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_QUALIFIED_REMOTE_PLUGIN_SHA"),
		},
		"componentEvidence": map[string]any{
			"kafkaImpactAndJarSha256": requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_QUALIFIED_REMOTE_IMPACT_EVIDENCE_SHA"),
			"kafkaEdgeLocalitySha256": requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_QUALIFIED_REMOTE_EDGE_EVIDENCE_SHA"),
		},
		"repository": map[string]any{"nameWithOwner": "apache/kafka", "releaseTag": "4.3.1", "revision": "26b251a451ce941d3d7a55e6487bcb7f16b5ad48", "gradleVersion": "9.2.1", "jdk": "temurin-25.0.3+9"},
		"runner": map[string]any{"id": "linux-amd64-12c-16659865600b-v1", "cpuCount": 12, "memoryBytes": 16659865600, "maxWorkers": 12},
		"network": map[string]any{"model": "INDEPENDENT_SOURCE_ARCHIVE_DERIVED_LOOPBACK_WAN", "latencyPerResponseMs": 337, "bandwidthBytesPerSecond": 6994831, "packetLossRatio": 0},
		"preparation": map[string]any{"measured": false, "dependenciesResolvedBeforeMeasurement": true, "externalDependencyNetworkBlocked": true, "seededObjectCount": len(seeded), "seededObjectBytes": seededBytes, "edgeWarmupOriginRequests": edgeWarmup.Requests, "edgeWarmupOriginBytes": edgeWarmup.Bytes},
		"observations": observations,
		"safety": map[string]any{"impactFallbackPassed": impactFallbackPassed, "edgeFailureFallbackPassed": edgeFailureFallbackPassed},
		"result": map[string]any{"pairs": 4, "controlMeanMs": controlMean, "candidateMeanMs": candidateMean, "meanSavedMs": savedMean, "reductionRatio": ratio, "interval95SavedMs": interval, "positivePairs": positive, "qualified": qualified, "decision": decisionName},
		"boundaries": map[string]any{"sameSharedOrigin": true, "sameCommittedObjectBytes": true, "sameSourceAndChange": true, "sameRequiredOutput": true, "testTasksForbidden": true, "runtimeTuningChanged": false, "safeCacheChanged": false, "hotStateChanged": false, "standardCopyChanged": false, "testSelectionChanged": false, "testExecutionChanged": false, "testOptimizationModified": false, "proofOfConcept": true, "productionReadinessClaimed": false, "soakRequired": false, "designPartnerRequired": false, "universalSavingsClaimed": false},
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(resultPath, append(encoded, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writePOCQualifiedRemoteInitScripts(t *testing.T, packagedInit string) (string, string) {
	t.Helper()
	remote := `
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
	root := t.TempDir()
	control := filepath.Join(root, "control.init.gradle")
	candidate := filepath.Join(root, "candidate.init.gradle")
	packaged, err := os.ReadFile(packagedInit)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(control, []byte(remote), 0o600); err != nil {
		t.Fatal(err)
	}
	combined := append(bytes.Clone(packaged), []byte("\n"+remote)...)
	if err := os.WriteFile(candidate, combined, 0o600); err != nil {
		t.Fatal(err)
	}
	return control, candidate
}

func runPOCQualifiedRemoteControl(t *testing.T, label, project, home, javaHome, initScript, remoteURL, logs string) pocQualifiedRemoteRun {
	t.Helper()
	removePOCRemoteCacheTransferOutputs(t, project)
	command := exec.Command(filepath.Join(project, "gradlew"), "--daemon", "--build-cache", "--no-configuration-cache", "--parallel", "--console=plain", "--max-workers=12", "--no-scan", "--init-script", initScript, "assemble")
	return executePOCQualifiedRemote(t, label, command, project, home, javaHome, remoteURL, false, logs)
}

func runPOCQualifiedRemoteCandidate(t *testing.T, label, project, home, javaHome, buildoptBin, initScript, pluginJar, remoteURL string, push, requireJarCache bool, logs string) pocQualifiedRemoteRun {
	t.Helper()
	removePOCRemoteCacheTransferOutputs(t, project)
	args := []string{
		"impact", "--repository-id", "apache/kafka", "--pipeline-class", "poc-kafka-packaging-v1",
		"--changes-file", ".buildopt-changes", "--manifest", "buildopt-impact-manifest.json",
		"--graph", "buildopt-impact-graph.generated.json", "--generated-manifest", "buildopt-impact.generated.json",
		"--cache-standard-jar-producers", "--gradle-option=--daemon", "--gradle-option=--build-cache",
		"--gradle-option=--no-configuration-cache", "--gradle-option=--parallel",
		"--gradle-option=--console=plain", "--gradle-option=--max-workers=12", "--gradle-option=--no-scan",
	}
	command := exec.Command(buildoptBin, args...)
	run := executePOCQualifiedRemote(t, label, command, project, home, javaHome, remoteURL, push, logs,
		"BUILDOPT_GRADLE_INIT_SCRIPT="+initScript,
		"BUILDOPT_GRADLE_PLUGIN_JAR="+pluginJar,
	)
	if requireJarCache && !run.JarFromCache {
		t.Fatalf("qualified candidate Jar was not restored from cache:\n%s", run.Output)
	}
	return run
}

func executePOCQualifiedRemote(t *testing.T, label string, command *exec.Cmd, project, home, javaHome, remoteURL string, push bool, logs string, extraEnvironment ...string) pocQualifiedRemoteRun {
	t.Helper()
	pushValue := "0"
	if push {
		pushValue = "1"
	}
	command.Dir = project
	command.Env = append(filteredPOCQualifiedRemoteEnv(os.Environ()),
		"JAVA_HOME="+javaHome,
		"GRADLE_USER_HOME="+home,
		"GRADLE_OPTS=-Dhttp.proxyHost=127.0.0.1 -Dhttp.proxyPort=1 -Dhttps.proxyHost=127.0.0.1 -Dhttps.proxyPort=1 -Dhttp.nonProxyHosts=127.0.0.1|localhost",
		"BUILDOPT_POC_REMOTE_CACHE_URL="+pocRemoteCacheEndpoint(remoteURL),
		"BUILDOPT_POC_REMOTE_CACHE_PUSH="+pushValue,
	)
	command.Env = append(command.Env, extraEnvironment...)
	started := time.Now()
	output, err := command.CombinedOutput()
	finished := time.Now()
	if writeErr := os.WriteFile(filepath.Join(logs, label+".log"), output, 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	if err != nil {
		t.Fatalf("qualified composition %s failed: %v\n%s", label, err, output)
	}
	if !bytes.Contains(output, []byte("BUILD SUCCESSFUL")) {
		t.Fatalf("qualified composition %s lacks success marker:\n%s", label, output)
	}
	lines := make([]string, 0)
	cacheHits := 0
	jarFromCache := false
	outsideTask := false
	for _, line := range strings.Split(string(output), "\n") {
		if !strings.HasPrefix(line, "> Task ") {
			continue
		}
		if strings.Contains(line, ":test ") || strings.HasSuffix(line, ":test") {
			t.Fatalf("qualified composition executed a Test task: %s", line)
		}
		lines = append(lines, line)
		if strings.HasSuffix(line, " FROM-CACHE") {
			cacheHits++
		}
		if strings.HasPrefix(line, "> Task :clients:jar") && strings.HasSuffix(line, " FROM-CACHE") {
			jarFromCache = true
		}
		if strings.HasPrefix(line, "> Task :streams:jar") {
			outsideTask = true
		}
	}
	sort.Strings(lines)
	taskDigest := sha256.Sum256([]byte(strings.Join(lines, "\n") + "\n"))
	outputBytes, outputHash := hashPOCQualifiedRemoteJar(t, project)
	text := string(output)
	fullGraph := strings.Contains(text, `"selectionMode":"FULL_GRAPH"`)
	planSelected := strings.Contains(text, `"selectionMode":"POC_CANDIDATE"`)
	alternative := ""
	if strings.Contains(text, `"alternativeId":"kafka-clients-jar"`) {
		alternative = "kafka-clients-jar"
	}
	reason := ""
	if strings.Contains(text, `"selectionReason":"IMPACT_GLOBAL_CHANGE"`) {
		reason = "IMPACT_GLOBAL_CHANGE"
	}
	return pocQualifiedRemoteRun{
		Duration: finished.Sub(started).Milliseconds(), Started: started, Finished: finished,
		OutputBytes: outputBytes, OutputHash: outputHash, TaskHash: hex.EncodeToString(taskDigest[:]),
		CacheHits: cacheHits, FullGraph: fullGraph, PlanSelected: planSelected,
		Alternative: alternative, JarFromCache: jarFromCache, OutsideTask: outsideTask,
		SelectionReason: reason, Output: text,
	}
}

func assertPOCQualifiedRemotePair(t *testing.T, pair int, control, candidate pocQualifiedRemoteRun) {
	t.Helper()
	if control.OutputHash != candidate.OutputHash || control.OutputHash != pocQualifiedRemoteJarSHA {
		t.Fatalf("qualified composition pair %d output mismatch: %s/%s", pair, control.OutputHash, candidate.OutputHash)
	}
	if !control.OutsideTask || !candidate.PlanSelected || candidate.Alternative != "kafka-clients-jar" || !candidate.JarFromCache {
		t.Fatalf("qualified composition pair %d did not exercise fixed arms", pair)
	}
}

func hashPOCQualifiedRemoteJar(t *testing.T, project string) (int64, string) {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(project, "clients", "build", "libs", "kafka-clients-*.jar"))
	if err != nil {
		t.Fatal(err)
	}
	files := make([]string, 0, len(paths))
	for _, path := range paths {
		if info, statErr := os.Stat(path); statErr == nil && info.Mode().IsRegular() {
			files = append(files, path)
		}
	}
	if len(files) != 1 {
		t.Fatalf("qualified composition output count = %d, want 1", len(files))
	}
	raw, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	actual := hex.EncodeToString(digest[:])
	if actual != pocQualifiedRemoteJarSHA {
		t.Fatalf("qualified composition JAR digest = %s, want %s", actual, pocQualifiedRemoteJarSHA)
	}
	return int64(len(raw)), actual
}

func filteredPOCQualifiedRemoteEnv(values []string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if strings.HasPrefix(value, "CI=") || strings.HasPrefix(value, "BUILDOPT_") ||
			strings.HasPrefix(value, "JAVA_HOME=") || strings.HasPrefix(value, "GRADLE_USER_HOME=") ||
			strings.HasPrefix(value, "GRADLE_OPTS=") {
			continue
		}
		filtered = append(filtered, value)
	}
	return filtered
}

func requiredPOCQualifiedRemoteEnv(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Fatalf("%s is required", name)
	}
	return value
}
