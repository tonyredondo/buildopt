// Package durablecatalog turns bounded structural observations into
// review-only native Gradle optimization opportunities.
//
// The package deliberately separates detection, patch transaction proof and
// value evidence. A detector never authorizes a source change, and a patch
// that is useful in this POC must leave ordinary Gradle as the execution path
// after review.
package durablecatalog

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"sort"
	"strings"

	"github.com/tonyredondo/buildopt/internal/adaptivefragment"
)

const (
	SchemaVersion = "buildopt.poc/sticky-durable-catalog/v1"

	StatusProposed   = "PROPOSED"
	KindTaskContract = "TASK_CONTRACT"
	KindGraphBreadth = "GRAPH_BREADTH"

	TaskContractRecipe = "CUSTOM_TASK_CONTRACT_JAVA_V1"
	GraphBreadthRecipe = "DECLARED_GRAPH_SCOPE_V1"
)

// Measurement is a paired native Gradle comparison. BuildOptRequiredAfterAcceptance
// must be false for a durable patch to receive value credit.
type Measurement struct {
	Pairs                           uint64    `json:"pairs"`
	ControlMeanMs                   float64   `json:"controlMeanMs"`
	CandidateMeanMs                 float64   `json:"candidateMeanMs"`
	MeanSavedMs                     float64   `json:"meanSavedMs"`
	MeanReductionRatio              float64   `json:"meanReductionRatio"`
	Interval95SavedMs               []float64 `json:"interval95SavedMs"`
	PositivePairs                   uint64    `json:"positivePairs"`
	RequiredOutputsIdentical        bool      `json:"requiredOutputsIdentical"`
	ProductAttributableFailures     uint64    `json:"productAttributableFailures"`
	BuildOptRequiredAfterAcceptance bool      `json:"buildoptRequiredAfterAcceptance"`
}

// Qualifies reports whether the measurement is strong enough for this POC's
// durable-value gate. It intentionally requires a majority of positive pairs
// and a positive lower confidence bound instead of accepting a favourable
// mean alone.
func (m Measurement) Qualifies() bool {
	return m.Pairs >= 8 && m.ControlMeanMs > 0 && m.CandidateMeanMs >= 0 &&
		m.MeanSavedMs > 0 && m.MeanReductionRatio > 0 &&
		len(m.Interval95SavedMs) == 2 && m.Interval95SavedMs[0] > 0 &&
		m.PositivePairs > m.Pairs/2 && m.RequiredOutputsIdentical &&
		m.ProductAttributableFailures == 0 && !m.BuildOptRequiredAfterAcceptance
}

// RecipeBinding records the exact source transformation that a reviewer may
// inspect. A recipe remains non-authorizing even after its bytes pass proof.
type RecipeBinding struct {
	ID                       string `json:"id"`
	Version                  string `json:"version"`
	TargetPath               string `json:"targetPath"`
	Transformation           string `json:"transformation"`
	PreimageSHA256           string `json:"preimageSha256"`
	PostimageSHA256          string `json:"postimageSha256"`
	OwnerReviewRequired      bool   `json:"ownerReviewRequired"`
	ExactRevertRequired      bool   `json:"exactRevertRequired"`
	AutomaticMergeAuthorized bool   `json:"automaticMergeAuthorized"`
}

// PatchTransaction is the proof that a recipe can be applied and reverted in
// a temporary workspace without mutating the repository checkout.
type PatchTransaction struct {
	AppliedOutsideCheckout      bool   `json:"appliedOutsideCheckout"`
	CheckoutUnchanged           bool   `json:"checkoutUnchanged"`
	ExactRevertRestoredPreimage bool   `json:"exactRevertRestoredPreimage"`
	PreimageSHA256              string `json:"preimageSha256"`
	PostimageSHA256             string `json:"postimageSha256"`
	RejectedProposalMutations   uint64 `json:"rejectedProposalMutations"`
}

