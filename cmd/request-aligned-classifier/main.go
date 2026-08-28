// Command request-aligned-classifier validates an adjacent ordinary-request
// transition and emits one typed, non-authorizing classification.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tonyredondo/buildopt/internal/requestaligned"
)

func main() {
	flags := flag.NewFlagSet("request-aligned-classifier", flag.ExitOnError)
	input := flags.String("input", "", "adjacent request transition")
	output := flags.String("output", "", "typed request classification")
	_ = flags.Parse(os.Args[1:])
	if flags.NArg() != 0 || *input == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "usage: request-aligned-classifier --input PATH --output PATH")
		os.Exit(64)
	}
	var transition requestaligned.Transition
	if err := readStrict(*input, &transition); err != nil {
		fail(err)
	}
	report, err := requestaligned.Classify(transition)
	if err != nil {
		fail(err)
	}
	if err := writeJSON(*output, report); err != nil {
		fail(err)
	}
}
