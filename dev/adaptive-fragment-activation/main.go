// Command adaptive-fragment-activation executes the AF-010 real-Gradle proof
// for independently invalidated Build Impact producer fragments.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/tonyredondo/buildopt/internal/adaptivefragment"
)

const (
	reportSchema = "buildopt.poc/adaptive-fragment-activation/v1"
	outcome      = "COMPOSABLE_BUILD_IMPACT_AVAILABLE"
	repositoryID = "fixture/adaptive-fragment-activation"
)

type scenarioResult struct {
	Name                    string                                  `json:"name"`
	ChangeScope             adaptivefragment.BuildImpactChangeScope `json:"changeScope"`
	Disposition             string                                  `json:"disposition"`
	Reason                  string                                  `json:"reason"`
	PlanID                  string                                  `json:"planId"`
	PlannerPlanID           string                                  `json:"plannerPlanId,omitempty"`
	Entrypoints             []string                                `json:"entrypoints"`
	SelectedFragmentCount   int                                     `json:"selectedFragmentCount"`
	RestoredProducers       []string                                `json:"restoredProducers"`
	RebuiltProducers        []string                                `json:"rebuiltProducers"`
	ExecutedProducerTasks   []string                                `json:"executedProducerTasks"`
	NativeBundleSHA256      string                                  `json:"nativeBundleSha256"`
	CandidateBundleSHA256   string                                  `json:"candidateBundleSha256"`
	ExactProducerOutputs    bool                                    `json:"exactProducerOutputs"`
	ExactRequiredBundle     bool                                    `json:"exactRequiredBundle"`
	ProductAttributableFail bool                                    `json:"productAttributableFailure"`
}

type reportSummary struct {
	ScenarioCount                int `json:"scenarioCount"`
	PartialGraphCount            int `json:"partialGraphCount"`
	NativeFallbackCount          int `json:"nativeFallbackCount"`
	IndependentProducerChanges   int `json:"independentProducerChanges"`
	ExactOutputScenarioCount     int `json:"exactOutputScenarioCount"`
	UnaffectedOutputRestorations int `json:"unaffectedOutputRestorations"`
	LocalizedProducerRebuilds    int `json:"localizedProducerRebuilds"`
	GlobalOrAmbiguousFallbacks   int `json:"globalOrAmbiguousFallbacks"`
	ProductAttributableFailures  int `json:"productAttributableFailures"`
}

type boundaries struct {
	ProofOfConcept      bool   `json:"proofOfConcept"`
	RealGradleExecution bool   `json:"realGradleExecution"`
	SyntheticEconomics  bool   `json:"syntheticEconomics"`
	NewTimingClaim      bool   `json:"newTimingClaim"`
	ProductionGranted   bool   `json:"productionGranted"`
	TestOptimization    string `json:"testOptimization"`
}

type report struct {
	SchemaVersion string           `json:"schemaVersion"`
	WorkItem      string           `json:"workItem"`
	CapturedAt    string           `json:"capturedAt"`
	GradleVersion string           `json:"gradleVersion"`
	FixtureSHA256 string           `json:"fixtureSha256"`
	Scenarios     []scenarioResult `json:"scenarios"`
	Summary       reportSummary    `json:"summary"`
	Boundaries    boundaries       `json:"boundaries"`
	Outcome       string           `json:"outcome"`
}

type storedProducer struct {
	name            string
	producerID      string
	subgraph        adaptivefragment.PersistedFragment
	materialization adaptivefragment.PersistedFragment
	sourceRevision  string
	outputPath      string
	outputBytes     []byte
	outputSHA256    string
}

type scenarioDefinition struct {
	name         string
	changeScope  adaptivefragment.BuildImpactChangeScope
	changed      string
	missingStore string
}