// ProvePatchTransaction applies and reverts bytes only inside an isolated
// temporary directory. The caller remains responsible for owner review.
func ProvePatchTransaction(preimage, postimage []byte) (PatchTransaction, error) {
	if len(preimage) == 0 || len(postimage) == 0 || bytes.Equal(preimage, postimage) {
		return PatchTransaction{}, errors.New("patch transaction requires distinct non-empty images")
	}
	preimageSHA, postimageSHA := DigestBytes(preimage), DigestBytes(postimage)
	root, err := os.MkdirTemp("", "buildopt-durable-catalog-transaction-")
	if err != nil {
		return PatchTransaction{}, err
	}
	defer os.RemoveAll(root)
	target := root + string(os.PathSeparator) + "target"
	if err := os.WriteFile(target, preimage, 0o600); err != nil {
		return PatchTransaction{}, err
	}
	if err := os.WriteFile(target, postimage, 0o600); err != nil {
		return PatchTransaction{}, err
	}
	applied, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(applied, postimage) {
		return PatchTransaction{}, errors.New("patch postimage was not applied exactly")
	}
	if err := os.WriteFile(target, preimage, 0o600); err != nil {
		return PatchTransaction{}, err
	}
	reverted, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(reverted, preimage) {
		return PatchTransaction{}, errors.New("patch preimage was not restored exactly")
	}
	return PatchTransaction{
		AppliedOutsideCheckout: true, CheckoutUnchanged: true,
		ExactRevertRestoredPreimage: true, PreimageSHA256: preimageSHA,
		PostimageSHA256: postimageSHA,
	}, nil
}

// GraphBreadthObservation records one requested build for a possible durable
// narrowing of a declared workflow. Counts and output digests are kept
// separate so the detector cannot confuse fewer projects with fewer required
// outputs.
type GraphBreadthObservation struct {
	RequestedBuildOrdinal uint64 `json:"requestedBuildOrdinal"`
	FullProjectCount      uint64 `json:"fullProjectCount"`
	CandidateProjectCount uint64 `json:"candidateProjectCount"`
	FullOutputSHA256      string `json:"fullOutputSha256"`
	CandidateOutputSHA256 string `json:"candidateOutputSha256"`
	ProductFailure        bool   `json:"productFailure"`
}

// GraphBreadthInput is intentionally repository-name agnostic. Scope and
// graph digests are provenance, not classification rules.
type GraphBreadthInput struct {
	EvidenceSHA256        string                    `json:"evidenceSha256"`
	RepositoryScopeSHA256 string                    `json:"repositoryScopeSha256"`
	ManifestSHA256        string                    `json:"manifestSha256"`
	GraphSHA256           string                    `json:"graphSha256"`
	Workflow              string                    `json:"workflow"`
	Observations          []GraphBreadthObservation `json:"observations"`
}

// GraphBreadthOpportunity is a review-only proposal. It carries no source
// bytes and cannot authorize selecting a reduced graph.
type GraphBreadthOpportunity struct {
	Status                          string `json:"status"`
	Kind                            string `json:"kind"`
	Workflow                        string `json:"workflow"`
	FullProjectCount                uint64 `json:"fullProjectCount"`
	CandidateProjectCount           uint64 `json:"candidateProjectCount"`
	OmittedProjectCount             uint64 `json:"omittedProjectCount"`
	RequestedBuilds                 uint64 `json:"requestedBuilds"`
	RecipeHint                      string `json:"recipeHint"`
	OwnerReviewRequired             bool   `json:"ownerReviewRequired"`
	TransactionalValidationRequired bool   `json:"transactionalValidationRequired"`
	ExactRevertRequired             bool   `json:"exactRevertRequired"`
	PatchAuthorized                 bool   `json:"patchAuthorized"`
	ActivationAuthorized            bool   `json:"activationAuthorized"`
}

