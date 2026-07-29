package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"github.com/tonyredondo/buildopt/internal/neutralenvelope"
)

const usage = `usage:
  neutral-envelope observe --arm ARM --pair N --order N --command-class NAME --deliverable PATH --output PATH -- <command> [args...]
  neutral-envelope report --observations DIR --execution-environment NAME --runner-spec PATH --metrics-catalog PATH --envelope PATH --launcher PATH --server PATH --plugin PATH --output PATH
  neutral-envelope validate --report PATH
`

var now = time.Now

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		_, _ = fmt.Fprint(os.Stderr, usage)
		return 64
	}
	switch args[0] {
	case "observe":
		return runObserve(args[1:])
	case "report":
		return runReport(args[1:])
	case "validate":
		return runValidate(args[1:])
	default:
		_, _ = fmt.Fprint(os.Stderr, usage)
		return 64
	}
}

func runObserve(args []string) int {
	flags := flag.NewFlagSet("observe", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	arm := flags.String("arm", "", "NATIVE or WRAPPER")
	pairIndex := flags.Int("pair", 0, "one-based pair index")
	order := flags.Int("order", 0, "one-based order within the pair")
	commandClass := flags.String("command-class", "", "bounded command identity")
	deliverable := flags.String("deliverable", "", "required deliverable path")
	output := flags.String("output", "", "observation output path")
	if err := flags.Parse(args); err != nil {
		return 64
	}
	commandArgs := flags.Args()
	if *arm == "" ||
		*pairIndex < 1 ||
		(*order != 1 && *order != 2) ||
		*commandClass == "" ||
		*deliverable == "" ||
		*output == "" ||
		len(commandArgs) == 0 {
		_, _ = fmt.Fprint(os.Stderr, usage)
		return 64
	}

	command := exec.Command(commandArgs[0], commandArgs[1:]...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	startedAt := now()
	err := command.Run()
	exitCode := processExitCode(err)
	if err != nil {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"neutral-envelope: command failed with exit %d: %v\n",
			exitCode,
			err,
		)
		return exitCode
	}
	deliverableDigest, deliverableSize, err :=
		neutralenvelope.FileSHA256(*deliverable)
	if err != nil {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"neutral-envelope: required deliverable invalid: %v\n",
			err,
		)
		return 1
	}
	completedAt := now()
	observation, err := neutralenvelope.NewObservation(
		*arm,
		*pairIndex,
		*order,
		*commandClass,
		startedAt,
		completedAt,
		exitCode,
		deliverableDigest,
		deliverableSize,
	)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "neutral-envelope: %v\n", err)
		return 1
	}
	if err := neutralenvelope.WriteJSON(*output, observation); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "neutral-envelope: %v\n", err)
		return 1
	}
	return 0
}

func runReport(args []string) int {
	flags := flag.NewFlagSet("report", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	observationsPath := flags.String(
		"observations",
		"",
		"directory containing raw observations",
	)
	executionEnvironment := flags.String(
		"execution-environment",
		"",
		"HOST_SMOKE or STRICT_GOLDEN_CONTAINER",
	)
	runnerSpec := flags.String("runner-spec", "", "golden runner spec")
	metricsCatalog := flags.String("metrics-catalog", "", "metrics catalog")
	envelope := flags.String("envelope", "", "neutral-envelope binary")
	launcher := flags.String("launcher", "", "measured launcher binary")
	server := flags.String("server", "", "measured server binary")
	plugin := flags.String("plugin", "", "measured plugin JAR")
	output := flags.String("output", "", "report output path")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return 64
	}
	if *observationsPath == "" ||
		*executionEnvironment == "" ||
		*runnerSpec == "" ||
		*metricsCatalog == "" ||
		*envelope == "" ||
		*launcher == "" ||
		*server == "" ||
		*plugin == "" ||
		*output == "" {
		_, _ = fmt.Fprint(os.Stderr, usage)
		return 64
	}

	paths, err := filepath.Glob(
		filepath.Join(*observationsPath, "observation-*.json"),
	)
	if err != nil || len(paths) == 0 {
		_, _ = fmt.Fprintln(
			os.Stderr,
			"neutral-envelope: no observations found",
		)
		return 1
	}
	sort.Strings(paths)
	observations := make([]neutralenvelope.Observation, 0, len(paths))
	for _, path := range paths {
		observation, err := neutralenvelope.LoadObservation(path)
		if err != nil {
			_, _ = fmt.Fprintf(
				os.Stderr,
				"neutral-envelope: load %s: %v\n",
				path,
				err,
			)
			return 1
		}
		observations = append(observations, observation)
	}
	report, err := neutralenvelope.BuildReport(
		observations,
		*executionEnvironment,
		*runnerSpec,
		*metricsCatalog,
		*envelope,
		*launcher,
		*server,
		*plugin,
		now(),
	)
	if err != nil {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"neutral-envelope: build report: %v\n",
			err,
		)
		return 1
	}
	if err := neutralenvelope.WriteJSON(*output, report); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "neutral-envelope: %v\n", err)
		return 1
	}
	return printSummary(report)
}

func runValidate(args []string) int {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	reportPath := flags.String("report", "", "overhead report path")
	if err := flags.Parse(args); err != nil ||
		flags.NArg() != 0 ||
		*reportPath == "" {
		return 64
	}
	report, err := neutralenvelope.LoadReport(*reportPath)
	if err != nil {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"neutral-envelope: invalid report: %v\n",
			err,
		)
		return 1
	}
	return printSummary(report)
}

func printSummary(report neutralenvelope.Report) int {
	_, _ = fmt.Fprintf(
		os.Stdout,
		"WS-009 overhead report valid: environment=%s pairs=%d firstMs=%.3f p50Ms=%.3f p95Ms=%.3f promotionGate=false\n",
		report.ExecutionEnvironment,
		report.Summary.PairCount,
		report.Summary.FirstProductOverheadMs,
		report.Summary.ProductOverheadP50Ms,
		report.Summary.ProductOverheadP95Ms,
	)
	return 0
}

func processExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		if exitCode := exitError.ExitCode(); exitCode >= 0 {
			return exitCode
		}
	}
	return 1
}
