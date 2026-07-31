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

const usage = "usage: beta-benchmark smoke --manifest PATH --state-dir ABSOLUTE_PATH --output-dir ABSOLUTE_PATH\n       beta-benchmark disk-faults --manifest PATH --state-dir ABSOLUTE_PATH --output-dir ABSOLUTE_PATH\n       beta-benchmark shared-faults --manifest PATH --state-dir ABSOLUTE_PATH --output-dir ABSOLUTE_PATH\n       beta-benchmark system-faults --manifest PATH --state-dir ABSOLUTE_PATH --output-dir ABSOLUTE_PATH --buildopt ABSOLUTE_PATH --server ABSOLUTE_PATH\n       beta-benchmark validate --manifest PATH --result PATH\n       beta-benchmark validate-disk-faults --manifest PATH --result PATH\n       beta-benchmark validate-shared-faults --manifest PATH --result PATH\n       beta-benchmark validate-system-faults --manifest PATH --result PATH\n"

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
	case "smoke", "disk-faults", "shared-faults":
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
		switch args[0] {
		case "smoke":
			result, err = betabenchmark.RunSmoke(
				ctx,
				*manifest,
				*stateDirectory,
				*outputDirectory,
			)
		case "disk-faults":
			result, err = betabenchmark.RunDiskFaults(
				ctx,
				*manifest,
				*stateDirectory,
				*outputDirectory,
			)
		case "shared-faults":
			result, err = betabenchmark.RunSharedFaults(
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
	case "system-faults":
		flags := flag.NewFlagSet("beta-benchmark system-faults", flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		manifest := flags.String("manifest", "", "benchmark manifest")
		stateDirectory := flags.String(
			"state-dir",
			"",
			"private system-fault state directory",
		)
		outputDirectory := flags.String(
			"output-dir",
			"",
			"empty private result directory",
		)
		buildoptExecutable := flags.String(
			"buildopt",
			"",
			"absolute buildopt executable",
		)
		serverExecutable := flags.String(
			"server",
			"",
			"absolute buildopt-server executable",
		)
		if err := flags.Parse(args[1:]); err != nil ||
			flags.NArg() != 0 ||
			*manifest == "" ||
			*stateDirectory == "" ||
			*outputDirectory == "" ||
			*buildoptExecutable == "" ||
			*serverExecutable == "" {
			_, _ = io.WriteString(stderr, usage)
			return 64
		}
		result, err := betabenchmark.RunSystemFaults(
			ctx,
			*manifest,
			*stateDirectory,
			*outputDirectory,
			betabenchmark.SystemFaultExecutables{
				BuildOpt: *buildoptExecutable,
				Server:   *serverExecutable,
			},
		)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "beta-benchmark: %v\n", err)
			return 1
		}
		_, _ = fmt.Fprintf(
			stdout,
			"benchmark system-faults result: %s\n",
			result,
		)
		return 0
	case "validate", "validate-disk-faults", "validate-shared-faults",
		"validate-system-faults":
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
		switch args[0] {
		case "validate":
			err = betabenchmark.ValidateResult(*manifest, *result)
		case "validate-disk-faults":
			err = betabenchmark.ValidateDiskFaultResult(*manifest, *result)
		case "validate-shared-faults":
			err = betabenchmark.ValidateSharedFaultResult(*manifest, *result)
		case "validate-system-faults":
			err = betabenchmark.ValidateSystemFaultResult(*manifest, *result)
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
