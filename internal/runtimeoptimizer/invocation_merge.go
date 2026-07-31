package runtimeoptimizer

import (
	"errors"
	"path/filepath"
	"slices"
)

// MergeInvocation describes one immutable Gradle process in a CI session.
type MergeInvocation struct {
	Arguments                   []string
	RepositoryID                string
	SourceRevision              string
	WorkingDirectory            string
	WrapperDigest               string
	JDKDigest                   string
	JVMArgumentsDigest          string
	GradlePropertiesDigest      string
	SystemPropertiesDigest      string
	EnvironmentDigest           string
	CredentialsDigest           string
	InitScriptsDigest           string
	CachePolicyDigest           string
	GradleUserHomeCompatibility string
}

// InvocationMergeContract contains the model and semantic proof for one pair.
type InvocationMergeContract struct {
	Authorized                      bool
	ModelVersion                    string
	ModelDigest                     string
	SecondTransitivelyContainsFirst bool
	IntermediateConsumers           []string
	ExternalEffects                 []string
	FailureSemanticsEquivalent      bool
	RetrySemanticsEquivalent        bool
	ContinueSemanticsEquivalent     bool
	ExclusionsEquivalent            bool
	FinalizersEquivalent            bool
	OrderPreserved                  bool
	CIBarrier                       bool
	IsolatedControlPassed           bool
	ReleaseContract                 bool
}

// InvocationMergeRequest binds two invocations to an exact proof contract.
type InvocationMergeRequest struct {
	First    MergeInvocation
	Second   MergeInvocation
	Contract InvocationMergeContract
}

// InvocationMergeDecision either keeps the pair or replaces it with the second.
type InvocationMergeDecision struct {
	Original  []MergeInvocation
	Effective []MergeInvocation
	Applied   bool
	Reason    string
}

// EvaluateInvocationMerge applies an all-or-nothing semantic merge decision.
func EvaluateInvocationMerge(request InvocationMergeRequest) (InvocationMergeDecision, error) {
	decision := preserveInvocations(request, "NOT_ELIGIBLE")
	if err := validateMergeInvocation(request.First); err != nil {
		return InvocationMergeDecision{}, err
	}
	if err := validateMergeInvocation(request.Second); err != nil {
		return InvocationMergeDecision{}, err
	}
	contract := request.Contract
	if !contract.Authorized {
		decision.Reason = "NOT_AUTHORIZED"
		return decision, nil
	}
	if !identifierPattern.MatchString(contract.ModelVersion) || !validDigest(contract.ModelDigest) {
		decision.Reason = "MODEL_UNAVAILABLE"
		return decision, nil
	}
	if contract.ReleaseContract {
		decision.Reason = "RELEASE_CONTRACT"
		return decision, nil
	}
	if reason := invocationIdentityMismatch(request.First, request.Second); reason != "" {
		decision.Reason = reason
		return decision, nil
	}
	if !contract.SecondTransitivelyContainsFirst {
		decision.Reason = "TRANSITIVE_SUBSUMPTION_UNPROVEN"
		return decision, nil
	}
	if len(contract.IntermediateConsumers) != 0 {
		decision.Reason = "INTERMEDIATE_CONSUMER"
		return decision, nil
	}
	if len(contract.ExternalEffects) != 0 {
		decision.Reason = "SIDE_EFFECTS"
		return decision, nil
	}
	if contract.CIBarrier {
		decision.Reason = "CI_BARRIER"
		return decision, nil
	}
	if !contract.FailureSemanticsEquivalent {
		decision.Reason = "FAILURE_SEMANTICS"
		return decision, nil
	}
	if !contract.RetrySemanticsEquivalent {
		decision.Reason = "RETRY_SEMANTICS"
		return decision, nil
	}
	if !contract.ContinueSemanticsEquivalent {
		decision.Reason = "CONTINUE_SEMANTICS"
		return decision, nil
	}
	if !contract.ExclusionsEquivalent {
		decision.Reason = "EXCLUSIONS"
		return decision, nil
	}
	if !contract.FinalizersEquivalent {
		decision.Reason = "FINALIZERS"
		return decision, nil
	}
	if !contract.OrderPreserved {
		decision.Reason = "ORDER"
		return decision, nil
	}
	if !contract.IsolatedControlPassed {
		decision.Reason = "CONTROL_DIVERGED"
		return decision, nil
	}
	decision.Effective = []MergeInvocation{cloneMergeInvocation(request.Second)}
	decision.Applied = true
	decision.Reason = "APPLIED"
	return decision, nil
}

func validateMergeInvocation(invocation MergeInvocation) error {
	if len(invocation.Arguments) == 0 || filepath.Base(invocation.Arguments[0]) != "gradlew" ||
		!identifierPattern.MatchString(invocation.RepositoryID) || !identifierPattern.MatchString(invocation.GradleUserHomeCompatibility) ||
		!filepath.IsAbs(invocation.WorkingDirectory) || filepath.Clean(invocation.WorkingDirectory) != invocation.WorkingDirectory {
		return errors.New("evaluate invocation merge: invalid Gradle invocation")
	}
	for _, digest := range []string{
		invocation.SourceRevision, invocation.WrapperDigest, invocation.JDKDigest, invocation.JVMArgumentsDigest,
		invocation.GradlePropertiesDigest, invocation.SystemPropertiesDigest, invocation.EnvironmentDigest,
		invocation.CredentialsDigest, invocation.InitScriptsDigest, invocation.CachePolicyDigest,
	} {
		if !validDigest(digest) {
			return errors.New("evaluate invocation merge: invalid identity digest")
		}
	}
	return nil
}

func invocationIdentityMismatch(first, second MergeInvocation) string {
	switch {
	case first.RepositoryID != second.RepositoryID:
		return "REPOSITORY_MISMATCH"
	case first.SourceRevision != second.SourceRevision:
		return "REVISION_MISMATCH"
	case first.WorkingDirectory != second.WorkingDirectory:
		return "WORKING_DIRECTORY_MISMATCH"
	case first.WrapperDigest != second.WrapperDigest || first.JDKDigest != second.JDKDigest:
		return "TOOLCHAIN_MISMATCH"
	case first.JVMArgumentsDigest != second.JVMArgumentsDigest || first.GradleUserHomeCompatibility != second.GradleUserHomeCompatibility:
		return "DAEMON_OR_USER_HOME_MISMATCH"
	case first.GradlePropertiesDigest != second.GradlePropertiesDigest || first.SystemPropertiesDigest != second.SystemPropertiesDigest:
		return "GRADLE_INPUT_MISMATCH"
	case first.EnvironmentDigest != second.EnvironmentDigest || first.CredentialsDigest != second.CredentialsDigest || first.InitScriptsDigest != second.InitScriptsDigest:
		return "ENVIRONMENT_OR_CREDENTIAL_MISMATCH"
	case first.CachePolicyDigest != second.CachePolicyDigest:
		return "CACHE_POLICY_MISMATCH"
	default:
		return ""
	}
}

func preserveInvocations(request InvocationMergeRequest, reason string) InvocationMergeDecision {
	original := []MergeInvocation{cloneMergeInvocation(request.First), cloneMergeInvocation(request.Second)}
	return InvocationMergeDecision{Original: original, Effective: []MergeInvocation{cloneMergeInvocation(original[0]), cloneMergeInvocation(original[1])}, Reason: reason}
}

func cloneMergeInvocation(invocation MergeInvocation) MergeInvocation {
	invocation.Arguments = slices.Clone(invocation.Arguments)
	return invocation
}
