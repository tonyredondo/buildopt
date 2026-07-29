package main

import (
	"fmt"
	"io"
	"os"

	schemavalidator "github.com/tonyredondo/buildopt/dev/schema-validator"
)

const usage = "usage: build-session-validator <schema.json> <build-session.json>\n"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) != 2 {
		_, _ = io.WriteString(stderr, usage)
		return 64
	}
	if err := schemavalidator.ValidateBuildSessionV1(
		args[0],
		args[1],
	); err != nil {
		_, _ = fmt.Fprintf(stderr, "build-session-validator: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintln(stdout, "BUILD_SESSION v1 valid")
	return 0
}
