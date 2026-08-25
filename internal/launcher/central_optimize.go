package launcher

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
	"github.com/tonyredondo/buildopt/internal/profilediscovery"
	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

const (
	optimizeCentralSchemaVersion = "buildopt.poc/central-optimize-integration/v1"
	optimizeCentralDisconnected  = "DISCONNECTED"
	optimizeCentralAvailable     = "REMOTE_AVAILABLE"
	optimizeCentralSelected      = "REMOTE_PROFILE_SELECTED"
	optimizeCentralSelectedLocal = "LOCAL_REFRESHED_PROFILE_SELECTED"
	optimizeCentralRetained      = "NATIVE_RETAINED"
	optimizeCentralPublished     = "STATE_SYNCHRONIZED"

	optimizeCentralReasonNoConnection = "NO_CENTRAL_CONNECTION"
	optimizeCentralReasonNoProfile    = "NO_REMOTE_PROFILE"
	optimizeCentralReasonInvalid      = "REMOTE_STATE_INVALID"
	optimizeCentralReasonUnavailable  = "CENTRAL_SERVICE_UNAVAILABLE"
	optimizeCentralReasonStructural   = "REMOTE_PROFILE_STRUCTURAL_DRIFT"
	optimizeCentralReasonPlan         = "REMOTE_PROFILE_PLAN_REJECTED"
	optimizeCentralReasonSelected     = "REMOTE_QUALIFIED_PROFILE_SELECTED"

	optimizeDiscoveryRemoteRevalidated = "REMOTE_REVALIDATED"
	optimizeCalibrationRemoteQualified = "REMOTE_QUALIFIED"
	optimizeSelectionSourceLocal       = "LOCAL_PORTFOLIO"
	optimizeSelectionSourceCentral     = "CENTRAL_PORTFOLIO"
)

var optimizeCentralReplayBindings = func() []string {
	bindings := make([]string, 0, len(optimizeReplayBindingNames)+4)
	for _, binding := range optimizeReplayBindingNames {
		if binding != "REPOSITORY_REVISION" {
			bindings = append(bindings, binding)
		}
	}
	return append(bindings,
		"CENTRAL_EVIDENCE_REFERENCE",
		"EVIDENCE_REVISION_ANCESTRY",
		"OUTPUT_REVISION_ANCESTRY",
		"STRUCTURAL_PROFILE_FINGERPRINT",
		"STRUCTURAL_DIFF",
	)
}()

// optimizeCentralResult reports optional central-state work separately from
// the authoritative Gradle execution. Central failure is always fail-open and
// never turns an otherwise valid native build into a failed build.
type optimizeCentralResult struct {
	SchemaVersion           string `json:"schemaVersion"`
	Status                  string `json:"status"`
	Reason                  string `json:"reason"`
	Connected               bool   `json:"connected"`
	PreSyncOnline           bool   `json:"preSyncOnline"`
	PreSyncDurationMS       int64  `json:"preSyncDurationMs"`
	SnapshotsVerified       bool   `json:"snapshotsVerified"`
	PortfolioGeneration     int64  `json:"portfolioGeneration"`
	EvidenceGeneration      int64  `json:"evidenceGeneration"`
	CheckpointGeneration    int64  `json:"checkpointGeneration"`
	PortfolioManifestSHA256 string `json:"portfolioManifestSha256"`
	EvidenceManifestSHA256  string `json:"evidenceManifestSha256"`
	EvidenceRevision        string `json:"evidenceRevision"`
	RevalidatedRevision     string `json:"revalidatedRevision"`
	SelectionSource         string `json:"selectionSource"`
	GradleCacheMode         string `json:"gradleCacheMode"`
	GradleCacheStatus       string `json:"gradleCacheStatus"`
	PostSyncStatus          string `json:"postSyncStatus"`
	PostSyncOnline          bool   `json:"postSyncOnline"`
	PostSyncDurationMS      int64  `json:"postSyncDurationMs"`
	NativeFallback          bool   `json:"nativeFallback"`
	ProductionAuthorized    bool   `json:"productionAuthorized"`
	TestOptimization        string `json:"testOptimization"`
}

type centralOptimizeIntegration struct {
	invocation       optimizeInvocation
	connection       centralConnection
	client           *centralStateClient
	diagnostics      io.Writer
	startedAt        time.Time
	result           optimizeCentralResult
	prequalification optimizeEconomicPrequalification

	portfolio      *centralRemoteSnapshot
	evidence       *centralRemoteSnapshot
	checkpoint     *centralRemoteSnapshot
	localPortfolio *centralRemoteSnapshot
	localEvidence  *centralRemoteSnapshot
	refresh        *centralOptimizeRefresh
}

type centralOptimizeReplay struct {
	discovery   optimizeDiscoveryResult
	calibration optimizeCalibrationResult
	portfolio   optimizePortfolioResult
}

// centralOptimizeRefresh retains a fully verified economic profile whose
// revision-bound output pack cannot be used by the current descendant. A
// successful authoritative native build may refresh those bytes, but only
// after current discovery proves the same repository-independent structure.
type centralOptimizeRefresh struct {
	entry          optimizePortfolioEntry
	portfolioFiles map[string][]byte
	calibration    optimizeCalibrationResult
	graph          optimizeDiscoveryGraph
}

type centralOptimizeFailure struct {
	reason string
	err    error
}

func (failure centralOptimizeFailure) Error() string {
	return fmt.Sprintf("%s: %v", failure.reason, failure.err)
}

func disconnectedOptimizeCentralResult() optimizeCentralResult {
	return optimizeCentralResult{
		SchemaVersion: optimizeCentralSchemaVersion,
		Status:        optimizeCentralDisconnected, Reason: optimizeCentralReasonNoConnection,
		SelectionSource: "NONE", GradleCacheMode: "DISCONNECTED",
		GradleCacheStatus: "NOT_CONFIGURED", PostSyncStatus: "NOT_ATTEMPTED",
		ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
	}
}