func main() {
	repositoryRoot := flag.String("repository-root", "", "BuildOpt repository root")
	output := flag.String("output", "", "write the AF-010 report")
	validate := flag.String("validate", "", "validate an AF-010 report")
	flag.Parse()
	if flag.NArg() != 0 || *repositoryRoot == "" || (*output == "") == (*validate == "") {
		fmt.Fprintln(os.Stderr, "usage: adaptive-fragment-activation --repository-root <path> (--output <path> | --validate <path>)")
		os.Exit(64)
	}
	expected, err := buildReport(*repositoryRoot)
	if err == nil {
		err = validateReport(expected)
	}
	if err == nil && *validate != "" {
		var candidate report
		err = readJSONStrict(*validate, &candidate)
		if err == nil && !reflect.DeepEqual(candidate, expected) {
			err = errors.New("activation report does not match the recomputed real-Gradle proof")
		}
	}
	if err == nil && *output != "" {
		err = writeJSON(*output, expected)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "adaptive fragment activation failed: %v\n", err)
		os.Exit(1)
	}
	if *validate != "" {
		fmt.Println("adaptive fragment activation: COMPOSABLE_BUILD_IMPACT_AVAILABLE")
	}
}

func buildReport(repositoryRoot string) (report, error) {
	repositoryRoot, err := filepath.Abs(repositoryRoot)
	if err != nil {
		return report{}, err
	}
	fixtureRoot := filepath.Join(repositoryRoot, "fixtures", "adaptive-fragment-activation")
	gradlew := filepath.Join(repositoryRoot, "gradlew")
	if info, statErr := os.Stat(gradlew); statErr != nil || info.Mode()&0o111 == 0 {
		return report{}, errors.New("Gradle Wrapper is unavailable")
	}
	fixtureSHA, err := fixtureDigest(fixtureRoot)
	if err != nil {
		return report{}, err
	}
	tempRoot, err := os.MkdirTemp("", "buildopt-af010-*")
	if err != nil {
		return report{}, err
	}
	defer os.RemoveAll(tempRoot)

	seedRoot := filepath.Join(tempRoot, "seed")
	if err := copyTree(fixtureRoot, seedRoot); err != nil {
		return report{}, err
	}
	if _, err := runGradle(repositoryRoot, gradlew, seedRoot, []string{"fullBuild"}); err != nil {
		return report{}, fmt.Errorf("seed complete workflow: %w", err)
	}
	store, err := captureStore(seedRoot)
	if err != nil {
		return report{}, err
	}

	definitions := []scenarioDefinition{
		{name: "UNRELATED_CHANGE", changeScope: adaptivefragment.BuildImpactChangeUnrelated},
		{name: "PRODUCER_A_CHANGE", changeScope: adaptivefragment.BuildImpactChangeLocalized, changed: "producer-a"},
		{name: "PRODUCER_B_CHANGE", changeScope: adaptivefragment.BuildImpactChangeLocalized, changed: "producer-b"},
		{name: "BUILD_LOGIC_CHANGE", changeScope: adaptivefragment.BuildImpactChangeGlobal, changed: "build-logic"},
		{name: "AMBIGUOUS_CHANGE", changeScope: adaptivefragment.BuildImpactChangeAmbiguous},
		{name: "MISSING_STORED_OUTPUT", changeScope: adaptivefragment.BuildImpactChangeUnrelated, missingStore: "producer-a"},
	}
	result := report{
		SchemaVersion: reportSchema, WorkItem: "AF-010", CapturedAt: "2026-08-25T20:00:00Z",
		GradleVersion: "9.6.1", FixtureSHA256: fixtureSHA,
		Scenarios: []scenarioResult{},
		Boundaries: boundaries{
			ProofOfConcept: true, RealGradleExecution: true, SyntheticEconomics: true,
			TestOptimization: "OUT_OF_SCOPE",
		},
		Outcome: outcome,
	}
	for index, definition := range definitions {
		scenario, scenarioErr := runScenario(repositoryRoot, gradlew, fixtureRoot, tempRoot, index, definition, store)
		if scenarioErr != nil {
			return report{}, scenarioErr
		}
		result.Scenarios = append(result.Scenarios, scenario)
	}
	result.Summary = summarize(result.Scenarios)
	return result, nil
}

