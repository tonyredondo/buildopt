// Command request-aligned-producer validates one ordinary Gradle request
// capture and emits a portable, non-authorizing observation.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/tonyredondo/buildopt/internal/requestaligned"
)

func main() {
	flags := flag.NewFlagSet("request-aligned-producer", flag.ExitOnError)
	input := flags.String("input", "", "request-aligned Gradle capture")
	output := flags.String("output", "", "typed request observation")
	validateStates := flags.String("validate-output-states", "", "expected output-state array to revalidate")
	_ = flags.Parse(os.Args[1:])
	produceMode := *output != "" && *validateStates == ""
	validateMode := *output == "" && *validateStates != ""
	if flags.NArg() != 0 || *input == "" || (!produceMode && !validateMode) {
		fmt.Fprintln(os.Stderr, "usage: request-aligned-producer --input PATH (--output PATH | --validate-output-states PATH)")
		os.Exit(64)
	}
	var capture requestaligned.Capture
	if err := readStrict(*input, &capture); err != nil {
		fail(err)
	}
	if validateMode {
		var expected []requestaligned.OutputState
		if err := readStrict(*validateStates, &expected); err != nil {
			fail(err)
		}
		if err := requestaligned.ValidateOutputStates(expected, capture); err != nil {
			fail(err)
		}
		return
	}
	observation, err := requestaligned.Produce(capture)
	if err != nil {
		fail(err)
	}
	if err := writeJSON(*output, observation); err != nil {
		fail(err)
	}
}

func readStrict(path string, target any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("JSON has trailing content")
	}
	return nil
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
	fmt.Fprintf(os.Stderr, "request-aligned producer failed: %v\n", err)
	os.Exit(1)
}
