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
	"time"
)

const (
	optimizeMaterializationSchema      = "buildopt.poc/verified-output-materialization/v2"
	optimizeMaterializationCaptured    = "CAPTURED"
	optimizeMaterializationNotRequired = "NOT_REQUIRED"
	optimizeMaterializationReasonReady = "VERIFIED_UNAFFECTED_OUTPUTS_CAPTURED"
	optimizeMaterializationReasonNone  = "NO_UNAFFECTED_OUTPUTS"
	optimizeMaterializationBinding     = "VERIFIED_UNAFFECTED_OUTPUTS"
	optimizeMaterializationFailure     = "VERIFIED_OUTPUT_MATERIALIZATION_FAILED"
	optimizeMaterializationMaxFiles    = 250000
	optimizeMaterializationMaxBytes    = int64(2 << 30)
	optimizeMaterializationMaxManifest = 32 << 20
	optimizeMaterializationPackName    = "payload.pack"
)

type optimizeOutputMaterialization struct {
	Status         string                           `json:"status"`
	Reason         string                           `json:"reason"`
	Patterns       []string                         `json:"patterns"`
	ManifestFile   string                           `json:"manifestFile,omitempty"`
	ManifestSHA256 string                           `json:"manifestSha256,omitempty"`
	FileCount      int                              `json:"fileCount"`
	ByteCount      int64                            `json:"byteCount"`
	Economics      optimizeMaterializationEconomics `json:"economics"`
}

type optimizeMaterializationEconomics struct {
	CollectMS  int64 `json:"collectMs"`
	PackMS     int64 `json:"packMs"`
	ManifestMS int64 `json:"manifestMs"`
	TotalMS    int64 `json:"totalMs"`
}

type optimizeOutputMaterializationManifest struct {
	SchemaVersion    string                               `json:"schemaVersion"`
	RepositoryID     string                               `json:"repositoryId"`
	TargetRevision   string                               `json:"targetRevision"`
	RequiredOutputs  []string                             `json:"requiredOutputs"`
	CandidateOutputs []string                             `json:"candidateOutputs"`
	PackFile         string                               `json:"packFile"`
	PackSHA256       string                               `json:"packSha256"`
	PackSize         int64                                `json:"packSize"`
	Entries          []optimizeOutputMaterializationEntry `json:"entries"`
}

type optimizeOutputMaterializationEntry struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Mode   uint32 `json:"mode"`
	Offset int64  `json:"offset"`
}

type optimizeMaterializationPayload struct {
	entry optimizeOutputMaterializationEntry
	raw   []byte
}

