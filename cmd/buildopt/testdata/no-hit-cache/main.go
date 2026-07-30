package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/tonyredondo/buildopt/internal/localauthority"
)

const usage = `usage:
  no-hit-cache prepare --root PATH --attempt-id UUID
  no-hit-cache serve --authority PATH --trust-root PATH --credential PATH --ready-file PATH --log PATH
`

var cachePathPattern = regexp.MustCompile(`^/cache/[A-Za-z0-9._-]{1,256}$`)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		_, _ = fmt.Fprint(os.Stderr, usage)
		return 64
	}
	switch args[0] {
	case "prepare":
		return runPrepare(args[1:])
	case "serve":
		return runServe(args[1:])
	default:
		_, _ = fmt.Fprint(os.Stderr, usage)
		return 64
	}
}

func runPrepare(args []string) int {
	flags := flag.NewFlagSet("prepare", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	root := flags.String("root", "", "private fixture root")
	attemptID := flags.String("attempt-id", "", "unique authority attempt UUID")
	if err := flags.Parse(args); err != nil ||
		flags.NArg() != 0 ||
		*root == "" ||
		*attemptID == "" {
		return 64
	}
	if err := prepareAuthority(*root, *attemptID, time.Now().UTC()); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "no-hit-cache: %v\n", err)
		return 1
	}
	return 0
}

func prepareAuthority(root string, attemptID string, now time.Time) error {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return errors.New("authority root must be an absolute clean path")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return fmt.Errorf("create authority root: %w", err)
	}
	if err := os.Chmod(root, 0o700); err != nil {
		return fmt.Errorf("set authority root mode: %w", err)
	}

	credential := bytes.Repeat([]byte{0x5a}, localauthority.CredentialBytes)
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x11}, 32))
	publicKey := privateKey.Public().(ed25519.PublicKey)
	credentialHash := sha256.Sum256(credential)
	document := localauthority.Document{
		Repository: localauthority.RepositoryIdentity{
			Tenant:      "tenant-internal",
			Repository:  "tonyredondo/buildopt",
			TrustDomain: "a0-no-hit",
		},
		SourceRevision:      strings.Repeat("a", 40),
		SourceStateDigest:   "hmac-sha256:" + strings.Repeat("1", 64),
		CacheContractDigest: "sha256:" + strings.Repeat("2", 64),
		Attempt: localauthority.AuthorityAttempt{
			AttemptID:        attemptID,
			OwnerID:          "a0-no-hit-runner",
			LeaseID:          "lease-" + attemptID,
			LeaseExpiresAt:   now.Add(45 * time.Minute).Format(time.RFC3339Nano),
			AllowRead:        true,
			AllowWrite:       false,
			CredentialDigest: "sha256:" + hex.EncodeToString(credentialHash[:]),
		},
		Policy: localauthority.OptimizationPolicy{
			SchemaVersion:               "1.0",
			RecordType:                  "OPTIMIZATION_POLICY",
			PolicyID:                    "a0-no-hit-policy",
			PolicyVersion:               1,
			ConfigurationPolicyDigest:   "sha256:" + strings.Repeat("3", 64),
			RevocationEpoch:             1,
			L1SecurityGeneration:        1,
			GatewayConnectionGeneration: 1,
			IssuedAt:                    now.Add(-time.Minute).Format(time.RFC3339Nano),
			LauncherVersionRange:        ">=0.1.0 <0.2.0",
			PluginVersionRange:          ">=0.1.0 <0.2.0",
			Mode:                        "VERIFIED",
			AllowedActions:              []string{"REMOTE_CACHE_ALLOWLISTED"},
			RemoteCache: localauthority.RemoteCachePolicy{
				Read:                true,
				Write:               "DISABLED",
				Namespace:           "a0-no-hit",
				NamespaceGeneration: 1,
			},
			ConfigurationCache: localauthority.ConfigurationCachePolicy{
				Enabled:         true,
				ContractVersion: "configuration-cache-v1",
			},
			ResourceProfile: localauthority.ResourceProfileReference{
				ProfileID:      "W4_H6G",
				ProfileDigest:  "sha256:" + strings.Repeat("4", 64),
				CatalogVersion: "resource-catalog-v1",
			},
			Budgets: localauthority.PolicyBudgets{
				MaxSynchronousOverheadMs:    500,
				MaxSynchronousOverheadRatio: 0.02,
				MaxValidationRunnerMsPerDay: 900000,
			},
			ExportProfile: "SUMMARY",
			QualifiedTasks: []localauthority.QualifiedTask{{
				ImplementationHash:  "sha256:" + strings.Repeat("5", 64),
				QualificationSource: "OFFICIAL",
				ContractRef:         "java-compile-v1",
				CacheContractDigest: "sha256:" + strings.Repeat("2", 64),
				QualificationState:  "CONTRACT_QUALIFIED",
				RepeatabilityGate:   "PASSED",
				RelocatabilityGate:  "PASSED",
			}},
			AffectedBuild: localauthority.AffectedBuild{
				EnabledInCI: true,
			},
			ExpiresAt: now.Add(time.Hour).Format(time.RFC3339Nano),
		},
		Revocation: localauthority.RevocationState{
			ContractVersion:      "buildopt-cache-control/v1",
			RequestID:            "a0-no-hit-revocation-1",
			TrustDomain:          "a0-no-hit",
			RevocationEpoch:      1,
			L1SecurityGeneration: 1,
			ValidUntil:           now.Add(2 * time.Hour).Format(time.RFC3339Nano),
		},
	}
	authority, err := localauthority.Sign(
		document,
		"a0-no-hit-key",
		privateKey,
	)
	if err != nil {
		return fmt.Errorf("sign authority: %w", err)
	}
	trustRoot, err := localauthority.EncodeTrustRoot(
		localauthority.TrustRoot{
			Keys: []localauthority.PublicKey{{
				KeyID: "a0-no-hit-key",
				PublicKey: base64.RawURLEncoding.EncodeToString(
					publicKey,
				),
			}},
		},
	)
	if err != nil {
		return fmt.Errorf("encode trust root: %w", err)
	}
	for path, content := range map[string][]byte{
		filepath.Join(root, "authority.json"):  authority,
		filepath.Join(root, "trust-root.json"): trustRoot,
		filepath.Join(root, "credential"): []byte(
			base64.RawURLEncoding.EncodeToString(credential),
		),
	} {
		if err := writePrivate(path, content); err != nil {
			return err
		}
	}
	return nil
}

