// Command central-two-machine-control prepares and commits the owner-side
// authority used only by the isolated central-cache POC harness.
package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
	"github.com/tonyredondo/buildopt/internal/localauthority"
	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

const (
	decisionKeyID = "poc-two-machine-owner"
	metadataFile  = "metadata.json"
	decisionKey   = "decision-key.json"
)

type metadata struct {
	SchemaVersion         string `json:"schemaVersion"`
	RepositoryID          string `json:"repositoryId"`
	RepositoryScopeSHA256 string `json:"repositoryScopeSha256"`
	Tenant                string `json:"tenant"`
	TrustDomain           string `json:"trustDomain"`
	Namespace             string `json:"namespace"`
	NamespaceGeneration   int64  `json:"namespaceGeneration"`
	AttemptID             string `json:"attemptId"`
	AuthorityDigest       string `json:"authorityDigest"`
	LeaseExpiresAt        string `json:"leaseExpiresAt"`
	RevocationEpoch       int64  `json:"revocationEpoch"`
	ProductionAuthorized  bool   `json:"productionAuthorized"`
	TestOptimization      string `json:"testOptimization"`
}

type tokenDocument struct {
	SchemaVersion         string                          `json:"schemaVersion"`
	TokenID               string                          `json:"tokenId"`
	Token                 string                          `json:"token"`
	RepositoryScopeSHA256 string                          `json:"repositoryScopeSha256"`
	Tenant                string                          `json:"tenant"`
	Repository            string                          `json:"repository"`
	TrustDomain           string                          `json:"trustDomain"`
	Namespace             string                          `json:"namespace"`
	NamespaceGeneration   int64                           `json:"namespaceGeneration"`
	Capabilities          []sharedcache.CentralCapability `json:"capabilities"`
	IssuedAt              string                          `json:"issuedAt"`
	ExpiresAt             string                          `json:"expiresAt"`
}

type commitResult struct {
	sharedcache.CommitResult
	ObjectBytes int64
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	switch os.Args[1] {
	case "prepare":
		err = prepare(os.Args[2:])
	case "commit":
		err = commit(os.Args[2:])
	default:
		usage()
	}
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "central-two-machine-control: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	_, _ = fmt.Fprintln(os.Stderr, "usage: central-two-machine-control prepare --state-dir PATH --output-dir PATH --repository-id OWNER/REPO --source-revision SHA --namespace ID [--lifetime DURATION]\n       central-two-machine-control commit --state-dir PATH --control-dir PATH [--abort-empty]")
	os.Exit(64)
}

