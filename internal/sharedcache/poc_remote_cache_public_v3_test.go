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
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/edgecache"
	"github.com/tonyredondo/buildopt/internal/localauthority"
)

type rcl3PublicSubject struct {
	Family            string   `json:"family"`
	Revision          string   `json:"revision"`
	Checkout          string   `json:"checkout"`
	Workflow          []string `json:"workflow"`
	OutputInclude     string   `json:"outputInclude"`
	OutputExclude     []string `json:"outputExclude"`
	InitScript        string   `json:"initScript"`
	JavaInstallations string   `json:"javaInstallations"`
}

type rcl3PublicBuild struct {
	Succeeded      bool     `json:"succeeded"`
	PrimaryPath    string   `json:"primaryPath"`
	PrimaryBytes   int64    `json:"primaryBytes"`
	PrimarySHA256  string   `json:"primarySha256"`
	TaskNames      []string `json:"taskNames"`
	FromCacheCount int      `json:"fromCacheCount"`
}

func TestRCL3PublicCorrectness(t *testing.T) {
	resultPath := os.Getenv("BUILDOPT_RCL3_RESULT")
	if resultPath == "" {
		t.Skip("RCL3 public correctness is not requested")
	}
	var subject rcl3PublicSubject
	if err := json.Unmarshal([]byte(requiredPOCRemoteCacheEnv(t, "BUILDOPT_RCL3_SUBJECT")), &subject); err != nil {
		t.Fatalf("decode RCL3 subject: %v", err)
	}
	if subject.Family == "" || len(subject.Workflow) == 0 || !filepath.IsAbs(subject.Checkout) || !filepath.IsAbs(subject.InitScript) {
		t.Fatal("RCL3 subject is incomplete")
	}

	root := t.TempDir()
	recorded := make(map[string][]byte)
	seedServer := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		key, ok := pocRemoteCacheKey(request.URL.Path)
		if !ok {
			response.WriteHeader(http.StatusNotFound)
			return
		}
		switch request.Method {
		case http.MethodGet:
			payload, found := recorded[key]
			if !found {
				response.WriteHeader(http.StatusNotFound)
				return
			}
			response.Header().Set("Content-Length", fmt.Sprint(len(payload)))
			_, _ = response.Write(payload)
		case http.MethodPut:
			payload, err := io.ReadAll(request.Body)
			if err != nil {
				response.WriteHeader(http.StatusBadRequest)
				return
			}
			recorded[key] = bytes.Clone(payload)
			response.WriteHeader(http.StatusOK)
		default:
			response.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	producerA := runRCL3PublicGradle(t, subject, filepath.Join(root, "producer-a-home"), pocRemoteCacheEndpoint(seedServer.URL), true, true)
	seedServer.Close()
	producerB := runRCL3PublicGradle(t, subject, filepath.Join(root, "producer-b-home"), "", false, false)

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
	keys := make([]string, 0, len(recorded))
	for key := range recorded {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	objects := make([]CommitObject, 0, len(keys))
	var objectBytes int64
	objectHasher := sha256.New()
	for _, key := range keys {
		payload := recorded[key]
		pending, putErr := storage.PutPending(ctx, binding.AttemptID, key, bytes.NewReader(payload))
		if putErr != nil {
			t.Fatalf("seed Shared %s: %v", key, putErr)
		}
		objects = append(objects, pending.Object)
		objectBytes += int64(len(payload))
		_, _ = objectHasher.Write([]byte(key))
		_, _ = objectHasher.Write([]byte{0})
		digest := sha256.Sum256(payload)
		_, _ = objectHasher.Write(digest[:])
	}
	if len(objects) > 0 {
		status, statusErr := storage.AttemptStatus(ctx, binding.AttemptID)
		if statusErr != nil {
			t.Fatal(statusErr)
		}
		request := StartAttemptRequest{AttemptID: status.AttemptID, AuthorityDigest: status.AuthorityDigest, Repository: status.Repository, NamespaceGeneration: status.NamespaceGeneration, SourceRevision: status.SourceRevision, SourceStateDigest: status.SourceStateDigest, PolicyDigest: status.PolicyDigest, ConfigurationPolicyDigest: status.ConfigurationPolicyDigest, CacheContractDigest: status.CacheContractDigest, OwnerID: status.OwnerID, LeaseID: status.LeaseID, LeaseExpiresAt: status.LeaseExpiresAt}
		canonical := signLifecycleDecision(t, privateKey, request, "rcl3-public-correctness", objects, testRevocationEpoch, now)
		decision, verifyErr := VerifyCommitDecision(ctx, canonical, map[string]ed25519.PublicKey{testDecisionKeyID: privateKey.Public().(ed25519.PublicKey)}, testRevocationEpoch, now)
		if verifyErr != nil {
			t.Fatal(verifyErr)
		}
		if _, commitErr := storage.CommitAttempt(ctx, status.StateVersion, testRevocationEpoch, decision); commitErr != nil {
			t.Fatal(commitErr)
		}
	}
	sharedHandler, err := NewBetaTokenHTTPHandler(storage, binding, BetaTokenPlaneStable)
	if err != nil {
		t.Fatal(err)
	}
	token := issueBetaTokenForTest(t, storage, betaScopeForBinding(binding, BetaTokenPlaneStable), BetaTokenRead, now, now.Add(30*time.Minute))
	metrics := &pocRemoteCacheMetrics{}
	shaped := pocRemoteCacheShapeWithProfile(metrics, pocRemoteCacheUpstreamAuth(sharedHandler, token.Token, binding.AuthorityDigest), 30*time.Millisecond, 100<<20)
	sharedServer := httptest.NewServer(shaped)
	defer sharedServer.Close()
	directServer := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		clone := request.Clone(request.Context())
		clone.Header = request.Header.Clone()
		clone.Header.Set("Authorization", "Bearer "+token.Token)
		clone.Header.Set(AuthorityDigestHeader, binding.AuthorityDigest)
		shaped.ServeHTTP(response, clone)
	}))
	defer directServer.Close()
	direct := runRCL3PublicGradle(t, subject, filepath.Join(root, "direct-home"), pocRemoteCacheEndpoint(directServer.URL), false, true)
	directTraffic := metrics.snapshot()

	edgeConfig := edgecache.Config{Shared: edgecache.Shared{BaseURL: sharedServer.URL, AllowInsecureLoopback: true}, Storage: edgecache.Storage{StateDirectory: filepath.Join(root, "edge"), FilesystemPolicy: edgecache.FilesystemPolicy, CapacityBytes: edgecache.MinimumCapacityBytes, MaximumObjectBytes: edgecache.MaximumObjectBytes, StableTTLSeconds: int64(edgecache.MaximumStableTTL / time.Second), PendingTTLSeconds: int64(edgecache.MaximumPendingTTL / time.Second), HighWatermarkPercent: edgecache.HighWatermarkPercent, LowWatermarkPercent: edgecache.LowWatermarkPercent, ProtectedPercent: edgecache.ProtectedPercent}}
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
	metrics.reset()
	proxy, err := edgecache.NewProxy(edgeStore, edgeClient, readAuthority, nil, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	edgeServer := httptest.NewServer(proxy.Handler())
	defer edgeServer.Close()
	edge := runRCL3PublicGradle(t, subject, filepath.Join(root, "edge-home"), pocRemoteCacheEndpoint(edgeServer.URL), false, true)
	edgeTraffic := metrics.snapshot()

	nativeStable := producerA.PrimarySHA256 == producerB.PrimarySHA256
	directExact := nativeStable && direct.PrimarySHA256 == producerA.PrimarySHA256
	edgeExact := nativeStable && edge.PrimarySHA256 == producerA.PrimarySHA256
	decisionName := "ELIGIBLE_EXACT"
	if !nativeStable {
		decisionName = "NATIVE_OUTPUT_UNSTABLE"
	} else if !directExact {
		decisionName = "DIRECT_CACHE_OUTPUT_MISMATCH"
	} else if !edgeExact || edgeTraffic.Requests != 0 {
		decisionName = "EDGE_PRODUCT_FAILURE"
	} else if direct.FromCacheCount == 0 || edge.FromCacheCount == 0 {
		decisionName = "INCOMPLETE_NO_CACHE_HITS"
	}
	result := map[string]any{
		"schemaVersion": "buildopt.evidence/remote-cache-locality-public-row/v3", "family": subject.Family, "revision": subject.Revision,
		"workflow": subject.Workflow, "producerA": producerA, "producerB": producerB, "directConsumer": direct, "edgeConsumer": edge,
		"remoteManifest": map[string]any{"objectCount": len(keys), "objectBytes": objectBytes, "sha256": hex.EncodeToString(objectHasher.Sum(nil))},
		"directOrigin":   directTraffic, "edgeConsumerOrigin": edgeTraffic, "nativeStable": nativeStable, "directExact": directExact, "edgeExact": edgeExact,
		"productAttributableFailure": decisionName == "EDGE_PRODUCT_FAILURE", "decision": decisionName,
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(resultPath, append(encoded, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func runRCL3PublicGradle(t *testing.T, subject rcl3PublicSubject, home, remoteURL string, push, buildCache bool) rcl3PublicBuild {
	t.Helper()
	t.Logf("%s: clean then build (cache=%t push=%t)", subject.Family, buildCache, push)
	prepareRCL3GradleHome(t, home)
	gradle := filepath.Join(subject.Checkout, "gradlew")
	toolchains := "-Dorg.gradle.java.installations.paths=" + subject.JavaInstallations
	clean := exec.Command(gradle, toolchains, "--no-daemon", "--no-build-cache", "--no-configuration-cache", "--console=plain", "--max-workers=4", "clean")
	clean.Dir = subject.Checkout
	clean.Env = append(os.Environ(), "GRADLE_USER_HOME="+home)
	if output, err := clean.CombinedOutput(); err != nil {
		t.Fatalf("%s clean failed: %v\n%s", subject.Family, err, output)
	}
	args := []string{toolchains, "--no-daemon", "--no-configuration-cache", "--console=plain", "--max-workers=4"}
	if buildCache {
		args = append(args, "--build-cache", "--init-script", subject.InitScript)
	} else {
		args = append(args, "--no-build-cache")
	}
	args = append(args, subject.Workflow...)
	command := exec.Command(gradle, args...)
	command.Dir = subject.Checkout
	command.Env = append(os.Environ(), "GRADLE_USER_HOME="+home, "BUILDOPT_RCL_CACHE_URL="+remoteURL, fmt.Sprintf("BUILDOPT_RCL_CACHE_PUSH=%d", boolInt(push)))
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s build failed: %v\n%s", subject.Family, err, output)
	}
	if !bytes.Contains(output, []byte("BUILD SUCCESSFUL")) {
		t.Fatalf("%s build lacks success marker\n%s", subject.Family, output)
	}
	return inspectRCL3PublicBuild(t, subject, output)
}

func prepareRCL3GradleHome(t *testing.T, home string) {
	t.Helper()
	sharedDistributions := filepath.Join(filepath.Dir(home), "wrapper-distributions")
	sharedModules := filepath.Join(filepath.Dir(home), "dependency-modules")
	if err := os.MkdirAll(sharedDistributions, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sharedModules, 0o700); err != nil {
		t.Fatal(err)
	}
	wrapper := filepath.Join(home, "wrapper")
	if err := os.MkdirAll(wrapper, 0o700); err != nil {
		t.Fatal(err)
	}
	distributions := filepath.Join(wrapper, "dists")
	if _, err := os.Lstat(distributions); os.IsNotExist(err) {
		if err := os.Symlink(sharedDistributions, distributions); err != nil {
			t.Fatal(err)
		}
	} else if err != nil {
		t.Fatal(err)
	}
	caches := filepath.Join(home, "caches")
	if err := os.MkdirAll(caches, 0o700); err != nil {
		t.Fatal(err)
	}
	modules := filepath.Join(caches, "modules-2")
	if _, err := os.Lstat(modules); os.IsNotExist(err) {
		if err := os.Symlink(sharedModules, modules); err != nil {
			t.Fatal(err)
		}
	} else if err != nil {
		t.Fatal(err)
	}
}

func inspectRCL3PublicBuild(t *testing.T, subject rcl3PublicSubject, output []byte) rcl3PublicBuild {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(subject.Checkout, subject.OutputInclude))
	if err != nil {
		t.Fatal(err)
	}
	selected := make([]string, 0, len(matches))
	for _, path := range matches {
		excluded := false
		for _, pattern := range subject.OutputExclude {
			match, matchErr := filepath.Match(pattern, filepath.Base(path))
			if matchErr != nil {
				t.Fatal(matchErr)
			}
			excluded = excluded || match
		}
		if !excluded {
			selected = append(selected, path)
		}
	}
	if len(selected) != 1 {
		t.Fatalf("%s primary output matches = %v", subject.Family, selected)
	}
	payload, err := os.ReadFile(selected[0])
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(payload)
	tasks := make([]string, 0)
	fromCache := 0
	for _, line := range strings.Split(string(output), "\n") {
		if !strings.HasPrefix(line, "> Task ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			tasks = append(tasks, fields[2])
		}
		if strings.HasSuffix(line, " FROM-CACHE") {
			fromCache++
		}
	}
	sort.Strings(tasks)
	relative, err := filepath.Rel(subject.Checkout, selected[0])
	if err != nil {
		t.Fatal(err)
	}
	return rcl3PublicBuild{Succeeded: true, PrimaryPath: filepath.ToSlash(relative), PrimaryBytes: int64(len(payload)), PrimarySHA256: hex.EncodeToString(digest[:]), TaskNames: tasks, FromCacheCount: fromCache}
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
