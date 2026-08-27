// Command change-aware-producer converts one Gradle capture into a typed,
// non-authorizing producer-closure report.
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

	"github.com/tonyredondo/buildopt/internal/changeaware"
)

func main() {
	flags := flag.NewFlagSet("change-aware-producer", flag.ExitOnError)
	input := flags.String("input", "", "change-aware Gradle capture")
	output := flags.String("output", "", "typed producer report")
	_ = flags.Parse(os.Args[1:])
	if flags.NArg() != 0 || *input == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "usage: change-aware-producer --input PATH --output PATH")
		os.Exit(64)
	}
	var capture changeaware.Capture
	if err := readStrict(*input, &capture); err != nil {
		fail(err)
	}
	report, err := changeaware.Analyze(capture)
	if err != nil {
		fail(err)
	}
	if err := writeJSON(*output, report); err != nil {
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
	fmt.Fprintf(os.Stderr, "change-aware producer failed: %v\n", err)
	os.Exit(1)
}
