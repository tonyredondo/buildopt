package launcher

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

const (
	centralConnectUsage = "usage: buildopt connect https://HOST[:PORT] --token-file PATH [--ca-file PATH] [--state-dir .buildopt/optimize/v1] [--connection-dir .buildopt/central/v1]\n       buildopt sync [--connection-dir .buildopt/central/v1]\n"

	centralConnectionSchema = "buildopt.poc/central-connection/v1"
	centralBundleSchema     = "buildopt.poc/central-state-bundle/v1"
	centralSyncResultSchema = "buildopt.poc/central-state-sync-result/v1"
	centralConnectionDir    = ".buildopt/central/v1"
	centralConnectionFile   = "connection.json"
	centralTokenFile        = "token"
	centralCAFile           = "ca.pem"
	centralSnapshotDir      = "snapshots"
	centralMaximumConfig    = 64 << 10
	centralMaximumCA        = 1 << 20
	centralMaximumDocument  = 1 << 20
	centralMaximumArtifact  = 16 << 20
	centralRequestTimeout   = 10 * time.Second
)

var centralBundleSchemaPattern = regexp.MustCompile(
	`^buildopt\.[a-z0-9.-]+/[a-z0-9.-]+/v[1-9][0-9]*$`,
)

type centralConnection struct {
	SchemaVersion         string `json:"schemaVersion"`
	ServerURL             string `json:"serverUrl"`
	RepositoryID          string `json:"repositoryId"`
	RepositoryScopeSHA256 string `json:"repositoryScopeSha256"`
	StateDirectory        string `json:"stateDirectory"`
	TokenFile             string `json:"tokenFile"`
	CAFile                string `json:"caFile,omitempty"`
	ConnectedAt           string `json:"connectedAt"`
	ProductionAuthorized  bool   `json:"productionAuthorized"`
	TestOptimization      string `json:"testOptimization"`
}

type centralIssuedTokenDocument struct {
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

type centralStateBundle struct {
	SchemaVersion         string                   `json:"schemaVersion"`
	RecordType            string                   `json:"recordType"`
	Kind                  sharedcache.StateKind    `json:"kind"`
	RepositoryScopeSHA256 string                   `json:"repositoryScopeSha256"`
	CompatibilitySHA256   string                   `json:"compatibilitySha256"`
	BindingsSHA256        string                   `json:"bindingsSha256"`
	Files                 []centralStateBundleFile `json:"files"`
	CreatedAt             string                   `json:"createdAt"`
	ProductionAuthorized  bool                     `json:"productionAuthorized"`
	TestOptimization      string                   `json:"testOptimization"`
}

type centralStateBundleFile struct {
	Path                 string `json:"path"`
	SHA256               string `json:"sha256"`
	SizeBytes            int64  `json:"sizeBytes"`
	PayloadSchemaVersion string `json:"payloadSchemaVersion"`
	ContentBase64        string `json:"contentBase64"`
}

type centralLocalPublication struct {
	kind          sharedcache.StateKind
	bundle        centralStateBundle
	bundleRaw     []byte
	bundleSHA256  string
	compatibility string
	bindings      string
	origin        sharedcache.StateOrigin
	createdAt     time.Time
}

type centralRemoteSnapshot struct {
	head           sharedcache.StateHead
	headRaw        []byte
	headSHA256     string
	manifest       sharedcache.StateManifest
	manifestRaw    []byte
	manifestSHA256 string
	bundle         centralStateBundle
	bundleRaw      []byte
	bundleSHA256   string
}

type centralSyncKindResult struct {
	Kind              sharedcache.StateKind `json:"kind"`
	Status            string                `json:"status"`
	LocalPresent      bool                  `json:"localPresent"`
	RemoteGeneration  int64                 `json:"remoteGeneration"`
	SnapshotVerified  bool                  `json:"snapshotVerified"`
	Compatible        bool                  `json:"compatible"`
	ObjectsUploaded   int                   `json:"objectsUploaded"`
	ManifestUploaded  bool                  `json:"manifestUploaded"`
	HeadAdvanced      bool                  `json:"headAdvanced"`
	ProductionAllowed bool                  `json:"productionAuthorized"`
}

type centralSyncResult struct {
	SchemaVersion         string                  `json:"schemaVersion"`
	Operation             string                  `json:"operation"`
	ServerURL             string                  `json:"serverUrl"`
	RepositoryID          string                  `json:"repositoryId"`
	RepositoryScopeSHA256 string                  `json:"repositoryScopeSha256"`
	LocalStateStatus      string                  `json:"localStateStatus"`
	Online                bool                    `json:"online"`
	UsedOfflineSnapshot   bool                    `json:"usedOfflineSnapshot"`
	NativeFallback        bool                    `json:"nativeFallback"`
	Kinds                 []centralSyncKindResult `json:"kinds"`
	CompletedAt           string                  `json:"completedAt"`
	ProductionAuthorized  bool                    `json:"productionAuthorized"`
	TestOptimization      string                  `json:"testOptimization"`
}

type centralStateClient struct {
	baseURL string
	token   string
	http    *http.Client
}

type centralHTTPStatusError struct {
	operation string
	status    int
}

func (err centralHTTPStatusError) Error() string {
	return fmt.Sprintf("central state %s returned HTTP %d", err.operation, err.status)
}

func runCentralConnect(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = io.WriteString(stderr, centralConnectUsage)
		return exitUsage
	}
	serverURL, err := canonicalCentralServerURL(args[0])
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: central connection unavailable: %v\n", err)
		return exitConfiguration
	}
	flags := flag.NewFlagSet("buildopt connect", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	tokenPath := flags.String("token-file", "", "owner-issued central token document")
	caPath := flags.String("ca-file", "", "optional trusted PEM certificate bundle")
	stateDirectory := flags.String("state-dir", optimizeDefaultStateDir, "generated repository-local optimize state")
	connectionDirectory := flags.String("connection-dir", centralConnectionDir, "private repository-local central connection")
	if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 || *tokenPath == "" {
		_, _ = io.WriteString(stderr, centralConnectUsage)
		return exitUsage
	}
	repositoryRoot, err := canonicalWorkingDirectory()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: central connection unavailable: %v\n", err)
		return exitConfiguration
	}
	statePath, stateRelative, err := resolveOptimizeStateDirectory(repositoryRoot, *stateDirectory, false)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: central connection unavailable: %v\n", err)
		return exitConfiguration
	}
	connectionPath, _, err := resolveCentralConnectionDirectory(repositoryRoot, *connectionDirectory, false)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: central connection unavailable: %v\n", err)
		return exitConfiguration
	}
	repositoryID := optimizeRepositoryID(repositoryRoot, os.Getenv)
	repositoryScope := optimizePortfolioRepositoryScope(repositoryID)
	token, tokenDocument, err := readCentralTokenFile(*tokenPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: central connection unavailable: %v\n", err)
		return exitConfiguration
	}
	defer clear(token)
	if tokenDocument != nil && (tokenDocument.RepositoryScopeSHA256 != repositoryScope ||
		tokenDocument.Repository != repositoryID || !centralHasCapability(tokenDocument.Capabilities, sharedcache.CentralStateRead)) {
		_, _ = fmt.Fprintln(stderr, "buildopt: central connection unavailable: token scope does not match this repository or lacks state-read")
		return exitConfiguration
	}
	ca, err := readCentralCAFile(*caPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: central connection unavailable: %v\n", err)
		return exitConfiguration
	}
	client, err := newCentralStateClient(serverURL, string(token), ca)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: central connection unavailable: %v\n", err)
		return exitConfiguration
	}
	if err := client.probe(context.Background(), repositoryScope); err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: central connection unavailable: %v\n", err)
		return 1
	}
	if err := ensurePrivateDirectory(connectionPath, true); err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: central connection unavailable: %v\n", err)
		return exitConfiguration
	}
	if err := writePrivateAtomicFile(filepath.Join(connectionPath, centralTokenFile), token); err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: central connection unavailable: persist token: %v\n", err)
		return exitConfiguration
	}
	caFile := ""
	if len(ca) > 0 {
		caFile = centralCAFile
		if err := writePrivateAtomicFile(filepath.Join(connectionPath, caFile), ca); err != nil {
			_, _ = fmt.Fprintf(stderr, "buildopt: central connection unavailable: persist CA: %v\n", err)
			return exitConfiguration
		}
	}
	connection := centralConnection{
		SchemaVersion: centralConnectionSchema, ServerURL: serverURL,
		RepositoryID: repositoryID, RepositoryScopeSHA256: repositoryScope,
		StateDirectory: stateRelative, TokenFile: centralTokenFile, CAFile: caFile,
		ConnectedAt:          time.Now().UTC().Format(time.RFC3339Nano),
		ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
	}
	if err := writeCanonicalPrivateJSON(filepath.Join(connectionPath, centralConnectionFile), connection); err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: central connection unavailable: persist connection: %v\n", err)
		return exitConfiguration
	}
	result, err := synchronizeCentralState(context.Background(), "CONNECT", repositoryRoot, statePath, connectionPath, connection, client)
	if writeErr := writeCentralSyncResult(stdout, result); writeErr != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: central synchronization result unavailable: %v\n", writeErr)
		return 1
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: central state synchronization incomplete: %v\n", err)
		return 1
	}
	return 0
}

