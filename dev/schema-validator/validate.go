package schemavalidator

import (
	"errors"
	"fmt"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// BuildSessionSchemaID is the canonical BUILD_SESSION v1 schema identifier.
const BuildSessionSchemaID = "https://schemas.buildopt.dev/build-session.v1.schema.json"

// ExperimentResultSchemaID is the canonical EXPERIMENT_RESULT v1 schema identifier.
const ExperimentResultSchemaID = "https://schemas.buildopt.dev/experiment-result.v1.schema.json"

// ActionRecordSchemaID is the canonical ACTION_RECORD v1 schema identifier.
const ActionRecordSchemaID = "https://schemas.buildopt.dev/action-record.v1.schema.json"

// EvidenceRecordSchemaID is the canonical EVIDENCE_RECORD v1 schema identifier.
const EvidenceRecordSchemaID = "https://schemas.buildopt.dev/evidence-record.v1.schema.json"

// OptimizationPolicySchemaID is the canonical OPTIMIZATION_POLICY v1 schema identifier.
const OptimizationPolicySchemaID = "https://schemas.buildopt.dev/optimization-policy.v1.schema.json"

// ResourceProfileSchemaID is the canonical RESOURCE_PROFILE v1 schema identifier.
const ResourceProfileSchemaID = "https://schemas.buildopt.dev/resource-profile.v1.schema.json"

// AttemptStateSchemaID is the canonical ATTEMPT_STATE v1 schema identifier.
const AttemptStateSchemaID = "https://schemas.buildopt.dev/attempt-state.v1.schema.json"

// CIValidationRequestSchemaID is the canonical CI_VALIDATION_REQUEST v1 schema identifier.
const CIValidationRequestSchemaID = "https://schemas.buildopt.dev/ci-validation-request.v1.schema.json"

// CommitDecisionSchemaID is the canonical COMMIT_DECISION v1 schema identifier.
const CommitDecisionSchemaID = "https://schemas.buildopt.dev/commit-decision.v1.schema.json"

// TestCacheGrantSchemaID is the canonical TEST_CACHE_GRANT v1 schema identifier.
const TestCacheGrantSchemaID = "https://schemas.buildopt.dev/test-cache-grant.v1.schema.json"

// TestValidationResultSchemaID is the canonical TEST_VALIDATION_RESULT v1 schema identifier.
const TestValidationResultSchemaID = "https://schemas.buildopt.dev/test-validation-result.v1.schema.json"

// PatchBundleSchemaID is the canonical PATCH_BUNDLE v1 schema identifier.
const PatchBundleSchemaID = "https://schemas.buildopt.dev/patch-bundle.v1.schema.json"

// CentralStateManifestSchemaID is the canonical immutable central-state
// manifest schema identifier.
const CentralStateManifestSchemaID = "https://schemas.buildopt.dev/central-state-manifest.v1.schema.json"

// CentralStateHeadSchemaID is the canonical central-state head schema
// identifier.
const CentralStateHeadSchemaID = "https://schemas.buildopt.dev/central-state-head.v1.schema.json"

// CentralStateCASSchemaID is the canonical central-state compare-and-swap
// request schema identifier.
const CentralStateCASSchemaID = "https://schemas.buildopt.dev/central-state-cas.v1.schema.json"

// ValidateBuildSessionV1 compiles the pinned Draft 2020-12 schema and validates
// one JSON document with format assertions enabled.
func ValidateBuildSessionV1(schemaPath string, instancePath string) error {
	return validateContract(
		schemaPath,
		instancePath,
		BuildSessionSchemaID,
		"BUILD_SESSION",
	)
}

// ValidateExperimentResultV1 compiles the pinned Draft 2020-12 schema and
// validates one immutable aggregate experiment result.
func ValidateExperimentResultV1(schemaPath string, instancePath string) error {
	return validateContract(
		schemaPath,
		instancePath,
		ExperimentResultSchemaID,
		"EXPERIMENT_RESULT",
	)
}

// ValidateActionRecordV1 compiles the pinned Draft 2020-12 schema and validates
// one append-only action transition record.
func ValidateActionRecordV1(schemaPath string, instancePath string) error {
	return validateContract(
		schemaPath,
		instancePath,
		ActionRecordSchemaID,
		"ACTION_RECORD",
	)
}

// ValidateEvidenceRecordV1 compiles the pinned Draft 2020-12 schema and
// validates one immutable task-evidence record.
func ValidateEvidenceRecordV1(schemaPath string, instancePath string) error {
	return validateContract(
		schemaPath,
		instancePath,
		EvidenceRecordSchemaID,
		"EVIDENCE_RECORD",
	)
}

// ValidateOptimizationPolicyV1 compiles the pinned Draft 2020-12 schema and
// validates one immutable optimization policy.
func ValidateOptimizationPolicyV1(schemaPath string, instancePath string) error {
	return validateContract(
		schemaPath,
		instancePath,
		OptimizationPolicySchemaID,
		"OPTIMIZATION_POLICY",
	)
}

// ValidateResourceProfileV1 compiles the pinned Draft 2020-12 schema and
// validates one finite resource-profile catalog arm.
func ValidateResourceProfileV1(schemaPath string, instancePath string) error {
	return validateContract(
		schemaPath,
		instancePath,
		ResourceProfileSchemaID,
		"RESOURCE_PROFILE",
	)
}

// ValidateAttemptStateV1 compiles the pinned Draft 2020-12 schema and validates
// one immutable attempt-state transition.
func ValidateAttemptStateV1(schemaPath string, instancePath string) error {
	return validateContract(
		schemaPath,
		instancePath,
		AttemptStateSchemaID,
		"ATTEMPT_STATE",
	)
}

// ValidateCIValidationRequestV1 compiles the pinned Draft 2020-12 schema and
// validates one isolated CI validation request.
func ValidateCIValidationRequestV1(schemaPath string, instancePath string) error {
	return validateContract(
		schemaPath,
		instancePath,
		CIValidationRequestSchemaID,
		"CI_VALIDATION_REQUEST",
	)
}

// ValidateCommitDecisionV1 compiles the pinned Draft 2020-12 schema and
// validates one atomic cache publication decision.
func ValidateCommitDecisionV1(schemaPath string, instancePath string) error {
	return validateContract(
		schemaPath,
		instancePath,
		CommitDecisionSchemaID,
		"COMMIT_DECISION",
	)
}

// ValidateTestCacheGrantV1 compiles the pinned Draft 2020-12 schema and
// validates one signed Test Optimization cache grant.
func ValidateTestCacheGrantV1(schemaPath string, instancePath string) error {
	return validateContract(
		schemaPath,
		instancePath,
		TestCacheGrantSchemaID,
		"TEST_CACHE_GRANT",
	)
}

// ValidateTestValidationResultV1 compiles the pinned Draft 2020-12 schema and
// validates one signed Test Optimization validation result.
func ValidateTestValidationResultV1(schemaPath string, instancePath string) error {
	return validateContract(
		schemaPath,
		instancePath,
		TestValidationResultSchemaID,
		"TEST_VALIDATION_RESULT",
	)
}

// ValidatePatchBundleV1 compiles the pinned Draft 2020-12 schema and validates
// one signed declarative patch bundle.
func ValidatePatchBundleV1(schemaPath string, instancePath string) error {
	return validateContract(
		schemaPath,
		instancePath,
		PatchBundleSchemaID,
		"PATCH_BUNDLE",
	)
}

func validateContract(
	schemaPath string,
	instancePath string,
	expectedSchemaID string,
	recordType string,
) error {
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()

	schema, err := compiler.Compile(schemaPath)
	if err != nil {
		return fmt.Errorf("compile %s v1 schema: %w", recordType, err)
	}
	if schema.ID != expectedSchemaID || schema.DraftVersion != 2020 {
		return errors.New("compiled " + recordType + " schema identity is invalid")
	}

	file, err := os.Open(instancePath)
	if err != nil {
		return fmt.Errorf("open %s JSON: %w", recordType, err)
	}
	instance, decodeErr := jsonschema.UnmarshalJSON(file)
	closeErr := file.Close()
	if decodeErr != nil {
		return fmt.Errorf("decode %s JSON: %w", recordType, decodeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close %s JSON: %w", recordType, closeErr)
	}
	if err := schema.Validate(instance); err != nil {
		return fmt.Errorf("validate %s JSON: %w", recordType, err)
	}
	return nil
}