func prepareCentralOptimizeIntegration(invocation optimizeInvocation, diagnostics io.Writer) *centralOptimizeIntegration {
	integration := &centralOptimizeIntegration{
		invocation: invocation, diagnostics: diagnostics, startedAt: time.Now(),
		result:           disconnectedOptimizeCentralResult(),
		prequalification: unevaluatedOptimizePrequalification(optimizePrequalificationReasonNoGraph),
	}
	if invocation.connectionDirectory == "" {
		return integration
	}
	if _, err := os.Lstat(filepath.Join(invocation.connectionDirectory, centralConnectionFile)); errors.Is(err, os.ErrNotExist) {
		return integration
	}
	connection, err := loadCentralConnection(invocation.repositoryRoot, invocation.connectionDirectory)
	if err != nil || connection.StateDirectory != invocation.stateRelative {
		integration.retain(optimizeCentralReasonInvalid)
		return integration
	}
	integration.connection = connection
	integration.result.Connected = true
	if connection.Cache != nil {
		integration.result.GradleCacheMode = connection.Cache.Mode
		integration.result.GradleCacheStatus = "AVAILABLE"
	}
	if !centralOptimizePreSyncEligible(invocation, connection) {
		integration.retain(optimizeCentralReasonStructural)
		return integration
	}
	local, localErr := collectCentralLocalPublications(
		invocation.repositoryRoot, invocation.stateDirectory, invocation.stateRelative,
		connection.RepositoryID, connection.RepositoryScopeSHA256,
	)
	integration.captureRefreshedLocalSnapshots(local, localErr)
	token, err := readPrivateCentralCredential(filepath.Join(invocation.connectionDirectory, connection.TokenFile), centralMaximumConfig)
	if err != nil {
		integration.retain(optimizeCentralReasonInvalid)
		return integration
	}
	defer clear(token)
	var ca []byte
	if connection.CAFile != "" {
		ca, err = readPrivateCentralCredential(filepath.Join(invocation.connectionDirectory, connection.CAFile), centralMaximumCA)
		if err != nil {
			integration.retain(optimizeCentralReasonInvalid)
			return integration
		}
	}
	client, err := newCentralStateClient(connection.ServerURL, string(token), ca)
	if err != nil {
		integration.retain(optimizeCentralReasonInvalid)
		return integration
	}
	integration.client = client

	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), centralRequestTimeout)
	syncResult, syncErr := synchronizeCentralStateWithLocal(
		ctx, "OPTIMIZE_PRE_SYNC", invocation.connectionDirectory,
		connection, client, local, localErr,
	)
	cancel()
	integration.result.PreSyncDurationMS = boundedDurationMS(time.Since(started))
	integration.result.PreSyncOnline = syncResult.Online
	if syncErr != nil && !syncResult.UsedOfflineSnapshot {
		integration.retain(optimizeCentralReasonUnavailable)
		_, _ = fmt.Fprintf(diagnostics, "buildopt: central optimize lookup unavailable; retaining native Gradle: %v\n", syncErr)
		return integration
	}

	integration.evidence, err = loadCentralSnapshot(
		invocation.connectionDirectory, connection.RepositoryScopeSHA256, sharedcache.StateKindEvidence,
	)
	if err != nil {
		integration.retain(optimizeCentralReasonNoProfile)
		return integration
	}
	integration.portfolio, err = loadCentralSnapshot(
		invocation.connectionDirectory, connection.RepositoryScopeSHA256, sharedcache.StateKindPortfolio,
	)
	if err != nil || !centralPortfolioReferencesEvidence(integration.portfolio, integration.evidence.manifestSHA256) {
		integration.retain(optimizeCentralReasonInvalid)
		return integration
	}
	integration.checkpoint, _ = loadCentralSnapshot(
		invocation.connectionDirectory, connection.RepositoryScopeSHA256, sharedcache.StateKindCheckpoint,
	)
	integration.result.Status = optimizeCentralAvailable
	integration.result.Reason = "VERIFIED_REMOTE_STATE_AVAILABLE"
	integration.result.SnapshotsVerified = true
	integration.result.PortfolioGeneration = integration.portfolio.head.Generation
	integration.result.EvidenceGeneration = integration.evidence.head.Generation
	if integration.checkpoint != nil {
		integration.result.CheckpointGeneration = integration.checkpoint.head.Generation
	}
	integration.result.PortfolioManifestSHA256 = integration.portfolio.manifestSHA256
	integration.result.EvidenceManifestSHA256 = integration.evidence.manifestSHA256
	return integration
}

func (integration *centralOptimizeIntegration) retain(reason string) {
	integration.result.Status = optimizeCentralRetained
	integration.result.Reason = reason
	integration.result.NativeFallback = true
}

func boundedDurationMS(duration time.Duration) int64 {
	value := duration.Milliseconds()
	if value < 1 {
		return 1
	}
	return value
}

func centralOptimizePreSyncEligible(invocation optimizeInvocation, connection centralConnection) bool {
	if !invocation.discovery.Ready || invocation.discovery.RepositoryID != connection.RepositoryID ||
		len(invocation.discovery.changedPaths) == 0 {
		return false
	}
	for _, changed := range invocation.discovery.changedPaths {
		if matchesAnyProposalGlob(optimizeGlobalChangePaths, changed) || centralOptimizeBuildLogicPath(changed) {
			return false
		}
	}
	return true
}

func (integration *centralOptimizeIntegration) prepareAutomaticReplay(run *optimizeRun) *impactInvocation {
	if integration == nil {
		return nil
	}
	if integration.localPortfolio != nil && integration.localEvidence != nil {
		remotePortfolio, remoteEvidence := integration.portfolio, integration.evidence
		integration.portfolio, integration.evidence = integration.localPortfolio, integration.localEvidence
		replay, impact, selection, err := integration.materializeReplay(run)
		integration.portfolio, integration.evidence = remotePortfolio, remoteEvidence
		if err == nil {
			selection.DurationNS = time.Since(integration.startedAt).Nanoseconds()
			if selection.DurationNS < 1 {
				selection.DurationNS = 1
			}
			selection.Source = optimizeSelectionSourceLocal
			selection.Reason = optimizeSelectionReasonSelected
			selection.ValidatedBindings = append([]string(nil), optimizeReplayBindingNames...)
			selection.RemotePortfolioManifestSHA256 = ""
			selection.RemoteEvidenceManifestSHA256 = ""
			integration.result.Status = optimizeCentralSelectedLocal
			integration.result.Reason = optimizeCentralSelectedLocal
			integration.result.SelectionSource = optimizeSelectionSourceLocal
			integration.result.EvidenceRevision = selection.EvidenceRevision
			integration.result.RevalidatedRevision = selection.RevalidatedRevision
			integration.result.NativeFallback = false
			run.centralReplay = replay
			run.selection = selection
			run.prequalification = unevaluatedOptimizePrequalification(optimizePrequalificationReasonSelected)
			return impact
		}
	}
	if integration.portfolio == nil || integration.evidence == nil {
		return nil
	}
	replay, impact, selection, err := integration.materializeReplay(run)
	selection.DurationNS = time.Since(integration.startedAt).Nanoseconds()
	if selection.DurationNS < 1 {
		selection.DurationNS = 1
	}
	if err != nil {
		reason := optimizeCentralReasonInvalid
		var failure centralOptimizeFailure
		if errors.As(err, &failure) {
			reason = failure.reason
		}
		integration.retain(reason)
		if integration.diagnostics != nil {
			_, _ = fmt.Fprintf(
				integration.diagnostics,
				"buildopt: central profile revalidation unavailable; retaining native Gradle: %v\n",
				err,
			)
		}
		selection = retainedOptimizeSelection("CENTRAL_PROFILE_REVALIDATION_FAILED", "CENTRAL_STATE")
		selection.DurationNS = time.Since(integration.startedAt).Nanoseconds()
		if selection.DurationNS < 1 {
			selection.DurationNS = 1
		}
		run.selection = selection
		run.prequalification = integration.prequalification
		return nil
	}
	integration.result.Status = optimizeCentralSelected
	integration.result.Reason = optimizeCentralReasonSelected
	integration.result.SelectionSource = optimizeSelectionSourceCentral
	integration.result.EvidenceRevision = selection.EvidenceRevision
	integration.result.RevalidatedRevision = selection.RevalidatedRevision
	integration.result.NativeFallback = false
	run.centralReplay = replay
	run.selection = selection
	run.prequalification = unevaluatedOptimizePrequalification(optimizePrequalificationReasonSelected)
	return impact
}

