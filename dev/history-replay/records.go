//go:build linux && amd64

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type StatePin struct {
	Schema           string  `json:"schema"`
	Replication      int     `json:"replication"`
	Arm              string  `json:"arm"`
	Ordinal          int     `json:"ordinal"`
	Revision         string  `json:"revision"`
	Branch           string  `json:"branch"`
	CommonGit        string  `json:"commonGit"`
	Root             string  `json:"root"`
	Previous         string  `json:"previous"`
	Entries          []Entry `json:"entries"`
	SemanticSHA256   string  `json:"semanticSHA256"`
	BaselineApplied  bool    `json:"baselineApplied"`
	CandidateApplied bool    `json:"candidateApplied"`
}

func semanticEntries(entries []Entry) []Entry {
	out := []Entry{}
	for _, e := range entries {
		// Only native daemon diagnostics and registry bookkeeping are volatile
		// between requests. They are retained separately as invocation evidence.
		// No task cache, project history or candidate state is exempted.
		parts := strings.Split(e.Path, "/")
		daemon := false
		for _, p := range parts {
			if p == "daemon" || p == "test-kit-daemon" {
				daemon = true
			}
		}
		base := filepath.Base(e.Path)
		if daemon && ((strings.HasPrefix(base, "daemon-") && strings.HasSuffix(base, ".out.log")) || base == "registry.bin" || base == "registry.bin.lock") {
			continue
		}
		out = append(out, e)
	}
	return out
}

func captureState(m Manifest, rep int, arm string, ordinal int, previous string, baseline, candidate bool) (StatePin, error) {
	root := filepath.Join(m.RunRoot, fmt.Sprintf("r%d", rep), arm)
	repo := filepath.Join(root, "repo")
	s := StatePin{Schema: recordSchema, Replication: rep, Arm: arm, Ordinal: ordinal, Revision: m.History[ordinal].Commit, CommonGit: m.CommonGit, Root: root, Previous: previous, BaselineApplied: baseline, CandidateApplied: candidate, Entries: []Entry{}}
	b, err := gitAt(repo, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return s, err
	}
	s.Branch = strings.TrimSpace(string(b))
	s.Entries, err = inventory(root)
	if err != nil {
		return s, err
	}
	for _, e := range s.Entries {
		if e.Kind == "symlink" {
			dest := e.Target
			if !filepath.IsAbs(dest) {
				dest = filepath.Join(root, filepath.Dir(e.Path), dest)
			}
			if !inside(root, dest) {
				allowed := false
				for _, b := range m.Runtime {
					if inside(b.Path, dest) {
						allowed = true
					}
				}
				if !allowed {
					return s, fmt.Errorf("state symlink escapes private arm: %s", e.Path)
				}
			}
		}
	}
	s.SemanticSHA256 = objectDigest(semanticEntries(s.Entries))
	return s, nil
}

func verifyState(m Manifest, s StatePin) error {
	if s.Schema != recordSchema || s.Ordinal < 0 || s.Ordinal > m.ExecutionEnd || s.Revision != m.History[s.Ordinal].Commit || s.Root != filepath.Join(m.RunRoot, fmt.Sprintf("r%d", s.Replication), s.Arm) || s.CommonGit != m.CommonGit {
		return errors.New("state identity/future ordinal differs")
	}
	if s.SemanticSHA256 != objectDigest(semanticEntries(s.Entries)) {
		return errors.New("forged state inventory digest")
	}
	now, err := captureState(m, s.Replication, s.Arm, s.Ordinal, s.Previous, s.BaselineApplied, s.CandidateApplied)
	if err != nil {
		return err
	}
	if now.Branch != s.Branch || now.SemanticSHA256 != s.SemanticSHA256 {
		return errors.New("state changed outside its recorded request lineage")
	}
	return nil
}

