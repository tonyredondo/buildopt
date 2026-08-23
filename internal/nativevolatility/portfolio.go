package nativevolatility

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
)

const (
	PortfolioSchema            = "buildopt.poc/native-volatility-portfolio/v1"
	PortfolioApplicationSchema = "buildopt.poc/native-volatility-portfolio-application/v1"
	PortfolioPreflightSchema   = "buildopt.poc/native-volatility-portfolio-preflight/v1"

	PortfolioDecisionApplied    = "PORTFOLIO_APPLIED"
	PortfolioDecisionCompatible = "COMPATIBLE"
	PortfolioDecisionRetained   = "NATIVE_RETAINED"
)

// PortfolioContext binds producer-volatility learning to one repository,
// workflow, Wrapper and owner-reviewed output contract without binding it to a
// single source revision. Every field is a digest supplied by the owning
// discovery layer; raw repository identities and local paths are never stored.
type PortfolioContext struct {
	RepositoryScopeSHA256 string `json:"repositoryScopeSha256"`
	WorkflowSHA256        string `json:"workflowSha256"`
	WrapperSHA256         string `json:"wrapperSha256"`
	OutputContractSHA256  string `json:"outputContractSha256"`
}

// PortfolioObservation records only producers proven volatile by two
// independent optimized-native observations. Output bytes remain revision
// bound in Result and are deliberately not copied into the portfolio.
type PortfolioObservation struct {
	RevisionSHA256       string   `json:"revisionSha256"`
	ResultSHA256         string   `json:"resultSha256"`
	QuarantinedProducers []string `json:"quarantinedProducers"`
	AuthoritativeNative  bool     `json:"authoritativeNative"`
	PerformanceClaimed   bool     `json:"performanceClaimed"`
}

// Portfolio accumulates producer volatility across compatible source
// revisions. It is POC evidence and cannot authorize production behavior.
type Portfolio struct {
	SchemaVersion        string                 `json:"schemaVersion"`
	ContextSHA256        string                 `json:"contextSha256"`
	Observations         []PortfolioObservation `json:"observations"`
	QuarantinedProducers []string               `json:"quarantinedProducers"`
	ProductionAuthorized bool                   `json:"productionAuthorized"`
	TestOptimization     string                 `json:"testOptimization"`
}

// PortfolioApplication partitions one current revision's exact output
// inventory using current and previously learned volatile producers. All entry
// hashes come from the current result; historical output hashes are never
// reused.
type PortfolioApplication struct {
	SchemaVersion                 string   `json:"schemaVersion"`
	Decision                      string   `json:"decision"`
	ContextSHA256                 string   `json:"contextSha256"`
	RevisionSHA256                string   `json:"revisionSha256"`
	CurrentBindingSHA256          string   `json:"currentBindingSha256"`
	CurrentResultSHA256           string   `json:"currentResultSha256"`
	PortfolioSHA256               string   `json:"portfolioSha256"`
	PortfolioQuarantinedProducers []string `json:"portfolioQuarantinedProducers"`
	EffectiveQuarantinedProducers []string `json:"effectiveQuarantinedProducers"`
	QuarantinedOutputs            []Entry  `json:"quarantinedOutputs"`
	TransportedOutputs            []Entry  `json:"transportedOutputs"`
	TransportSHA256               string   `json:"transportSha256"`
	ProductionAuthorized          bool     `json:"productionAuthorized"`
	TestOptimization              string   `json:"testOptimization"`
}

// PortfolioPreflight decides whether a learned portfolio may justify an
// independent native observation for the current revision. It performs no
// build work and grants no transport or production authority.
type PortfolioPreflight struct {
	SchemaVersion                          string           `json:"schemaVersion"`
	Decision                               string           `json:"decision"`
	Reason                                 string           `json:"reason"`
	LearnedContext                         PortfolioContext `json:"learnedContext"`
	CurrentContext                         PortfolioContext `json:"currentContext"`
	LearnedContextSHA256                   string           `json:"learnedContextSha256"`
	CurrentContextSHA256                   string           `json:"currentContextSha256"`
	DriftedBindings                        []string         `json:"driftedBindings"`
	IndependentNativeObservationAuthorized bool             `json:"independentNativeObservationAuthorized"`
	ProductionAuthorized                   bool             `json:"productionAuthorized"`
	TestOptimization                       string           `json:"testOptimization"`
}

