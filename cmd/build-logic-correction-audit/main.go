package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tonyredondo/buildopt/internal/buildlogiccorrection"
)

func main() {
	family := flag.String("family", "", "stable family label")
	revision := flag.String("revision", "", "source revision")
	sourceTree := flag.String("source-tree", "", "source Git tree object")
	sourceRoot := flag.String("source-root", "", "root containing the frozen source tree")
	analysis := flag.String("analysis", "", "critical-path analysis JSON")
	flag.Parse()
	if *family == "" || *revision == "" || *sourceTree == "" || *sourceRoot == "" || *analysis == "" {
		fmt.Fprintln(os.Stderr, "family, revision, source-tree, source-root and analysis are required")
		os.Exit(2)
	}
	report, err := buildlogiccorrection.Scan(*family, *revision, *sourceTree, *sourceRoot, *analysis)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := buildlogiccorrection.WriteJSON(os.Stdout, report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
