// Command request-hit-shadow-evaluator runs the deterministic VRH-003
// correctness matrix. It deliberately records no duration or performance data.
package main

import (
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/requesthit"
)

const generatedAt = "2026-08-30T00:00:00Z"

type syntheticCase struct {
	GradleVersion     string `json:"gradleVersion"`
	DSL               string `json:"dsl"`
	RepeatCount       int    `json:"repeatCount"`
	ExactMatches      int    `json:"exactMatches"`
	NativeInvocations int    `json:"nativeInvocations"`
	RecordSHA256      string `json:"recordSha256"`
}

type mismatchProof struct {
	GradleVersion            string `json:"gradleVersion"`
	DSL                      string `json:"dsl"`
	FirstDisposition         string `json:"firstDisposition"`
	FirstReason              string `json:"firstReason"`
	NextDisposition          string `json:"nextDisposition"`
	NextReason               string `json:"nextReason"`
	FirstMismatchQuarantined bool   `json:"firstMismatchQuarantined"`
	NextReplayPredicted      bool   `json:"nextReplayPredicted"`
	NativeExecutedBothTimes  bool   `json:"nativeExecutedBothTimes"`
}

type publicFamily struct {
	Family                   string `json:"family"`
	DSL                      string `json:"dsl"`
	PotentialRows            int    `json:"potentialRows"`
	NativeOutcomeMatches     int    `json:"nativeOutcomeMatches"`
	NativeOutputMatches      int    `json:"nativeOutputMatches"`
	NativeOutputMismatches   int    `json:"nativeOutputMismatches"`
	SafetyAdmittedRows       int    `json:"safetyAdmittedRows"`
	ExactOutputOrdinals      []int  `json:"exactOutputOrdinals"`
	MismatchedOutputOrdinals []int  `json:"mismatchedOutputOrdinals"`
	RetentionReason          string `json:"retentionReason"`
}

type result struct {
	SchemaVersion string `json:"schemaVersion"`
	WorkItem      string `json:"workItem"`
	GeneratedAt   string `json:"generatedAt"`
	Inputs        struct {
		RouteContractSHA256        string `json:"routeContractSha256"`
		SafetySchemaSHA256         string `json:"safetySchemaSha256"`
		ShadowImplementationSHA256 string `json:"shadowImplementationSha256"`
		EvaluatorSourceSHA256      string `json:"evaluatorSourceSha256"`
		FixtureTreeSHA256          string `json:"fixtureTreeSha256"`
		PublicContractSHA256       string `json:"publicContractSha256"`
		PublicObservationsSHA256   string `json:"publicObservationsSha256"`
		PublicTransitionsSHA256    string `json:"publicTransitionsSha256"`
		PublicSummarySHA256        string `json:"publicSummarySha256"`
	} `json:"inputs"`
	Thresholds struct {
		SyntheticRepeatsPerCell          int `json:"syntheticRepeatsPerCell"`
		SyntheticCellsRequired           int `json:"syntheticCellsRequired"`
		PublicExactRowsPerFamilyRequired int `json:"publicExactRowsPerFamilyRequired"`
		PublicFamiliesRequired           int `json:"publicFamiliesRequired"`
		ProductFailureBudget             int `json:"productFailureBudget"`
	} `json:"thresholds"`
	Synthetic struct {
		Cases           []syntheticCase `json:"cases"`
		ExactMatches    int             `json:"exactMatches"`
		RequiredMatches int             `json:"requiredMatches"`
		MatrixPass      bool            `json:"matrixPass"`
		Mismatch        mismatchProof   `json:"mismatch"`
		QuarantinePass  bool            `json:"quarantinePass"`
	} `json:"synthetic"`
	Public struct {
		Families                          []publicFamily `json:"families"`
		PotentialRows                     int            `json:"potentialRows"`
		NativeOutputMatches               int            `json:"nativeOutputMatches"`
		NativeOutputMismatches            int            `json:"nativeOutputMismatches"`
		SafetyAdmittedRows                int            `json:"safetyAdmittedRows"`
		FamiliesMeetingExactOutputBreadth int            `json:"familiesMeetingExactOutputBreadth"`
		FamiliesMeetingAdmissionBreadth   int            `json:"familiesMeetingAdmissionBreadth"`
		OutputBreadthPass                 bool           `json:"outputBreadthPass"`
		AdmissionBreadthPass              bool           `json:"admissionBreadthPass"`
	} `json:"public"`
	Totals struct {
		GradleInvocations     int `json:"gradleInvocations"`
		Predictions           int `json:"predictions"`
		ShadowMatches         int `json:"shadowMatches"`
		QuarantinedIdentities int `json:"quarantinedIdentities"`
		ActionSelections      int `json:"actionSelections"`
		TimingSamples         int `json:"timingSamples"`
	} `json:"totals"`
	Authority struct {
		SelectionAuthorized  bool `json:"selectionAuthorized"`
		ActivationAuthorized bool `json:"activationAuthorized"`
		PerformanceMeasured  bool `json:"performanceMeasured"`
		SpeedupClaim         bool `json:"speedupClaim"`
	} `json:"authority"`
	Decision   string `json:"decision"`
	Conclusion string `json:"conclusion"`
	NextBlock  string `json:"nextBlock"`
}

