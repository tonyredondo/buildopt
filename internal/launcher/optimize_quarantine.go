package launcher

import (
	"bytes"
	"crypto/sha256"
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

	"github.com/tonyredondo/buildopt/internal/nativevolatility"
)

const (
	profileNativeObserveUsage = "usage: buildopt profile native-observe --state-dir PATH --root DIR\n"
	profileQuarantineUsage    = "usage: buildopt profile quarantine --state-dir PATH --second-observation FILE\n"
	optimizeQuarantineReason  = "NATIVE_VOLATILE_PRODUCERS_QUARANTINED"
)

func runOptimizeNativeObservation(args []string, stdout, stderr io.Writer) int {
	if isHelp(args) {
		_, _ = io.WriteString(stdout, profileNativeObserveUsage)
		return 0
	}
	flags := flag.NewFlagSet("buildopt profile native-observe", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	stateRelative := flags.String("state-dir", optimizeDefaultStateDir, "repository-local optimize state")
	root := flags.String("root", "", "independent native workspace")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 || *root == "" {
		_, _ = io.WriteString(stderr, profileNativeObserveUsage)
		return exitUsage
	}
	invocation, state, err := loadOptimizeQuarantineState(*stateRelative)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: native output observation unavailable: %v\n", err)
		return exitConfiguration
	}
	manifest, _, err := loadOptimizeOutputMaterialization(invocation, state.Discovery)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: native output observation unavailable: %v\n", err)
		return exitConfiguration
	}
	inventory, err := optimizeNativeInventory(manifest)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: native output observation unavailable: %v\n", err)
		return exitConfiguration
	}
	observation, err := nativevolatility.Observe(*root, state.Bindings.SHA256, inventory)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: native output observation unavailable: %v\n", err)
		return exitConfiguration
	}
	if err := encodeOptimizeJSON(stdout, observation); err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: write native output observation: %v\n", err)
		return exitConfiguration
	}
	return 0
}

