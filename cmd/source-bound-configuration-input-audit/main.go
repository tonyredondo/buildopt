// Command source-bound-configuration-input-audit renders SBIC v1 decisions
// from one exact source file and its independently supplied semantic facts.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/tonyredondo/buildopt/internal/configurationinputsource"
)

func main() {
	sourcePath := flag.String("source", "", "repository-owned build-logic source")
	factsPath := flag.String("facts", "", "source binding facts JSON")
	flag.Parse()
	if *sourcePath == "" || *factsPath == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: source-bound-configuration-input-audit --source FILE --facts FILE")
		os.Exit(64)
	}
	source, err := os.ReadFile(*sourcePath)
	if err != nil {
		fatal(err)
	}
	factsContent, err := os.ReadFile(*factsPath)
	if err != nil {
		fatal(err)
	}
	var facts configurationinputsource.Facts
	decoder := json.NewDecoder(bytes.NewReader(factsContent))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&facts); err != nil {
		fatal(configurationinputsource.ErrInvalidFacts)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		fatal(configurationinputsource.ErrInvalidFacts)
	}
	rows, err := configurationinputsource.Scan(source, facts)
	if err != nil {
		fatal(err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(rows); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
