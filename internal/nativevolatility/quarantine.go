// Package nativevolatility identifies native outputs that are unsafe to move
// between workspaces and derives a producer-atomic transport boundary.
package nativevolatility

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const (
	ObservationSchema = "buildopt.poc/native-output-observation/v1"
	ResultSchema      = "buildopt.poc/native-volatility-quarantine/v1"

	DecisionTransportReady = "TRANSPORT_READY"
	DecisionNativeRetained = "NATIVE_RETAINED"

	ReasonExact             = "INDEPENDENT_NATIVE_OUTPUTS_EXACT"
	ReasonQuarantined       = "VOLATILE_PRODUCERS_QUARANTINED"
	ReasonInvalidEvidence   = "NATIVE_OUTPUT_EVIDENCE_INVALID"
	ReasonBindingMismatch   = "NATIVE_OUTPUT_BINDING_MISMATCH"
	ReasonPathMismatch      = "NATIVE_OUTPUT_PATH_SET_MISMATCH"
	ReasonProducerAmbiguous = "NATIVE_OUTPUT_PRODUCER_AMBIGUOUS"
)

const maximumOutputs = 250000
const maximumObservedBytes = int64(2 << 30)

// Entry binds one regular output to its exact bytes and Gradle producer tasks.
// Multiple producers are accepted, but volatility in the entry quarantines all
// of them and every other output produced by any of those tasks.
type Entry struct {
	Path          string   `json:"path"`
	SHA256        string   `json:"sha256"`
	ProducerTasks []string `json:"producerTasks"`
}

// Observation is the exact output view from one independent native workspace.
type Observation struct {
	SchemaVersion string  `json:"schemaVersion"`
	BindingSHA256 string  `json:"bindingSha256"`
	Entries       []Entry `json:"entries"`
}

// Observe hashes a complete producer-bound output inventory in one native
// workspace. Every path component must remain a regular, non-symlink path
// below root; callers cannot turn a captured relative path into host access.
func Observe(root, binding string, inventory []Entry) (Observation, error) {
	if !validSHA(binding) || len(inventory) == 0 || len(inventory) > maximumOutputs {
		return Observation{}, errors.New("invalid native output observation request")
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return Observation{}, errors.New("resolve native output root")
	}
	rootInfo, err := os.Lstat(absoluteRoot)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return Observation{}, errors.New("native output root is unavailable")
	}
	entries := make([]Entry, 0, len(inventory))
	var observedBytes int64
	for _, source := range inventory {
		entry := cloneEntry(source)
		entry.SHA256 = strings.Repeat("0", sha256.Size*2)
		if _, _, validateErr := validateObservation(Observation{
			SchemaVersion: ObservationSchema, BindingSHA256: binding, Entries: []Entry{entry},
		}, false); validateErr != nil {
			return Observation{}, validateErr
		}
		absolute, pathErr := secureObservationPath(absoluteRoot, entry.Path)
		if pathErr != nil {
			return Observation{}, pathErr
		}
		raw, readErr := os.ReadFile(absolute)
		if readErr != nil {
			return Observation{}, fmt.Errorf("read native output %s", entry.Path)
		}
		observedBytes += int64(len(raw))
		if observedBytes > maximumObservedBytes {
			return Observation{}, errors.New("native output observation exceeds the byte bound")
		}
		digest := sha256.Sum256(raw)
		entry.SHA256 = hex.EncodeToString(digest[:])
		entries = append(entries, entry)
	}
	observation := Observation{SchemaVersion: ObservationSchema, BindingSHA256: binding, Entries: entries}
	if _, _, err := validateObservation(observation, false); err != nil {
		return Observation{}, err
	}
	return observation, nil
}

func secureObservationPath(root, relative string) (string, error) {
	if !validRelativePath(relative) {
		return "", errors.New("invalid native output path")
	}
	current := root
	parts := strings.Split(relative, "/")
	for index, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil || info.Mode()&os.ModeSymlink != 0 ||
			(index < len(parts)-1 && !info.IsDir()) ||
			(index == len(parts)-1 && !info.Mode().IsRegular()) {
			return "", fmt.Errorf("native output %s is not a regular in-root file", relative)
		}
	}
	return current, nil
}

