// Command request-aligned-cause-audit explains the terminal request-aligned
// POC result from its immutable capture ledger. It does not execute actions or
// authorize performance timing.
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/tonyredondo/buildopt/internal/changeaware"
	"github.com/tonyredondo/buildopt/internal/requestaligned"
)

const (
	schemaVersion = "buildopt.poc/request-aligned-terminal-cause-audit/v1"
	workItem      = "SWL-PORTFOLIO-000"
)

type ledgerRow struct {
	Family               string   `json:"family"`
	AttemptOrdinal       int      `json:"attemptOrdinal"`
	ChangedPaths         []string `json:"changedPaths"`
	BaseCapture          string   `json:"baseCapture"`
	TargetCapture        string   `json:"targetCapture"`
	Report               string   `json:"report"`
	Status               string   `json:"status"`
	Reason               string   `json:"reason"`
	TestableActions      int      `json:"testableActions"`
	PerformanceMeasured  bool     `json:"performanceMeasured"`
	ActivationAuthorized bool     `json:"activationAuthorized"`
}

type terminalDecision struct {
	GeneratedAt         string `json:"generatedAt"`
	Decision            string `json:"decision"`
	SpeedupClaim        bool   `json:"speedupClaim"`
	SuccessorAuthorized bool   `json:"successorAuthorized"`
}

type evidence struct {
	RouteContractSHA256      string `json:"routeContractSha256"`
	TerminalDecisionSHA256   string `json:"terminalDecisionSha256"`
	CaptureContractSHA256    string `json:"captureContractSha256"`
	TransitionsSHA256        string `json:"transitionsSha256"`
	TransitionCount          int    `json:"transitionCount"`
	HistoricalTimingImported bool   `json:"historicalTimingImported"`
	HistoricalActionImported bool   `json:"historicalActionImported"`
}

type identityPattern struct {
	Pattern     string `json:"pattern"`
	Transitions int    `json:"transitions"`
}

type cause struct {
	Cause                       string            `json:"cause"`
	Transitions                 int               `json:"transitions"`
	Families                    []string          `json:"families"`
	RequestInputIntersections   int               `json:"requestInputIntersections"`
	IdentityPatterns            []identityPattern `json:"identityPatterns,omitempty"`
	EquivalentOutputPath        string            `json:"equivalentOutputPath,omitempty"`
	EquivalentProducers         []string          `json:"equivalentProducers,omitempty"`
	DependencyOrdered           *bool             `json:"dependencyOrdered,omitempty"`
	SameBytesWithinCapture      *bool             `json:"sameBytesWithinCapture,omitempty"`
	RequiresAbsentOutputBinding bool              `json:"requiresAbsentOutputBinding,omitempty"`
	AbsentOutputPath            string            `json:"absentOutputPath,omitempty"`
	RecoverableRelevantRows     int               `json:"recoverableRelevantRows"`
	RecoverableActionRows       int               `json:"recoverableActionRows"`
	Disposition                 string            `json:"disposition"`
}

type familyAssessment struct {
	Family                        string `json:"family"`
	ObservedTransitions           int    `json:"observedTransitions"`
	CurrentRelevantActions        int    `json:"currentRelevantActions"`
	RecoverableRelevantActions    int    `json:"recoverableRelevantActions"`
	CounterfactualRelevantActions int    `json:"counterfactualRelevantActions"`
	MeetsFiveRelevantActions      bool   `json:"meetsFiveRelevantActions"`
}