type observation struct {
	Family             string   `json:"family"`
	DSL                string   `json:"dsl"`
	ObservationOrdinal int      `json:"observationOrdinal"`
	Revision           string   `json:"revision"`
	Command            []string `json:"command"`
	ExitCode           int      `json:"exitCode"`
	ProductFailure     bool     `json:"productFailure"`
	CaptureComplete    bool     `json:"captureComplete"`
	Capture            string   `json:"capture"`
	Evidence           string   `json:"evidence"`
}

type transition struct {
	Family             string   `json:"family"`
	DSL                string   `json:"dsl"`
	ObservationOrdinal int      `json:"observationOrdinal"`
	Command            []string `json:"command"`
	BaseRevision       string   `json:"baseRevision"`
	TargetRevision     string   `json:"targetRevision"`
	Status             string   `json:"status"`
}

type capture struct {
	Status string        `json:"status"`
	Tasks  []captureTask `json:"tasks"`
}

type captureTask struct {
	Path    string          `json:"path"`
	Outputs []captureOutput `json:"outputs"`
}

type captureOutput struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	SHA256 string `json:"sha256"`
	Exists bool   `json:"exists"`
}

type requestEvidence struct {
	ArgumentsSHA256             string `json:"argumentsSha256"`
	CompatibilityIdentitySHA256 string `json:"compatibilityIdentitySha256"`
	RequestGraphIdentitySHA256  string `json:"requestGraphIdentitySha256"`
}

