package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/tonyredondo/buildopt/internal/profilediscovery"
)

const (
	optimizeBindingPortfolio              = "PROFILE_PORTFOLIO_COMPLETE"
	optimizePortfolioSchemaVersion        = "buildopt.poc/optimize-profile-portfolio/v1"
	optimizePortfolioComplete             = "COMPLETE"
	optimizePortfolioSkipped              = "SKIPPED"
	optimizePortfolioRetained             = "NATIVE_RETAINED"
	optimizePortfolioReasonStored         = "QUALIFIED_PROFILE_STORED"
	optimizePortfolioReasonReused         = "QUALIFIED_PROFILE_REUSED"
	optimizePortfolioIndexFile            = "portfolio.json"
	optimizePortfolioMaximumEntries       = 64
	optimizePortfolioMaximumIndexBytes    = 1 << 20
	optimizePortfolioMaximumArtifactBytes = 16 << 20
)

type optimizePortfolioResult struct {
	Status               string   `json:"status"`
	Reason               string   `json:"reason"`
	Performed            bool     `json:"performed"`
	Reused               bool     `json:"reused"`
	ChangeFamily         string   `json:"changeFamily"`
	FamilySHA256         string   `json:"familySha256"`
	ProfileSHA256        string   `json:"profileSha256"`
	ProfileCount         int      `json:"profileCount"`
	PortfolioFile        string   `json:"portfolioFile"`
	ProfileFile          string   `json:"profileFile"`
	GeneratedFiles       []string `json:"generatedFiles"`
	SelectionPerformed   bool     `json:"selectionPerformed"`
	ProductionAuthorized bool     `json:"productionAuthorized"`
	TestOptimization     string   `json:"testOptimization"`
}

type optimizeProfilePortfolio struct {
	SchemaVersion         string                   `json:"schemaVersion"`
	Generation            int                      `json:"generation"`
	RepositoryScopeSHA256 string                   `json:"repositoryScopeSha256"`
	Profiles              []optimizePortfolioEntry `json:"profiles"`
	UpdatedAt             string                   `json:"updatedAt"`
	SelectionAuthorized   bool                     `json:"selectionAuthorized"`
	ProductionAuthorized  bool                     `json:"productionAuthorized"`
}

type optimizePortfolioEntry struct {
	Family               string                            `json:"family"`
	FamilySHA256         string                            `json:"familySha256"`
	ChangedProjects      []string                          `json:"changedProjects"`
	RepositoryID         string                            `json:"repositoryId"`
	Entrypoints          []string                          `json:"entrypoints"`
	CandidateEntrypoints []string                          `json:"candidateEntrypoints"`
	RequiredOutputs      []string                          `json:"requiredOutputs"`
	CandidateOutputs     []string                          `json:"candidateOutputs,omitempty"`
	Materialization      *optimizePortfolioMaterialization `json:"materialization,omitempty"`
	TargetRevision       string                            `json:"targetRevision"`
	WrapperSHA256        string                            `json:"wrapperSha256"`
	ExecutableSHA256     string                            `json:"executableSha256"`
	ManifestSHA256       string                            `json:"manifestSha256"`
	GraphSHA256          string                            `json:"graphSha256"`
	GeneratedSHA256      string                            `json:"generatedManifestSha256"`
	EvidenceSHA256       string                            `json:"evidenceSha256"`
	ProfileSHA256        string                            `json:"profileSha256"`
	ProfilePath          string                            `json:"profilePath"`
	State                string                            `json:"state"`
	SelectionAuthorized  bool                              `json:"selectionAuthorized"`
	ProductionAuthorized bool                              `json:"productionAuthorized"`
}

// optimizePortfolioMaterialization binds the verified output pack needed when
// a selected profile omits producers whose outputs still belong to the owner
// workflow. Chunks are transport identities, not Gradle cache keys.
type optimizePortfolioMaterialization struct {
	ManifestSHA256       string   `json:"manifestSha256"`
	PackSHA256           string   `json:"packSha256"`
	PackSize             int64    `json:"packSize"`
	ChunkSHA256          []string `json:"chunkSha256"`
	MaterializedProjects []string `json:"materializedProjects"`
}

type optimizePortfolioArtifacts struct {
	entry optimizePortfolioEntry
	files map[string][]byte
}

