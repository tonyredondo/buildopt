// Command source-bound-configuration-input-diagnostic-audit reparses one raw
// Gradle report and binds it to independently reconstructed source rows.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/tonyredondo/buildopt/internal/configurationinput"
	"github.com/tonyredondo/buildopt/internal/configurationinputbinding"
	"github.com/tonyredondo/buildopt/internal/configurationinputsource"
)

func main() {
	reportPath := flag.String("report", "", "raw Gradle Configuration Cache HTML report")
	sourcePath := flag.String("source", "", "repository-owned build-logic source")
	factsPath := flag.String("facts", "", "source binding facts JSON")
	flag.Parse()
	if *reportPath == "" || *sourcePath == "" || *factsPath == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: source-bound-configuration-input-diagnostic-audit --report FILE --source FILE --facts FILE")
		os.Exit(64)
	}
	report := read(*reportPath)
	source := read(*sourcePath)
	var facts configurationinputsource.Facts
	decoder := json.NewDecoder(bytes.NewReader(read(*factsPath)))
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
	problems, err := configurationinput.ParseReport(report)
	if err != nil {
		fatal(err)
	}
	result := struct {
		SourceRows []configurationinputsource.Row   `json:"sourceRows"`
		Problems   []configurationinput.Problem     `json:"problems"`
		Binding    configurationinputbinding.Result `json:"binding"`
	}{rows, problems, configurationinputbinding.Bind(source, rows, problems)}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fatal(err)
	}
}

func read(path string) []byte {
	content, err := os.ReadFile(path)
	if err != nil {
		fatal(err)
	}
	return content
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