func main() {
	if len(os.Args) != 8 {
		fatalf("usage: request-hit-shadow-evaluator FIXTURE_ROOT GRADLE_8 GRADLE_9 PUBLIC_RESULT ROUTE_CONTRACT SAFETY_SCHEMA OUTPUT")
	}
	fixtureRoot, gradle8, gradle9 := cleanAbsolute(os.Args[1]), cleanAbsolute(os.Args[2]), cleanAbsolute(os.Args[3])
	publicRoot, routePath, schemaPath, outputPath := cleanAbsolute(os.Args[4]), cleanAbsolute(os.Args[5]), cleanAbsolute(os.Args[6]), cleanAbsolute(os.Args[7])
	repositoryRoot := filepath.Dir(filepath.Dir(routePath))

	result := result{SchemaVersion: "buildopt.poc/verified-request-hit-shadow-replay/v1", WorkItem: "VRH-003", GeneratedAt: generatedAt}
	result.Inputs.RouteContractSHA256 = digestFile(routePath)
	result.Inputs.SafetySchemaSHA256 = digestFile(schemaPath)
	result.Inputs.ShadowImplementationSHA256 = digestFile(filepath.Join(repositoryRoot, "internal", "requesthit", "shadow.go"))
	result.Inputs.EvaluatorSourceSHA256 = digestFile(filepath.Join(repositoryRoot, "dev", "request-hit-shadow-evaluator", "main.go"))
	result.Inputs.FixtureTreeSHA256 = digestTree(fixtureRoot)
	result.Inputs.PublicContractSHA256 = digestFile(filepath.Join(publicRoot, "contract.json"))
	result.Inputs.PublicObservationsSHA256 = digestFile(filepath.Join(publicRoot, "observations.jsonl"))
	result.Inputs.PublicTransitionsSHA256 = digestFile(filepath.Join(publicRoot, "transitions.jsonl"))
	result.Inputs.PublicSummarySHA256 = digestFile(filepath.Join(publicRoot, "summary.json"))
	result.Thresholds.SyntheticRepeatsPerCell = 2
	result.Thresholds.SyntheticCellsRequired = 4
	result.Thresholds.PublicExactRowsPerFamilyRequired = 5
	result.Thresholds.PublicFamiliesRequired = 3
	result.Thresholds.ProductFailureBudget = 0

	temporary, err := os.MkdirTemp("", "buildopt-request-hit-shadow-")
	if err != nil {
		fatalf("create temporary root: %v", err)
	}
	defer os.RemoveAll(temporary)

	gradles := []struct{ version, path string }{{"8.14.3", gradle8}, {"9.6.1", gradle9}}
	for _, gradle := range gradles {
		for _, dsl := range []string{"KOTLIN", "GROOVY"} {
			cell, mismatch, invocations := runSyntheticCell(temporary, fixtureRoot, gradle.version, gradle.path, dsl, gradle.version == "9.6.1" && dsl == "KOTLIN")
			result.Synthetic.Cases = append(result.Synthetic.Cases, cell)
			result.Synthetic.ExactMatches += cell.ExactMatches
			result.Totals.GradleInvocations += invocations
			result.Totals.Predictions += cell.ExactMatches
			result.Totals.ShadowMatches += cell.ExactMatches
			if mismatch != nil {
				result.Synthetic.Mismatch = *mismatch
				result.Synthetic.QuarantinePass = mismatch.FirstMismatchQuarantined && !mismatch.NextReplayPredicted && mismatch.NativeExecutedBothTimes
				result.Totals.Predictions++
				result.Totals.QuarantinedIdentities++
			}
		}
	}
	result.Synthetic.RequiredMatches = result.Thresholds.SyntheticRepeatsPerCell * result.Thresholds.SyntheticCellsRequired
	result.Synthetic.MatrixPass = len(result.Synthetic.Cases) == result.Thresholds.SyntheticCellsRequired && result.Synthetic.ExactMatches == result.Synthetic.RequiredMatches

	result.Public.Families = analyzePublic(publicRoot)
	for _, family := range result.Public.Families {
		result.Public.PotentialRows += family.PotentialRows
		result.Public.NativeOutputMatches += family.NativeOutputMatches
		result.Public.NativeOutputMismatches += family.NativeOutputMismatches
		result.Public.SafetyAdmittedRows += family.SafetyAdmittedRows
		if family.NativeOutputMatches >= result.Thresholds.PublicExactRowsPerFamilyRequired {
			result.Public.FamiliesMeetingExactOutputBreadth++
		}
		if family.SafetyAdmittedRows >= result.Thresholds.PublicExactRowsPerFamilyRequired {
			result.Public.FamiliesMeetingAdmissionBreadth++
		}
	}
	result.Public.OutputBreadthPass = result.Public.FamiliesMeetingExactOutputBreadth >= result.Thresholds.PublicFamiliesRequired
	result.Public.AdmissionBreadthPass = result.Public.FamiliesMeetingAdmissionBreadth >= result.Thresholds.PublicFamiliesRequired

	if !result.Synthetic.MatrixPass || !result.Synthetic.QuarantinePass {
		fatalf("synthetic shadow or quarantine contract failed")
	}
	if result.Public.PotentialRows != 69 || result.Public.NativeOutputMatches != 34 || result.Public.NativeOutputMismatches != 35 {
		fatalf("public shadow preflight drifted: rows=%d exact=%d mismatch=%d", result.Public.PotentialRows, result.Public.NativeOutputMatches, result.Public.NativeOutputMismatches)
	}
	result.Decision = "STOP_VERIFIED_REQUEST_HIT_BEFORE_GRADLE_FREE_EXECUTION"
	result.Conclusion = "SHADOW_MECHANICS_PASS_BUT_PUBLIC_BREADTH_AND_COMPLETE_SAFETY_ADMISSION_FAIL"
	result.NextBlock = "NONE"
	writeJSON(outputPath, result)
}