// Result is a producer-atomic plan. Transported entries retain the first
// observation's exact digest; quarantined entries must be rebuilt locally.
type Result struct {
	SchemaVersion        string   `json:"schemaVersion"`
	Decision             string   `json:"decision"`
	Reason               string   `json:"reason"`
	BindingSHA256        string   `json:"bindingSha256"`
	FirstObservationSHA  string   `json:"firstObservationSha256"`
	SecondObservationSHA string   `json:"secondObservationSha256"`
	ComparedOutputCount  int      `json:"comparedOutputCount"`
	VolatilePaths        []string `json:"volatilePaths"`
	QuarantinedProducers []string `json:"quarantinedProducers"`
	QuarantinedOutputs   []Entry  `json:"quarantinedOutputs"`
	TransportedOutputs   []Entry  `json:"transportedOutputs"`
	TransportSHA256      string   `json:"transportSha256"`
	ProductionAuthorized bool     `json:"productionAuthorized"`
	TestOptimization     string   `json:"testOptimization"`
}

// Analyze compares two independent native observations. It fails closed when
// their bindings, path universe or producer attribution cannot be proven equal.
func Analyze(first, second Observation) Result {
	result := Result{
		SchemaVersion:        ResultSchema,
		Decision:             DecisionNativeRetained,
		Reason:               ReasonInvalidEvidence,
		VolatilePaths:        []string{},
		QuarantinedProducers: []string{},
		QuarantinedOutputs:   []Entry{},
		TransportedOutputs:   []Entry{},
		ProductionAuthorized: false,
		TestOptimization:     "OUT_OF_SCOPE",
	}
	firstEntries, firstDigest, err := validateObservation(first, false)
	if err != nil {
		return result
	}
	secondEntries, secondDigest, err := validateObservation(second, false)
	if err != nil {
		return result
	}
	result.BindingSHA256 = first.BindingSHA256
	result.FirstObservationSHA = firstDigest
	result.SecondObservationSHA = secondDigest
	if first.BindingSHA256 != second.BindingSHA256 {
		result.Reason = ReasonBindingMismatch
		return result
	}
	if len(firstEntries) != len(secondEntries) {
		result.Reason = ReasonPathMismatch
		return result
	}

	quarantined := map[string]bool{}
	for path, left := range firstEntries {
		right, ok := secondEntries[path]
		if !ok {
			result.Reason = ReasonPathMismatch
			return result
		}
		if !equalStrings(left.ProducerTasks, right.ProducerTasks) {
			result.Reason = ReasonProducerAmbiguous
			return result
		}
		if left.SHA256 != right.SHA256 {
			result.VolatilePaths = append(result.VolatilePaths, path)
			for _, producer := range left.ProducerTasks {
				quarantined[producer] = true
			}
		}
	}

	for producer := range quarantined {
		result.QuarantinedProducers = append(result.QuarantinedProducers, producer)
	}
	sort.Strings(result.VolatilePaths)
	sort.Strings(result.QuarantinedProducers)
	ordered := make([]Entry, 0, len(firstEntries))
	for _, entry := range firstEntries {
		ordered = append(ordered, cloneEntry(entry))
	}
	sortEntries(ordered)
	for _, entry := range ordered {
		if intersects(entry.ProducerTasks, quarantined) {
			result.QuarantinedOutputs = append(result.QuarantinedOutputs, cloneEntry(entry))
			continue
		}
		result.TransportedOutputs = append(result.TransportedOutputs, cloneEntry(entry))
	}
	sortEntries(result.QuarantinedOutputs)
	sortEntries(result.TransportedOutputs)
	result.ComparedOutputCount = len(firstEntries)
	result.TransportSHA256 = entriesDigest(result.TransportedOutputs)
	result.Decision = DecisionTransportReady
	result.Reason = ReasonExact
	if len(result.QuarantinedProducers) > 0 {
		result.Reason = ReasonQuarantined
	}
	return result
}

