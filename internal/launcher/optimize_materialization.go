package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	optimizeMaterializationSchema      = "buildopt.poc/verified-output-materialization/v1"
	optimizeMaterializationCaptured    = "CAPTURED"
	optimizeMaterializationNotRequired = "NOT_REQUIRED"
	optimizeMaterializationReasonReady = "VERIFIED_UNAFFECTED_OUTPUTS_CAPTURED"
	optimizeMaterializationReasonNone  = "NO_UNAFFECTED_OUTPUTS"
	optimizeMaterializationBinding     = "VERIFIED_UNAFFECTED_OUTPUTS"
	optimizeMaterializationFailure     = "VERIFIED_OUTPUT_MATERIALIZATION_FAILED"
	optimizeMaterializationMaxFiles    = 250000
	optimizeMaterializationMaxBytes    = int64(2 << 30)
	optimizeMaterializationMaxManifest = 32 << 20
)

type optimizeOutputMaterialization struct {
	Status         string   `json:"status"`
	Reason         string   `json:"reason"`
	Patterns       []string `json:"patterns"`
	ManifestFile   string   `json:"manifestFile,omitempty"`
	ManifestSHA256 string   `json:"manifestSha256,omitempty"`
	FileCount      int      `json:"fileCount"`
	ByteCount      int64    `json:"byteCount"`
}

type optimizeOutputMaterializationManifest struct {
	SchemaVersion    string                               `json:"schemaVersion"`
	RepositoryID     string                               `json:"repositoryId"`
	TargetRevision   string                               `json:"targetRevision"`
	RequiredOutputs  []string                             `json:"requiredOutputs"`
	CandidateOutputs []string                             `json:"candidateOutputs"`
	Entries          []optimizeOutputMaterializationEntry `json:"entries"`
}

