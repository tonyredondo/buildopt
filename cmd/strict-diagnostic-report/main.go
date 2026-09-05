// Command strict-diagnostic-report selects and validates the Configuration
// Cache report referenced by one captured Gradle child log.
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
	reportRoot := flag.String("expected-root", "", "expected root Configuration Cache report directory")
	flag.Parse()
	if *logPath == "" || *reportRoot == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: strict-diagnostic-report --log FILE --expected-root DIRECTORY")
		os.Exit(64)
	}
	log, err := os.ReadFile(*logPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read child log: %v\n", err)
		os.Exit(1)
	}
	selection := strictdiagnostic.SelectRootReport(log, *reportRoot)
	if err := json.NewEncoder(os.Stdout).Encode(selection); err != nil {
		fmt.Fprintf(os.Stderr, "encode selection: %v\n", err)
		os.Exit(1)
	}
}