func runCentralSync(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("buildopt sync", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	connectionDirectory := flags.String("connection-dir", centralConnectionDir, "private repository-local central connection")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		_, _ = io.WriteString(stderr, centralConnectUsage)
		return exitUsage
	}
	repositoryRoot, err := canonicalWorkingDirectory()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: central state synchronization unavailable: %v\n", err)
		return exitConfiguration
	}
	connectionPath, _, err := resolveCentralConnectionDirectory(repositoryRoot, *connectionDirectory, false)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: central state synchronization unavailable: %v\n", err)
		return exitConfiguration
	}
	connection, err := loadCentralConnection(repositoryRoot, connectionPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: central state synchronization unavailable: %v\n", err)
		return exitConfiguration
	}
	statePath, _, err := resolveOptimizeStateDirectory(repositoryRoot, connection.StateDirectory, false)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: central state synchronization unavailable: %v\n", err)
		return exitConfiguration
	}
	token, err := readPrivateCentralCredential(filepath.Join(connectionPath, connection.TokenFile), centralMaximumConfig)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: central state synchronization unavailable: %v\n", err)
		return exitConfiguration
	}
	defer clear(token)
	var ca []byte
	if connection.CAFile != "" {
		ca, err = readPrivateCentralCredential(filepath.Join(connectionPath, connection.CAFile), centralMaximumCA)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "buildopt: central state synchronization unavailable: %v\n", err)
			return exitConfiguration
		}
	}
	client, err := newCentralStateClient(connection.ServerURL, string(token), ca)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: central state synchronization unavailable: %v\n", err)
		return exitConfiguration
	}
	result, syncErr := synchronizeCentralState(context.Background(), "SYNC", repositoryRoot, statePath, connectionPath, connection, client)
	if err := writeCentralSyncResult(stdout, result); err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: central synchronization result unavailable: %v\n", err)
		return 1
	}
	if syncErr != nil && !result.UsedOfflineSnapshot {
		_, _ = fmt.Fprintf(stderr, "buildopt: central state synchronization incomplete: %v\n", syncErr)
		return 1
	}
	return 0
}

func synchronizeCentralState(
	ctx context.Context,
	operation string,
	repositoryRoot string,
	stateDirectory string,
	connectionDirectory string,
	connection centralConnection,
	client *centralStateClient,
) (centralSyncResult, error) {
	result := centralSyncResult{
		SchemaVersion: centralSyncResultSchema, Operation: operation,
		ServerURL: connection.ServerURL, RepositoryID: connection.RepositoryID,
		RepositoryScopeSHA256: connection.RepositoryScopeSHA256,
		Online:                true, Kinds: []centralSyncKindResult{},
		CompletedAt:          time.Now().UTC().Format(time.RFC3339Nano),
		ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
	}
	local, localErr := collectCentralLocalPublications(
		repositoryRoot, stateDirectory, connection.StateDirectory,
		connection.RepositoryID, connection.RepositoryScopeSHA256,
	)
	result.LocalStateStatus = "ABSENT"
	if localErr != nil {
		result.LocalStateStatus = "INCOMPATIBLE"
		result.NativeFallback = true
		local = map[sharedcache.StateKind]*centralLocalPublication{}
	} else if len(local) > 0 {
		result.LocalStateStatus = "VALID"
	}
	kinds := []sharedcache.StateKind{
		sharedcache.StateKindEvidence,
		sharedcache.StateKindPortfolio,
		sharedcache.StateKindCheckpoint,
	}
	var evidenceManifest string
	for _, kind := range kinds {
		publication := local[kind]
		kindResult, snapshot, err := synchronizeCentralKind(
			ctx, client, connectionDirectory, connection.RepositoryScopeSHA256,
			publication, kind, evidenceManifest,
		)
		result.Kinds = append(result.Kinds, kindResult)
		if kind == sharedcache.StateKindEvidence && snapshot != nil {
			evidenceManifest = snapshot.manifestSHA256
		}
		if err != nil {
			result.Online = false
			result.NativeFallback = true
			result.UsedOfflineSnapshot = applyVerifiedOfflineSnapshots(
				connectionDirectory, connection.RepositoryScopeSHA256,
				result.Kinds, kinds[len(result.Kinds):], &result,
			)
			result.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
			return result, err
		}
		if kindResult.Status == "INCOMPATIBLE_REMOTE_RETAINED" ||
			kindResult.Status == "CONCURRENT_REMOTE_WON" {
			result.NativeFallback = true
		}
	}
	result.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
	return result, nil
}

