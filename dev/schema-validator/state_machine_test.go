package schemavalidator

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

type stateMachineCatalog struct {
	SchemaVersion string                 `json:"schemaVersion"`
	Machines      []stateMachine         `json:"machines"`
	Dependencies  []stateDependency      `json:"dependencies"`
	Scenarios     []stateMachineScenario `json:"scenarios"`
}

type stateMachine struct {
	ID             string            `json:"id"`
	InitialState   string            `json:"initialState"`
	States         []string          `json:"states"`
	TerminalStates []string          `json:"terminalStates"`
	Transitions    []stateTransition `json:"transitions"`
}

type stateTransition struct {
	From  string `json:"from"`
	Event string `json:"event"`
	To    string `json:"to"`
}

type stateDependency struct {
	SourceMachine string `json:"sourceMachine"`
	SourceState   string `json:"sourceState"`
	TargetMachine string `json:"targetMachine"`
	TargetEvent   string `json:"targetEvent"`
}

type stateMachineScenario struct {
	ID       string                  `json:"id"`
	Initial  map[string]stateVersion `json:"initial"`
	Steps    []stateMachineStep      `json:"steps"`
	Expected map[string]stateVersion `json:"expected"`
}

type stateVersion struct {
	State   string `json:"state"`
	Version int    `json:"version"`
}

type stateMachineStep struct {
	Machine         string `json:"machine"`
	CommandID       string `json:"commandId"`
	ExpectedVersion int    `json:"expectedVersion"`
	Event           string `json:"event"`
	Response        string `json:"response"`
	ExpectedResult  string `json:"expectedResult"`
}

type appliedStateCommand struct {
	Event           string
	ExpectedVersion int
	Resulting       stateVersion
}

func TestStateMachinesV1Definitions(t *testing.T) {
	t.Parallel()

	catalog := loadStateMachineCatalog(t)
	machines := validateStateMachineCatalog(t, catalog)
	assertStateMachineSchemaEnum(
		t,
		filepath.Join(
			findRepositoryRoot(t),
			"contracts",
			"jsonschema",
			"attempt-state.v1.schema.json",
		),
		"state",
		machines["ATTEMPT"].States,
	)
	assertStateMachineSchemaEnum(
		t,
		filepath.Join(
			findRepositoryRoot(t),
			"contracts",
			"jsonschema",
			"action-record.v1.schema.json",
		),
		"actionState",
		machines["ACTION_ROLLOUT"].States,
	)
	wantTaskStates := []string{
		"UNKNOWN",
		"OBSERVING",
		"CONTRACT_QUALIFIED",
		"QUARANTINE_VALIDATED",
		"REJECTED",
		"SUSPENDED",
	}
	if !sameStringSet(machines["TASK_QUALIFICATION"].States, wantTaskStates) {
		t.Errorf(
			"task states = %v, want %v",
			machines["TASK_QUALIFICATION"].States,
			wantTaskStates,
		)
	}
}

func TestStateMachinesV1Scenarios(t *testing.T) {
	t.Parallel()

	catalog := loadStateMachineCatalog(t)
	machines := validateStateMachineCatalog(t, catalog)
	for _, scenario := range catalog.Scenarios {
		scenario := scenario
		t.Run(scenario.ID, func(t *testing.T) {
			t.Parallel()
			executeStateMachineScenario(t, machines, catalog.Dependencies, scenario)
		})
	}
}

func loadStateMachineCatalog(t *testing.T) stateMachineCatalog {
	t.Helper()
	path := filepath.Join(
		findRepositoryRoot(t),
		"contracts",
		"test-vectors",
		"state-machines",
		"state-machines.v1.json",
	)
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var catalog stateMachineCatalog
	if err := decoder.Decode(&catalog); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatalf("%s has trailing JSON: %v", path, err)
	}
	return catalog
}