// PreflightPortfolio compares every stable portfolio binding before the
// caller starts an independent native observation. A drift decision is a
// successful fail-closed result rather than an evaluation error.
func PreflightPortfolio(
	portfolio Portfolio,
	learned PortfolioContext,
	current PortfolioContext,
) (PortfolioPreflight, error) {
	if err := ValidatePortfolio(portfolio); err != nil {
		return PortfolioPreflight{}, errors.New("native volatility portfolio is invalid")
	}
	learnedSHA, err := ContextSHA256(learned)
	if err != nil || learnedSHA != portfolio.ContextSHA256 {
		return PortfolioPreflight{}, errors.New("native volatility learned context is invalid")
	}
	currentSHA, err := ContextSHA256(current)
	if err != nil {
		return PortfolioPreflight{}, errors.New("native volatility current context is invalid")
	}
	drifted := portfolioDriftedBindings(learned, current)
	result := PortfolioPreflight{
		SchemaVersion:                          PortfolioPreflightSchema,
		Decision:                               PortfolioDecisionCompatible,
		Reason:                                 "PORTFOLIO_CONTEXT_COMPATIBLE",
		LearnedContext:                         learned,
		CurrentContext:                         current,
		LearnedContextSHA256:                   learnedSHA,
		CurrentContextSHA256:                   currentSHA,
		DriftedBindings:                        drifted,
		IndependentNativeObservationAuthorized: true,
		ProductionAuthorized:                   false,
		TestOptimization:                       "OUT_OF_SCOPE",
	}
	if len(drifted) > 0 {
		result.Decision = PortfolioDecisionRetained
		result.Reason = "PORTFOLIO_CONTEXT_DRIFT"
		result.IndependentNativeObservationAuthorized = false
	}
	if err := ValidatePortfolioPreflight(result, portfolio); err != nil {
		return PortfolioPreflight{}, err
	}
	return result, nil
}

// ValidatePortfolioPreflight rejects incomplete, inconsistent or
// authority-expanding compatibility decisions.
func ValidatePortfolioPreflight(result PortfolioPreflight, portfolio Portfolio) error {
	if result.SchemaVersion != PortfolioPreflightSchema || result.ProductionAuthorized ||
		result.TestOptimization != "OUT_OF_SCOPE" || ValidatePortfolio(portfolio) != nil {
		return errors.New("native volatility portfolio preflight is invalid")
	}
	learnedSHA, learnedErr := ContextSHA256(result.LearnedContext)
	currentSHA, currentErr := ContextSHA256(result.CurrentContext)
	drifted := portfolioDriftedBindings(result.LearnedContext, result.CurrentContext)
	if learnedErr != nil || currentErr != nil || learnedSHA != portfolio.ContextSHA256 ||
		result.LearnedContextSHA256 != learnedSHA || result.CurrentContextSHA256 != currentSHA ||
		!equalStrings(result.DriftedBindings, drifted) {
		return errors.New("native volatility portfolio preflight binding drifted")
	}
	if len(drifted) == 0 {
		if result.Decision != PortfolioDecisionCompatible ||
			result.Reason != "PORTFOLIO_CONTEXT_COMPATIBLE" ||
			!result.IndependentNativeObservationAuthorized {
			return errors.New("compatible native volatility preflight is invalid")
		}
		return nil
	}
	if result.Decision != PortfolioDecisionRetained || result.Reason != "PORTFOLIO_CONTEXT_DRIFT" ||
		result.IndependentNativeObservationAuthorized {
		return errors.New("drifted native volatility preflight is invalid")
	}
	return nil
}

func portfolioDriftedBindings(learned, current PortfolioContext) []string {
	drifted := []string{}
	if learned.RepositoryScopeSHA256 != current.RepositoryScopeSHA256 {
		drifted = append(drifted, "REPOSITORY_SCOPE_SHA256")
	}
	if learned.WorkflowSHA256 != current.WorkflowSHA256 {
		drifted = append(drifted, "WORKFLOW_SHA256")
	}
	if learned.WrapperSHA256 != current.WrapperSHA256 {
		drifted = append(drifted, "GRADLE_WRAPPER_SHA256")
	}
	if learned.OutputContractSHA256 != current.OutputContractSHA256 {
		drifted = append(drifted, "OUTPUT_CONTRACT_SHA256")
	}
	return drifted
}

