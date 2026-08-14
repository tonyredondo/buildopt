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
	"runtime"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
	"github.com/tonyredondo/buildopt/internal/localauthority"
	"github.com/tonyredondo/buildopt/internal/platformfs"
	"github.com/tonyredondo/buildopt/internal/profilediscovery"
)

const (
	profileProposalCacheSchema = "buildopt.poc/profile-proposal-cache/v1"
	maximumProposalCacheBytes  = 4 << 20
)

type profileProposalCacheBinding struct {
	SchemaVersion           string            `json:"schemaVersion"`
	Digest                  string            `json:"digest,omitempty"`
	RepositoryID            string            `json:"repositoryId"`
	PipelineClass           string            `json:"pipelineClass"`
	BaseRevision            string            `json:"baseRevision"`
	TargetRevision          string            `json:"targetRevision"`
	ChangedPaths            []string          `json:"changedPaths"`
	Entrypoints             []string          `json:"entrypoints"`
	RequiredOutputs         []string          `json:"requiredOutputs"`
	GlobalChanges           []string          `json:"globalChanges"`
	GradleCommand           string            `json:"gradleCommand"`
	GradleOptions           []string          `json:"gradleOptions"`
	TimeoutNanoseconds      int64             `json:"timeoutNanoseconds"`
	OwnerInputSHA256        string            `json:"ownerInputSha256,omitempty"`
	OutputEquivalenceSHA256 string            `json:"outputEquivalenceSha256,omitempty"`
	OutputPaths             []string          `json:"outputPaths"`
	WrapperFiles            map[string]string `json:"wrapperFiles"`
	ExecutableSHA256        string            `json:"executableSha256"`
}

type profileProposalCacheRecord struct {
	SchemaVersion   string                      `json:"schemaVersion"`
	Binding         profileProposalCacheBinding `json:"binding"`
	Report          profileProposalReport       `json:"report"`
	Documents       map[string]string           `json:"documents"`
	Snapshot        string                      `json:"snapshot"`
	ArtifactDigests map[string]string           `json:"artifactDigests"`
}

func prepareProfileProposalCacheBinding(root, targetRevision string, changedPaths []string, config structuralProposalConfig, outputEquivalenceSHA256 string) (profileProposalCacheBinding, error) {
	executable, err := os.Executable()
	if err != nil {
		return profileProposalCacheBinding{}, fmt.Errorf("resolve BuildOpt executable for proposal replay: %w", err)
	}
	executableSHA256, err := hashMeasurementFile(executable)
	if err != nil {
		return profileProposalCacheBinding{}, fmt.Errorf("hash BuildOpt executable for proposal replay: %w", err)
	}
	wrapperFiles, err := profileProposalWrapperDigests(root, config.gradleCommand)
	if err != nil {
		return profileProposalCacheBinding{}, err
	}
	binding := profileProposalCacheBinding{
		SchemaVersion: profileProposalCacheSchema,
		RepositoryID:  config.repositoryID, PipelineClass: config.pipelineClass,
		BaseRevision: config.baseRevision, TargetRevision: targetRevision,
		ChangedPaths:            append([]string(nil), changedPaths...),
		Entrypoints:             append([]string(nil), config.entrypoints...),
		RequiredOutputs:         append([]string(nil), config.requiredOutputs...),
		GlobalChanges:           append([]string(nil), config.globalChanges...),
		GradleCommand:           config.gradleCommand,
		GradleOptions:           append([]string(nil), config.gradleOptions...),
		TimeoutNanoseconds:      config.timeout.Nanoseconds(),
		OwnerInputSHA256:        config.ownerInputSHA256,
		OutputEquivalenceSHA256: outputEquivalenceSHA256,
		OutputPaths: []string{
			config.outputContractOutput, config.manifestOutput, config.graphOutput,
			config.generatedOutput, config.fallbackOutput, config.proposalOutput,
		},
		WrapperFiles: wrapperFiles, ExecutableSHA256: executableSHA256,
	}
	canonical, err := json.Marshal(binding)
	if err != nil {
		return profileProposalCacheBinding{}, fmt.Errorf("encode proposal replay binding: %w", err)
	}
	digest := sha256.Sum256(canonical)
	binding.Digest = hex.EncodeToString(digest[:])
	return binding, nil
}