func runScenario(repositoryRoot, gradlew, fixtureRoot, tempRoot string, index int, definition scenarioDefinition, store []storedProducer) (scenarioResult, error) {
	nativeRoot := filepath.Join(tempRoot, fmt.Sprintf("%02d-%s-native", index, strings.ToLower(definition.name)))
	candidateRoot := filepath.Join(tempRoot, fmt.Sprintf("%02d-%s-candidate", index, strings.ToLower(definition.name)))
	if err := copyTree(fixtureRoot, nativeRoot); err != nil {
		return scenarioResult{}, err
	}
	if err := copyTree(fixtureRoot, candidateRoot); err != nil {
		return scenarioResult{}, err
	}
	if err := applyScenarioMutation(nativeRoot, definition); err != nil {
		return scenarioResult{}, err
	}
	if err := applyScenarioMutation(candidateRoot, definition); err != nil {
		return scenarioResult{}, err
	}
	nativeLog, err := runGradle(repositoryRoot, gradlew, nativeRoot, []string{"fullBuild"})
	if err != nil {
		return scenarioResult{}, fmt.Errorf("%s native workflow: %w", definition.name, err)
	}
	request, err := activationRequest(candidateRoot, definition, store)
	if err != nil {
		return scenarioResult{}, err
	}
	plan := adaptivefragment.ActivateBuildImpactFragments(request)
	if err := materializeSelectedOutputs(candidateRoot, plan, store); err != nil {
		return scenarioResult{}, fmt.Errorf("%s restore outputs: %w", definition.name, err)
	}
	candidateLog, err := runGradle(repositoryRoot, gradlew, candidateRoot, plan.Entrypoints)
	if err != nil {
		return scenarioResult{}, fmt.Errorf("%s candidate workflow: %w", definition.name, err)
	}
	nativeBundle, err := fileSHA(filepath.Join(nativeRoot, "build", "distribution", "adaptive-fragments.zip"))
	if err != nil {
		return scenarioResult{}, err
	}
	candidateBundle, err := fileSHA(filepath.Join(candidateRoot, "build", "distribution", "adaptive-fragments.zip"))
	if err != nil {
		return scenarioResult{}, err
	}
	exactProducers := true
	for _, producer := range store {
		nativeSHA, nativeErr := fileSHA(filepath.Join(nativeRoot, filepath.FromSlash(producer.outputPath)))
		candidateSHA, candidateErr := fileSHA(filepath.Join(candidateRoot, filepath.FromSlash(producer.outputPath)))
		if nativeErr != nil || candidateErr != nil || nativeSHA != candidateSHA {
			exactProducers = false
		}
	}
	restored, rebuilt := activationProducerSets(plan.Producers)
	executed := executedProducerTasks(candidateLog)
	if !strings.Contains(nativeLog, "> Task :producer-a:produce") || !strings.Contains(nativeLog, "> Task :producer-b:produce") {
		return scenarioResult{}, errors.New("native complete workflow did not execute both producers")
	}
	return scenarioResult{
		Name: definition.name, ChangeScope: definition.changeScope,
		Disposition: plan.Disposition, Reason: plan.Reason, PlanID: plan.PlanID, PlannerPlanID: plan.PlannerPlanID,
		Entrypoints: append([]string{}, plan.Entrypoints...), SelectedFragmentCount: len(plan.SelectedFragments),
		RestoredProducers: restored, RebuiltProducers: rebuilt, ExecutedProducerTasks: executed,
		NativeBundleSHA256: nativeBundle, CandidateBundleSHA256: candidateBundle,
		ExactProducerOutputs: exactProducers, ExactRequiredBundle: nativeBundle == candidateBundle,
	}, nil
}