func runSyntheticCell(temporary, fixtureRoot, version, gradle, dsl string, proveMismatch bool) (syntheticCase, *mismatchProof, int) {
	repository := filepath.Join(temporary, strings.ReplaceAll(version+"-"+strings.ToLower(dsl), ".", "_"))
	if err := os.CopyFS(repository, os.DirFS(filepath.Join(fixtureRoot, strings.ToLower(dsl)))); err != nil {
		fatalf("copy %s fixture: %v", dsl, err)
	}
	wrapper := filepath.Join(repository, "gradle", "wrapper", "gradle-wrapper.properties")
	if err := os.MkdirAll(filepath.Dir(wrapper), 0o755); err != nil {
		fatalf("create wrapper directory: %v", err)
	}
	if err := os.WriteFile(wrapper, []byte("distributionUrl=https\\://services.gradle.org/distributions/gradle-"+version+"-bin.zip\n"), 0o644); err != nil {
		fatalf("write wrapper binding: %v", err)
	}
	arguments := []string{":requestHitOutput", "--no-daemon", "--no-build-cache", "--no-configuration-cache", "--console=plain"}
	if code := runGradle(gradle, repository, arguments, ""); code != 0 {
		fatalf("baseline Gradle %s/%s exited %d", version, dsl, code)
	}
	record := syntheticRecord(repository, version, dsl, arguments)
	quarantine := requesthit.NewQuarantine()
	cell := syntheticCase{GradleVersion: version, DSL: dsl, RepeatCount: 2, NativeInvocations: 3}
	_, recordDigest, err := requesthit.CanonicalRecord(record)
	if err != nil {
		fatalf("canonicalize synthetic record: %v", err)
	}
	cell.RecordSHA256 = recordDigest
	for repeat := 0; repeat < cell.RepeatCount; repeat++ {
		resetOutput(repository)
		code := runGradle(gradle, repository, arguments, "")
		native := observeNative(repository, code)
		verdict := requesthit.Replay(record, requesthit.MatchingProbe(record), native, quarantine, verificationTime())
		if verdict.Disposition != requesthit.ShadowDispositionMatched || !verdict.Matched || !verdict.NativeExecuted || verdict.SelectionAuthorized || verdict.ActivationAuthorized || verdict.PerformanceMeasured {
			fatalf("synthetic shadow mismatch %s/%s repeat %d: %+v", version, dsl, repeat+1, verdict)
		}
		cell.ExactMatches++
	}
	if !proveMismatch {
		return cell, nil, cell.NativeInvocations
	}
	resetOutput(repository)
	mismatchCode := runGradle(gradle, repository, arguments, "unexpected")
	first := requesthit.Replay(record, requesthit.MatchingProbe(record), observeNative(repository, mismatchCode), quarantine, verificationTime())
	resetOutput(repository)
	exactCode := runGradle(gradle, repository, arguments, "")
	next := requesthit.Replay(record, requesthit.MatchingProbe(record), observeNative(repository, exactCode), quarantine, verificationTime())
	proof := &mismatchProof{
		GradleVersion: version, DSL: dsl, FirstDisposition: first.Disposition, FirstReason: first.Reason,
		NextDisposition: next.Disposition, NextReason: next.Reason,
		FirstMismatchQuarantined: first.Quarantined, NextReplayPredicted: next.Predicted,
		NativeExecutedBothTimes: first.NativeExecuted && next.NativeExecuted,
	}
	return cell, proof, cell.NativeInvocations + 2
}