func synchronizeCentralKind(
	ctx context.Context,
	client *centralStateClient,
	connectionDirectory string,
	repositoryScope string,
	local *centralLocalPublication,
	kind sharedcache.StateKind,
	evidenceManifest string,
) (centralSyncKindResult, *centralRemoteSnapshot, error) {
	result := centralSyncKindResult{
		Kind: kind, Status: "EMPTY", LocalPresent: local != nil,
		ProductionAllowed: false,
	}
	remote, found, err := client.loadSnapshot(ctx, repositoryScope, kind)
	if err != nil {
		result.Status = "REMOTE_UNAVAILABLE"
		return result, nil, err
	}
	if found {
		result.RemoteGeneration = remote.head.Generation
		result.Compatible = local == nil || local.compatibility == remote.manifest.CompatibilitySHA256
	}
	if local == nil {
		if !found {
			return result, nil, nil
		}
		if err := storeCentralSnapshot(connectionDirectory, remote); err != nil {
			return result, nil, err
		}
		result.Status = "PULLED"
		result.SnapshotVerified = true
		return result, remote, nil
	}
	if found && local.bundleSHA256 == remote.bundleSHA256 {
		if err := storeCentralSnapshot(connectionDirectory, remote); err != nil {
			return result, nil, err
		}
		result.Status = "NO_CHANGE"
		result.SnapshotVerified = true
		result.Compatible = true
		return result, remote, nil
	}
	if found && !local.createdAt.After(parseCentralTimestamp(remote.manifest.CreatedAt)) {
		if err := storeCentralSnapshot(connectionDirectory, remote); err != nil {
			return result, nil, err
		}
		result.Status = "INCOMPATIBLE_REMOTE_RETAINED"
		if result.Compatible {
			result.Status = "REMOTE_NEWER_RETAINED"
		}
		result.SnapshotVerified = true
		return result, remote, nil
	}
	if kind == sharedcache.StateKindPortfolio && evidenceManifest == "" {
		result.Status = "EVIDENCE_REQUIRED"
		return result, remote, nil
	}
	manifest, manifestRaw, manifestSHA, err := buildCentralStateManifest(local, remote, evidenceManifest)
	if err != nil {
		return result, remote, err
	}
	created, err := client.putObject(ctx, repositoryScope, kind, local.bundleSHA256, local.bundleRaw)
	if err != nil {
		if centralStatus(err) == http.StatusForbidden {
			result.Status = "REMOTE_READ_ONLY"
			if found && storeCentralSnapshot(connectionDirectory, remote) == nil {
				result.SnapshotVerified = true
			}
			return result, remote, nil
		}
		result.Status = "REMOTE_UNAVAILABLE"
		return result, remote, err
	}
	if created {
		result.ObjectsUploaded = 1
	}
	created, err = client.putManifest(ctx, repositoryScope, kind, manifestSHA, manifestRaw)
	if err != nil {
		if centralStatus(err) == http.StatusConflict {
			return retainCentralConcurrentWinner(
				ctx, client, connectionDirectory, repositoryScope, kind, local, remote, result, err,
			)
		}
		result.Status = "REMOTE_UNAVAILABLE"
		return result, remote, err
	}
	result.ManifestUploaded = created
	next, err := client.advanceHead(ctx, repositoryScope, kind, manifest, manifestSHA, remote)
	if err != nil {
		if centralStatus(err) == http.StatusPreconditionFailed || centralStatus(err) == http.StatusConflict {
			return retainCentralConcurrentWinner(
				ctx, client, connectionDirectory, repositoryScope, kind, local, remote, result, err,
			)
		}
		result.Status = "REMOTE_UNAVAILABLE"
		return result, remote, err
	}
	if err := storeCentralSnapshot(connectionDirectory, next); err != nil {
		return result, next, err
	}
	result.Status = "PUSHED"
	result.RemoteGeneration = next.head.Generation
	result.Compatible = true
	result.SnapshotVerified = true
	result.HeadAdvanced = true
	return result, next, nil
}

func retainCentralConcurrentWinner(
	ctx context.Context,
	client *centralStateClient,
	connectionDirectory string,
	repositoryScope string,
	kind sharedcache.StateKind,
	local *centralLocalPublication,
	previous *centralRemoteSnapshot,
	result centralSyncKindResult,
	conflict error,
) (centralSyncKindResult, *centralRemoteSnapshot, error) {
	previousGeneration := int64(0)
	if previous != nil {
		previousGeneration = previous.head.Generation
	}
	for attempt := 0; attempt < 20; attempt++ {
		winner, found, err := client.loadSnapshot(ctx, repositoryScope, kind)
		if err != nil {
			return result, previous, errors.Join(conflict, err)
		}
		if found && winner.head.Generation > previousGeneration {
			if err := storeCentralSnapshot(connectionDirectory, winner); err != nil {
				return result, winner, err
			}
			result.Status = "CONCURRENT_REMOTE_WON"
			result.RemoteGeneration = winner.head.Generation
			result.Compatible = winner.manifest.CompatibilitySHA256 == local.compatibility
			result.SnapshotVerified = true
			return result, winner, nil
		}
		select {
		case <-ctx.Done():
			return result, previous, errors.Join(conflict, ctx.Err())
		case <-time.After(25 * time.Millisecond):
		}
	}
	return result, previous, errors.Join(conflict, errors.New("concurrent central state winner was not published"))
}

func buildCentralStateManifest(
	local *centralLocalPublication,
	remote *centralRemoteSnapshot,
	evidenceManifest string,
) (sharedcache.StateManifest, []byte, string, error) {
	generation := int64(1)
	if remote != nil {
		generation = remote.head.Generation + 1
	}
	role := map[sharedcache.StateKind]string{
		sharedcache.StateKindEvidence:   "CALIBRATION_EVIDENCE",
		sharedcache.StateKindPortfolio:  "PORTFOLIO_INDEX",
		sharedcache.StateKindCheckpoint: "OPTIMIZE_STATE",
	}[local.kind]
	manifest := sharedcache.StateManifest{
		SchemaVersion: "buildopt.central/state-manifest/v1",
		RecordType:    "CENTRAL_STATE_MANIFEST", Kind: local.kind,
		RepositoryScopeSHA256: local.bundle.RepositoryScopeSHA256,
		Generation:            generation, CompatibilitySHA256: local.compatibility,
		BindingsSHA256: local.bindings, Origin: local.origin,
		Artifacts: []sharedcache.StateArtifact{{
			Role: role, SHA256: local.bundleSHA256, SizeBytes: int64(len(local.bundleRaw)),
			PayloadSchemaVersion: centralBundleSchema,
		}},
		References: []sharedcache.StateReference{},
		Status:     "COMPLETE", RetentionClass: "WHILE_REFERENCED_PLUS_30_DAYS",
		CreatedAt: local.createdAt.UTC().Format(time.RFC3339Nano),
		Authority: sharedcache.StateAuthority{
			SelectionRequiresLocalRevalidation: true,
			ProductionAuthorized:               false, TestOptimization: "OUT_OF_SCOPE",
		},
	}
	switch local.kind {
	case sharedcache.StateKindPortfolio:
		manifest.RetentionClass = "CURRENT_PLUS_30_DAYS_AFTER_SUPERSEDED"
		manifest.References = []sharedcache.StateReference{{
			Kind: sharedcache.StateKindEvidence, ManifestSHA256: evidenceManifest,
			Relation: "QUALIFICATION",
		}}
	case sharedcache.StateKindCheckpoint:
		manifest.Status = "RESUMABLE"
		manifest.RetentionClass = "24_HOURS_FROM_CREATED_AT"
		manifest.ExpiresAt = local.createdAt.Add(24 * time.Hour).UTC().Format(time.RFC3339Nano)
	}
	raw, digest, err := sharedcache.CanonicalCentralStateValue(manifest)
	return manifest, raw, digest, err
}

