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
