package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

func TestCentralTokenCLIEmitsSecretOnceAndRevokesByID(t *testing.T) {
	stateRoot := filepath.Join(t.TempDir(), "shared")
	expiresAt := time.Now().UTC().Add(time.Hour).Format(time.RFC3339Nano)
	var issueOutput bytes.Buffer
	var issueError bytes.Buffer
	exitCode := run(
		context.Background(),
		[]string{
			"central-token", "issue", "--state-dir", stateRoot,
			"--repository-scope-sha256", strings.Repeat("1", 64),
			"--tenant", "tenant-test", "--repository", "repository-test",
			"--trust-domain", "trust-test", "--namespace", "gradle/project",
			"--namespace-generation", "1",
			"--capabilities", "state-write,cache-read", "--expires-at", expiresAt,
		},
		func(string) string { return "" }, &issueOutput, &issueError,
	)
	if exitCode != 0 || issueError.Len() != 0 {
		t.Fatalf("issue exit=%d stdout=%q stderr=%q", exitCode, issueOutput.String(), issueError.String())
	}
	var issued struct {
		SchemaVersion string   `json:"schemaVersion"`
		TokenID       string   `json:"tokenId"`
		Token         string   `json:"token"`
		Capabilities  []string `json:"capabilities"`
	}
	if err := json.Unmarshal(issueOutput.Bytes(), &issued); err != nil {
		t.Fatal(err)
	}
	if issued.SchemaVersion != "buildopt.central/access-token/v1" ||
		len(issued.TokenID) != 32 || len(issued.Token) != 43 ||
		len(issued.Capabilities) != 2 || issued.Capabilities[0] != "CACHE_READ" ||
		issued.Capabilities[1] != "STATE_WRITE" {
		t.Fatalf("issue document = %+v", issued)
	}
	controlBytes, err := os.ReadFile(filepath.Join(stateRoot, "control.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(controlBytes, []byte(issued.Token)) {
		t.Fatal("raw token persisted in control database")
	}
	var revokeOutput bytes.Buffer
	var revokeError bytes.Buffer
	exitCode = run(
		context.Background(),
		[]string{"central-token", "revoke", "--state-dir", stateRoot, "--token-id", issued.TokenID},
		func(string) string { return "" }, &revokeOutput, &revokeError,
	)
	if exitCode != 0 || revokeError.Len() != 0 ||
		!strings.Contains(revokeOutput.String(), `"revoked":true`) ||
		strings.Contains(revokeOutput.String(), issued.Token) {
		t.Fatalf("revoke exit=%d stdout=%q stderr=%q", exitCode, revokeOutput.String(), revokeError.String())
	}
}

func TestCentralHTTPSServerUsesTrustedTLSAndLiveRevocation(t *testing.T) {
	root := t.TempDir()
	stateRoot := filepath.Join(root, "shared")
	certificatePath, keyPath, roots := writeCentralTLSTestIdentity(t, root)
	now := time.Now().UTC()
	registry, err := sharedcache.OpenCentralTokenRegistry(context.Background(), stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	issued, err := registry.Issue(context.Background(), sharedcache.CentralTokenIssueRequest{
		Scope: sharedcache.CentralTokenScope{
			RepositoryScopeSHA256: strings.Repeat("1", 64),
			Tenant:                "tenant-test", Repository: "repository-test", TrustDomain: "trust-test",
			Namespace: "gradle/project", NamespaceGeneration: 1,
		},
		Capabilities: []sharedcache.CentralCapability{sharedcache.CentralCacheRead},
		ExpiresAt:    now.Add(time.Hour),
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Close(); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	output := newNotifyingWriter()
	var stderr bytes.Buffer
	exited := make(chan int, 1)
	go func() {
		exited <- run(ctx, []string{
			"serve", "--listen", "0.0.0.0:0", "--state-dir", stateRoot,
			"--tls-cert", certificatePath, "--tls-key", keyPath, "--central-auth",
		}, func(string) string { return "" }, output, &stderr)
	}()
	waitForServerOutput(t, output, "central HTTPS cache/state routing enabled")
	endpoint := serverEndpointFromOutput(t, output.String())
	if !strings.HasPrefix(endpoint, "https://") {
		t.Fatalf("endpoint = %q", endpoint)
	}
	endpoint = strings.Replace(endpoint, "0.0.0.0", "127.0.0.1", 1)
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: centralTestTLSClientConfig(roots)}}
	response, err := client.Get(endpoint + readinessPath)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("trusted readiness = %d", response.StatusCode)
	}
	if _, err := http.Get(endpoint + readinessPath); err == nil {
		t.Fatal("untrusted client accepted the private test certificate")
	}
	request, err := http.NewRequest(http.MethodGet, endpoint+"/cache/absent", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+issued.Token)
	request.Header.Set(sharedcache.CentralNamespaceHeader, "gradle/project")
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("authorized cache miss = %d", response.StatusCode)
	}

	registry, err = sharedcache.OpenCentralTokenRegistry(context.Background(), stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	revoked, err := registry.Revoke(context.Background(), issued.TokenID, time.Now().UTC())
	closeErr := registry.Close()
	if err != nil || closeErr != nil || !revoked {
		t.Fatalf("live revoke = %t/%v/%v", revoked, err, closeErr)
	}
	request, _ = http.NewRequest(http.MethodGet, endpoint+"/cache/absent", nil)
	request.Header.Set("Authorization", "Bearer "+issued.Token)
	request.Header.Set(sharedcache.CentralNamespaceHeader, "gradle/project")
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("revoked cache request = %d", response.StatusCode)
	}
	if strings.Contains(output.String(), issued.Token) || strings.Contains(stderr.String(), issued.Token) {
		t.Fatal("central token leaked to server output")
	}

	cancel()
	select {
	case exitCode := <-exited:
		if exitCode != 0 {
			t.Fatalf("server exit=%d stderr=%q", exitCode, stderr.String())
		}
	case <-time.After(nativeTestTimeout(3 * time.Second)):
		t.Fatal("central HTTPS server did not stop")
	}
}