func collectCentralLocalPublications(
	repositoryRoot string,
	stateDirectory string,
	stateRelative string,
	repositoryID string,
	repositoryScope string,
) (map[sharedcache.StateKind]*centralLocalPublication, error) {
	publications := map[sharedcache.StateKind]*centralLocalPublication{}
	statePath := filepath.Join(stateDirectory, optimizeStateFile)
	raw, err := os.ReadFile(statePath)
	if errors.Is(err, os.ErrNotExist) {
		return publications, nil
	}
	if err != nil || len(raw) < 1 || len(raw) > optimizeMaximumStateBytes {
		return publications, errors.New("local optimize checkpoint is unreadable")
	}
	var state optimizeState
	if decodeCentralStrictJSON(raw, &state) != nil || !validOptimizeState(state) {
		return publications, errors.New("local optimize checkpoint is invalid")
	}
	if !validMeasurementRevision(state.Discovery.BaseRevision) ||
		!validMeasurementRevision(state.Discovery.TargetRevision) {
		return publications, errors.New("local optimize state has no publishable revision origin")
	}
	gradleVersion, err := centralGradleVersion(repositoryRoot)
	if err != nil {
		return publications, err
	}
	createdAt, _ := time.Parse(time.RFC3339Nano, state.UpdatedAt)
	origin := sharedcache.StateOrigin{
		BaseRevision: state.Discovery.BaseRevision, TargetRevision: state.Discovery.TargetRevision,
		BuildOptExecutableSHA256: state.Bindings.ExecutableSHA256,
		WrapperSHA256:            state.Bindings.WrapperSHA256, GradleVersion: gradleVersion,
	}
	checkpointFiles := map[string][]byte{optimizeStateFile: raw}
	checkpointCompatibility := state.Bindings.SHA256
	checkpoint, err := newCentralLocalPublication(
		sharedcache.StateKindCheckpoint, repositoryScope,
		checkpointCompatibility, state.Bindings.SHA256, origin, createdAt,
		checkpointFiles,
	)
	if err != nil {
		return publications, err
	}
	publications[sharedcache.StateKindCheckpoint] = checkpoint

	indexRelative := filepath.ToSlash(filepath.Join(stateRelative, "portfolio", optimizePortfolioIndexFile))
	indexRaw, err := readOptimizePortfolioSource(repositoryRoot, indexRelative)
	if errors.Is(err, os.ErrNotExist) {
		return publications, nil
	}
	if err != nil {
		return publications, errors.New("local optimize portfolio is unreadable")
	}
	var portfolio optimizeProfilePortfolio
	if decodeCentralStrictJSON(indexRaw, &portfolio) != nil ||
		portfolio.SchemaVersion != optimizePortfolioSchemaVersion ||
		portfolio.RepositoryScopeSHA256 != optimizePortfolioRepositoryScope(repositoryID) ||
		portfolio.ProductionAuthorized || portfolio.SelectionAuthorized ||
		len(portfolio.Profiles) < 1 || len(portfolio.Profiles) > optimizePortfolioMaximumEntries {
		return publications, errors.New("local optimize portfolio is invalid")
	}
	portfolioFiles := map[string][]byte{
		filepath.ToSlash(filepath.Join("portfolio", optimizePortfolioIndexFile)): indexRaw,
	}
	evidenceFiles := map[string][]byte{}
	families := make([]string, 0, len(portfolio.Profiles))
	for _, entry := range portfolio.Profiles {
		if !validOptimizeSHA(entry.FamilySHA256) || !validOptimizeSHA(entry.ProfileSHA256) ||
			!validOptimizeSHA(entry.ManifestSHA256) || !validOptimizeSHA(entry.GraphSHA256) ||
			!validOptimizeSHA(entry.GeneratedSHA256) || !validOptimizeSHA(entry.EvidenceSHA256) ||
			entry.RepositoryID != repositoryID || entry.State != "QUALIFIED" ||
			entry.ProductionAuthorized || entry.SelectionAuthorized ||
			len(entry.ChangedProjects) < 1 || len(entry.Entrypoints) < 1 ||
			len(entry.CandidateEntrypoints) < 1 || len(entry.RequiredOutputs) < 1 {
			return publications, errors.New("local optimize portfolio entry is invalid")
		}
		families = append(families, entry.FamilySHA256)
		directory := filepath.ToSlash(filepath.Dir(entry.ProfilePath))
		bindings := []struct {
			path   string
			digest string
		}{
			{filepath.ToSlash(filepath.Join(directory, "manifest.json")), entry.ManifestSHA256},
			{filepath.ToSlash(filepath.Join(directory, "graph.json")), entry.GraphSHA256},
			{filepath.ToSlash(filepath.Join(directory, "generated-manifest.json")), entry.GeneratedSHA256},
			{filepath.ToSlash(filepath.Join(directory, "evidence.json")), entry.EvidenceSHA256},
			{entry.ProfilePath, entry.ProfileSHA256},
		}
		for _, binding := range bindings {
			fileRaw, readErr := readOptimizePortfolioSource(repositoryRoot, binding.path)
			if readErr != nil {
				return publications, errors.New("local optimize portfolio artifact is unreadable")
			}
			digest := sha256.Sum256(fileRaw)
			if hex.EncodeToString(digest[:]) != binding.digest {
				return publications, errors.New("local optimize portfolio artifact digest is invalid")
			}
			relative, ok := centralStateRelativePath(stateRelative, binding.path)
			if !ok {
				return publications, errors.New("local optimize portfolio artifact escapes its state directory")
			}
			portfolioFiles[relative] = fileRaw
			if filepath.Base(binding.path) == "evidence.json" {
				evidenceFiles[relative] = fileRaw
			}
		}
	}
	sort.Strings(families)
	portfolioCompatibility := optimizeDigest(
		"buildopt-central-portfolio-compatibility-v1",
		append([]string{repositoryScope, state.Bindings.WrapperSHA256, state.Bindings.ExecutableSHA256}, families...)...,
	)
	evidenceCompatibility := optimizeDigest(
		"buildopt-central-evidence-compatibility-v1",
		repositoryScope, state.Bindings.WrapperSHA256, state.Calibration.DiscoverySHA256,
	)
	evidence, err := newCentralLocalPublication(
		sharedcache.StateKindEvidence, repositoryScope, evidenceCompatibility,
		state.Bindings.SHA256, origin, createdAt, evidenceFiles,
	)
	if err != nil {
		return publications, err
	}
	portfolioPublication, err := newCentralLocalPublication(
		sharedcache.StateKindPortfolio, repositoryScope, portfolioCompatibility,
		state.Bindings.SHA256, origin, createdAt, portfolioFiles,
	)
	if err != nil {
		return publications, err
	}
	publications[sharedcache.StateKindEvidence] = evidence
	publications[sharedcache.StateKindPortfolio] = portfolioPublication
	return publications, nil
}