func profileProposalWrapperDigests(root, configuredCommand string) (map[string]string, error) {
	command := configuredCommand
	if command == "" {
		command = "gradlew"
		if runtime.GOOS == "windows" {
			command += ".bat"
		}
	}
	if !filepath.IsAbs(command) {
		command = filepath.Join(root, command)
	}
	command = filepath.Clean(command)
	files := []string{command}
	base := filepath.Base(command)
	if base == "gradlew" || base == "gradlew.bat" {
		wrapperRoot := filepath.Dir(command)
		files = append(files,
			filepath.Join(wrapperRoot, "gradle", "wrapper", "gradle-wrapper.properties"),
			filepath.Join(wrapperRoot, "gradle", "wrapper", "gradle-wrapper.jar"),
		)
	}
	digests := make(map[string]string, len(files))
	for _, candidate := range files {
		info, err := os.Lstat(candidate)
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("proposal replay requires a regular Gradle command or Wrapper file: %s", candidate)
		}
		digest, err := hashMeasurementFile(candidate)
		if err != nil {
			return nil, fmt.Errorf("hash proposal replay Gradle input %s: %w", candidate, err)
		}
		name := candidate
		if relative, relativeErr := filepath.Rel(root, candidate); relativeErr == nil && relative != ".." && !filepath.IsAbs(relative) {
			name = filepath.ToSlash(relative)
		}
		digests[name] = digest
	}
	return digests, nil
}

func loadProfileProposalCache(cacheRoot string, binding profileProposalCacheBinding, config structuralProposalConfig) (profileProposalReport, map[string][]byte, bool, error) {
	if err := prepareProfileProposalCacheRoot(cacheRoot); err != nil {
		return profileProposalReport{}, nil, false, err
	}
	path := filepath.Join(cacheRoot, binding.Digest+".json")
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return profileProposalReport{}, nil, false, nil
	} else if err != nil {
		return profileProposalReport{}, nil, false, fmt.Errorf("inspect proposal replay state: %w", err)
	}
	raw, err := localauthority.ReadPrivateFile(path, maximumProposalCacheBytes)
	if err != nil {
		return profileProposalReport{}, nil, false, fmt.Errorf("read proposal replay state: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var record profileProposalCacheRecord
	if err := decoder.Decode(&record); err != nil {
		return profileProposalReport{}, nil, false, fmt.Errorf("decode proposal replay state: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return profileProposalReport{}, nil, false, errors.New("proposal replay state has trailing content")
	}
	if record.SchemaVersion != profileProposalCacheSchema || !equalProfileProposalCacheBindings(record.Binding, binding) {
		return profileProposalReport{}, nil, false, errors.New("proposal replay binding drift")
	}
	documents, err := validateProfileProposalCacheRecord(record, config)
	if err != nil {
		return profileProposalReport{}, nil, false, err
	}
	return record.Report, documents, true, nil
}

func storeProfileProposalCache(cacheRoot string, binding profileProposalCacheBinding, config structuralProposalConfig, report profileProposalReport, documents map[string][]byte, snapshot buildimpact.DiscoverySnapshot) error {
	if err := prepareProfileProposalCacheRoot(cacheRoot); err != nil {
		return err
	}
	reportRaw, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("encode proposal replay report: %w", err)
	}
	snapshotRaw, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("encode proposal replay snapshot: %w", err)
	}
	record := profileProposalCacheRecord{
		SchemaVersion: profileProposalCacheSchema, Binding: binding, Report: report,
		Documents: make(map[string]string, len(documents)), Snapshot: string(snapshotRaw),
		ArtifactDigests: map[string]string{
			"report":   profileProposalCacheSHA(reportRaw),
			"snapshot": profileProposalCacheSHA(snapshotRaw),
		},
	}
	for name, raw := range documents {
		record.Documents[name] = string(raw)
		record.ArtifactDigests["document:"+name] = profileProposalCacheSHA(raw)
	}
	if _, err := validateProfileProposalCacheRecord(record, config); err != nil {
		return fmt.Errorf("validate proposal replay state before publication: %w", err)
	}
	raw, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("encode proposal replay state: %w", err)
	}
	raw = append(raw, '\n')
	if len(raw) > maximumProposalCacheBytes {
		return errors.New("proposal replay state exceeds the bounded cache record size")
	}
	path := filepath.Join(cacheRoot, binding.Digest+".json")
	temporary, err := os.CreateTemp(cacheRoot, ".proposal-replay-*.tmp")
	if err != nil {
		return fmt.Errorf("create proposal replay temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("protect proposal replay temporary file: %w", err)
	}
	if _, err := temporary.Write(raw); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write proposal replay state: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync proposal replay state: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close proposal replay state: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("publish proposal replay state: %w", err)
	}
	return nil
}