func TestCentralHTTPSConfigurationFailsClosed(t *testing.T) {
	root := t.TempDir()
	certificatePath, keyPath, _ := writeCentralTLSTestIdentity(t, root)
	for _, testCase := range []struct {
		name string
		args []string
		want string
	}{
		{name: "central without TLS", args: []string{"serve", "--central-auth", "--state-dir", filepath.Join(root, "one")}, want: "central authentication requires TLS"},
		{name: "certificate without key", args: []string{"serve", "--tls-cert", certificatePath}, want: "TLS requires both certificate and private key"},
		{name: "invalid private key", args: []string{"serve", "--tls-cert", certificatePath, "--tls-key", certificatePath}, want: "invalid TLS configuration"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exitCode := run(context.Background(), testCase.args, func(string) string { return "" }, &stdout, &stderr)
			if exitCode != exitConfiguration || !strings.Contains(stderr.String(), testCase.want) {
				t.Fatalf("exit=%d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
			}
		})
	}
	if err := validateListenAddress("0.0.0.0:8042", true); err != nil {
		t.Fatalf("external TLS address rejected: %v", err)
	}
	if err := validateListenAddress("0.0.0.0:8042", false); err == nil {
		t.Fatal("external plaintext address accepted")
	}
	if _, err := loadServerTLS(certificatePath, keyPath); err != nil {
		t.Fatalf("valid TLS identity rejected: %v", err)
	}
}

func writeCentralTLSTestIdentity(t *testing.T, root string) (string, string, *x509.CertPool) {
	t.Helper()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "buildopt-central-test"},
		NotBefore: now.Add(-time.Minute), NotAfter: now.Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		BasicConstraintsValid: true, IsCA: true,
	}
	certificateDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	privateDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	certificatePEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificateDER})
	privatePEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateDER})
	certificatePath := filepath.Join(root, "central-cert.pem")
	keyPath := filepath.Join(root, "central-key.pem")
	if err := os.WriteFile(certificatePath, certificatePEM, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, privatePEM, 0o600); err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(certificatePEM) {
		t.Fatal("append test certificate")
	}
	return certificatePath, keyPath, roots
}

func centralTestTLSClientConfig(roots *x509.CertPool) *tls.Config {
	return &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS13}
}