func runOptimizeNativeQuarantine(args []string, stdout, stderr io.Writer) int {
	if isHelp(args) {
		_, _ = io.WriteString(stdout, profileQuarantineUsage)
		return 0
	}
	flags := flag.NewFlagSet("buildopt profile quarantine", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	stateRelative := flags.String("state-dir", optimizeDefaultStateDir, "repository-local optimize state")
	secondPath := flags.String("second-observation", "", "independent native output observation")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 || *secondPath == "" {
		_, _ = io.WriteString(stderr, profileQuarantineUsage)
		return exitUsage
	}
	invocation, state, err := loadOptimizeQuarantineState(*stateRelative)
	if err == nil {
		var result nativevolatility.Result
		result, err = applyOptimizeNativeQuarantine(invocation, &state, *secondPath)
		if err == nil {
			err = encodeOptimizeJSON(stdout, result)
		}
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: native output quarantine unavailable: %v\n", err)
		return exitConfiguration
	}
	return 0
}

func loadOptimizeQuarantineState(relative string) (optimizeInvocation, optimizeState, error) {
	repositoryRoot, err := canonicalWorkingDirectory()
	if err != nil {
		return optimizeInvocation{}, optimizeState{}, err
	}
	stateDirectory, normalized, err := resolveOptimizeStateDirectory(repositoryRoot, relative, false)
	if err != nil {
		return optimizeInvocation{}, optimizeState{}, err
	}
	raw, err := os.ReadFile(filepath.Join(stateDirectory, optimizeStateFile))
	if err != nil || len(raw) == 0 || len(raw) > optimizeMaximumStateBytes {
		return optimizeInvocation{}, optimizeState{}, errors.New("optimize state is unavailable")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var state optimizeState
	if err := decoder.Decode(&state); err != nil {
		return optimizeInvocation{}, optimizeState{}, errors.New("decode optimize state")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) || !validOptimizeState(state) {
		return optimizeInvocation{}, optimizeState{}, errors.New("optimize state is invalid")
	}
	if state.Discovery.Status != optimizeDiscoveryComplete ||
		state.Discovery.Materialization.Status != optimizeMaterializationCaptured {
		return optimizeInvocation{}, optimizeState{}, errors.New("complete captured materialization is required")
	}
	return optimizeInvocation{
		repositoryRoot: repositoryRoot, stateDirectory: stateDirectory, stateRelative: normalized,
		bindingSHA256: state.Bindings.SHA256, calibrationPairs: state.Budget.Pairs,
		maxBreakEvenBuilds: state.Budget.MaxBreakEvenBuilds,
		calibrationBudget:  time.Duration(state.Budget.WallTimeSeconds) * time.Second,
	}, state, nil
}

func optimizeNativeInventory(manifest optimizeOutputMaterializationManifest) ([]nativevolatility.Entry, error) {
	inventory := make([]nativevolatility.Entry, 0, len(manifest.Entries))
	for _, entry := range manifest.Entries {
		if len(entry.ProducerTasks) == 0 {
			return nil, fmt.Errorf("materialized output %s has no producer attribution", entry.Path)
		}
		inventory = append(inventory, nativevolatility.Entry{
			Path: entry.Path, SHA256: entry.SHA256,
			ProducerTasks: append([]string(nil), entry.ProducerTasks...),
		})
	}
	return inventory, nil
}

func applyOptimizeNativeQuarantine(
	invocation optimizeInvocation,
	state *optimizeState,
	secondPath string,
) (nativevolatility.Result, error) {
	manifest, payloads, err := loadOptimizeOutputMaterialization(invocation, state.Discovery)
	if err != nil {
		return nativevolatility.Result{}, err
	}
	inventory, err := optimizeNativeInventory(manifest)
	if err != nil {
		return nativevolatility.Result{}, err
	}
	second, err := readOptimizeNativeObservation(secondPath)
	if err != nil {
		return nativevolatility.Result{}, err
	}
	first := nativevolatility.Observation{
		SchemaVersion: nativevolatility.ObservationSchema,
		BindingSHA256: state.Bindings.SHA256,
		Entries:       inventory,
	}
	result := nativevolatility.Analyze(first, second)
	if result.Decision != nativevolatility.DecisionTransportReady {
		return result, fmt.Errorf("native output evidence retained Gradle: %s", result.Reason)
	}
	if len(result.QuarantinedProducers) == 0 {
		return result, errors.New("native outputs are exact; quarantine is unnecessary")
	}
	if len(result.TransportedOutputs) == 0 {
		return result, errors.New("quarantine leaves no transportable outputs")
	}

	stable := make(map[string]nativevolatility.Entry, len(result.TransportedOutputs))
	for _, entry := range result.TransportedOutputs {
		stable[entry.Path] = entry
	}
	filtered := make([]optimizeMaterializationPayload, 0, len(stable))
	for _, payload := range payloads {
		entry, ok := stable[payload.entry.Path]
		if !ok {
			continue
		}
		if entry.SHA256 != payload.entry.SHA256 ||
			!equalOptimizeStrings(entry.ProducerTasks, payload.entry.ProducerTasks) {
			return result, fmt.Errorf("transported output %s drifted before quarantine", entry.Path)
		}
		filtered = append(filtered, payload)
	}
	if len(filtered) != len(stable) {
		return result, errors.New("transported output set is incomplete")
	}

	discovery := state.Discovery
	discovery.CandidateEntrypoints = mergeOptimizeStrings(
		discovery.CandidateEntrypoints, result.QuarantinedProducers,
	)
	if len(discovery.CandidateEntrypoints) > maximumStructuralAlternativeEntrypoints {
		return result, errors.New("quarantine candidate task set exceeds the POC bound")
	}
	quarantinedPaths := make([]string, 0, len(result.QuarantinedOutputs))
	for _, entry := range result.QuarantinedOutputs {
		quarantinedPaths = append(quarantinedPaths, entry.Path)
	}
	discovery.CandidateOutputs = mergeOptimizeStrings(discovery.CandidateOutputs, quarantinedPaths)
	if len(discovery.CandidateOutputs) > optimizeMaterializationMaxFiles {
		return result, errors.New("quarantine candidate output set exceeds the POC bound")
	}
	updateOptimizeQuarantinePartition(&discovery, result.QuarantinedProducers)

	quarantineDirectory := filepath.Join(invocation.stateDirectory, "materialization", "quarantine")
	if err := os.MkdirAll(quarantineDirectory, 0o700); err != nil {
		return result, fmt.Errorf("create quarantine materialization directory: %w", err)
	}
	manifest, materialization, err := writeOptimizeQuarantineMaterialization(
		invocation, discovery, manifest, filtered, quarantineDirectory,
	)
	if err != nil {
		return result, err
	}
	_ = manifest
	resultRelative := filepath.ToSlash(filepath.Join(
		invocation.stateRelative, "materialization", "quarantine", "native-volatility.json",
	))
	resultAbsolute := filepath.Join(invocation.repositoryRoot, filepath.FromSlash(resultRelative))
	if err := writeCanonicalPrivateJSON(resultAbsolute, result); err != nil {
		return result, fmt.Errorf("write native volatility result: %w", err)
	}
	resultRaw, err := os.ReadFile(resultAbsolute)
	if err != nil {
		return result, err
	}
	resultDigest := sha256.Sum256(resultRaw)
	materialization.QuarantineFile = resultRelative
	materialization.QuarantineSHA256 = hex.EncodeToString(resultDigest[:])
	discovery.Materialization = materialization

	state.Phase = "DISCOVERED"
	state.LastOutcome = optimizeOutcomeLearning
	state.LastReason = optimizeQuarantineReason
	state.Bindings.Completeness = optimizeBindingDiscovery
	state.Discovery = discovery
	state.IncrementalLearning = emptyOptimizeIncrementalLearning()
	state.Calibration = emptyOptimizeCalibration(invocation, optimizeQuarantineReason)
	state.Portfolio = emptyOptimizePortfolio(optimizeQuarantineReason)
	state.Selection = emptyOptimizeSelection(optimizeSelectionSkipped, optimizeSelectionReasonNone, false)
	state.Selection.DurationNS = 1
	state.Value = optimizeValueState{}
	state.LastExitCode = 0
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if !validOptimizeState(*state) {
		return result, errors.New("quarantined optimize state is invalid")
	}
	if err := writeCanonicalPrivateJSON(filepath.Join(invocation.stateDirectory, optimizeStateFile), state); err != nil {
		return result, fmt.Errorf("write quarantined optimize state: %w", err)
	}
	return result, nil
}

func writeOptimizeQuarantineMaterialization(
	invocation optimizeInvocation,
	discovery optimizeDiscoveryResult,
	manifest optimizeOutputMaterializationManifest,
	payloads []optimizeMaterializationPayload,
	directory string,
) (optimizeOutputMaterializationManifest, optimizeOutputMaterialization, error) {
	packRelative := filepath.ToSlash(filepath.Join(
		invocation.stateRelative, "materialization", "quarantine", optimizeMaterializationPackName,
	))
	packAbsolute := filepath.Join(invocation.repositoryRoot, filepath.FromSlash(packRelative))
	pack, err := os.CreateTemp(directory, ".buildopt-quarantine-pack-*")
	if err != nil {
		return manifest, optimizeOutputMaterialization{}, err
	}
	temporary := pack.Name()
	defer os.Remove(temporary)
	if err := pack.Chmod(0o600); err != nil {
		_ = pack.Close()
		return manifest, optimizeOutputMaterialization{}, err
	}
	packDigest := sha256.New()
	var offset int64
	entries := make([]optimizeOutputMaterializationEntry, 0, len(payloads))
	for _, payload := range payloads {
		entry := payload.entry
		entry.Offset = offset
		if _, err := io.Copy(io.MultiWriter(pack, packDigest), bytes.NewReader(payload.raw)); err != nil {
			_ = pack.Close()
			return manifest, optimizeOutputMaterialization{}, err
		}
		offset += entry.Size
		entries = append(entries, entry)
	}
	if err := pack.Sync(); err != nil {
		_ = pack.Close()
		return manifest, optimizeOutputMaterialization{}, err
	}
	if err := pack.Close(); err != nil {
		return manifest, optimizeOutputMaterialization{}, err
	}
	if err := replaceManagedFile(temporary, packAbsolute); err != nil {
		return manifest, optimizeOutputMaterialization{}, err
	}

	manifest.RequiredOutputs = append([]string(nil), discovery.RequiredOutputs...)
	manifest.CandidateOutputs = append([]string(nil), discovery.CandidateOutputs...)
	manifest.PackFile = packRelative
	manifest.PackSHA256 = hex.EncodeToString(packDigest.Sum(nil))
	manifest.PackSize = offset
	manifest.Entries = entries
	manifestRelative := filepath.ToSlash(filepath.Join(
		invocation.stateRelative, "materialization", "quarantine", "manifest.json",
	))
	manifestAbsolute := filepath.Join(invocation.repositoryRoot, filepath.FromSlash(manifestRelative))
	if err := writeCanonicalPrivateJSON(manifestAbsolute, manifest); err != nil {
		return manifest, optimizeOutputMaterialization{}, err
	}
	manifestRaw, err := os.ReadFile(manifestAbsolute)
	if err != nil {
		return manifest, optimizeOutputMaterialization{}, err
	}
	manifestDigest := sha256.Sum256(manifestRaw)
	materialization := discovery.Materialization
	materialization.ManifestFile = manifestRelative
	materialization.ManifestSHA256 = hex.EncodeToString(manifestDigest[:])
	materialization.FileCount = len(entries)
	materialization.ByteCount = offset
	return manifest, materialization, nil
}

func updateOptimizeQuarantinePartition(discovery *optimizeDiscoveryResult, producers []string) {
	if discovery.AggregatePartition == nil {
		return
	}
	projects := make([]string, 0, len(producers))
	for _, producer := range producers {
		project := optimizeTaskProject(producer)
		if project != "" {
			projects = append(projects, project)
		}
	}
	projects = mergeOptimizeStrings(nil, projects)
	discovery.AggregatePartition.RebuildProjects = mergeOptimizeStrings(
		discovery.AggregatePartition.RebuildProjects, projects,
	)
	discovery.AggregatePartition.MaterializedProjects = subtractOptimizeStrings(
		discovery.AggregatePartition.MaterializedProjects, projects,
	)
	discovery.AggregatePartition.CandidateEntrypointCount = len(discovery.CandidateEntrypoints)
	discovery.AggregatePartition.CandidateOutputCount = len(discovery.CandidateOutputs)
	discovery.Graph.SelectedProjects = len(mergeOptimizeStrings(
		discovery.ChangedProjects, discovery.AggregatePartition.RebuildProjects,
	))
	discovery.Graph.OmittedProjects = discovery.Graph.TotalProjects - discovery.Graph.SelectedProjects
}

func optimizeTaskProject(task string) string {
	if !strings.HasPrefix(task, ":") {
		return ""
	}
	last := strings.LastIndex(task, ":")
	if last == 0 {
		return ":"
	}
	return task[:last]
}

func mergeOptimizeStrings(left, right []string) []string {
	seen := make(map[string]bool, len(left)+len(right))
	for _, value := range append(append([]string(nil), left...), right...) {
		if value != "" {
			seen[value] = true
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func subtractOptimizeStrings(values, removed []string) []string {
	set := make(map[string]bool, len(removed))
	for _, value := range removed {
		set[value] = true
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if !set[value] {
			result = append(result, value)
		}
	}
	return result
}

func readOptimizeNativeObservation(path string) (nativevolatility.Observation, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nativevolatility.Observation{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var observation nativevolatility.Observation
	if err := decoder.Decode(&observation); err != nil {
		return nativevolatility.Observation{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nativevolatility.Observation{}, errors.New("native observation has trailing JSON")
	}
	return observation, nil
}

func loadOptimizeNativeQuarantine(
	invocation optimizeInvocation,
	discovery optimizeDiscoveryResult,
) (nativevolatility.Result, error) {
	materialization := discovery.Materialization
	if materialization.QuarantineFile == "" || materialization.QuarantineSHA256 == "" {
		return nativevolatility.Result{}, errors.New("native volatility quarantine is unavailable")
	}
	raw, err := os.ReadFile(filepath.Join(
		invocation.repositoryRoot, filepath.FromSlash(materialization.QuarantineFile),
	))
	if err != nil || len(raw) == 0 || len(raw) > optimizeMaterializationMaxManifest {
		return nativevolatility.Result{}, errors.New("native volatility quarantine file is unavailable")
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != materialization.QuarantineSHA256 {
		return nativevolatility.Result{}, errors.New("native volatility quarantine digest drifted")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var result nativevolatility.Result
	if err := decoder.Decode(&result); err != nil {
		return nativevolatility.Result{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nativevolatility.Result{}, errors.New("native volatility quarantine has trailing JSON")
	}
	if err := nativevolatility.ValidateResult(result); err != nil {
		return nativevolatility.Result{}, err
	}
	if result.BindingSHA256 != invocation.bindingSHA256 ||
		!equalOptimizeStrings(result.QuarantinedProducers,
			intersectOptimizeStrings(discovery.CandidateEntrypoints, result.QuarantinedProducers)) {
		return nativevolatility.Result{}, errors.New("native volatility quarantine binding drifted")
	}
	return result, nil
}

func intersectOptimizeStrings(values, selected []string) []string {
	set := make(map[string]bool, len(selected))
	for _, value := range selected {
		set[value] = true
	}
	result := make([]string, 0, len(selected))
	for _, value := range values {
		if set[value] {
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func encodeOptimizeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
