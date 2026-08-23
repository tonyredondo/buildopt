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

	"github.com/tonyredondo/buildopt/internal/buildimpact"
	"github.com/tonyredondo/buildopt/internal/nativevolatility"
	"github.com/tonyredondo/buildopt/internal/profilediscovery"
)

const (
	profileNativeObserveUsage              = "usage: buildopt profile native-observe --state-dir PATH --root DIR\n"
	profileNativeContextUsage              = "usage: buildopt profile native-context --state-dir PATH\n"
	profileNativePreflightUsage            = "usage: buildopt profile native-preflight --state-dir PATH --portfolio FILE --portfolio-context FILE\n"
	profileQuarantineUsage                 = "usage: buildopt profile quarantine --state-dir PATH --second-observation FILE [--portfolio FILE --portfolio-context FILE --revision SHA256]\n"
	optimizeQuarantineReason               = "NATIVE_VOLATILE_PRODUCERS_QUARANTINED"
	optimizePortfolioEvidenceSchema        = "buildopt.poc/optimize-native-volatility-portfolio-evidence/v1"
	optimizePortfolioRetentionSchema       = "buildopt.poc/optimize-native-volatility-portfolio-retention/v1"
	optimizePortfolioReasonMaterialization = "CURRENT_MATERIALIZATION_UNAVAILABLE"
	optimizePortfolioReasonLineage         = "PORTFOLIO_PRODUCER_LINEAGE_UNAVAILABLE"
)

var errOptimizeNativePortfolioContextDrift = errors.New("native volatility portfolio context drifted")
var errOptimizeNativePortfolioLineage = errors.New("native volatility portfolio producer lineage is unavailable")

type optimizeNativeQuarantinePlan struct {
	BindingSHA256        string
	QuarantinedProducers []string
	QuarantinedOutputs   []nativevolatility.Entry
	TransportedOutputs   []nativevolatility.Entry
}

type optimizeNativePortfolioEvidence struct {
	SchemaVersion string                                `json:"schemaVersion"`
	Portfolio     nativevolatility.Portfolio            `json:"portfolio"`
	Current       nativevolatility.Result               `json:"current"`
	Application   nativevolatility.PortfolioApplication `json:"application"`
}

type optimizeNativePortfolioRetention struct {
	SchemaVersion        string                            `json:"schemaVersion"`
	Portfolio            nativevolatility.Portfolio        `json:"portfolio"`
	Current              nativevolatility.Result           `json:"current"`
	LearnedContext       nativevolatility.PortfolioContext `json:"learnedContext"`
	CurrentContext       nativevolatility.PortfolioContext `json:"currentContext"`
	LearnedContextSHA    string                            `json:"learnedContextSha256"`
	CurrentContextSHA    string                            `json:"currentContextSha256"`
	Decision             string                            `json:"decision"`
	Reason               string                            `json:"reason"`
	DriftedBindings      []string                          `json:"driftedBindings"`
	MissingProducers     []string                          `json:"missingProducers"`
	ProductionAuthorized bool                              `json:"productionAuthorized"`
	TestOptimization     string                            `json:"testOptimization"`
}

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
	invocation, state, err := loadOptimizeStateForNativeEvidence(*stateRelative)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: native output observation unavailable: %v\n", err)
		return exitConfiguration
	}
	inventory, err := optimizeNativeObservationInventory(invocation, state)
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

func runOptimizeNativePortfolioContext(args []string, stdout, stderr io.Writer) int {
	if isHelp(args) {
		_, _ = io.WriteString(stdout, profileNativeContextUsage)
		return 0
	}
	flags := flag.NewFlagSet("buildopt profile native-context", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	stateRelative := flags.String("state-dir", optimizeDefaultStateDir, "repository-local optimize state")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		_, _ = io.WriteString(stderr, profileNativeContextUsage)
		return exitUsage
	}
	_, state, err := loadOptimizeStateForNativeEvidence(*stateRelative)
	if err == nil {
		var context nativevolatility.PortfolioContext
		context, err = optimizeNativePortfolioContext(state)
		if err == nil {
			err = encodeOptimizeJSON(stdout, context)
		}
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: native portfolio context unavailable: %v\n", err)
		return exitConfiguration
	}
	return 0
}

