// Package configurationinput parses Gradle Configuration Cache reports and
// classifies repository-owned external configuration inputs without using
// repository, task, plugin, path, variable, or executable names as rules.
package configurationinput

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const (
	DecisionSimpleProviderExec = "SIMPLE_PROVIDER_EXEC_ELIGIBLE"
	DecisionValueSourceReview  = "VALUE_SOURCE_REVIEW_REQUIRED"
	DecisionTypedProvider      = "TYPED_PROVIDER_ELIGIBLE"
	DecisionExternalOwner      = "EXTERNAL_OR_GENERATED_OWNER"
	DecisionUnsafe             = "SIDE_EFFECTING_OR_SECRET_BEARING"
	DecisionAmbiguous          = "AMBIGUOUS_CONTROL_FLOW"
	DecisionSourceDrifted      = "SOURCE_DRIFTED"
	DecisionAlreadySupported   = "ALREADY_SUPPORTED"
	DecisionNoAction           = "NO_ACTION"
)

// ErrInvalidFacts means a fixture or future source binder omitted a mandatory
// fact. Missing facts never default to a safe correction.
var ErrInvalidFacts = errors.New("configuration-input facts are invalid")

// SourceSpan binds a declaration or operation to one-indexed source text.
type SourceSpan struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn"`
	EndLine     int `json:"endLine"`
	EndColumn   int `json:"endColumn"`
}

func (span SourceSpan) valid() bool {
	return span.StartLine > 0 && span.StartColumn > 0 && span.EndLine >= span.StartLine && span.EndColumn > 0
}

// SourceFacts are produced by a source binder after a problem report has
// identified the configuration-time call. Names and literal values are retained
// for evidence but the classifier consumes only the typed booleans and enums.
type SourceFacts struct {
	ProblemIndex         int        `json:"problemIndex"`
	SourcePath           string     `json:"sourcePath"`
	SourceSHA256         string     `json:"sourceSha256"`
	CurrentSourceSHA256  string     `json:"currentSourceSha256"`
	Language             string     `json:"language"`
	DeclarationSpan      SourceSpan `json:"declarationSpan"`
	OperationSpan        SourceSpan `json:"operationSpan"`
	OperationKind        string     `json:"operationKind"`
	Phase                string     `json:"phase"`
	SourceOwned          bool       `json:"sourceOwned"`
	ExternalPluginOwned  bool       `json:"externalPluginOwned"`
	GeneratedOrVendor    bool       `json:"generatedOrVendor"`
	BindingAmbiguous     bool       `json:"bindingAmbiguous"`
	AlreadySupported     bool       `json:"alreadySupported"`
	SideEffecting        bool       `json:"sideEffecting"`
	SecretBearing        bool       `json:"secretBearing"`
	CommandComplete      bool       `json:"commandComplete"`
	ConsumesOnlyResult   bool       `json:"consumesOnlyResult"`
	StreamsConfigured    bool       `json:"streamsConfigured"`
	ExitHandlingComplete bool       `json:"exitHandlingComplete"`
	DirectProvider       bool       `json:"directProvider"`
	BoundedValueSource   bool       `json:"boundedValueSource"`
	ProposedAPI          string     `json:"proposedApi"`
	RepositoryLabel      string     `json:"repositoryLabel,omitempty"`
	TaskLabel            string     `json:"taskLabel,omitempty"`
	ExecutableLabel      string     `json:"executableLabel,omitempty"`
}

// Row is the deterministic CINC v1 result for one strict-report problem.
type Row struct {
	ProblemIndex         int        `json:"problemIndex"`
	ProblemKind          string     `json:"problemKind"`
	ProblemMessageSHA256 string     `json:"problemMessageSha256"`
	TraceOwner           string     `json:"traceOwner"`
	SourcePath           string     `json:"sourcePath"`
	SourceSHA256         string     `json:"sourceSha256"`
	Language             string     `json:"language"`
	DeclarationSpan      SourceSpan `json:"declarationSpan"`
	OperationSpan        SourceSpan `json:"operationSpan"`
	OperationKind        string     `json:"operationKind"`
	ProposedAPI          string     `json:"proposedApi,omitempty"`
	RecipeClass          string     `json:"recipeClass,omitempty"`
	Decision             string     `json:"decision"`
	Reason               string     `json:"reason"`
}