func checkInitialState(m Manifest, s StatePin) error {
	if s.Ordinal != 0 || s.Previous != "" || s.CandidateApplied {
		return errors.New("initial state is not a cold native anchor")
	}
	files, err := sourceTree(m.CommonGit, s.Revision)
	if err != nil {
		return err
	}
	source := map[string]bool{"repo/.git": true}
	for _, f := range files {
		source["repo/"+f.path] = true
	}
	if s.BaselineApplied {
		for _, f := range baselineForArm(m, s.Arm).Files {
			source["repo/"+f.Path] = true
		}
	}
	allowed := map[string]Entry{}
	for _, layer := range m.Acquisition {
		entries, err := inventory(layer.Source.Path)
		if err != nil {
			return err
		}
		for _, e := range entries {
			e.Path = filepath.Join(layer.Destination, e.Path)
			allowed[e.Path] = e
		}
	}
	for _, e := range s.Entries {
		if e.Kind == "directory" || source[e.Path] {
			continue
		}
		if expected, ok := allowed[e.Path]; !ok || !equalJSON(expected, e) {
			return fmt.Errorf("future/unapproved initial mutable state: %s", e.Path)
		}
	}
	return nil
}

func recordedApplicable(s StatePin, p Patch) bool {
	entries := map[string]Entry{}
	for _, entry := range s.Entries {
		entries[entry.Path] = entry
	}
	for path, hash := range p.Prerequisites {
		e, ok := entries["repo/"+path]
		if !ok || e.Kind != "file" || e.SHA256 != hash {
			return false
		}
	}
	for _, f := range p.Files {
		e, ok := entries["repo/"+f.Path]
		if f.BeforeSHA256 == "" {
			if ok {
				return false
			}
		} else if !ok || e.Kind != "file" || e.Mode != f.BeforeMode || e.SHA256 != f.BeforeSHA256 {
			return false
		}
	}
	return true
}

func checkStateTransition(m Manifest, previous, next StatePin) error {
	if previous.Arm != next.Arm || previous.Replication != next.Replication || next.Ordinal != previous.Ordinal+1 {
		return errors.New("state transition crosses arm/replication/ordinal")
	}
	source := map[string]bool{"repo/.git": true}
	for _, revision := range []string{previous.Revision, next.Revision} {
		files, err := sourceTree(m.CommonGit, revision)
		if err != nil {
			return err
		}
		for _, f := range files {
			source["repo/"+f.path] = true
		}
	}
	for _, p := range []Patch{baselineForArm(m, next.Arm), m.Candidate} {
		for _, f := range p.Files {
			source["repo/"+f.Path] = true
		}
	}
	retained := func(s StatePin) []Entry {
		out := []Entry{}
		for _, e := range semanticEntries(s.Entries) {
			if e.Kind != "directory" && !source[e.Path] {
				out = append(out, e)
			}
		}
		return out
	}
	if !equalJSON(retained(previous), retained(next)) {
		return errors.New("future/unowned generated state introduced during source advancement")
	}
	return nil
}

type AttemptStart struct {
	Schema               string            `json:"schema"`
	ID                   string            `json:"id"`
	Replication          int               `json:"replication"`
	Ordinal              int               `json:"ordinal"`
	Arm                  string            `json:"arm"`
	Generation           int               `json:"generation"`
	ManifestSHA256       string            `json:"manifestSHA256"`
	Before               Binding           `json:"before"`
	SourceSHA256         string            `json:"sourceSHA256"`
	Command              []string          `json:"command"`
	Environment          map[string]string `json:"environment"`
	ReservedGradleStarts int               `json:"reservedGradleStarts"`
	At                   Stamp             `json:"at"`
}

type TaskOutcome struct {
	Identity string `json:"identity"`
	Outcome  string `json:"outcome"`
	Action   bool   `json:"action"`
}

type OutputEntry struct {
	Entry            Entry   `json:"entry"`
	Producer         string  `json:"producer"`
	Transform        string  `json:"transform"`
	Raw              Binding `json:"raw"`
	ProjectionSHA256 string  `json:"projectionSHA256"`
}

