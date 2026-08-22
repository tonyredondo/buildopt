// Command native-volatility-evaluation derives and optionally verifies a
// producer-atomic output transport plan from two independent native builds.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/tonyredondo/buildopt/internal/nativevolatility"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "native-volatility-evaluation: %v\n", err)
		os.Exit(1)
	}
}

func run(arguments []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("native-volatility-evaluation", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	firstPath := flags.String("first", "", "first native observation")
	secondPath := flags.String("second", "", "second native observation")
	reusedPath := flags.String("reused", "", "candidate transported-output observation")
	rebuiltPath := flags.String("rebuilt", "", "candidate locally-rebuilt observation")
	observeRoot := flags.String("observe-root", "", "native workspace to observe")
	manifestPath := flags.String("materialization-manifest", "", "producer-bound materialization manifest")
	binding := flags.String("binding", "", "observation environment binding SHA-256")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		return errors.New("usage: native-volatility-evaluation --first FILE --second FILE [--reused FILE --rebuilt FILE] | --observe-root DIR --materialization-manifest FILE --binding SHA256")
	}
	if *observeRoot != "" || *manifestPath != "" || *binding != "" {
		if *observeRoot == "" || *manifestPath == "" || *binding == "" ||
			*firstPath != "" || *secondPath != "" || *reusedPath != "" || *rebuiltPath != "" {
			return errors.New("observe mode requires root, materialization manifest and binding only")
		}
		observation, err := observeMaterialization(*observeRoot, *manifestPath, *binding)
		if err != nil {
			return err
		}
		return writeJSON(stdout, observation)
	}
	if *firstPath == "" || *secondPath == "" {
		return errors.New("usage: native-volatility-evaluation --first FILE --second FILE [--reused FILE --rebuilt FILE]")
	}
	if (*reusedPath == "") != (*rebuiltPath == "") {
		return errors.New("both --reused and --rebuilt are required for candidate verification")
	}
	first, err := readObservation(*firstPath)
	if err != nil {
		return fmt.Errorf("read first observation: %w", err)
	}
	second, err := readObservation(*secondPath)
	if err != nil {
		return fmt.Errorf("read second observation: %w", err)
	}
	result := nativevolatility.Analyze(first, second)
	if *reusedPath != "" {
		reused, readErr := readObservation(*reusedPath)
		if readErr != nil {
			return fmt.Errorf("read reused observation: %w", readErr)
		}
		rebuilt, readErr := readObservation(*rebuiltPath)
		if readErr != nil {
			return fmt.Errorf("read rebuilt observation: %w", readErr)
		}
		if err := nativevolatility.VerifyCandidate(result, reused, rebuilt); err != nil {
			return err
		}
	}
	return writeJSON(stdout, result)
}

func observeMaterialization(root, manifestPath, binding string) (nativevolatility.Observation, error) {
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return nativevolatility.Observation{}, fmt.Errorf("read materialization manifest: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var manifest struct {
		SchemaVersion    string   `json:"schemaVersion"`
		RepositoryID     string   `json:"repositoryId"`
		TargetRevision   string   `json:"targetRevision"`
		RequiredOutputs  []string `json:"requiredOutputs"`
		CandidateOutputs []string `json:"candidateOutputs"`
		PackFile         string   `json:"packFile"`
		PackSHA256       string   `json:"packSha256"`
		PackSize         int64    `json:"packSize"`
		Entries          []struct {
			Path          string   `json:"path"`
			SHA256        string   `json:"sha256"`
			Size          int64    `json:"size"`
			Mode          uint32   `json:"mode"`
			Offset        int64    `json:"offset"`
			ProducerTasks []string `json:"producerTasks"`
		} `json:"entries"`
	}
	if err := decoder.Decode(&manifest); err != nil {
		return nativevolatility.Observation{}, fmt.Errorf("decode materialization manifest: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nativevolatility.Observation{}, errors.New("materialization manifest contains trailing JSON")
	}
	if manifest.SchemaVersion != "buildopt.poc/verified-output-materialization/v2" || len(manifest.Entries) == 0 {
		return nativevolatility.Observation{}, errors.New("producer-bound materialization manifest is invalid")
	}
	inventory := make([]nativevolatility.Entry, 0, len(manifest.Entries))
	for _, entry := range manifest.Entries {
		if len(entry.ProducerTasks) == 0 {
			return nativevolatility.Observation{}, errors.New("materialization entry has no producer attribution")
		}
		inventory = append(inventory, nativevolatility.Entry{
			Path: entry.Path, ProducerTasks: append([]string(nil), entry.ProducerTasks...),
		})
	}
	return nativevolatility.Observe(root, binding, inventory)
}

func writeJSON(stdout io.Writer, value any) error {
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func readObservation(path string) (nativevolatility.Observation, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nativevolatility.Observation{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var observation nativevolatility.Observation
	if err := decoder.Decode(&observation); err != nil {
		return nativevolatility.Observation{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nativevolatility.Observation{}, errors.New("observation contains trailing JSON")
	}
	return observation, nil
}