func emptyOptimizePortfolio(reason string) optimizePortfolioResult {
	return optimizePortfolioResult{
		Status: optimizePortfolioSkipped, Reason: reason,
		GeneratedFiles: []string{}, SelectionPerformed: false,
		ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
	}
}

func validOptimizePortfolioCheckpoint(state optimizeState) bool {
	portfolio := state.Portfolio
	if portfolio.ProductionAuthorized || portfolio.SelectionPerformed {
		return false
	}
	switch portfolio.Status {
	case "":
		return state.Phase == optimizePhaseUnseen && !state.BuildStarted &&
			portfolio.Reason == "" && portfolio.TestOptimization == ""
	case optimizePortfolioSkipped:
		return portfolio.Reason != "" && !portfolio.Performed && !portfolio.Reused &&
			len(portfolio.GeneratedFiles) == 0 && portfolio.TestOptimization == "OUT_OF_SCOPE"
	case optimizePortfolioRetained:
		return state.Phase == "QUALIFIED" && portfolio.Reason != "" && !portfolio.Performed &&
			!portfolio.Reused && len(portfolio.GeneratedFiles) == 0 && portfolio.TestOptimization == "OUT_OF_SCOPE"
	case optimizePortfolioComplete:
		return optimizeStringIn(state.Phase, "QUALIFIED", "ACTIVE", "STALE") && portfolio.Reason != "" &&
			(portfolio.Performed != portfolio.Reused) &&
			optimizeStringIn(portfolio.Reason, optimizePortfolioReasonStored, optimizePortfolioReasonReused) &&
			validOptimizeFamily(portfolio.ChangeFamily) && validOptimizeSHA(portfolio.FamilySHA256) &&
			validOptimizeSHA(portfolio.ProfileSHA256) && portfolio.ProfileCount >= 1 &&
			portfolio.ProfileCount <= optimizePortfolioMaximumEntries &&
			validOptimizeGeneratedPath(portfolio.PortfolioFile) &&
			validOptimizeGeneratedPath(portfolio.ProfileFile) &&
			validOptimizePortfolioGeneratedFiles(portfolio.GeneratedFiles) &&
			portfolio.TestOptimization == "OUT_OF_SCOPE"
	default:
		return false
	}
}

func validOptimizePortfolioGeneratedFiles(paths []string) bool {
	if len(paths) != 6 {
		return false
	}
	seen := make(map[string]bool, len(paths))
	for _, path := range paths {
		if !validOptimizeGeneratedPath(path) || seen[path] {
			return false
		}
		seen[path] = true
	}
	return true
}

func validOptimizeFamily(value string) bool {
	return optimizeStringIn(value, optimizeFamilyDependency, optimizeFamilyResource, optimizeFamilyLeaf, optimizeFamilyMixed)
}