func (integration *centralOptimizeIntegration) captureRefreshedLocalSnapshots(
	publications map[sharedcache.StateKind]*centralLocalPublication,
	err error,
) {
	if err != nil {
		return
	}
	portfolio := publications[sharedcache.StateKindPortfolio]
	evidence := publications[sharedcache.StateKindEvidence]
	if portfolio == nil || evidence == nil {
		return
	}
	files, decodeErr := decodedCentralBundleFiles(portfolio.bundle)
	if decodeErr != nil {
		return
	}
	indexRaw := files[filepath.ToSlash(filepath.Join("portfolio", optimizePortfolioIndexFile))]
	var index optimizeProfilePortfolio
	if decodeCentralStrictJSON(indexRaw, &index) != nil || len(index.Profiles) < 1 {
		return
	}
	refreshed := false
	for _, entry := range index.Profiles {
		if validMeasurementRevision(entry.RevalidatedRevision) &&
			entry.RevalidatedRevision != entry.TargetRevision &&
			optimizePortfolioOutputRevision(entry) == entry.RevalidatedRevision {
			refreshed = true
			break
		}
	}
	if !refreshed {
		return
	}
	integration.localPortfolio = centralSnapshotFromLocalPublication(portfolio)
	integration.localEvidence = centralSnapshotFromLocalPublication(evidence)
}

func centralSnapshotFromLocalPublication(publication *centralLocalPublication) *centralRemoteSnapshot {
	objects := make(map[string][]byte, len(publication.objects))
	for _, object := range publication.objects {
		objects[object.sha256] = object.raw
	}
	return &centralRemoteSnapshot{
		manifestSHA256: publication.bundleSHA256,
		bundle:         publication.bundle,
		bundleRaw:      publication.bundleRaw,
		bundleSHA256:   publication.bundleSHA256,
		objects:        objects,
	}
}

func (integration *centralOptimizeIntegration) materializeReplay(
	run *optimizeRun,
) (*centralOptimizeReplay, *impactInvocation, optimizeSelectionResult, error) {
	invocation := run.invocation
	if !invocation.discovery.Ready {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{
			optimizeCentralReasonStructural,
			fmt.Errorf("current repository discovery is unavailable: %s", invocation.discovery.Reason),
		}
	}
	if integration.connection.RepositoryID != invocation.discovery.RepositoryID {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonStructural, errors.New("current repository identity differs from central state")}
	}
	portfolioFiles, err := decodedCentralBundleFiles(integration.portfolio.bundle)
	if err != nil {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonInvalid, err}
	}
	evidenceFiles, err := decodedCentralBundleFiles(integration.evidence.bundle)
	if err != nil {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonInvalid, err}
	}
	indexRaw, ok := portfolioFiles[filepath.ToSlash(filepath.Join("portfolio", optimizePortfolioIndexFile))]
	if !ok {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonInvalid, errors.New("remote portfolio index is missing")}
	}
	var portfolio optimizeProfilePortfolio
	if decodeCentralStrictJSON(indexRaw, &portfolio) != nil || portfolio.SchemaVersion != optimizePortfolioSchemaVersion ||
		portfolio.RepositoryScopeSHA256 != integration.connection.RepositoryScopeSHA256 ||
		portfolio.SelectionAuthorized || portfolio.ProductionAuthorized || len(portfolio.Profiles) < 1 ||
		len(portfolio.Profiles) > optimizePortfolioMaximumEntries {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonInvalid, errors.New("remote portfolio index is invalid")}
	}
	var lastCandidateError error
	for _, entry := range portfolio.Profiles {
		replay, impact, selection, candidateErr := integration.tryPortfolioEntry(
			invocation, portfolio, entry, portfolioFiles, evidenceFiles,
		)
		if candidateErr == nil {
			return replay, impact, selection, nil
		}
		lastCandidateError = candidateErr
	}
	if lastCandidateError != nil {
		return nil, nil, optimizeSelectionResult{}, lastCandidateError
	}
	return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonStructural, errors.New("no remotely qualified family matches the current structure")}
}

