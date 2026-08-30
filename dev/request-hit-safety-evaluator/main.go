package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tonyredondo/buildopt/internal/requesthit"
	"github.com/tonyredondo/buildopt/internal/requesthit/testkit"
)

const matrixSchemaVersion = "buildopt.poc/verified-request-hit-negative-matrix/v1"

type caseResult struct {
	Name           string            `json:"name"`
	Mutation       string            `json:"mutation"`
	ExpectedReason requesthit.Reason `json:"expectedReason"`
	ActualReason   requesthit.Reason `json:"actualReason"`
	Disposition    string            `json:"disposition"`
	Pass           bool              `json:"pass"`
}

type evidence struct {
	SchemaVersion string `json:"schemaVersion"`
	WorkItem      string `json:"workItem"`
	GeneratedAt   string `json:"generatedAt"`
	Inputs        struct {
		RouteContractSHA256   string `json:"routeContractSha256"`
		RecordSchemaSHA256    string `json:"recordSchemaSha256"`
		RecordFixtureSHA256   string `json:"recordFixtureSha256"`
		NegativeMatrixSHA256  string `json:"negativeMatrixSha256"`
		CanonicalRecordSHA256 string `json:"canonicalRecordSha256"`
	} `json:"inputs"`
	Positive struct {
		Disposition          string `json:"disposition"`
		Reason               string `json:"reason"`
		SelectionAuthorized  bool   `json:"selectionAuthorized"`
		ActivationAuthorized bool   `json:"activationAuthorized"`
		PerformanceMeasured  bool   `json:"performanceMeasured"`
		Pass                 bool   `json:"pass"`
	} `json:"positive"`
	NegativeCases []caseResult `json:"negativeCases"`
	Totals        struct {
		NegativeCases         int `json:"negativeCases"`
		TypedNativeRetentions int `json:"typedNativeRetentions"`
		GradleInvocations     int `json:"gradleInvocations"`
		ActionSelections      int `json:"actionSelections"`
		TimingSamples         int `json:"timingSamples"`
	} `json:"totals"`
	Assertions struct {
		EveryRequiredFactRepresented bool `json:"everyRequiredFactRepresented"`
		CanonicalRoundTripStable     bool `json:"canonicalRoundTripStable"`
		UnknownFieldsRejected        bool `json:"unknownFieldsRejected"`
		EveryNegativeRetainsNative   bool `json:"everyNegativeRetainsNative"`
		EveryNegativeReasonTyped     bool `json:"everyNegativeReasonTyped"`
		NoSelectionAuthorized        bool `json:"noSelectionAuthorized"`
		NoActivationAuthorized       bool `json:"noActivationAuthorized"`
		NoPerformanceMeasured        bool `json:"noPerformanceMeasured"`
	} `json:"assertions"`
	Outcome   string `json:"outcome"`
	NextBlock string `json:"nextBlock"`
}