// VerifyCandidate proves that every transported output remains byte-exact and
// every quarantined producer output is supplied by the local native rebuild.
func VerifyCandidate(result Result, reused, rebuilt Observation) error {
	if err := ValidateResult(result); err != nil {
		return err
	}
	if reused.BindingSHA256 != result.BindingSHA256 || rebuilt.BindingSHA256 != result.BindingSHA256 {
		return errors.New("candidate output binding mismatch")
	}
	reusedEntries, _, err := validateObservation(reused, true)
	if err != nil {
		return fmt.Errorf("validate reused outputs: %w", err)
	}
	rebuiltEntries, _, err := validateObservation(rebuilt, true)
	if err != nil {
		return fmt.Errorf("validate rebuilt outputs: %w", err)
	}
	if err := verifyExactSet(result.TransportedOutputs, reusedEntries); err != nil {
		return fmt.Errorf("verify transported outputs: %w", err)
	}
	if err := verifyRebuiltSet(result.QuarantinedOutputs, rebuiltEntries, result.QuarantinedProducers); err != nil {
		return fmt.Errorf("verify rebuilt outputs: %w", err)
	}
	return nil
}

// ValidateResult checks the complete producer-atomic partition without
// requiring candidate observations. It is used when the optimize checkpoint
// consumes a previously produced quarantine decision.
func ValidateResult(result Result) error {
	if result.SchemaVersion != ResultSchema || result.Decision != DecisionTransportReady ||
		result.ProductionAuthorized || result.TestOptimization != "OUT_OF_SCOPE" ||
		!validSHA(result.BindingSHA256) || !validSHA(result.FirstObservationSHA) ||
		!validSHA(result.SecondObservationSHA) || result.ComparedOutputCount < 1 ||
		result.ComparedOutputCount > maximumOutputs || !uniqueNonEmpty(result.QuarantinedProducers) {
		return errors.New("native volatility result is not transport-ready POC evidence")
	}
	if result.Reason != ReasonExact && result.Reason != ReasonQuarantined {
		return errors.New("native volatility result reason is invalid")
	}
	quarantined, _, err := validateObservation(Observation{
		SchemaVersion: ObservationSchema, BindingSHA256: result.BindingSHA256,
		Entries: result.QuarantinedOutputs,
	}, true)
	if err != nil {
		return errors.New("native volatility quarantined output set is invalid")
	}
	transported, _, err := validateObservation(Observation{
		SchemaVersion: ObservationSchema, BindingSHA256: result.BindingSHA256,
		Entries: result.TransportedOutputs,
	}, true)
	if err != nil || len(quarantined)+len(transported) != result.ComparedOutputCount {
		return errors.New("native volatility output partition is incomplete")
	}
	producerSet := make(map[string]bool, len(result.QuarantinedProducers))
	for _, producer := range result.QuarantinedProducers {
		producerSet[producer] = true
	}
	for path, entry := range quarantined {
		if _, exists := transported[path]; exists || !intersects(entry.ProducerTasks, producerSet) {
			return errors.New("native volatility quarantine partition overlaps")
		}
	}
	for _, entry := range transported {
		if intersects(entry.ProducerTasks, producerSet) {
			return errors.New("native volatility transported output uses a quarantined producer")
		}
	}
	if result.TransportSHA256 != entriesDigest(result.TransportedOutputs) {
		return errors.New("native volatility transport plan was modified")
	}
	if len(result.QuarantinedProducers) == 0 {
		if result.Reason != ReasonExact || len(result.QuarantinedOutputs) != 0 || len(result.VolatilePaths) != 0 {
			return errors.New("native volatility exact partition is inconsistent")
		}
		return nil
	}
	if result.Reason != ReasonQuarantined || len(result.QuarantinedOutputs) == 0 ||
		len(result.VolatilePaths) == 0 || !uniqueNonEmpty(result.VolatilePaths) {
		return errors.New("native volatility quarantine metadata is incomplete")
	}
	for _, volatile := range result.VolatilePaths {
		if _, exists := quarantined[volatile]; !exists {
			return errors.New("native volatility path is outside the quarantine")
		}
	}
	return nil
}