func (integration *centralOptimizeIntegration) tryPortfolioEntry(
	invocation optimizeInvocation,
	portfolio optimizeProfilePortfolio,
	entry optimizePortfolioEntry,
	portfolioFiles map[string][]byte,
	evidenceFiles map[string][]byte,
) (*centralOptimizeReplay, *impactInvocation, optimizeSelectionResult, error) {
	if entry.RepositoryID != invocation.discovery.RepositoryID {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonStructural, errors.New("remote repository identity drift")}
	}
	if entry.WrapperSHA256 != invocation.wrapperSHA256 {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonStructural, errors.New("remote Wrapper binding drift")}
	}
	if entry.ExecutableSHA256 != invocation.executableSHA256 {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonStructural, errors.New("remote executable binding drift")}
	}
	if !equalOptimizeStrings(entry.Entrypoints, invocation.discovery.Entrypoints) {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonStructural, errors.New("remote entrypoint binding drift")}
	}
	if !validMeasurementRevision(entry.TargetRevision) {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonStructural, errors.New("remote evidence revision is invalid")}
	}
	if entry.RevalidatedRevision != "" && !validMeasurementRevision(entry.RevalidatedRevision) {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonStructural, errors.New("remote revalidated revision is invalid")}
	}
	if _, err := gitOutput(invocation.repositoryRoot, "merge-base", "--is-ancestor", entry.TargetRevision, invocation.discovery.TargetRevision); err != nil {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonStructural, errors.New("remote evidence commit is not an ancestor")}
	}
	outputRevision := optimizePortfolioOutputRevision(entry)
	if _, err := gitOutput(invocation.repositoryRoot, "merge-base", "--is-ancestor", outputRevision, invocation.discovery.TargetRevision); err != nil {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonStructural, errors.New("remote output revision is not an ancestor")}
	}
	sinceOutput, err := centralOptimizeChangedPathsSinceEvidence(
		invocation.repositoryRoot,
		outputRevision,
		invocation.discovery.TargetRevision,
	)
	if err != nil {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonStructural, err}
	}
	refreshRequired := false
	refreshReason := ""
	for _, changed := range sinceOutput {
		if matchesAnyProposalGlob(optimizeGlobalChangePaths, changed) || centralOptimizeBuildLogicPath(changed) {
			refreshRequired = true
			refreshReason = "build logic changed after the remote output snapshot"
			break
		}
	}

	paths, profile, err := integration.materializeEntryFiles(
		invocation, entry, portfolioFiles, evidenceFiles,
	)
	if err != nil {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonInvalid, err}
	}
	manifest, err := buildimpact.LoadRepositoryManifest(
		invocation.repositoryRoot, filepath.FromSlash(paths.manifest), entry.RepositoryID, profile.PipelineClass,
	)
	if err != nil {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonInvalid, err}
	}
	graphRaw, err := os.ReadFile(filepath.Join(invocation.repositoryRoot, filepath.FromSlash(paths.graph)))
	if err != nil {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonInvalid, err}
	}
	graph, err := buildimpact.ParseDeclaredGraph(graphRaw, manifest)
	if err != nil {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonInvalid, err}
	}
	snapshot := centralOptimizeDiscoverySnapshot(graph.Graph)
	if entry.Materialization != nil && centralOptimizeMaterializedProducerChanged(
		snapshot, sinceOutput, entry.Materialization.MaterializedProjects,
	) {
		refreshRequired = true
		refreshReason = "a materialized producer changed after the remote output snapshot"
	}
	owners, err := buildimpact.ResolveProjectOwners(snapshot, invocation.discovery.changedPaths)
	if err != nil {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonStructural, err}
	}
	// Source and documentation commits between qualification and the current
	// event do not alter the structural binding. The current event is re-owned
	// and replanned below, while Wrapper and build-logic drift remain rejected
	// above. Requiring every historical path to belong to the stored graph made
	// an unrelated README or automation edit permanently strand a valid profile.
	family := optimizeChangeFamily(snapshot, invocation.discovery.changedPaths, owners)
	familySHA := optimizePortfolioFamilyDigest(
		entry.RepositoryID, family, owners, entry.Entrypoints,
		entry.CandidateEntrypoints, entry.RequiredOutputs, entry.CandidateOutputs,
	)
	if family != entry.Family || familySHA != entry.FamilySHA256 || !equalOptimizeStrings(owners, entry.ChangedProjects) {
		if integration.prequalification.Decision == optimizePrequalificationNotEvaluated {
			integration.prequalification = prequalifyOptimizeDiscovery(invocation, snapshot, owners, family)
		}
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonStructural, errors.New("current change family differs from remote qualification")}
	}
	if !optimizeStructuralBindingMatchesCurrent(entry, invocation, family, owners) {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{
			optimizeCentralReasonStructural,
			errors.New("current structure differs from the qualified profile fingerprint"),
		}
	}

	arguments := []string{
		"--config", filepath.FromSlash(paths.profile),
		"--changes-file", filepath.FromSlash(paths.changes),
	}
	expectedOptions, reason := optimizeCalibrationGradleOptions(invocation.discovery.gradleOptions)
	if reason != "" {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonPlan, errors.New(reason)}
	}
	for _, option := range expectedOptions {
		arguments = append(arguments, "--gradle-option="+option)
	}
	impact, err := prepareQualifiedPOCProfileInvocation(arguments, false)
	if err != nil {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonPlan, err}
	}
	if !impact.plan.CandidateSelected || impact.qualifiedProfile == nil ||
		!equalOptimizeStrings(impact.plan.Entrypoints, entry.CandidateEntrypoints) ||
		!equalOptimizeStrings(impact.qualifiedProfile.ExpectedOutputs, entry.RequiredOutputs) {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{
			optimizeCentralReasonPlan,
			fmt.Errorf("revalidated plan retained the full graph: %s", impact.plan.Reason),
		}
	}

	evidenceRaw := evidenceFiles[paths.bundleEvidence]
	if len(evidenceRaw) == 0 || !bytesEqual(evidenceRaw, portfolioFiles[paths.bundleEvidence]) {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonInvalid, errors.New("portfolio evidence is not the referenced evidence bundle")}
	}
	analysis, err := profilediscovery.AnalyzeOpportunity(profilediscovery.AnalysisOptions{
		RepositoryRoot: invocation.repositoryRoot,
		ManifestPath:   filepath.FromSlash(paths.manifest),
		GraphPath:      filepath.FromSlash(paths.graph),
		GeneratedPath:  filepath.FromSlash(paths.generated),
	})
	if err != nil {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonInvalid, err}
	}
	summary, err := profilediscovery.InspectStructuralMeasurementEvidence(evidenceRaw, analysis)
	if err != nil || !summary.Qualified || summary.ProductionAuthorized || !summary.FallbackSuccessful ||
		profile.Qualification == nil || profile.Qualification.RepositoryRevision != entry.TargetRevision ||
		profile.Qualification.SHA256 != entry.EvidenceSHA256 || profile.Qualification.Pairs != summary.Pairs ||
		profile.Qualification.MeanSavedMS != summary.MeanSavedMS || profile.Qualification.ReductionRatio != summary.ReductionRatio ||
		!sameFloatSlice(profile.Qualification.Interval95SavedMS, summary.Interval95SavedMS) {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonInvalid, errors.New("remote qualification evidence did not recompute")}
	}
	discoverySHA, err := optimizeGeneratedDocumentsSHA(invocation.repositoryRoot, paths.discoveryFiles)
	if err != nil {
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonInvalid, err}
	}
	calibration := optimizeCalibrationResult{
		Status: optimizeCalibrationRemoteQualified, Reason: optimizeCalibrationReasonQualified,
		Performed: false, Reused: true, PairsRequested: summary.Pairs, PairsMeasured: summary.Pairs,
		ControlMeanMS: summary.ControlMeanMS, CandidateMeanMS: summary.CandidateMeanMS,
		MeanSavedMS: summary.MeanSavedMS, ReductionRatio: summary.ReductionRatio,
		Interval95SavedMS: append([]float64(nil), summary.Interval95SavedMS...),
		PositivePairs:     summary.PositivePairs, ControlP95MS: summary.ControlP95MS,
		CandidateP95MS: summary.CandidateP95MS, MaximumBreakEvenBuilds: invocation.maxBreakEvenBuilds,
		QualificationPolicy: summary.QualificationPolicy,
		ValueGatePassed:     true, Qualified: true, FallbackSuccessful: true,
		EvidenceSHA256: entry.EvidenceSHA256, DiscoverySHA256: discoverySHA,
		GeneratedFiles: []string{paths.evidence}, ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
	}
	selectedProjects := len(graph.Graph.Projects) - len(impact.plan.OmittedProjects)
	if refreshRequired {
		integration.refresh = &centralOptimizeRefresh{
			entry: entry, portfolioFiles: portfolioFiles, calibration: calibration,
			graph: optimizeDiscoveryGraph{
				TotalProjects: len(graph.Graph.Projects), SelectedProjects: selectedProjects,
				OmittedProjects: len(impact.plan.OmittedProjects),
			},
		}
		return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{
			optimizeCentralReasonStructural,
			fmt.Errorf("%s; authoritative native refresh required", refreshReason),
		}
	}
	if entry.Materialization != nil {
		materialization, materializationErr := integration.materializeOutputPack(
			invocation, entry, portfolioFiles, filepath.ToSlash(filepath.Dir(paths.profile)),
		)
		if materializationErr != nil {
			return nil, nil, optimizeSelectionResult{}, centralOptimizeFailure{optimizeCentralReasonInvalid, materializationErr}
		}
		paths.materialization = materialization
	}
	candidateOutputs := append([]string(nil), entry.CandidateOutputs...)
	if len(candidateOutputs) == 0 {
		candidateOutputs = append([]string(nil), entry.RequiredOutputs...)
	}
	discovery := optimizeDiscoveryResult{
		Status: optimizeDiscoveryRemoteRevalidated, Reason: "REMOTE_STRUCTURAL_PROFILE_REVALIDATED",
		Source: invocation.discovery.Source, RepositoryID: invocation.discovery.RepositoryID,
		BaseRevision: invocation.discovery.BaseRevision, TargetRevision: invocation.discovery.TargetRevision,
		ChangeSHA256: invocation.discovery.ChangeSHA256, ChangedPathCount: invocation.discovery.ChangedPathCount,
		Entrypoints:          append([]string(nil), invocation.discovery.Entrypoints...),
		RequiredOutputs:      append([]string(nil), entry.RequiredOutputs...),
		CandidateOutputs:     candidateOutputs,
		CandidateEntrypoints: append([]string(nil), entry.CandidateEntrypoints...),
		StructuralBinding:    entry.StructuralBinding,
		Materialization:      paths.materialization,
		ChangeFamily:         family, ChangedProjects: append([]string(nil), owners...),
		Graph:          optimizeDiscoveryGraph{TotalProjects: len(graph.Graph.Projects), SelectedProjects: selectedProjects, OmittedProjects: len(impact.plan.OmittedProjects)},
		GeneratedFiles: append([]string(nil), paths.discoveryFiles...), ReviewRequired: true,
		ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
	}
	stagedEntry := centralOptimizeStagedEntry(entry, paths)
	portfolioResult := optimizePortfolioResultForEntry(
		optimizePortfolioReasonReused, false, true, paths.portfolio,
		len(portfolio.Profiles), stagedEntry,
	)
	selection := optimizeSelectionResult{
		Status: optimizeSelectionSelected, Reason: optimizeCentralReasonSelected,
		Performed: true, Selected: true, CompletedBeforeGradle: true,
		ChangeFamily: family, FamilySHA256: entry.FamilySHA256,
		ProfileSHA256: stagedEntry.ProfileSHA256, ProfileFile: paths.profile,
		OriginalEntrypoints: append([]string(nil), entry.Entrypoints...),
		SelectedEntrypoints: append([]string(nil), entry.CandidateEntrypoints...),
		ValidatedBindings:   append([]string(nil), optimizeCentralReplayBindings...), FailedBindings: []string{},
		Source: optimizeSelectionSourceCentral, EvidenceRevision: entry.TargetRevision,
		RevalidatedRevision:           invocation.discovery.TargetRevision,
		RemotePortfolioManifestSHA256: integration.portfolio.manifestSHA256,
		RemoteEvidenceManifestSHA256:  integration.evidence.manifestSHA256,
		ProductionAuthorized:          false, TestOptimization: "OUT_OF_SCOPE",
	}
	return &centralOptimizeReplay{discovery: discovery, calibration: calibration, portfolio: portfolioResult}, &impact, selection, nil
}