func captureOptimizeOutputMaterialization(
	invocation optimizeInvocation,
	discovery optimizeDiscoveryResult,
) (optimizeOutputMaterialization, error) {
	totalStarted := time.Now()
	patterns := optimizeMaterializationPatterns(discovery.RequiredOutputs, discovery.CandidateOutputs)
	if len(patterns) == 0 {
		return optimizeOutputMaterialization{
			Status: optimizeMaterializationNotRequired, Reason: optimizeMaterializationReasonNone,
			Patterns: []string{},
		}, nil
	}
	collectStarted := time.Now()
	files, err := collectOptimizeMaterializationFiles(invocation.repositoryRoot, patterns)
	if err != nil {
		return optimizeOutputMaterialization{}, err
	}
	collectMS := elapsedOptimizeEconomicsMS(collectStarted)
	directory := filepath.Join(invocation.stateDirectory, "materialization")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return optimizeOutputMaterialization{}, fmt.Errorf("create output materialization state: %w", err)
	}
	packStarted := time.Now()
	packRelative := filepath.ToSlash(filepath.Join(invocation.stateRelative, "materialization", optimizeMaterializationPackName))
	packAbsolute := filepath.Join(invocation.repositoryRoot, filepath.FromSlash(packRelative))
	pack, err := os.CreateTemp(directory, ".buildopt-pack-*")
	if err != nil {
		return optimizeOutputMaterialization{}, fmt.Errorf("create output materialization pack: %w", err)
	}
	packTemporary := pack.Name()
	defer pack.Close()
	defer os.Remove(packTemporary)
	if err := pack.Chmod(0o600); err != nil {
		_ = pack.Close()
		return optimizeOutputMaterialization{}, fmt.Errorf("protect output materialization pack: %w", err)
	}
	packDigest := sha256.New()
	entries := make([]optimizeOutputMaterializationEntry, 0, len(files))
	var byteCount int64
	for _, relative := range files {
		absolute := filepath.Join(invocation.repositoryRoot, filepath.FromSlash(relative))
		info, err := os.Lstat(absolute)
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return optimizeOutputMaterialization{}, fmt.Errorf("capture output %s: not a regular file", relative)
		}
		file, err := os.Open(absolute)
		if err != nil {
			_ = pack.Close()
			return optimizeOutputMaterialization{}, fmt.Errorf("open output %s: %w", relative, err)
		}
		digest := sha256.New()
		size, copyErr := io.Copy(io.MultiWriter(pack, packDigest, digest), file)
		closeErr := file.Close()
		if copyErr != nil || closeErr != nil || size != info.Size() {
			_ = pack.Close()
			return optimizeOutputMaterialization{}, fmt.Errorf("pack output %s", relative)
		}
		byteCount += size
		if byteCount > optimizeMaterializationMaxBytes {
			_ = pack.Close()
			return optimizeOutputMaterialization{}, errors.New("output materialization exceeds the POC byte bound")
		}
		entries = append(entries, optimizeOutputMaterializationEntry{
			Path: relative, SHA256: hex.EncodeToString(digest.Sum(nil)),
			Size: size, Mode: uint32(info.Mode().Perm()), Offset: byteCount - size,
		})
	}
	if err := pack.Sync(); err != nil {
		_ = pack.Close()
		return optimizeOutputMaterialization{}, fmt.Errorf("sync output materialization pack: %w", err)
	}
	if err := pack.Close(); err != nil {
		return optimizeOutputMaterialization{}, fmt.Errorf("close output materialization pack: %w", err)
	}
	if err := replaceManagedFile(packTemporary, packAbsolute); err != nil {
		return optimizeOutputMaterialization{}, fmt.Errorf("publish output materialization pack: %w", err)
	}
	if err := syncManagedDirectory(directory); err != nil {
		return optimizeOutputMaterialization{}, fmt.Errorf("sync output materialization directory: %w", err)
	}
	packMS := elapsedOptimizeEconomicsMS(packStarted)
	manifestStarted := time.Now()
	manifest := optimizeOutputMaterializationManifest{
		SchemaVersion: optimizeMaterializationSchema,
		RepositoryID:  discovery.RepositoryID, TargetRevision: discovery.TargetRevision,
		RequiredOutputs:  append([]string(nil), discovery.RequiredOutputs...),
		CandidateOutputs: append([]string(nil), discovery.CandidateOutputs...),
		PackFile:         packRelative, PackSHA256: hex.EncodeToString(packDigest.Sum(nil)), PackSize: byteCount,
		Entries: entries,
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
	manifestMS := elapsedOptimizeEconomicsMS(manifestStarted)
	return optimizeOutputMaterialization{
		Status: optimizeMaterializationCaptured, Reason: optimizeMaterializationReasonReady,
		Patterns: patterns, ManifestFile: manifestRelative,
		ManifestSHA256: hex.EncodeToString(digest[:]), FileCount: len(entries), ByteCount: byteCount,
		Economics: optimizeMaterializationEconomics{CollectMS: collectMS, PackMS: packMS,
			ManifestMS: manifestMS, TotalMS: elapsedOptimizeEconomicsMS(totalStarted)},
	}, nil
}

