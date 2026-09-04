// Command configuration-input-native-corrections-audit independently renders
// CINC v1 rows from one raw Gradle report and source-fact binding file.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/tonyredondo/buildopt/internal/configurationinput"
)

func main() {
	reportPath := flag.String("report", "", "raw Gradle Configuration Cache HTML report")
	factsPath := flag.String("facts", "", "source facts JSON")
	flag.Parse()
	if *reportPath == "" || *factsPath == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: configuration-input-native-corrections-audit --report FILE --facts FILE")
		os.Exit(64)
	}
	report, err := os.ReadFile(*reportPath)
	if err != nil {
		fatal(err)
	}
	factsContent, err := os.ReadFile(*factsPath)
	if err != nil {
		fatal(err)
	}
	problems, err := configurationinput.ParseReport(report)
	if err != nil {
		fatal(err)
	}
	facts, err := configurationinput.DecodeFacts(factsContent)
	if err != nil || len(facts) != len(problems) {
		fatal(configurationinput.ErrInvalidFacts)
	}
	rows := make([]configurationinput.Row, 0, len(problems))
	for index, problem := range problems {
		if facts[index].ProblemIndex != problem.Index {
			fatal(configurationinput.ErrInvalidFacts)
		}
		row, classifyErr := configurationinput.Classify(problem, facts[index])
		if classifyErr != nil {
			fatal(classifyErr)
		}
		rows = append(rows, row)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(rows); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
