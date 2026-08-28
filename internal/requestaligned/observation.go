// Package requestaligned derives portable, observation-only identities and
// current producer outputs for ordinary recurrent Gradle requests.
package requestaligned

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"sort"
	"strings"

	"github.com/tonyredondo/buildopt/internal/changeaware"
)

const (
	CaptureSchemaVersion     = "buildopt.poc/request-aligned-producer-capture/v1"
	ObservationSchemaVersion = "buildopt.poc/request-aligned-observation/v1"

	CaptureComplete    = "COMPLETE"
	CaptureUnavailable = "UNAVAILABLE"
	CaptureFailed      = "FAILED"

	StatusComplete    = "COMPLETE"
	StatusUnavailable = "UNAVAILABLE"
	StatusFailed      = "PRODUCER_FAILED"
)

// Capture is the portable evidence emitted after one ordinary requested
// Gradle graph has completed. It contains no candidate or timing authority.
type Capture struct {
	SchemaVersion            string                     `json:"schemaVersion"`
	GeneratedAt              string                     `json:"generatedAt"`
	Status                   string                     `json:"status"`
	Reason                   string                     `json:"reason"`
	GradleArguments          []string                   `json:"gradleArguments"`
	RequestedTasks           []string                   `json:"requestedTasks"`
	GradleVersion            string                     `json:"gradleVersion"`
	JavaRuntime              JavaRuntime                `json:"javaRuntime"`
	EnvironmentBindingSHA256 string                     `json:"environmentBindingSha256"`
	WrapperFiles             []FileBinding              `json:"wrapperFiles"`
	BuildLogicFiles          []FileBinding              `json:"buildLogicFiles"`
	Tasks                    []changeaware.TaskEvidence `json:"tasks"`
}

// JavaRuntime contains portable JVM compatibility facts. java.home and other
// machine paths are deliberately excluded.
type JavaRuntime struct {
	Version      string `json:"version"`
	Vendor       string `json:"vendor"`
	RuntimeName  string `json:"runtimeName"`
	VMName       string `json:"vmName"`
	Architecture string `json:"architecture"`
}

// FileBinding binds one repository-relative configuration file to its bytes.
type FileBinding struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// ProducerOutput is a current path reported by Gradle and its unique producer.
type ProducerOutput struct {
	ProducerTask string `json:"producerTask"`
	Path         string `json:"path"`
	Kind         string `json:"kind"`
	SHA256       string `json:"sha256"`
}

// Observation is a validated, non-authorizing request identity and current
// producer-output inventory.
type Observation struct {
	SchemaVersion            string                     `json:"schemaVersion"`
	GeneratedAt              string                     `json:"generatedAt"`
	Status                   string                     `json:"status"`
	Reason                   string                     `json:"reason"`
	GradleArguments          []string                   `json:"gradleArguments"`
	RequestedTasks           []string                   `json:"requestedTasks"`
	GradleVersion            string                     `json:"gradleVersion"`
	JavaRuntime              JavaRuntime                `json:"javaRuntime"`
	EnvironmentBindingSHA256 string                     `json:"environmentBindingSha256"`
	WrapperSHA256            string                     `json:"wrapperSha256"`
	BuildLogicSHA256         string                     `json:"buildLogicSha256"`
	TaskGraphSHA256          string                     `json:"taskGraphSha256"`
	RequestIdentitySHA256    string                     `json:"requestIdentitySha256"`
	Tasks                    []changeaware.TaskEvidence `json:"tasks"`
	CurrentOutputs           []ProducerOutput           `json:"currentOutputs"`
	PerformanceMeasured      bool                       `json:"performanceMeasured"`
	ActivationAuthorized     bool                       `json:"activationAuthorized"`
}