func runOptimizeNativePortfolioPreflight(args []string, stdout, stderr io.Writer) int {
	if isHelp(args) {
		_, _ = io.WriteString(stdout, profileNativePreflightUsage)
		return 0
	}
	flags := flag.NewFlagSet("buildopt profile native-preflight", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	stateRelative := flags.String("state-dir", optimizeDefaultStateDir, "repository-local optimize state")
	portfolioPath := flags.String("portfolio", "", "producer-volatility portfolio")
	contextPath := flags.String("portfolio-context", "", "learned portfolio context binding")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 ||
		*portfolioPath == "" || *contextPath == "" {
		_, _ = io.WriteString(stderr, profileNativePreflightUsage)
		return exitUsage
	}
	_, state, err := loadOptimizeStateForNativeEvidence(*stateRelative)
	if err == nil {
		var result nativevolatility.PortfolioPreflight
		result, err = optimizeNativePortfolioPreflight(state, *portfolioPath, *contextPath)
		if err == nil {
			err = encodeOptimizeJSON(stdout, result)
		}
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: native portfolio preflight unavailable: %v\n", err)
		return exitConfiguration
	}
	return 0
}

func optimizeNativePortfolioPreflight(
	state optimizeState,
	portfolioPath string,
	contextPath string,
) (nativevolatility.PortfolioPreflight, error) {
	portfolio, err := readOptimizeNativePortfolio(portfolioPath)
	if err != nil {
		return nativevolatility.PortfolioPreflight{}, err
	}
	learned, err := readOptimizeNativePortfolioContext(contextPath)
	if err != nil {
		return nativevolatility.PortfolioPreflight{}, err
	}
	current, err := optimizeNativePortfolioContext(state)
	if err != nil {
		return nativevolatility.PortfolioPreflight{}, err
	}
	return nativevolatility.PreflightPortfolio(portfolio, learned, current)
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
	portfolioPath := flags.String("portfolio", "", "compatible producer-volatility portfolio")
	portfolioContextPath := flags.String("portfolio-context", "", "portfolio context binding")
	revision := flags.String("revision", "", "current source revision SHA-256")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 || *secondPath == "" {
		_, _ = io.WriteString(stderr, profileQuarantineUsage)
		return exitUsage
	}
	invocation, state, err := loadOptimizeStateForNativeEvidence(*stateRelative)
	if err == nil {
		var result nativevolatility.Result
		var application *nativevolatility.PortfolioApplication
		var retention *optimizeNativePortfolioRetention
		if state.Discovery.Status == optimizeDiscoveryComplete &&
			state.Discovery.Materialization.Status == optimizeMaterializationCaptured {
			result, application, err = applyOptimizeNativeQuarantineWithPortfolio(
				invocation, &state, *secondPath, *portfolioPath, *portfolioContextPath, *revision,
			)
		} else if *portfolioPath != "" || *portfolioContextPath != "" || *revision != "" {
			result, err = analyzeDiagnosticNativeObservations(invocation, state, *secondPath)
			if err == nil {
				var value optimizeNativePortfolioRetention
				value, err = optimizeNativePortfolioRetentionEvidence(
					state, result, *portfolioPath, *portfolioContextPath,
					optimizePortfolioReasonMaterialization,
				)
				retention = &value
			}
		} else {
			result, err = analyzeDiagnosticNativeObservations(invocation, state, *secondPath)
		}
		if err == nil {
			if retention != nil {
				err = encodeOptimizeJSON(stdout, *retention)
			} else if application != nil {
				var portfolio nativevolatility.Portfolio
				portfolio, err = readOptimizeNativePortfolio(*portfolioPath)
				if err == nil {
					err = encodeOptimizeJSON(stdout, optimizeNativePortfolioEvidence{
						SchemaVersion: optimizePortfolioEvidenceSchema,
						Portfolio:     portfolio, Current: result, Application: *application,
					})
				}
			} else {
				err = encodeOptimizeJSON(stdout, result)
			}
		} else if errors.Is(err, errOptimizeNativePortfolioContextDrift) {
			err = encodeOptimizeNativePortfolioRetention(
				stdout, state, result, *portfolioPath, *portfolioContextPath,
				"PORTFOLIO_CONTEXT_DRIFT",
			)
		} else if errors.Is(err, errOptimizeNativePortfolioLineage) {
			err = encodeOptimizeNativePortfolioRetention(
				stdout, state, result, *portfolioPath, *portfolioContextPath,
				optimizePortfolioReasonLineage,
			)
		}
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: native output quarantine unavailable: %v\n", err)
		return exitConfiguration
	}
	return 0
}

func encodeOptimizeNativePortfolioRetention(
	stdout io.Writer,
	state optimizeState,
	current nativevolatility.Result,
	portfolioPath string,
	contextPath string,
	reason string,
) error {
	retention, err := optimizeNativePortfolioRetentionEvidence(
		state, current, portfolioPath, contextPath, reason,
	)
	if err != nil {
		return err
	}
	return encodeOptimizeJSON(stdout, retention)
}

func optimizeNativePortfolioRetentionEvidence(
	state optimizeState,
	current nativevolatility.Result,
	portfolioPath string,
	contextPath string,
	reason string,
) (optimizeNativePortfolioRetention, error) {
	portfolio, err := readOptimizeNativePortfolio(portfolioPath)
	if err != nil {
		return optimizeNativePortfolioRetention{}, err
	}
	learned, err := readOptimizeNativePortfolioContext(contextPath)
	if err != nil {
		return optimizeNativePortfolioRetention{}, err
	}
	learnedSHA, err := nativevolatility.ContextSHA256(learned)
	if err != nil || learnedSHA != portfolio.ContextSHA256 {
		return optimizeNativePortfolioRetention{}, errors.New("native volatility portfolio context is invalid")
	}
	currentContext, err := optimizeNativePortfolioContext(state)
	if err != nil {
		return optimizeNativePortfolioRetention{}, err
	}
	preflight, err := nativevolatility.PreflightPortfolio(portfolio, learned, currentContext)
	if err != nil {
		return optimizeNativePortfolioRetention{}, errors.New("native volatility portfolio retention context is invalid")
	}
	missingProducers := optimizeMissingPortfolioProducers(portfolio, current)
	if reason == "PORTFOLIO_CONTEXT_DRIFT" {
		if preflight.Decision != nativevolatility.PortfolioDecisionRetained {
			return optimizeNativePortfolioRetention{}, errors.New("native volatility portfolio retention context is invalid")
		}
	} else if reason == optimizePortfolioReasonMaterialization {
		if preflight.Decision != nativevolatility.PortfolioDecisionCompatible ||
			len(preflight.DriftedBindings) != 0 {
			return optimizeNativePortfolioRetention{}, errors.New("native volatility portfolio materialization retention is invalid")
		}
	} else if reason == optimizePortfolioReasonLineage {
		if preflight.Decision != nativevolatility.PortfolioDecisionCompatible ||
			len(preflight.DriftedBindings) != 0 || len(missingProducers) == 0 {
			return optimizeNativePortfolioRetention{}, errors.New("native volatility portfolio lineage retention is invalid")
		}
	} else {
		return optimizeNativePortfolioRetention{}, errors.New("native volatility portfolio retention reason is invalid")
	}
	return optimizeNativePortfolioRetention{
		SchemaVersion: optimizePortfolioRetentionSchema,
		Portfolio:     portfolio, Current: current,
		LearnedContext: learned, CurrentContext: currentContext,
		LearnedContextSHA: learnedSHA, CurrentContextSHA: preflight.CurrentContextSHA256,
		Decision: "NATIVE_RETAINED", Reason: reason,
		DriftedBindings: preflight.DriftedBindings, MissingProducers: missingProducers,
		ProductionAuthorized: false,
		TestOptimization:     "OUT_OF_SCOPE",
	}, nil
}

func optimizeMissingPortfolioProducers(
	portfolio nativevolatility.Portfolio,
	current nativevolatility.Result,
) []string {
	present := make(map[string]bool)
	entries := append([]nativevolatility.Entry(nil), current.QuarantinedOutputs...)
	entries = append(entries, current.TransportedOutputs...)
	for _, entry := range entries {
		for _, producer := range entry.ProducerTasks {
			present[producer] = true
		}
	}
	missing := make([]string, 0, len(portfolio.QuarantinedProducers))
	for _, producer := range portfolio.QuarantinedProducers {
		if !present[producer] {
			missing = append(missing, producer)
		}
	}
	return missing
}

func loadOptimizeQuarantineState(relative string) (optimizeInvocation, optimizeState, error) {
	invocation, state, err := loadOptimizeStateForNativeEvidence(relative)
	if err != nil {
		return optimizeInvocation{}, optimizeState{}, err
	}
	if state.Discovery.Status != optimizeDiscoveryComplete ||
		state.Discovery.Materialization.Status != optimizeMaterializationCaptured {
		return optimizeInvocation{}, optimizeState{}, errors.New("complete captured materialization is required")
	}
	return invocation, state, nil
}

func loadOptimizeStateForNativeEvidence(relative string) (optimizeInvocation, optimizeState, error) {
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
	return optimizeInvocation{
		repositoryRoot: repositoryRoot, stateDirectory: stateDirectory, stateRelative: normalized,
		bindingSHA256: state.Bindings.SHA256, calibrationPairs: state.Budget.Pairs,
		maxBreakEvenBuilds: state.Budget.MaxBreakEvenBuilds,
		calibrationBudget:  time.Duration(state.Budget.WallTimeSeconds) * time.Second,
	}, state, nil
}

func optimizeNativeObservationInventory(
	invocation optimizeInvocation,
	state optimizeState,
) ([]nativevolatility.Entry, error) {
	if state.Discovery.Status == optimizeDiscoveryComplete &&
		state.Discovery.Materialization.Status == optimizeMaterializationCaptured {
		manifest, _, err := loadOptimizeOutputMaterialization(invocation, state.Discovery)
		if err != nil {
			return nil, err
		}
		return optimizeNativeInventory(manifest)
	}
	first, err := readOptimizeDiagnosticNativeObservation(invocation, state)
	if err != nil {
		return nil, err
	}
	return append([]nativevolatility.Entry(nil), first.Entries...), nil
}

func analyzeDiagnosticNativeObservations(
	invocation optimizeInvocation,
	state optimizeState,
	secondPath string,
) (nativevolatility.Result, error) {
	first, err := readOptimizeDiagnosticNativeObservation(invocation, state)
	if err != nil {
		return nativevolatility.Result{}, err
	}
	second, err := readOptimizeNativeObservation(secondPath)
	if err != nil {
		return nativevolatility.Result{}, err
	}
	result := nativevolatility.Analyze(first, second)
	if result.Decision != nativevolatility.DecisionTransportReady {
		return result, fmt.Errorf("diagnostic native output evidence retained Gradle: %s", result.Reason)
	}
	return result, nil
}

func readOptimizeDiagnosticNativeObservation(
	invocation optimizeInvocation,
	state optimizeState,
) (nativevolatility.Observation, error) {
	metadata := state.Discovery.NativeObservation
	if metadata.Status != optimizeNativeObservationCaptured ||
		!validOptimizeNativeObservationShape(metadata) {
		return nativevolatility.Observation{}, errors.New("diagnostic native output observation is unavailable")
	}
	expected := filepath.ToSlash(filepath.Join(
		invocation.stateRelative, "native-volatility", "first-observation.json",
	))
	if metadata.File != expected {
		return nativevolatility.Observation{}, errors.New("diagnostic native output observation path drifted")
	}
	absolute := filepath.Join(invocation.repositoryRoot, filepath.FromSlash(metadata.File))
	raw, err := os.ReadFile(absolute)
	if err != nil || len(raw) == 0 || len(raw) > optimizeMaterializationMaxManifest {
		return nativevolatility.Observation{}, errors.New("diagnostic native output observation is unavailable")
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != metadata.SHA256 {
		return nativevolatility.Observation{}, errors.New("diagnostic native output observation digest drifted")
	}
	observation, err := readOptimizeNativeObservation(absolute)
	if err != nil || observation.BindingSHA256 != state.Bindings.SHA256 ||
		len(observation.Entries) != metadata.OutputCount {
		return nativevolatility.Observation{}, errors.New("diagnostic native output observation binding drifted")
	}
	contractSHA, err := nativevolatility.OutputContractSHA256(observation)
	if err != nil || contractSHA != metadata.OutputContractSHA256 {
		return nativevolatility.Observation{}, errors.New("diagnostic native output contract drifted")
	}
	return observation, nil
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
	result, _, err := applyOptimizeNativeQuarantineWithPortfolio(
		invocation, state, secondPath, "", "", "",
	)
	return result, err
}

func applyOptimizeNativeQuarantineWithPortfolio(
	invocation optimizeInvocation,
	state *optimizeState,
	secondPath string,
	portfolioPath string,
	portfolioContextPath string,
	revision string,
) (nativevolatility.Result, *nativevolatility.PortfolioApplication, error) {
	portfolioRequested := portfolioPath != "" || portfolioContextPath != "" || revision != ""
	if portfolioRequested && (portfolioPath == "" || portfolioContextPath == "" || !validOptimizeSHA(revision)) {
		return nativevolatility.Result{}, nil, errors.New("complete portfolio, context and revision are required")
	}
	manifest, payloads, err := loadOptimizeOutputMaterialization(invocation, state.Discovery)
	if err != nil {
		return nativevolatility.Result{}, nil, err
	}
	inventory, err := optimizeNativeInventory(manifest)
	if err != nil {
		return nativevolatility.Result{}, nil, err
	}
	second, err := readOptimizeNativeObservation(secondPath)
	if err != nil {
		return nativevolatility.Result{}, nil, err
	}
	first := nativevolatility.Observation{
		SchemaVersion: nativevolatility.ObservationSchema,
		BindingSHA256: state.Bindings.SHA256,
		Entries:       inventory,
	}
	result := nativevolatility.Analyze(first, second)
	if result.Decision != nativevolatility.DecisionTransportReady {
		return result, nil, fmt.Errorf("native output evidence retained Gradle: %s", result.Reason)
	}
	plan := optimizeNativeQuarantinePlan{
		BindingSHA256:        result.BindingSHA256,
		QuarantinedProducers: append([]string(nil), result.QuarantinedProducers...),
		QuarantinedOutputs:   append([]nativevolatility.Entry(nil), result.QuarantinedOutputs...),
		TransportedOutputs:   append([]nativevolatility.Entry(nil), result.TransportedOutputs...),
	}
	var portfolio nativevolatility.Portfolio
	var application *nativevolatility.PortfolioApplication
	if portfolioRequested {
		revisionDigest := sha256.Sum256([]byte(state.Discovery.TargetRevision))
		if revision != hex.EncodeToString(revisionDigest[:]) {
			return result, nil, errors.New("native volatility portfolio revision does not bind the current optimize state")
		}
		portfolio, err = readOptimizeNativePortfolio(portfolioPath)
		if err != nil {
			return result, nil, err
		}
		context, contextErr := readOptimizeNativePortfolioContext(portfolioContextPath)
		if contextErr != nil {
			return result, nil, contextErr
		}
		contextSHA, contextErr := nativevolatility.ContextSHA256(context)
		if contextErr != nil || contextSHA != portfolio.ContextSHA256 {
			return result, nil, errors.New("native volatility portfolio context is invalid")
		}
		expectedContext, contextErr := optimizeNativePortfolioContext(*state)
		if contextErr != nil {
			return result, nil, contextErr
		}
		if context != expectedContext {
			return result, nil, fmt.Errorf("%w: native volatility portfolio does not bind the current optimize context", errOptimizeNativePortfolioContextDrift)
		}
		if missing := optimizeMissingPortfolioProducers(portfolio, result); len(missing) > 0 {
			return result, nil, fmt.Errorf(
				"%w: %s", errOptimizeNativePortfolioLineage, strings.Join(missing, ","),
			)
		}
		applied, applyErr := nativevolatility.ApplyPortfolio(portfolio, context, revision, result)
		if applyErr != nil {
			return result, nil, applyErr
		}
		application = &applied
		plan.QuarantinedProducers = append([]string(nil), applied.EffectiveQuarantinedProducers...)
		plan.QuarantinedOutputs = append([]nativevolatility.Entry(nil), applied.QuarantinedOutputs...)
		plan.TransportedOutputs = append([]nativevolatility.Entry(nil), applied.TransportedOutputs...)
	}
	if len(plan.TransportedOutputs) == 0 {
		return result, application, errors.New("quarantine leaves no transportable outputs")
	}

	stable := make(map[string]nativevolatility.Entry, len(plan.TransportedOutputs))
	for _, entry := range plan.TransportedOutputs {
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
			return result, application, fmt.Errorf("transported output %s drifted before quarantine", entry.Path)
		}
		filtered = append(filtered, payload)
	}
	if len(filtered) != len(stable) {
		return result, application, errors.New("transported output set is incomplete")
	}
	if len(plan.QuarantinedProducers) == 0 {
		if len(filtered) != len(payloads) ||
			len(result.TransportedOutputs) != result.ComparedOutputCount {
			return result, application, errors.New("exact native output set is incomplete")
		}
		resultRelative, resultSHA, persistErr := persistOptimizeNativeVolatilityResult(
			invocation, result,
		)
		if persistErr != nil {
			return result, application, persistErr
		}
		state.Discovery.Materialization.QuarantineFile = resultRelative
		state.Discovery.Materialization.QuarantineSHA256 = resultSHA
		state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		if !validOptimizeState(*state) {
			return result, application, errors.New("exact native output optimize state is invalid")
		}
		if err := writeCanonicalPrivateJSON(
			filepath.Join(invocation.stateDirectory, optimizeStateFile), state,
		); err != nil {
			return result, application, fmt.Errorf("write exact native output optimize state: %w", err)
		}
		return result, application, nil
	}

	discovery := state.Discovery
	discovery.CandidateEntrypoints = mergeOptimizeStrings(
		discovery.CandidateEntrypoints, plan.QuarantinedProducers,
	)
	if len(discovery.CandidateEntrypoints) > maximumStructuralAlternativeEntrypoints {
		return result, application, errors.New("quarantine candidate task set exceeds the POC bound")
	}
	quarantinedPaths := make([]string, 0, len(plan.QuarantinedOutputs))
	for _, entry := range plan.QuarantinedOutputs {
		quarantinedPaths = append(quarantinedPaths, entry.Path)
	}
	discovery.CandidateOutputs = mergeOptimizeStrings(discovery.CandidateOutputs, quarantinedPaths)
	if len(discovery.CandidateOutputs) > optimizeMaterializationMaxFiles {
		return result, application, errors.New("quarantine candidate output set exceeds the POC bound")
	}
	updateOptimizeQuarantinePartition(&discovery, plan.QuarantinedProducers)

	quarantineDirectory := filepath.Join(invocation.stateDirectory, "materialization", "quarantine")
	if err := os.MkdirAll(quarantineDirectory, 0o700); err != nil {
		return result, application, fmt.Errorf("create quarantine materialization directory: %w", err)
	}
	manifest, materialization, err := writeOptimizeQuarantineMaterialization(
		invocation, discovery, manifest, filtered, quarantineDirectory,
	)
	if err != nil {
		return result, application, err
	}
	_ = manifest
	if err := rewriteOptimizeQuarantineDiscoveryDocuments(invocation, discovery); err != nil {
		return result, application, fmt.Errorf("rewrite quarantined discovery documents: %w", err)
	}
	persisted := any(result)
	if application != nil {
		persisted = optimizeNativePortfolioEvidence{
			SchemaVersion: optimizePortfolioEvidenceSchema,
			Portfolio:     portfolio, Current: result, Application: *application,
		}
	}
	resultRelative, resultSHA, err := persistOptimizeNativeVolatilityEvidence(invocation, persisted)
	if err != nil {
		return result, application, err
	}
	materialization.QuarantineFile = resultRelative
	materialization.QuarantineSHA256 = resultSHA
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
		return result, application, errors.New("quarantined optimize state is invalid")
	}
	if err := writeCanonicalPrivateJSON(filepath.Join(invocation.stateDirectory, optimizeStateFile), state); err != nil {
		return result, application, fmt.Errorf("write quarantined optimize state: %w", err)
	}
	return result, application, nil
}