func validateProfileProposalCacheRecord(record profileProposalCacheRecord, config structuralProposalConfig) (map[string][]byte, error) {
	if len(record.Documents) != 5 || len(record.ArtifactDigests) != 7 {
		return nil, errors.New("proposal replay artifact inventory is incomplete")
	}
	reportRaw, err := json.Marshal(record.Report)
	if err != nil || record.ArtifactDigests["report"] != profileProposalCacheSHA(reportRaw) || record.ArtifactDigests["snapshot"] != profileProposalCacheSHA([]byte(record.Snapshot)) {
		return nil, errors.New("proposal replay report or snapshot digest drift")
	}
	documents := make(map[string][]byte, len(record.Documents))
	for name, content := range record.Documents {
		raw := []byte(content)
		if record.ArtifactDigests["document:"+name] != profileProposalCacheSHA(raw) {
			return nil, fmt.Errorf("proposal replay document digest drift: %s", name)
		}
		documents[name] = raw
	}
	expectedDocuments := []string{config.outputContractOutput, config.manifestOutput, config.graphOutput, config.generatedOutput, config.fallbackOutput}
	for _, name := range expectedDocuments {
		if _, ok := documents[name]; !ok {
			return nil, fmt.Errorf("proposal replay document is missing: %s", name)
		}
	}
	outputContract, err := parseConfirmedOutputContract(documents[config.outputContractOutput])
	if err != nil || outputContract.RepositoryID != config.repositoryID || outputContract.PipelineClass != config.pipelineClass || outputContract.RepositoryRevision != record.Binding.TargetRevision || !sameStringSet(outputContract.OriginalEntrypoints, config.entrypoints) || !sameStringSet(outputContract.DeclaredOutputs, config.requiredOutputs) {
		return nil, errors.New("proposal replay output contract drift")
	}
	manifest, err := buildimpact.ParseManifest(documents[config.manifestOutput], config.repositoryID, config.pipelineClass)
	if err != nil {
		return nil, fmt.Errorf("proposal replay manifest drift: %w", err)
	}
	generated, err := buildimpact.GenerateImpact(manifest, []byte(record.Snapshot))
	if err != nil {
		return nil, fmt.Errorf("proposal replay discovery snapshot drift: %w", err)
	}
	if !bytes.Equal(generated.GraphJSON, documents[config.graphOutput]) || !bytes.Equal(generated.GeneratedJSON, documents[config.generatedOutput]) {
		return nil, errors.New("proposal replay graph binding drift")
	}
	if !generated.Snapshot.Complete {
		return nil, errors.New("proposal replay snapshot is incomplete")
	}
	if _, err := buildimpact.ResolveProjectOwners(generated.Snapshot, record.Binding.ChangedPaths); err != nil {
		return nil, errors.New("proposal replay source ownership drift")
	}
	analysis := profilediscovery.AnalyzeGeneratedOpportunity(manifest, generated.Graph, generated.Generated)
	if record.Report.SchemaVersion != "buildopt.poc/profile-proposal/v1" || record.Report.Decision != profilediscovery.DecisionMeasure || record.Report.Reason != analysis.Reason || record.Report.Analysis == nil || record.Report.Analysis.Decision != analysis.Decision || record.Report.RepositoryID != config.repositoryID || record.Report.PipelineClass != config.pipelineClass || record.Report.BaseRevision != config.baseRevision || record.Report.TargetRevision != record.Binding.TargetRevision || record.Report.UnknownRelationships || record.Report.ActivationAutomatic || record.Report.ProductionAuthorized || record.Report.TestOptimization != "OUT_OF_SCOPE" || record.Report.Documents.OutputContract != config.outputContractOutput || record.Report.Documents.Manifest != config.manifestOutput || record.Report.Documents.Graph != config.graphOutput || record.Report.Documents.Generated != config.generatedOutput || record.Report.Documents.FallbackChanges != config.fallbackOutput || record.Report.Documents.Proposal != config.proposalOutput || record.Report.OwnerInputSHA256 != config.ownerInputSHA256 || record.Report.OutputEquivalenceSHA256 != record.Binding.OutputEquivalenceSHA256 {
		return nil, errors.New("proposal replay report drift")
	}
	return documents, nil
}

func prepareProfileProposalCacheRoot(root string) error {
	if root == "" || !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return errors.New("proposal replay cache directory must be absolute and clean")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return fmt.Errorf("create proposal replay cache directory: %w", err)
	}
	if err := platformfs.ValidateNoLinks(root); err != nil {
		return errors.New("proposal replay cache directory must not resolve through symlinks")
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("proposal replay cache path is not a real directory")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o700 {
		return errors.New("proposal replay cache directory must have mode 0700")
	}
	return nil
}

func equalProfileProposalCacheBindings(left, right profileProposalCacheBinding) bool {
	leftRaw, leftErr := json.Marshal(left)
	rightRaw, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftRaw, rightRaw)
}

func profileProposalCacheSHA(raw []byte) string {
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}