func validateStateMachineCatalog(
	t *testing.T,
	catalog stateMachineCatalog,
) map[string]stateMachine {
	t.Helper()
	if catalog.SchemaVersion != "buildopt.contracts/state-machines/v1" {
		t.Errorf("schemaVersion = %q", catalog.SchemaVersion)
	}
	if len(catalog.Machines) != 3 || len(catalog.Scenarios) < 10 {
		t.Fatalf(
			"catalog has %d machines and %d scenarios",
			len(catalog.Machines),
			len(catalog.Scenarios),
		)
	}
	machines := make(map[string]stateMachine, len(catalog.Machines))
	for _, machine := range catalog.Machines {
		if _, duplicate := machines[machine.ID]; duplicate {
			t.Errorf("duplicate machine %s", machine.ID)
		}
		states := stringSet(machine.States)
		if len(states) != len(machine.States) || len(states) == 0 {
			t.Errorf("%s states must be non-empty and unique", machine.ID)
		}
		if _, exists := states[machine.InitialState]; !exists {
			t.Errorf("%s initial state %s is absent", machine.ID, machine.InitialState)
		}
		terminals := stringSet(machine.TerminalStates)
		for terminal := range terminals {
			if _, exists := states[terminal]; !exists {
				t.Errorf("%s terminal state %s is absent", machine.ID, terminal)
			}
		}
		transitionKeys := make(map[string]struct{})
		for _, transition := range machine.Transitions {
			key := transition.From + "\x00" + transition.Event
			if _, duplicate := transitionKeys[key]; duplicate {
				t.Errorf("%s has duplicate transition %s/%s", machine.ID, transition.From, transition.Event)
			}
			transitionKeys[key] = struct{}{}
			if transition.From != "*" {
				if _, exists := states[transition.From]; !exists {
					t.Errorf("%s transition source %s is absent", machine.ID, transition.From)
				}
			}
			if _, exists := states[transition.To]; !exists {
				t.Errorf("%s transition target %s is absent", machine.ID, transition.To)
			}
		}
		machines[machine.ID] = machine
	}
	for _, dependency := range catalog.Dependencies {
		source, sourceExists := machines[dependency.SourceMachine]
		target, targetExists := machines[dependency.TargetMachine]
		if !sourceExists || !targetExists {
			t.Errorf("dependency references an unknown machine: %+v", dependency)
			continue
		}
		if _, exists := stringSet(source.States)[dependency.SourceState]; !exists {
			t.Errorf("dependency source state is absent: %+v", dependency)
		}
		if _, exists := findStateTransition(target, target.InitialState, dependency.TargetEvent); !exists {
			t.Errorf("dependency target event is absent: %+v", dependency)
		}
	}
	return machines
}

func executeStateMachineScenario(
	t *testing.T,
	machines map[string]stateMachine,
	dependencies []stateDependency,
	scenario stateMachineScenario,
) {
	t.Helper()
	current := make(map[string]stateVersion, len(scenario.Initial))
	for machineID, initial := range scenario.Initial {
		machine, exists := machines[machineID]
		if !exists {
			t.Fatalf("unknown initial machine %s", machineID)
		}
		if _, exists := stringSet(machine.States)[initial.State]; !exists ||
			initial.Version < 0 {
			t.Fatalf("invalid initial state %s %+v", machineID, initial)
		}
		current[machineID] = initial
	}
	history := make(map[string]appliedStateCommand)
	for _, step := range scenario.Steps {
		result := applyStateMachineStep(
			machines,
			dependencies,
			current,
			history,
			step,
		)
		if result != step.ExpectedResult {
			t.Fatalf(
				"%s/%s result = %s, want %s",
				step.Machine,
				step.CommandID,
				result,
				step.ExpectedResult,
			)
		}
	}
	if len(current) != len(scenario.Expected) {
		t.Fatalf("final machine count = %d, want %d: %+v", len(current), len(scenario.Expected), current)
	}
	for machineID, expected := range scenario.Expected {
		if actual := current[machineID]; actual != expected {
			t.Errorf("%s final = %+v, want %+v", machineID, actual, expected)
		}
	}
}