// Produce validates a capture and derives its checkout-independent identity.
// Ambiguous or missing output ownership remains typed unavailable.
func Produce(capture Capture) (Observation, error) {
	observation := baseObservation(capture)
	if capture.SchemaVersion != CaptureSchemaVersion || capture.GeneratedAt == "" {
		return Observation{}, errors.New("request-aligned capture identity is invalid")
	}
	switch capture.Status {
	case CaptureUnavailable:
		if capture.Reason == "" || len(capture.Tasks) != 0 {
			return Observation{}, errors.New("unavailable request-aligned capture is invalid")
		}
		observation.Status, observation.Reason = StatusUnavailable, capture.Reason
		return observation, nil
	case CaptureFailed:
		if capture.Reason == "" || len(capture.Tasks) != 0 {
			return Observation{}, errors.New("failed request-aligned capture is invalid")
		}
		observation.Status, observation.Reason = StatusFailed, capture.Reason
		return observation, nil
	case CaptureComplete:
		if capture.Reason != "" {
			return Observation{}, errors.New("complete request-aligned capture has a failure reason")
		}
	default:
		return Observation{}, errors.New("unknown request-aligned capture status")
	}

	if err := validateArguments(capture.GradleArguments); err != nil ||
		validateStringSet(capture.RequestedTasks, safeTaskPath) != nil ||
		!safeToken(capture.GradleVersion) || !validRuntime(capture.JavaRuntime) ||
		!validSHA(capture.EnvironmentBindingSHA256) {
		return Observation{}, errors.New("request-aligned request binding is invalid")
	}
	wrapperRows, wrapperSHA, err := canonicalFiles(capture.WrapperFiles)
	if err != nil {
		return Observation{}, errors.New("request-aligned Wrapper binding is invalid")
	}
	buildRows, buildSHA, err := canonicalFiles(capture.BuildLogicFiles)
	if err != nil {
		return Observation{}, errors.New("request-aligned build-logic binding is invalid")
	}
	tasks, graphRows, err := validateTasks(capture.Tasks)
	if err != nil {
		return Observation{}, err
	}
	for _, requested := range capture.RequestedTasks {
		if _, exists := tasks[requested]; !exists {
			return unavailable(observation, "REQUESTED_TASK_UNAVAILABLE"), nil
		}
	}

	outputs := []ProducerOutput{}
	owners := map[string]string{}
	for _, task := range tasks {
		for _, output := range task.Outputs {
			if !output.Exists {
				continue
			}
			identity := output.Kind + "\x00" + output.Path
			if owner, exists := owners[identity]; exists && owner != task.Path {
				return unavailable(observation, "CURRENT_OUTPUT_PRODUCER_AMBIGUOUS"), nil
			}
			owners[identity] = task.Path
			outputs = append(outputs, ProducerOutput{
				ProducerTask: task.Path, Path: output.Path, Kind: output.Kind, SHA256: output.SHA256,
			})
		}
	}
	if len(outputs) == 0 {
		return unavailable(observation, "CURRENT_PRODUCER_OUTPUTS_EMPTY"), nil
	}
	sort.Slice(outputs, func(i, j int) bool {
		if outputs[i].ProducerTask != outputs[j].ProducerTask {
			return outputs[i].ProducerTask < outputs[j].ProducerTask
		}
		if outputs[i].Path != outputs[j].Path {
			return outputs[i].Path < outputs[j].Path
		}
		return outputs[i].Kind < outputs[j].Kind
	})

	graphSHA := digest("buildopt-request-task-graph-v1", graphRows...)
	runtimeSHA := digest("buildopt-request-java-runtime-v1",
		capture.JavaRuntime.Version, capture.JavaRuntime.Vendor,
		capture.JavaRuntime.RuntimeName, capture.JavaRuntime.VMName,
		capture.JavaRuntime.Architecture)
	argumentSHA := digest("buildopt-request-gradle-arguments-v1", capture.GradleArguments...)
	requested := append([]string(nil), capture.RequestedTasks...)
	sort.Strings(requested)
	requestedSHA := digest("buildopt-request-task-roots-v1", requested...)

	observation.Status, observation.Reason = StatusComplete, "CURRENT_REQUEST_AND_OUTPUT_EVIDENCE_COMPLETE"
	observation.WrapperSHA256 = wrapperSHA
	observation.BuildLogicSHA256 = buildSHA
	observation.TaskGraphSHA256 = graphSHA
	observation.RequestIdentitySHA256 = digest("buildopt-request-identity-v1",
		argumentSHA, requestedSHA, capture.GradleVersion, runtimeSHA,
		capture.EnvironmentBindingSHA256, wrapperSHA, buildSHA, graphSHA,
		digest("buildopt-request-wrapper-files-v1", wrapperRows...),
		digest("buildopt-request-build-logic-files-v1", buildRows...))
	observation.CurrentOutputs = outputs
	return observation, nil
}

func baseObservation(capture Capture) Observation {
	return Observation{
		SchemaVersion: ObservationSchemaVersion, GeneratedAt: capture.GeneratedAt,
		GradleArguments: append([]string(nil), capture.GradleArguments...),
		RequestedTasks:  append([]string(nil), capture.RequestedTasks...),
		GradleVersion:   capture.GradleVersion, JavaRuntime: capture.JavaRuntime,
		EnvironmentBindingSHA256: capture.EnvironmentBindingSHA256,
		Tasks:                    append([]changeaware.TaskEvidence(nil), capture.Tasks...),
		CurrentOutputs:           []ProducerOutput{}, PerformanceMeasured: false, ActivationAuthorized: false,
	}
}

func unavailable(observation Observation, reason string) Observation {
	observation.Status, observation.Reason = StatusUnavailable, reason
	observation.CurrentOutputs = []ProducerOutput{}
	return observation
}