type Capture struct {
	Schema       string        `json:"schema"`
	Outputs      []OutputEntry `json:"outputs"`
	Tasks        []TaskOutcome `json:"tasks"`
	Graph        []Binding     `json:"graph"`
	DaemonLogs   []Binding     `json:"daemonLogs"`
	GradleBuilds []GradleBuild `json:"gradleBuilds"`
	Error        string        `json:"error"`
}

type AttemptEnd struct {
	Schema           string   `json:"schema"`
	ID               string   `json:"id"`
	Begin            Stamp    `json:"begin"`
	End              Stamp    `json:"end"`
	DurationNS       int64    `json:"durationNS"`
	Native           Binding  `json:"native"`
	After            Binding  `json:"after"`
	Capture          Binding  `json:"capture"`
	CandidateApplied bool     `json:"candidateApplied"`
	Class            string   `json:"class"`
	Reason           string   `json:"reason"`
	Costs            []string `json:"costs"`
}

type PairRecord struct {
	Schema      string    `json:"schema"`
	Replication int       `json:"replication"`
	Ordinal     int       `json:"ordinal"`
	Generation  int       `json:"generation"`
	Attempts    []Binding `json:"attempts"`
	Slot        Slot      `json:"slot"`
	Previous    string    `json:"previous"`
	At          Stamp     `json:"at"`
}

type Checkpoint struct {
	Schema             string         `json:"schema"`
	Manifest           Binding        `json:"manifest"`
	Replication        int            `json:"replication"`
	LastOrdinal        int            `json:"lastOrdinal"`
	LastPair           Binding        `json:"lastPair"`
	States             []Binding      `json:"states"`
	Remaining          []int          `json:"remaining"`
	Attempts           int            `json:"attempts"`
	GradleReservations int            `json:"gradleReservations"`
	RetryPairs         int            `json:"retryPairs"`
	Sessions           []WorkerConfig `json:"sessions"`
	At                 Stamp          `json:"at"`
}

type RunResult struct {
	Schema                    string      `json:"schema"`
	ManifestSHA256            string      `json:"manifestSHA256"`
	Replications              []Economics `json:"replications"`
	Slots                     [][]Slot    `json:"slots"`
	WorkflowStarts            int         `json:"workflowStarts"`
	GradleReservations        int         `json:"gradleReservations"`
	ActualGradleStarts        int         `json:"actualGradleStarts"`
	NestedStarts              int         `json:"nestedStarts"`
	UnknownGradleReservations int         `json:"unknownGradleReservations"`
	Decision                  string      `json:"decision"`
	Reasons                   []string    `json:"reasons"`
}

func bind(path string) (Binding, error) { h, e := bindingDigest(path); return Binding{path, h}, e }
func saveBound(path string, v any) (Binding, error) {
	if err := writeExclusive(path, jsonBytes(v), 0600); err != nil {
		return Binding{}, err
	}
	return bind(path)
}

type phaseStart struct {
	Schema string `json:"schema"`
	Cost   Cost   `json:"cost"`
	At     Stamp  `json:"at"`
}

func beginCost(root string, c Cost) (phaseStart, error) {
	s := phaseStart{recordSchema, c, stamp()}
	s.Cost.StartNS = s.At.NS
	if !namePattern.MatchString(c.ID) {
		return s, errors.New("invalid phase ID")
	}
	if err := writeExclusive(filepath.Join(root, "costs", c.ID+".start.json"), jsonBytes(s), 0600); err != nil {
		return s, err
	}
	// Reserve durably before work, then start the actual interval. Persist its
	// boundaries after completion so research fsync latency is not customer work.
	s.At = stamp()
	s.Cost.StartNS = s.At.NS
	return s, nil
}

func finishCost(root string, s phaseStart) (Cost, error) {
	return finishCostAt(root, s, stamp())
}