func activationRequest(root string, definition scenarioDefinition, store []storedProducer) (adaptivefragment.BuildImpactActivationRequest, error) {
	producers := make([]adaptivefragment.BuildImpactProducerPair, 0, len(store))
	for _, stored := range store {
		currentRevision, err := fileSHA(filepath.Join(root, stored.name, "src", "value.txt"))
		if err != nil {
			return adaptivefragment.BuildImpactActivationRequest{}, err
		}
		subgraphContext := adaptivefragment.Context{RepositoryID: repositoryID, Bindings: cloneBindings(stored.subgraph.Bindings), Ambiguous: []adaptivefragment.BindingKey{}}
		materializationContext := adaptivefragment.Context{RepositoryID: repositoryID, Bindings: cloneBindings(stored.materialization.Bindings), Ambiguous: []adaptivefragment.BindingKey{}}
		if currentRevision != stored.sourceRevision {
			subgraphContext.Bindings[adaptivefragment.BindingChangeFamily] = hashStrings("changed-family", stored.name, currentRevision)
		}
		outputs := []adaptivefragment.StoredBuildImpactOutput{{RelativePath: stored.outputPath, ContentSHA256: stored.outputSHA256}}
		if definition.missingStore == stored.name {
			outputs = []adaptivefragment.StoredBuildImpactOutput{}
		}
		producers = append(producers, adaptivefragment.BuildImpactProducerPair{
			ProducerID: stored.producerID, Subgraph: stored.subgraph, Materialization: stored.materialization,
			SubgraphContext: subgraphContext, MaterializationContext: materializationContext,
			CurrentOutputRevisionSHA256: currentRevision, StoredOutputRevisionSHA256: stored.sourceRevision,
			StoredOutputs: outputs, RebuildEntrypoints: []string{":" + stored.name + ":produce"},
		})
	}
	predictions := []adaptivefragment.CompositionPrediction{
		prediction(1200, producers[0].Subgraph.FamilyID, producers[0].Materialization.FamilyID),
		prediction(1400, producers[1].Subgraph.FamilyID, producers[1].Materialization.FamilyID),
		prediction(3200,
			producers[0].Subgraph.FamilyID, producers[0].Materialization.FamilyID,
			producers[1].Subgraph.FamilyID, producers[1].Materialization.FamilyID,
		),
	}
	return adaptivefragment.BuildImpactActivationRequest{
		RepositoryScopeSHA256: producers[0].Subgraph.RepositoryScopeSHA256,
		NativeWorkflowSHA256:  hashStrings("native-workflow", "fullBuild"),
		DecisionAt:            "2026-08-25T19:30:00Z", ChangeScope: definition.changeScope,
		MinimumPredictedNetMs: 100, NativeEntrypoints: []string{"fullBuild"},
		FinalizationEntrypoints: []string{"packageAll"}, Producers: producers, Predictions: predictions,
	}, nil
}

func captureStore(root string) ([]storedProducer, error) {
	result := []storedProducer{}
	for _, name := range []string{"producer-a", "producer-b"} {
		materialization, err := deriveFragment(adaptivefragment.KindOutputMaterialization, adaptivefragment.AuthorityVerifiedProducer, name+"-materialization", nil, name)
		if err != nil {
			return nil, err
		}
		subgraph, err := deriveFragment(adaptivefragment.KindSubgraph, adaptivefragment.AuthorityGradleModel, name+"-subgraph", []string{materialization.FamilyID}, name)
		if err != nil {
			return nil, err
		}
		outputPath := filepath.ToSlash(filepath.Join(name, "build", "fragment", "value.txt"))
		outputBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(outputPath)))
		if err != nil {
			return nil, err
		}
		sourceRevision, err := fileSHA(filepath.Join(root, name, "src", "value.txt"))
		if err != nil {
			return nil, err
		}
		result = append(result, storedProducer{
			name: name, producerID: hashStrings("producer", name),
			subgraph: persisted(subgraph), materialization: persisted(materialization),
			sourceRevision: sourceRevision, outputPath: outputPath,
			outputBytes: append([]byte{}, outputBytes...), outputSHA256: hashBytes(outputBytes),
		})
	}
	return result, nil
}

