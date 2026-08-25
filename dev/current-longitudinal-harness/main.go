// Command current-longitudinal-harness independently validates the AF-014A
// installed-package harness result against its frozen machine contract.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
)

const (
	contractSchema = "buildopt.specs/poc-current-longitudinal-harness/v1"
	resultSchema   = "buildopt.poc/current-longitudinal-harness/v1"
)

var shaPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var revisionPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

type contract struct {
	SchemaVersion string `json:"schemaVersion"`
	WorkItem      string `json:"workItem"`
	Methodology   struct {
		Control                       string `json:"control"`
		Candidate                     string `json:"candidate"`
		LearningAttempts              int    `json:"learningAttempts"`
		ArmOrder                      string `json:"armOrder"`
		UntimedCandidateLearning      bool   `json:"untimedCandidateLearningAllowed"`
		MutableArmStateSharingAllowed bool   `json:"mutableArmStateSharingAllowed"`
		PercentagesAdded              bool   `json:"percentagesAdded"`
	} `json:"methodology"`
	Workflow struct {
		Control   []string `json:"control"`
		Candidate []string `json:"candidate"`
	} `json:"workflow"`
	RequiredScenarios []string   `json:"requiredScenarios"`
	Boundaries        boundaries `json:"boundaries"`
}

type result struct {
	SchemaVersion     string `json:"schemaVersion"`
	WorkItem          string `json:"workItem"`
	CapturedAt        string `json:"capturedAt"`
	Outcome           string `json:"outcome"`
	EvaluatedRevision string `json:"evaluatedRevision"`
	SourceArchiveSHA  string `json:"sourceArchiveSha256"`
	Contract          struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	} `json:"contract"`
	Package struct {
		Version          string `json:"version"`
		ArchiveSHA256    string `json:"archiveSha256"`
		ExecutableSHA256 string `json:"executableSha256"`
		Installed        bool   `json:"installed"`
	} `json:"package"`
	Harness struct {
		EnvironmentFingerprintInput  string   `json:"environmentFingerprintInput"`
		EnvironmentFingerprintSHA256 string   `json:"environmentFingerprintSha256"`
		ControlCheckoutSHA256        string   `json:"controlCheckoutSha256"`
		CandidateCheckoutSHA256      string   `json:"candidateCheckoutSha256"`
		SeparateCheckouts            bool     `json:"separateCheckouts"`
		SeparateGradleHomes          bool     `json:"separateGradleHomes"`
		SeparateNativeCaches         bool     `json:"separateNativeCaches"`
		SeparateDaemonRegistries     bool     `json:"separateDaemonRegistries"`
		ControlStateAbsent           bool     `json:"controlStateAbsent"`
		CandidateStatePrivate        bool     `json:"candidateStatePrivate"`
		UntimedCandidateLearning     int      `json:"untimedCandidateLearning"`
		ForwardRevisions             []string `json:"forwardRevisions"`
	} `json:"harness"`
	Workflow struct {
		Control   []string `json:"control"`
		Candidate []string `json:"candidate"`
	} `json:"workflow"`
	Learning   []observation `json:"learning"`
	Scenarios  []scenario    `json:"scenarios"`
	Boundaries boundaries    `json:"boundaries"`
}

type observation struct {
	Sequence          int         `json:"sequence"`
	Order             string      `json:"order"`
	Revision          string      `json:"revision"`
	ControlWallNS     int64       `json:"controlWallNs"`
	CandidateWallNS   int64       `json:"candidateWallNs"`
	ExactOutputs      bool        `json:"exactOutputs"`
	OutputSHA256      string      `json:"outputSha256"`
	Generation        int         `json:"generation"`
	Attempt           int         `json:"attempt"`
	Outcome           string      `json:"outcome"`
	Phase             string      `json:"phase"`
	ExecutionMode     string      `json:"executionMode"`
	SelectionStatus   string      `json:"selectionStatus"`
	SelectionSelected bool        `json:"selectionSelected"`
	Timing            phaseTiming `json:"timing"`
}

type scenario struct {
	Name              string       `json:"name"`
	Revision          string       `json:"revision"`
	ExternalWallNS    int64        `json:"externalWallNs"`
	ExactOutputs      bool         `json:"exactOutputs"`
	OutputSHA256      string       `json:"outputSha256"`
	Generation        int          `json:"generation"`
	Attempt           int          `json:"attempt"`
	Outcome           string       `json:"outcome"`
	Phase             string       `json:"phase"`
	ExecutionMode     string       `json:"executionMode"`
	SelectionSelected bool         `json:"selectionSelected"`
	StateChanged      bool         `json:"stateChanged"`
	BypassRemoved     bool         `json:"bypassRemoved"`
	Timing            *phaseTiming `json:"timing"`
}

