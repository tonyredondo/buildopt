package stickywrapper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/stickydecision"
	"github.com/tonyredondo/buildopt/internal/stickyobservation"
)

// StatusSchemaVersion identifies the read-only customer status surface. The
// report intentionally contains bounded facts and digests, never credentials
// or checkout paths.
const (
	StatusSchemaVersion          = "buildopt.sticky/status/v1"
	ordinaryObservationOutputEnv = "BUILDOPT_STICKY_OBSERVATION_OUTPUT"
	learningLifecycleOutputEnv   = "BUILDOPT_STICKY_LIFECYCLE_OUTPUT"
)

// Measurement is an explicitly available or unavailable value. A missing
// value is never represented as zero because zero can be a valid measurement.
type Measurement struct {
	State  string `json:"state"`
	Value  *int64 `json:"value,omitempty"`
	Unit   string `json:"unit,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// WrapperStatus describes the committed wrapper without exposing its endpoint
// or credential environment name.
type WrapperStatus struct {
	DistributionVersion string `json:"distributionVersion"`
	Mode                string `json:"mode"`
	ServerConfigured    bool   `json:"serverConfigured"`
	ProjectScopeBound   bool   `json:"projectScopeBound"`
}

// DecisionStatus describes what the next build may safely do. An unverified
// local decision is reported as unavailable and therefore retains native
// Gradle; status must not become an authorization path.
type DecisionStatus struct {
	State          string `json:"state"`
	StoredDecision string `json:"storedDecision,omitempty"`
	Reason         string `json:"reason"`
	Generation     uint64 `json:"generation,omitempty"`
	RecordDigest   string `json:"recordDigest,omitempty"`
}

// ObservationStatus aggregates only records that passed the strict ordinary
// observation validator. Phase values are sums of observed contributions.
type ObservationStatus struct {
	Count         int64       `json:"count"`
	Successful    int64       `json:"successful"`
	BuildFailures int64       `json:"buildFailures"`
	Cancellations int64       `json:"cancellations"`
	LastOutcome   string      `json:"lastOutcome,omitempty"`
	WallTime      Measurement `json:"wallTimeMs"`
	GradleTime    Measurement `json:"gradleTimeMs"`
	CacheTime     Measurement `json:"cacheTimeMs"`
}

// TrialStatus is unavailable until a scope-bound lifecycle result is present.
// It prevents ordinary observations from being mistaken for candidate/control
// experiments.
type TrialStatus struct {
	Count  Measurement `json:"count"`
	Reason string      `json:"reason"`
}

// CacheStatus separates usage facts from timing. The ordinary observer does
// not claim hit/miss counts, so those values remain explicitly unavailable.
type CacheStatus struct {
	Transport string      `json:"transport"`
	Hits      Measurement `json:"hits"`
	Misses    Measurement `json:"misses"`
	Reason    string      `json:"reason"`
}

// EconomicsStatus is populated only by a validated, scope-bound lifecycle
// ledger. Missing evidence remains unavailable rather than zero.
type EconomicsStatus struct {
	GrossSavedMs   Measurement `json:"grossSavedMs"`
	BuildOptCostMs Measurement `json:"buildoptCostMs"`
	NetSavedMs     Measurement `json:"netSavedMs"`
	Reason         string      `json:"reason"`
}

// FallbackStatus explains why native Gradle was retained for this report.
type FallbackStatus struct {
	Applied bool   `json:"applied"`
	Reason  string `json:"reason"`
}

// BindingStatus exposes exact input digests from the latest validated
// observation. Values that were not observed remain unavailable.
type BindingStatus struct {
	State            string `json:"state"`
	RepositoryScope  string `json:"repositoryScopeSha256"`
	SourceRevision   string `json:"sourceRevision,omitempty"`
	GradleVersion    string `json:"gradleVersion,omitempty"`
	WrapperSHA256    string `json:"wrapperSha256,omitempty"`
	ArgumentsSHA256  string `json:"argumentsSha256,omitempty"`
	BuildOptSHA256   string `json:"buildoptSha256,omitempty"`
	UnavailableCause string `json:"unavailableCause,omitempty"`
}

// StatusReport is the common data model for both status and explain. Human
// output is rendered from this value and JSON is its lossless representation.
type StatusReport struct {
	SchemaVersion string            `json:"schemaVersion"`
	ReportType    string            `json:"reportType"`
	Repository    string            `json:"repositoryScopeSha256"`
	Wrapper       WrapperStatus     `json:"wrapper"`
	Decision      DecisionStatus    `json:"decision"`
	Observations  ObservationStatus `json:"observations"`
	Trials        TrialStatus       `json:"trials"`
	Cache         CacheStatus       `json:"cache"`
	Economics     EconomicsStatus   `json:"economics"`
	Fallback      FallbackStatus    `json:"fallback"`
	Bindings      BindingStatus     `json:"bindings"`
	Explanation   []string          `json:"explanation"`
}

// BuildStatus loads committed wrapper state and private ordinary observations
// without creating or changing any file. It is safe to call from a clean
// checkout before the first build.
func BuildStatus(root, reportType string) (StatusReport, error) {
	if reportType != "STATUS" && reportType != "EXPLAIN" {
		return StatusReport{}, errors.New("status report type is invalid")
	}
	if root == "" || !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return StatusReport{}, errors.New("wrapper root must be one clean absolute path")
	}
	generator := Generator{Root: root}
	snapshot, err := generator.Check()
	if err != nil {
		return StatusReport{}, err
	}
	identity := root
	if snapshot.Config.ProjectScope != "" {
		identity = snapshot.Config.ProjectScope
	}
	scope := stickyobservation.ScopeForRoot(identity)
	report := StatusReport{
		SchemaVersion: StatusSchemaVersion,
		ReportType:    reportType,
		Repository:    scope,
		Wrapper: WrapperStatus{
			DistributionVersion: snapshot.Release.Version,
			Mode:                snapshot.Config.Mode,
			ServerConfigured:    snapshot.Config.ServerURL != "",
			ProjectScopeBound:   snapshot.Config.ProjectScope != "",
		},
		Observations: ObservationStatus{
			WallTime:   unavailableMeasurement("No ordinary build observation is available."),
			GradleTime: unavailableMeasurement("No ordinary build observation is available."),
			CacheTime:  unavailableMeasurement("No ordinary build observation is available."),
		},
		Decision: DecisionStatus{
			State:  "NATIVE",
			Reason: "No verified local decision snapshot is available; native Gradle is retained.",
		},
		Trials: TrialStatus{
			Count:  unavailableMeasurement("No verified trial ledger is available."),
			Reason: "Ordinary builds are not candidate/control trials.",
		},
		Cache: CacheStatus{
			Transport: "GRADLE_HTTP_OR_LOCAL",
			Hits:      unavailableMeasurement("Ordinary observations do not record cache hit counts."),
			Misses:    unavailableMeasurement("Ordinary observations do not record cache miss counts."),
			Reason:    "Cache objects never authorize a decision.",
		},
		Economics: EconomicsStatus{
			GrossSavedMs:   unavailableMeasurement("No verified signed economic ledger is available."),
			BuildOptCostMs: unavailableMeasurement("No verified signed economic ledger is available."),
			NetSavedMs:     unavailableMeasurement("No verified signed economic ledger is available."),
			Reason:         "A missing ledger is not treated as zero value.",
		},
		Fallback: FallbackStatus{
			Applied: true,
			Reason:  "Native Gradle is the fail-closed path until a compatible decision is verified.",
		},
		Bindings: BindingStatus{
			State:            "UNAVAILABLE",
			RepositoryScope:  scope,
			UnavailableCause: "No ordinary build observation has been recorded yet.",
		},
	}

	if decision, decisionErr := readLocalDecisionStatus(scope); decisionErr != nil {
		return StatusReport{}, decisionErr
	} else if decision != nil {
		report.Decision = *decision
	}
	records, err := loadOrdinaryObservations(root, snapshot.Config, scope)
	if err != nil {
		return StatusReport{}, err
	}
	if len(records) > 0 {
		report.Observations = summarizeObservations(records)
		last := records[len(records)-1]
		report.Bindings = BindingStatus{
			State:           "EXACT",
			RepositoryScope: last.Provenance.RepositoryScopeSHA256,
			SourceRevision:  last.Provenance.SourceRevision,
			GradleVersion:   last.Provenance.GradleVersion,
			WrapperSHA256:   last.Provenance.WrapperSHA256,
			ArgumentsSHA256: last.Provenance.ArgumentsSHA256,
			BuildOptSHA256:  last.Provenance.BuildOptSHA256,
		}
	}
	if lifecycle, lifecycleErr := loadLearningLifecycleStatus(scope); lifecycleErr != nil {
		return StatusReport{}, lifecycleErr
	} else if lifecycle != nil {
		report.Trials = lifecycle.trials
		report.Economics = lifecycle.economics
		report.Decision = lifecycle.decision
		report.Fallback = FallbackStatus{Applied: true, Reason: "The composed fixture suspended and retired the regressed action; native Gradle is retained."}
	}
	report.Explanation = explainReport(report)
	if err := report.Validate(); err != nil {
		return StatusReport{}, err
	}
	return report, nil
}

type learningLifecycleStatus struct {
	trials    TrialStatus
	economics EconomicsStatus
	decision  DecisionStatus
}

func loadLearningLifecycleStatus(scope string) (*learningLifecycleStatus, error) {
	path := os.Getenv(learningLifecycleOutputEnv)
	if path == "" {
		cacheRoot, err := os.UserCacheDir()
		if err != nil {
			return nil, nil
		}
		path = filepath.Join(cacheRoot, "buildopt", "sticky", "state", scope, "lifecycle.json")
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, errors.New("learning lifecycle path must be one clean absolute path")
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read learning lifecycle: %w", err)
	}
	var document struct {
		SchemaVersion string                        `json:"schemaVersion"`
		Ledger        stickydecision.EconomicLedger `json:"ledger"`
		Recomputed    struct {
			PairEffectsNs []int64 `json:"pairEffectsNs"`
			Qualified     bool    `json:"qualified"`
		} `json:"recomputed"`
		Outcome string `json:"outcome"`
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, errors.New("learning lifecycle evidence is invalid")
	}
	for name, target := range map[string]any{
		"schemaVersion": &document.SchemaVersion, "ledger": &document.Ledger,
		"recomputed": &document.Recomputed, "outcome": &document.Outcome,
	} {
		value, ok := envelope[name]
		if !ok || json.Unmarshal(value, target) != nil {
			return nil, errors.New("learning lifecycle evidence is incomplete")
		}
	}
	ledgerRaw, err := stickydecision.MarshalCanonical(document.Ledger)
	if err != nil {
		return nil, errors.New("learning lifecycle ledger is invalid")
	}
	if decoded, decodeErr := stickydecision.DecodeDocument(ledgerRaw, time.Now().UTC()); decodeErr != nil || decoded.Ledger == nil {
		return nil, errors.New("learning lifecycle ledger is invalid")
	}
	if document.SchemaVersion != "buildopt.poc/sticky-wrapper-learning-lifecycle/v1" ||
		document.Outcome != "LIFECYCLE_COMPOSED_NATIVE_FALLBACK_PROVEN" ||
		!document.Recomputed.Qualified || len(document.Recomputed.PairEffectsNs) == 0 ||
		document.Ledger.Binding.RepositoryScopeSHA256 != scope ||
		document.Ledger.BuildOptCostMs > uint64(^uint64(0)>>1) ||
		document.Ledger.NetSavedMs != document.Ledger.GrossSavedMs-int64(document.Ledger.BuildOptCostMs) {
		return nil, errors.New("learning lifecycle evidence does not reconcile")
	}
	count := int64(len(document.Recomputed.PairEffectsNs))
	gross, cost, net := document.Ledger.GrossSavedMs, int64(document.Ledger.BuildOptCostMs), document.Ledger.NetSavedMs
	return &learningLifecycleStatus{
		trials: TrialStatus{Count: availableMeasurement(count, "pairs"), Reason: "Verified paired lifecycle evidence is available."},
		economics: EconomicsStatus{
			GrossSavedMs:   availableMeasurement(gross, "milliseconds"),
			BuildOptCostMs: availableMeasurement(cost, "milliseconds"),
			NetSavedMs:     availableMeasurement(net, "milliseconds"),
			Reason:         "Values come from the verified economic ledger and include signed regressions.",
		},
		decision: DecisionStatus{State: "NATIVE", StoredDecision: stickydecision.ExecutionRetired, Reason: "The lifecycle retired its suspended action; native Gradle is retained.", Generation: document.Ledger.StoreGeneration},
	}, nil
}

func loadOrdinaryObservations(root string, config Config, scope string) ([]stickyobservation.Record, error) {
	path := os.Getenv(ordinaryObservationOutputEnv)
	if path == "" {
		cacheRoot, err := os.UserCacheDir()
		if err != nil {
			return nil, nil
		}
		path = filepath.Join(cacheRoot, "buildopt", "sticky", "observations", scope, "builds.jsonl")
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, errors.New("ordinary observation path must be one clean absolute path")
	}
	records, err := stickyobservation.Load(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load ordinary observations: %w", err)
	}
	for _, record := range records {
		if record.Provenance.RepositoryScopeSHA256 != scope {
			return nil, errors.New("ordinary observation scope does not match wrapper repository")
		}
	}
	return records, nil
}

func readLocalDecisionStatus(scope string) (*DecisionStatus, error) {
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return nil, nil
	}
	root, err := stickydecision.LocalDecisionRoot(cacheRoot, scope)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	store, err := stickydecision.OpenLocalReadOnly(root, scope, stickydecision.StoreOptions{})
	if err != nil {
		return nil, fmt.Errorf("inspect local decision state: %w", err)
	}
	snapshot, err := store.CurrentReadOnly(context.Background())
	if errors.Is(err, stickydecision.ErrNotFound) {
		return nil, nil
	}
	if errors.Is(err, stickydecision.ErrExpired) || errors.Is(err, stickydecision.ErrRevoked) {
		return &DecisionStatus{
			State:  "NATIVE",
			Reason: "The local decision is no longer valid; native Gradle is retained.",
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect local decision state: %w", err)
	}
	if snapshot.Document.Decision == nil {
		return nil, nil
	}
	return &DecisionStatus{
		State:          "UNAVAILABLE",
		StoredDecision: snapshot.Document.Decision.ExecutionDecision,
		Reason:         "A local decision exists but no pinned public-key registry is available to verify it; native Gradle is retained.",
		Generation:     snapshot.Head.Generation,
		RecordDigest:   snapshot.RecordDigest,
	}, nil
}

func summarizeObservations(records []stickyobservation.Record) ObservationStatus {
	status := ObservationStatus{Count: int64(len(records)), LastOutcome: records[len(records)-1].Outcome}
	for _, record := range records {
		switch record.Outcome {
		case "SUCCESS":
			status.Successful++
		case "BUILD_FAILURE", "INFRA_FAILURE":
			status.BuildFailures++
		case "CANCELLED":
			status.Cancellations++
		}
	}
	status.WallTime = sumPhase(records, func(record stickyobservation.Record) (stickyobservation.Phase, bool) {
		return stickyobservation.Phase{DurationNs: record.Timing.TotalNs, Evidence: "EXACT"}, true
	})
	status.GradleTime = sumPhase(records, func(record stickyobservation.Record) (stickyobservation.Phase, bool) {
		phase := record.Timing.Gradle
		return phase, phase.Evidence != "UNAVAILABLE"
	})
	status.CacheTime = sumPhase(records, func(record stickyobservation.Record) (stickyobservation.Phase, bool) {
		phase := record.Timing.Cache
		return phase, phase.Evidence != "UNAVAILABLE"
	})
	return status
}

func sumPhase(records []stickyobservation.Record, selectPhase func(stickyobservation.Record) (stickyobservation.Phase, bool)) Measurement {
	var total int64
	available := false
	for _, record := range records {
		phase, ok := selectPhase(record)
		if !ok {
			continue
		}
		available = true
		total += phase.DurationNs
	}
	if !available {
		return unavailableMeasurement("This phase was not observed.")
	}
	value := total / int64(time.Millisecond)
	return Measurement{State: "AVAILABLE", Value: &value, Unit: "milliseconds"}
}

func availableMeasurement(value int64, unit string) Measurement {
	return Measurement{State: "AVAILABLE", Value: &value, Unit: unit}
}

func unavailableMeasurement(reason string) Measurement {
	return Measurement{State: "UNAVAILABLE", Reason: reason}
}

// Validate rejects malformed status data before it can be printed or consumed
// by a script. In particular, unavailable values cannot smuggle a numeric 0.
func (report StatusReport) Validate() error {
	if report.SchemaVersion != StatusSchemaVersion || (report.ReportType != "STATUS" && report.ReportType != "EXPLAIN") || report.Repository == "" {
		return errors.New("status report identity is invalid")
	}
	if report.Observations.Count < 0 || report.Observations.Successful < 0 || report.Observations.BuildFailures < 0 || report.Observations.Cancellations < 0 || report.Observations.Successful+report.Observations.BuildFailures+report.Observations.Cancellations > report.Observations.Count {
		return errors.New("status observation counts are invalid")
	}
	for _, measurement := range []Measurement{
		report.Observations.WallTime, report.Observations.GradleTime, report.Observations.CacheTime,
		report.Trials.Count, report.Cache.Hits, report.Cache.Misses,
		report.Economics.GrossSavedMs, report.Economics.BuildOptCostMs, report.Economics.NetSavedMs,
	} {
		if err := measurement.Validate(); err != nil {
			return err
		}
	}
	if report.Bindings.State != "EXACT" && report.Bindings.State != "UNAVAILABLE" {
		return errors.New("status binding state is invalid")
	}
	if report.Decision.State != "NATIVE" && report.Decision.State != "UNAVAILABLE" && report.Decision.State != "ACTIVE" {
		return errors.New("status decision state is invalid")
	}
	return nil
}

func (measurement Measurement) Validate() error {
	if measurement.State != "AVAILABLE" && measurement.State != "UNAVAILABLE" {
		return errors.New("status measurement state is invalid")
	}
	if measurement.State == "AVAILABLE" {
		if measurement.Value == nil || *measurement.Value < 0 || measurement.Unit == "" || measurement.Reason != "" {
			return errors.New("available status measurement is invalid")
		}
	} else if measurement.Value != nil || measurement.Unit != "" || measurement.Reason == "" {
		return errors.New("unavailable status measurement is invalid")
	}
	return nil
}

func explainReport(report StatusReport) []string {
	lines := []string{
		fmt.Sprintf("Decision: %s — %s", strings.ToLower(report.Decision.State), report.Decision.Reason),
		fmt.Sprintf("Wrapper: version %s, mode %s.", report.Wrapper.DistributionVersion, report.Wrapper.Mode),
	}
	if report.Observations.Count == 0 {
		lines = append(lines, "Observations: none recorded yet.")
	} else {
		lines = append(lines, fmt.Sprintf(
			"Observations: %d builds (%d successful, %d failed, %d cancelled); last outcome %s.",
			report.Observations.Count, report.Observations.Successful, report.Observations.BuildFailures,
			report.Observations.Cancellations, report.Observations.LastOutcome,
		))
	}
	lines = append(lines,
		measurementSentence("Wall time", report.Observations.WallTime),
		measurementSentence("Gradle time", report.Observations.GradleTime),
		"Cache hit/miss counts: unavailable because ordinary observations do not record them.",
		"Economics: unavailable until a verified signed ledger is present; unavailable is not zero.",
		"Native fallback: "+report.Fallback.Reason,
	)
	if report.Bindings.State == "EXACT" {
		lines = append(lines, fmt.Sprintf("Latest binding: repository scope %s, Gradle %s, source revision %s.", shortDigest(report.Bindings.RepositoryScope), report.Bindings.GradleVersion, valueOrUnavailable(report.Bindings.SourceRevision)))
	} else {
		lines = append(lines, "Latest binding: unavailable because no validated ordinary build has been recorded.")
	}
	return lines
}

func measurementSentence(label string, measurement Measurement) string {
	if measurement.State != "AVAILABLE" || measurement.Value == nil {
		return fmt.Sprintf("%s: unavailable.", label)
	}
	return fmt.Sprintf("%s: %d %s.", label, *measurement.Value, measurement.Unit)
}

func shortDigest(value string) string {
	if len(value) <= 16 {
		return value
	}
	return value[:16] + "…"
}

func valueOrUnavailable(value string) string {
	if value == "" {
		return "unavailable"
	}
	return value
}

// WriteReport writes the machine-readable report. Human output is generated
// from the same model, so a consumer can recompute every displayed number.
func WriteReport(report StatusReport, jsonOutput bool, stdout io.Writer) error {
	if err := report.Validate(); err != nil {
		return err
	}
	if jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(report)
	}
	for _, line := range report.Explanation {
		if _, err := fmt.Fprintln(stdout, line); err != nil {
			return err
		}
	}
	return nil
}