func syntheticRecord(repository, version, dsl string, arguments []string) requesthit.SafetyRecord {
	resultPath := filepath.Join(repository, "build", "request-hit", "result.txt")
	resultSHA := digestFile(resultPath)
	info, err := os.Stat(resultPath)
	if err != nil {
		fatalf("stat baseline result: %v", err)
	}
	buildFile := "build.gradle"
	settingsFile := "settings.gradle"
	if dsl == "KOTLIN" {
		buildFile += ".kts"
		settingsFile += ".kts"
	}
	logicSHA := digestStrings(digestFile(filepath.Join(repository, buildFile)), digestFile(filepath.Join(repository, settingsFile)))
	gradleSHA := digestStrings("gradle", version)
	jdkSHA := digestStrings("jdk", os.Getenv("JAVA_HOME"), os.Getenv("JAVA_VERSION"))
	record := requesthit.SafetyRecord{
		SchemaVersion: requesthit.SchemaVersion, RecordType: requesthit.RecordType,
		RecordID: digestStrings("record", version, dsl, resultSHA), CapturedAt: "2026-08-30T08:00:00Z", ExpiresAt: "2026-08-31T08:00:00Z",
		Request: requesthit.RequestBinding{
			ArgumentEncoding: requesthit.ArgumentEncoding, ArgumentCount: len(arguments), ArgumentsSHA256: requesthit.DigestArgumentVector(arguments),
			WorkingDirectory: ".", RepositoryScopeSHA256: digestStrings("scope", dsl), RepositoryIdentitySHA256: digestStrings("repository", dsl),
		},
		Execution: requesthit.ExecutionBinding{
			WrapperSHA256: digestFile(filepath.Join(repository, "gradle", "wrapper", "gradle-wrapper.properties")), GradleVersion: version,
			GradleDistributionSHA256: gradleSHA, JDKVendor: "Eclipse Adoptium", JDKVersion: "21", JDKRuntimeSHA256: jdkSHA,
			SafeEnvironmentSHA256: digestStrings("BUILDOPT_SHADOW_MISMATCH", "ABSENT"), RequestGraphSHA256: digestStrings(":requestHitOutput"),
			TaskImplementationSHA256: digestStrings("RequestHitOutput", logicSHA), BuildLogicSHA256: logicSHA,
		},
		Inputs: requesthit.InputBinding{
			RepositoryInputsComplete: true, RepositoryManifestSHA256: digestFile(filepath.Join(repository, "inputs", "payload.txt")), ExternalInputsComplete: true,
			ExternalInputs: []requesthit.ExternalInput{
				{Kind: "GRADLE_DISTRIBUTION", Identity: "gradle:" + version, Present: true, SHA256: gradleSHA},
				{Kind: "JDK_RUNTIME", Identity: "java-home", Present: true, SHA256: jdkSHA},
			},
		},
		Outputs: requesthit.OutputContract{Complete: true, States: []requesthit.OutputState{
			{Path: "build/request-hit/forbidden.txt", Kind: "FILE", Tracked: true},
			{Path: "build/request-hit/result.txt", Kind: "FILE", Exists: true, Tracked: true, SHA256: resultSHA, Size: info.Size(), Mode: uint32(info.Mode().Perm()), MaterializationRef: "sha256:" + resultSHA},
		}},
		Tasks:       []requesthit.TaskSafety{{Path: ":requestHitOutput", Cacheable: true, Tracked: true}},
		PriorResult: requesthit.PriorResult{Outcome: "SUCCESS", ExitCode: 0, OutputsVerified: true},
	}
	return record
}