type optimizeOutputMaterializationEntry struct {
	Path   string `json:"path"`
	Blob   string `json:"blob"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Mode   uint32 `json:"mode"`
}

type optimizeMaterializationPayload struct {
	entry optimizeOutputMaterializationEntry
	raw   []byte
}

func captureOptimizeOutputMaterialization(
	invocation optimizeInvocation,
	discovery optimizeDiscoveryResult,
) (optimizeOutputMaterialization, error) {
	patterns := optimizeMaterializationPatterns(discovery.RequiredOutputs, discovery.CandidateOutputs)
	if len(patterns) == 0 {
		return optimizeOutputMaterialization{
			Status: optimizeMaterializationNotRequired, Reason: optimizeMaterializationReasonNone,
			Patterns: []string{},
		}, nil
	}
	files, err := collectOptimizeMaterializationFiles(invocation.repositoryRoot, patterns)
	if err != nil {
		return optimizeOutputMaterialization{}, err
	}
	directory := filepath.Join(invocation.stateDirectory, "materialization")
	blobDirectory := filepath.Join(directory, "blobs")
	if err := os.MkdirAll(blobDirectory, 0o700); err != nil {
		return optimizeOutputMaterialization{}, fmt.Errorf("create output materialization state: %w", err)
	}
	entries := make([]optimizeOutputMaterializationEntry, 0, len(files))
	var byteCount int64
	for _, relative := range files {
		absolute := filepath.Join(invocation.repositoryRoot, filepath.FromSlash(relative))
		info, err := os.Lstat(absolute)
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return optimizeOutputMaterialization{}, fmt.Errorf("capture output %s: not a regular file", relative)
		}
		raw, err := os.ReadFile(absolute)
		if err != nil {
			return optimizeOutputMaterialization{}, fmt.Errorf("read output %s: %w", relative, err)
		}
		byteCount += int64(len(raw))
		if byteCount > optimizeMaterializationMaxBytes {
			return optimizeOutputMaterialization{}, errors.New("output materialization exceeds the POC byte bound")
		}
		digest := sha256.Sum256(raw)
		sha := hex.EncodeToString(digest[:])
		blobRelative := filepath.ToSlash(filepath.Join(invocation.stateRelative, "materialization", "blobs", sha))
		blobAbsolute := filepath.Join(invocation.repositoryRoot, filepath.FromSlash(blobRelative))
		if existing, readErr := os.ReadFile(blobAbsolute); readErr == nil {
			existingDigest := sha256.Sum256(existing)
			if hex.EncodeToString(existingDigest[:]) != sha {
				return optimizeOutputMaterialization{}, errors.New("existing materialization blob is corrupt")
			}
		} else if !os.IsNotExist(readErr) {
			return optimizeOutputMaterialization{}, fmt.Errorf("inspect materialization blob: %w", readErr)
		} else if err := writePrivateAtomicFile(blobAbsolute, raw); err != nil {
			return optimizeOutputMaterialization{}, fmt.Errorf("write materialization blob: %w", err)
		}
		entries = append(entries, optimizeOutputMaterializationEntry{
			Path: relative, Blob: blobRelative, SHA256: sha,
			Size: int64(len(raw)), Mode: uint32(info.Mode().Perm()),
		})
	}
	manifest := optimizeOutputMaterializationManifest{
		SchemaVersion: optimizeMaterializationSchema,
		RepositoryID:  discovery.RepositoryID, TargetRevision: discovery.TargetRevision,
		RequiredOutputs:  append([]string(nil), discovery.RequiredOutputs...),
		CandidateOutputs: append([]string(nil), discovery.CandidateOutputs...),
		Entries:          entries,
	}
	manifestRelative := filepath.ToSlash(filepath.Join(invocation.stateRelative, "materialization", "manifest.json"))
	manifestAbsolute := filepath.Join(invocation.repositoryRoot, filepath.FromSlash(manifestRelative))
	if err := writeCanonicalPrivateJSON(manifestAbsolute, manifest); err != nil {
		return optimizeOutputMaterialization{}, fmt.Errorf("write output materialization manifest: %w", err)
	}
	raw, err := os.ReadFile(manifestAbsolute)
	if err != nil {
		return optimizeOutputMaterialization{}, err
	}
	digest := sha256.Sum256(raw)
	return optimizeOutputMaterialization{
		Status: optimizeMaterializationCaptured, Reason: optimizeMaterializationReasonReady,
		Patterns: patterns, ManifestFile: manifestRelative,
		ManifestSHA256: hex.EncodeToString(digest[:]), FileCount: len(entries), ByteCount: byteCount,
	}, nil
}

func (run *optimizeRun) materializeCandidateOutputs() error {
	discovery := optimizeDiscoveryResult{}
	if run.centralReplay != nil {
		discovery = run.centralReplay.discovery
	} else if run.previousState != nil {
		discovery = run.previousState.Discovery
	}
	if discovery.Status != optimizeDiscoveryComplete && discovery.Status != optimizeDiscoveryRemoteRevalidated {
		return errors.New("verified output discovery is unavailable")
	}
	materialization := discovery.Materialization
	if materialization.Status == optimizeMaterializationNotRequired {
		return nil
	}
	manifest, payloads, err := loadOptimizeOutputMaterialization(run.invocation, discovery)
	if err != nil {
		return err
	}
	for _, payload := range payloads {
		destination := filepath.Join(run.invocation.repositoryRoot, filepath.FromSlash(payload.entry.Path))
		if existing, readErr := os.ReadFile(destination); readErr == nil {
			digest := sha256.Sum256(existing)
			if hex.EncodeToString(digest[:]) != payload.entry.SHA256 {
				return fmt.Errorf("required output %s contains stale bytes", payload.entry.Path)
			}
			continue
		} else if !os.IsNotExist(readErr) {
			return fmt.Errorf("inspect required output %s: %w", payload.entry.Path, readErr)
		}
		if err := ensureOptimizeMaterializationParent(run.invocation.repositoryRoot, payload.entry.Path); err != nil {
			return err
		}
		if err := writeOptimizeMaterializedFile(destination, payload.raw, fs.FileMode(payload.entry.Mode)); err != nil {
			return fmt.Errorf("materialize required output %s: %w", payload.entry.Path, err)
		}
	}
	if len(manifest.Entries) != materialization.FileCount {
		return errors.New("materialized output count drifted")
	}
	return nil
}

func loadOptimizeOutputMaterialization(
	invocation optimizeInvocation,
	discovery optimizeDiscoveryResult,
) (optimizeOutputMaterializationManifest, []optimizeMaterializationPayload, error) {
	materialization := discovery.Materialization
	if !validOptimizeOutputMaterializationShape(materialization, true) || materialization.Status != optimizeMaterializationCaptured {
		return optimizeOutputMaterializationManifest{}, nil, errors.New("output materialization metadata is invalid")
	}
	raw, err := os.ReadFile(filepath.Join(invocation.repositoryRoot, filepath.FromSlash(materialization.ManifestFile)))
	if err != nil || len(raw) == 0 || len(raw) > optimizeMaterializationMaxManifest {
		return optimizeOutputMaterializationManifest{}, nil, errors.New("output materialization manifest is unavailable")
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != materialization.ManifestSHA256 {
		return optimizeOutputMaterializationManifest{}, nil, errors.New("output materialization manifest digest drifted")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var manifest optimizeOutputMaterializationManifest
	if err := decoder.Decode(&manifest); err != nil {
		return optimizeOutputMaterializationManifest{}, nil, fmt.Errorf("decode output materialization manifest: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return optimizeOutputMaterializationManifest{}, nil, errors.New("output materialization manifest has trailing content")
	}
	if manifest.SchemaVersion != optimizeMaterializationSchema ||
		manifest.RepositoryID != discovery.RepositoryID || manifest.TargetRevision != discovery.TargetRevision ||
		!equalOptimizeStrings(manifest.RequiredOutputs, discovery.RequiredOutputs) ||
		!equalOptimizeStrings(manifest.CandidateOutputs, discovery.CandidateOutputs) ||
		len(manifest.Entries) != materialization.FileCount || len(manifest.Entries) == 0 {
		return optimizeOutputMaterializationManifest{}, nil, errors.New("output materialization manifest binding drifted")
	}
	payloads := make([]optimizeMaterializationPayload, 0, len(manifest.Entries))
	previous := ""
	var byteCount int64
	for _, entry := range manifest.Entries {
		if entry.Path <= previous || !validObservedOutputPath(entry.Path) || !validOptimizeSHA(entry.SHA256) ||
			entry.Blob != filepath.ToSlash(filepath.Join(invocation.stateRelative, "materialization", "blobs", entry.SHA256)) ||
			entry.Size < 0 || entry.Mode > 0o777 {
			return optimizeOutputMaterializationManifest{}, nil, errors.New("output materialization entry is invalid")
		}
		previous = entry.Path
		blob, err := os.ReadFile(filepath.Join(invocation.repositoryRoot, filepath.FromSlash(entry.Blob)))
		if err != nil || int64(len(blob)) != entry.Size {
			return optimizeOutputMaterializationManifest{}, nil, fmt.Errorf("materialization blob for %s is unavailable", entry.Path)
		}
		blobDigest := sha256.Sum256(blob)
		if hex.EncodeToString(blobDigest[:]) != entry.SHA256 {
			return optimizeOutputMaterializationManifest{}, nil, fmt.Errorf("materialization blob for %s is corrupt", entry.Path)
		}
		byteCount += entry.Size
		if byteCount > optimizeMaterializationMaxBytes {
			return optimizeOutputMaterializationManifest{}, nil, errors.New("output materialization exceeds the POC byte bound")
		}
		payloads = append(payloads, optimizeMaterializationPayload{entry: entry, raw: blob})
	}
	if byteCount != materialization.ByteCount {
		return optimizeOutputMaterializationManifest{}, nil, errors.New("output materialization byte count drifted")
	}
	return manifest, payloads, nil
}

func optimizeMaterializationPatterns(required, candidate []string) []string {
	selected := make(map[string]bool, len(candidate))
	for _, pattern := range candidate {
		selected[pattern] = true
	}
	patterns := make([]string, 0, len(required))
	for _, pattern := range required {
		if !selected[pattern] {
			patterns = append(patterns, pattern)
		}
	}
	sort.Strings(patterns)
	return patterns
}

func collectOptimizeMaterializationFiles(repositoryRoot string, patterns []string) ([]string, error) {
	if len(patterns) == 0 || len(patterns) > 256 || !uniqueMeasurementStrings(patterns) {
		return nil, errors.New("output materialization patterns are invalid")
	}
	for _, pattern := range patterns {
		if !validOutputContractPattern(pattern) {
			return nil, errors.New("output materialization pattern is unsafe")
		}
	}
	files := []string{}
	err := filepath.WalkDir(repositoryRoot, func(candidate string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(repositoryRoot, candidate)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.IsDir() {
			if relative == ".git" || relative == ".buildopt" {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		matched := false
		for _, pattern := range patterns {
			if matchProposalGlob(pattern, relative) {
				matched = true
				break
			}
		}
		if !matched {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("materialized output %s is not a regular file", relative)
		}
		files = append(files, relative)
		if len(files) > optimizeMaterializationMaxFiles {
			return errors.New("output materialization file count exceeds the POC bound")
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("collect output materialization files: %w", err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil, errors.New("unaffected output materialization produced no files")
	}
	return files, nil
}

func validOptimizeOutputMaterializationShape(materialization optimizeOutputMaterialization, complete bool) bool {
	if !complete {
		return materialization.Status == "" && materialization.Reason == "" &&
			len(materialization.Patterns) == 0 && materialization.ManifestFile == "" &&
			materialization.ManifestSHA256 == "" && materialization.FileCount == 0 &&
			materialization.ByteCount == 0
	}
	if materialization.Status == optimizeMaterializationNotRequired {
		return materialization.Reason == optimizeMaterializationReasonNone && len(materialization.Patterns) == 0 &&
			materialization.ManifestFile == "" && materialization.ManifestSHA256 == "" &&
			materialization.FileCount == 0 && materialization.ByteCount == 0
	}
	if materialization.Status != optimizeMaterializationCaptured || materialization.Reason != optimizeMaterializationReasonReady ||
		len(materialization.Patterns) == 0 || len(materialization.Patterns) > 256 ||
		!uniqueMeasurementStrings(materialization.Patterns) || !validOptimizeGeneratedPath(materialization.ManifestFile) ||
		!validOptimizeSHA(materialization.ManifestSHA256) || materialization.FileCount < 1 ||
		materialization.FileCount > optimizeMaterializationMaxFiles || materialization.ByteCount < 0 ||
		materialization.ByteCount > optimizeMaterializationMaxBytes {
		return false
	}
	return true
}

func ensureOptimizeMaterializationParent(repositoryRoot, relative string) error {
	parent := filepath.Dir(filepath.FromSlash(relative))
	current := repositoryRoot
	if parent == "." {
		return nil
	}
	for _, segment := range strings.Split(parent, string(filepath.Separator)) {
		current = filepath.Join(current, segment)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			if err := os.Mkdir(current, 0o700); err != nil && !os.IsExist(err) {
				return fmt.Errorf("create materialization parent: %w", err)
			}
			info, err = os.Lstat(current)
		}
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("materialization parent is not a safe directory")
		}
	}
	return nil
}

func writeOptimizeMaterializedFile(path string, raw []byte, mode fs.FileMode) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".buildopt-output-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(mode.Perm()); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(raw); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := replaceManagedFile(temporaryPath, path); err != nil {
		return err
	}
	return syncManagedDirectory(filepath.Dir(path))
}
