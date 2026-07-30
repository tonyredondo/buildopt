package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"syscall"
	"time"

	"github.com/tonyredondo/buildopt/internal/neutralenvelope"
)

const usage = `usage:
  neutral-envelope observe --arm ARM --pair N --order N --command-class NAME --deliverable PATH --output PATH -- <command> [args...]
  neutral-envelope report --observations DIR --execution-environment NAME --runner-spec PATH --metrics-catalog PATH --envelope PATH --launcher PATH --server PATH --plugin PATH --output PATH
  neutral-envelope validate --report PATH
  neutral-envelope pilot-assign --experiment ID --epoch N --action ID --baseline-digest SHA256 --control-digest SHA256 --cohort ID --environment NAME --pipeline NAME --runner NAME --work-units HMAC_SHA256 --required-deliverable NAME --pair N --arm ARM --output PATH
  neutral-envelope pilot-observe --assignment PATH --deliverable PATH --output PATH -- <command> [args...]
  neutral-envelope pilot-report --observations DIR --incremental-overhead-ms N --export-dir DIR
  neutral-envelope pilot-validate --result PATH --observations DIR
  neutral-envelope pilot-export --export-dir DIR
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
	case "pilot-assign":
		return runPilotAssign(args[1:])
	case "pilot-observe":
		return runPilotObserve(args[1:])
	case "pilot-report":
		return runPilotReport(args[1:])
	case "pilot-validate":
		return runPilotValidate(args[1:])
	case "pilot-export":
		return runPilotExport(args[1:])
	default:
		_, _ = fmt.Fprint(os.Stderr, usage)
		return 64
	}
}

func runPilotAssign(args []string) int {
	flags := flag.NewFlagSet("pilot-assign", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	experimentID := flags.String("experiment", "", "experiment identity")
	epoch := flags.Int("epoch", 0, "positive measurement epoch")
	actionID := flags.String("action", "", "evaluated action identity")
	baselineDigest := flags.String(
		"baseline-digest",
		"",
		"versioned customer baseline SHA-256",
	)
	controlDigest := flags.String(
		"control-digest",
		"",
		"action-off control definition SHA-256",
	)
	cohortID := flags.String("cohort", "", "fixed internal cohort")
	environment := flags.String("environment", "", "environment stratum")
	pipelineClass := flags.String("pipeline", "", "pipeline stratum")
	runnerClass := flags.String("runner", "", "runner stratum")
	workUnits := flags.String(
		"work-units",
		"",
		"tokenized work-units fingerprint",
	)
	requiredDeliverable := flags.String(
		"required-deliverable",
		"",
		"logical required deliverable",
	)
	pairIndex := flags.Int("pair", 0, "one-based pair index")
	arm := flags.String("arm", "", "CONTROL or CANDIDATE")
	output := flags.String("output", "", "immutable assignment output")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return 64
	}
	if *experimentID == "" ||
		*epoch < 1 ||
		*actionID == "" ||
		*baselineDigest == "" ||
		*controlDigest == "" ||
		*cohortID == "" ||
		*environment == "" ||
		*pipelineClass == "" ||
		*runnerClass == "" ||
		*workUnits == "" ||
		*requiredDeliverable == "" ||
		*pairIndex < 1 ||
		(*arm != "CONTROL" && *arm != "CANDIDATE") ||
		*output == "" {
		_, _ = fmt.Fprint(os.Stderr, usage)
		return 64
	}
	assignment, err := neutralenvelope.NewPilotAssignment(
		neutralenvelope.PilotDefinition{
			ExperimentID:             *experimentID,
			MeasurementEpoch:         *epoch,
			ActionID:                 *actionID,
			BaselineDefinitionDigest: *baselineDigest,
			ControlDefinitionDigest:  *controlDigest,
			CohortID:                 *cohortID,
			Environment:              *environment,
			PipelineClass:            *pipelineClass,
			RunnerClass:              *runnerClass,
			WorkUnitsFingerprint:     *workUnits,
			RequiredDeliverable:      *requiredDeliverable,
		},
		*pairIndex,
		*arm,
		now(),
	)
	if err != nil {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"neutral-envelope: create pilot assignment: %v\n",
			err,
		)
		return 1
	}
	if err := neutralenvelope.WritePilotAssignment(
		*output,
		assignment,
	); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "neutral-envelope: %v\n", err)
		return 1
	}
	return 0
}

func runPilotObserve(args []string) int {
	flags := flag.NewFlagSet("pilot-observe", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	assignmentPath := flags.String(
		"assignment",
		"",
		"immutable pre-outcome assignment",
	)
	deliverable := flags.String(
		"deliverable",
		"",
		"required deliverable path",
	)
	output := flags.String("output", "", "immutable observation output")
	if err := flags.Parse(args); err != nil {
		return 64
	}
	commandArgs := flags.Args()
	if *assignmentPath == "" ||
		*deliverable == "" ||
		*output == "" ||
		len(commandArgs) == 0 {
		_, _ = fmt.Fprint(os.Stderr, usage)
		return 64
	}
	assignment, err := neutralenvelope.LoadPilotAssignment(*assignmentPath)
	if err != nil {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"neutral-envelope: invalid pilot assignment: %v\n",
			err,
		)
		return 1
	}

	command := exec.Command(commandArgs[0], commandArgs[1:]...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	startedAt := now()
	commandErr := command.Run()
	exitCode := processExitCode(commandErr)
	outcome := "SUCCESS"
	if commandErr != nil {
		var exitError *exec.ExitError
		if !errors.As(commandErr, &exitError) {
			outcome = "INFRA_FAILURE"
		} else {
			outcome = "BUILD_FAILURE"
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok &&
				status.Signaled() {
				outcome = "CANCELLED"
			}
		}
	}
	deliverableStatus := "NOT_AVAILABLE"
	deliverableDigest := ""
	deliverableSize := int64(0)
	if digest, size, digestErr := neutralenvelope.FileSHA256(
		*deliverable,
	); digestErr == nil {
		deliverableStatus = "AVAILABLE"
		deliverableDigest = digest
		deliverableSize = size
	} else if commandErr == nil {
		outcome = "INFRA_FAILURE"
		exitCode = 0
	}
	completedAt := now()
	observation, err := neutralenvelope.NewPilotObservation(
		assignment,
		startedAt,
		completedAt,
		outcome,
		exitCode,
		deliverableStatus,
		deliverableDigest,
		deliverableSize,
	)
	if err != nil {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"neutral-envelope: create pilot observation: %v\n",
			err,
		)
		return 1
	}
	if err := neutralenvelope.WritePilotObservation(
		*output,
		observation,
	); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "neutral-envelope: %v\n", err)
		return 1
	}
	if commandErr != nil {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"neutral-envelope: pilot command failed with exit %d: %v\n",
			exitCode,
			commandErr,
		)
		return exitCode
	}
	if outcome != "SUCCESS" {
		_, _ = fmt.Fprintln(
			os.Stderr,
			"neutral-envelope: required pilot deliverable is unavailable",
		)
		return 1
	}
	return 0
}

func runPilotReport(args []string) int {
	flags := flag.NewFlagSet("pilot-report", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	observationsPath := flags.String(
		"observations",
		"",
		"directory containing pilot observations",
	)
	incrementalOverhead := flags.Int64(
		"incremental-overhead-ms",
		0,
		"non-negative separately observed action overhead",
	)
	exportDirectory := flags.String(
		"export-dir",
		"",
		"private result export directory",
	)
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return 64
	}
	if *observationsPath == "" ||
		*incrementalOverhead < 0 ||
		*exportDirectory == "" {
		_, _ = fmt.Fprint(os.Stderr, usage)
		return 64
	}
	observations, err := loadPilotObservations(*observationsPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "neutral-envelope: %v\n", err)
		return 1
	}
	result, err := neutralenvelope.BuildPilotResult(
		observations,
		*incrementalOverhead,
		now(),
	)
	if err != nil {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"neutral-envelope: build pilot result: %v\n",
			err,
		)
		return 1
	}
	documentPath, _, err := neutralenvelope.PublishPilotResult(
		*exportDirectory,
		result,
	)
	if err != nil {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"neutral-envelope: publish pilot result: %v\n",
			err,
		)
		return 1
	}
	return printPilotSummary(result, documentPath)
}

func runPilotValidate(args []string) int {
	flags := flag.NewFlagSet("pilot-validate", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	resultPath := flags.String("result", "", "EXPERIMENT_RESULT JSON")
	observationsPath := flags.String(
		"observations",
		"",
		"directory containing pilot observations",
	)
	if err := flags.Parse(args); err != nil ||
		flags.NArg() != 0 ||
		*resultPath == "" ||
		*observationsPath == "" {
		return 64
	}
	result, err := neutralenvelope.LoadExperimentResult(*resultPath)
	if err != nil {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"neutral-envelope: invalid pilot result: %v\n",
			err,
		)
		return 1
	}
	observations, err := loadPilotObservations(*observationsPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "neutral-envelope: %v\n", err)
		return 1
	}
	if err := neutralenvelope.ValidatePilotResult(
		result,
		observations,
	); err != nil {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"neutral-envelope: invalid pilot result: %v\n",
			err,
		)
		return 1
	}
	return printPilotSummary(result, *resultPath)
}

func runPilotExport(args []string) int {
	flags := flag.NewFlagSet("pilot-export", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	exportDirectory := flags.String(
		"export-dir",
		"",
		"private result export directory",
	)
	if err := flags.Parse(args); err != nil ||
		flags.NArg() != 0 ||
		*exportDirectory == "" {
		return 64
	}
	if err := neutralenvelope.WritePilotResultStream(
		*exportDirectory,
		os.Stdout,
	); err != nil {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"neutral-envelope: export pilot result: %v\n",
			err,
		)
		return 1
	}
	return 0
}

func loadPilotObservations(
	directory string,
) ([]neutralenvelope.PilotObservation, error) {
	paths, err := filepath.Glob(
		filepath.Join(directory, "observation-*.json"),
	)
	if err != nil || len(paths) == 0 {
		return nil, errors.New("no causal-pilot observations found")
	}
	sort.Strings(paths)
	observations := make(
		[]neutralenvelope.PilotObservation,
		0,
		len(paths),
	)
	for _, path := range paths {
		observation, err := neutralenvelope.LoadPilotObservation(path)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", path, err)
		}
		observations = append(observations, observation)
	}
	return observations, nil
}

func printPilotSummary(
	result neutralenvelope.ExperimentResult,
	path string,
) int {
	_, _ = fmt.Fprintf(
		os.Stdout,
		"A0-009 causal pilot valid: result=%s pairs=%d savedMs=%d lower95Ms=%d preliminary=true netSavings=%t\n",
		path,
		result.Samples.Analyzed.Control,
		result.Effects.ObservedNetBuildTimeSavedMs,
		result.Effects.ObservedNetBuildTimeSavedInterval95Ms[0],
		result.DemonstratesNetCausalSavings(),
	)
	return 0
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