func (run *optimizeRun) materializeCandidateOutputs() error {
	started := time.Now()
	defer func() { run.materializationTime += time.Since(started) }()
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
	directories := map[string]bool{}
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
		if err := writeOptimizeMaterializationFile(destination, payload.raw, fs.FileMode(payload.entry.Mode)); err != nil {
			return fmt.Errorf("materialize required output %s: %w", payload.entry.Path, err)
		}
		directories[filepath.Dir(destination)] = true
	}
	sortedDirectories := make([]string, 0, len(directories))
	for directory := range directories {
		sortedDirectories = append(sortedDirectories, directory)
	}
	sort.Strings(sortedDirectories)
	for _, directory := range sortedDirectories {
		if err := syncManagedDirectory(directory); err != nil {
			return fmt.Errorf("sync materialized output directory: %w", err)
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
	if manifest.PackFile != filepath.ToSlash(filepath.Join(invocation.stateRelative, "materialization", optimizeMaterializationPackName)) ||
		!validOptimizeSHA(manifest.PackSHA256) || manifest.PackSize != materialization.ByteCount {
		return optimizeOutputMaterializationManifest{}, nil, errors.New("output materialization pack binding drifted")
	}
	pack, err := os.Open(filepath.Join(invocation.repositoryRoot, filepath.FromSlash(manifest.PackFile)))
	if err != nil {
		return optimizeOutputMaterializationManifest{}, nil, errors.New("output materialization pack is unavailable")
	}
	defer pack.Close()
	packInfo, err := pack.Stat()
	if err != nil || !packInfo.Mode().IsRegular() || packInfo.Mode()&os.ModeSymlink != 0 || packInfo.Size() != manifest.PackSize {
		return optimizeOutputMaterializationManifest{}, nil, errors.New("output materialization pack is invalid")
	}
	packDigest := sha256.New()
	previous := ""
	var byteCount int64
	for _, entry := range manifest.Entries {
		if entry.Path <= previous || !validObservedOutputPath(entry.Path) || !validOptimizeSHA(entry.SHA256) ||
			entry.Size < 0 || entry.Mode > 0o777 || entry.Offset != byteCount {
			return optimizeOutputMaterializationManifest{}, nil, errors.New("output materialization entry is invalid")
		}
		previous = entry.Path
		if entry.Size > int64(int(entry.Size)) {
			return optimizeOutputMaterializationManifest{}, nil, errors.New("output materialization entry is too large")
		}
		raw := make([]byte, int(entry.Size))
		if _, err := io.ReadFull(pack, raw); err != nil {
			return optimizeOutputMaterializationManifest{}, nil, fmt.Errorf("materialization payload for %s is unavailable", entry.Path)
		}
		_, _ = packDigest.Write(raw)
		entryDigest := sha256.Sum256(raw)
		if hex.EncodeToString(entryDigest[:]) != entry.SHA256 {
			return optimizeOutputMaterializationManifest{}, nil, fmt.Errorf("materialization payload for %s is corrupt", entry.Path)
		}
		byteCount += entry.Size
		if byteCount > optimizeMaterializationMaxBytes {
			return optimizeOutputMaterializationManifest{}, nil, errors.New("output materialization exceeds the POC byte bound")
		}
		payloads = append(payloads, optimizeMaterializationPayload{entry: entry, raw: raw})
	}
	if byteCount != materialization.ByteCount || byteCount != manifest.PackSize ||
		hex.EncodeToString(packDigest.Sum(nil)) != manifest.PackSHA256 {
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
			materialization.ByteCount == 0 && materialization.Economics == (optimizeMaterializationEconomics{})
	}
	if materialization.Status == optimizeMaterializationNotRequired {
		return materialization.Reason == optimizeMaterializationReasonNone && len(materialization.Patterns) == 0 &&
			materialization.ManifestFile == "" && materialization.ManifestSHA256 == "" &&
			materialization.FileCount == 0 && materialization.ByteCount == 0 &&
			materialization.Economics == (optimizeMaterializationEconomics{})
	}
	if materialization.Status != optimizeMaterializationCaptured || materialization.Reason != optimizeMaterializationReasonReady ||
		len(materialization.Patterns) == 0 || len(materialization.Patterns) > 256 ||
		!uniqueMeasurementStrings(materialization.Patterns) || !validOptimizeGeneratedPath(materialization.ManifestFile) ||
		!validOptimizeSHA(materialization.ManifestSHA256) || materialization.FileCount < 1 ||
		materialization.FileCount > optimizeMaterializationMaxFiles || materialization.ByteCount < 0 ||
		materialization.ByteCount > optimizeMaterializationMaxBytes {
		return false
	}
	if materialization.Economics.TotalMS < 1 || materialization.Economics.CollectMS < 0 ||
		materialization.Economics.PackMS < 0 || materialization.Economics.ManifestMS < 0 ||
		materialization.Economics.CollectMS+materialization.Economics.PackMS+materialization.Economics.ManifestMS >
			materialization.Economics.TotalMS+3 {
		return false
	}
	return true
}

func elapsedOptimizeEconomicsMS(started time.Time) int64 {
	elapsed := time.Since(started).Milliseconds()
	if elapsed < 1 {
		return 1
	}
	return elapsed
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

// writeOptimizeMaterializationFile atomically publishes one verified payload.
// Bulk capture and restore synchronize each affected directory once after all
// renames; syncing every small file made clean-workspace materialization scale
// with filesystem barrier latency rather than bytes.
func writeOptimizeMaterializationFile(path string, raw []byte, mode fs.FileMode) error {
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
	if err := temporary.Close(); err != nil {
		return err
	}
	return replaceManagedFile(temporaryPath, path)
}
