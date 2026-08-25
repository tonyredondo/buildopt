package adaptivefragment

import (
	"errors"
	"path"
	"sort"
	"strings"
)

const (
	PatchOpportunityStatusProposed = "PROPOSED"
	PatchOpportunityKindTaskContract = "TASK_CONTRACT_PATCH"
)

// TaskContractObservation is ordinary-build evidence for one task execution.
// It contains no repository name and grants no permission to change source.
type TaskContractObservation struct {
	RequestedBuildOrdinal uint64 `json:"requestedBuildOrdinal"`
	DurationMs            uint64 `json:"durationMs"`
	Executed              bool   `json:"executed"`
	Cacheable             bool   `json:"cacheable"`
	UpToDate              bool   `json:"upToDate"`
	InputSnapshotSHA256   string `json:"inputSnapshotSha256"`
	OutputSnapshotSHA256  string `json:"outputSnapshotSha256"`
	ProductFailure        bool   `json:"productFailure"`
}

// JavaTaskContractFacts are review facts extracted from a bounded Java custom
// task. They identify a possible missing Gradle contract, not its semantics.
type JavaTaskContractFacts struct {
	ExtendsDefaultTask   bool   `json:"extendsDefaultTask"`
	InternalInputCount  uint64 `json:"internalInputCount"`
	InternalOutputCount uint64 `json:"internalOutputCount"`
	TaskActionCount     uint64 `json:"taskActionCount"`
	UnknownSideEffects  bool   `json:"unknownSideEffects"`
}

// PatchOpportunityInput combines generic runtime and source-shape evidence.
// RepositoryScopeSHA256 is provenance only and does not affect detection.
type PatchOpportunityInput struct {
	EvidenceSHA256        string                    `json:"evidenceSha256"`
	RepositoryScopeSHA256 string                    `json:"repositoryScopeSha256"`
	TaskImplementationSHA256 string                 `json:"taskImplementationSha256"`
	RelativeSourcePath    string                    `json:"relativeSourcePath"`
	SourcePreimageSHA256  string                    `json:"sourcePreimageSha256"`
	Facts                 JavaTaskContractFacts     `json:"facts"`
	Observations          []TaskContractObservation `json:"observations"`
}

// PatchOpportunity is a reviewable hint. The explicit false authority fields
// prevent a detector result from being mistaken for an accepted patch.
type PatchOpportunity struct {
	Status                 string   `json:"status"`
	Kind                   string   `json:"kind"`
	RelativeSourcePath     string   `json:"relativeSourcePath"`
	SourcePreimageSHA256   string   `json:"sourcePreimageSha256"`
	MedianRepeatedCostMs   uint64   `json:"medianRepeatedCostMs"`
	RequestedBuilds        uint64   `json:"requestedBuilds"`
	SuggestedAnnotations   []string `json:"suggestedAnnotations"`
	RecipeHint             string   `json:"recipeHint"`
	OwnerReviewRequired    bool     `json:"ownerReviewRequired"`
	TransactionalValidationRequired bool `json:"transactionalValidationRequired"`
	ExactRevertRequired    bool     `json:"exactRevertRequired"`
	PatchAuthorized        bool     `json:"patchAuthorized"`
	ActivationAuthorized   bool     `json:"activationAuthorized"`
}

// DetectTaskContractOpportunity identifies a recurring expensive task whose
// observable shape is consistent with a missing input/output/cache contract.
// It intentionally does not infer whether the proposed annotations are true;
// owner review and transactional native Gradle validation remain mandatory.
func DetectTaskContractOpportunity(input PatchOpportunityInput) (PatchOpportunity, error) {
	if !validSHA(input.EvidenceSHA256) || !validSHA(input.RepositoryScopeSHA256) ||
		!validSHA(input.TaskImplementationSHA256) || !validSHA(input.SourcePreimageSHA256) ||
		!safeJavaSourcePath(input.RelativeSourcePath) {
		return PatchOpportunity{}, errors.New("patch opportunity identity is invalid")
	}
	if !input.Facts.ExtendsDefaultTask || input.Facts.InternalInputCount == 0 ||
		input.Facts.InternalOutputCount == 0 || input.Facts.TaskActionCount != 1 ||
		input.Facts.UnknownSideEffects {
		return PatchOpportunity{}, errors.New("patch opportunity source shape requires review before proposal")
	}
	if len(input.Observations) < 3 {
		return PatchOpportunity{}, errors.New("patch opportunity requires three requested builds")
	}
	durations := make([]uint64, 0, len(input.Observations))
	seenOrdinals := map[uint64]bool{}
	var inputDigest, outputDigest string
	for _, observation := range input.Observations {
		if observation.RequestedBuildOrdinal == 0 || seenOrdinals[observation.RequestedBuildOrdinal] ||
			observation.DurationMs == 0 || !observation.Executed || observation.Cacheable || observation.UpToDate ||
			observation.ProductFailure || !validSHA(observation.InputSnapshotSHA256) ||
			!validSHA(observation.OutputSnapshotSHA256) {
			return PatchOpportunity{}, errors.New("patch opportunity observation is unsafe")
		}
		seenOrdinals[observation.RequestedBuildOrdinal] = true
		if inputDigest == "" {
			inputDigest, outputDigest = observation.InputSnapshotSHA256, observation.OutputSnapshotSHA256
		}
		if observation.InputSnapshotSHA256 != inputDigest || observation.OutputSnapshotSHA256 != outputDigest {
			return PatchOpportunity{}, errors.New("patch opportunity repeated state is not stable")
		}
		durations = append(durations, observation.DurationMs)
	}
	sort.Slice(durations, func(left, right int) bool { return durations[left] < durations[right] })
	median := durations[len(durations)/2]
	if median < 500 {
		return PatchOpportunity{}, errors.New("patch opportunity repeated cost is below the POC floor")
	}
	return PatchOpportunity{
		Status: PatchOpportunityStatusProposed, Kind: PatchOpportunityKindTaskContract,
		RelativeSourcePath: input.RelativeSourcePath, SourcePreimageSHA256: input.SourcePreimageSHA256,
		MedianRepeatedCostMs: median, RequestedBuilds: uint64(len(input.Observations)),
		SuggestedAnnotations: []string{"CacheableTask", "Input", "OutputFile"},
		RecipeHint: "CUSTOM_TASK_CONTRACT_JAVA_V1", OwnerReviewRequired: true,
		TransactionalValidationRequired: true, ExactRevertRequired: true,
	}, nil
}

func safeJavaSourcePath(value string) bool {
	return value != "" && strings.HasSuffix(value, ".java") && !strings.HasPrefix(value, "/") &&
		path.Clean(value) == value && value != "." && !strings.HasPrefix(value, "../") &&
		!strings.HasPrefix(value, ".git/") && !strings.ContainsRune(value, '\x00')
}