// ContextSHA256 returns the stable cross-revision portfolio binding.
func ContextSHA256(context PortfolioContext) (string, error) {
	if !validSHA(context.RepositoryScopeSHA256) || !validSHA(context.WorkflowSHA256) ||
		!validSHA(context.WrapperSHA256) || !validSHA(context.OutputContractSHA256) {
		return "", errors.New("native volatility portfolio context is invalid")
	}
	return digestStrings(
		"buildopt-native-volatility-portfolio-context-v1",
		context.RepositoryScopeSHA256,
		context.WorkflowSHA256,
		context.WrapperSHA256,
		context.OutputContractSHA256,
	), nil
}

// LearnPortfolio adds one diagnostic native observation. The input result must
// already be a valid producer-atomic partition with at least one volatile
// producer. Replaying the same revision is idempotent only when the result is
// byte-identical.
func LearnPortfolio(
	portfolio Portfolio,
	context PortfolioContext,
	revision string,
	result Result,
	authoritativeNative bool,
) (Portfolio, error) {
	contextSHA, err := ContextSHA256(context)
	if err != nil || !validSHA(revision) || !authoritativeNative {
		return Portfolio{}, errors.New("native volatility learning is not authoritative")
	}
	if err := ValidateResult(result); err != nil || len(result.QuarantinedProducers) == 0 {
		return Portfolio{}, errors.New("native volatility learning result is invalid")
	}
	resultSHA, err := resultSHA256(result)
	if err != nil {
		return Portfolio{}, err
	}
	if portfolio.SchemaVersion == "" {
		portfolio = Portfolio{
			SchemaVersion: PortfolioSchema, ContextSHA256: contextSHA,
			Observations: []PortfolioObservation{}, QuarantinedProducers: []string{},
			ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
		}
	} else if err := ValidatePortfolio(portfolio); err != nil || portfolio.ContextSHA256 != contextSHA {
		return Portfolio{}, errors.New("native volatility portfolio context drifted")
	}
	for _, observation := range portfolio.Observations {
		if observation.RevisionSHA256 != revision {
			continue
		}
		if observation.ResultSHA256 == resultSHA {
			return portfolio, nil
		}
		return Portfolio{}, errors.New("native volatility revision evidence drifted")
	}
	portfolio.Observations = append(portfolio.Observations, PortfolioObservation{
		RevisionSHA256: revision, ResultSHA256: resultSHA,
		QuarantinedProducers: append([]string(nil), result.QuarantinedProducers...),
		AuthoritativeNative:  true, PerformanceClaimed: false,
	})
	portfolio.QuarantinedProducers = mergeStrings(
		portfolio.QuarantinedProducers, result.QuarantinedProducers,
	)
	if err := ValidatePortfolio(portfolio); err != nil {
		return Portfolio{}, err
	}
	return portfolio, nil
}

