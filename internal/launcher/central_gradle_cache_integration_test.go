package launcher

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
	"github.com/tonyredondo/buildopt/internal/localauthority"
	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

const centralGradleDecisionKeyID = "central-cache-owner"

const (
	centralGradleFixtureUsername = "buildopt-poc"
	centralGradleFixturePassword = "remote-cache-value"
)

func TestCentralGradleCacheProducerConsumerAndOutage(t *testing.T) {
	fixture := os.Getenv("BUILDOPT_CENTRAL_GRADLE_CACHE_FIXTURE")
	gradle := os.Getenv("BUILDOPT_CENTRAL_GRADLE_CACHE_GRADLE")
	if fixture == "" || gradle == "" {
		t.Skip("central Gradle-cache integration is not requested")
	}
	for name, path := range map[string]string{"fixture": fixture, "Gradle": gradle} {
		if !filepath.IsAbs(path) {
			t.Fatalf("%s path is not absolute: %q", name, path)
		}
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	authorityEnvironment, authorityDocument := writeLauncherAuthorityFixtureAt(
		t,
		"https://central.invalid",
		now,
	)
	verified, _, localCredential, err := localauthority.LoadFiles(
		context.Background(),
		authorityEnvironment[localAuthorityPathEnvironment],
		authorityEnvironment[localTrustRootPathEnvironment],
		authorityEnvironment[localCredentialPathEnvironment],
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer clearBytes(localCredential)

	storage, err := sharedcache.Open(
		context.Background(),
		filepath.Join(t.TempDir(), "central-storage"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	binding, _, err := storage.InstallLocalAuthority(
		context.Background(),
		verified,
		localCredential,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}

	scope := sharedcache.CentralTokenScope{
		RepositoryScopeSHA256: centralGradleRepositoryScope(authorityDocument),
		Tenant:                authorityDocument.Repository.Tenant,
		Repository:            authorityDocument.Repository.Repository,
		TrustDomain:           authorityDocument.Repository.TrustDomain,
		Namespace:             authorityDocument.Policy.RemoteCache.Namespace,
		NamespaceGeneration:   authorityDocument.Policy.RemoteCache.NamespaceGeneration,
	}
	producerToken, err := storage.IssueCentralToken(
		context.Background(),
		sharedcache.CentralTokenIssueRequest{
			Scope: scope,
			Capabilities: []sharedcache.CentralCapability{
				sharedcache.CentralCacheWrite,
			},
			ExpiresAt: now.Add(time.Hour),
		},
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	consumerToken, err := storage.IssueCentralToken(
		context.Background(),
		sharedcache.CentralTokenIssueRequest{
			Scope: scope,
			Capabilities: []sharedcache.CentralCapability{
				sharedcache.CentralCacheRead,
			},
			ExpiresAt: now.Add(time.Hour),
		},
		now,
	)
	if err != nil {
		t.Fatal(err)
	}

	centralHandler, err := sharedcache.NewCentralHTTPSHandler(storage)
	if err != nil {
		t.Fatal(err)
	}
	capture := newCentralGradleCapture(centralHandler, scope.NamespaceGeneration)
	server := httptest.NewUnstartedServer(capture)
	server.TLS = &tls.Config{MinVersion: tls.VersionTLS13}
	server.StartTLS()
	t.Cleanup(server.Close)

	producer := startCentralGradleGateway(
		t,
		server,
		producerToken.Token,
		binding.AuthorityDigest,
		binding.AttemptID,
		scope.Namespace,
		false,
		true,
		producerToken.ExpiresAt,
	)
	producerProject := copyCentralGradleFixture(t, fixture, "producer")
	producerRun := runCentralGradleFixture(t, gradle, producerProject, producer, true)
	if strings.Contains(producerRun, " FROM-CACHE") {
		t.Fatalf("clean producer unexpectedly reused central state\n%s", producerRun)
	}
	producerDigest := centralGradleOutputsDigest(t, producerProject)
	objects := capture.pendingObjects()
	status, err := storage.AttemptStatus(context.Background(), binding.AttemptID)
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) < 8 || status.PendingObjectCount != len(objects) {
		t.Fatalf(
			"central producer objects = %d/%d with %d PUTs, want at least eight exact pending objects\n%s",
			len(objects),
			status.PendingObjectCount,
			capture.putCount(),
			producerRun,
		)
	}
	commitCentralGradleAttempt(
		t,
		storage,
		status,
		objects,
		time.Now().UTC().Truncate(time.Millisecond),
	)
	if err := producer.close(); err != nil {
		t.Fatal(err)
	}

	consumer := startCentralGradleGateway(
		t,
		server,
		consumerToken.Token,
		binding.AuthorityDigest,
		binding.AttemptID,
		scope.Namespace,
		true,
		false,
		consumerToken.ExpiresAt,
	)
	consumerProject := copyCentralGradleFixture(t, fixture, "consumer")
	consumerRun := runCentralGradleFixture(t, gradle, consumerProject, consumer, false)
	if hits := strings.Count(consumerRun, " FROM-CACHE"); hits < 8 {
		t.Fatalf("central consumer hits = %d, want at least eight\n%s", hits, consumerRun)
	}
	if consumerDigest := centralGradleOutputsDigest(t, consumerProject); consumerDigest != producerDigest {
		t.Fatalf("consumer outputs = %s, want %s", consumerDigest, producerDigest)
	}
	putsBefore := capture.putCount()
	statusCode, _, _, err := requestLocalGateway(
		consumer.endpoint,
		consumer.username,
		consumer.password,
		http.MethodPut,
		"/cache/readonly-probe",
	)
	if err != nil || statusCode != http.StatusServiceUnavailable || capture.putCount() != putsBefore {
		t.Fatalf("read-only publication = %d/%d/%v", statusCode, capture.putCount()-putsBefore, err)
	}
	if strings.Contains(consumerRun, consumerToken.Token) ||
		strings.Contains(consumerRun, producerToken.Token) {
		t.Fatal("central token escaped into Gradle output")
	}

	server.Close()
	outageProject := copyCentralGradleFixture(t, fixture, "outage")
	outageRun := runCentralGradleFixture(t, gradle, outageProject, consumer, false)
	if strings.Contains(outageRun, " FROM-CACHE") {
		t.Fatalf("central outage unexpectedly reported a remote hit\n%s", outageRun)
	}
	if outageDigest := centralGradleOutputsDigest(t, outageProject); outageDigest != producerDigest {
		t.Fatalf("outage outputs = %s, want %s", outageDigest, producerDigest)
	}
	if err := consumer.close(); err != nil {
		t.Fatal(err)
	}
	if capture.getCount() < 8 {
		t.Fatalf("central consumer GETs = %d, want at least eight", capture.getCount())
	}
}

type centralGradleCapture struct {
	handler             http.Handler
	namespaceGeneration int64

	mutex   sync.Mutex
	objects map[string]sharedcache.CommitObject
	gets    int
	puts    int
}

func newCentralGradleCapture(handler http.Handler, generation int64) *centralGradleCapture {
	return &centralGradleCapture{
		handler: handler, namespaceGeneration: generation,
		objects: make(map[string]sharedcache.CommitObject),
	}
}

func (capture *centralGradleCapture) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	recorder := &centralGradleResponseWriter{ResponseWriter: writer}
	capture.handler.ServeHTTP(recorder, request)
	if !strings.HasPrefix(request.URL.Path, "/cache/") {
		return
	}
	capture.mutex.Lock()
	defer capture.mutex.Unlock()
	switch request.Method {
	case http.MethodGet:
		capture.gets++
	case http.MethodPut:
		capture.puts++
		if recorder.status >= 200 && recorder.status < 300 {
			key := strings.TrimPrefix(request.URL.Path, "/cache/")
			capture.objects[key] = sharedcache.CommitObject{
				NamespaceGeneration: capture.namespaceGeneration,
				Key:                 key,
				Checksum:            recorder.Header().Get("X-BuildOpt-Blob-Digest"),
				SizeBytes:           request.ContentLength,
			}
		}
	}
}

func (capture *centralGradleCapture) pendingObjects() []sharedcache.CommitObject {
	capture.mutex.Lock()
	defer capture.mutex.Unlock()
	objects := make([]sharedcache.CommitObject, 0, len(capture.objects))
	for _, object := range capture.objects {
		objects = append(objects, object)
	}
	sort.Slice(objects, func(left, right int) bool {
		return objects[left].Key < objects[right].Key
	})
	return objects
}

func (capture *centralGradleCapture) getCount() int {
	capture.mutex.Lock()
	defer capture.mutex.Unlock()
	return capture.gets
}

func (capture *centralGradleCapture) putCount() int {
	capture.mutex.Lock()
	defer capture.mutex.Unlock()
	return capture.puts
}

type centralGradleResponseWriter struct {
	http.ResponseWriter
	status int
}

func (writer *centralGradleResponseWriter) WriteHeader(status int) {
	writer.status = status
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *centralGradleResponseWriter) Write(content []byte) (int, error) {
	if writer.status == 0 {
		writer.status = http.StatusOK
	}
	return writer.ResponseWriter.Write(content)
}

func startCentralGradleGateway(
	t *testing.T,
	server *httptest.Server,
	token string,
	authorityDigest string,
	attemptID string,
	namespace string,
	allowRead bool,
	allowWrite bool,
	expiresAt time.Time,
) *localGateway {
	t.Helper()
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := newGatewayCacheBinding(
		server.URL,
		raw,
		authorityDigest,
		attemptID,
		namespace,
		allowRead,
		allowWrite,
		expiresAt,
	)
	clearBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	gateway, err := startLocalGatewayWithCache(binding)
	if err != nil {
		t.Fatal(err)
	}
	client := server.Client()
	client.Timeout = gatewayOperationTimeout
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	gateway.cacheClient = client
	gateway.username = centralGradleFixtureUsername
	gateway.password = centralGradleFixturePassword
	t.Cleanup(func() {
		_ = gateway.close()
	})
	return gateway
}

func runCentralGradleFixture(
	t *testing.T,
	gradle string,
	project string,
	gateway *localGateway,
	push bool,
) string {
	t.Helper()
	pushValue := "0"
	if push {
		pushValue = "1"
	}
	command := exec.Command(
		gradle,
		"--no-daemon",
		"--no-configuration-cache",
		"--build-cache",
		"--console=plain",
		"remoteCacheFixture",
	)
	command.Dir = project
	command.Env = replaceEnvironment(os.Environ(), map[string]string{
		"BUILDOPT_POC_REMOTE_CACHE_URL":  gateway.endpoint + "/cache/",
		"BUILDOPT_POC_REMOTE_CACHE_PUSH": pushValue,
		"GRADLE_USER_HOME":               filepath.Join(project, ".gradle-user-home"),
	})
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("central Gradle fixture failed: %v\n%s", err, output)
	}
	return string(output)
}

func copyCentralGradleFixture(t *testing.T, source string, name string) string {
	t.Helper()
	target := filepath.Join(t.TempDir(), name)
	if err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return os.Mkdir(target, 0o700)
		}
		if entry.IsDir() && (entry.Name() == "build" || entry.Name() == ".gradle") {
			return filepath.SkipDir
		}
		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.Mkdir(destination, 0o700)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, content, 0o600)
	}); err != nil {
		t.Fatal(err)
	}
	return target
}