// DetectGraphBreadthOpportunity finds a stable output-preserving reduction in
// a declared workflow. It does not inspect repository or task names.
func DetectGraphBreadthOpportunity(input GraphBreadthInput) (GraphBreadthOpportunity, error) {
	if !validSHA(input.EvidenceSHA256) || !validSHA(input.RepositoryScopeSHA256) ||
		!validSHA(input.ManifestSHA256) || !validSHA(input.GraphSHA256) ||
		!safeLabel(input.Workflow) || len(input.Observations) < 3 {
		return GraphBreadthOpportunity{}, errors.New("graph opportunity identity or evidence is invalid")
	}
	seen := map[uint64]bool{}
	var fullProjects, candidateProjects uint64
	var fullOutput, candidateOutput string
	for _, observation := range input.Observations {
		if observation.RequestedBuildOrdinal == 0 || seen[observation.RequestedBuildOrdinal] ||
			observation.FullProjectCount == 0 || observation.CandidateProjectCount == 0 ||
			observation.CandidateProjectCount >= observation.FullProjectCount ||
			!validSHA(observation.FullOutputSHA256) || !validSHA(observation.CandidateOutputSHA256) ||
			observation.FullOutputSHA256 != observation.CandidateOutputSHA256 || observation.ProductFailure {
			return GraphBreadthOpportunity{}, errors.New("graph opportunity observation is unsafe")
		}
		seen[observation.RequestedBuildOrdinal] = true
		if fullProjects == 0 {
			fullProjects, candidateProjects = observation.FullProjectCount, observation.CandidateProjectCount
			fullOutput, candidateOutput = observation.FullOutputSHA256, observation.CandidateOutputSHA256
		}
		if observation.FullProjectCount != fullProjects || observation.CandidateProjectCount != candidateProjects ||
			observation.FullOutputSHA256 != fullOutput || observation.CandidateOutputSHA256 != candidateOutput {
			return GraphBreadthOpportunity{}, errors.New("graph opportunity state is not stable")
		}
	}
	return GraphBreadthOpportunity{
		Status: StatusProposed, Kind: KindGraphBreadth, Workflow: input.Workflow,
		FullProjectCount: fullProjects, CandidateProjectCount: candidateProjects,
		OmittedProjectCount: fullProjects - candidateProjects,
		RequestedBuilds:     uint64(len(input.Observations)), RecipeHint: GraphBreadthRecipe,
		OwnerReviewRequired: true, TransactionalValidationRequired: true,
		ExactRevertRequired: true, PatchAuthorized: false, ActivationAuthorized: false,
	}, nil
}

// DigestBytes returns a lowercase SHA-256 digest without a prefix.
func DigestBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func validSHA(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if !(character >= '0' && character <= '9') && !(character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}

func safeLabel(value string) bool {
	return value != "" && len(value) <= 160 && !strings.ContainsRune(value, '\x00') &&
		strings.TrimSpace(value) == value
}

// SortObservations is a helper for report producers that need stable JSON.
func SortObservations(observations []GraphBreadthObservation) {
	sort.Slice(observations, func(left, right int) bool {
		return observations[left].RequestedBuildOrdinal < observations[right].RequestedBuildOrdinal
	})
}

// TaskProposal wraps the existing bounded Java detector while making its
// generic/no-authority boundary explicit to catalog consumers.
func TaskProposal(input adaptivefragment.PatchOpportunityInput) (adaptivefragment.PatchOpportunity, error) {
	proposal, err := adaptivefragment.DetectTaskContractOpportunity(input)
	if err != nil {
		return adaptivefragment.PatchOpportunity{}, err
	}
	if proposal.Status != adaptivefragment.PatchOpportunityStatusProposed ||
		proposal.Kind != adaptivefragment.PatchOpportunityKindTaskContract ||
		proposal.PatchAuthorized || proposal.ActivationAuthorized {
		return adaptivefragment.PatchOpportunity{}, errors.New("task detector transferred authority")
	}
	return proposal, nil
}
