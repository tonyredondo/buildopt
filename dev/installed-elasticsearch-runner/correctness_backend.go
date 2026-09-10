package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/tonyredondo/buildopt/internal/sharedcache"
	"github.com/tonyredondo/buildopt/internal/stickywrapper"
	"github.com/tonyredondo/buildopt/internal/wcncpobserve"
)

const correctnessScope = "elastic/elasticsearch"

type correctnessPost struct {
	AtNS   int64                           `json:"receivedBootNanoseconds"`
	Status int                             `json:"status"`
	Facts  []wcncpobserve.ObservationFacts `json:"facts"`
}

type correctnessBackend struct {
	URL        string
	CA         string
	Credential fileBinding
	client     *http.Client
	root       string
	pkg        fileBinding
	storage    *sharedcache.Storage
	server     *http.Server
	listener   net.Listener
	done       chan error
	mutex      sync.Mutex
	posts      []correctnessPost
}

func correctnessScopeDigest() string {
	h := sha256.New()
	for _, s := range []string{"buildopt-optimize-portfolio-repository-v1", correctnessScope} {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(s)))
		_, _ = h.Write(size[:])
		_, _ = io.WriteString(h, s)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func startCorrectnessBackend(ctx context.Context, root string, pkg fileBinding) (_ *correctnessBackend, retErr error) {
	return startCorrectnessBackendAt(ctx, root, pkg, "127.0.0.1:0")
}

// A delivery probe can reuse an installed wrapper's frozen loopback endpoint
// without rewriting its committed configuration. Binding an occupied port fails.
func startCorrectnessBackendAt(ctx context.Context, root string, pkg fileBinding, address string) (_ *correctnessBackend, retErr error) {
	host, _, err := net.SplitHostPort(address)
	if err != nil || host != "127.0.0.1" {
		return nil, errors.New("owned IPv4 loopback endpoint required")
	}
	if err := checkBinding(pkg); err != nil {
		return nil, err
	}
	if err := canonicalExisting(filepath.Dir(root)); err != nil {
		return nil, err
	}
	if err := os.Mkdir(root, 0700); err != nil {
		return nil, err
	}
	b := &correctnessBackend{root: root, pkg: pkg, done: make(chan error, 1)}
	defer func() {
		if retErr != nil {
			_ = b.close()
		}
	}()
	b.storage, err = sharedcache.Open(ctx, filepath.Join(root, "state"))
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Truncate(time.Second)
	issued, err := b.storage.IssueCentralToken(ctx, sharedcache.CentralTokenIssueRequest{Scope: sharedcache.CentralTokenScope{RepositoryScopeSHA256: correctnessScopeDigest(), Tenant: "eic-local", Repository: correctnessScope, TrustDomain: "eic-local", Namespace: "gradle/elasticsearch", NamespaceGeneration: 1}, Capabilities: []sharedcache.CentralCapability{sharedcache.CentralStateRead, sharedcache.CentralStateWrite}, ExpiresAt: now.Add(4 * time.Hour)}, now)
	if err != nil {
		return nil, err
	}
	if _, err = b.storage.GrantWCNCPActor(ctx, issued.TokenID, sharedcache.WCNCPActorTrustedObserver, now); err != nil {
		return nil, err
	}
	document := map[string]any{"schemaVersion": "buildopt.central/access-token/v1", "tokenId": issued.TokenID, "token": issued.Token, "repositoryScopeSha256": issued.Scope.RepositoryScopeSHA256, "tenant": issued.Scope.Tenant, "repository": issued.Scope.Repository, "trustDomain": issued.Scope.TrustDomain, "namespace": issued.Scope.Namespace, "namespaceGeneration": issued.Scope.NamespaceGeneration, "capabilities": issued.Capabilities, "issuedAt": issued.IssuedAt.Format(time.RFC3339Nano), "expiresAt": issued.ExpiresAt.Format(time.RFC3339Nano)}
	b.Credential.Path = filepath.Join(root, "credential.json")
	if err = writeJSON(b.Credential.Path, document); err != nil {
		return nil, err
	}
	b.Credential.SHA256, err = hashFile(b.Credential.Path)
	if err != nil {
		return nil, err
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}
	certificate := &x509.Certificate{SerialNumber: serial, Subject: pkix.Name{CommonName: "EIC owned loopback backend"}, NotBefore: now.Add(-time.Minute), NotAfter: now.Add(4 * time.Hour), IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, BasicConstraintsValid: true, IsCA: true}
	der, err := x509.CreateCertificate(rand.Reader, certificate, certificate, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	b.CA = filepath.Join(root, "ca.pem")
	if err = writeNew(b.CA, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})); err != nil {
		return nil, err
	}
	handler, err := sharedcache.NewCentralHTTPSHandler(b.storage)
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	b.listener = tls.NewListener(listener, &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key, Leaf: cert}}})
	b.URL = "https://" + listener.Addr().String()
	b.server = &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/0.0.1/linux-amd64" {
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				http.Error(w, "method", 405)
				return
			}
			if err := checkBinding(b.pkg); err != nil {
				http.Error(w, "package drift", 409)
				return
			}
			http.ServeFile(w, r, b.pkg.Path)
			return
		}
		post := correctnessPost{}
		if r.Method == http.MethodPost {
			post.AtNS, _ = bootNow()
			raw, e := io.ReadAll(io.LimitReader(r.Body, (1<<20)+1))
			if e != nil || len(raw) > 1<<20 {
				http.Error(w, "bounded observations required", 400)
				return
			}
			if e = json.Unmarshal(raw, &post.Facts); e != nil {
				http.Error(w, "invalid observation batch", 400)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(raw))
		}
		response := &correctnessHTTPResponse{ResponseWriter: w}
		handler.ServeHTTP(response, r)
		if r.Method == http.MethodPost {
			post.Status = response.status
			if post.Status == 0 {
				post.Status = 200
			}
			b.mutex.Lock()
			b.posts = append(b.posts, post)
			b.mutex.Unlock()
		}
	}), ReadHeaderTimeout: 2 * time.Second, IdleTimeout: 5 * time.Second}
	roots := x509.NewCertPool()
	roots.AddCert(cert)
	b.client = &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: roots}}, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirect refused") }}
	go func() { b.done <- b.server.Serve(b.listener) }()
	return b, nil
}

