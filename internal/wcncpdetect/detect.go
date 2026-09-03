// Package wcncpdetect owns WCNCP-004 aggregation and detection as pure
// deterministic functions over typed facts. No repository, task, project, or
// checkout-root name may influence classification; names are report labels
// only. The independent checker reconstructs every row from raw observations.
package wcncpdetect

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
)

var (
	// ErrIncomplete means required facts are missing; the row cannot open
	// validation and must request diagnostics instead of guessing.
	ErrIncomplete = errors.New("BuildOpt WCNCP detector input is incomplete")
)

// CompatibilityKey is the exact identity under which observations may be
// combined. Incompatible observations remain separate facts; the aggregator
// never averages across workflows, revisions with relevant drift, warm/cold
// states, cache policies, or resource classes.
type CompatibilityKey struct {
	RepositoryScope      string
	SourceTreeSHA256     string
	WrapperSHA256        string
	GradleMajor          string
	JDKMajor             string
	BuildOptPackageSHA256 string
	WorkflowSHA256       string
	BuildCacheMode       string
	ConfigurationCache   string
	EnvironmentSHA256    string
	OutputContractSHA256 string
}

// ObservationSummary is the minimal typed fact set the aggregator consumes.
// It carries source/runtime facts only, never repository or task names as
// classification inputs.
type ObservationSummary struct {
	ObservationID      string
	RepositoryScope    string
	SourceTreeSHA256   string
	WrapperSHA256      string
	GradleVersion      string
	JDKSHA256          string
	PackageSHA256      string
	WorkflowSHA256     string
	BuildCacheMode     string
	ConfigurationCache string
	EnvironmentSHA256  string
	EnvironmentClass   string
	OutputContractSHA256 string
	CriticalPathMs     *int64
	WorkflowMs         *int64
}

// CompatibilityOf derives the grouping key. Gradle and JDK versions reduce to
// major bindings so patch upgrades do not fragment compatible evidence while
// major drift still splits groups.
func CompatibilityOf(observation ObservationSummary) CompatibilityKey {
	return CompatibilityKey{
		RepositoryScope:      observation.RepositoryScope,
		SourceTreeSHA256:     observation.SourceTreeSHA256,
		WrapperSHA256:        observation.WrapperSHA256,
		GradleMajor:          gradleMajor(observation.GradleVersion),
		JDKMajor:             jdkMajor(observation.JDKSHA256),
		BuildOptPackageSHA256: observation.PackageSHA256,
		WorkflowSHA256:       observation.WorkflowSHA256,
		BuildCacheMode:       observation.BuildCacheMode,
		ConfigurationCache:   observation.ConfigurationCache,
		EnvironmentSHA256:    observation.EnvironmentSHA256,
		OutputContractSHA256: observation.OutputContractSHA256,
	}
}

func gradleMajor(version string) string {
	if index := strings.Index(version, "."); index > 0 {
		return version[:index]
	}
	return version
}

func jdkMajor(digest string) string {
	// JDK identity binds by digest; major splitting happens at the environment
	// binding layer. The digest prefix keeps groups stable without parsing
	// vendor strings, and no repository name enters the key.
	if len(digest) >= 16 {
		return digest[:16]
	}
	return digest
}

// GroupCompatible buckets observation IDs by compatibility key with
// deduplication by canonical observation identity. Order is deterministic:
// keys sort lexicographically and IDs sort within each group.
func GroupCompatible(observations []ObservationSummary) map[CompatibilityKey][]string {
	groups := map[CompatibilityKey][]string{}
	seen := map[string]bool{}
	for _, observation := range observations {
		if observation.ObservationID == "" || seen[observation.ObservationID] {
			continue
		}
		seen[observation.ObservationID] = true
		key := CompatibilityOf(observation)
		groups[key] = append(groups[key], observation.ObservationID)
	}
	for key := range groups {
		sort.Strings(groups[key])
	}
	return groups
}

// DetectorRow is one repository/detector outcome. Only
// ACTIONABLE_MATERIAL_CORRECTION opens validation.
type DetectorRow struct {
	Decision            string
	DetectorID          string
	DetectorVersion     string
	SourcePath          string
	RecipeClass         string
	CriticalPathMs      int64
	WorkflowPercentMilli int64
	RequiredDiagnostics []string
	Reason              string
}

// DetectorInput carries source/runtime facts for one compatible group. Names
// are labels for reports; classification reads only the typed fields.
type DetectorInput struct {
	DetectorID         string
	DetectorVersion    string
	ProblemClass       string
	ProblemReproducible bool
	SourceOwned        bool
	SourcePath         string
	SourceReversible   bool
	ExternalPluginOwned bool
	GeneratedOrVendor  bool
	AbsolutePathDependent bool
	RequiresOwnerSemantics bool
	SuppressionStyle   bool
	DisablesConfigCache bool
	ConfigCacheEnabled bool
	SourceDrifted      bool
	BindingAmbiguous   bool
	WorkflowReachable  bool
	Repetition         int
	CriticalPathMs     int64
	WorkflowMs         int64
	EnvironmentClass   string
	HasGradleProblemData bool
}