func (run *optimizeRun) materializePortfolio(discovery optimizeDiscoveryResult, calibration optimizeCalibrationResult) optimizePortfolioResult {
	result := emptyOptimizePortfolio(calibration.Reason)
	if calibration.Status != optimizeCalibrationComplete || !calibration.Qualified {
		return result
	}
	artifacts, err := prepareOptimizePortfolioArtifacts(run.invocation, discovery, calibration)
	if err != nil {
		result.Status = optimizePortfolioRetained
		result.Reason = "PROFILE_MATERIALIZATION_FAILED"
		return result
	}
	indexPath := filepath.ToSlash(filepath.Join(run.invocation.stateRelative, "portfolio", optimizePortfolioIndexFile))
	portfolioScope := optimizePortfolioRepositoryScope(discovery.RepositoryID)
	portfolio, valid := loadOptimizePortfolio(run.invocation.repositoryRoot, indexPath, portfolioScope)
	if !valid {
		portfolio = optimizeProfilePortfolio{
			SchemaVersion:         optimizePortfolioSchemaVersion,
			Generation:            run.state.Generation,
			RepositoryScopeSHA256: portfolioScope,
			Profiles:              []optimizePortfolioEntry{},
			SelectionAuthorized:   false, ProductionAuthorized: false,
		}
	}
	for _, entry := range portfolio.Profiles {
		if entry.FamilySHA256 == artifacts.entry.FamilySHA256 && entry.ProfileSHA256 == artifacts.entry.ProfileSHA256 {
			return optimizePortfolioResultForEntry(optimizePortfolioReasonReused, false, true, indexPath, len(portfolio.Profiles), entry)
		}
	}
	profiles := upsertOptimizePortfolioEntry(portfolio.Profiles, artifacts.entry)
	if len(profiles) > optimizePortfolioMaximumEntries {
		result.Status = optimizePortfolioRetained
		result.Reason = "PROFILE_PORTFOLIO_CAPACITY_REACHED"
		return result
	}
	for relative, raw := range artifacts.files {
		absolute := filepath.Join(run.invocation.repositoryRoot, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(absolute), 0o700); err != nil || writePrivateAtomicFile(absolute, raw) != nil {
			result.Status = optimizePortfolioRetained
			result.Reason = "PROFILE_PORTFOLIO_WRITE_FAILED"
			return result
		}
	}
	if _, err := loadQualifiedPOCProfile(run.invocation.repositoryRoot, artifacts.entry.ProfilePath); err != nil {
		result.Status = optimizePortfolioRetained
		result.Reason = "PROFILE_VALIDATION_FAILED"
		return result
	}
	portfolio.Generation = run.state.Generation
	portfolio.Profiles = profiles
	portfolio.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := writeCanonicalPrivateJSON(filepath.Join(run.invocation.repositoryRoot, filepath.FromSlash(indexPath)), portfolio); err != nil {
		result.Status = optimizePortfolioRetained
		result.Reason = "PROFILE_PORTFOLIO_WRITE_FAILED"
		return result
	}
	if _, ok := loadOptimizePortfolio(run.invocation.repositoryRoot, indexPath, portfolioScope); !ok {
		result.Status = optimizePortfolioRetained
		result.Reason = "PROFILE_PORTFOLIO_VALIDATION_FAILED"
		return result
	}
	return optimizePortfolioResultForEntry(optimizePortfolioReasonStored, true, false, indexPath, len(profiles), artifacts.entry)
}

func upsertOptimizePortfolioEntry(existing []optimizePortfolioEntry, replacement optimizePortfolioEntry) []optimizePortfolioEntry {
	profiles := make([]optimizePortfolioEntry, 0, len(existing)+1)
	for _, entry := range existing {
		if entry.FamilySHA256 != replacement.FamilySHA256 {
			profiles = append(profiles, entry)
		}
	}
	profiles = append(profiles, replacement)
	sort.Slice(profiles, func(left, right int) bool { return profiles[left].FamilySHA256 < profiles[right].FamilySHA256 })
	return profiles
}

