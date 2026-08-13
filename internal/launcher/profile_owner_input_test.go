package launcher

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tonyredondo/buildopt/internal/outputequivalence"
)

func TestParseConfirmedOutputContractRequiresValidatedDeclarations(t *testing.T) {
	report := outputContractReport{
		SchemaVersion: outputContractSchema, Decision: "VALIDATED_REQUIRED_OUTPUTS",
		Reason:       "DECLARED_OUTPUTS_MATCH_EXECUTED_WORKFLOW",
		RepositoryID: "example/repository", PipelineClass: "assemble",
		RepositoryRevision: strings.Repeat("a", 40), OriginalEntrypoints: []string{"assemble"},
		DeclaredOutputs: []string{"module/target/libs/**"},
		Validations: []outputContractValidation{{
			Pattern: "module/target/libs/**", Status: "VALIDATED", MatchedFiles: 1,
			OwnerProjects: []string{":module"}, ProducerTasks: []string{":module:jar"},
		}},
		ReviewRequired: true, TestOptimization: "OUT_OF_SCOPE",
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseConfirmedOutputContract(raw); err != nil {
		t.Fatal(err)
	}
	report.Validations[0].Status = "EMPTY"
	raw, _ = json.Marshal(report)
	if _, err := parseConfirmedOutputContract(raw); err == nil {
		t.Fatal("expected an empty output declaration to be rejected")
	}
}

func TestReadProfileOwnerInputRejectsTamperingAndSymlinks(t *testing.T) {
	root := t.TempDir()
	digest := sha256.Sum256([]byte("contract"))
	input := profileOwnerInput{
		SchemaVersion: profileOwnerInputSchema, RepositoryID: "example/repository",
		PipelineClass: "assemble", Entrypoints: []string{"assemble"},
		RequiredOutputs: []string{"module/target/libs/**"},
		ChangeSource:    "GIT_DIFF_BASE_TO_HEAD",
		GlobalChanges:   append([]string(nil), defaultProposalGlobalChanges...),
		GradleCommand:   "./gradlew", TimeoutMinutes: 5,
		OutputConfirmation: profileOwnerOutputConfirmation{
			Status: "OWNER_CONFIRMED", ObservedRevision: strings.Repeat("b", 40),
			ContractSHA256: hex.EncodeToString(digest[:]),
		},
		ReviewRequired: true, TestOptimization: "OUT_OF_SCOPE",
	}
	raw, err := renderProfileOwnerInput(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "profile.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readProfileOwnerInput(root, "profile.json"); err != nil {
		t.Fatal(err)
	}
	tampered := append([]byte(nil), raw[:len(raw)-2]...)
	tampered = append(tampered, []byte(",\n  \"unknown\": true\n}\n")...)
	if err := os.WriteFile(filepath.Join(root, "profile.json"), tampered, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readProfileOwnerInput(root, "profile.json"); err == nil {
		t.Fatal("expected an unknown field to be rejected")
	}
	if err := os.WriteFile(filepath.Join(root, "outside.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("outside.json", filepath.Join(root, "profile.json")); err == nil {
		t.Fatal("expected replacing an existing file with a symlink to fail")
	}
	if err := os.Remove(filepath.Join(root, "profile.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("outside.json", filepath.Join(root, "profile.json")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readProfileOwnerInput(root, "profile.json"); err == nil {
		t.Fatal("expected a symlinked owner input to be rejected")
	}
}

func TestReadProfileOwnerInputBindsOutputEquivalence(t *testing.T) {
	root := t.TempDir()
	contract := []byte(`{"schemaVersion":"buildopt.poc/output-equivalence/v1","rules":[{"pattern":"build/*.zip","mode":"CANONICAL_ZIP"}],"reviewRequired":true,"activationAutomatic":false,"productionAuthorized":false}`)
	if err := os.WriteFile(filepath.Join(root, "equivalence.json"), contract, 0o644); err != nil {
		t.Fatal(err)
	}
	input := profileOwnerInput{
		SchemaVersion: profileOwnerInputSchema, RepositoryID: "example/repository",
		PipelineClass: "assemble", Entrypoints: []string{"assemble"},
		RequiredOutputs: []string{"build/*.zip"}, ChangeSource: "GIT_DIFF_BASE_TO_HEAD",
		GlobalChanges:  append([]string(nil), defaultProposalGlobalChanges...),
		TimeoutMinutes: 5,
		OutputConfirmation: profileOwnerOutputConfirmation{
			Status: "OWNER_CONFIRMED", ObservedRevision: strings.Repeat("b", 40),
			ContractSHA256: strings.Repeat("c", 64),
		},
		OutputEquivalence: &profileOwnerDocumentBinding{
			Path: "equivalence.json", SHA256: outputequivalence.SHA256(contract),
		},
		ReviewRequired: true, TestOptimization: "OUT_OF_SCOPE",
	}
	raw, err := renderProfileOwnerInput(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "profile.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readProfileOwnerInput(root, "profile.json"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "equivalence.json"), append(contract, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readProfileOwnerInput(root, "profile.json"); err == nil {
		t.Fatal("equivalence-contract drift was accepted")
	}
}