type result struct {
	SchemaVersion string   `json:"schemaVersion"`
	WorkItem      string   `json:"workItem"`
	GeneratedAt   string   `json:"generatedAt"`
	Evidence      evidence `json:"evidence"`
	Population    struct {
		Transitions       int `json:"transitions"`
		CurrentActionRows int `json:"currentActionRows"`
		NonActionRows     int `json:"nonActionRows"`
	} `json:"population"`
	CauseAnalysis  []cause `json:"causeAnalysis"`
	Counterfactual struct {
		FamilyAssessments           []familyAssessment `json:"familyAssessments"`
		CurrentActionFamilies       int                `json:"currentActionFamilies"`
		RecoverableActionFamilies   int                `json:"recoverableActionFamilies"`
		FamiliesMeetingFiveRelevant int                `json:"familiesMeetingFiveRelevant"`
		RequiredFamiliesMeetingFive int                `json:"requiredFamiliesMeetingFive"`
		EvidenceRepairsSatisfyGate  bool               `json:"evidenceRepairsSatisfyGate"`
	} `json:"counterfactual"`
	Decision struct {
		Outcome                     string `json:"outcome"`
		SelectedHypothesis          string `json:"selectedHypothesis"`
		Reason                      string `json:"reason"`
		NextBlock                   string `json:"nextBlock"`
		CommandSubstitutionAllowed  bool   `json:"commandSubstitutionAllowed"`
		RepositoryRulesAllowed      bool   `json:"repositoryRulesAllowed"`
		PerformanceTimingAuthorized bool   `json:"performanceTimingAuthorized"`
		ActionActivationAuthorized  bool   `json:"actionActivationAuthorized"`
		SpeedupClaim                bool   `json:"speedupClaim"`
	} `json:"decision"`
}

func main() {
	flags := flag.NewFlagSet("request-aligned-cause-audit", flag.ExitOnError)
	captureDir := flags.String("capture-dir", "", "immutable request-aligned capture directory")
	terminalPath := flags.String("terminal", "", "terminal request-aligned decision")
	routeContract := flags.String("route-contract", "", "observed request portfolio contract")
	outputPath := flags.String("output", "", "cause-audit result")
	_ = flags.Parse(os.Args[1:])
	if flags.NArg() != 0 || *captureDir == "" || *terminalPath == "" || *routeContract == "" || *outputPath == "" {
		fail(errors.New("usage: request-aligned-cause-audit --capture-dir PATH --terminal PATH --route-contract PATH --output PATH"))
	}

	audit, err := analyze(*captureDir, *terminalPath, *routeContract)
	if err != nil {
		fail(err)
	}
	if err := writeJSON(*outputPath, audit); err != nil {
		fail(err)
	}
}

