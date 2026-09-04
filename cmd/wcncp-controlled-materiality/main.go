// Command wcncp-controlled-materiality reconstructs one controlled
// critical-path materiality row from primary Gradle diagnostics.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/tonyredondo/buildopt/internal/wcncpmateriality"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "wcncp-controlled-materiality: %v\n", err)
		os.Exit(1)
	}
}

func run(arguments []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("wcncp-controlled-materiality", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	operations := flags.String("operations", "", "Gradle operation trace")
	graph := flags.String("graph", "", "resolved task graph")
	family := flags.String("family", "", "frozen family key")
	method := flags.String("method", "", "materiality method")
	taskClass := flags.String("task-class", "", "exact affected task class")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || *operations == "" || *graph == "" || *family == "" || *method == "" {
		return errors.New("usage: wcncp-controlled-materiality --operations FILE --graph FILE --family KEY --method CONFIGURATION_CACHE_UNLOCK|CRITICAL_TASK_CLASS [--task-class CLASS]")
	}
	report, err := wcncpmateriality.Analyze(*operations, *graph, *family, *method, *taskClass)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
