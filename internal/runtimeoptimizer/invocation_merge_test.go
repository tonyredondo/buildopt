package runtimeoptimizer

import (
	"reflect"
	"testing"
)

func TestInvocationMergeAppliesOnlyValidatedSubsumption(t *testing.T) {
	request := testInvocationMergeRequest()
	decision, err := EvaluateInvocationMerge(request)
	if err != nil || !decision.Applied || decision.Reason != "APPLIED" || len(decision.Effective) != 1 ||
		!reflect.DeepEqual(decision.Effective[0].Arguments, request.Second.Arguments) {
		t.Fatalf("decision = %+v/%v", decision, err)
	}
	request.Second.Arguments[1] = "mutated"
	if decision.Original[1].Arguments[1] != "assemble" || decision.Effective[0].Arguments[1] != "assemble" {
		t.Fatal("merge decision did not retain immutable argv")
	}
}

func TestInvocationMergePreservesPairForEveryUnsafeBoundary(t *testing.T) {
	tests := []struct {
		name   string
		reason string
		mutate func(*InvocationMergeRequest)
	}{
		{name: "unauthorized", reason: "NOT_AUTHORIZED", mutate: func(request *InvocationMergeRequest) { request.Contract.Authorized = false }},
		{name: "model unavailable", reason: "MODEL_UNAVAILABLE", mutate: func(request *InvocationMergeRequest) { request.Contract.ModelDigest = "" }},
		{name: "subsumption", reason: "TRANSITIVE_SUBSUMPTION_UNPROVEN", mutate: func(request *InvocationMergeRequest) { request.Contract.SecondTransitivelyContainsFirst = false }},
		{name: "failure", reason: "FAILURE_SEMANTICS", mutate: func(request *InvocationMergeRequest) { request.Contract.FailureSemanticsEquivalent = false }},
		{name: "retry", reason: "RETRY_SEMANTICS", mutate: func(request *InvocationMergeRequest) { request.Contract.RetrySemanticsEquivalent = false }},
		{name: "continue", reason: "CONTINUE_SEMANTICS", mutate: func(request *InvocationMergeRequest) { request.Contract.ContinueSemanticsEquivalent = false }},
		{name: "exclusion", reason: "EXCLUSIONS", mutate: func(request *InvocationMergeRequest) { request.Contract.ExclusionsEquivalent = false }},
		{name: "finalizer", reason: "FINALIZERS", mutate: func(request *InvocationMergeRequest) { request.Contract.FinalizersEquivalent = false }},
		{name: "ordering", reason: "ORDER", mutate: func(request *InvocationMergeRequest) { request.Contract.OrderPreserved = false }},
		{name: "consumer", reason: "INTERMEDIATE_CONSUMER", mutate: func(request *InvocationMergeRequest) {
			request.Contract.IntermediateConsumers = []string{"artifact-upload"}
		}},
		{name: "side effect", reason: "SIDE_EFFECTS", mutate: func(request *InvocationMergeRequest) { request.Contract.ExternalEffects = []string{"publish"} }},
		{name: "barrier", reason: "CI_BARRIER", mutate: func(request *InvocationMergeRequest) { request.Contract.CIBarrier = true }},
		{name: "control", reason: "CONTROL_DIVERGED", mutate: func(request *InvocationMergeRequest) { request.Contract.IsolatedControlPassed = false }},
		{name: "release", reason: "RELEASE_CONTRACT", mutate: func(request *InvocationMergeRequest) { request.Contract.ReleaseContract = true }},
		{name: "revision", reason: "REVISION_MISMATCH", mutate: func(request *InvocationMergeRequest) { request.Second.SourceRevision = "sha256:" + repeat("9", 64) }},
		{name: "workspace", reason: "WORKING_DIRECTORY_MISMATCH", mutate: func(request *InvocationMergeRequest) { request.Second.WorkingDirectory = "/workspace/other" }},
		{name: "environment", reason: "ENVIRONMENT_OR_CREDENTIAL_MISMATCH", mutate: func(request *InvocationMergeRequest) { request.Second.EnvironmentDigest = "sha256:" + repeat("8", 64) }},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			request := testInvocationMergeRequest()
			testCase.mutate(&request)
			decision, err := EvaluateInvocationMerge(request)
			if err != nil || decision.Applied || decision.Reason != testCase.reason || len(decision.Effective) != 2 {
				t.Fatalf("decision = %+v/%v", decision, err)
			}
		})
	}
}

func TestInvocationMergeRejectsMalformedInvocationIdentity(t *testing.T) {
	request := testInvocationMergeRequest()
	request.First.Arguments = []string{"gradle", "compileJava"}
	if _, err := EvaluateInvocationMerge(request); err == nil {
		t.Fatal("non-Wrapper invocation was accepted")
	}
	request = testInvocationMergeRequest()
	request.First.CachePolicyDigest = "invalid"
	if _, err := EvaluateInvocationMerge(request); err == nil {
		t.Fatal("invalid identity digest was accepted")
	}
}

func testInvocationMergeRequest() InvocationMergeRequest {
	first := testMergeInvocation([]string{"./gradlew", "compileJava"})
	second := testMergeInvocation([]string{"./gradlew", "assemble"})
	return InvocationMergeRequest{
		First:  first,
		Second: second,
		Contract: InvocationMergeContract{
			Authorized: true, ModelVersion: "gradle-entrypoints-v1", ModelDigest: "sha256:" + repeat("7", 64), SecondTransitivelyContainsFirst: true,
			FailureSemanticsEquivalent: true, RetrySemanticsEquivalent: true, ContinueSemanticsEquivalent: true,
			ExclusionsEquivalent: true, FinalizersEquivalent: true, OrderPreserved: true, IsolatedControlPassed: true,
		},
	}
}

func testMergeInvocation(arguments []string) MergeInvocation {
	return MergeInvocation{
		Arguments: arguments, RepositoryID: "repository-1", SourceRevision: "sha256:" + repeat("1", 64), WorkingDirectory: "/workspace/repository",
		WrapperDigest: "sha256:" + repeat("2", 64), JDKDigest: "sha256:" + repeat("3", 64), JVMArgumentsDigest: "sha256:" + repeat("4", 64),
		GradlePropertiesDigest: "sha256:" + repeat("5", 64), SystemPropertiesDigest: "sha256:" + repeat("6", 64),
		EnvironmentDigest: "sha256:" + repeat("a", 64), CredentialsDigest: "sha256:" + repeat("b", 64),
		InitScriptsDigest: "sha256:" + repeat("c", 64), CachePolicyDigest: "sha256:" + repeat("d", 64),
		GradleUserHomeCompatibility: "gradle-user-home-v1",
	}
}