func prepareOptimizePortfolioArtifacts(invocation optimizeInvocation, discovery optimizeDiscoveryResult, calibration optimizeCalibrationResult) (optimizePortfolioArtifacts, error) {
	if !validOptimizeFamily(discovery.ChangeFamily) || len(discovery.ChangedProjects) == 0 || len(calibration.GeneratedFiles) != 1 {
		return optimizePortfolioArtifacts{}, errors.New("qualified portfolio inputs are incomplete")
	}
	discoveryDirectory := filepath.ToSlash(filepath.Join(invocation.stateRelative, "discovery"))
	sourcePaths := []string{
		filepath.ToSlash(filepath.Join(discoveryDirectory, "manifest.json")),
		filepath.ToSlash(filepath.Join(discoveryDirectory, "graph.json")),
		filepath.ToSlash(filepath.Join(discoveryDirectory, "generated-manifest.json")),
		calibration.GeneratedFiles[0],
	}
	profile, err := profilediscovery.QualifyStructuralProfile(profilediscovery.StructuralOptions{
		RepositoryRoot: invocation.repositoryRoot,
		ManifestPath:   sourcePaths[0], GraphPath: sourcePaths[1], GeneratedPath: sourcePaths[2], EvidencePath: sourcePaths[3],
	})
	if err != nil {
		return optimizePortfolioArtifacts{}, err
	}
	familySHA := optimizePortfolioFamilySHA(discovery)
	profileDirectory := filepath.ToSlash(filepath.Join(invocation.stateRelative, "portfolio", "profiles", familySHA))
	destinationPaths := []string{
		filepath.ToSlash(filepath.Join(profileDirectory, "manifest.json")),
		filepath.ToSlash(filepath.Join(profileDirectory, "graph.json")),
		filepath.ToSlash(filepath.Join(profileDirectory, "generated-manifest.json")),
		filepath.ToSlash(filepath.Join(profileDirectory, "evidence.json")),
	}
	profile.Impact.Manifest = destinationPaths[0]
	profile.Impact.Graph = destinationPaths[1]
	profile.Impact.GeneratedManifest = destinationPaths[2]
	for index := 0; index < 3; index++ {
		profile.Preconditions[index].Path = destinationPaths[index]
	}
	profilePath := filepath.ToSlash(filepath.Join(profileDirectory, "profile.json"))
	profileRaw, err := profilediscovery.RenderStructuralProfile(profile)
	if err != nil {
		return optimizePortfolioArtifacts{}, err
	}
	files := make(map[string][]byte, 5)
	digests := make([]string, 4)
	for index, source := range sourcePaths {
		raw, err := readOptimizePortfolioSource(invocation.repositoryRoot, source)
		if err != nil {
			return optimizePortfolioArtifacts{}, err
		}
		files[destinationPaths[index]] = raw
		digest := sha256.Sum256(raw)
		digests[index] = hex.EncodeToString(digest[:])
	}
	profileDigest := sha256.Sum256(profileRaw)
	profileSHA := hex.EncodeToString(profileDigest[:])
	files[profilePath] = profileRaw
	entry := optimizePortfolioEntry{
		Family: discovery.ChangeFamily, FamilySHA256: familySHA,
		ChangedProjects:      append([]string(nil), discovery.ChangedProjects...),
		RepositoryID:         discovery.RepositoryID,
		Entrypoints:          append([]string(nil), discovery.Entrypoints...),
		CandidateEntrypoints: append([]string(nil), discovery.CandidateEntrypoints...),
		RequiredOutputs:      append([]string(nil), discovery.RequiredOutputs...),
		CandidateOutputs:     append([]string(nil), discovery.CandidateOutputs...),
		TargetRevision:       discovery.TargetRevision,
		WrapperSHA256:        invocation.wrapperSHA256, ExecutableSHA256: invocation.executableSHA256,
		ManifestSHA256: digests[0], GraphSHA256: digests[1], GeneratedSHA256: digests[2],
		EvidenceSHA256: digests[3], ProfileSHA256: profileSHA, ProfilePath: profilePath,
		State: "QUALIFIED", SelectionAuthorized: false, ProductionAuthorized: false,
	}
	if discovery.Materialization.Status == optimizeMaterializationCaptured {
		materialization, err := prepareOptimizePortfolioMaterialization(invocation, discovery)
		if err != nil {
			return optimizePortfolioArtifacts{}, err
		}
		entry.Materialization = materialization
	}
	return optimizePortfolioArtifacts{entry: entry, files: files}, nil
}

func optimizePortfolioFamilySHA(discovery optimizeDiscoveryResult) string {
	return optimizePortfolioFamilyDigest(
		discovery.RepositoryID,
		discovery.ChangeFamily,
		discovery.ChangedProjects,
		discovery.Entrypoints,
		discovery.CandidateEntrypoints,
		discovery.RequiredOutputs,
		discovery.CandidateOutputs,
	)
}

func optimizePortfolioFamilyDigest(repositoryID, family string, projects, entrypoints, candidateEntrypoints, outputs, candidateOutputs []string) string {
	if len(candidateOutputs) == 0 {
		return optimizeDigest(
			"buildopt-optimize-profile-family-v1",
			optimizePortfolioRepositoryScope(repositoryID),
			family,
			optimizeDigest("buildopt-optimize-family-projects-v1", projects...),
			optimizeDigest("buildopt-optimize-family-entrypoints-v1", entrypoints...),
			optimizeDigest("buildopt-optimize-family-candidate-v1", candidateEntrypoints...),
			optimizeDigest("buildopt-optimize-family-outputs-v1", outputs...),
		)
	}
	return optimizeDigest(
		"buildopt-optimize-profile-family-v2",
		optimizePortfolioRepositoryScope(repositoryID),
		family,
		optimizeDigest("buildopt-optimize-family-projects-v1", projects...),
		optimizeDigest("buildopt-optimize-family-entrypoints-v1", entrypoints...),
		optimizeDigest("buildopt-optimize-family-candidate-v1", candidateEntrypoints...),
		optimizeDigest("buildopt-optimize-family-outputs-v1", outputs...),
		optimizeDigest("buildopt-optimize-family-candidate-outputs-v1", candidateOutputs...),
	)
}