func runGradle(gradle, repository string, arguments []string, mismatch string) int {
	command := exec.Command(gradle, arguments...)
	command.Dir = repository
	command.Env = filteredEnvironment("BUILDOPT_SHADOW_MISMATCH")
	if mismatch != "" {
		command.Env = append(command.Env, "BUILDOPT_SHADOW_MISMATCH="+mismatch)
	}
	output, err := command.CombinedOutput()
	if err == nil {
		return 0
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return exitError.ExitCode()
	}
	fatalf("start Gradle: %v\n%s", err, output)
	return 1
}

func observeNative(repository string, exitCode int) requesthit.NativeResult {
	resultPath := filepath.Join(repository, "build", "request-hit", "result.txt")
	result := requesthit.NativeResult{Outcome: "SUCCESS", ExitCode: exitCode, ExactCommandPreserved: true, GradleProcessObserved: true}
	if exitCode != 0 {
		result.Outcome = "BUILD_FAILURE"
	}
	result.Outputs = []requesthit.ObservedOutput{{Path: "build/request-hit/forbidden.txt"}, {Path: "build/request-hit/result.txt"}}
	if _, err := os.Stat(resultPath); err == nil {
		result.Outputs[1].WorkspaceExists = true
		result.Outputs[1].WorkspaceSHA256 = digestFile(resultPath)
	}
	return result
}

func resetOutput(repository string) {
	output := filepath.Join(repository, "build")
	if err := os.RemoveAll(output); err != nil {
		fatalf("reset synthetic output: %v", err)
	}
}

func analyzePublic(root string) []publicFamily {
	observations := readJSONL[observation](filepath.Join(root, "observations.jsonl"))
	transitions := readJSONL[transition](filepath.Join(root, "transitions.jsonl"))
	byOrdinal := make(map[string]observation)
	byRevisionCommand := make(map[string]observation)
	for _, row := range observations {
		byOrdinal[fmt.Sprintf("%s\x00%d", row.Family, row.ObservationOrdinal)] = row
		byRevisionCommand[observationKey(row.Family, row.Revision, row.Command)] = row
	}
	families := make(map[string]*publicFamily)
	for _, row := range transitions {
		if row.Status != "IRRELEVANT_TO_REQUEST" {
			continue
		}
		family := families[row.Family]
		if family == nil {
			family = &publicFamily{
				Family: row.Family, DSL: row.DSL,
				ExactOutputOrdinals: []int{}, MismatchedOutputOrdinals: []int{},
				RetentionReason: "HISTORICAL_VRH002_SAFETY_RECORD_UNAVAILABLE",
			}
			families[row.Family] = family
		}
		family.PotentialRows++
		base, baseOK := byRevisionCommand[observationKey(row.Family, row.BaseRevision, row.Command)]
		target, targetOK := byOrdinal[fmt.Sprintf("%s\x00%d", row.Family, row.ObservationOrdinal)]
		if !baseOK || !targetOK || base.Revision != row.BaseRevision || target.Revision != row.TargetRevision {
			fatalf("public observation pair is unavailable: %s/%d", row.Family, row.ObservationOrdinal)
		}
		if base.ExitCode == 0 && target.ExitCode == 0 && !base.ProductFailure && !target.ProductFailure {
			family.NativeOutcomeMatches++
		}
		exact := samePublicRequestAndOutputs(root, base, target)
		if exact {
			family.NativeOutputMatches++
			family.ExactOutputOrdinals = append(family.ExactOutputOrdinals, row.ObservationOrdinal)
		} else {
			family.NativeOutputMismatches++
			family.MismatchedOutputOrdinals = append(family.MismatchedOutputOrdinals, row.ObservationOrdinal)
		}
	}
	result := make([]publicFamily, 0, len(families))
	for _, family := range families {
		result = append(result, *family)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Family < result[j].Family })
	return result
}

