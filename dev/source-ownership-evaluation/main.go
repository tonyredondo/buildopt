// Command source-ownership-evaluation applies BuildOpt's generic ownership
// resolver to stored Gradle observations and hashes the required native output
// boundary. It produces POC evidence; it cannot authorize a profile.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
	"github.com/tonyredondo/buildopt/internal/outputequivalence"
)

const schemaVersion = "buildopt.evidence/source-ownership-evaluation/v1"

type evaluation struct {
	SchemaVersion        string   `json:"schemaVersion"`
	Status               string   `json:"status"`
	Reason               string   `json:"reason"`
	Owners               []string `json:"owners"`
	IgnoredPaths         []string `json:"ignoredPaths"`
	ConsumedUnownedPaths []string `json:"consumedUnownedPaths"`
	UnattributedPaths    []string `json:"unattributedPaths"`
	ProjectCount         int      `json:"projectCount"`
	TaskCount            int      `json:"taskCount"`
	RequiredOutputCount  int      `json:"requiredOutputCount"`
	RequiredOutputSHA256 string   `json:"requiredOutputSha256"`
	ProductionAuthorized bool     `json:"productionAuthorized"`
	TestOptimization     string   `json:"testOptimization"`
}

func main() {
	var snapshotPath, inputsPath, changesPath, patternsPath, repositoryRoot string
	flag.StringVar(&snapshotPath, "snapshot", "", "validated Gradle discovery snapshot")
	flag.StringVar(&inputsPath, "inputs", "", "validated workflow input observation")
	flag.StringVar(&changesPath, "changes", "", "JSON array of changed repository paths")
	flag.StringVar(&patternsPath, "patterns", "", "JSON array of required output patterns")
	flag.StringVar(&repositoryRoot, "repository", "", "repository root containing native outputs")
	flag.Parse()
	if flag.NArg() != 0 || snapshotPath == "" || inputsPath == "" || changesPath == "" || patternsPath == "" || repositoryRoot == "" {
		fmt.Fprintln(os.Stderr, "usage: source-ownership-evaluation --snapshot FILE --inputs FILE --changes FILE --patterns FILE --repository ROOT")
		os.Exit(64)
	}

	changedPaths := readStrings(changesPath)
	patterns := readStrings(patternsPath)
	snapshotRaw := read(snapshotPath)
	var snapshotIdentity struct {
		Entrypoints []buildimpact.DiscoveredEntrypoint `json:"entrypoints"`
	}
	decode(snapshotRaw, &snapshotIdentity)
	entrypoints := make([]string, 0, len(snapshotIdentity.Entrypoints))
	for _, entrypoint := range snapshotIdentity.Entrypoints {
		entrypoints = append(entrypoints, entrypoint.Name)
	}
	snapshot, err := buildimpact.ParseObservedDiscoverySnapshot(snapshotRaw, entrypoints)
	check(err)
	observation, err := buildimpact.ParseWorkflowInputRelevance(read(inputsPath), changedPaths)
	check(err)
	ownership, ownershipErr := buildimpact.ResolveWorkflowProjectOwnership(snapshot, observation, changedPaths)

	digest, count, err := outputequivalence.HashOutputs(repositoryRoot, patterns, nil)
	check(err)
	result := evaluation{
		SchemaVersion: schemaVersion,
		Owners:        ownership.Owners, IgnoredPaths: ownership.IgnoredPaths,
		ConsumedUnownedPaths: ownership.ConsumedUnownedPaths,
		UnattributedPaths:    ownership.UnattributedPaths,
		ProjectCount:         len(snapshot.Projects), TaskCount: len(snapshot.Tasks),
		RequiredOutputCount: count, RequiredOutputSHA256: digest,
		ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
	}
	switch {
	case ownershipErr == nil:
		result.Status = "OWNERSHIP_PROVEN"
		result.Reason = "COMPLETE_WORKFLOW_OWNERSHIP"
	case errors.Is(ownershipErr, buildimpact.ErrConfigurationInputOwnershipUnproven):
		result.Status = "NATIVE_RETAINED"
		result.Reason = "CONFIGURATION_INPUT_OWNERSHIP_UNPROVEN"
	default:
		result.Status = "NATIVE_RETAINED"
		result.Reason = "SOURCE_OWNERSHIP_AMBIGUOUS"
	}
	check(json.NewEncoder(os.Stdout).Encode(result))
}

func read(path string) []byte {
	raw, err := os.ReadFile(path)
	check(err)
	return raw
}

func readStrings(path string) []string {
	var values []string
	decode(read(path), &values)
	if len(values) == 0 {
		check(errors.New("JSON string array is empty"))
	}
	return values
}

func decode(raw []byte, destination any) {
	check(json.Unmarshal(raw, destination))
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