func optimizePortfolioRepositoryScope(repositoryID string) string {
	return optimizeDigest("buildopt-optimize-portfolio-repository-v1", repositoryID)
}

func optimizePortfolioResultForEntry(reason string, performed, reused bool, indexPath string, count int, entry optimizePortfolioEntry) optimizePortfolioResult {
	generated := []string{
		indexPath,
		filepath.ToSlash(filepath.Join(filepath.Dir(entry.ProfilePath), "manifest.json")),
		filepath.ToSlash(filepath.Join(filepath.Dir(entry.ProfilePath), "graph.json")),
		filepath.ToSlash(filepath.Join(filepath.Dir(entry.ProfilePath), "generated-manifest.json")),
		filepath.ToSlash(filepath.Join(filepath.Dir(entry.ProfilePath), "evidence.json")),
		entry.ProfilePath,
	}
	sort.Strings(generated)
	return optimizePortfolioResult{
		Status: optimizePortfolioComplete, Reason: reason, Performed: performed, Reused: reused,
		ChangeFamily: entry.Family, FamilySHA256: entry.FamilySHA256, ProfileSHA256: entry.ProfileSHA256,
		ProfileCount: count, PortfolioFile: indexPath, ProfileFile: entry.ProfilePath, GeneratedFiles: generated,
		SelectionPerformed: false, ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
	}
}