func analyze(captureDir, terminalPath, routeContract string) (result, error) {
	var terminal terminalDecision
	if err := readJSON(terminalPath, &terminal, false); err != nil {
		return result{}, err
	}
	if terminal.Decision != "STOP_REQUEST_ALIGNED_RECURRENT_CLOSURE_POC_FOR_CURRENT_DETECTOR" ||
		terminal.SpeedupClaim || terminal.SuccessorAuthorized || terminal.GeneratedAt == "" {
		return result{}, errors.New("terminal request-aligned decision is not the closed non-authorizing result")
	}

	rows, err := readLedger(filepath.Join(captureDir, "transitions.jsonl"))
	if err != nil {
		return result{}, err
	}
	if len(rows) != 110 {
		return result{}, fmt.Errorf("expected 110 transitions, got %d", len(rows))
	}

	audit := result{SchemaVersion: schemaVersion, WorkItem: workItem, GeneratedAt: terminal.GeneratedAt}
	audit.Evidence = evidence{
		RouteContractSHA256:    mustSHA(routeContract),
		TerminalDecisionSHA256: mustSHA(terminalPath),
		CaptureContractSHA256:  mustSHA(filepath.Join(captureDir, "contract.json")),
		TransitionsSHA256:      mustSHA(filepath.Join(captureDir, "transitions.jsonl")),
		TransitionCount:        len(rows), HistoricalTimingImported: false, HistoricalActionImported: false,
	}
	audit.Population.Transitions = len(rows)

	familyOrder := []string{}
	familySeen := map[string]bool{}
	observed := map[string]int{}
	currentActions := map[string]int{}
	recoverableActions := map[string]int{}
	causes := map[string]int{}
	causeFamilies := map[string]map[string]bool{}
	identityPatterns := map[string]int{}
	identityIntersections := 0
	ambiguousIntersections := 0
	ambiguousActions := 0
	absentIntersections := 0
	absentActions := 0
	var aliasPath string
	var aliasProducers []string
	aliasOrdered := true
	aliasSameBytes := true
	var absentPath string

	for _, row := range rows {
		if row.Family == "" || row.AttemptOrdinal <= 0 || row.PerformanceMeasured || row.ActivationAuthorized {
			return result{}, errors.New("transition ledger contains invalid or authorizing evidence")
		}
		if !familySeen[row.Family] {
			familySeen[row.Family] = true
			familyOrder = append(familyOrder, row.Family)
		}
		observed[row.Family]++
		if row.TestableActions > 0 {
			currentActions[row.Family] += row.TestableActions
			audit.Population.CurrentActionRows += row.TestableActions
			continue
		}

		causeKey := row.Reason
		causes[causeKey]++
		if causeFamilies[causeKey] == nil {
			causeFamilies[causeKey] = map[string]bool{}
		}
		causeFamilies[causeKey][row.Family] = true

		base, target, err := readCaptures(captureDir, row)
		if err != nil {
			return result{}, err
		}
		intersections := inputIntersections(row.ChangedPaths, target.Tasks)

		switch row.Reason {
		case "NO_CHANGED_PATH_INTERSECTS_REQUEST_INPUTS":
			if intersections != 0 {
				return result{}, errors.New("irrelevant transition intersects a request input")
			}
		case "REQUEST_IDENTITY_CHANGED":
			identityIntersections += intersections
			identityPatterns[identityPatternFor(base, target)]++
		case "CURRENT_OUTPUT_PRODUCER_AMBIGUOUS":
			groups := equivalentOutputGroups(target.Tasks)
			if len(groups) != 1 || len(groups[0].Producers) != 2 {
				return result{}, errors.New("ambiguous transition is not one equivalent two-producer output group")
			}
			group := groups[0]
			if aliasPath == "" {
				aliasPath, aliasProducers = group.Path, append([]string(nil), group.Producers...)
			}
			if aliasPath != group.Path || !reflect.DeepEqual(aliasProducers, group.Producers) {
				return result{}, errors.New("ambiguous producer group changes across the capture")
			}
			aliasSameBytes = aliasSameBytes && group.SameBytes
			aliasOrdered = aliasOrdered && dependencyOrdered(target.Tasks, group.Producers[0], group.Producers[1])
			ambiguousIntersections += intersections
			if intersections > 0 && hasRecoverablePartialClosure(row.ChangedPaths, target) {
				ambiguousActions++
				recoverableActions[row.Family]++
			}
		case "OMITTED_OUTPUT_EVIDENCE_INCOMPLETE":
			var report requestaligned.Classification
			if err := readStrict(filepath.Join(captureDir, row.Report), &report); err != nil {
				return result{}, err
			}
			missing := absentOmittedOutputs(report, target)
			if len(missing) != 1 {
				return result{}, errors.New("incomplete omitted-output row is not one absent output")
			}
			absentPath = missing[0]
			absentIntersections += intersections
			if intersections > 0 && len(report.CandidateTasks) > 0 && len(report.OmittedTasks) > 0 {
				absentActions++
				recoverableActions[row.Family]++
			}
		default:
			return result{}, fmt.Errorf("unexpected non-action reason %q", row.Reason)
		}
	}

	audit.Population.NonActionRows = audit.Population.Transitions - audit.Population.CurrentActionRows
	if audit.Population.CurrentActionRows != 10 || audit.Population.NonActionRows != 100 {
		return result{}, errors.New("terminal action/non-action population changed")
	}
	if causes["NO_CHANGED_PATH_INTERSECTS_REQUEST_INPUTS"] != 44 ||
		causes["REQUEST_IDENTITY_CHANGED"] != 25 ||
		causes["CURRENT_OUTPUT_PRODUCER_AMBIGUOUS"] != 30 ||
		causes["OMITTED_OUTPUT_EVIDENCE_INCOMPLETE"] != 1 {
		return result{}, errors.New("terminal cause population changed")
	}
	if identityIntersections != 0 || ambiguousIntersections <= 0 || absentIntersections <= 0 ||
		ambiguousActions != 2 || absentActions != 1 || !aliasOrdered || !aliasSameBytes {
		return result{}, errors.New("recoverability facts changed")
	}

	trueValue := true
	audit.CauseAnalysis = []cause{
		{Cause: "NO_CHANGED_PATH_INTERSECTS_REQUEST_INPUTS", Transitions: 44,
			Families:                  sortedKeys(causeFamilies["NO_CHANGED_PATH_INTERSECTS_REQUEST_INPUTS"]),
			RequestInputIntersections: 0, RecoverableRelevantRows: 0, RecoverableActionRows: 0,
			Disposition: "EXPECTED_NATIVE_RETENTION_FOR_THIS_EXACT_REQUEST"},
		{Cause: "REQUEST_IDENTITY_CHANGED", Transitions: 25,
			Families:                  sortedKeys(causeFamilies["REQUEST_IDENTITY_CHANGED"]),
			RequestInputIntersections: identityIntersections, IdentityPatterns: sortedPatterns(identityPatterns),
			RecoverableRelevantRows: 0, RecoverableActionRows: 0,
			Disposition: "IDENTITY_PRECISION_CAN_IMPROVE_CLASSIFICATION_BUT_CANNOT_CREATE_REQUEST_RELEVANCE"},
		{Cause: "CURRENT_OUTPUT_PRODUCER_AMBIGUOUS", Transitions: 30,
			Families:                  sortedKeys(causeFamilies["CURRENT_OUTPUT_PRODUCER_AMBIGUOUS"]),
			RequestInputIntersections: ambiguousIntersections, EquivalentOutputPath: aliasPath,
			EquivalentProducers: aliasProducers, DependencyOrdered: &trueValue, SameBytesWithinCapture: &trueValue,
			RequiresAbsentOutputBinding: true,
			RecoverableRelevantRows:     ambiguousActions, RecoverableActionRows: ambiguousActions,
			Disposition: "GENERIC_EQUIVALENT_PRODUCER_GROUP_PLUS_ABSENCE_BINDING_CAN_RECOVER_TWO_RELEVANT_KAFKA_ROWS"},
		{Cause: "OMITTED_OUTPUT_EVIDENCE_INCOMPLETE", Transitions: 1,
			Families:                  sortedKeys(causeFamilies["OMITTED_OUTPUT_EVIDENCE_INCOMPLETE"]),
			RequestInputIntersections: absentIntersections, AbsentOutputPath: absentPath,
			RecoverableRelevantRows: absentActions, RecoverableActionRows: absentActions,
			Disposition: "GENERIC_ABSENCE_BINDING_CAN_RECOVER_ONE_RELEVANT_MICRONAUT_ROW"},
	}

	for _, family := range familyOrder {
		assessment := familyAssessment{
			Family: family, ObservedTransitions: observed[family],
			CurrentRelevantActions:        currentActions[family],
			RecoverableRelevantActions:    recoverableActions[family],
			CounterfactualRelevantActions: currentActions[family] + recoverableActions[family],
		}
		assessment.MeetsFiveRelevantActions = assessment.CounterfactualRelevantActions >= 5
		audit.Counterfactual.FamilyAssessments = append(audit.Counterfactual.FamilyAssessments, assessment)
		if assessment.CurrentRelevantActions > 0 {
			audit.Counterfactual.CurrentActionFamilies++
		}
		if assessment.CounterfactualRelevantActions > 0 {
			audit.Counterfactual.RecoverableActionFamilies++
		}
		if assessment.MeetsFiveRelevantActions {
			audit.Counterfactual.FamiliesMeetingFiveRelevant++
		}
	}
	audit.Counterfactual.RequiredFamiliesMeetingFive = 5
	audit.Counterfactual.EvidenceRepairsSatisfyGate = audit.Counterfactual.FamiliesMeetingFiveRelevant >= 5

	audit.Decision.Outcome = "PREREGISTER_OBSERVED_RECURRENT_REQUEST_PORTFOLIO_V1"
	audit.Decision.SelectedHypothesis = "OBSERVED_RECURRENT_REQUEST_PORTFOLIO_V1"
	audit.Decision.Reason = "EVIDENCE_REPAIRS_RECOVER_THREE_RELEVANT_ROWS_BUT_FIXED_SINGLE_REQUESTS_STILL_MEET_THE_FIVE_RELEVANT_THRESHOLD_IN_ONLY_TWO_OF_FIVE_FAMILIES"
	audit.Decision.NextBlock = "SWL-PORTFOLIO-001"
	audit.Decision.CommandSubstitutionAllowed = false
	audit.Decision.RepositoryRulesAllowed = false
	audit.Decision.PerformanceTimingAuthorized = false
	audit.Decision.ActionActivationAuthorized = false
	audit.Decision.SpeedupClaim = false
	return audit, nil
}