func newCentralLocalPublication(
	kind sharedcache.StateKind,
	repositoryScope string,
	compatibility string,
	bindings string,
	origin sharedcache.StateOrigin,
	createdAt time.Time,
	files map[string][]byte,
) (*centralLocalPublication, error) {
	if len(files) < 1 || len(files) > 64 {
		return nil, errors.New("central state bundle has no bounded files")
	}
	bundle := centralStateBundle{
		SchemaVersion: centralBundleSchema, RecordType: "BUILDOPT_CENTRAL_STATE_BUNDLE",
		Kind: kind, RepositoryScopeSHA256: repositoryScope,
		CompatibilitySHA256: compatibility, BindingsSHA256: bindings,
		Files: []centralStateBundleFile{}, CreatedAt: createdAt.UTC().Format(time.RFC3339Nano),
		ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
	}
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		raw := files[path]
		if !validCentralBundlePath(path) || len(raw) < 1 || len(raw) > centralMaximumArtifact {
			return nil, errors.New("central state bundle contains an invalid file")
		}
		var header struct {
			SchemaVersion string `json:"schemaVersion"`
		}
		if json.Unmarshal(raw, &header) != nil || !centralBundleSchemaPattern.MatchString(header.SchemaVersion) {
			return nil, errors.New("central state bundle file has no supported schema version")
		}
		digest := sha256.Sum256(raw)
		bundle.Files = append(bundle.Files, centralStateBundleFile{
			Path: path, SHA256: hex.EncodeToString(digest[:]), SizeBytes: int64(len(raw)),
			PayloadSchemaVersion: header.SchemaVersion,
			ContentBase64:        base64.RawStdEncoding.EncodeToString(raw),
		})
	}
	raw, err := json.Marshal(bundle)
	if err != nil {
		return nil, err
	}
	canonical, err := contractcrypto.CanonicalizeJCS(raw)
	if err != nil || len(canonical) > centralMaximumArtifact {
		return nil, errors.New("central state bundle cannot be canonicalized within its limit")
	}
	digest := sha256.Sum256(canonical)
	return &centralLocalPublication{
		kind: kind, bundle: bundle, bundleRaw: canonical,
		bundleSHA256: hex.EncodeToString(digest[:]), compatibility: compatibility,
		bindings: bindings, origin: origin, createdAt: createdAt.UTC(),
	}, nil
}

func (client *centralStateClient) probe(ctx context.Context, repositoryScope string) error {
	for _, kind := range []sharedcache.StateKind{
		sharedcache.StateKindEvidence,
		sharedcache.StateKindPortfolio,
		sharedcache.StateKindCheckpoint,
	} {
		_, _, err := client.get(ctx, centralStatePath(repositoryScope, kind, "head", ""), centralMaximumDocument)
		if err == nil || centralStatus(err) == http.StatusNotFound {
			continue
		}
		return err
	}
	return nil
}

func (client *centralStateClient) loadSnapshot(
	ctx context.Context,
	repositoryScope string,
	kind sharedcache.StateKind,
) (*centralRemoteSnapshot, bool, error) {
	headRaw, headETag, err := client.get(ctx, centralStatePath(repositoryScope, kind, "head", ""), centralMaximumDocument)
	if centralStatus(err) == http.StatusNotFound {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	head, headSHA, err := sharedcache.DecodeCentralStateHead(headRaw)
	if err != nil || headETag != headSHA || head.RepositoryScopeSHA256 != repositoryScope || head.Kind != kind {
		return nil, false, errors.New("central state head failed canonical verification")
	}
	manifestRaw, manifestETag, err := client.get(
		ctx,
		centralStatePath(repositoryScope, kind, "manifests", head.ManifestSHA256),
		centralMaximumDocument,
	)
	if err != nil {
		return nil, false, err
	}
	manifest, manifestSHA, err := sharedcache.DecodeCentralStateManifest(manifestRaw)
	if err != nil || manifestETag != manifestSHA || manifestSHA != head.ManifestSHA256 ||
		manifest.RepositoryScopeSHA256 != repositoryScope || manifest.Kind != kind ||
		manifest.Generation != head.Generation || manifest.CompatibilitySHA256 != head.CompatibilitySHA256 ||
		len(manifest.Artifacts) != 1 || manifest.Artifacts[0].PayloadSchemaVersion != centralBundleSchema {
		return nil, false, errors.New("central state manifest failed canonical verification")
	}
	artifact := manifest.Artifacts[0]
	bundleRaw, bundleETag, err := client.get(
		ctx,
		centralStatePath(repositoryScope, kind, "objects", artifact.SHA256),
		centralMaximumArtifact,
	)
	if err != nil {
		return nil, false, err
	}
	if int64(len(bundleRaw)) != artifact.SizeBytes || bundleETag != artifact.SHA256 {
		return nil, false, errors.New("central state bundle length or identity drift")
	}
	bundle, bundleSHA, err := decodeCentralStateBundle(bundleRaw)
	if err != nil || bundleSHA != artifact.SHA256 || bundle.RepositoryScopeSHA256 != repositoryScope ||
		bundle.Kind != kind || bundle.CompatibilitySHA256 != manifest.CompatibilitySHA256 ||
		bundle.BindingsSHA256 != manifest.BindingsSHA256 {
		return nil, false, errors.New("central state bundle failed verification")
	}
	return &centralRemoteSnapshot{
		head: head, headRaw: headRaw, headSHA256: headSHA,
		manifest: manifest, manifestRaw: manifestRaw, manifestSHA256: manifestSHA,
		bundle: bundle, bundleRaw: bundleRaw, bundleSHA256: bundleSHA,
	}, true, nil
}

func (client *centralStateClient) putObject(
	ctx context.Context,
	repositoryScope string,
	kind sharedcache.StateKind,
	digest string,
	raw []byte,
) (bool, error) {
	status, etag, err := client.sendImmutable(
		ctx, centralStatePath(repositoryScope, kind, "objects", digest), raw,
	)
	if err != nil {
		return false, err
	}
	if (status != http.StatusCreated && status != http.StatusOK) || etag != digest {
		return false, centralHTTPStatusError{operation: "put object", status: status}
	}
	return status == http.StatusCreated, nil
}

func (client *centralStateClient) putManifest(
	ctx context.Context,
	repositoryScope string,
	kind sharedcache.StateKind,
	digest string,
	raw []byte,
) (bool, error) {
	status, etag, err := client.sendImmutable(
		ctx, centralStatePath(repositoryScope, kind, "manifests", digest), raw,
	)
	if err != nil {
		return false, err
	}
	if (status != http.StatusCreated && status != http.StatusOK) || etag != digest {
		return false, centralHTTPStatusError{operation: "put manifest", status: status}
	}
	return status == http.StatusCreated, nil
}

func (client *centralStateClient) sendImmutable(
	ctx context.Context,
	path string,
	raw []byte,
) (int, string, error) {
	for attempt := 0; attempt < 20; attempt++ {
		status, etag, err := client.send(
			ctx, http.MethodPut, path, raw,
			map[string]string{"If-None-Match": "*"}, centralMaximumDocument,
		)
		if centralStatus(err) != http.StatusServiceUnavailable {
			return status, etag, err
		}
		select {
		case <-ctx.Done():
			return 0, "", errors.Join(err, ctx.Err())
		case <-time.After(25 * time.Millisecond):
		}
	}
	return 0, "", centralHTTPStatusError{operation: "put", status: http.StatusServiceUnavailable}
}

func (client *centralStateClient) advanceHead(
	ctx context.Context,
	repositoryScope string,
	kind sharedcache.StateKind,
	manifest sharedcache.StateManifest,
	manifestSHA string,
	previous *centralRemoteSnapshot,
) (*centralRemoteSnapshot, error) {
	expectedGeneration := int64(0)
	expectedHead := ""
	previousManifest := ""
	if previous != nil {
		expectedGeneration = previous.head.Generation
		expectedHead = previous.headSHA256
		previousManifest = previous.manifestSHA256
	}
	head := sharedcache.StateHead{
		SchemaVersion: "buildopt.central/state-head/v1", RecordType: "CENTRAL_STATE_HEAD",
		Kind: kind, RepositoryScopeSHA256: repositoryScope,
		Generation: expectedGeneration + 1, ManifestSHA256: manifestSHA,
		PreviousManifestSHA256: previousManifest,
		CompatibilitySHA256:    manifest.CompatibilitySHA256,
		UpdatedAt:              manifest.CreatedAt, Authority: manifest.Authority,
	}
	var expectedHeadValue *string
	if expectedHead != "" {
		expectedHeadValue = &expectedHead
	}
	document := struct {
		SchemaVersion      string                `json:"schemaVersion"`
		RecordType         string                `json:"recordType"`
		Operation          string                `json:"operation"`
		IdempotencyKey     string                `json:"idempotencyKey"`
		ExpectedGeneration int64                 `json:"expectedGeneration"`
		ExpectedHeadSHA256 *string               `json:"expectedHeadSha256"`
		Next               sharedcache.StateHead `json:"next"`
	}{
		SchemaVersion: "buildopt.central/state-cas/v1", RecordType: "CENTRAL_STATE_CAS",
		Operation: "CREATE_OR_ADVANCE",
		IdempotencyKey: optimizeDigest(
			"buildopt-central-state-cas-v1", repositoryScope, string(kind),
			strconv.FormatInt(expectedGeneration, 10), expectedHead, manifestSHA,
		),
		ExpectedGeneration: expectedGeneration, ExpectedHeadSHA256: expectedHeadValue,
		Next: head,
	}
	raw, _, err := sharedcache.CanonicalCentralStateValue(document)
	if err != nil {
		return nil, err
	}
	headers := map[string]string{"If-None-Match": "*"}
	if expectedGeneration > 0 {
		delete(headers, "If-None-Match")
		headers["If-Match"] = `"` + expectedHead + `"`
	}
	status, err := client.sendCAS(
		ctx, centralStatePath(repositoryScope, kind, "head:cas", ""), raw, headers,
	)
	if err != nil {
		return nil, err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return nil, centralHTTPStatusError{operation: "advance head", status: status}
	}
	loaded, found, err := client.loadSnapshot(ctx, repositoryScope, kind)
	if err != nil || !found || loaded.manifestSHA256 != manifestSHA {
		return nil, errors.Join(errors.New("central state head did not expose the committed manifest"), err)
	}
	return loaded, nil
}

func (client *centralStateClient) sendCAS(
	ctx context.Context,
	path string,
	raw []byte,
	headers map[string]string,
) (int, error) {
	for attempt := 0; attempt < 20; attempt++ {
		status, _, err := client.send(
			ctx, http.MethodPost, path, raw, headers, centralMaximumDocument,
		)
		if centralStatus(err) != http.StatusServiceUnavailable {
			return status, err
		}
		select {
		case <-ctx.Done():
			return 0, errors.Join(err, ctx.Err())
		case <-time.After(25 * time.Millisecond):
		}
	}
	return 0, centralHTTPStatusError{operation: "post", status: http.StatusServiceUnavailable}
}

func (client *centralStateClient) get(
	ctx context.Context,
	path string,
	maximum int64,
) ([]byte, string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.baseURL+path, nil)
	if err != nil {
		return nil, "", err
	}
	request.Header.Set("Authorization", "Bearer "+client.token)
	response, err := client.http.Do(request)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return nil, "", centralHTTPStatusError{operation: "get", status: response.StatusCode}
	}
	if response.ContentLength < 1 || response.ContentLength > maximum {
		return nil, "", errors.New("central state response has an invalid content length")
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, maximum+1))
	if err != nil || len(raw) < 1 || int64(len(raw)) > maximum {
		return nil, "", errors.New("central state response exceeds its bounded size")
	}
	etag, ok := centralETag(response.Header.Get("ETag"))
	if !ok || response.Header.Get("Cache-Control") != "no-store" {
		return nil, "", errors.New("central state response is missing its verified identity")
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != etag {
		return nil, "", errors.New("central state response digest mismatch")
	}
	return raw, etag, nil
}

