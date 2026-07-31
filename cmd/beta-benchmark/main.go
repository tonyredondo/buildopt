package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/tonyredondo/buildopt/internal/betabenchmark"
)

const usage = "usage: beta-benchmark smoke --manifest PATH --state-dir ABSOLUTE_PATH --output-dir ABSOLUTE_PATH\n       beta-benchmark validate --manifest PATH --result PATH\n"

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}

func run(
	ctx context.Context,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	if len(args) == 0 {
		_, _ = io.WriteString(stderr, usage)
		return 64
	}
	switch args[0] {
	case "smoke":
		flags := flag.NewFlagSet("beta-benchmark smoke", flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		manifest := flags.String("manifest", "", "benchmark manifest")
		stateDirectory := flags.String(
			"state-dir",
			"",
			"private Shared state directory",
		)
		outputDirectory := flags.String(
			"output-dir",
			"",
			"empty private result directory",
		)
		if err := flags.Parse(args[1:]); err != nil ||
			flags.NArg() != 0 ||
			*manifest == "" ||
			*stateDirectory == "" ||
			*outputDirectory == "" {
			_, _ = io.WriteString(stderr, usage)
			return 64
		}
		result, err := betabenchmark.RunSmoke(
			ctx,
			*manifest,
			*stateDirectory,
			*outputDirectory,
		)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "beta-benchmark: %v\n", err)
			return 1
		}
		_, _ = fmt.Fprintf(stdout, "benchmark smoke result: %s\n", result)
		return 0
	case "validate":
		flags := flag.NewFlagSet(
			"beta-benchmark validate",
			flag.ContinueOnError,
		)
		flags.SetOutput(io.Discard)
		manifest := flags.String("manifest", "", "benchmark manifest")
		result := flags.String("result", "", "benchmark result")
		if err := flags.Parse(args[1:]); err != nil ||
			flags.NArg() != 0 ||
			*manifest == "" ||
			*result == "" {
			_, _ = io.WriteString(stderr, usage)
			return 64
		}
		if err := betabenchmark.ValidateResult(*manifest, *result); err != nil {
			_, _ = fmt.Fprintf(stderr, "beta-benchmark: %v\n", err)
			return 1
		}
		_, _ = io.WriteString(stdout, "benchmark smoke result valid\n")
		return 0
	default:
		_, _ = io.WriteString(stderr, usage)
		return 64
	}
}