func centralGradleOutputsDigest(t *testing.T, project string) string {
	t.Helper()
	root := filepath.Join(project, "build", "remote-cache")
	var paths []string
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	digest := sha256.New()
	for _, path := range paths {
		relative, err := filepath.Rel(root, path)
		if err != nil {
			t.Fatal(err)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.WriteString(digest, relative)
		_, _ = digest.Write([]byte{0})
		_, _ = digest.Write(content)
	}
	if len(paths) != 8 {
		t.Fatalf("central Gradle outputs = %d, want 8", len(paths))
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func centralGradleRepositoryScope(document localauthority.Document) string {
	digest := sha256.Sum256([]byte(
		document.Repository.Tenant + "\x00" +
			document.Repository.Repository + "\x00" +
			document.Repository.TrustDomain,
	))
	return hex.EncodeToString(digest[:])
}

func commitCentralGradleAttempt(
	t *testing.T,
	storage *sharedcache.Storage,
	status sharedcache.AttemptStatus,
	objects []sharedcache.CommitObject,
	now time.Time,
) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	decision := sharedcache.CommitDecision{
		SchemaVersion:             "1.0",
		RecordType:                "COMMIT_DECISION",
		ContractVersion:           "buildopt-cache-commit/v1",
		DecisionID:                "central-gradle-cache-commit",
		AttemptID:                 status.AttemptID,
		Repository:                status.Repository,
		SourceRevision:            status.SourceRevision,
		SourceStateDigest:         status.SourceStateDigest,
		Objects:                   objects,
		PolicyDigest:              status.PolicyDigest,
		ConfigurationPolicyDigest: status.ConfigurationPolicyDigest,
		CacheContractDigest:       status.CacheContractDigest,
		TestOptimizationGrant: sharedcache.TestOptimizationGrant{
			State: "NOT_REQUIRED", Reason: "NO_TEST_OUTPUTS",
		},
		RevocationEpoch: 7,
		Validation: sharedcache.CommitValidation{
			Status: "NOT_REQUIRED", Reason: "ALLOWLISTED_DIRECT_ACTION",
		},
		IssuedAt:  now.Format(time.RFC3339Nano),
		ExpiresAt: now.Add(30 * time.Minute).Format(time.RFC3339Nano),
		Authentication: sharedcache.CommitAuthentication{
			Algorithm: "Ed25519", KeyID: centralGradleDecisionKeyID,
		},
	}
	provisional := canonicalCentralGradleValue(t, decision)
	digest := centralGradleDecisionDigest(t, provisional)
	decision.DecisionDigest = digest
	decision.Authentication.Signature = base64.RawURLEncoding.EncodeToString(
		ed25519.Sign(
			privateKey,
			[]byte("buildopt-cache-commit/v1\x00"+centralGradleDecisionKeyID+"\x00"+digest),
		),
	)
	canonical := canonicalCentralGradleValue(t, decision)
	verified, err := sharedcache.VerifyCommitDecision(
		context.Background(),
		canonical,
		map[string]ed25519.PublicKey{centralGradleDecisionKeyID: publicKey},
		7,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := storage.CommitAttempt(
		context.Background(),
		status.StateVersion,
		7,
		verified,
	); err != nil {
		t.Fatal(err)
	}
}

func canonicalCentralGradleValue(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := contractcrypto.CanonicalizeJCS(encoded)
	if err != nil {
		t.Fatal(err)
	}
	return canonical
}

func centralGradleDecisionDigest(t *testing.T, canonical []byte) string {
	t.Helper()
	var document map[string]any
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.UseNumber()
	if err := decoder.Decode(&document); err != nil {
		t.Fatal(err)
	}
	delete(document, "decisionDigest")
	authentication, ok := document["authentication"].(map[string]any)
	if !ok {
		t.Fatal("central Gradle decision authentication is absent")
	}
	delete(authentication, "signature")
	payload := canonicalCentralGradleValue(t, document)
	digest := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(digest[:])
}