func deriveFragment(kind adaptivefragment.Kind, authority adaptivefragment.Authority, selector string, requires []string, producer string) (adaptivefragment.Fragment, error) {
	bindings := map[adaptivefragment.BindingKey]string{
		adaptivefragment.BindingWrapper:         hashStrings("wrapper", "gradle-9.6.1"),
		adaptivefragment.BindingProducerLineage: hashStrings("producer-lineage", producer),
		adaptivefragment.BindingOutputContract:  hashStrings("output-contract", producer),
	}
	if kind == adaptivefragment.KindSubgraph {
		bindings[adaptivefragment.BindingWorkflow] = hashStrings("workflow", "fullBuild")
		bindings[adaptivefragment.BindingChangeFamily] = hashStrings("change-family", producer, "unaffected")
	}
	return adaptivefragment.Derive(adaptivefragment.Input{
		RepositoryID: repositoryID, Kind: kind, Selector: []string{selector},
		Authority: authority, AuthoritySHA256: hashStrings("authority", selector),
		Bindings: bindings, Requires: requires, ConflictsWith: []string{},
	})
}

func persisted(fragment adaptivefragment.Fragment) adaptivefragment.PersistedFragment {
	return adaptivefragment.PersistedFragment{
		SchemaVersion: adaptivefragment.FragmentStateSchemaVersion, RecordType: "ADAPTIVE_FRAGMENT",
		FamilyID: fragment.FamilyID, RevisionID: fragment.RevisionID, RepositoryScopeSHA256: fragment.RepositoryScopeSHA256,
		Kind: fragment.Kind, SelectorSHA256: fragment.SelectorSHA256, Authority: fragment.Authority,
		AuthoritySHA256: fragment.AuthoritySHA256, Bindings: cloneBindings(fragment.Bindings),
		Requires: append([]string{}, fragment.Requires...), ConflictsWith: append([]string{}, fragment.ConflictsWith...),
		State: adaptivefragment.StateQualified, Generation: 1,
		CreatedAt: "2026-08-25T17:00:00Z", UpdatedAt: "2026-08-25T18:00:00Z", EvidenceExpiresAt: "2026-09-25T18:00:00Z",
	}
}

func prediction(value int64, families ...string) adaptivefragment.CompositionPrediction {
	copyFamilies := append([]string{}, families...)
	sort.Strings(copyFamilies)
	return adaptivefragment.CompositionPrediction{
		Families: copyFamilies, EvidenceSHA256: hashStrings("prediction", strings.Join(copyFamilies, ",")),
		HorizonBuilds: 5, PredictedNetMs: value,
	}
}

func applyScenarioMutation(root string, definition scenarioDefinition) error {
	switch definition.changed {
	case "":
		return os.WriteFile(filepath.Join(root, "unrelated-note.txt"), []byte("unrelated\n"), 0o644)
	case "producer-a", "producer-b":
		return os.WriteFile(filepath.Join(root, definition.changed, "src", "value.txt"), []byte(definition.changed+"-v2\n"), 0o644)
	case "build-logic":
		file := filepath.Join(root, "build.gradle.kts")
		content, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		return os.WriteFile(file, append(content, []byte("\n// global build-logic change\n")...), 0o644)
	default:
		return errors.New("unsupported activation scenario mutation")
	}
}

