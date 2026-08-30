package requesthit

import (
	"path"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Reason is a stable fail-closed classification. Empty means the contract is
// complete; every non-empty value retains exact native Gradle execution.
type Reason string

const (
	ReasonNone                          Reason = ""
	ReasonRecordIdentityInvalid         Reason = "RECORD_IDENTITY_INVALID"
	ReasonArgumentVectorUnavailable     Reason = "ARGUMENT_VECTOR_UNAVAILABLE"
	ReasonArgumentVectorDrift           Reason = "ARGUMENT_VECTOR_DRIFT"
	ReasonWorkingDirectoryInvalid       Reason = "WORKING_DIRECTORY_INVALID"
	ReasonWorkingDirectoryDrift         Reason = "WORKING_DIRECTORY_DRIFT"
	ReasonRepositoryIdentityUnavailable Reason = "REPOSITORY_IDENTITY_UNAVAILABLE"
	ReasonRepositoryIdentityDrift       Reason = "REPOSITORY_IDENTITY_DRIFT"
	ReasonWrapperDrift                  Reason = "WRAPPER_DRIFT"
	ReasonGradleDrift                   Reason = "GRADLE_DRIFT"
	ReasonJDKDrift                      Reason = "JDK_DRIFT"
	ReasonEnvironmentDrift              Reason = "ENVIRONMENT_DRIFT"
	ReasonRequestGraphDrift             Reason = "REQUEST_GRAPH_DRIFT"
	ReasonTaskImplementationDrift       Reason = "TASK_IMPLEMENTATION_DRIFT"
	ReasonBuildLogicDrift               Reason = "BUILD_LOGIC_DRIFT"
	ReasonRepositoryInputsIncomplete    Reason = "REPOSITORY_INPUTS_INCOMPLETE"
	ReasonRepositoryInputDrift          Reason = "REPOSITORY_INPUT_DRIFT"
	ReasonExternalInputsIncomplete      Reason = "EXTERNAL_INPUTS_INCOMPLETE"
	ReasonExternalInputMissing          Reason = "EXTERNAL_INPUT_MISSING"
	ReasonExternalInputDrift            Reason = "EXTERNAL_INPUT_DRIFT"
	ReasonOutputContractIncomplete      Reason = "OUTPUT_CONTRACT_INCOMPLETE"
	ReasonOutputContractDrift           Reason = "OUTPUT_CONTRACT_DRIFT"
	ReasonUntrackedOutput               Reason = "UNTRACKED_OUTPUT"
	ReasonOutputMissing                 Reason = "OUTPUT_MISSING"
	ReasonOutputAltered                 Reason = "OUTPUT_ALTERED"
	ReasonAbsentOutputPresent           Reason = "ABSENT_OUTPUT_PRESENT"
	ReasonMaterializationInvalid        Reason = "MATERIALIZATION_REFERENCE_INVALID"
	ReasonTaskInventoryUnavailable      Reason = "TASK_INVENTORY_UNAVAILABLE"
	ReasonNonCacheableTask              Reason = "NON_CACHEABLE_TASK"
	ReasonUntrackedTask                 Reason = "UNTRACKED_TASK"
	ReasonAlwaysRunTask                 Reason = "ALWAYS_RUN_TASK"
	ReasonLocalState                    Reason = "LOCAL_STATE_DECLARED"
	ReasonDestroyables                  Reason = "DESTROYABLES_DECLARED"
	ReasonUntrackedWrite                Reason = "UNTRACKED_WRITE_DECLARED"
	ReasonSideEffectfulTask             Reason = "SIDE_EFFECTFUL_TASK"
	ReasonPriorFailure                  Reason = "PRIOR_BUILD_FAILURE"
	ReasonPriorCancellation             Reason = "PRIOR_BUILD_CANCELLED"
	ReasonPriorOutcomeUnsafe            Reason = "PRIOR_OUTCOME_UNSAFE"
	ReasonPriorOutputsUnverified        Reason = "PRIOR_OUTPUTS_UNVERIFIED"
	ReasonEvidenceExpired               Reason = "EVIDENCE_EXPIRED"
	ReasonEvidenceRevoked               Reason = "EVIDENCE_REVOKED"
	ReasonCurrentEvidenceIncomplete     Reason = "CURRENT_EVIDENCE_INCOMPLETE"
)

var (
	digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
	gradlePattern = regexp.MustCompile(`^[0-9]+\.[0-9]+(?:\.[0-9]+)?(?:[-+][A-Za-z0-9.-]+)?$`)
	taskPattern   = regexp.MustCompile(`^(:[A-Za-z0-9_.-]+)+$`)
)

// Verify compares a record with current facts without selecting or executing
// an action. Callers must run native Gradle for every non-complete verdict.
func Verify(record SafetyRecord, probe Probe, now time.Time) Verdict {
	_, digest, _ := CanonicalRecord(record)
	retain := func(reason Reason) Verdict {
		return Verdict{Disposition: DispositionRetainNative, Reason: reason, RecordSHA256: digest}
	}
	if reason := validateRecord(record); reason != ReasonNone {
		return retain(reason)
	}
	if !probe.EvidenceComplete {
		return retain(ReasonCurrentEvidenceIncomplete)
	}
	if now.IsZero() {
		return retain(ReasonCurrentEvidenceIncomplete)
	}
	expires, _ := parseTimestamp(record.ExpiresAt)
	if !now.UTC().Before(expires) {
		return retain(ReasonEvidenceExpired)
	}
	if probe.CurrentRevocationEpoch > record.RevocationEpoch {
		return retain(ReasonEvidenceRevoked)
	}
	if probe.CurrentRevocationEpoch < 0 || probe.CurrentRevocationEpoch < record.RevocationEpoch {
		return retain(ReasonCurrentEvidenceIncomplete)
	}
	if reason := validateRequest(probe.Request); reason != ReasonNone {
		return retain(reason)
	}
	if probe.Request.ArgumentsSHA256 != record.Request.ArgumentsSHA256 || probe.Request.ArgumentCount != record.Request.ArgumentCount || probe.Request.ArgumentEncoding != record.Request.ArgumentEncoding {
		return retain(ReasonArgumentVectorDrift)
	}
	if probe.Request.WorkingDirectory != record.Request.WorkingDirectory {
		return retain(ReasonWorkingDirectoryDrift)
	}
	if probe.Request.RepositoryScopeSHA256 != record.Request.RepositoryScopeSHA256 || probe.Request.RepositoryIdentitySHA256 != record.Request.RepositoryIdentitySHA256 {
		return retain(ReasonRepositoryIdentityDrift)
	}
	if reason := compareExecution(record.Execution, probe.Execution); reason != ReasonNone {
		return retain(reason)
	}
	if reason := compareInputs(record.Inputs, probe.Inputs); reason != ReasonNone {
		return retain(reason)
	}
	if reason := validateTasks(probe.Tasks); reason != ReasonNone {
		return retain(reason)
	}
	if !reflect.DeepEqual(probe.Tasks, record.Tasks) {
		return retain(ReasonTaskImplementationDrift)
	}
	if reason := compareOutputs(record.Outputs, probe.Outputs); reason != ReasonNone {
		return retain(reason)
	}
	return Verdict{Disposition: DispositionContractComplete, RecordSHA256: digest}
}

func validateRecord(record SafetyRecord) Reason {
	if record.SchemaVersion != SchemaVersion || record.RecordType != RecordType || !digestPattern.MatchString(record.RecordID) || record.RevocationEpoch < 0 {
		return ReasonRecordIdentityInvalid
	}
	captured, capturedOK := parseTimestamp(record.CapturedAt)
	expires, expiresOK := parseTimestamp(record.ExpiresAt)
	if !capturedOK || !expiresOK || !expires.After(captured) {
		return ReasonRecordIdentityInvalid
	}
	if reason := validateRequest(record.Request); reason != ReasonNone {
		return reason
	}
	if reason := validateExecution(record.Execution); reason != ReasonNone {
		return reason
	}
	if reason := validateInputs(record.Inputs); reason != ReasonNone {
		return reason
	}
	if reason := validateOutputContract(record.Outputs); reason != ReasonNone {
		return reason
	}
	if reason := validateTasks(record.Tasks); reason != ReasonNone {
		return reason
	}
	switch record.PriorResult.Outcome {
	case "BUILD_FAILURE", "INFRA_FAILURE":
		return ReasonPriorFailure
	case "CANCELLED":
		return ReasonPriorCancellation
	case "SUCCESS":
		if record.PriorResult.ExitCode != 0 {
			return ReasonPriorOutcomeUnsafe
		}
	default:
		return ReasonPriorOutcomeUnsafe
	}
	if !record.PriorResult.OutputsVerified {
		return ReasonPriorOutputsUnverified
	}
	return ReasonNone
}

func validateRequest(value RequestBinding) Reason {
	if value.ArgumentEncoding != ArgumentEncoding || value.ArgumentCount < 1 || !digestPattern.MatchString(value.ArgumentsSHA256) {
		return ReasonArgumentVectorUnavailable
	}
	if !validRelativePath(value.WorkingDirectory, true) {
		return ReasonWorkingDirectoryInvalid
	}
	if !digestPattern.MatchString(value.RepositoryScopeSHA256) || !digestPattern.MatchString(value.RepositoryIdentitySHA256) {
		return ReasonRepositoryIdentityUnavailable
	}
	return ReasonNone
}

func validateExecution(value ExecutionBinding) Reason {
	if !digestPattern.MatchString(value.WrapperSHA256) || !digestPattern.MatchString(value.GradleDistributionSHA256) {
		return ReasonWrapperDrift
	}
	if !gradlePattern.MatchString(value.GradleVersion) {
		return ReasonGradleDrift
	}
	if value.JDKVendor == "" || value.JDKVersion == "" || !digestPattern.MatchString(value.JDKRuntimeSHA256) {
		return ReasonJDKDrift
	}
	if !digestPattern.MatchString(value.SafeEnvironmentSHA256) {
		return ReasonEnvironmentDrift
	}
	if !digestPattern.MatchString(value.RequestGraphSHA256) {
		return ReasonRequestGraphDrift
	}
	if !digestPattern.MatchString(value.TaskImplementationSHA256) {
		return ReasonTaskImplementationDrift
	}
	if !digestPattern.MatchString(value.BuildLogicSHA256) {
		return ReasonBuildLogicDrift
	}
	return ReasonNone
}

func compareExecution(record, probe ExecutionBinding) Reason {
	if reason := validateExecution(probe); reason != ReasonNone {
		return reason
	}
	if record.WrapperSHA256 != probe.WrapperSHA256 || record.GradleDistributionSHA256 != probe.GradleDistributionSHA256 {
		return ReasonWrapperDrift
	}
	if record.GradleVersion != probe.GradleVersion {
		return ReasonGradleDrift
	}
	if record.JDKVendor != probe.JDKVendor || record.JDKVersion != probe.JDKVersion || record.JDKRuntimeSHA256 != probe.JDKRuntimeSHA256 {
		return ReasonJDKDrift
	}
	if record.SafeEnvironmentSHA256 != probe.SafeEnvironmentSHA256 {
		return ReasonEnvironmentDrift
	}
	if record.RequestGraphSHA256 != probe.RequestGraphSHA256 {
		return ReasonRequestGraphDrift
	}
	if record.TaskImplementationSHA256 != probe.TaskImplementationSHA256 {
		return ReasonTaskImplementationDrift
	}
	if record.BuildLogicSHA256 != probe.BuildLogicSHA256 {
		return ReasonBuildLogicDrift
	}
	return ReasonNone
}

func validateInputs(value InputBinding) Reason {
	if !value.RepositoryInputsComplete || !digestPattern.MatchString(value.RepositoryManifestSHA256) {
		return ReasonRepositoryInputsIncomplete
	}
	if !value.ExternalInputsComplete {
		return ReasonExternalInputsIncomplete
	}
	if len(value.ExternalInputs) == 0 || !sortedExternalInputs(value.ExternalInputs) {
		return ReasonExternalInputsIncomplete
	}
	for _, input := range value.ExternalInputs {
		if input.Kind == "" || input.Identity == "" || !input.Present {
			return ReasonExternalInputMissing
		}
		if !digestPattern.MatchString(input.SHA256) {
			return ReasonExternalInputDrift
		}
	}
	return ReasonNone
}

func compareInputs(record, probe InputBinding) Reason {
	if reason := validateInputs(probe); reason != ReasonNone {
		return reason
	}
	if record.RepositoryManifestSHA256 != probe.RepositoryManifestSHA256 {
		return ReasonRepositoryInputDrift
	}
	if len(record.ExternalInputs) != len(probe.ExternalInputs) {
		return ReasonExternalInputMissing
	}
	for index := range record.ExternalInputs {
		if record.ExternalInputs[index].Kind != probe.ExternalInputs[index].Kind || record.ExternalInputs[index].Identity != probe.ExternalInputs[index].Identity || !probe.ExternalInputs[index].Present {
			return ReasonExternalInputMissing
		}
		if record.ExternalInputs[index].SHA256 != probe.ExternalInputs[index].SHA256 {
			return ReasonExternalInputDrift
		}
	}
	return ReasonNone
}

func validateOutputContract(value OutputContract) Reason {
	if !value.Complete || len(value.States) == 0 || !sortedOutputStates(value.States) {
		return ReasonOutputContractIncomplete
	}
	for _, output := range value.States {
		if !validRelativePath(output.Path, false) || (output.Kind != "FILE" && output.Kind != "DIRECTORY") || output.Size < 0 || output.Mode > 0o777 {
			return ReasonOutputContractIncomplete
		}
		if !output.Tracked {
			return ReasonUntrackedOutput
		}
		if output.Exists {
			if !digestPattern.MatchString(output.SHA256) || output.MaterializationRef != "sha256:"+output.SHA256 {
				return ReasonMaterializationInvalid
			}
		} else if output.SHA256 != "" || output.MaterializationRef != "" || output.Size != 0 || output.Mode != 0 {
			return ReasonOutputContractIncomplete
		}
	}
	return ReasonNone
}

func compareOutputs(contract OutputContract, observed []ObservedOutput) Reason {
	if len(observed) != len(contract.States) || !sortedObservedOutputs(observed) {
		return ReasonOutputContractDrift
	}
	for index, expected := range contract.States {
		current := observed[index]
		if current.Path != expected.Path {
			return ReasonOutputContractDrift
		}
		if !expected.Exists {
			if current.WorkspaceExists {
				return ReasonAbsentOutputPresent
			}
			if current.WorkspaceSHA256 != "" || current.MaterializationAvailable || current.MaterializationRef != "" || current.MaterializationSHA256 != "" {
				return ReasonOutputContractDrift
			}
			continue
		}
		if current.WorkspaceExists {
			if current.WorkspaceSHA256 != expected.SHA256 {
				return ReasonOutputAltered
			}
			continue
		}
		if !current.MaterializationAvailable {
			return ReasonOutputMissing
		}
		if current.MaterializationRef != expected.MaterializationRef {
			return ReasonMaterializationInvalid
		}
		if current.MaterializationSHA256 != expected.SHA256 {
			return ReasonOutputAltered
		}
	}
	return ReasonNone
}

func validateTasks(tasks []TaskSafety) Reason {
	if len(tasks) == 0 || !sortedTasks(tasks) {
		return ReasonTaskInventoryUnavailable
	}
	for _, task := range tasks {
		if !taskPattern.MatchString(task.Path) {
			return ReasonTaskInventoryUnavailable
		}
		if !task.Cacheable {
			return ReasonNonCacheableTask
		}
		if !task.Tracked {
			return ReasonUntrackedTask
		}
		if task.AlwaysRun {
			return ReasonAlwaysRunTask
		}
		if task.LocalState {
			return ReasonLocalState
		}
		if task.Destroyables {
			return ReasonDestroyables
		}
		if task.UntrackedWrites {
			return ReasonUntrackedWrite
		}
		if task.SideEffects {
			return ReasonSideEffectfulTask
		}
	}
	return ReasonNone
}

// MatchingProbe constructs a complete no-drift probe for conformance and
// later shadow fixtures. It grants no runtime authority.
func MatchingProbe(record SafetyRecord) Probe {
	probe := Probe{
		EvidenceComplete: true, CurrentRevocationEpoch: record.RevocationEpoch,
		Request: record.Request, Execution: record.Execution, Inputs: record.Inputs,
		Tasks: append([]TaskSafety(nil), record.Tasks...),
	}
	probe.Inputs.ExternalInputs = append([]ExternalInput(nil), record.Inputs.ExternalInputs...)
	for _, output := range record.Outputs.States {
		observed := ObservedOutput{Path: output.Path}
		if output.Exists {
			observed.MaterializationAvailable = true
			observed.MaterializationRef = output.MaterializationRef
			observed.MaterializationSHA256 = output.SHA256
		}
		probe.Outputs = append(probe.Outputs, observed)
	}
	return probe
}

func sortedExternalInputs(values []ExternalInput) bool {
	return sort.SliceIsSorted(values, func(i, j int) bool {
		return values[i].Kind+"\x00"+values[i].Identity < values[j].Kind+"\x00"+values[j].Identity
	})
}

func sortedOutputStates(values []OutputState) bool {
	return sort.SliceIsSorted(values, func(i, j int) bool { return values[i].Path < values[j].Path }) && uniqueOutputPaths(values)
}

func sortedObservedOutputs(values []ObservedOutput) bool {
	return sort.SliceIsSorted(values, func(i, j int) bool { return values[i].Path < values[j].Path }) && uniqueObservedPaths(values)
}

func sortedTasks(values []TaskSafety) bool {
	return sort.SliceIsSorted(values, func(i, j int) bool { return values[i].Path < values[j].Path }) && uniqueTaskPaths(values)
}

func uniqueOutputPaths(values []OutputState) bool {
	for index := 1; index < len(values); index++ {
		if values[index].Path == values[index-1].Path {
			return false
		}
	}
	return true
}

func uniqueObservedPaths(values []ObservedOutput) bool {
	for index := 1; index < len(values); index++ {
		if values[index].Path == values[index-1].Path {
			return false
		}
	}
	return true
}

func uniqueTaskPaths(values []TaskSafety) bool {
	for index := 1; index < len(values); index++ {
		if values[index].Path == values[index-1].Path {
			return false
		}
	}
	return true
}

func validRelativePath(value string, allowDot bool) bool {
	if value == "" || strings.Contains(value, "\\") || strings.HasPrefix(value, "/") || path.Clean(value) != value || strings.HasPrefix(value, "../") {
		return false
	}
	return allowDot || value != "."
}
