package launcher

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/stickyobservation"
	"github.com/tonyredondo/buildopt/internal/stickywrapper"
)

const (
	stickyObservationOutputEnvironment = "BUILDOPT_STICKY_OBSERVATION_OUTPUT"
	stickyObservationModeEnvironment   = "BUILDOPT_STICKY_OBSERVATION"
	ordinaryObservationModeLight       = "light"
	ordinaryObservationModeFull        = "full"
	ordinaryObservationModeDisabled    = "disabled"
)

// ordinaryObservationState is intentionally a best-effort companion to the
// requested Gradle build. It never changes arguments, cache policy or exit
// status; failures to write evidence are reported after the build.
type ordinaryObservationState struct {
	root         string
	outputPath   string
	recorder     *stickyobservation.Recorder
	buildOptHash <-chan string
	record       stickyobservation.Record
	startedAt    time.Time
	connection   time.Time
	cacheStart   time.Time
	gradleRun    childExecution
	gradleSeen   bool
	obsStart     time.Time
}

func newOrdinaryObservationState(root string, args []string) *ordinaryObservationState {
	return newOrdinaryObservationStateAt(root, args, time.Now().UTC())
}

func newOrdinaryObservationStateAt(root string, args []string, startedAt time.Time) *ordinaryObservationState {
	mode, ok := stickyObservationMode(os.Getenv)
	if root == "" || !ok || mode == ordinaryObservationModeDisabled || !isGradleChild(args) {
		return nil
	}
	absolute, err := filepath.Abs(root)
	if err != nil || filepath.Clean(absolute) != absolute {
		return nil
	}
	identity := absolute
	if config, configErr := stickywrapper.LoadConfig(absolute); configErr == nil && config.ProjectScope != "" {
		identity = config.ProjectScope
	}
	scope := stickyobservation.ScopeForRoot(identity)
	output := os.Getenv(stickyObservationOutputEnvironment)
	if output == "" {
		cacheRoot, cacheErr := os.UserCacheDir()
		if cacheErr != nil {
			return nil
		}
		output = filepath.Join(cacheRoot, "buildopt", "sticky", "observations", scope, "builds.jsonl")
	}
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	provenance := ordinaryObservationProvenanceForMode(absolute, scope, args, mode)
	var buildOptHash <-chan string
	if mode == ordinaryObservationModeLight {
		// Hashing the launcher binary can be relatively expensive on a cold
		// filesystem. Do it concurrently with Gradle and keep the explicit
		// unavailable digest when a short build exits first; it must never
		// delay the requested build's start or completion.
		buildOptHash = startOrdinaryObservationBuildOptHash()
	}
	record := stickyobservation.Record{
		SchemaVersion: stickyobservation.SchemaVersion,
		RecordType:    stickyobservation.RecordType,
		ObservationID: "build-" + stickyobservation.Digest(startedAt.Format(time.RFC3339Nano), strings.Join(args, "\x00"))[:24],
		Provenance:    provenance,
		StartedAt:     startedAt.Format(time.RFC3339Nano),
		ConfigurationCache: stickyobservation.ConfigurationCache{
			Requested: hasGradleArgument(args, "--configuration-cache"),
			State:     "NOT_REQUESTED",
		},
	}
	record.Timing.Decision = stickyobservation.Phase{Evidence: "UNAVAILABLE"}
	record.Timing.Network = stickyobservation.Phase{Evidence: "UNAVAILABLE"}
	record.Timing.Cache = stickyobservation.Phase{Evidence: "UNAVAILABLE"}
	record.Timing.Gradle = stickyobservation.Phase{Evidence: "UNAVAILABLE"}
	record.Timing.Observation = stickyobservation.Phase{Evidence: "UNAVAILABLE"}
	record.Timing.Wrapper = stickyobservation.Phase{Evidence: "UNAVAILABLE"}
	record.Timing.Bootstrap = stickyobservation.Phase{Evidence: "UNAVAILABLE"}
	record.IdempotencyKey = stickyobservation.Digest(record.ObservationID, record.Provenance.ArgumentsSHA256)
	return &ordinaryObservationState{
		root: root, outputPath: output, buildOptHash: buildOptHash,
		record: record, startedAt: startedAt,
	}
}