func finishCostAt(root string, s phaseStart, end Stamp) (Cost, error) {
	c := s.Cost
	c.EndNS = end.NS
	c.DurationNS = c.EndNS - c.StartNS
	if err := writeExclusive(filepath.Join(root, "costs", c.ID+".work-start.json"), jsonBytes(s.At), 0600); err != nil {
		return c, err
	}
	if err := writeExclusive(filepath.Join(root, "costs", c.ID+".end.json"), jsonBytes(end), 0600); err != nil {
		return c, err
	}
	return c, writeExclusive(filepath.Join(root, "costs", c.ID+".json"), jsonBytes(c), 0600)
}

func loadCosts(root string) ([]Cost, error) {
	files, err := os.ReadDir(filepath.Join(root, "costs"))
	if err != nil {
		return nil, err
	}
	costs := []Cost{}
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".json") || strings.HasSuffix(f.Name(), ".start.json") || strings.HasSuffix(f.Name(), ".end.json") || strings.HasSuffix(f.Name(), ".work-start.json") {
			continue
		}
		var c Cost
		if err = readJSON(filepath.Join(root, "costs", f.Name()), &c); err != nil {
			return nil, err
		}
		var start phaseStart
		var end Stamp
		var workStart Stamp
		if err = readJSON(filepath.Join(root, "costs", c.ID+".start.json"), &start); err != nil {
			return nil, err
		}
		if err = readJSON(filepath.Join(root, "costs", c.ID+".end.json"), &end); err != nil {
			return nil, err
		}
		if err = readJSON(filepath.Join(root, "costs", c.ID+".work-start.json"), &workStart); err != nil {
			return nil, err
		}
		expected := start.Cost
		expected.StartNS = workStart.NS
		expected.EndNS = end.NS
		expected.DurationNS = end.NS - workStart.NS
		if start.Schema != recordSchema || start.At.Boot != end.Boot || workStart.Boot != end.Boot || workStart.NS < start.At.NS || end.NS < workStart.NS || objectDigest(c) != objectDigest(expected) {
			return nil, errors.New("cost differs from independent boundary records")
		}
		if err = checkCostPolicy(c); err != nil {
			return nil, err
		}
		costs = append(costs, c)
	}
	// An unfinished phase is evidence, never an invisible free preparation.
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".start.json") {
			name := strings.TrimSuffix(f.Name(), ".start.json") + ".json"
			if _, err = os.Stat(filepath.Join(root, "costs", name)); err != nil {
				return nil, fmt.Errorf("unfinished/unaccounted cost phase %s", f.Name())
			}
		}
	}
	return costs, nil
}

func checkCostPolicy(c Cost) error {
	if c.ID == "" || c.Replication < 1 || c.Ordinal < 0 {
		return errors.New("invalid cost identity")
	}
	switch c.Purpose {
	case "complete-request":
		if c.Class != "customer-machine" || !c.InsideEnvelope || c.Attempt == "" || (c.Arm != "N" && c.Arm != "I") || c.ID != c.Attempt+"-request" || c.WorkflowStarts != 1 {
			return errors.New("request cost classification changed")
		}
	case "candidate-inverse":
		if c.Class != "customer-machine" || c.InsideEnvelope || c.Arm != "I" || c.Ordinal == 0 || c.ID != slotKey(c.Replication, c.Ordinal)+"-I-inverse" {
			return errors.New("hidden candidate maintenance cost")
		}
	case "observer-start", "observer-close", "owner-output-comparison", "native-state-preparation", "neutral-source-advancement", "source-input-state-proof", "owned-process-supervision", "complete-output-state-and-invocation-capture", "equal-state-recovery-checkpoint", "projector-qualification":
		if c.Class != "research" || c.InsideEnvelope || c.WorkflowStarts != 0 {
			return errors.New("research phase classification changed")
		}
	default:
		return errors.New("undeclared cost purpose")
	}
	return nil
}