type phaseTiming struct {
	PreExecutionNS    int64 `json:"preExecutionNs"`
	GradleExecutionNS int64 `json:"gradleExecutionNs"`
	FinalizationNS    int64 `json:"finalizationNs"`
	UnattributedNS    int64 `json:"unattributedNs"`
	TotalNS           int64 `json:"totalNs"`
	Diagnostics       struct {
		GradleSetupNS        int64 `json:"gradleSetupNs"`
		MatchingNS           int64 `json:"matchingNs"`
		LocalStateNS         int64 `json:"localStateNs"`
		CentralStateNS       int64 `json:"centralStateNs"`
		MaterializationNS    int64 `json:"materializationNs"`
		OutputVerificationNS int64 `json:"outputVerificationNs"`
		DiscoveryLearningNS  int64 `json:"discoveryLearningNs"`
	} `json:"diagnostics"`
}

type boundaries struct {
	ProofOfConcept        bool   `json:"proofOfConcept"`
	PerformanceClaim      bool   `json:"performanceClaim"`
	ProductionAuthorized  bool   `json:"productionAuthorized"`
	SoakRequired          bool   `json:"soakRequired"`
	DesignPartnerRequired bool   `json:"designPartnerRequired"`
	TestOptimization      string `json:"testOptimization"`
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: current-longitudinal-harness CONTRACT_JSON RESULT_JSON")
		os.Exit(64)
	}
	contractRaw, err := os.ReadFile(os.Args[1])
	if err != nil {
		fatal(err)
	}
	resultRaw, err := os.ReadFile(os.Args[2])
	if err != nil {
		fatal(err)
	}
	var spec contract
	if err := decodeStrict(contractRaw, &spec); err != nil {
		fatal(err)
	}
	var report result
	if err := decodeStrict(resultRaw, &report); err != nil {
		fatal(err)
	}
	if err := validate(contractRaw, spec, report); err != nil {
		fatal(err)
	}
	fmt.Printf("current installed longitudinal harness: %d timed learning pairs and four exact scenarios\n", len(report.Learning))
}