func (integration *centralOptimizeIntegration) refreshQualifiedProfile(
	run *optimizeRun,
	discovery optimizeDiscoveryResult,
) (optimizeCalibrationResult, optimizePortfolioResult, bool) {
	refresh := integration.refresh
	if refresh == nil || discovery.Status != optimizeDiscoveryComplete ||
		discovery.RepositoryID != refresh.entry.RepositoryID ||
		discovery.TargetRevision != run.invocation.discovery.TargetRevision ||
		discovery.ChangeFamily != refresh.entry.Family ||
		discovery.Graph != refresh.graph ||
		!equalOptimizeStrings(discovery.ChangedProjects, refresh.entry.ChangedProjects) ||
		!equalOptimizeStrings(discovery.Entrypoints, refresh.entry.Entrypoints) ||
		!equalOptimizeStrings(discovery.CandidateEntrypoints, refresh.entry.CandidateEntrypoints) ||
		!equalOptimizeStrings(discovery.RequiredOutputs, refresh.entry.RequiredOutputs) ||
		!equalOptimizeStrings(discovery.CandidateOutputs, refresh.entry.CandidateOutputs) ||
		optimizePortfolioFamilySHA(discovery) != refresh.entry.FamilySHA256 {
		return optimizeCalibrationResult{}, optimizePortfolioResult{}, false
	}
	if refresh.entry.Materialization != nil {
		if discovery.AggregatePartition == nil ||
			!equalOptimizeStrings(
				discovery.AggregatePartition.MaterializedProjects,
				refresh.entry.Materialization.MaterializedProjects,
			) || discovery.Materialization.Status != optimizeMaterializationCaptured {
			return optimizeCalibrationResult{}, optimizePortfolioResult{}, false
		}
	}

	entry := refresh.entry
	if entry.Materialization != nil {
		materialization, err := prepareOptimizePortfolioMaterialization(run.invocation, discovery)
		if err != nil {
			return optimizeCalibrationResult{}, optimizePortfolioResult{}, false
		}
		entry.Materialization = materialization
	}
	bundleProfile, ok := centralStateRelativePath(run.invocation.stateRelative, entry.ProfilePath)
	if !ok {
		return optimizeCalibrationResult{}, optimizePortfolioResult{}, false
	}
	entry.RevalidatedRevision = discovery.TargetRevision
	bundleDirectory := filepath.ToSlash(filepath.Dir(bundleProfile))
	destinationDirectory := filepath.ToSlash(filepath.Dir(entry.ProfilePath))
	destinationPaths := []string{
		filepath.ToSlash(filepath.Join(destinationDirectory, "manifest.json")),
		filepath.ToSlash(filepath.Join(destinationDirectory, "graph.json")),
		filepath.ToSlash(filepath.Join(destinationDirectory, "generated-manifest.json")),
	}
	profileRaw := refresh.portfolioFiles[filepath.ToSlash(filepath.Join(bundleDirectory, "profile.json"))]
	var profile qualifiedPOCProfile
	if decodeCentralStrictJSON(profileRaw, &profile) != nil || validateQualifiedPOCProfile(profile) != nil ||
		profile.RepositoryID != entry.RepositoryID || profile.Qualification == nil ||
		profile.Qualification.SHA256 != entry.EvidenceSHA256 || len(profile.Preconditions) != 3 {
		return optimizeCalibrationResult{}, optimizePortfolioResult{}, false
	}
	profile.Impact.Manifest = destinationPaths[0]
	profile.Impact.Graph = destinationPaths[1]
	profile.Impact.GeneratedManifest = destinationPaths[2]
	for index, path := range destinationPaths {
		profile.Preconditions[index].Path = path
	}
	profileRaw, err := json.Marshal(profile)
	if err != nil {
		return optimizeCalibrationResult{}, optimizePortfolioResult{}, false
	}
	profileDigest := sha256.Sum256(profileRaw)
	entry.ProfileSHA256 = hex.EncodeToString(profileDigest[:])
	profileDestination := filepath.Join(run.invocation.repositoryRoot, filepath.FromSlash(entry.ProfilePath))
	if err := os.MkdirAll(filepath.Dir(profileDestination), 0o700); err != nil ||
		writePrivateAtomicFile(profileDestination, profileRaw) != nil {
		return optimizeCalibrationResult{}, optimizePortfolioResult{}, false
	}
	for _, name := range []string{"manifest.json", "graph.json", "generated-manifest.json", "evidence.json"} {
		source := filepath.ToSlash(filepath.Join(bundleDirectory, name))
		raw := refresh.portfolioFiles[source]
		if len(raw) == 0 {
			return optimizeCalibrationResult{}, optimizePortfolioResult{}, false
		}
		destination := filepath.Join(
			run.invocation.repositoryRoot,
			filepath.FromSlash(filepath.ToSlash(filepath.Join(destinationDirectory, name))),
		)
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil ||
			writePrivateAtomicFile(destination, raw) != nil {
			return optimizeCalibrationResult{}, optimizePortfolioResult{}, false
		}
	}

	indexPath := filepath.ToSlash(filepath.Join(run.invocation.stateRelative, "portfolio", optimizePortfolioIndexFile))
	portfolio := optimizeProfilePortfolio{
		SchemaVersion: optimizePortfolioSchemaVersion,
		Generation:    run.state.Generation, RepositoryScopeSHA256: optimizePortfolioRepositoryScope(entry.RepositoryID),
		Profiles: []optimizePortfolioEntry{entry}, UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
		SelectionAuthorized: false, ProductionAuthorized: false,
	}
	indexAbsolute := filepath.Join(run.invocation.repositoryRoot, filepath.FromSlash(indexPath))
	if err := writeCanonicalPrivateJSON(indexAbsolute, portfolio); err != nil {
		return optimizeCalibrationResult{}, optimizePortfolioResult{}, false
	}
	if _, valid := loadOptimizePortfolio(
		run.invocation.repositoryRoot,
		indexPath,
		optimizePortfolioRepositoryScope(entry.RepositoryID),
	); !valid {
		return optimizeCalibrationResult{}, optimizePortfolioResult{}, false
	}

	calibration := refresh.calibration
	calibration.GeneratedFiles = []string{filepath.ToSlash(filepath.Join(destinationDirectory, "evidence.json"))}
	result := optimizePortfolioResultForEntry(
		optimizePortfolioReasonRefreshed,
		true,
		false,
		indexPath,
		1,
		entry,
	)
	integration.result.Reason = "REMOTE_QUALIFIED_PROFILE_OUTPUTS_REFRESHED"
	integration.result.EvidenceRevision = entry.TargetRevision
	integration.result.RevalidatedRevision = discovery.TargetRevision
	integration.result.NativeFallback = true
	if integration.diagnostics != nil {
		_, _ = fmt.Fprintf(
			integration.diagnostics,
			"buildopt: remote profile structure remained compatible; refreshed exact outputs at %s after authoritative native Gradle\n",
			discovery.TargetRevision,
		)
	}
	return calibration, result, true
}
func centralOptimizeChangedPathsSinceEvidence(repositoryRoot, evidenceRevision, currentRevision string) ([]string, error) {
	if evidenceRevision == currentRevision {
		return nil, nil
	}
	return proposalGitChangedPaths(repositoryRoot, evidenceRevision, currentRevision)
}