func runServe(args []string) int {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	authority := flags.String("authority", "", "current authority file")
	trustRoot := flags.String("trust-root", "", "pinned trust root file")
	credential := flags.String("credential", "", "cache credential file")
	readyFile := flags.String("ready-file", "", "private endpoint output")
	logPath := flags.String("log", "", "private request JSONL")
	if err := flags.Parse(args); err != nil ||
		flags.NArg() != 0 ||
		*authority == "" ||
		*trustRoot == "" ||
		*credential == "" ||
		*readyFile == "" ||
		*logPath == "" {
		return 64
	}
	if err := serve(
		*authority,
		*trustRoot,
		*credential,
		*readyFile,
		*logPath,
	); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "no-hit-cache: %v\n", err)
		return 1
	}
	return 0
}

type missHandler struct {
	authorityPath  string
	trustRootPath  string
	credentialPath string
	bearer         string
	log            *os.File
	mutex          sync.Mutex
}

func serve(
	authorityPath string,
	trustRootPath string,
	credentialPath string,
	readyPath string,
	logPath string,
) error {
	_, _, credential, err := localauthority.LoadFiles(
		context.Background(),
		authorityPath,
		trustRootPath,
		credentialPath,
		time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("load initial authority: %w", err)
	}
	defer clear(credential)
	logFile, err := os.OpenFile(
		logPath,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0o600,
	)
	if err != nil {
		return fmt.Errorf("open request log: %w", err)
	}
	defer logFile.Close()
	if err := logFile.Chmod(0o600); err != nil {
		return fmt.Errorf("set request log mode: %w", err)
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer listener.Close()

	handler := &missHandler{
		authorityPath:  authorityPath,
		trustRootPath:  trustRootPath,
		credentialPath: credentialPath,
		bearer: base64.RawURLEncoding.EncodeToString(
			credential,
		),
		log: logFile,
	}
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 2 * time.Second,
	}
	serveDone := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serveDone <- err
	}()
	endpoint := "http://" + listener.Addr().String()
	if err := writePrivate(readyPath, []byte(endpoint+"\n")); err != nil {
		_ = server.Close()
		<-serveDone
		return fmt.Errorf("write ready file: %w", err)
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()
	select {
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			_ = server.Close()
			return err
		}
		return <-serveDone
	case err := <-serveDone:
		return err
	}
}

func (handler *missHandler) ServeHTTP(
	writer http.ResponseWriter,
	request *http.Request,
) {
	writer.Header().Set("Cache-Control", "no-store")
	if request.Method != http.MethodGet ||
		request.URL.RawQuery != "" ||
		!cachePathPattern.MatchString(request.URL.Path) {
		writer.Header().Set("Allow", http.MethodGet)
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	verified, _, currentCredential, err := localauthority.LoadFiles(
		request.Context(),
		handler.authorityPath,
		handler.trustRootPath,
		handler.credentialPath,
		time.Now().UTC(),
	)
	clear(currentCredential)
	if err != nil ||
		request.Header.Get("Authorization") != "Bearer "+handler.bearer ||
		request.Header.Get("X-BuildOpt-Authority-Digest") !=
			verified.Document().AuthorityDigest {
		writer.WriteHeader(http.StatusUnauthorized)
		return
	}
	record, err := json.Marshal(map[string]string{
		"method":  request.Method,
		"path":    request.URL.Path,
		"outcome": "MISS",
	})
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	handler.mutex.Lock()
	_, writeErr := handler.log.Write(append(record, '\n'))
	if writeErr == nil {
		writeErr = handler.log.Sync()
	}
	handler.mutex.Unlock()
	if writeErr != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writer.WriteHeader(http.StatusNotFound)
}

func writePrivate(path string, content []byte) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return errors.New("private output path must be absolute and clean")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create private output directory: %w", err)
	}
	temporary, err := os.CreateTemp(
		filepath.Dir(path),
		".no-hit-cache-*.tmp",
	)
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(content); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	return nil
}