// Classify creates one fail-closed decision. It intentionally ignores every
// label field in SourceFacts.
func Classify(problem Problem, facts SourceFacts) (Row, error) {
	if facts.ProblemIndex < 0 || facts.SourcePath == "" || !facts.DeclarationSpan.valid() || !facts.OperationSpan.valid() ||
		!validSHA256(facts.SourceSHA256) || !validSHA256(facts.CurrentSourceSHA256) || facts.Language == "" {
		return Row{}, ErrInvalidFacts
	}
	row := Row{
		ProblemIndex: facts.ProblemIndex, ProblemKind: problem.Kind,
		ProblemMessageSHA256: digest(problem.Message), TraceOwner: problem.TraceOwner,
		SourcePath: facts.SourcePath, SourceSHA256: facts.SourceSHA256, Language: facts.Language,
		DeclarationSpan: facts.DeclarationSpan, OperationSpan: facts.OperationSpan,
		OperationKind: facts.OperationKind,
	}
	decide := func(decision, reason string) (Row, error) {
		row.Decision, row.Reason = decision, reason
		return row, nil
	}
	if facts.SourceSHA256 != facts.CurrentSourceSHA256 {
		return decide(DecisionSourceDrifted, "bound source digest does not match current source")
	}
	if facts.ExternalPluginOwned || facts.GeneratedOrVendor || !facts.SourceOwned {
		return decide(DecisionExternalOwner, "correction must bind to repository-owned non-generated source")
	}
	if facts.Phase != "CONFIGURATION" || problem.Kind == ProblemOther {
		return decide(DecisionNoAction, "problem is outside the supported configuration-input class")
	}
	if facts.BindingAmbiguous || facts.OperationKind == "" || problem.Kind != facts.OperationKind {
		return decide(DecisionAmbiguous, "phase, source operation, and consumed result must bind uniquely")
	}
	if facts.AlreadySupported {
		return decide(DecisionAlreadySupported, "source already uses a supported tracked Gradle API")
	}
	if facts.SideEffecting || facts.SecretBearing || facts.StreamsConfigured {
		return decide(DecisionUnsafe, "side effects, secrets, or configured streams are outside v1")
	}
	switch problem.Kind {
	case ProblemExternalProcess:
		if facts.DirectProvider && facts.CommandComplete && facts.ConsumesOnlyResult && facts.ExitHandlingComplete && facts.ProposedAPI == "ProviderFactory.exec" {
			row.ProposedAPI = facts.ProposedAPI
			row.RecipeClass = "PROVIDER_FACTORY_EXEC_V1"
			return decide(DecisionSimpleProviderExec, "complete result-only process read has a direct tracked provider")
		}
		if facts.BoundedValueSource && facts.CommandComplete && facts.ExitHandlingComplete && facts.ProposedAPI == "ValueSource" {
			row.ProposedAPI = facts.ProposedAPI
			row.RecipeClass = "TYPED_VALUE_SOURCE_V1"
			return decide(DecisionValueSourceReview, "bounded process read requires explicit ValueSource semantics review")
		}
	case ProblemFileRead, ProblemPropertyRead:
		if facts.DirectProvider && facts.ProposedAPI != "" {
			row.ProposedAPI = facts.ProposedAPI
			if problem.Kind == ProblemFileRead {
				row.RecipeClass = "PROVIDER_FACTORY_FILE_CONTENTS_V1"
			} else {
				row.RecipeClass = "TYPED_GRADLE_PROVIDER_V1"
			}
			return decide(DecisionTypedProvider, "reported external input has a direct tracked provider")
		}
	}
	return decide(DecisionAmbiguous, "supported replacement facts are incomplete")
}

func validSHA256(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && strings.ToLower(value) == value
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", sum)
}