func centralOptimizeMaterializedProducerChanged(
	snapshot buildimpact.DiscoverySnapshot,
	changedPaths []string,
	materializedProjects []string,
) bool {
	materialized := make(map[string]bool, len(materializedProjects))
	for _, project := range materializedProjects {
		materialized[project] = true
	}
	for _, changed := range changedPaths {
		owners, err := buildimpact.ResolveProjectOwners(snapshot, []string{changed})
		if err != nil {
			continue
		}
		for _, owner := range owners {
			if materialized[owner] {
				return true
			}
		}
	}
	return false
}

type centralOptimizePaths struct {
	portfolio, profile, manifest, graph, generated, evidence, changes string
	bundleEvidence                                                    string
	profileSHA                                                        string
	discoveryFiles                                                    []string
	materialization                                                   optimizeOutputMaterialization
}

func (integration *centralOptimizeIntegration) materializeEntryFiles(
	invocation optimizeInvocation,
	entry optimizePortfolioEntry,
	portfolioFiles map[string][]byte,
	evidenceFiles map[string][]byte,
) (centralOptimizePaths, *qualifiedPOCProfile, error) {
	bundleProfile, ok := centralStateRelativePath(invocation.stateRelative, entry.ProfilePath)
	if !ok {
		return centralOptimizePaths{}, nil, errors.New("remote profile path escapes optimize state")
	}
	bundleDirectory := filepath.ToSlash(filepath.Dir(bundleProfile))
	bundlePaths := map[string]string{
		"profile":   bundleProfile,
		"manifest":  filepath.ToSlash(filepath.Join(bundleDirectory, "manifest.json")),
		"graph":     filepath.ToSlash(filepath.Join(bundleDirectory, "graph.json")),
		"generated": filepath.ToSlash(filepath.Join(bundleDirectory, "generated-manifest.json")),
		"evidence":  filepath.ToSlash(filepath.Join(bundleDirectory, "evidence.json")),
	}
	for name, path := range bundlePaths {
		raw := portfolioFiles[path]
		if len(raw) == 0 {
			return centralOptimizePaths{}, nil, fmt.Errorf("remote %s artifact is missing", name)
		}
	}
	bindings := map[string]string{
		"profile": entry.ProfileSHA256, "manifest": entry.ManifestSHA256,
		"graph": entry.GraphSHA256, "generated": entry.GeneratedSHA256, "evidence": entry.EvidenceSHA256,
	}
	for name, expected := range bindings {
		digest := sha256.Sum256(portfolioFiles[bundlePaths[name]])
		if hex.EncodeToString(digest[:]) != expected {
			return centralOptimizePaths{}, nil, fmt.Errorf("remote %s binding drift", name)
		}
	}
	if evidenceRaw := evidenceFiles[bundlePaths["evidence"]]; !bytesEqual(evidenceRaw, portfolioFiles[bundlePaths["evidence"]]) {
		return centralOptimizePaths{}, nil, errors.New("remote evidence bundle does not match the portfolio")
	}
	var profile qualifiedPOCProfile
	if decodeCentralStrictJSON(portfolioFiles[bundlePaths["profile"]], &profile) != nil || validateQualifiedPOCProfile(profile) != nil ||
		profile.RepositoryID != entry.RepositoryID || profile.Qualification == nil || len(profile.Preconditions) != 3 ||
		profile.Qualification.SHA256 != entry.EvidenceSHA256 {
		return centralOptimizePaths{}, nil, errors.New("remote qualified profile is invalid")
	}

	// The materialized profile embeds target-specific changed paths and
	// revalidation metadata. Bind the complete portfolio, family, and target
	// revision into one directory digest: separate commits cannot replace one
	// another's files, while the path remains usable on Windows.
	materializedBinding := optimizeDigest(
		"buildopt-central-materialization-v1",
		integration.portfolio.manifestSHA256,
		entry.FamilySHA256,
		invocation.discovery.TargetRevision,
	)
	materializedRoot := filepath.ToSlash(filepath.Join(
		invocation.connectionRelative, "materialized",
		materializedBinding,
	))
	paths := centralOptimizePaths{
		portfolio:      filepath.ToSlash(filepath.Join(materializedRoot, optimizePortfolioIndexFile)),
		profile:        filepath.ToSlash(filepath.Join(materializedRoot, "profile.json")),
		manifest:       filepath.ToSlash(filepath.Join(materializedRoot, "manifest.json")),
		graph:          filepath.ToSlash(filepath.Join(materializedRoot, "graph.json")),
		generated:      filepath.ToSlash(filepath.Join(materializedRoot, "generated-manifest.json")),
		evidence:       filepath.ToSlash(filepath.Join(materializedRoot, "evidence.json")),
		changes:        filepath.ToSlash(filepath.Join(materializedRoot, "changes.txt")),
		bundleEvidence: bundlePaths["evidence"],
		materialization: optimizeOutputMaterialization{
			Status: optimizeMaterializationNotRequired, Reason: optimizeMaterializationReasonNone,
			Patterns: []string{},
		},
	}
	profile.Impact.Manifest = paths.manifest
	profile.Impact.Graph = paths.graph
	profile.Impact.GeneratedManifest = paths.generated
	for index, path := range []string{paths.manifest, paths.graph, paths.generated} {
		profile.Preconditions[index].Path = path
	}
	profileRaw, err := json.Marshal(profile)
	if err != nil {
		return centralOptimizePaths{}, nil, err
	}
	profileDigest := sha256.Sum256(profileRaw)
	paths.profileSHA = hex.EncodeToString(profileDigest[:])
	var stagedPortfolio optimizeProfilePortfolio
	indexBundlePath := filepath.ToSlash(filepath.Join("portfolio", optimizePortfolioIndexFile))
	if decodeCentralStrictJSON(portfolioFiles[indexBundlePath], &stagedPortfolio) != nil {
		return centralOptimizePaths{}, nil, errors.New("remote portfolio index cannot be staged")
	}
	for index := range stagedPortfolio.Profiles {
		if stagedPortfolio.Profiles[index].FamilySHA256 == entry.FamilySHA256 {
			stagedPortfolio.Profiles[index].ProfilePath = paths.profile
			stagedPortfolio.Profiles[index].ProfileSHA256 = paths.profileSHA
		}
	}
	portfolioRaw, err := json.Marshal(stagedPortfolio)
	if err != nil {
		return centralOptimizePaths{}, nil, err
	}
	files := map[string][]byte{
		paths.portfolio: portfolioRaw,
		paths.profile:   profileRaw, paths.manifest: portfolioFiles[bundlePaths["manifest"]],
		paths.graph: portfolioFiles[bundlePaths["graph"]], paths.generated: portfolioFiles[bundlePaths["generated"]],
		paths.evidence: portfolioFiles[bundlePaths["evidence"]],
		paths.changes:  []byte(strings.Join(invocation.discovery.changedPaths, "\n") + "\n"),
	}
	fallback := filepath.ToSlash(filepath.Join(materializedRoot, "fallback-changes.txt"))
	revalidation := filepath.ToSlash(filepath.Join(materializedRoot, "revalidation.json"))
	source := filepath.ToSlash(filepath.Join(materializedRoot, "source.json"))
	files[fallback] = files[paths.changes]
	files[revalidation] = centralOptimizeMetadataJSON(
		"buildopt.poc/central-optimize-revalidation/v1", entry.TargetRevision, invocation.discovery.TargetRevision,
	)
	files[source] = centralOptimizeMetadataJSON(
		"buildopt.poc/central-optimize-source/v1", integration.portfolio.manifestSHA256, integration.evidence.manifestSHA256,
	)
	for relative, raw := range files {
		absolute := filepath.Join(invocation.repositoryRoot, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(absolute), 0o700); err != nil || writePrivateAtomicFile(absolute, raw) != nil {
			return centralOptimizePaths{}, nil, errors.New("materialize verified central profile")
		}
	}
	paths.discoveryFiles = []string{paths.manifest, paths.graph, paths.generated, paths.changes, fallback, revalidation, source}
	sort.Strings(paths.discoveryFiles)
	return paths, &profile, nil
}

