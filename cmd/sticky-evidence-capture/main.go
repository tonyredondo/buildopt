// Command sticky-evidence-capture converts one generic capture document into
// typed detector outcomes. It neither executes Gradle nor authorizes actions.
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

	"github.com/tonyredondo/buildopt/internal/durablecatalog"
)

func main() {
	flags := flag.NewFlagSet("sticky-evidence-capture", flag.ExitOnError)
	inputPath := flags.String("input", "", "generic capture input")
	outputPath := flags.String("output", "", "typed producer output")
	_ = flags.Parse(os.Args[1:])
	if flags.NArg() != 0 || *inputPath == "" || *outputPath == "" {
		fmt.Fprintln(os.Stderr, "usage: sticky-evidence-capture --input PATH --output PATH")
		os.Exit(64)
	}
	var input durablecatalog.FreshEvidenceInput
	if err := readStrict(*inputPath, &input); err != nil {
		fail(err)
	}
	report, err := durablecatalog.ProduceFreshEvidence(input)
	if err != nil {
		fail(err)
	}
	if err := writeJSON(*outputPath, report); err != nil {
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
	fmt.Fprintf(os.Stderr, "sticky evidence capture failed: %v\n", err)
	os.Exit(1)
}