func (client *centralStateClient) send(
	ctx context.Context,
	method string,
	path string,
	raw []byte,
	headers map[string]string,
	maximumResponse int64,
) (int, string, error) {
	request, err := http.NewRequestWithContext(ctx, method, client.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return 0, "", err
	}
	request.Header.Set("Authorization", "Bearer "+client.token)
	request.Header.Set("Content-Type", "application/json")
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := client.http.Do(request)
	if err != nil {
		return 0, "", err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return 0, "", centralHTTPStatusError{operation: strings.ToLower(method), status: response.StatusCode}
	}
	if response.ContentLength > maximumResponse {
		return 0, "", errors.New("central state write response exceeds its bounded size")
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maximumResponse+1))
	etag, _ := centralETag(response.Header.Get("ETag"))
	return response.StatusCode, etag, nil
}

func newCentralStateClient(serverURL, token string, ca []byte) (*centralStateClient, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(decoded) != sharedcache.CentralTokenBytes {
		return nil, errors.New("central token is not a 32-byte base64url credential")
	}
	clear(decoded)
	configuration := &tls.Config{MinVersion: tls.VersionTLS13}
	if len(ca) > 0 {
		roots, err := x509.SystemCertPool()
		if err != nil {
			roots = x509.NewCertPool()
		}
		if !roots.AppendCertsFromPEM(ca) {
			return nil, errors.New("central CA file contains no trusted certificate")
		}
		configuration.RootCAs = roots
	}
	transport := &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		TLSClientConfig:   configuration,
		ForceAttemptHTTP2: true,
		MaxIdleConns:      4, MaxIdleConnsPerHost: 4,
		IdleConnTimeout: 30 * time.Second,
	}
	return &centralStateClient{
		baseURL: serverURL, token: token,
		http: &http.Client{
			Transport: transport, Timeout: centralRequestTimeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return errors.New("central state redirects are disabled")
			},
		},
	}, nil
}

func storeCentralSnapshot(connectionDirectory string, snapshot *centralRemoteSnapshot) error {
	if snapshot == nil {
		return errors.New("central snapshot is nil")
	}
	directory := filepath.Join(
		connectionDirectory, centralSnapshotDir,
		strings.ToLower(string(snapshot.head.Kind)),
	)
	if err := ensurePrivateDirectory(directory, true); err != nil {
		return err
	}
	for _, file := range []struct {
		name string
		raw  []byte
	}{
		{name: "bundle.json", raw: snapshot.bundleRaw},
		{name: "manifest.json", raw: snapshot.manifestRaw},
		{name: "head.json", raw: snapshot.headRaw},
	} {
		if err := writePrivateAtomicFile(filepath.Join(directory, file.name), file.raw); err != nil {
			return err
		}
	}
	return nil
}