type outputGroup struct {
	Path      string
	Producers []string
	SameBytes bool
}

func equivalentOutputGroups(tasks []changeaware.TaskEvidence) []outputGroup {
	type row struct{ producer, sha string }
	groups := map[string][]row{}
	for _, task := range tasks {
		for _, output := range task.Outputs {
			if output.Exists {
				key := output.Kind + "\x00" + output.Path
				groups[key] = append(groups[key], row{task.Path, output.SHA256})
			}
		}
	}
	result := []outputGroup{}
	for key, values := range groups {
		if len(values) < 2 {
			continue
		}
		parts := strings.SplitN(key, "\x00", 2)
		group := outputGroup{Path: parts[1], SameBytes: true}
		for _, value := range values {
			group.Producers = append(group.Producers, value.producer)
			group.SameBytes = group.SameBytes && value.sha == values[0].sha
		}
		sort.Strings(group.Producers)
		result = append(result, group)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}

func dependencyOrdered(tasks []changeaware.TaskEvidence, left, right string) bool {
	graph := map[string][]string{}
	for _, task := range tasks {
		graph[task.Path] = task.DependsOn
	}
	return reachable(graph, left, right) || reachable(graph, right, left)
}

func reachable(graph map[string][]string, start, target string) bool {
	seen := map[string]bool{}
	pending := []string{start}
	for len(pending) > 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if current == target {
			return true
		}
		if seen[current] {
			continue
		}
		seen[current] = true
		pending = append(pending, graph[current]...)
	}
	return false
}