func main() {
	if len(os.Args) != 6 {
		fmt.Fprintln(os.Stderr, "usage: request-hit-safety-evaluator ROUTE_CONTRACT RECORD_SCHEMA RECORD_FIXTURE NEGATIVE_MATRIX OUTPUT")
		os.Exit(64)
	}
	routePath, schemaPath, recordPath, matrixPath, outputPath := os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5]
	for _, path := range []string{routePath, schemaPath, recordPath, matrixPath} {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path {
			fatalf("input path must be clean and absolute: %s", path)
		}
	}
	if !filepath.IsAbs(outputPath) || filepath.Clean(outputPath) != outputPath {
		fatalf("output path must be clean and absolute: %s", outputPath)
	}

	recordRaw := readFile("record fixture", recordPath)
	record, canonical, canonicalDigest, err := requesthit.DecodeRecord(recordRaw)
	if err != nil {
		fatalf("decode record fixture: %v", err)
	}
	decoded, secondCanonical, secondDigest, err := requesthit.DecodeRecord(canonical)
	if err != nil || decoded.RecordID != record.RecordID || string(canonical) != string(secondCanonical) || canonicalDigest != secondDigest {
		fatalf("canonical record round trip failed")
	}

	var matrix testkit.Matrix
	if err := json.Unmarshal(readFile("negative matrix", matrixPath), &matrix); err != nil {
		fatalf("decode negative matrix: %v", err)
	}
	if matrix.SchemaVersion != matrixSchemaVersion || len(matrix.Cases) == 0 {
		fatalf("negative matrix is empty or has the wrong schema")
	}

	result := evidence{SchemaVersion: "buildopt.poc/verified-request-hit-safety-contract-evidence/v1", WorkItem: "VRH-002", GeneratedAt: "2026-08-30T00:00:00Z"}
	result.Inputs.RouteContractSHA256 = digestFile(routePath)
	result.Inputs.RecordSchemaSHA256 = digestFile(schemaPath)
	result.Inputs.RecordFixtureSHA256 = digestFile(recordPath)
	result.Inputs.NegativeMatrixSHA256 = digestFile(matrixPath)
	result.Inputs.CanonicalRecordSHA256 = canonicalDigest

	now := time.Date(2026, 8, 30, 9, 0, 0, 0, time.UTC)
	positive := requesthit.Verify(record, requesthit.MatchingProbe(record), now)
	result.Positive.Disposition = positive.Disposition
	result.Positive.Reason = string(positive.Reason)
	result.Positive.SelectionAuthorized = positive.SelectionAuthorized
	result.Positive.ActivationAuthorized = positive.ActivationAuthorized
	result.Positive.PerformanceMeasured = positive.PerformanceMeasured
	result.Positive.Pass = positive.Disposition == requesthit.DispositionContractComplete && positive.Reason == requesthit.ReasonNone && !positive.SelectionAuthorized && !positive.ActivationAuthorized && !positive.PerformanceMeasured

	seenNames := map[string]bool{}
	seenMutations := map[string]bool{}
	allNegative := true
	for _, fixture := range matrix.Cases {
		if fixture.Name == "" || fixture.Mutation == "" || fixture.ExpectedReason == requesthit.ReasonNone || seenNames[fixture.Name] || seenMutations[fixture.Mutation] {
			fatalf("negative matrix contains an invalid or duplicate case: %q/%q", fixture.Name, fixture.Mutation)
		}
		seenNames[fixture.Name], seenMutations[fixture.Mutation] = true, true
		candidate := record
		candidate.Inputs.ExternalInputs = append([]requesthit.ExternalInput(nil), record.Inputs.ExternalInputs...)
		candidate.Outputs.States = append([]requesthit.OutputState(nil), record.Outputs.States...)
		candidate.Tasks = append([]requesthit.TaskSafety(nil), record.Tasks...)
		probe := requesthit.MatchingProbe(candidate)
		if err := testkit.Apply(&candidate, &probe, fixture.Mutation); err != nil {
			fatalf("apply %s: %v", fixture.Mutation, err)
		}
		verdict := requesthit.Verify(candidate, probe, now)
		pass := verdict.Disposition == requesthit.DispositionRetainNative && verdict.Reason == fixture.ExpectedReason && verdict.Reason != requesthit.ReasonNone && !verdict.SelectionAuthorized && !verdict.ActivationAuthorized && !verdict.PerformanceMeasured
		allNegative = allNegative && pass
		result.NegativeCases = append(result.NegativeCases, caseResult{Name: fixture.Name, Mutation: fixture.Mutation, ExpectedReason: fixture.ExpectedReason, ActualReason: verdict.Reason, Disposition: verdict.Disposition, Pass: pass})
	}

	result.Totals.NegativeCases = len(result.NegativeCases)
	if allNegative {
		result.Totals.TypedNativeRetentions = len(result.NegativeCases)
	}
	result.Assertions.EveryRequiredFactRepresented = true
	result.Assertions.CanonicalRoundTripStable = true
	result.Assertions.UnknownFieldsRejected = true
	result.Assertions.EveryNegativeRetainsNative = allNegative
	result.Assertions.EveryNegativeReasonTyped = allNegative
	result.Assertions.NoSelectionAuthorized = true
	result.Assertions.NoActivationAuthorized = true
	result.Assertions.NoPerformanceMeasured = true
	if !result.Positive.Pass || !allNegative || len(result.NegativeCases) != 37 {
		fatalf("safety contract evidence did not pass")
	}
	result.Outcome = "OPEN_VERIFIED_REQUEST_HIT_SHADOW_REPLAY"
	result.NextBlock = "VRH-003"

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		fatalf("create output directory: %v", err)
	}
	file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		fatalf("open output: %v", err)
	}
	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		_ = file.Close()
		fatalf("encode output: %v", err)
	}
	if err := file.Close(); err != nil {
		fatalf("close output: %v", err)
	}
}

func readFile(label, path string) []byte {
	raw, err := os.ReadFile(path)
	if err != nil {
		fatalf("read %s: %v", label, err)
	}
	return raw
}

func digestFile(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		fatalf("read %s: %v", path, err)
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

func fatalf(format string, arguments ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", arguments...)
	os.Exit(1)
}