func loadCentralSnapshot(
	connectionDirectory string,
	repositoryScope string,
	kind sharedcache.StateKind,
) (*centralRemoteSnapshot, error) {
	directory := filepath.Join(connectionDirectory, centralSnapshotDir, strings.ToLower(string(kind)))
	read := func(name string, maximum int64) ([]byte, error) {
		return readPrivateCentralCredential(filepath.Join(directory, name), maximum)
	}
	headRaw, err := read("head.json", centralMaximumDocument)
	if err != nil {
		return nil, err
	}
	manifestRaw, err := read("manifest.json", centralMaximumDocument)
	if err != nil {
		return nil, err
	}
	bundleRaw, err := read("bundle.json", centralMaximumArtifact)
	if err != nil {
		return nil, err
	}
	head, headSHA, err := sharedcache.DecodeCentralStateHead(headRaw)
	if err != nil || head.RepositoryScopeSHA256 != repositoryScope || head.Kind != kind {
		return nil, errors.New("offline central head failed verification")
	}
	manifest, manifestSHA, err := sharedcache.DecodeCentralStateManifest(manifestRaw)
	if err != nil || manifestSHA != head.ManifestSHA256 || manifest.RepositoryScopeSHA256 != repositoryScope ||
		manifest.Kind != kind || len(manifest.Artifacts) != 1 {
		return nil, errors.New("offline central manifest failed verification")
	}
	bundle, bundleSHA, err := decodeCentralStateBundle(bundleRaw)
	if err != nil || bundleSHA != manifest.Artifacts[0].SHA256 ||
		int64(len(bundleRaw)) != manifest.Artifacts[0].SizeBytes ||
		bundle.RepositoryScopeSHA256 != repositoryScope || bundle.Kind != kind ||
		bundle.CompatibilitySHA256 != manifest.CompatibilitySHA256 {
		return nil, errors.New("offline central bundle failed verification")
	}
	return &centralRemoteSnapshot{
		head: head, headRaw: headRaw, headSHA256: headSHA,
		manifest: manifest, manifestRaw: manifestRaw, manifestSHA256: manifestSHA,
		bundle: bundle, bundleRaw: bundleRaw, bundleSHA256: bundleSHA,
	}, nil
}

func applyVerifiedOfflineSnapshots(
	connectionDirectory string,
	repositoryScope string,
	existing []centralSyncKindResult,
	remaining []sharedcache.StateKind,
	result *centralSyncResult,
) bool {
	verified := false
	verifiedEvidence := ""
	for index := range existing {
		snapshot, err := loadCentralSnapshot(connectionDirectory, repositoryScope, existing[index].Kind)
		if err == nil && existing[index].Kind == sharedcache.StateKindPortfolio &&
			!centralPortfolioReferencesEvidence(snapshot, verifiedEvidence) {
			err = errors.New("offline central portfolio has no verified evidence snapshot")
		}
		if err != nil {
			existing[index].Status = "OFFLINE_NO_SNAPSHOT"
			existing[index].RemoteGeneration = 0
			existing[index].SnapshotVerified = false
			continue
		}
		existing[index].Status = "OFFLINE_SNAPSHOT"
		existing[index].RemoteGeneration = snapshot.head.Generation
		existing[index].SnapshotVerified = true
		if existing[index].Kind == sharedcache.StateKindEvidence {
			verifiedEvidence = snapshot.manifestSHA256
		}
		verified = true
	}
	result.Kinds = existing
	for _, kind := range remaining {
		kindResult := centralSyncKindResult{Kind: kind, Status: "OFFLINE_NO_SNAPSHOT"}
		snapshot, err := loadCentralSnapshot(connectionDirectory, repositoryScope, kind)
		if err == nil && kind == sharedcache.StateKindPortfolio &&
			!centralPortfolioReferencesEvidence(snapshot, verifiedEvidence) {
			err = errors.New("offline central portfolio has no verified evidence snapshot")
		}
		if err == nil {
			kindResult.Status = "OFFLINE_SNAPSHOT"
			kindResult.RemoteGeneration = snapshot.head.Generation
			kindResult.SnapshotVerified = true
			if kind == sharedcache.StateKindEvidence {
				verifiedEvidence = snapshot.manifestSHA256
			}
			verified = true
		}
		result.Kinds = append(result.Kinds, kindResult)
	}
	return verified
}

func centralPortfolioReferencesEvidence(snapshot *centralRemoteSnapshot, evidenceManifest string) bool {
	return snapshot != nil && evidenceManifest != "" && len(snapshot.manifest.References) == 1 &&
		snapshot.manifest.References[0].Kind == sharedcache.StateKindEvidence &&
		snapshot.manifest.References[0].ManifestSHA256 == evidenceManifest &&
		snapshot.manifest.References[0].Relation == "QUALIFICATION"
}

func decodeCentralStateBundle(raw []byte) (centralStateBundle, string, error) {
	canonical, err := contractcrypto.CanonicalizeJCS(raw)
	if err != nil || !bytes.Equal(canonical, raw) {
		return centralStateBundle{}, "", errors.New("central state bundle is not canonical")
	}
	var bundle centralStateBundle
	if decodeCentralStrictJSON(raw, &bundle) != nil ||
		bundle.SchemaVersion != centralBundleSchema || bundle.RecordType != "BUILDOPT_CENTRAL_STATE_BUNDLE" ||
		!validOptimizeSHA(bundle.RepositoryScopeSHA256) || !validOptimizeSHA(bundle.CompatibilitySHA256) ||
		!validOptimizeSHA(bundle.BindingsSHA256) || bundle.ProductionAuthorized ||
		bundle.TestOptimization != "OUT_OF_SCOPE" || len(bundle.Files) < 1 || len(bundle.Files) > 64 {
		return centralStateBundle{}, "", errors.New("central state bundle is invalid")
	}
	if bundle.Kind != sharedcache.StateKindEvidence && bundle.Kind != sharedcache.StateKindPortfolio &&
		bundle.Kind != sharedcache.StateKindCheckpoint {
		return centralStateBundle{}, "", errors.New("central state bundle kind is invalid")
	}
	if _, err := time.Parse(time.RFC3339Nano, bundle.CreatedAt); err != nil {
		return centralStateBundle{}, "", errors.New("central state bundle timestamp is invalid")
	}
	previous := ""
	for _, file := range bundle.Files {
		if !validCentralBundlePath(file.Path) || file.Path <= previous || !validOptimizeSHA(file.SHA256) ||
			file.SizeBytes < 1 || file.SizeBytes > centralMaximumArtifact ||
			!centralBundleSchemaPattern.MatchString(file.PayloadSchemaVersion) {
			return centralStateBundle{}, "", errors.New("central state bundle file binding is invalid")
		}
		decoded, err := base64.RawStdEncoding.DecodeString(file.ContentBase64)
		if err != nil || int64(len(decoded)) != file.SizeBytes {
			return centralStateBundle{}, "", errors.New("central state bundle file encoding is invalid")
		}
		digest := sha256.Sum256(decoded)
		clear(decoded)
		if hex.EncodeToString(digest[:]) != file.SHA256 {
			return centralStateBundle{}, "", errors.New("central state bundle file digest mismatch")
		}
		previous = file.Path
	}
	digest := sha256.Sum256(raw)
	return bundle, hex.EncodeToString(digest[:]), nil
}