func loadOptimizePortfolio(repositoryRoot, relativePath, repositoryScopeSHA string) (optimizeProfilePortfolio, bool) {
	raw, err := readOptimizePortfolioSource(repositoryRoot, relativePath)
	if err != nil || len(raw) > optimizePortfolioMaximumIndexBytes {
		return optimizeProfilePortfolio{}, false
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var portfolio optimizeProfilePortfolio
	if err := decoder.Decode(&portfolio); err != nil {
		return optimizeProfilePortfolio{}, false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) ||
		!validOptimizePortfolio(repositoryRoot, portfolio, repositoryScopeSHA) {
		return optimizeProfilePortfolio{}, false
	}
	return portfolio, true
}

func validOptimizePortfolio(repositoryRoot string, portfolio optimizeProfilePortfolio, repositoryScopeSHA string) bool {
	if portfolio.SchemaVersion != optimizePortfolioSchemaVersion || portfolio.Generation < 1 ||
		portfolio.RepositoryScopeSHA256 != repositoryScopeSHA || !validOptimizeSHA(repositoryScopeSHA) ||
		portfolio.SelectionAuthorized || portfolio.ProductionAuthorized || len(portfolio.Profiles) < 1 ||
		len(portfolio.Profiles) > optimizePortfolioMaximumEntries {
		return false
	}
	if _, err := time.Parse(time.RFC3339Nano, portfolio.UpdatedAt); err != nil {
		return false
	}
	previous := ""
	for _, entry := range portfolio.Profiles {
		if entry.FamilySHA256 <= previous ||
			optimizePortfolioRepositoryScope(entry.RepositoryID) != portfolio.RepositoryScopeSHA256 ||
			!validOptimizePortfolioEntry(repositoryRoot, entry) {
			return false
		}
		previous = entry.FamilySHA256
	}
	return true
}

func validOptimizePortfolioEntry(repositoryRoot string, entry optimizePortfolioEntry) bool {
	if !validOptimizeFamily(entry.Family) || !validOptimizeSHA(entry.FamilySHA256) ||
		!validOptimizeSHA(entry.WrapperSHA256) || !validOptimizeSHA(entry.ExecutableSHA256) ||
		!validOptimizeSHA(entry.ManifestSHA256) || !validOptimizeSHA(entry.GraphSHA256) ||
		!validOptimizeSHA(entry.GeneratedSHA256) || !validOptimizeSHA(entry.EvidenceSHA256) ||
		!validOptimizeSHA(entry.ProfileSHA256) || entry.State != "QUALIFIED" ||
		entry.SelectionAuthorized || entry.ProductionAuthorized || entry.RepositoryID == "" ||
		!validMeasurementRevision(entry.TargetRevision) || len(entry.ChangedProjects) < 1 ||
		len(entry.Entrypoints) < 1 || len(entry.CandidateEntrypoints) < 1 || len(entry.RequiredOutputs) < 1 ||
		!uniqueMeasurementStrings(entry.ChangedProjects) || !uniqueMeasurementStrings(entry.Entrypoints) ||
		!uniqueMeasurementStrings(entry.CandidateEntrypoints) || !uniqueMeasurementStrings(entry.RequiredOutputs) ||
		(len(entry.CandidateOutputs) > 0 && !uniqueMeasurementStrings(entry.CandidateOutputs)) ||
		!validOptimizePortfolioMaterialization(entry) ||
		!validOptimizeGeneratedPath(entry.ProfilePath) ||
		entry.FamilySHA256 != optimizePortfolioFamilyDigest(
			entry.RepositoryID, entry.Family, entry.ChangedProjects, entry.Entrypoints,
			entry.CandidateEntrypoints, entry.RequiredOutputs, entry.CandidateOutputs,
		) || filepath.Base(entry.ProfilePath) != "profile.json" ||
		filepath.Base(filepath.Dir(entry.ProfilePath)) != entry.FamilySHA256 {
		return false
	}
	directory := filepath.Dir(entry.ProfilePath)
	bindings := []struct{ path, sha string }{
		{filepath.ToSlash(filepath.Join(directory, "manifest.json")), entry.ManifestSHA256},
		{filepath.ToSlash(filepath.Join(directory, "graph.json")), entry.GraphSHA256},
		{filepath.ToSlash(filepath.Join(directory, "generated-manifest.json")), entry.GeneratedSHA256},
		{filepath.ToSlash(filepath.Join(directory, "evidence.json")), entry.EvidenceSHA256},
		{entry.ProfilePath, entry.ProfileSHA256},
	}
	for _, binding := range bindings {
		raw, err := readOptimizePortfolioSource(repositoryRoot, binding.path)
		if err != nil {
			return false
		}
		digest := sha256.Sum256(raw)
		if hex.EncodeToString(digest[:]) != binding.sha {
			return false
		}
	}
	profile, err := loadQualifiedPOCProfile(repositoryRoot, entry.ProfilePath)
	if err != nil || profile.RepositoryID != entry.RepositoryID || profile.Qualification == nil ||
		profile.Qualification.SHA256 != entry.EvidenceSHA256 ||
		profile.Qualification.RepositoryRevision != entry.TargetRevision {
		return false
	}
	_, preconditionsValid := evaluateQualifiedPOCPreconditions(repositoryRoot, profile.Preconditions)
	if !preconditionsValid {
		return false
	}
	return true
}

func validOptimizePortfolioMaterialization(entry optimizePortfolioEntry) bool {
	requiresMaterialization := len(entry.CandidateOutputs) > 0 &&
		!equalOptimizeStrings(entry.RequiredOutputs, entry.CandidateOutputs)
	if entry.Materialization == nil {
		return !requiresMaterialization
	}
	materialization := entry.Materialization
	if !requiresMaterialization || !validOptimizeSHA(materialization.ManifestSHA256) ||
		!validOptimizeSHA(materialization.PackSHA256) || materialization.PackSize < 1 ||
		materialization.PackSize > optimizeMaterializationMaxBytes ||
		len(materialization.ChunkSHA256) < 1 || len(materialization.ChunkSHA256) > 63 ||
		len(materialization.MaterializedProjects) < 1 ||
		!uniqueMeasurementStrings(materialization.MaterializedProjects) {
		return false
	}
	for _, digest := range materialization.ChunkSHA256 {
		if !validOptimizeSHA(digest) {
			return false
		}
	}
	return true
}

func readOptimizePortfolioSource(repositoryRoot, relativePath string) ([]byte, error) {
	if !validOptimizeGeneratedPath(relativePath) {
		return nil, errors.New("portfolio source path is invalid")
	}
	path := filepath.Join(repositoryRoot, filepath.FromSlash(relativePath))
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() < 1 || info.Size() > optimizePortfolioMaximumArtifactBytes {
		return nil, fmt.Errorf("portfolio source must be one bounded regular file: %s", relativePath)
	}
	return os.ReadFile(path)
}
