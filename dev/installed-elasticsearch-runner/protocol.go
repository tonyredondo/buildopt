package main

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
)

const protocolSchema = "buildopt.eic/protocol/v1"

type row struct {
	ID              string   `json:"id"`
	Block           string   `json:"block"`
	Arm             string   `json:"arm"`
	Revision        string   `json:"revision"`
	Workload        string   `json:"workload"`
	Mode            string   `json:"mode"`
	State           string   `json:"state"`
	Pair            int      `json:"pair"`
	Arguments       []string `json:"arguments"`
	NestedStarts    int      `json:"nestedStarts"`
	ExpectedFailure bool     `json:"expectedFailure"`
}

type protocol struct {
	Schema            string        `json:"schemaVersion"`
	Subject           subjectInputs `json:"subject"`
	Revisions         []string      `json:"revisions"`
	Rows              []row         `json:"rows"`
	RequiredStarts    int           `json:"requiredStarts"`
	ReserveStarts     int           `json:"reserveStarts"`
	MaximumStarts     int           `json:"maximumStarts"`
	StatusCalls       int           `json:"statusCalls"`
	QuiescenceSeconds int           `json:"quiescenceSeconds"`
	ClosureSeconds    int           `json:"closureSeconds"`
	CeilingSeconds    int           `json:"ceilingSeconds"`
}

// makeProtocol compiles the accepted allocation, without reading outcomes or
// admitting execution. Mutation commands name the later isolated fixture roots;
// their Gradle implementation is a separate public-runtime prerequisite.
func makeProtocol() protocol {
	p := protocol{Schema: protocolSchema, Subject: sourceInputs(), Revisions: []string{
		"16bd5bc5355ac7c6ad736f8a6f93281b24a05ab7",
		"8dd9fdf41d70d6c533743a0896627fa209431978",
		"53baef580481f72a976743b0cdf2e8b72afe5491",
		"c022e4ad3d7682ea42f9ad629e710e098b70b1b2",
		"02b9f37942b5dab253e4fadb5b8e598340e27be5",
		"cb032821c0ca309df5d52a3f2c8e04c50ff58e9d",
	}, ReserveStarts: 2, StatusCalls: 22, QuiescenceSeconds: 120, ClosureSeconds: 600, CeilingSeconds: 7200}
	indices := map[string]int{}
	add := func(block, arm, workload, mode, state string, pair int) {
		indices[block]++
		r := row{ID: fmt.Sprintf("%s%03d", block, indices[block]), Block: block, Arm: arm, Revision: p.Revisions[0], Workload: workload, Mode: mode, State: state, Pair: pair}
		r.Arguments = arguments(r)
		if workload == "functional" {
			r.NestedStarts = 6
		}
		p.RequiredStarts += 1 + r.NestedStarts
		p.Rows = append(p.Rows, r)
	}
	for _, arm := range []string{"N0", "N1", "W1"} {
		add("P", arm, "workflow", "native", "unpatched-preparation", 0)
	}
	for i := 1; i <= 2; i++ {
		add("D", "N0", "workflow", "native", "full-graph-diagnostic", i)
	}
	correctness := []struct{ arm, state string }{
		{"N0", "native-success"}, {"N0", "native-repeat"}, {"N1", "populate"}, {"N1", "restore-same-root"},
		{"W1", "restore-cross-root"}, {"W1", "normal-repeat"}, {"N0", "benign-comment"}, {"W1", "benign-comment"},
		{"N0", "tab-violation"}, {"W1", "tab-violation"}, {"W1", "revert"}, {"N0", "revert"},
	}
	for _, c := range correctness {
		workload := "workflow"
		if c.state == "tab-violation" {
			workload = "probe"
		}
		mode := "native"
		if c.arm == "W1" {
			mode = "upload"
		}
		add("C", c.arm, workload, mode, c.state, 0)
		p.Rows[len(p.Rows)-1].ExpectedFailure = c.state == "tab-violation"
	}
	for i, fixture := range []string{"rules", "excludes", "relative-rename", "all-source-removal", "malformed-utf8", "relative-root"} {
		for _, change := range []string{"before", "after"} {
			for _, arm := range []string{"N0", "N1"} {
				add("M", arm, "mutation", "native", fixture+":"+change, i+1)
				p.Rows[len(p.Rows)-1].ExpectedFailure = fixture == "malformed-utf8" && change == "after"
			}
		}
	}
	for _, workload := range []string{"unit", "functional"} {
		for _, arm := range []string{"N0", "N1"} {
			add("T", arm, workload, "native", "owner-tests", 0)
		}
	}
	orders := [][]string{{"N0", "N1", "W1"}, {"N0", "N1", "W1"}, {"W1", "N1", "N0"}, {"N1", "W1", "N0"}, {"N0", "W1", "N1"}, {"W1", "N0", "N1"}, {"N1", "N0", "W1"}, {"N0", "N1", "W1"}, {"W1", "N1", "N0"}}
	for pair, order := range orders {
		for _, arm := range order {
			mode := "native"
			if arm == "W1" {
				mode = "upload"
			}
			state := "restore-clean-output-snapshot"
			if pair == 0 {
				state = "excluded-stabilization"
			}
			add("V", arm, "workflow", mode, state, pair)
		}
	}
	for pair := 1; pair <= 20; pair++ {
		for _, arm := range pairOrder(pair, "N1", "W1") {
			mode := "native"
			if arm == "W1" {
				mode = "enqueue"
			}
			add("L", arm, "workflow", mode, "retain-evolving-state", pair)
		}
	}
	for i, revision := range p.Revisions[1:] {
		for request := 1; request <= 2; request++ {
			for _, arm := range pairOrder(request, "N0", "W1") {
				mode := "native"
				if arm == "W1" {
					mode = "upload"
				}
				state := "revision-first"
				if request == 2 {
					state = "revision-repeat"
				}
				add("H", arm, "workflow", mode, state, i*2+request)
				p.Rows[len(p.Rows)-1].Revision = revision
			}
		}
	}
	for _, mode := range []string{"off", "enqueue", "upload", "offline"} {
		for pair := 0; pair <= 20; pair++ {
			for _, arm := range pairOrder(pair, "N1", "W1") {
				block := "O"
				if pair == 0 {
					block = "O-setup"
				}
				add(block, arm, "help", mode, "retain-evolving-state", pair)
			}
		}
	}
	p.MaximumStarts = p.RequiredStarts + p.ReserveStarts
	return p
}