func (integration *centralOptimizeIntegration) materializeOutputPack(
	invocation optimizeInvocation,
	entry optimizePortfolioEntry,
	portfolioFiles map[string][]byte,
	materializedRoot string,
) (optimizeOutputMaterialization, error) {
	started := time.Now()
	metadata := entry.Materialization
	if metadata == nil || integration.portfolio == nil || !validOptimizePortfolioMaterialization(entry) {
		return optimizeOutputMaterialization{}, errors.New("remote output materialization metadata is invalid")
	}
	bundleManifest := filepath.ToSlash(filepath.Join(
		"portfolio", "profiles", entry.FamilySHA256, "materialization-manifest.json",
	))
	manifestRaw := portfolioFiles[bundleManifest]
	manifestDigest := sha256.Sum256(manifestRaw)
	if len(manifestRaw) == 0 || hex.EncodeToString(manifestDigest[:]) != metadata.ManifestSHA256 {
		return optimizeOutputMaterialization{}, errors.New("remote output materialization manifest is missing or corrupt")
	}
	var manifest optimizeOutputMaterializationManifest
	if decodeCentralStrictJSON(manifestRaw, &manifest) != nil ||
		manifest.SchemaVersion != optimizeMaterializationSchema ||
		manifest.RepositoryID != entry.RepositoryID || manifest.TargetRevision != optimizePortfolioOutputRevision(entry) ||
		!equalOptimizeStrings(manifest.RequiredOutputs, entry.RequiredOutputs) ||
		!equalOptimizeStrings(manifest.CandidateOutputs, entry.CandidateOutputs) ||
		manifest.PackSHA256 != metadata.PackSHA256 || manifest.PackSize != metadata.PackSize ||
		len(manifest.Entries) < 1 || len(manifest.Entries) > optimizeMaterializationMaxFiles {
		return optimizeOutputMaterialization{}, errors.New("remote output materialization manifest binding drifted")
	}
	var byteCount int64
	previous := ""
	for _, materialized := range manifest.Entries {
		if materialized.Path <= previous || !validObservedOutputPath(materialized.Path) ||
			!validOptimizeSHA(materialized.SHA256) || materialized.Size < 0 ||
			materialized.Mode > 0o777 || materialized.Offset != byteCount {
			return optimizeOutputMaterialization{}, errors.New("remote output materialization entry is invalid")
		}
		previous = materialized.Path
		byteCount += materialized.Size
	}
	if byteCount != metadata.PackSize {
		return optimizeOutputMaterialization{}, errors.New("remote output materialization entry size drifted")
	}
	packRelative := filepath.ToSlash(filepath.Join(materializedRoot, optimizeMaterializationPackName))
	packAbsolute := filepath.Join(invocation.repositoryRoot, filepath.FromSlash(packRelative))
	if err := os.MkdirAll(filepath.Dir(packAbsolute), 0o700); err != nil {
		return optimizeOutputMaterialization{}, errors.New("create remote output materialization directory")
	}
	pack, err := os.CreateTemp(filepath.Dir(packAbsolute), ".buildopt-remote-pack-*")
	if err != nil {
		return optimizeOutputMaterialization{}, errors.New("create remote output materialization pack")
	}
	temporary := pack.Name()
	defer os.Remove(temporary)
	if pack.Chmod(0o600) != nil {
		_ = pack.Close()
		return optimizeOutputMaterialization{}, errors.New("protect remote output materialization pack")
	}
	packDigest := sha256.New()
	var written int64
	for _, digest := range metadata.ChunkSHA256 {
		raw := integration.portfolio.objects[digest]
		chunkDigest := sha256.Sum256(raw)
		if len(raw) < 1 || len(raw) > optimizeMaterializationChunkBytes ||
			hex.EncodeToString(chunkDigest[:]) != digest {
			_ = pack.Close()
			return optimizeOutputMaterialization{}, errors.New("remote output materialization chunk is unavailable")
		}
		count, writeErr := io.Copy(io.MultiWriter(pack, packDigest), bytes.NewReader(raw))
		if writeErr != nil || count != int64(len(raw)) {
			_ = pack.Close()
			return optimizeOutputMaterialization{}, errors.New("write remote output materialization chunk")
		}
		written += count
	}
	if written != metadata.PackSize || hex.EncodeToString(packDigest.Sum(nil)) != metadata.PackSHA256 {
		_ = pack.Close()
		return optimizeOutputMaterialization{}, errors.New("remote output materialization pack failed final verification")
	}
	syncErr := pack.Sync()
	closeErr := pack.Close()
	if syncErr != nil || closeErr != nil || replaceManagedFile(temporary, packAbsolute) != nil ||
		syncManagedDirectory(filepath.Dir(packAbsolute)) != nil {
		return optimizeOutputMaterialization{}, errors.New("remote output materialization pack failed final verification")
	}
	manifest.TargetRevision = invocation.discovery.TargetRevision
	manifest.PackFile = packRelative
	manifestRelative := filepath.ToSlash(filepath.Join(materializedRoot, "materialization-manifest.json"))
	manifestAbsolute := filepath.Join(invocation.repositoryRoot, filepath.FromSlash(manifestRelative))
	if err := writeCanonicalPrivateJSON(manifestAbsolute, manifest); err != nil {
		return optimizeOutputMaterialization{}, errors.New("write remote output materialization manifest")
	}
	revalidatedRaw, err := os.ReadFile(manifestAbsolute)
	if err != nil {
		return optimizeOutputMaterialization{}, errors.New("read remote output materialization manifest")
	}
	revalidatedDigest := sha256.Sum256(revalidatedRaw)
	totalMS := boundedDurationMS(time.Since(started))
	return optimizeOutputMaterialization{
		Status: optimizeMaterializationCaptured, Reason: optimizeMaterializationReasonReady,
		Patterns:     optimizeMaterializationPatterns(entry.RequiredOutputs, entry.CandidateOutputs),
		ManifestFile: manifestRelative, ManifestSHA256: hex.EncodeToString(revalidatedDigest[:]),
		FileCount: len(manifest.Entries), ByteCount: metadata.PackSize,
		Economics: optimizeMaterializationEconomics{PackMS: totalMS, TotalMS: totalMS},
	}, nil
}