func materializeSelectedOutputs(root string, plan adaptivefragment.BuildImpactActivationPlan, store []storedProducer) error {
	storedByPath := map[string]storedProducer{}
	for _, producer := range store {
		storedByPath[producer.outputPath] = producer
	}
	for _, restoration := range plan.Restorations {
		producer, exists := storedByPath[restoration.RelativePath]
		if !exists || hashBytes(producer.outputBytes) != restoration.ContentSHA256 {
			return errors.New("restoration does not match the verified store")
		}
		target := filepath.Join(root, filepath.FromSlash(restoration.RelativePath))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, producer.outputBytes, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func runGradle(repositoryRoot, gradlew, projectRoot string, tasks []string) (string, error) {
	args := []string{"--offline", "--no-daemon", "--console=plain", "-p", projectRoot}
	args = append(args, tasks...)
	command := exec.Command(gradlew, args...)
	environment := []string{}
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "GRADLE_USER_HOME=") {
			environment = append(environment, value)
		}
	}
	command.Env = append(environment, "GRADLE_USER_HOME="+filepath.Join(repositoryRoot, ".tools", "gradle-user-home", "local"))
	output, err := command.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("%w\n%s", err, output)
	}
	return string(output), nil
}

func executedProducerTasks(output string) []string {
	result := []string{}
	for _, producer := range []string{"producer-a", "producer-b"} {
		if strings.Contains(output, "> Task :"+producer+":produce") {
			result = append(result, producer)
		}
	}
	return result
}

func activationProducerSets(decisions []adaptivefragment.BuildImpactProducerDecision) ([]string, []string) {
	restored, rebuilt := []string{}, []string{}
	for _, decision := range decisions {
		switch decision.Disposition {
		case "RESTORE":
			restored = append(restored, decision.ProducerID)
		case "REBUILD":
			rebuilt = append(rebuilt, decision.ProducerID)
		}
	}
	sort.Strings(restored)
	sort.Strings(rebuilt)
	return restored, rebuilt
}

func summarize(scenarios []scenarioResult) reportSummary {
	result := reportSummary{ScenarioCount: len(scenarios)}
	for _, scenario := range scenarios {
		if scenario.Disposition == "PARTIAL_GRAPH" {
			result.PartialGraphCount++
		} else if scenario.Disposition == "NATIVE_GRADLE" {
			result.NativeFallbackCount++
		}
		if scenario.Name == "PRODUCER_A_CHANGE" || scenario.Name == "PRODUCER_B_CHANGE" {
			result.IndependentProducerChanges++
			result.LocalizedProducerRebuilds += len(scenario.RebuiltProducers)
		}
		if scenario.ExactProducerOutputs && scenario.ExactRequiredBundle {
			result.ExactOutputScenarioCount++
		}
		result.UnaffectedOutputRestorations += len(scenario.RestoredProducers)
		if scenario.Name == "BUILD_LOGIC_CHANGE" || scenario.Name == "AMBIGUOUS_CHANGE" {
			if scenario.Disposition == "NATIVE_GRADLE" {
				result.GlobalOrAmbiguousFallbacks++
			}
		}
		if scenario.ProductAttributableFail {
			result.ProductAttributableFailures++
		}
	}
	return result
}