func hasRecoverablePartialClosure(changed []string, capture requestaligned.Capture) bool {
	tasks := map[string]changeaware.TaskEvidence{}
	reverse := map[string][]string{}
	seeds := map[string]bool{}
	for _, task := range capture.Tasks {
		tasks[task.Path] = task
		if inputIntersections(changed, []changeaware.TaskEvidence{task}) > 0 {
			seeds[task.Path] = true
		}
		for _, dependency := range task.DependsOn {
			reverse[dependency] = append(reverse[dependency], task.Path)
		}
	}
	required := walk(capture.RequestedTasks, func(path string) []string { return tasks[path].DependsOn })
	affectedRoots := sortedKeys(seeds)
	affected := walk(affectedRoots, func(path string) []string { return reverse[path] })
	candidate, omitted, boundOutputStates := 0, 0, 0
	for path := range required {
		if affected[path] {
			candidate++
		} else {
			omitted++
			for _, output := range tasks[path].Outputs {
				if (output.Exists && output.SHA256 == "") || (!output.Exists && output.SHA256 != "") {
					return false
				}
				boundOutputStates++
			}
		}
	}
	return candidate > 0 && omitted > 0 && boundOutputStates > 0
}

func walk(roots []string, next func(string) []string) map[string]bool {
	result := map[string]bool{}
	pending := append([]string(nil), roots...)
	for len(pending) > 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if result[current] {
			continue
		}
		result[current] = true
		pending = append(pending, next(current)...)
	}
	return result
}

func absentOmittedOutputs(report requestaligned.Classification, capture requestaligned.Capture) []string {
	omitted := map[string]bool{}
	for _, task := range report.OmittedTasks {
		omitted[task] = true
	}
	paths := []string{}
	for _, task := range capture.Tasks {
		if !omitted[task.Path] {
			continue
		}
		for _, output := range task.Outputs {
			if !output.Exists {
				paths = append(paths, output.Path)
			}
		}
	}
	sort.Strings(paths)
	return paths
}