// ApplyPortfolio uses only the current revision's exact result entries and
// expands its local-rebuild partition with compatible historical producers.
func ApplyPortfolio(
	portfolio Portfolio,
	context PortfolioContext,
	revision string,
	current Result,
) (PortfolioApplication, error) {
	contextSHA, err := ContextSHA256(context)
	if err != nil || !validSHA(revision) || ValidatePortfolio(portfolio) != nil ||
		portfolio.ContextSHA256 != contextSHA {
		return PortfolioApplication{}, errors.New("native volatility portfolio application context drifted")
	}
	if err := ValidateResult(current); err != nil {
		return PortfolioApplication{}, errors.New("current native volatility result is invalid")
	}
	currentSHA, err := resultSHA256(current)
	if err != nil {
		return PortfolioApplication{}, err
	}
	portfolioSHA, err := portfolioSHA256(portfolio)
	if err != nil {
		return PortfolioApplication{}, err
	}
	effectiveProducers := mergeStrings(
		portfolio.QuarantinedProducers, current.QuarantinedProducers,
	)
	producerSet := make(map[string]bool, len(effectiveProducers))
	matchedPortfolio := make(map[string]bool, len(portfolio.QuarantinedProducers))
	for _, producer := range effectiveProducers {
		producerSet[producer] = true
	}
	entries := append([]Entry(nil), current.QuarantinedOutputs...)
	entries = append(entries, current.TransportedOutputs...)
	sortEntries(entries)
	application := PortfolioApplication{
		SchemaVersion: PortfolioApplicationSchema, Decision: PortfolioDecisionApplied,
		ContextSHA256: contextSHA, RevisionSHA256: revision,
		CurrentBindingSHA256: current.BindingSHA256, CurrentResultSHA256: currentSHA,
		PortfolioSHA256:               portfolioSHA,
		PortfolioQuarantinedProducers: append([]string(nil), portfolio.QuarantinedProducers...),
		EffectiveQuarantinedProducers: effectiveProducers,
		QuarantinedOutputs:            []Entry{}, TransportedOutputs: []Entry{},
		ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
	}
	for _, entry := range entries {
		for _, producer := range entry.ProducerTasks {
			if containsString(portfolio.QuarantinedProducers, producer) {
				matchedPortfolio[producer] = true
			}
		}
		for _, producer := range entry.ProducerLineage {
			if containsString(portfolio.QuarantinedProducers, producer) {
				matchedPortfolio[producer] = true
			}
		}
		if entryIntersects(entry, producerSet) {
			application.QuarantinedOutputs = append(application.QuarantinedOutputs, cloneEntry(entry))
		} else {
			application.TransportedOutputs = append(application.TransportedOutputs, cloneEntry(entry))
		}
	}
	for _, producer := range portfolio.QuarantinedProducers {
		if !matchedPortfolio[producer] {
			return PortfolioApplication{}, errors.New("portfolio producer is absent from the current output contract")
		}
	}
	if len(application.TransportedOutputs) == 0 {
		return PortfolioApplication{}, errors.New("portfolio leaves no transportable outputs")
	}
	application.TransportSHA256 = entriesDigest(application.TransportedOutputs)
	if err := ValidatePortfolioApplication(application, portfolio, current); err != nil {
		return PortfolioApplication{}, err
	}
	return application, nil
}

// VerifyPortfolioCandidate proves exact current-revision transport and local
// rebuild coverage for an applied cross-revision portfolio.
func VerifyPortfolioCandidate(
	application PortfolioApplication,
	portfolio Portfolio,
	current Result,
	reused Observation,
	rebuilt Observation,
) error {
	if err := ValidatePortfolioApplication(application, portfolio, current); err != nil {
		return err
	}
	if reused.BindingSHA256 != current.BindingSHA256 || rebuilt.BindingSHA256 != current.BindingSHA256 {
		return errors.New("portfolio candidate output binding mismatch")
	}
	reusedEntries, _, err := validateObservation(reused, true)
	if err != nil {
		return err
	}
	rebuiltEntries, _, err := validateObservation(rebuilt, true)
	if err != nil {
		return err
	}
	if err := verifyExactSet(application.TransportedOutputs, reusedEntries); err != nil {
		return err
	}
	return verifyRebuiltSet(
		application.QuarantinedOutputs, rebuiltEntries, application.EffectiveQuarantinedProducers,
	)
}

// ValidatePortfolio checks canonical, diagnostic-only accumulated evidence.
func ValidatePortfolio(portfolio Portfolio) error {
	if portfolio.SchemaVersion != PortfolioSchema || !validSHA(portfolio.ContextSHA256) ||
		portfolio.ProductionAuthorized || portfolio.TestOptimization != "OUT_OF_SCOPE" ||
		len(portfolio.Observations) == 0 || !uniqueNonEmpty(portfolio.QuarantinedProducers) {
		return errors.New("native volatility portfolio is invalid")
	}
	observedRevisions := map[string]bool{}
	union := []string{}
	for _, observation := range portfolio.Observations {
		if !validSHA(observation.RevisionSHA256) || !validSHA(observation.ResultSHA256) ||
			!observation.AuthoritativeNative || observation.PerformanceClaimed ||
			!uniqueNonEmpty(observation.QuarantinedProducers) ||
			observedRevisions[observation.RevisionSHA256] {
			return errors.New("native volatility portfolio observation is invalid")
		}
		observedRevisions[observation.RevisionSHA256] = true
		union = mergeStrings(union, observation.QuarantinedProducers)
	}
	if !equalStrings(union, portfolio.QuarantinedProducers) {
		return errors.New("native volatility portfolio producer union drifted")
	}
	return nil
}

