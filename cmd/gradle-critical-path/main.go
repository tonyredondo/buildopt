// Command gradle-critical-path attributes Gradle task durations to the
// longest hard-dependency chain in each build of one diagnostic invocation.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/tonyredondo/buildopt/internal/gradlecriticalpath"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "gradle-critical-path: %v\n", err)
		os.Exit(1)
	}
}

func run(arguments []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("gradle-critical-path", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	operations := flags.String("operations", "", "Gradle operation trace log")
	graph := flags.String("graph", "", "resolved task graph JSONL")
	arm := flags.String("arm", "", "control or candidate")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 ||
		*operations == "" || *graph == "" || *arm == "" {
		return errors.New("usage: gradle-critical-path --operations FILE --graph FILE --arm control|candidate")
	}
	report, err := gradlecriticalpath.Analyze(*operations, *graph, *arm)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
