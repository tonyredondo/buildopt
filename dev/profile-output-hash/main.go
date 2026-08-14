// Command profile-output-hash recomputes the owner-reviewed semantic digest
// of a qualified profile's required outputs for POC evidence capture.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
	"github.com/tonyredondo/buildopt/internal/outputequivalence"
)

type result struct {
	SHA256 string `json:"sha256"`
	Count  int    `json:"count"`
}

func main() {
	if len(os.Args) != 6 {
		fmt.Fprintln(os.Stderr, "usage: profile-output-hash ROOT MANIFEST REPOSITORY_ID PIPELINE_CLASS EQUIVALENCE")
		os.Exit(64)
	}
	root, manifestPath, repositoryID, pipelineClass, equivalencePath :=
		os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5]
	manifestRaw, err := os.ReadFile(manifestPath)
	if err != nil {
		fail(err)
	}
	manifest, err := buildimpact.ParseManifest(manifestRaw, repositoryID, pipelineClass)
	if err != nil {
		fail(err)
	}
	var contract *outputequivalence.Contract
	if equivalencePath != "-" {
		raw, readErr := os.ReadFile(equivalencePath)
		if readErr != nil {
			fail(readErr)
		}
		parsed, parseErr := outputequivalence.Parse(raw)
		if parseErr != nil {
			fail(parseErr)
		}
		contract = &parsed
	}
	patterns := make([]string, 0, len(manifest.Manifest.RequiredArtifacts))
	for _, artifact := range manifest.Manifest.RequiredArtifacts {
		patterns = append(patterns, artifact.Path)
	}
	digest, count, err := outputequivalence.HashOutputs(root, patterns, contract)
	if err != nil {
		fail(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(result{SHA256: digest, Count: count}); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "profile output hash: %v\n", err)
	os.Exit(1)
}