// ConfigurationCacheReadinessV1 proposes only a small repository-owned
// versioned native correction for a reproducible Configuration Cache blocker.
// It never silences a problem, adds broad opt-outs, disables the cache,
// ignores failures, or infers semantics from names.
func ConfigurationCacheReadinessV1(input DetectorInput) DetectorRow {
	base := DetectorRow{DetectorID: "CONFIGURATION_CACHE_READINESS_PATCH", DetectorVersion: "v1"}
	if !input.HasGradleProblemData {
		base.Decision = "INCOMPLETE_OBSERVATION"
		base.Reason = "missing machine-bindable Gradle problem data"
		base.RequiredDiagnostics = []string{"gradle-configuration-cache-problem-report"}
		return base
	}
	if input.BindingAmbiguous || !input.ProblemReproducible {
		base.Decision = "SOURCE_OR_BINDING_AMBIGUOUS"
		if !input.ProblemReproducible {
			base.Decision = "NO_REPRODUCIBLE_BLOCKER"
		}
		base.Reason = "unambiguous source binding and repetition required"
		return base
	}
	if input.SourceDrifted {
		base.Decision = "SOURCE_DRIFTED"
		base.Reason = "source postimage no longer matches preimage"
		return base
	}
	if input.ExternalPluginOwned || input.GeneratedOrVendor {
		base.Decision = "UNSUPPORTED_PROBLEM_CLASS"
		base.Reason = "correction must be repository-owned source"
		return base
	}
	if input.AbsolutePathDependent {
		base.Decision = "UNSAFE_OR_NON_REVERSIBLE"
		base.Reason = "absolute-path dependent edit is not portable"
		return base
	}
	if !input.SourceOwned || !input.SourceReversible {
		base.Decision = "UNSAFE_OR_NON_REVERSIBLE"
		base.Reason = "exact inverse required"
		return base
	}
	if input.RequiresOwnerSemantics {
		base.Decision = "OWNER_SEMANTICS_REQUIRED"
		base.Reason = "semantics need an owner answer; no candidate may compile or time"
		return base
	}
	if input.SuppressionStyle || input.DisablesConfigCache {
		base.Decision = "UNSUPPORTED_PROBLEM_CLASS"
		base.Reason = "suppression and opt-outs are not corrections"
		return base
	}
	if !input.ConfigCacheEnabled {
		base.Decision = "UNSUPPORTED_PROBLEM_CLASS"
		base.Reason = "configuration cache not enabled for this workflow"
		return base
	}
	if !input.WorkflowReachable {
		base.Decision = "NO_REPRODUCIBLE_BLOCKER"
		base.Reason = "blocker outside the requested workflow"
		return base
	}
	materiality, _, _ := Materiality(input.EnvironmentClass, input.CriticalPathMs, input.WorkflowMs)
	if materiality == "REQUIRES_CONTROLLED_MEASUREMENT" {
		base.Decision = "INCOMPLETE_OBSERVATION"
		base.Reason = "materiality requires controlled measurement"
		base.RequiredDiagnostics = []string{"controlled-critical-path-measurement"}
		return base
	}
	if materiality == "FAILED" {
		base.Decision = "NON_MATERIAL_BLOCKER"
		base.CriticalPathMs = input.CriticalPathMs
		base.Reason = "below 500 ms and 2 percent materiality"
		return base
	}
	base.Decision = "ACTIONABLE_MATERIAL_CORRECTION"
	base.SourcePath = input.SourcePath
	base.RecipeClass = "CONFIGURATION_CACHE_READINESS_PATCH_V1"
	base.CriticalPathMs = input.CriticalPathMs
	if input.WorkflowMs > 0 {
		base.WorkflowPercentMilli = input.CriticalPathMs * 100000 / input.WorkflowMs
	}
	return base
}

// Materiality reports whether a blocker contributes at least 500 ms and 2
// percent of optimized-native workflow wall time. Only controlled
// observations decide; hosted traces yield
// MATERIALITY_REQUIRES_CONTROLLED_MEASUREMENT and never admit or reject.
func Materiality(environmentClass string, criticalPathMs, workflowMs int64) (string, int64, int64) {
	if environmentClass != "CONTROLLED_PERFORMANCE" {
		return "REQUIRES_CONTROLLED_MEASUREMENT", 0, 0
	}
	if workflowMs <= 0 || criticalPathMs < 500 {
		return "FAILED", criticalPathMs, 0
	}
	percentMilli := criticalPathMs * 100000 / workflowMs
	if percentMilli < 2000 {
		return "FAILED", criticalPathMs, percentMilli
	}
	return "PASSED", criticalPathMs, percentMilli
}

// GroupDigest renders one compatibility group deterministically for
// independent reconstruction.
func GroupDigest(key CompatibilityKey, observationIDs []string) string {
	parts := append([]string{
		key.RepositoryScope, key.SourceTreeSHA256, key.WrapperSHA256, key.GradleMajor,
		key.JDKMajor, key.BuildOptPackageSHA256, key.WorkflowSHA256, key.BuildCacheMode,
		key.ConfigurationCache, key.EnvironmentSHA256, key.OutputContractSHA256,
	}, observationIDs...)
	digest := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(digest[:])
}