// ValidatePortfolioApplication binds an effective partition to the exact
// portfolio and current native result that produced it.
func ValidatePortfolioApplication(
	application PortfolioApplication,
	portfolio Portfolio,
	current Result,
) error {
	if application.SchemaVersion != PortfolioApplicationSchema ||
		application.Decision != PortfolioDecisionApplied || application.ProductionAuthorized ||
		application.TestOptimization != "OUT_OF_SCOPE" || !validSHA(application.ContextSHA256) ||
		!validSHA(application.RevisionSHA256) || !validSHA(application.CurrentBindingSHA256) ||
		!validSHA(application.CurrentResultSHA256) || !validSHA(application.PortfolioSHA256) ||
		!uniqueNonEmpty(application.PortfolioQuarantinedProducers) ||
		!uniqueNonEmpty(application.EffectiveQuarantinedProducers) ||
		len(application.QuarantinedOutputs) == 0 || len(application.TransportedOutputs) == 0 {
		return errors.New("native volatility portfolio application is invalid")
	}
	if err := ValidatePortfolio(portfolio); err != nil || ValidateResult(current) != nil {
		return errors.New("native volatility portfolio application evidence is invalid")
	}
	currentSHA, _ := resultSHA256(current)
	portfolioSHA, _ := portfolioSHA256(portfolio)
	if application.ContextSHA256 != portfolio.ContextSHA256 ||
		application.CurrentBindingSHA256 != current.BindingSHA256 ||
		application.CurrentResultSHA256 != currentSHA || application.PortfolioSHA256 != portfolioSHA ||
		!equalStrings(application.PortfolioQuarantinedProducers, portfolio.QuarantinedProducers) ||
		!equalStrings(application.EffectiveQuarantinedProducers,
			mergeStrings(portfolio.QuarantinedProducers, current.QuarantinedProducers)) {
		return errors.New("native volatility portfolio application binding drifted")
	}
	if application.TransportSHA256 != entriesDigest(application.TransportedOutputs) {
		return errors.New("native volatility portfolio transport plan drifted")
	}
	all := append([]Entry(nil), application.QuarantinedOutputs...)
	all = append(all, application.TransportedOutputs...)
	currentAll := append([]Entry(nil), current.QuarantinedOutputs...)
	currentAll = append(currentAll, current.TransportedOutputs...)
	sortEntries(all)
	sortEntries(currentAll)
	if entriesDigest(all) != entriesDigest(currentAll) {
		return errors.New("native volatility portfolio output universe drifted")
	}
	producerSet := make(map[string]bool, len(application.EffectiveQuarantinedProducers))
	for _, producer := range application.EffectiveQuarantinedProducers {
		producerSet[producer] = true
	}
	for _, entry := range application.QuarantinedOutputs {
		if !entryIntersects(entry, producerSet) {
			return errors.New("portfolio quarantine contains an unrelated output")
		}
	}
	for _, entry := range application.TransportedOutputs {
		if entryIntersects(entry, producerSet) {
			return errors.New("portfolio transported output uses a quarantined producer")
		}
	}
	return nil
}

func resultSHA256(result Result) (string, error) {
	if err := ValidateResult(result); err != nil {
		return "", err
	}
	return jsonSHA256(result)
}

func portfolioSHA256(portfolio Portfolio) (string, error) {
	if err := ValidatePortfolio(portfolio); err != nil {
		return "", err
	}
	return jsonSHA256(portfolio)
}

func jsonSHA256(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}

func digestStrings(values ...string) string {
	hash := sha256.New()
	for _, value := range values {
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(value))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func mergeStrings(left, right []string) []string {
	values := make(map[string]bool, len(left)+len(right))
	for _, value := range append(append([]string(nil), left...), right...) {
		values[value] = true
	}
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func containsString(values []string, expected string) bool {
	index := sort.SearchStrings(values, expected)
	return index < len(values) && values[index] == expected
}