func prepare(args []string) error {
	flags := flag.NewFlagSet("prepare", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	stateDir := flags.String("state-dir", "", "central storage root")
	outputDir := flags.String("output-dir", "", "private fixture output")
	repositoryID := flags.String("repository-id", "", "portable repository identity")
	sourceRevision := flags.String("source-revision", "", "producer source revision")
	namespace := flags.String("namespace", "", "Gradle cache namespace")
	lifetime := flags.Duration("lifetime", 3*time.Hour, "POC authority lifetime")
	if flags.Parse(args) != nil || flags.NArg() != 0 ||
		!absoluteClean(*stateDir) || !absoluteClean(*outputDir) ||
		*repositoryID == "" || *sourceRevision == "" || *namespace == "" ||
		*lifetime <= 0 || *lifetime > 24*time.Hour {
		return errors.New("invalid prepare arguments")
	}
	if err := os.MkdirAll(*outputDir, 0o700); err != nil {
		return err
	}
	now := time.Now().UTC().Truncate(time.Millisecond)
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	credential := make([]byte, localauthority.CredentialBytes)
	if _, err := io.ReadFull(rand.Reader, credential); err != nil {
		return err
	}
	defer clear(credential)
	credentialDigest := sha256.Sum256(credential)
	sourceDigest := sha256.Sum256([]byte(*sourceRevision))
	leaseExpiresAt := now.Add(*lifetime)
	document := localauthority.Document{
		Repository: localauthority.RepositoryIdentity{
			Tenant: "poc-owner", Repository: *repositoryID, TrustDomain: "poc-two-machine",
		},
		SourceRevision:      strings.ToLower(*sourceRevision),
		SourceStateDigest:   "sha256:" + hex.EncodeToString(sourceDigest[:]),
		CacheContractDigest: "sha256:" + strings.Repeat("2", 64),
		Attempt: localauthority.AuthorityAttempt{
			AttemptID: "22222222-2222-4222-8222-222222222222",
			OwnerID:   "trusted-poc-producer", LeaseID: "two-machine-producer-lease",
			LeaseExpiresAt: leaseExpiresAt.Format(time.RFC3339Nano),
			AllowRead:      true, AllowWrite: true,
			CredentialDigest: "sha256:" + hex.EncodeToString(credentialDigest[:]),
		},
		Policy: localauthority.OptimizationPolicy{
			SchemaVersion: "1.0", RecordType: "OPTIMIZATION_POLICY",
			PolicyID: "two-machine-poc", PolicyVersion: 1,
			ConfigurationPolicyDigest: "sha256:" + strings.Repeat("3", 64),
			RevocationEpoch:           1, L1SecurityGeneration: 1, GatewayConnectionGeneration: 1,
			IssuedAt:             now.Add(-time.Minute).Format(time.RFC3339Nano),
			LauncherVersionRange: ">=0.1.0 <0.2.0", PluginVersionRange: ">=0.1.0 <0.2.0",
			Mode: "VERIFIED", AllowedActions: []string{"REMOTE_CACHE_ALLOWLISTED"},
			RemoteCache: localauthority.RemoteCachePolicy{
				Read: true, Write: "TRUSTED_CI_ONLY", Namespace: *namespace, NamespaceGeneration: 1,
			},
			ConfigurationCache: localauthority.ConfigurationCachePolicy{
				Enabled: false, ContractVersion: "configuration-cache-v1",
			},
			Budgets: localauthority.PolicyBudgets{
				MaxSynchronousOverheadMs: 500, MaxSynchronousOverheadRatio: 0.02,
				MaxValidationRunnerMsPerDay: 60000,
			},
			ExportProfile: "SUMMARY",
			QualifiedTasks: []localauthority.QualifiedTask{{
				ImplementationHash:  "sha256:" + strings.Repeat("5", 64),
				QualificationSource: "OFFICIAL", ContractRef: "java-compile-v1",
				CacheContractDigest: "sha256:" + strings.Repeat("2", 64),
				QualificationState:  "CONTRACT_QUALIFIED",
				RepeatabilityGate:   "PASSED", RelocatabilityGate: "PASSED",
			}},
			AffectedBuild: localauthority.AffectedBuild{EnabledInCI: true},
			ExpiresAt:     now.Add(*lifetime).Format(time.RFC3339Nano),
		},
		Revocation: localauthority.RevocationState{
			ContractVersion: "buildopt-cache-control/v1", RequestID: "two-machine-revocation-1",
			TrustDomain: "poc-two-machine", RevocationEpoch: 1, L1SecurityGeneration: 1,
			ValidUntil: now.Add(*lifetime).Format(time.RFC3339Nano),
		},
	}
	authority, err := localauthority.Sign(document, decisionKeyID, privateKey)
	if err != nil {
		return err
	}
	verified, err := localauthority.Verify(
		context.Background(), authority,
		map[string]ed25519.PublicKey{decisionKeyID: publicKey}, credential, now,
	)
	if err != nil {
		return err
	}
	trustRoot, err := localauthority.EncodeTrustRoot(localauthority.TrustRoot{
		Keys: []localauthority.PublicKey{{
			KeyID:     decisionKeyID,
			PublicKey: base64.RawURLEncoding.EncodeToString(publicKey),
		}},
	})
	if err != nil {
		return err
	}
	keyDocument, err := localauthority.EncodeSigningKey(decisionKeyID, privateKey)
	if err != nil {
		return err
	}
	storage, err := sharedcache.Open(context.Background(), *stateDir)
	if err != nil {
		return err
	}
	defer storage.Close()
	binding, _, err := storage.InstallLocalAuthority(context.Background(), verified, credential, now)
	if err != nil {
		return err
	}
	scope := sharedcache.CentralTokenScope{
		RepositoryScopeSHA256: repositoryScope(*repositoryID),
		Tenant:                "poc-owner", Repository: *repositoryID, TrustDomain: "poc-two-machine",
		Namespace: *namespace, NamespaceGeneration: 1,
	}
	producer, err := storage.IssueCentralToken(context.Background(), sharedcache.CentralTokenIssueRequest{
		Scope: scope,
		Capabilities: []sharedcache.CentralCapability{
			sharedcache.CentralCacheRead, sharedcache.CentralCacheWrite,
			sharedcache.CentralStateRead, sharedcache.CentralStateWrite,
		},
		ExpiresAt: now.Add(*lifetime),
	}, now)
	if err != nil {
		return err
	}
	consumer, err := storage.IssueCentralToken(context.Background(), sharedcache.CentralTokenIssueRequest{
		Scope: scope,
		Capabilities: []sharedcache.CentralCapability{
			sharedcache.CentralCacheRead, sharedcache.CentralStateRead,
		},
		ExpiresAt: now.Add(*lifetime),
	}, now)
	if err != nil {
		return err
	}
	meta := metadata{
		SchemaVersion: "buildopt.poc/central-two-machine-control/v1",
		RepositoryID:  *repositoryID, RepositoryScopeSHA256: scope.RepositoryScopeSHA256,
		Tenant: scope.Tenant, TrustDomain: scope.TrustDomain, Namespace: scope.Namespace,
		NamespaceGeneration: scope.NamespaceGeneration, AttemptID: binding.AttemptID,
		AuthorityDigest: binding.AuthorityDigest,
		LeaseExpiresAt:  leaseExpiresAt.Format(time.RFC3339Nano), RevocationEpoch: 1,
		ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
	}
	files := map[string][]byte{
		"authority.json":      authority,
		"trust-root.json":     trustRoot,
		"credential":          []byte(base64.RawURLEncoding.EncodeToString(credential)),
		"producer-token.json": encodeTokenDocument(producer),
		"producer-token.raw":  []byte(producer.Token),
		"consumer-token.json": encodeTokenDocument(consumer),
		decisionKey:           keyDocument,
		metadataFile:          mustJSON(meta),
	}
	for name, raw := range files {
		if err := writePrivate(filepath.Join(*outputDir, name), raw); err != nil {
			return err
		}
	}
	return json.NewEncoder(os.Stdout).Encode(meta)
}

func commit(args []string) error {
	flags := flag.NewFlagSet("commit", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	stateDir := flags.String("state-dir", "", "central storage root")
	controlDir := flags.String("control-dir", "", "private fixture control root")
	abortEmpty := flags.Bool(
		"abort-empty",
		false,
		"abort a cache attempt that produced no pending objects",
	)
	if flags.Parse(args) != nil || flags.NArg() != 0 ||
		!absoluteClean(*stateDir) || !absoluteClean(*controlDir) {
		return errors.New("invalid commit arguments")
	}
	var meta metadata
	if err := decodeFile(filepath.Join(*controlDir, metadataFile), &meta); err != nil {
		return err
	}
	keyRaw, err := os.ReadFile(filepath.Join(*controlDir, decisionKey))
	if err != nil {
		return err
	}
	keyID, privateKey, err := localauthority.ParseSigningKey(keyRaw)
	if err != nil || keyID != decisionKeyID {
		return errors.New("invalid owner decision key")
	}
	storage, err := sharedcache.Open(context.Background(), *stateDir)
	if err != nil {
		return err
	}
	defer storage.Close()
	status, err := storage.AttemptStatus(context.Background(), meta.AttemptID)
	if err != nil {
		return err
	}
	objects, err := storage.PendingAttemptObjects(context.Background(), meta.AttemptID)
	if err != nil {
		return fmt.Errorf("inspect pending producer objects: %w", err)
	}
	if len(objects) == 0 {
		if *abortEmpty {
			result, abortErr := storage.AbortAttempt(
				context.Background(),
				sharedcache.AbortAttemptRequest{
					RequestID:            "two-machine-empty-producer-abort",
					AttemptID:            status.AttemptID,
					ExpectedStateVersion: status.StateVersion,
					Reason:               "INCOMPLETE_COMMIT_DECISION",
				},
			)
			if abortErr != nil {
				return fmt.Errorf("abort empty producer attempt: %w", abortErr)
			}
			return json.NewEncoder(os.Stdout).Encode(commitResult{
				CommitResult: sharedcache.CommitResult{
					AttemptID:    result.Status.AttemptID,
					Outcome:      "EMPTY_ABORTED",
					ObjectCount:  0,
					StateVersion: result.Status.StateVersion,
				},
			})
		}
		return errors.New("pending producer objects are empty")
	}
	sort.Slice(objects, func(left, right int) bool {
		if objects[left].NamespaceGeneration != objects[right].NamespaceGeneration {
			return objects[left].NamespaceGeneration < objects[right].NamespaceGeneration
		}
		return objects[left].Key < objects[right].Key
	})
	now := time.Now().UTC().Truncate(time.Millisecond)
	leaseExpiresAt, err := time.Parse(time.RFC3339Nano, meta.LeaseExpiresAt)
	if err != nil {
		return err
	}
	expiresAt := now.Add(30 * time.Minute)
	if !expiresAt.Before(leaseExpiresAt) {
		expiresAt = leaseExpiresAt.Add(-time.Second)
	}
	decision := sharedcache.CommitDecision{
		SchemaVersion: "1.0", RecordType: "COMMIT_DECISION",
		ContractVersion: "buildopt-cache-commit/v1", DecisionID: "two-machine-cache-commit",
		AttemptID: status.AttemptID, Repository: status.Repository,
		SourceRevision: status.SourceRevision, SourceStateDigest: status.SourceStateDigest,
		Objects: objects, PolicyDigest: status.PolicyDigest,
		ConfigurationPolicyDigest: status.ConfigurationPolicyDigest,
		CacheContractDigest:       status.CacheContractDigest,
		TestOptimizationGrant: sharedcache.TestOptimizationGrant{
			State: "NOT_REQUIRED", Reason: "NO_TEST_OUTPUTS",
		},
		RevocationEpoch: meta.RevocationEpoch,
		Validation: sharedcache.CommitValidation{
			Status: "NOT_REQUIRED", Reason: "ALLOWLISTED_DIRECT_ACTION",
		},
		IssuedAt: now.Format(time.RFC3339Nano), ExpiresAt: expiresAt.Format(time.RFC3339Nano),
		Authentication: sharedcache.CommitAuthentication{Algorithm: "Ed25519", KeyID: keyID},
	}
	canonical, err := signDecision(decision, keyID, privateKey)
	if err != nil {
		return err
	}
	verified, err := sharedcache.VerifyCommitDecision(
		context.Background(), canonical,
		map[string]ed25519.PublicKey{keyID: privateKey.Public().(ed25519.PublicKey)},
		meta.RevocationEpoch, now,
	)
	if err != nil {
		return err
	}
	result, err := storage.CommitAttempt(context.Background(), status.StateVersion, meta.RevocationEpoch, verified)
	if err != nil {
		return err
	}
	var objectBytes int64
	for _, object := range objects {
		objectBytes += object.SizeBytes
	}
	return json.NewEncoder(os.Stdout).Encode(commitResult{
		CommitResult: result,
		ObjectBytes:  objectBytes,
	})
}

func signDecision(decision sharedcache.CommitDecision, keyID string, privateKey ed25519.PrivateKey) ([]byte, error) {
	provisional, err := json.Marshal(decision)
	if err != nil {
		return nil, err
	}
	var document map[string]any
	decoder := json.NewDecoder(bytes.NewReader(provisional))
	decoder.UseNumber()
	if err := decoder.Decode(&document); err != nil {
		return nil, err
	}
	delete(document, "decisionDigest")
	authentication, ok := document["authentication"].(map[string]any)
	if !ok {
		return nil, errors.New("missing decision authentication")
	}
	delete(authentication, "signature")
	payload, err := json.Marshal(document)
	if err != nil {
		return nil, err
	}
	payload, err = contractcrypto.CanonicalizeJCS(payload)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(payload)
	decision.DecisionDigest = "sha256:" + hex.EncodeToString(digest[:])
	decision.Authentication.Signature = base64.RawURLEncoding.EncodeToString(ed25519.Sign(
		privateKey,
		[]byte("buildopt-cache-commit/v1\x00"+keyID+"\x00"+decision.DecisionDigest),
	))
	encoded, err := json.Marshal(decision)
	if err != nil {
		return nil, err
	}
	return contractcrypto.CanonicalizeJCS(encoded)
}

func repositoryScope(repositoryID string) string {
	digest := sha256.New()
	writeDigestValue(digest, "buildopt-optimize-portfolio-repository-v1")
	writeDigestValue(digest, repositoryID)
	return hex.EncodeToString(digest.Sum(nil))
}

func writeDigestValue(writer io.Writer, value string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = writer.Write(size[:])
	_, _ = io.WriteString(writer, value)
}

func encodeTokenDocument(issued sharedcache.IssuedCentralToken) []byte {
	return mustJSON(tokenDocument{
		SchemaVersion: "buildopt.central/access-token/v1",
		TokenID:       issued.TokenID, Token: issued.Token,
		RepositoryScopeSHA256: issued.Scope.RepositoryScopeSHA256,
		Tenant:                issued.Scope.Tenant, Repository: issued.Scope.Repository,
		TrustDomain: issued.Scope.TrustDomain, Namespace: issued.Scope.Namespace,
		NamespaceGeneration: issued.Scope.NamespaceGeneration,
		Capabilities:        issued.Capabilities,
		IssuedAt:            issued.IssuedAt.Format(time.RFC3339Nano),
		ExpiresAt:           issued.ExpiresAt.Format(time.RFC3339Nano),
	})
}

func mustJSON(value any) []byte {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return append(raw, '\n')
}

func writePrivate(path string, raw []byte) error {
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func decodeFile(path string, destination any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	return nil
}

func absoluteClean(path string) bool {
	return path != "" && filepath.IsAbs(path) && filepath.Clean(path) == path
}

func clear(value []byte) {
	for index := range value {
		value[index] = 0
	}
}
