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
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || *firstPath == "" || *secondPath == "" {
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
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
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