func samePublicRequestAndOutputs(root string, base, target observation) bool {
	if !base.CaptureComplete || !target.CaptureComplete {
		return false
	}
	baseEvidence := readGzipJSON[requestEvidence](filepath.Join(root, base.Evidence+".gz"))
	targetEvidence := readGzipJSON[requestEvidence](filepath.Join(root, target.Evidence+".gz"))
	if baseEvidence != targetEvidence {
		return false
	}
	baseCapture := readGzipJSON[capture](filepath.Join(root, base.Capture+".gz"))
	targetCapture := readGzipJSON[capture](filepath.Join(root, target.Capture+".gz"))
	if baseCapture.Status != "COMPLETE" || targetCapture.Status != "COMPLETE" {
		return false
	}
	return normalizedOutputs(baseCapture) == normalizedOutputs(targetCapture)
}

func normalizedOutputs(value capture) string {
	for index := range value.Tasks {
		sort.Slice(value.Tasks[index].Outputs, func(i, j int) bool {
			left, right := value.Tasks[index].Outputs[i], value.Tasks[index].Outputs[j]
			return left.Kind+"\x00"+left.Path < right.Kind+"\x00"+right.Path
		})
	}
	sort.Slice(value.Tasks, func(i, j int) bool { return value.Tasks[i].Path < value.Tasks[j].Path })
	normalized := make([]captureTask, len(value.Tasks))
	for index, task := range value.Tasks {
		normalized[index] = captureTask{Path: task.Path, Outputs: task.Outputs}
	}
	raw, err := json.Marshal(normalized)
	if err != nil {
		fatalf("normalize public outputs: %v", err)
	}
	return string(raw)
}

func readJSONL[T any](path string) []T {
	file, err := os.Open(path)
	if err != nil {
		fatalf("open %s: %v", path, err)
	}
	defer file.Close()
	var rows []T
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		var row T
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			fatalf("decode %s: %v", path, err)
		}
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		fatalf("scan %s: %v", path, err)
	}
	return rows
}

func readGzipJSON[T any](path string) T {
	file, err := os.Open(path)
	if err != nil {
		fatalf("open %s: %v", path, err)
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		fatalf("open gzip %s: %v", path, err)
	}
	defer reader.Close()
	var value T
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&value); err != nil {
		fatalf("decode %s: %v", path, err)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		fatalf("trailing JSON in %s", path)
	}
	return value
}

func observationKey(family, revision string, command []string) string {
	return family + "\x00" + revision + "\x00" + strings.Join(command, "\x00")
}

func filteredEnvironment(name string) []string {
	prefix := name + "="
	var result []string
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, prefix) {
			result = append(result, value)
		}
	}
	return result
}

func verificationTime() time.Time {
	return time.Date(2026, 8, 30, 9, 0, 0, 0, time.UTC)
}

func digestTree(root string) string {
	var rows []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symbolic fixture member: %s", path)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rows = append(rows, filepath.ToSlash(relative)+"\x00"+digestFile(path))
		return nil
	})
	if err != nil {
		fatalf("digest fixture tree: %v", err)
	}
	sort.Strings(rows)
	return digestStrings(rows...)
}

func digestFile(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		fatalf("read %s: %v", path, err)
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

func digestStrings(values ...string) string {
	hash := sha256.New()
	for _, value := range values {
		var length [8]byte
		for index := 0; index < 8; index++ {
			length[7-index] = byte(uint64(len(value)) >> (8 * index))
		}
		hash.Write(length[:])
		hash.Write([]byte(value))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func cleanAbsolute(value string) string {
	if !filepath.IsAbs(value) || filepath.Clean(value) != value {
		fatalf("path must be clean and absolute: %s", value)
	}
	return value
}

func writeJSON(path string, value any) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fatalf("create output directory: %v", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		fatalf("open output: %v", err)
	}
	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		_ = file.Close()
		fatalf("encode output: %v", err)
	}
	if err := file.Close(); err != nil {
		fatalf("close output: %v", err)
	}
}

func fatalf(format string, arguments ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", arguments...)
	os.Exit(1)
}