func ordinaryObservationProvenance(root, scope string, args []string) stickyobservation.Provenance {
	return ordinaryObservationProvenanceForMode(root, scope, args, ordinaryObservationModeFull)
}

func ordinaryObservationProvenanceForMode(root, scope string, args []string, mode string) stickyobservation.Provenance {
	provenance := stickyobservation.Provenance{
		RepositoryScopeSHA256:  scope,
		SourceRevisionEvidence: "UNAVAILABLE",
		GradleVersion:          "UNAVAILABLE",
		WrapperSHA256:          stickyobservation.Digest("unavailable-wrapper"),
		BuildOptSHA256:         stickyobservation.Digest("unavailable-buildopt"),
		ArgumentsSHA256:        stickyobservation.Digest("gradle-arguments-v1", strings.Join(args, "\x00")),
	}
	if mode == ordinaryObservationModeFull {
		if revision, err := gitOutput(root, "rev-parse", "HEAD"); err == nil {
			revision = strings.ToLower(strings.TrimSpace(revision))
			if validMeasurementRevision(revision) {
				provenance.SourceRevision = revision
				provenance.SourceRevisionEvidence = "EXACT"
			}
		}
	}
	if version, err := centralGradleVersion(root); err == nil {
		provenance.GradleVersion = version
	}
	if hash, err := optimizeFileSHA256(filepath.Join(root, "gradle", "wrapper", "gradle-wrapper.properties"), true); err == nil {
		provenance.WrapperSHA256 = hash
	}
	if mode == ordinaryObservationModeFull {
		if executable, err := os.Executable(); err == nil {
			if hash, hashErr := optimizeFileSHA256(executable, false); hashErr == nil {
				provenance.BuildOptSHA256 = hash
			}
		}
	}
	return provenance
}

func startOrdinaryObservationBuildOptHash() <-chan string {
	result := make(chan string, 1)
	go func() {
		hash := ""
		if executable, err := os.Executable(); err == nil {
			if value, hashErr := optimizeFileSHA256(executable, false); hashErr == nil {
				hash = value
			}
		}
		result <- hash
	}()
	return result
}

func stickyObservationMode(getenv func(string) string) (string, bool) {
	if getenv == nil {
		return ordinaryObservationModeDisabled, false
	}
	switch strings.ToLower(strings.TrimSpace(getenv(stickyObservationModeEnvironment))) {
	case "", "1", ordinaryObservationModeLight:
		return ordinaryObservationModeLight, true
	case ordinaryObservationModeFull:
		return ordinaryObservationModeFull, true
	case "0":
		return ordinaryObservationModeDisabled, true
	default:
		return ordinaryObservationModeDisabled, false
	}
}

func isGradleChild(args []string) bool {
	if len(args) == 0 {
		return false
	}
	base := filepath.Base(args[0])
	return base == "gradlew" || base == "gradlew.bat"
}

func ordinaryChildArguments(args []string) []string {
	if len(args) >= 3 && args[0] == "run" && args[1] == "--" {
		return args[2:]
	}
	return nil
}

func hasGradleArgument(args []string, wanted string) bool {
	for _, arg := range args[1:] {
		if arg == wanted {
			return true
		}
	}
	return false
}

func (state *ordinaryObservationState) markConnection(start time.Time, attempted bool) {
	if state == nil {
		return
	}
	if start.IsZero() || start.Before(state.startedAt) {
		start = state.startedAt
	}
	state.record.Timing.Decision = stickyobservation.Phase{
		DurationNs: start.Sub(state.startedAt).Nanoseconds(), Evidence: "EXACT",
	}
	state.connection = time.Now()
	if attempted {
		state.record.Timing.Network = stickyobservation.Phase{
			DurationNs: state.connection.Sub(start).Nanoseconds(), Evidence: "EXACT",
		}
	} else {
		state.record.Timing.Network = stickyobservation.Phase{Evidence: "UNAVAILABLE"}
	}
	state.cacheStart = state.connection
}

