// Command strict-diagnostic-report-v2 selects the one logical root Gradle
// Configuration Cache report referenced by a child log. Repeated identical
// references are one identity; distinct references remain ambiguous.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/tonyredondo/buildopt/internal/strictdiagnostic"
)

func main() {
	logPath := flag.String("log", "", "captured Gradle child log")
	expectedRoot := flag.String("expected-root", "", "expected root-build Configuration Cache report directory")
	flag.Parse()
	if *logPath == "" || *expectedRoot == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: strict-diagnostic-report-v2 --log FILE --expected-root DIRECTORY")
		os.Exit(64)
	}
	log, err := os.ReadFile(*logPath)
	if err != nil {
		fatal(err)
	}
	selection := strictdiagnostic.SelectRootReportV2(log, *expectedRoot)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(selection); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