func persistOptimizeNativeVolatilityResult(
	invocation optimizeInvocation,
	result nativevolatility.Result,
) (string, string, error) {
	return persistOptimizeNativeVolatilityEvidence(invocation, result)
}

func persistOptimizeNativeVolatilityEvidence(
	invocation optimizeInvocation,
	evidence any,
) (string, string, error) {
	resultRelative := filepath.ToSlash(filepath.Join(
		invocation.stateRelative, "materialization", "quarantine", "native-volatility.json",
	))
	resultAbsolute := filepath.Join(invocation.repositoryRoot, filepath.FromSlash(resultRelative))
	if err := os.MkdirAll(filepath.Dir(resultAbsolute), 0o700); err != nil {
		return "", "", fmt.Errorf("create native volatility result directory: %w", err)
	}
	if err := writeCanonicalPrivateJSON(resultAbsolute, evidence); err != nil {
		return "", "", fmt.Errorf("write native volatility result: %w", err)
	}
	resultRaw, err := os.ReadFile(resultAbsolute)
	if err != nil {
		return "", "", err
	}
	resultDigest := sha256.Sum256(resultRaw)
	return resultRelative, hex.EncodeToString(resultDigest[:]), nil
}

// rewriteOptimizeQuarantineDiscoveryDocuments preserves the fail-closed
// Build Impact bindings after native volatility requires additional producers
// to run locally. The persisted configured-model snapshot is the authority for
// deriving those task entrypoints; no task-to-project relationship is guessed
// from names or from execution timing.
func rewriteOptimizeQuarantineDiscoveryDocuments(
	invocation optimizeInvocation,
	discovery optimizeDiscoveryResult,
) error {
	directory := filepath.Join(invocation.stateDirectory, "discovery")
	read := func(name string, maximum int64) ([]byte, error) {
		raw, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil || len(raw) == 0 || int64(len(raw)) > maximum {
			return nil, fmt.Errorf("%s is unavailable", name)
		}
		return raw, nil
	}
	decode := func(raw []byte, destination any) error {
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(destination); err != nil {
			return err
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			return errors.New("document has trailing JSON")
		}
		return nil
	}

	manifestRaw, err := read("manifest.json", 256<<10)
	if err != nil {
		return err
	}
	var manifest buildimpact.Manifest
	if err := decode(manifestRaw, &manifest); err != nil {
		return fmt.Errorf("decode manifest: %w", err)
	}
	loadedManifest, err := buildimpact.ParseManifest(
		manifestRaw, discovery.RepositoryID, manifest.PipelineClass,
	)
	if err != nil {
		return err
	}

	graphRaw, err := read("graph.json", 2<<20)
	if err != nil {
		return err
	}
	loadedGraph, err := buildimpact.ParseDeclaredGraph(graphRaw, loadedManifest)
	if err != nil {
		return err
	}
	generatedRaw, err := read("generated-manifest.json", 256<<10)
	if err != nil {
		return err
	}
	var generated buildimpact.GeneratedManifest
	if err := decode(generatedRaw, &generated); err != nil {
		return fmt.Errorf("decode generated manifest: %w", err)
	}
	if generated.SchemaVersion != buildimpact.GeneratedManifestSchemaVersion ||
		generated.RepositoryID != loadedManifest.Manifest.RepositoryID ||
		generated.PipelineClass != loadedManifest.Manifest.PipelineClass ||
		generated.ManifestDigest != loadedManifest.Digest ||
		generated.GraphDigest != loadedGraph.Digest {
		return errors.New("generated manifest does not bind the current discovery documents")
	}

	proposalRaw, err := read("proposal.json", 2<<20)
	if err != nil {
		return err
	}
	var proposal profileProposalReport
	if err := decode(proposalRaw, &proposal); err != nil {
		return fmt.Errorf("decode proposal: %w", err)
	}
	if proposal.Analysis == nil || proposal.Analysis.Plan == nil ||
		!equalOptimizeStrings(proposal.CandidateEntrypoints, proposal.Analysis.Plan.Entrypoints) {
		return errors.New("proposal has no bound structural candidate")
	}
	alternativeID := proposal.Analysis.Plan.AlternativeID
	alternativeFound := false
	for index := range manifest.AllowedAlternatives {
		if manifest.AllowedAlternatives[index].ID != alternativeID {
			continue
		}
		if !equalOptimizeStrings(
			manifest.AllowedAlternatives[index].Entrypoints,
			proposal.CandidateEntrypoints,
		) {
			return errors.New("proposal and manifest alternatives differ")
		}
		manifest.AllowedAlternatives[index].Entrypoints = append(
			[]string(nil), discovery.CandidateEntrypoints...,
		)
		alternativeFound = true
	}
	if !alternativeFound {
		return errors.New("proposal alternative is absent from the manifest")
	}
	updatedManifestRaw, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	updatedManifest, err := buildimpact.ParseManifest(
		updatedManifestRaw, discovery.RepositoryID, manifest.PipelineClass,
	)
	if err != nil {
		return err
	}

	snapshotRaw, err := read("snapshot.json", 4<<20)
	if err != nil {
		return err
	}
	var snapshot buildimpact.DiscoverySnapshot
	if err := decode(snapshotRaw, &snapshot); err != nil {
		return fmt.Errorf("decode configured-model snapshot: %w", err)
	}
	derived, err := buildimpact.DeriveProjectEntrypoints(
		snapshot, discovery.CandidateEntrypoints,
	)
	if err != nil {
		return err
	}
	derivedRaw, err := json.Marshal(derived)
	if err != nil {
		return err
	}
	updated, err := buildimpact.GenerateImpact(updatedManifest, derivedRaw)
	if err != nil {
		return err
	}
	analysis := profilediscovery.AnalyzeGeneratedOpportunity(
		updated.Manifest, updated.Graph, updated.Generated,
	)
	if analysis.Decision != profilediscovery.DecisionMeasure || analysis.Plan == nil ||
		!equalOptimizeStrings(analysis.Plan.Entrypoints, discovery.CandidateEntrypoints) {
		return errors.New("quarantined candidate no longer provides structural reduction")
	}

	proposal.CandidateEntrypoints = append([]string(nil), discovery.CandidateEntrypoints...)
	proposal.OmittedProjects = proposalOmittedProjects(
		updated.Graph.Graph, discovery.CandidateEntrypoints,
	)
	proposal.Analysis = &analysis
	proposal.Decision = analysis.Decision
	proposal.Reason = analysis.Reason
	documents := optimizeDiscoveryDocuments{values: map[string][]byte{
		filepath.Join(invocation.stateRelative, "discovery", "manifest.json"):           updatedManifestRaw,
		filepath.Join(invocation.stateRelative, "discovery", "graph.json"):              updated.GraphJSON,
		filepath.Join(invocation.stateRelative, "discovery", "generated-manifest.json"): updated.GeneratedJSON,
		filepath.Join(invocation.stateRelative, "discovery", "snapshot.json"):           derivedRaw,
	}}
	proposalRaw, err = json.MarshalIndent(proposal, "", "  ")
	if err != nil {
		return err
	}
	documents.values[filepath.Join(
		invocation.stateRelative, "discovery", "proposal.json",
	)] = append(proposalRaw, '\n')
	return writeOptimizeDiscoveryDocuments(invocation.repositoryRoot, documents)
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

func readOptimizeNativePortfolio(path string) (nativevolatility.Portfolio, error) {
	var portfolio nativevolatility.Portfolio
	if err := readOptimizeStrictJSON(path, optimizeMaterializationMaxManifest, &portfolio); err != nil {
		return portfolio, err
	}
	if err := nativevolatility.ValidatePortfolio(portfolio); err != nil {
		return portfolio, err
	}
	return portfolio, nil
}

func readOptimizeNativePortfolioContext(path string) (nativevolatility.PortfolioContext, error) {
	var context nativevolatility.PortfolioContext
	if err := readOptimizeStrictJSON(path, 16<<10, &context); err != nil {
		return context, err
	}
	if _, err := nativevolatility.ContextSHA256(context); err != nil {
		return context, err
	}
	return context, nil
}

func optimizeNativePortfolioContext(state optimizeState) (nativevolatility.PortfolioContext, error) {
	if state.Discovery.RepositoryID == "" ||
		!validOptimizeSHA(state.Bindings.InvocationSHA256) ||
		!validOptimizeSHA(state.Bindings.WrapperSHA256) {
		return nativevolatility.PortfolioContext{}, errors.New("complete optimize portfolio context is unavailable")
	}
	outputContractSHA := ""
	if validOptimizeNativeObservationShape(state.Discovery.NativeObservation) &&
		state.Discovery.NativeObservation.Status == optimizeNativeObservationCaptured {
		outputContractSHA = state.Discovery.NativeObservation.OutputContractSHA256
	} else {
		if state.Discovery.Status != optimizeDiscoveryComplete ||
			len(state.Discovery.Entrypoints) == 0 || len(state.Discovery.RequiredOutputs) == 0 ||
			state.Discovery.AggregatePartition == nil {
			return nativevolatility.PortfolioContext{}, errors.New("complete optimize portfolio context is unavailable")
		}
		contract := struct {
			Entrypoints      []string                     `json:"entrypoints"`
			RequiredOutputs  []string                     `json:"requiredOutputs"`
			TaskGroups       []optimizeAggregateTaskGroup `json:"taskGroups"`
			OriginalProjects int                          `json:"originalProjects"`
		}{
			Entrypoints:     append([]string(nil), state.Discovery.Entrypoints...),
			RequiredOutputs: append([]string(nil), state.Discovery.RequiredOutputs...),
			TaskGroups: append(
				[]optimizeAggregateTaskGroup(nil), state.Discovery.AggregatePartition.TaskGroups...,
			),
			OriginalProjects: state.Discovery.Graph.TotalProjects,
		}
		raw, err := json.Marshal(contract)
		if err != nil {
			return nativevolatility.PortfolioContext{}, err
		}
		digest := sha256.Sum256(raw)
		outputContractSHA = hex.EncodeToString(digest[:])
	}
	context := nativevolatility.PortfolioContext{
		RepositoryScopeSHA256: optimizePortfolioRepositoryScope(state.Discovery.RepositoryID),
		WorkflowSHA256:        state.Bindings.InvocationSHA256,
		WrapperSHA256:         state.Bindings.WrapperSHA256,
		OutputContractSHA256:  outputContractSHA,
	}
	if _, err := nativevolatility.ContextSHA256(context); err != nil {
		return nativevolatility.PortfolioContext{}, err
	}
	return context, nil
}

func readOptimizeStrictJSON(path string, maximum int64, destination any) error {
	raw, err := os.ReadFile(path)
	if err != nil || len(raw) == 0 || int64(len(raw)) > maximum {
		return errors.New("native volatility evidence is unavailable")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("native volatility evidence has trailing JSON")
	}
	return nil
}

func loadOptimizeNativeQuarantine(
	invocation optimizeInvocation,
	discovery optimizeDiscoveryResult,
) (optimizeNativeQuarantinePlan, error) {
	materialization := discovery.Materialization
	if materialization.QuarantineFile == "" || materialization.QuarantineSHA256 == "" {
		return optimizeNativeQuarantinePlan{}, errors.New("native volatility quarantine is unavailable")
	}
	raw, err := os.ReadFile(filepath.Join(
		invocation.repositoryRoot, filepath.FromSlash(materialization.QuarantineFile),
	))
	if err != nil || len(raw) == 0 || len(raw) > optimizeMaterializationMaxManifest {
		return optimizeNativeQuarantinePlan{}, errors.New("native volatility quarantine file is unavailable")
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != materialization.QuarantineSHA256 {
		return optimizeNativeQuarantinePlan{}, errors.New("native volatility quarantine digest drifted")
	}
	var envelope struct {
		SchemaVersion string `json:"schemaVersion"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return optimizeNativeQuarantinePlan{}, err
	}
	var plan optimizeNativeQuarantinePlan
	switch envelope.SchemaVersion {
	case nativevolatility.ResultSchema:
		var result nativevolatility.Result
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&result); err != nil || nativevolatility.ValidateResult(result) != nil {
			return plan, errors.New("native volatility quarantine is invalid")
		}
		plan = optimizeNativeQuarantinePlan{
			BindingSHA256:        result.BindingSHA256,
			QuarantinedProducers: result.QuarantinedProducers,
			QuarantinedOutputs:   result.QuarantinedOutputs,
			TransportedOutputs:   result.TransportedOutputs,
		}
	case optimizePortfolioEvidenceSchema:
		var evidence optimizeNativePortfolioEvidence
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&evidence); err != nil ||
			evidence.SchemaVersion != optimizePortfolioEvidenceSchema ||
			nativevolatility.ValidatePortfolioApplication(
				evidence.Application, evidence.Portfolio, evidence.Current,
			) != nil {
			return plan, errors.New("native volatility portfolio evidence is invalid")
		}
		plan = optimizeNativeQuarantinePlan{
			BindingSHA256:        evidence.Current.BindingSHA256,
			QuarantinedProducers: evidence.Application.EffectiveQuarantinedProducers,
			QuarantinedOutputs:   evidence.Application.QuarantinedOutputs,
			TransportedOutputs:   evidence.Application.TransportedOutputs,
		}
	default:
		return plan, errors.New("native volatility quarantine schema is unsupported")
	}
	if plan.BindingSHA256 != invocation.bindingSHA256 ||
		!equalOptimizeStrings(plan.QuarantinedProducers,
			intersectOptimizeStrings(discovery.CandidateEntrypoints, plan.QuarantinedProducers)) {
		return optimizeNativeQuarantinePlan{}, errors.New("native volatility quarantine binding drifted")
	}
	return plan, nil
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