type correctnessHTTPResponse struct {
	http.ResponseWriter
	status int
}

func (w *correctnessHTTPResponse) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
func (w *correctnessHTTPResponse) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(200)
	}
	return w.ResponseWriter.Write(p)
}

func (b *correctnessBackend) close() error {
	if b == nil {
		return nil
	}
	var result error
	if b.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		result = b.server.Shutdown(ctx)
		cancel()
		if result != nil {
			result = errors.Join(result, b.server.Close())
		}
		if err := <-b.done; err != nil && !errors.Is(err, http.ErrServerClosed) {
			result = errors.Join(result, err)
		}
		b.server = nil
		b.listener = nil
	} else if b.listener != nil {
		result = b.listener.Close()
		b.listener = nil
	}
	if b.client != nil {
		b.client.CloseIdleConnections()
	}
	if b.storage != nil {
		result = errors.Join(result, b.storage.Close())
		b.storage = nil
	}
	return result
}

func (b *correctnessBackend) observations() []correctnessPost {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	result := make([]correctnessPost, len(b.posts))
	copy(result, b.posts)
	return result
}

type correctnessRelease struct{ url, digest string }

func (r correctnessRelease) Latest(context.Context) (stickywrapper.Release, error) {
	release := stickywrapper.Release{Version: "0.0.1", Distributions: map[string]stickywrapper.Distribution{}}
	for _, platform := range []string{"linux-amd64", "macos-amd64", "macos-arm64", "windows-amd64"} {
		release.Distributions[platform] = stickywrapper.Distribution{URL: r.url + "/0.0.1/" + platform, SHA256: r.digest}
	}
	return release, nil
}
func (r correctnessRelease) Version(ctx context.Context, _ string) (stickywrapper.Release, error) {
	return r.Latest(ctx)
}

func (b *correctnessBackend) install(ctx context.Context, work string) ([]entry, error) {
	config := stickywrapper.Config{Mode: "observe", ServerURL: b.URL, ProjectScope: correctnessScope, CredentialEnv: "BUILDOPT_TEAM_TOKEN", TrialBudgetPercent: 5}
	snapshot, err := (stickywrapper.Generator{Root: work, Resolver: correctnessRelease{b.URL, b.pkg.SHA256}}).Init(ctx, config)
	if err != nil {
		return nil, err
	}
	selectors := []string{}
	for name := range snapshot.Files {
		selectors = append(selectors, name)
	}
	return inventory(work, selectors)
}