func centralOptimizeMetadataJSON(schema, left, right string) []byte {
	raw, _ := json.Marshal(struct {
		SchemaVersion string `json:"schemaVersion"`
		Left          string `json:"left"`
		Right         string `json:"right"`
	}{SchemaVersion: schema, Left: left, Right: right})
	return raw
}

func centralOptimizeDiscoverySnapshot(graph buildimpact.DeclaredGraph) buildimpact.DiscoverySnapshot {
	snapshot := buildimpact.DiscoverySnapshot{
		SchemaVersion: buildimpact.DiscoverySchemaVersion, Complete: graph.Complete,
		Projects: []buildimpact.DiscoveredProject{}, Entrypoints: []buildimpact.DiscoveredEntrypoint{},
	}
	for _, project := range graph.Projects {
		snapshot.Projects = append(snapshot.Projects, buildimpact.DiscoveredProject{
			Path: project.Path, SourcePaths: append([]string(nil), project.SourcePaths...),
			OwnedSourcePaths:     append([]string(nil), project.OwnedSourcePaths...),
			DependsOn:            append([]string(nil), project.DependsOn...),
			UnknownRelationships: project.UnknownRelationships,
		})
	}
	for _, entrypoint := range graph.Entrypoints {
		snapshot.Entrypoints = append(snapshot.Entrypoints, buildimpact.DiscoveredEntrypoint{
			Name: entrypoint.Name, ReachesProjects: append([]string(nil), entrypoint.ReachesProjects...),
			ContainsTestTasks: entrypoint.ContainsTestTasks, UnknownRelationships: entrypoint.UnknownRelationships,
		})
	}
	return snapshot
}

func centralOptimizeStagedEntry(entry optimizePortfolioEntry, paths centralOptimizePaths) optimizePortfolioEntry {
	entry.ProfilePath = paths.profile
	entry.ProfileSHA256 = paths.profileSHA
	return entry
}

func centralOptimizeBuildLogicPath(path string) bool {
	normalized := strings.ToLower(filepath.ToSlash(path))
	return strings.HasSuffix(normalized, ".gradle") || strings.HasSuffix(normalized, ".gradle.kts") ||
		strings.HasSuffix(normalized, "/gradle.properties") || normalized == "gradle.properties" ||
		strings.HasSuffix(normalized, "/libs.versions.toml") || normalized == "libs.versions.toml"
}

func decodedCentralBundleFiles(bundle centralStateBundle) (map[string][]byte, error) {
	files := make(map[string][]byte, len(bundle.Files))
	for _, file := range bundle.Files {
		raw, err := base64.RawStdEncoding.DecodeString(file.ContentBase64)
		if err != nil || int64(len(raw)) != file.SizeBytes {
			return nil, errors.New("decode central bundle file")
		}
		digest := sha256.Sum256(raw)
		if hex.EncodeToString(digest[:]) != file.SHA256 {
			return nil, errors.New("central bundle file digest mismatch")
		}
		files[file.Path] = raw
	}
	return files, nil
}

func bytesEqual(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func (integration *centralOptimizeIntegration) publish(run *optimizeRun, diagnostics io.Writer) {
	if integration.client == nil || !integration.result.Connected {
		return
	}
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), centralRequestTimeout)
	result, err := synchronizeCentralState(
		ctx, "OPTIMIZE_POST_SYNC", run.invocation.repositoryRoot,
		run.invocation.stateDirectory, run.invocation.connectionDirectory,
		integration.connection, integration.client,
	)
	cancel()
	integration.result.PostSyncDurationMS = boundedDurationMS(time.Since(started))
	integration.result.PostSyncOnline = result.Online
	if err != nil && !result.UsedOfflineSnapshot {
		integration.result.PostSyncStatus = "UNAVAILABLE"
		_, _ = fmt.Fprintf(diagnostics, "buildopt: central optimize publication unavailable; local result remains authoritative: %v\n", err)
		return
	}
	integration.result.PostSyncStatus = optimizeCentralPublished
}