func applyStateMachineStep(
	machines map[string]stateMachine,
	dependencies []stateDependency,
	current map[string]stateVersion,
	history map[string]appliedStateCommand,
	step stateMachineStep,
) string {
	machine, exists := machines[step.Machine]
	if !exists {
		return "UNKNOWN_MACHINE"
	}
	state, exists := current[step.Machine]
	if !exists {
		state = stateVersion{State: machine.InitialState, Version: 0}
		current[step.Machine] = state
	}
	historyKey := step.Machine + "\x00" + step.CommandID
	if previous, replay := history[historyKey]; replay {
		if previous.Event != step.Event ||
			previous.ExpectedVersion != step.ExpectedVersion {
			return "IDEMPOTENCY_CONFLICT"
		}
		return "REPLAYED"
	}
	if state.Version != step.ExpectedVersion {
		return "STATE_PRECONDITION_FAILED"
	}
	if _, terminal := stringSet(machine.TerminalStates)[state.State]; terminal {
		return "TERMINAL_STATE"
	}
	transition, valid := findStateTransition(machine, state.State, step.Event)
	if !valid {
		if step.Event == "INCONCLUSIVE" {
			return "INCONCLUSIVE_NO_PROMOTION"
		}
		return "INVALID_TRANSITION"
	}
	next := stateVersion{State: transition.To, Version: state.Version + 1}
	current[step.Machine] = next
	history[historyKey] = appliedStateCommand{
		Event:           step.Event,
		ExpectedVersion: step.ExpectedVersion,
		Resulting:       next,
	}
	applyStateDependencies(machines, dependencies, current, step.Machine, next)
	if step.Response == "LOST" {
		return "UNKNOWN_APPLIED"
	}
	return "APPLIED"
}

func applyStateDependencies(
	machines map[string]stateMachine,
	dependencies []stateDependency,
	current map[string]stateVersion,
	sourceMachine string,
	source stateVersion,
) {
	for _, dependency := range dependencies {
		if dependency.SourceMachine != sourceMachine ||
			dependency.SourceState != source.State {
			continue
		}
		targetState, exists := current[dependency.TargetMachine]
		if !exists {
			continue
		}
		targetMachine := machines[dependency.TargetMachine]
		if _, terminal := stringSet(targetMachine.TerminalStates)[targetState.State]; terminal {
			continue
		}
		transition, valid := findStateTransition(
			targetMachine,
			targetState.State,
			dependency.TargetEvent,
		)
		if !valid || transition.To == targetState.State {
			continue
		}
		current[dependency.TargetMachine] = stateVersion{
			State:   transition.To,
			Version: targetState.Version + 1,
		}
	}
}

func findStateTransition(
	machine stateMachine,
	from string,
	event string,
) (stateTransition, bool) {
	for _, transition := range machine.Transitions {
		if transition.From == from && transition.Event == event {
			return transition, true
		}
	}
	if _, terminal := stringSet(machine.TerminalStates)[from]; terminal {
		return stateTransition{}, false
	}
	for _, transition := range machine.Transitions {
		if transition.From == "*" && transition.Event == event {
			return transition, true
		}
	}
	return stateTransition{}, false
}

func assertStateMachineSchemaEnum(
	t *testing.T,
	path string,
	definition string,
	expected []string,
) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var schema struct {
		Definitions map[string]struct {
			Enum []string `json:"enum"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(content, &schema); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	actual := schema.Definitions[definition].Enum
	if !sameStringSet(actual, expected) {
		t.Errorf("%s/%s enum = %v, want %v", path, definition, actual, expected)
	}
}

func sameStringSet(first []string, second []string) bool {
	if len(first) != len(second) {
		return false
	}
	firstCopy := append([]string(nil), first...)
	secondCopy := append([]string(nil), second...)
	sort.Strings(firstCopy)
	sort.Strings(secondCopy)
	return fmt.Sprint(firstCopy) == fmt.Sprint(secondCopy)
}
