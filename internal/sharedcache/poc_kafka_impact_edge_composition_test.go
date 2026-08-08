package sharedcache

import (
	"bytes"
	"context"
	"crypto/ed25519"
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

type pocKafkaImpactEdgeRun struct {
	pocQualifiedRemoteRun
	ShadowJarFromCache      bool
	StandardJarAdapter      bool
}

type pocKafkaImpactEdgeObservation struct {
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
	ControlShadowJarFromCache  bool   `json:"controlShadowJarFromCache"`
	CandidateShadowJarFromCache bool  `json:"candidateShadowJarFromCache"`
	CandidateStandardJarAdapter bool  `json:"candidateStandardJarAdapter"`
	RequiredOutputCount        int    `json:"requiredOutputCount"`
	RequiredOutputBytes        int64  `json:"requiredOutputBytes"`
	RequiredOutputSHA256       string `json:"requiredOutputSha256"`
	ControlTaskOutcomeSHA256   string `json:"controlTaskOutcomeSha256"`
	CandidateTaskOutcomeSHA256 string `json:"candidateTaskOutcomeSha256"`
	OutputsIdentical           bool   `json:"outputsIdentical"`
	ProductAttributableFailure bool   `json:"productAttributableFailure"`
}

func TestPOCKafkaImpactEdgeCompositionExperiment(t *testing.T) {
	resultPath := os.Getenv("BUILDOPT_POC_KAFKA_IMPACT_EDGE_RESULT")
	if resultPath == "" {
		t.Skip("Kafka Impact/Edge composition is not requested")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" || runtime.NumCPU() != 12 {
		t.Fatalf("Kafka Impact/Edge runner = %s/%s/%d, want linux/amd64/12", runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
	}

	seedProject := requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_KAFKA_IMPACT_EDGE_SEED_PROJECT")
	controlProject := requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_KAFKA_IMPACT_EDGE_CONTROL_PROJECT")
	candidateProject := requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_KAFKA_IMPACT_EDGE_CANDIDATE_PROJECT")
	seedHome := requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_KAFKA_IMPACT_EDGE_SEED_HOME")
	controlHome := requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_KAFKA_IMPACT_EDGE_CONTROL_HOME")
	candidateHome := requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_KAFKA_IMPACT_EDGE_CANDIDATE_HOME")
	javaHome := requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_KAFKA_IMPACT_EDGE_JAVA_HOME")
	buildoptBin := requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_KAFKA_IMPACT_EDGE_BUILDOPT_BIN")
	packagedInit := requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_KAFKA_IMPACT_EDGE_INIT_SCRIPT")
	pluginJar := requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_KAFKA_IMPACT_EDGE_PLUGIN_JAR")
	logs := requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_KAFKA_IMPACT_EDGE_LOGS")
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
	seedRun := runPOCKafkaImpactEdgeControl(t, "seed", seedProject, seedHome, javaHome, controlInit, seedServer.URL, true, logs)
	seedServer.Close()
	if seedRun.ShadowJarFromCache || !seedRun.ShadowJarExecuted || seedRun.OutputHash == "" {
		t.Fatalf("Kafka Impact/Edge seed did not produce the shaded output:\n%s", seedRun.Output)
	}
	seedOutputSHA := seedRun.OutputHash
	seedOutputBytes := seedRun.OutputBytes

	recordedMutex.Lock()
	seeded := make(map[string][]byte, len(recorded))
	var seededBytes int64
	for key, payload := range recorded {
		seeded[key] = bytes.Clone(payload)
		seededBytes += int64(len(payload))
	}
	recordedMutex.Unlock()
	if len(seeded) == 0 || seededBytes < 1<<20 {
		t.Fatalf("Kafka Impact/Edge committed entries = %d/%d", len(seeded), seededBytes)
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
			t.Fatalf("seed Kafka Impact/Edge Shared %s: %v", key, putErr)
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
	canonical := signLifecycleDecision(t, privateKey, request, "poc-kafka-impact-edge-composition-decision", objects, testRevocationEpoch, now)
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
			t.Fatalf("warm Kafka Impact/Edge %s: %v", key, readErr)
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
	controlWarmup := runPOCKafkaImpactEdgeControl(t, "warmup-control", controlProject, controlHome, javaHome, controlInit, controlServer.URL, false, logs)
	candidateWarmup := runPOCKafkaImpactEdgeCandidate(t, "warmup-candidate", candidateProject, candidateHome, javaHome, buildoptBin, candidateInit, pluginJar, edgeServer.URL, logs)
	assertPOCKafkaImpactEdgePair(t, 0, seedOutputSHA, controlWarmup, candidateWarmup)
	metrics.reset()

	orders := []string{"CONTROL_FIRST", "CANDIDATE_FIRST", "CONTROL_FIRST", "CANDIDATE_FIRST"}
	observations := make([]pocKafkaImpactEdgeObservation, 0, len(orders))
	for index, order := range orders {
		var control, candidate pocKafkaImpactEdgeRun
		var controlTraffic, candidateTraffic pocRemoteCacheMetricSnapshot
		if order == "CONTROL_FIRST" {
			metrics.reset()
			control = runPOCKafkaImpactEdgeControl(t, fmt.Sprintf("pair-%d-control", index+1), controlProject, controlHome, javaHome, controlInit, controlServer.URL, false, logs)
			controlTraffic = metrics.snapshot()
			metrics.reset()
			candidate = runPOCKafkaImpactEdgeCandidate(t, fmt.Sprintf("pair-%d-candidate", index+1), candidateProject, candidateHome, javaHome, buildoptBin, candidateInit, pluginJar, edgeServer.URL, logs)
			candidateTraffic = metrics.snapshot()
		} else {
			metrics.reset()
			candidate = runPOCKafkaImpactEdgeCandidate(t, fmt.Sprintf("pair-%d-candidate", index+1), candidateProject, candidateHome, javaHome, buildoptBin, candidateInit, pluginJar, edgeServer.URL, logs)
			candidateTraffic = metrics.snapshot()
			metrics.reset()
			control = runPOCKafkaImpactEdgeControl(t, fmt.Sprintf("pair-%d-control", index+1), controlProject, controlHome, javaHome, controlInit, controlServer.URL, false, logs)
			controlTraffic = metrics.snapshot()
		}
		gap := candidate.Started.Sub(control.Finished)
		if order == "CANDIDATE_FIRST" {
			gap = control.Started.Sub(candidate.Finished)
		}
		if gap < 0 || gap > 15*time.Second {
			t.Fatalf("Kafka Impact/Edge pair %d gap = %v", index+1, gap)
		}
		assertPOCKafkaImpactEdgePair(t, index+1, seedOutputSHA, control, candidate)
		if controlTraffic.Requests == 0 || controlTraffic.Bytes == 0 {
			t.Fatalf("Kafka Impact/Edge pair %d control Shared traffic = %+v", index+1, controlTraffic)
		}
		if candidateTraffic.Requests != 0 || candidateTraffic.Bytes != 0 {
			t.Fatalf("Kafka Impact/Edge pair %d Edge fetched from Shared = %+v", index+1, candidateTraffic)
		}
		observations = append(observations, pocKafkaImpactEdgeObservation{
			Pair: index + 1, Order: order,
			ControlDurationMs: control.Duration, CandidateDurationMs: candidate.Duration,
			SavedMs: control.Duration - candidate.Duration, InterArmIdleGapMs: gap.Milliseconds(),
			ControlOriginRequests: controlTraffic.Requests, ControlOriginBytes: controlTraffic.Bytes,
			CandidateOriginRequests: candidateTraffic.Requests, CandidateOriginBytes: candidateTraffic.Bytes,
			ControlRemoteCacheHits: control.CacheHits, CandidateRemoteCacheHits: candidate.CacheHits,
			ControlFullGraph: control.FullGraph, CandidatePlanSelected: candidate.PlanSelected,
			CandidateAlternative: candidate.Alternative,
			ControlShadowJarFromCache: control.ShadowJarFromCache,
			CandidateShadowJarFromCache: candidate.ShadowJarFromCache,
			CandidateStandardJarAdapter: candidate.StandardJarAdapter,
			RequiredOutputCount: 1, RequiredOutputBytes: control.OutputBytes,
			RequiredOutputSHA256: control.OutputHash,
			ControlTaskOutcomeSHA256: control.TaskHash, CandidateTaskOutcomeSHA256: candidate.TaskHash,
			OutputsIdentical: true, ProductAttributableFailure: false,
		})
		t.Logf("Kafka Impact/Edge pair %d/4: native=%dms candidate=%dms saved=%dms order=%s", index+1, control.Duration, candidate.Duration, control.Duration-candidate.Duration, order)
	}

	if err := os.WriteFile(filepath.Join(candidateProject, ".buildopt-changes"), []byte("gradle.properties\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	impactFallback := runPOCKafkaImpactEdgeCandidate(t, "fallback-impact", candidateProject, candidateHome, javaHome, buildoptBin, candidateInit, pluginJar, edgeServer.URL, logs)
	impactFallbackPassed := impactFallback.FullGraph && impactFallback.SelectionReason == "IMPACT_GLOBAL_CHANGE" && impactFallback.OutsideTask && impactFallback.OutputHash == seedOutputSHA
	if !impactFallbackPassed {
		t.Fatalf("Kafka Impact/Edge impact fallback failed:\n%s", impactFallback.Output)
	}
	changePath := "clients/src/main/java/org/apache/kafka/clients/Metadata.java"
	if err := os.WriteFile(filepath.Join(candidateProject, ".buildopt-changes"), []byte(changePath+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	failingEdge := httptest.NewServer(pocRemoteCacheBasicAuth(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusServiceUnavailable)
	})))
	edgeFailure := runPOCKafkaImpactEdgeCandidate(t, "fallback-edge", candidateProject, candidateHome, javaHome, buildoptBin, candidateInit, pluginJar, failingEdge.URL, logs)
	failingEdge.Close()
	edgeFailureFallbackPassed := edgeFailure.PlanSelected && !edgeFailure.StandardJarAdapter && edgeFailure.OutputHash == seedOutputSHA
	if !edgeFailureFallbackPassed {
		t.Fatalf("Kafka Impact/Edge failure fallback failed:\n%s", edgeFailure.Output)
	}

	summary := make([]pocRemoteCacheObservation, len(observations))
	for index, observation := range observations {
		summary[index] = pocRemoteCacheObservation{ControlDurationMs: observation.ControlDurationMs, CandidateDurationMs: observation.CandidateDurationMs, SavedMs: observation.SavedMs}
	}
	controlMean, candidateMean, savedMean, ratio, interval, positive, qualified := summarizePOCRemoteCache(summary)
	qualified = qualified && impactFallbackPassed && edgeFailureFallbackPassed
	decisionName := "RETAIN_SEPARATE_KAFKA_IMPACT_AND_EDGE"
	if qualified {
		decisionName = "QUALIFY_KAFKA_IMPACT_EDGE_COMPOSITION"
	}
	result := map[string]any{
		"schemaVersion": "buildopt.evidence/poc-kafka-impact-edge-composition/v1",
		"workItem": "POC-KAFKA-IMPACT-EDGE-COMPOSITION-001",
		"capturedAt": time.Now().UTC().Truncate(time.Second).Format(time.RFC3339),
		"buildoptRevision": requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_KAFKA_IMPACT_EDGE_REVISION"),
		"runnerScriptSha256": requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_KAFKA_IMPACT_EDGE_RUNNER_SHA"),
		"specSha256": requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_KAFKA_IMPACT_EDGE_SPEC_SHA"),
		"harnessSha256": requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_KAFKA_IMPACT_EDGE_HARNESS_SHA"),
		"sourceArchiveSha256": requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_KAFKA_IMPACT_EDGE_SOURCE_SHA"),
		"dependencyCacheSha256": requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_KAFKA_IMPACT_EDGE_DEPENDENCY_SHA"),
		"installedAssets": map[string]any{
			"binarySha256": requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_KAFKA_IMPACT_EDGE_BINARY_SHA"),
			"initScriptSha256": requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_KAFKA_IMPACT_EDGE_INIT_SHA"),
			"pluginJarSha256": requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_KAFKA_IMPACT_EDGE_PLUGIN_SHA"),
		},
		"componentEvidence": map[string]any{
			"kafkaImpactScopeSha256": requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_KAFKA_IMPACT_EDGE_IMPACT_SHA"),
			"kafkaEdgeLocalitySha256": requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_KAFKA_IMPACT_EDGE_EDGE_SHA"),
			"invalidJarCompositionSha256": requiredPOCQualifiedRemoteEnv(t, "BUILDOPT_POC_KAFKA_IMPACT_EDGE_INVALID_SHA"),
		},
		"repository": map[string]any{"nameWithOwner": "apache/kafka", "releaseTag": "4.3.1", "revision": "26b251a451ce941d3d7a55e6487bcb7f16b5ad48", "gradleVersion": "9.2.1", "jdk": "temurin-25.0.3+9"},
		"runner": map[string]any{"id": "linux-amd64-12c-16659865600b-v1", "cpuCount": 12, "memoryBytes": int64(16659865600), "maxWorkers": 12},
		"network": map[string]any{"model": "INDEPENDENT_SOURCE_ARCHIVE_DERIVED_LOOPBACK_WAN", "latencyPerResponseMs": 337, "bandwidthBytesPerSecond": 6994831, "packetLossRatio": 0},
		"preparation": map[string]any{
			"measured": false, "dependenciesResolvedBeforeMeasurement": true,
			"externalDependencyNetworkBlocked": true, "seededObjectCount": len(seeded),
			"seededObjectBytes": seededBytes, "seedOutputBytes": seedOutputBytes,
			"seedOutputSha256": seedOutputSHA, "seedOutputProducingTask": ":clients:shadowJar",
			"edgeWarmupOriginRequests": edgeWarmup.Requests, "edgeWarmupOriginBytes": edgeWarmup.Bytes,
		},
		"mechanisms": map[string]any{
			"buildImpact": true, "edgeLocality": true, "standardJarAdapter": false,
			"standardCopyAdapter": false, "safeCache": false, "runtimeTuning": false,
			"hotState": false, "testOptimization": false,
		},
		"observations": observations,
		"safety": map[string]any{"impactFallbackPassed": impactFallbackPassed, "edgeFailureFallbackPassed": edgeFailureFallbackPassed},
		"result": map[string]any{"pairs": 4, "controlMeanMs": controlMean, "candidateMeanMs": candidateMean, "meanSavedMs": savedMean, "reductionRatio": ratio, "interval95SavedMs": interval, "positivePairs": positive, "qualified": qualified, "decision": decisionName},
		"boundaries": map[string]any{"sameSharedOrigin": true, "sameCommittedObjectBytes": true, "sameSourceAndChange": true, "sameRequiredOutput": true, "testTasksForbidden": true, "runtimeTuningChanged": false, "safeCacheChanged": false, "hotStateChanged": false, "standardJarChanged": false, "standardCopyChanged": false, "testSelectionChanged": false, "testExecutionChanged": false, "testOptimizationModified": false, "proofOfConcept": true, "productionReadinessClaimed": false, "soakRequired": false, "designPartnerRequired": false, "universalSavingsClaimed": false},
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(resultPath, append(encoded, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func runPOCKafkaImpactEdgeControl(t *testing.T, label, project, home, javaHome, initScript, remoteURL string, push bool, logs string) pocKafkaImpactEdgeRun {
	t.Helper()
	removePOCRemoteCacheTransferOutputs(t, project)
	command := exec.Command(filepath.Join(project, "gradlew"), "--daemon", "--build-cache", "--no-configuration-cache", "--parallel", "--console=plain", "--max-workers=12", "--no-scan", "--init-script", initScript, "assemble")
	run := executePOCQualifiedRemote(t, label, command, project, home, javaHome, remoteURL, push, logs)
	run.FullGraph = true
	return pocKafkaImpactEdgeRun{pocQualifiedRemoteRun: run, ShadowJarFromCache: strings.Contains(run.Output, "> Task :clients:shadowJar FROM-CACHE")}
}

func runPOCKafkaImpactEdgeCandidate(t *testing.T, label, project, home, javaHome, buildoptBin, initScript, pluginJar, remoteURL, logs string) pocKafkaImpactEdgeRun {
	t.Helper()
	removePOCRemoteCacheTransferOutputs(t, project)
	args := []string{
		"impact", "--repository-id", "apache/kafka", "--pipeline-class", "poc-kafka-packaging-v1",
		"--changes-file", ".buildopt-changes", "--manifest", "buildopt-impact-manifest.json",
		"--graph", "buildopt-impact-graph.generated.json", "--generated-manifest", "buildopt-impact.generated.json",
		"--gradle-option=--daemon", "--gradle-option=--build-cache",
		"--gradle-option=--no-configuration-cache", "--gradle-option=--parallel",
		"--gradle-option=--console=plain", "--gradle-option=--max-workers=12", "--gradle-option=--no-scan",
	}
	command := exec.Command(buildoptBin, args...)
	run := executePOCQualifiedRemote(t, label, command, project, home, javaHome, remoteURL, false, logs,
		"BUILDOPT_GRADLE_INIT_SCRIPT="+initScript,
		"BUILDOPT_GRADLE_PLUGIN_JAR="+pluginJar,
	)
	if strings.Contains(run.Output, "retained the full graph (IMPACT_GLOBAL_CHANGE)") {
		run.FullGraph = true
		run.SelectionReason = "IMPACT_GLOBAL_CHANGE"
	}
	return pocKafkaImpactEdgeRun{
		pocQualifiedRemoteRun: run,
		ShadowJarFromCache: strings.Contains(run.Output, "> Task :clients:shadowJar FROM-CACHE"),
		StandardJarAdapter: strings.Contains(run.Output, "BUILDOPT_CACHE_STANDARD_JAR_PRODUCERS=1"),
	}
}

func assertPOCKafkaImpactEdgePair(t *testing.T, pair int, seedOutputSHA string, control, candidate pocKafkaImpactEdgeRun) {
	t.Helper()
	if control.OutputHash != candidate.OutputHash || control.OutputHash != seedOutputSHA {
		t.Fatalf("Kafka Impact/Edge pair %d output mismatch: seed=%s control=%s candidate=%s", pair, seedOutputSHA, control.OutputHash, candidate.OutputHash)
	}
	if !control.FullGraph || !control.OutsideTask || !control.ShadowJarFromCache ||
		!candidate.PlanSelected || candidate.Alternative != "kafka-clients-jar" ||
		!candidate.ShadowJarFromCache || candidate.StandardJarAdapter {
		t.Fatalf("Kafka Impact/Edge pair %d did not exercise the fixed arms", pair)
	}
}