func validate(contractRaw []byte, spec contract, report result) error {
	if spec.SchemaVersion != contractSchema || spec.WorkItem != "AF-014A" ||
		spec.Methodology.Control != "OPTIMIZED_NATIVE_GRADLE" ||
		spec.Methodology.Candidate != "CURRENT_INSTALLED_BUILDOPT_OPTIMIZE" ||
		spec.Methodology.LearningAttempts != 18 ||
		spec.Methodology.ArmOrder != "CONTROL_FIRST_THEN_ALTERNATE" ||
		spec.Methodology.UntimedCandidateLearning ||
		spec.Methodology.MutableArmStateSharingAllowed || spec.Methodology.PercentagesAdded ||
		len(spec.Workflow.Control) == 0 || len(spec.Workflow.Candidate) == 0 ||
		!validBoundaries(spec.Boundaries) {
		return errors.New("invalid AF-014A contract")
	}
	contractDigest := sha256.Sum256(contractRaw)
	if report.SchemaVersion != resultSchema || report.WorkItem != spec.WorkItem ||
		report.Outcome != "CURRENT_LONGITUDINAL_HARNESS_READY" ||
		report.Contract.Path != "specs/poc-current-longitudinal-harness-v1.json" ||
		report.Contract.SHA256 != hex.EncodeToString(contractDigest[:]) ||
		!validRevision(report.EvaluatedRevision) || !validSHA(report.SourceArchiveSHA) ||
		!validSHA(report.Package.ArchiveSHA256) || !validSHA(report.Package.ExecutableSHA256) ||
		!report.Package.Installed || report.Package.Version == "" ||
		!equalStrings(report.Workflow.Control, spec.Workflow.Control) ||
		!equalStrings(report.Workflow.Candidate, spec.Workflow.Candidate) ||
		report.Boundaries != spec.Boundaries {
		return errors.New("result identity does not match the contract")
	}
	harness := report.Harness
	fingerprint := sha256.Sum256([]byte(harness.EnvironmentFingerprintInput))
	if harness.EnvironmentFingerprintSHA256 != hex.EncodeToString(fingerprint[:]) ||
		!validSHA(harness.ControlCheckoutSHA256) || !validSHA(harness.CandidateCheckoutSHA256) ||
		harness.ControlCheckoutSHA256 == harness.CandidateCheckoutSHA256 ||
		!harness.SeparateCheckouts || !harness.SeparateGradleHomes ||
		!harness.SeparateNativeCaches || !harness.SeparateDaemonRegistries ||
		!harness.ControlStateAbsent || !harness.CandidateStatePrivate ||
		harness.UntimedCandidateLearning != 0 || len(harness.ForwardRevisions) != 2 ||
		!validRevision(harness.ForwardRevisions[0]) || !validRevision(harness.ForwardRevisions[1]) ||
		harness.ForwardRevisions[0] == harness.ForwardRevisions[1] {
		return errors.New("mutable arm isolation or forward-only revision evidence is invalid")
	}
	if len(report.Learning) != spec.Methodology.LearningAttempts {
		return errors.New("learning sequence length does not match the contract")
	}
	for index, item := range report.Learning {
		expectedOrder := "CONTROL_FIRST"
		if index%2 == 1 {
			expectedOrder = "CANDIDATE_FIRST"
		}
		if item.Sequence != index+1 || item.Attempt != index+1 || item.Generation != 1 ||
			item.Order != expectedOrder || !validRevision(item.Revision) ||
			item.Revision != harness.ForwardRevisions[0] ||
			item.ControlWallNS <= 0 || item.CandidateWallNS <= 0 || !item.ExactOutputs ||
			!validSHA(item.OutputSHA256) || item.Outcome == "" || item.Phase == "" ||
			item.ExecutionMode == "" || item.SelectionStatus == "" ||
			item.CandidateWallNS < item.Timing.TotalNS {
			return fmt.Errorf("invalid learning observation %d", index+1)
		}
		if err := validateTiming(item.Timing); err != nil {
			return fmt.Errorf("learning observation %d: %w", index+1, err)
		}
		if index == len(report.Learning)-1 &&
			(!item.SelectionSelected || item.ExecutionMode != "SELECTIVE_PROFILE") {
			return errors.New("final learning observation is not the first selected replay")
		}
	}
	if len(report.Scenarios) != len(spec.RequiredScenarios) {
		return errors.New("scenario count does not match the contract")
	}
	for index, expected := range spec.RequiredScenarios {
		item := report.Scenarios[index]
		if item.Name != expected || item.ExternalWallNS <= 0 || !item.ExactOutputs ||
			!validSHA(item.OutputSHA256) || !validRevision(item.Revision) {
			return fmt.Errorf("invalid %s scenario", expected)
		}
		switch expected {
		case "CONTROL":
			if item.Timing != nil || item.StateChanged || item.Generation != 0 || item.Attempt != 0 {
				return errors.New("control scenario contains product state")
			}
		case "SELECTED":
			if item.Timing == nil || !item.SelectionSelected || item.ExecutionMode != "SELECTIVE_PROFILE" ||
				item.Generation != 1 || item.Attempt != spec.Methodology.LearningAttempts ||
				item.Revision != harness.ForwardRevisions[0] || !item.StateChanged {
				return errors.New("selected scenario is invalid")
			}
		case "NATIVE_RETAINED":
			if item.Timing == nil || item.SelectionSelected || item.Outcome != "NATIVE_RETAINED" ||
				item.ExecutionMode != "OPTIMIZED_NATIVE" || item.Generation != 2 || item.Attempt != 1 ||
				item.Revision != harness.ForwardRevisions[1] || !item.StateChanged {
				return errors.New("native-retained scenario is invalid")
			}
		case "BYPASS":
			if item.Timing != nil || item.StateChanged || !item.BypassRemoved ||
				item.Revision != harness.ForwardRevisions[1] {
				return errors.New("bypass scenario is invalid")
			}
		default:
			return fmt.Errorf("unknown scenario %q", expected)
		}
		if item.Timing != nil {
			if item.ExternalWallNS < item.Timing.TotalNS {
				return fmt.Errorf("%s internal timing exceeds external wall", expected)
			}
			if err := validateTiming(*item.Timing); err != nil {
				return fmt.Errorf("%s scenario: %w", expected, err)
			}
		}
	}
	return nil
}

func validateTiming(timing phaseTiming) error {
	if timing.PreExecutionNS < 0 || timing.GradleExecutionNS < 0 ||
		timing.FinalizationNS < 0 || timing.UnattributedNS < 0 || timing.TotalNS <= 0 ||
		timing.PreExecutionNS+timing.GradleExecutionNS+timing.FinalizationNS+
			timing.UnattributedNS != timing.TotalNS {
		return errors.New("top-level phase timing does not reconcile")
	}
	diagnostics := timing.Diagnostics
	if diagnostics.GradleSetupNS < 0 || diagnostics.MatchingNS < 0 ||
		diagnostics.LocalStateNS <= 0 || diagnostics.CentralStateNS < 0 ||
		diagnostics.MaterializationNS < 0 || diagnostics.OutputVerificationNS < 0 ||
		diagnostics.DiscoveryLearningNS < 0 {
		return errors.New("nested timing diagnostics are invalid")
	}
	return nil
}

func validBoundaries(value boundaries) bool {
	return value.ProofOfConcept && !value.PerformanceClaim && !value.ProductionAuthorized &&
		!value.SoakRequired && !value.DesignPartnerRequired && value.TestOptimization == "OUT_OF_SCOPE"
}

func validSHA(value string) bool { return shaPattern.MatchString(value) }

func validRevision(value string) bool { return revisionPattern.MatchString(value) }

func equalStrings(first, second []string) bool {
	if len(first) != len(second) {
		return false
	}
	for index := range first {
		if first[index] != second[index] {
			return false
		}
	}
	return true
}

func decodeStrict(raw []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("JSON contains trailing data")
	}
	return nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