func pairOrder(pair int, direct, candidate string) []string {
	if pair == 0 || pair%2 == 1 {
		return []string{direct, candidate}
	}
	return []string{candidate, direct}
}

func arguments(r row) []string {
	entry := "./gradlew"
	if r.Arm == "W1" && r.Block != "P" {
		entry = "./buildoptw"
	}
	args := []string{entry, "--no-daemon", "--console=plain", "--max-workers=8"}
	if r.Workload == "unit" || r.Workload == "functional" {
		task, class := "test", "org.elasticsearch.gradle.internal.precommit.ForbiddenPatternsTaskTests"
		if r.Workload == "functional" {
			task, class = "integTest", "org.elasticsearch.gradle.internal.precommit.ForbiddenPatternsPrecommitPluginFuncTest"
		}
		return append(args, "-p", "build-tools-internal", task, "--tests", class)
	}
	task := map[string]string{"workflow": ":server:precommit", "probe": ":server:forbiddenPatterns", "mutation": "forbiddenPatterns", "help": "help"}[r.Workload]
	return append(args, "--build-cache", task)
}

func validateProtocol(p protocol) error {
	if !reflect.DeepEqual(p, makeProtocol()) {
		return errors.New("frozen EIC protocol drift")
	}
	return nil
}

func costProfile(r row) string {
	// Warm help/workflow and clean-workspace workflow are distinct scenarios.
	return r.Block + ":" + r.Workload + ":" + r.Arm + ":" + r.Mode
}

func costProfiles(p protocol) []string {
	seen := map[string]bool{"infrastructure-reserve": true}
	for _, r := range p.Rows {
		seen[costProfile(r)] = true
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