func validateArguments(values []string) error {
	if len(values) == 0 || len(values) > 1024 {
		return errors.New("Gradle arguments are empty or excessive")
	}
	for _, value := range values {
		if value == "" || len(value) > 4096 || strings.ContainsRune(value, '\x00') {
			return errors.New("Gradle argument is invalid")
		}
	}
	return nil
}

func validRuntime(value JavaRuntime) bool {
	return safeToken(value.Version) && safeToken(value.Vendor) && safeToken(value.RuntimeName) &&
		safeToken(value.VMName) && safeToken(value.Architecture)
}

func safeToken(value string) bool {
	return value != "" && len(value) <= 512 && !strings.ContainsAny(value, "\x00\r\n")
}

func canonicalFiles(values []FileBinding) ([]string, string, error) {
	if len(values) == 0 || len(values) > 100000 {
		return nil, "", errors.New("file bindings are empty or excessive")
	}
	rows := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		if !safeRepositoryPath(value.Path) || !validSHA(value.SHA256) || seen[value.Path] {
			return nil, "", errors.New("file binding is invalid")
		}
		seen[value.Path] = true
		rows = append(rows, value.Path+"\x00"+value.SHA256)
	}
	sort.Strings(rows)
	return rows, digest("buildopt-request-file-bindings-v1", rows...), nil
}

func validateTasks(values []changeaware.TaskEvidence) (map[string]changeaware.TaskEvidence, []string, error) {
	if len(values) == 0 || len(values) > 250000 {
		return nil, nil, errors.New("request-aligned task graph is empty or excessive")
	}
	tasks := make(map[string]changeaware.TaskEvidence, len(values))
	for _, task := range values {
		if !safeTaskPath(task.Path) || tasks[task.Path].Path != "" ||
			validateStringSetAllowEmpty(task.DependsOn, safeTaskPath) != nil ||
			validatePaths(task.Inputs) != nil || validateOutputs(task.Outputs) != nil {
			return nil, nil, errors.New("request-aligned task evidence is invalid")
		}
		tasks[task.Path] = task
	}
	rows := make([]string, 0, len(tasks))
	for taskPath, task := range tasks {
		dependencies := append([]string(nil), task.DependsOn...)
		sort.Strings(dependencies)
		for _, dependency := range dependencies {
			if dependency == taskPath || tasks[dependency].Path == "" {
				return nil, nil, errors.New("request-aligned task dependency is incomplete")
			}
		}
		rows = append(rows, taskPath+"\x00"+strings.Join(dependencies, "\x00"))
	}
	sort.Strings(rows)
	return tasks, rows, nil
}

func validatePaths(values []changeaware.PathEvidence) error {
	seen := map[string]bool{}
	for _, value := range values {
		identity := value.Kind + "\x00" + value.Path
		if !safeRepositoryPath(value.Path) || (value.Kind != "FILE" && value.Kind != "DIRECTORY") || seen[identity] {
			return errors.New("invalid task input")
		}
		seen[identity] = true
	}
	return nil
}

func validateOutputs(values []changeaware.OutputEvidence) error {
	seen := map[string]bool{}
	for _, value := range values {
		identity := value.Kind + "\x00" + value.Path
		if !safeRepositoryPath(value.Path) || (value.Kind != "FILE" && value.Kind != "DIRECTORY") ||
			(value.Exists && !validSHA(value.SHA256)) || (!value.Exists && value.SHA256 != "") || seen[identity] {
			return errors.New("invalid task output")
		}
		seen[identity] = true
	}
	return nil
}

func validateStringSet(values []string, valid func(string) bool) error {
	if len(values) == 0 {
		return errors.New("values are empty")
	}
	return validateStringSetAllowEmpty(values, valid)
}

func validateStringSetAllowEmpty(values []string, valid func(string) bool) error {
	seen := map[string]bool{}
	for _, value := range values {
		if !valid(value) || seen[value] {
			return errors.New("values are invalid")
		}
		seen[value] = true
	}
	return nil
}

func safeTaskPath(value string) bool {
	if value == ":" || !strings.HasPrefix(value, ":") || len(value) > 1024 || strings.ContainsAny(value, "\x00\r\n/\\") {
		return false
	}
	for _, part := range strings.Split(strings.TrimPrefix(value, ":"), ":") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func safeRepositoryPath(value string) bool {
	if value == "" || len(value) > 4096 || strings.ContainsAny(value, "\x00\r\n\\") || strings.HasPrefix(value, "/") {
		return false
	}
	cleaned := strings.TrimPrefix(value, "./")
	return cleaned == value && !strings.Contains(value, "//") &&
		!strings.Contains("/"+value+"/", "/../") && !strings.Contains("/"+value+"/", "/./")
}

func validSHA(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func digest(domain string, values ...string) string {
	hash := sha256.New()
	writeValue(hash, domain)
	for _, value := range values {
		writeValue(hash, value)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func writeValue(writer io.Writer, value string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = writer.Write(size[:])
	_, _ = io.WriteString(writer, value)
}