func validateObservation(observation Observation, allowEmpty bool) (map[string]Entry, string, error) {
	if observation.SchemaVersion != ObservationSchema || !validSHA(observation.BindingSHA256) ||
		(!allowEmpty && len(observation.Entries) == 0) || len(observation.Entries) > maximumOutputs {
		return nil, "", errors.New("invalid native output observation")
	}
	entries := make(map[string]Entry, len(observation.Entries))
	ordered := make([]Entry, 0, len(observation.Entries))
	for _, source := range observation.Entries {
		entry := cloneEntry(source)
		rawPath := entry.Path
		entry.Path = path.Clean(entry.Path)
		sort.Strings(entry.ProducerTasks)
		if rawPath != entry.Path || !validRelativePath(entry.Path) || !validSHA(entry.SHA256) ||
			len(entry.ProducerTasks) == 0 || !uniqueNonEmpty(entry.ProducerTasks) {
			return nil, "", errors.New("invalid native output entry")
		}
		if _, exists := entries[entry.Path]; exists {
			return nil, "", errors.New("duplicate native output path")
		}
		entries[entry.Path] = entry
		ordered = append(ordered, entry)
	}
	sortEntries(ordered)
	return entries, entriesDigest(ordered), nil
}

func verifyExactSet(expected []Entry, actual map[string]Entry) error {
	if len(expected) != len(actual) {
		return errors.New("transported output set is incomplete")
	}
	for _, entry := range expected {
		observed, ok := actual[entry.Path]
		if !ok || observed.SHA256 != entry.SHA256 || !equalStrings(observed.ProducerTasks, entry.ProducerTasks) {
			return fmt.Errorf("transported output %s is not exact", entry.Path)
		}
	}
	return nil
}

func verifyRebuiltSet(expected []Entry, actual map[string]Entry, producers []string) error {
	if len(expected) != len(actual) {
		return errors.New("locally rebuilt output set is incomplete")
	}
	quarantined := make(map[string]bool, len(producers))
	for _, producer := range producers {
		quarantined[producer] = true
	}
	for _, entry := range expected {
		observed, ok := actual[entry.Path]
		if !ok || !equalStrings(observed.ProducerTasks, entry.ProducerTasks) ||
			!intersects(observed.ProducerTasks, quarantined) {
			return fmt.Errorf("quarantined output %s was not rebuilt by its producer", entry.Path)
		}
	}
	return nil
}

func entriesDigest(entries []Entry) string {
	hash := sha256.New()
	for _, entry := range entries {
		_, _ = hash.Write([]byte(entry.Path))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(entry.SHA256))
		for _, producer := range entry.ProducerTasks {
			_, _ = hash.Write([]byte{0})
			_, _ = hash.Write([]byte(producer))
		}
		_, _ = hash.Write([]byte{'\n'})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func validSHA(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && value == strings.ToLower(value)
}

func validRelativePath(path string) bool {
	return path != "" && path != "." && !strings.ContainsRune(path, '\\') && !strings.HasPrefix(path, "/") &&
		path != ".." && !strings.HasPrefix(path, "../") && !strings.ContainsRune(path, '\x00')
}

func uniqueNonEmpty(values []string) bool {
	for index, value := range values {
		if value == "" || (index > 0 && values[index-1] == value) {
			return false
		}
	}
	return true
}

func equalStrings(left, right []string) bool {
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

func intersects(values []string, set map[string]bool) bool {
	for _, value := range values {
		if set[value] {
			return true
		}
	}
	return false
}

func cloneEntry(entry Entry) Entry {
	entry.ProducerTasks = append([]string(nil), entry.ProducerTasks...)
	sort.Strings(entry.ProducerTasks)
	return entry
}

func sortEntries(entries []Entry) {
	sort.Slice(entries, func(left, right int) bool { return entries[left].Path < entries[right].Path })
}