func validateReport(candidate report) error {
	if candidate.SchemaVersion != reportSchema || candidate.WorkItem != "AF-010" || candidate.Outcome != outcome ||
		candidate.GradleVersion != "9.6.1" || !validSHA(candidate.FixtureSHA256) ||
		candidate.Summary.ScenarioCount != 6 || candidate.Summary.PartialGraphCount != 3 || candidate.Summary.NativeFallbackCount != 3 ||
		candidate.Summary.IndependentProducerChanges != 2 || candidate.Summary.ExactOutputScenarioCount != 6 ||
		candidate.Summary.UnaffectedOutputRestorations != 4 || candidate.Summary.LocalizedProducerRebuilds != 2 ||
		candidate.Summary.GlobalOrAmbiguousFallbacks != 2 || candidate.Summary.ProductAttributableFailures != 0 ||
		!candidate.Boundaries.ProofOfConcept || !candidate.Boundaries.RealGradleExecution || !candidate.Boundaries.SyntheticEconomics ||
		candidate.Boundaries.NewTimingClaim || candidate.Boundaries.ProductionGranted || candidate.Boundaries.TestOptimization != "OUT_OF_SCOPE" {
		return errors.New("activation report invariant failed")
	}
	expected := map[string]struct {
		disposition string
		reason      string
		restored    int
		rebuilt     int
		executed    int
	}{
		"UNRELATED_CHANGE":      {"PARTIAL_GRAPH", "COMPOSABLE_BUILD_IMPACT", 2, 0, 0},
		"PRODUCER_A_CHANGE":     {"PARTIAL_GRAPH", "COMPOSABLE_BUILD_IMPACT", 1, 1, 1},
		"PRODUCER_B_CHANGE":     {"PARTIAL_GRAPH", "COMPOSABLE_BUILD_IMPACT", 1, 1, 1},
		"BUILD_LOGIC_CHANGE":    {"NATIVE_GRADLE", "GLOBAL_CHANGE", 0, 0, 2},
		"AMBIGUOUS_CHANGE":      {"NATIVE_GRADLE", "AMBIGUOUS_CHANGE", 0, 0, 2},
		"MISSING_STORED_OUTPUT": {"NATIVE_GRADLE", "INVALID_STORED_OUTPUT", 0, 0, 2},
	}
	if len(candidate.Scenarios) != len(expected) {
		return errors.New("activation scenario coverage is incomplete")
	}
	for _, scenario := range candidate.Scenarios {
		want, exists := expected[scenario.Name]
		if !exists || scenario.Disposition != want.disposition || scenario.Reason != want.reason ||
			len(scenario.RestoredProducers) != want.restored || len(scenario.RebuiltProducers) != want.rebuilt ||
			len(scenario.ExecutedProducerTasks) != want.executed || !scenario.ExactProducerOutputs || !scenario.ExactRequiredBundle ||
			scenario.NativeBundleSHA256 != scenario.CandidateBundleSHA256 || scenario.ProductAttributableFail || scenario.PlanID == "" {
			return fmt.Errorf("activation scenario is unsafe: %s", scenario.Name)
		}
	}
	return nil
}

func fixtureDigest(root string) (string, error) {
	rows := []string{}
	err := filepath.WalkDir(root, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, current)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(current)
		if err != nil {
			return err
		}
		rows = append(rows, filepath.ToSlash(relative)+"="+hashBytes(content))
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(rows)
	return hashStrings(append([]string{"fixture"}, rows...)...), nil
}

func copyTree(sourceRoot, destinationRoot string) error {
	return filepath.WalkDir(sourceRoot, func(source string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(sourceRoot, source)
		if err != nil {
			return err
		}
		destination := filepath.Join(destinationRoot, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		input, err := os.Open(source)
		if err != nil {
			return err
		}
		defer input.Close()
		output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		if _, err := io.Copy(output, input); err != nil {
			output.Close()
			return err
		}
		return output.Close()
	})
}

func fileSHA(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return hashBytes(content), nil
}

func hashBytes(content []byte) string {
	digest := sha256.Sum256(content)
	return hex.EncodeToString(digest[:])
}

func hashStrings(values ...string) string {
	digest := sha256.New()
	for _, value := range values {
		_, _ = io.WriteString(digest, value)
		_, _ = io.WriteString(digest, "\x00")
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func cloneBindings(values map[adaptivefragment.BindingKey]string) map[adaptivefragment.BindingKey]string {
	result := make(map[adaptivefragment.BindingKey]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func validSHA(value string) bool {
	if len(value) != sha256.Size*2 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func readJSONStrict(path string, target any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return errors.New("activation report contains trailing data")
	}
	return nil
}

func writeJSON(path string, value any) error {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	return os.WriteFile(path, content, 0o644)
}
