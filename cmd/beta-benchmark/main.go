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

const usage = "usage: beta-benchmark smoke --manifest PATH --state-dir ABSOLUTE_PATH --output-dir ABSOLUTE_PATH\n       beta-benchmark disk-faults --manifest PATH --state-dir ABSOLUTE_PATH --output-dir ABSOLUTE_PATH\n       beta-benchmark validate --manifest PATH --result PATH\n       beta-benchmark validate-disk-faults --manifest PATH --result PATH\n"

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
	case "smoke", "disk-faults":
		flags := flag.NewFlagSet("beta-benchmark "+args[0], flag.ContinueOnError)
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
		var (
			result string
			err    error
		)
		if args[0] == "smoke" {
			result, err = betabenchmark.RunSmoke(
				ctx,
				*manifest,
				*stateDirectory,
				*outputDirectory,
			)
		} else {
			result, err = betabenchmark.RunDiskFaults(
				ctx,
				*manifest,
				*stateDirectory,
				*outputDirectory,
			)
		}
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "beta-benchmark: %v\n", err)
			return 1
		}
		_, _ = fmt.Fprintf(
			stdout,
			"benchmark %s result: %s\n",
			args[0],
			result,
		)
		return 0
	case "validate", "validate-disk-faults":
		flags := flag.NewFlagSet(
			"beta-benchmark "+args[0],
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
		var err error
		if args[0] == "validate" {
			err = betabenchmark.ValidateResult(*manifest, *result)
		} else {
			err = betabenchmark.ValidateDiskFaultResult(*manifest, *result)
		}
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "beta-benchmark: %v\n", err)
			return 1
		}
		_, _ = fmt.Fprintf(stdout, "benchmark %s result valid\n", args[0])
		return 0
	default:
		_, _ = io.WriteString(stderr, usage)
		return 64
	}
}