func (state *ordinaryObservationState) finishCache(at time.Time) {
	if state == nil || state.cacheStart.IsZero() {
		return
	}
	if at.Before(state.cacheStart) {
		at = state.cacheStart
	}
	state.record.Timing.Cache = stickyobservation.Phase{
		DurationNs: at.Sub(state.cacheStart).Nanoseconds(), Evidence: "APPROXIMATED",
	}
}

func (state *ordinaryObservationState) finishGradle(execution childExecution, args []string) {
	if state == nil {
		return
	}
	state.gradleRun = execution
	state.gradleSeen = execution.started
	if execution.started && !execution.completedAt.Before(execution.startedAt) {
		state.record.Timing.Gradle = stickyobservation.Phase{
			DurationNs: execution.completedAt.Sub(execution.startedAt).Nanoseconds(), Evidence: "EXACT",
		}
	}
	state.record.ConfigurationCache.Requested = hasGradleArgument(args, "--configuration-cache")
	if !state.record.ConfigurationCache.Requested {
		state.record.ConfigurationCache.State = "NOT_REQUESTED"
		return
	}
	cacheState := filepath.Join(state.root, ".gradle", "configuration-cache")
	if entries, err := os.ReadDir(cacheState); err == nil && len(entries) > 0 {
		state.record.ConfigurationCache.State = "PRESENT"
	} else if errors.Is(err, os.ErrNotExist) {
		state.record.ConfigurationCache.State = "ABSENT"
	} else {
		state.record.ConfigurationCache.State = "UNAVAILABLE"
	}
}

func (state *ordinaryObservationState) startObservation() {
	if state != nil {
		state.obsStart = time.Now()
	}
}

func (state *ordinaryObservationState) finishObservation(at time.Time) {
	if state == nil || state.obsStart.IsZero() {
		return
	}
	if at.Before(state.obsStart) {
		at = state.obsStart
	}
	state.record.Timing.Observation = stickyobservation.Phase{
		DurationNs: at.Sub(state.obsStart).Nanoseconds(), Evidence: "EXACT",
	}
}

func (state *ordinaryObservationState) finish(exitCode int, completedAt time.Time) error {
	if state == nil || state.outputPath == "" {
		return nil
	}
	if state.buildOptHash != nil {
		select {
		case hash := <-state.buildOptHash:
			if hash != "" {
				state.record.Provenance.BuildOptSHA256 = hash
			}
		default:
		}
	}
	if state.recorder == nil {
		recorder, err := stickyobservation.NewRecorder(state.outputPath)
		if err != nil {
			return err
		}
		state.recorder = recorder
	}
	if completedAt.Before(state.startedAt) {
		completedAt = state.startedAt
	}
	total := completedAt.Sub(state.startedAt).Nanoseconds()
	if total <= 0 {
		return fmt.Errorf("ordinary observation wall time is not positive")
	}
	state.record.CompletedAt = completedAt.UTC().Format(time.RFC3339Nano)
	state.record.Timing.TotalNs = total
	if state.gradleSeen {
		if state.gradleRun.cancelled {
			state.record.Outcome = "CANCELLED"
		} else if exitCode == 0 {
			state.record.Outcome = "SUCCESS"
		} else {
			state.record.Outcome = "BUILD_FAILURE"
		}
	} else {
		state.record.Outcome = "INFRA_FAILURE"
	}
	state.record.ExitCode = exitCode
	accounted := state.record.Timing.Decision.DurationNs + state.record.Timing.Cache.DurationNs +
		state.record.Timing.Gradle.DurationNs + state.record.Timing.Observation.DurationNs
	if state.record.Timing.Network.Evidence != "UNAVAILABLE" {
		accounted += state.record.Timing.Network.DurationNs
	}
	if accounted > total {
		return fmt.Errorf("ordinary observation phase timings exceed wall time")
	}
	state.record.Timing.UnattributedNs = total - accounted
	state.record.Timing.Wrapper = stickyobservation.Phase{Evidence: "UNAVAILABLE"}
	state.record.Timing.Bootstrap = stickyobservation.Phase{Evidence: "UNAVAILABLE"}
	return state.recorder.Append(state.record)
}