func identityPatternFor(base, target requestaligned.Capture) string {
	arguments := !reflect.DeepEqual(base.GradleArguments, target.GradleArguments)
	requested := !reflect.DeepEqual(base.RequestedTasks, target.RequestedTasks)
	gradle := base.GradleVersion != target.GradleVersion
	runtime := !reflect.DeepEqual(base.JavaRuntime, target.JavaRuntime)
	environment := base.EnvironmentBindingSHA256 != target.EnvironmentBindingSHA256
	wrapper := !reflect.DeepEqual(base.WrapperFiles, target.WrapperFiles)
	buildLogic := !reflect.DeepEqual(base.BuildLogicFiles, target.BuildLogicFiles)
	tasks := !reflect.DeepEqual(taskGraph(base.Tasks), taskGraph(target.Tasks))
	switch {
	case !arguments && !requested && !gradle && !runtime && !environment && !wrapper && buildLogic && !tasks:
		return "BUILD_LOGIC_ONLY_STABLE_REQUEST_GRAPH"
	case !arguments && !requested && !gradle && !runtime && !environment && !wrapper && buildLogic && tasks:
		return "BUILD_LOGIC_AND_REQUEST_GRAPH_CHANGED"
	case !arguments && !requested && gradle && !runtime && !environment && wrapper && !buildLogic && !tasks:
		return "WRAPPER_AND_GRADLE_VERSION_CHANGED_STABLE_REQUEST_GRAPH"
	case !arguments && !requested && gradle && !runtime && !environment && wrapper && !buildLogic && tasks:
		return "WRAPPER_GRADLE_VERSION_AND_REQUEST_GRAPH_CHANGED"
	case !arguments && !requested && gradle && !runtime && !environment && wrapper && buildLogic && !tasks:
		return "WRAPPER_GRADLE_VERSION_AND_BUILD_LOGIC_CHANGED_STABLE_REQUEST_GRAPH"
	default:
		return "OTHER_IDENTITY_CHANGE"
	}
}

func taskGraph(tasks []changeaware.TaskEvidence) map[string][]string {
	result := map[string][]string{}
	for _, task := range tasks {
		dependencies := append([]string(nil), task.DependsOn...)
		sort.Strings(dependencies)
		result[task.Path] = dependencies
	}
	return result
}

func inputIntersections(changed []string, tasks []changeaware.TaskEvidence) int {
	count := 0
	for _, path := range changed {
		matched := false
		for _, task := range tasks {
			for _, input := range task.Inputs {
				if input.Path == path || (input.Kind == "DIRECTORY" && strings.HasPrefix(path, strings.TrimSuffix(input.Path, "/")+"/")) {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
		if matched {
			count++
		}
	}
	return count
}

func readCaptures(root string, row ledgerRow) (requestaligned.Capture, requestaligned.Capture, error) {
	var base, target requestaligned.Capture
	if err := readStrict(filepath.Join(root, row.BaseCapture), &base); err != nil {
		return base, target, err
	}
	if err := readStrict(filepath.Join(root, row.TargetCapture), &target); err != nil {
		return base, target, err
	}
	return base, target, nil
}

func readLedger(path string) ([]ledgerRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	rows := []ledgerRow{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		var row ledgerRow
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, scanner.Err()
}

func readStrict(path string, target any) error {
	return readJSON(path, target, true)
}

func readJSON(path string, target any, strict bool) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	if strict {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("JSON has trailing content")
	}
	return nil
}

func sortedPatterns(values map[string]int) []identityPattern {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]identityPattern, 0, len(keys))
	for _, key := range keys {
		result = append(result, identityPattern{Pattern: key, Transitions: values[key]})
	}
	return result
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func mustSHA(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		fail(err)
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

func writeJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "request-aligned cause audit failed: %v\n", err)
	os.Exit(1)
}
