// Package testkit applies the committed VRH-002 semantic fixture matrix.
// It is shared only by conformance tests and the deterministic evidence tool.
package testkit

import (
	"fmt"

	"github.com/tonyredondo/buildopt/internal/requesthit"
)

// Matrix is the closed semantic mutation inventory consumed by conformance.
type Matrix struct {
	SchemaVersion string `json:"schemaVersion"`
	Cases         []Case `json:"cases"`
}

// Case expects one mutation to produce exactly one typed retention reason.
type Case struct {
	Name           string            `json:"name"`
	Mutation       string            `json:"mutation"`
	ExpectedReason requesthit.Reason `json:"expectedReason"`
}

// Apply mutates an otherwise matching record/probe pair in exactly one way.
func Apply(record *requesthit.SafetyRecord, probe *requesthit.Probe, mutation string) error {
	changed := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	switch mutation {
	case "ARGUMENT_UNAVAILABLE":
		probe.Request.ArgumentsSHA256 = ""
	case "ARGUMENT_DRIFT":
		probe.Request.ArgumentsSHA256 = changed
	case "CWD_DRIFT":
		probe.Request.WorkingDirectory = "subproject"
	case "REPOSITORY_DRIFT":
		probe.Request.RepositoryIdentitySHA256 = changed
	case "WRAPPER_DRIFT":
		probe.Execution.WrapperSHA256 = changed
	case "GRADLE_DRIFT":
		probe.Execution.GradleVersion = "8.14.3"
	case "JDK_DRIFT":
		probe.Execution.JDKVersion = "17.0.16+8"
	case "ENVIRONMENT_DRIFT":
		probe.Execution.SafeEnvironmentSHA256 = changed
	case "GRAPH_DRIFT":
		probe.Execution.RequestGraphSHA256 = changed
	case "TASK_IMPLEMENTATION_DRIFT":
		probe.Execution.TaskImplementationSHA256 = changed
	case "BUILD_LOGIC_DRIFT":
		probe.Execution.BuildLogicSHA256 = changed
	case "REPOSITORY_INPUTS_INCOMPLETE":
		probe.Inputs.RepositoryInputsComplete = false
	case "REPOSITORY_INPUT_DRIFT":
		probe.Inputs.RepositoryManifestSHA256 = changed
	case "EXTERNAL_INPUTS_INCOMPLETE":
		probe.Inputs.ExternalInputsComplete = false
	case "EXTERNAL_INPUT_MISSING":
		probe.Inputs.ExternalInputs[0].Present = false
	case "EXTERNAL_INPUT_DRIFT":
		probe.Inputs.ExternalInputs[0].SHA256 = changed
	case "OUTPUT_CONTRACT_INCOMPLETE":
		record.Outputs.Complete = false
	case "OUTPUT_CONTRACT_DRIFT":
		probe.Outputs = probe.Outputs[:1]
	case "UNTRACKED_OUTPUT":
		record.Outputs.States[0].Tracked = false
	case "OUTPUT_MISSING":
		probe.Outputs[0].MaterializationAvailable = false
		probe.Outputs[0].MaterializationRef = ""
		probe.Outputs[0].MaterializationSHA256 = ""
	case "OUTPUT_ALTERED":
		probe.Outputs[0].MaterializationSHA256 = changed
	case "ABSENT_OUTPUT_PRESENT":
		probe.Outputs[1].WorkspaceExists = true
		probe.Outputs[1].WorkspaceSHA256 = changed
	case "MATERIALIZATION_INVALID":
		probe.Outputs[0].MaterializationRef = "sha256:" + changed
	case "TASK_INVENTORY_UNAVAILABLE":
		probe.Tasks = nil
	case "NON_CACHEABLE_TASK":
		probe.Tasks[0].Cacheable = false
	case "UNTRACKED_TASK":
		probe.Tasks[0].Tracked = false
	case "ALWAYS_RUN_TASK":
		probe.Tasks[0].AlwaysRun = true
	case "LOCAL_STATE":
		probe.Tasks[0].LocalState = true
	case "DESTROYABLES":
		probe.Tasks[0].Destroyables = true
	case "UNTRACKED_WRITE":
		probe.Tasks[0].UntrackedWrites = true
	case "SIDE_EFFECT":
		probe.Tasks[0].SideEffects = true
	case "PRIOR_FAILURE":
		record.PriorResult.Outcome = "BUILD_FAILURE"
		record.PriorResult.ExitCode = 1
	case "PRIOR_CANCELLATION":
		record.PriorResult.Outcome = "CANCELLED"
		record.PriorResult.ExitCode = 130
	case "PRIOR_OUTPUTS_UNVERIFIED":
		record.PriorResult.OutputsVerified = false
	case "EXPIRED":
		record.ExpiresAt = "2026-08-30T08:30:00Z"
	case "REVOKED":
		probe.CurrentRevocationEpoch++
	case "EVIDENCE_INCOMPLETE":
		probe.EvidenceComplete = false
	default:
		return fmt.Errorf("unknown request-hit fixture mutation %q", mutation)
	}
	return nil
}