func loadCentralConnection(repositoryRoot, connectionDirectory string) (centralConnection, error) {
	raw, err := readPrivateCentralCredential(filepath.Join(connectionDirectory, centralConnectionFile), centralMaximumConfig)
	if err != nil {
		return centralConnection{}, err
	}
	var connection centralConnection
	if decodeCentralStrictJSON(raw, &connection) != nil || connection.SchemaVersion != centralConnectionSchema ||
		connection.ProductionAuthorized || connection.TestOptimization != "OUT_OF_SCOPE" ||
		!validOptimizeSHA(connection.RepositoryScopeSHA256) || connection.TokenFile != centralTokenFile ||
		(connection.CAFile != "" && connection.CAFile != centralCAFile) ||
		!validOptimizeStateRelative(connection.StateDirectory) {
		return centralConnection{}, errors.New("central connection document is invalid")
	}
	serverURL, err := canonicalCentralServerURL(connection.ServerURL)
	if err != nil || serverURL != connection.ServerURL {
		return centralConnection{}, errors.New("central connection URL is invalid")
	}
	if _, err := time.Parse(time.RFC3339Nano, connection.ConnectedAt); err != nil {
		return centralConnection{}, errors.New("central connection timestamp is invalid")
	}
	repositoryID := optimizeRepositoryID(repositoryRoot, os.Getenv)
	if connection.RepositoryID != repositoryID ||
		connection.RepositoryScopeSHA256 != optimizePortfolioRepositoryScope(repositoryID) {
		return centralConnection{}, errors.New("central connection belongs to another repository")
	}
	return connection, nil
}

func resolveCentralConnectionDirectory(repositoryRoot, relative string, create bool) (string, string, error) {
	normalized, valid := normalizeOptimizeStateRelative(relative)
	if !valid {
		return "", "", errors.New("--connection-dir must be a clean repository-relative path under .buildopt")
	}
	path := filepath.Join(repositoryRoot, normalized)
	if create {
		if err := ensurePrivateDirectory(path, true); err != nil {
			return "", "", err
		}
	} else if _, err := os.Lstat(path); err == nil {
		if err := ensurePrivateDirectory(path, false); err != nil {
			return "", "", err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", "", err
	}
	return path, filepath.ToSlash(normalized), nil
}

func readCentralTokenFile(path string) ([]byte, *centralIssuedTokenDocument, error) {
	raw, err := readPrivateCentralCredential(path, centralMaximumConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("read central token: %w", err)
	}
	trimmed := bytes.TrimSuffix(bytes.TrimSuffix(raw, []byte("\n")), []byte("\r"))
	if len(trimmed) > 0 && trimmed[0] == '{' {
		var document centralIssuedTokenDocument
		if decodeCentralStrictJSON(trimmed, &document) != nil ||
			document.SchemaVersion != "buildopt.central/access-token/v1" ||
			!validOptimizeSHA(document.RepositoryScopeSHA256) || document.Repository == "" ||
			document.NamespaceGeneration < 1 {
			return nil, nil, errors.New("central token document is invalid")
		}
		if _, err := time.Parse(time.RFC3339Nano, document.ExpiresAt); err != nil {
			return nil, nil, errors.New("central token expiration is invalid")
		}
		return []byte(document.Token), &document, nil
	}
	return append([]byte(nil), trimmed...), nil, nil
}

func readCentralCAFile(path string) ([]byte, error) {
	if path == "" {
		return nil, nil
	}
	return readBoundedCentralFile(path, centralMaximumCA, false)
}

func readPrivateCentralCredential(path string, maximum int64) ([]byte, error) {
	return readBoundedCentralFile(path, maximum, true)
}

func readBoundedCentralFile(path string, maximum int64, private bool) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 ||
		info.Size() < 1 || info.Size() > maximum {
		return nil, errors.New("central input must be one bounded regular file without symlinks")
	}
	if private && runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("central credential and snapshot files must not be accessible by group or others")
	}
	return os.ReadFile(path)
}

func canonicalCentralServerURL(value string) (string, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" || parsed.RawPath != "" ||
		(parsed.Path != "" && parsed.Path != "/") {
		return "", errors.New("central server URL must be one canonical HTTPS origin")
	}
	canonical := "https://" + parsed.Host
	if value != canonical && value != canonical+"/" {
		return "", errors.New("central server URL must be one canonical HTTPS origin")
	}
	return canonical, nil
}

func centralGradleVersion(repositoryRoot string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(repositoryRoot, "gradle", "wrapper", "gradle-wrapper.properties"))
	if err != nil || len(raw) > gradleBootstrapMaximumPropertiesSize {
		return "", errors.New("Gradle Wrapper properties are unavailable")
	}
	text := string(raw)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if !strings.HasPrefix(line, "distributionUrl=") {
			continue
		}
		value := strings.ReplaceAll(strings.TrimPrefix(line, "distributionUrl="), `\:`, ":")
		archive := filepath.Base(value)
		matches := gradleDistributionNamePattern.FindStringSubmatch(archive)
		if len(matches) == 3 {
			return matches[1], nil
		}
	}
	return "", errors.New("Gradle Wrapper version is outside the tested matrix")
}

func centralStateRelativePath(stateRelative, path string) (string, bool) {
	prefix := strings.TrimSuffix(filepath.ToSlash(stateRelative), "/") + "/"
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	relative := strings.TrimPrefix(path, prefix)
	return relative, validCentralBundlePath(relative)
}

func validCentralBundlePath(path string) bool {
	native := filepath.FromSlash(path)
	return path != "" && path == filepath.ToSlash(native) && !filepath.IsAbs(native) &&
		filepath.Clean(native) == native && native != "." &&
		!strings.HasPrefix(native, ".."+string(filepath.Separator))
}

func centralStatePath(repositoryScope string, kind sharedcache.StateKind, resource, digest string) string {
	kindPath := map[sharedcache.StateKind]string{
		sharedcache.StateKindPortfolio:  "portfolios",
		sharedcache.StateKindEvidence:   "evidence",
		sharedcache.StateKindCheckpoint: "checkpoints",
	}[kind]
	path := "/api/v1/repositories/" + repositoryScope + "/state/" + kindPath + "/" + resource
	if digest != "" {
		path += "/" + digest
	}
	return path
}

func centralETag(value string) (string, bool) {
	if len(value) != 66 || value[0] != '"' || value[len(value)-1] != '"' {
		return "", false
	}
	digest := value[1 : len(value)-1]
	return digest, validOptimizeSHA(digest)
}

func centralStatus(err error) int {
	var status centralHTTPStatusError
	if errors.As(err, &status) {
		return status.status
	}
	return 0
}

func centralHasCapability(values []sharedcache.CentralCapability, wanted sharedcache.CentralCapability) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func decodeCentralStrictJSON(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON value")
	}
	return nil
}

func parseCentralTimestamp(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339Nano, value)
	return parsed
}

func writeCentralSyncResult(writer io.Writer, result centralSyncResult) error {
	raw, err := json.Marshal(result)
	if err != nil {
		return err
	}
	canonical, err := contractcrypto.CanonicalizeJCS(raw)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(writer, string(canonical))
	return err
}
