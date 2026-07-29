package schemavalidator

import (
	"errors"
	"fmt"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// BuildSessionSchemaID is the canonical BUILD_SESSION v1 schema identifier.
const BuildSessionSchemaID = "https://schemas.buildopt.dev/build-session.v1.schema.json"

// ValidateBuildSessionV1 compiles the pinned Draft 2020-12 schema and validates
// one JSON document with format assertions enabled.
func ValidateBuildSessionV1(schemaPath string, instancePath string) error {
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()

	schema, err := compiler.Compile(schemaPath)
	if err != nil {
		return fmt.Errorf("compile BUILD_SESSION v1 schema: %w", err)
	}
	if schema.ID != BuildSessionSchemaID || schema.DraftVersion != 2020 {
		return errors.New("compiled BUILD_SESSION schema identity is invalid")
	}

	file, err := os.Open(instancePath)
	if err != nil {
		return errors.New("open BUILD_SESSION JSON")
	}
	instance, decodeErr := jsonschema.UnmarshalJSON(file)
	closeErr := file.Close()
	if decodeErr != nil {
		return fmt.Errorf("decode BUILD_SESSION JSON: %w", decodeErr)
	}
	if closeErr != nil {
		return errors.New("close BUILD_SESSION JSON")
	}
	if err := schema.Validate(instance); err != nil {
		return fmt.Errorf("validate BUILD_SESSION JSON: %w", err)
	}
	return nil
}
