package buildimpact

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
)

const (
	POCCandidatePlanSchemaVersion = "buildopt.build-impact/poc-candidate-plan/v1"
	POCCandidateMode              = "POC_CANDIDATE"
	maximumGeneratedManifestBytes = 256 << 10
)

var gradleVersionPattern = regexp.MustCompile(`^[0-9][0-9A-Za-z.+_-]{0,63}$`)

// POCCandidateOptions names the reviewed repository state used by one explicit
// owner-operated candidate invocation. It does not represent production
// promotion authority.
type POCCandidateOptions struct {
	RepositoryRoot        string
	ManifestPath          string
	GraphPath             string
	GeneratedManifestPath string
	RepositoryID          string
	PipelineClass         string
	ChangedPaths          []string
	LocalBypass           bool
}

// POCCandidatePlan is an executable POC decision. CandidateSelected can only
// choose an alternative already owned by the repository manifest; every
// ambiguous or incomplete input retains the original full graph.
type POCCandidatePlan struct {
	SchemaVersion         string   `json:"schemaVersion"`
	Mode                  string   `json:"mode"`
	Reason                string   `json:"reason"`
	Entrypoints           []string `json:"entrypoints"`
	AlternativeID         string   `json:"alternativeId,omitempty"`
	AffectedProjects      []string `json:"affectedProjects"`
	OmittedProjects       []string `json:"omittedProjects"`
	PreservedTestCheckIDs []string `json:"preservedTestCheckIds"`
	CandidateSelected     bool     `json:"candidateSelected"`
	ProductionAuthorized  bool     `json:"productionAuthorized"`
}

// PlanPOCCandidate validates checked-in reviewable state and evaluates one
// explicit changed-path set. It deliberately does not call or weaken BIA-002:
// this owner-operated POC command measures a candidate, never promotes it.
func PlanPOCCandidate(options POCCandidateOptions) (POCCandidatePlan, error) {
	manifest, err := LoadRepositoryManifest(
		options.RepositoryRoot,
		options.ManifestPath,
		options.RepositoryID,
		options.PipelineClass,
	)
	if err != nil {
		return POCCandidatePlan{}, err
	}
	plan := POCCandidatePlan{
		SchemaVersion:        POCCandidatePlanSchemaVersion,
		Mode:                 DecisionFullGraph,
		Reason:               "GRAPH_INVALID",
		Entrypoints:          cloneSlice(manifest.Manifest.OriginalEntrypoints),
		ProductionAuthorized: false,
	}
	_, plan.PreservedTestCheckIDs = manifestCheckIDs(manifest.Manifest)
	if options.LocalBypass {
		plan.Reason = "LOCAL_BYPASS"
		return plan, nil
	}

	graphRaw, err := readRepositoryFile(options.RepositoryRoot, options.GraphPath)
	if err != nil {
		return POCCandidatePlan{}, err
	}
	graph, err := ParseDeclaredGraph(graphRaw, manifest)
	if err != nil {
		return POCCandidatePlan{}, err
	}
	generatedRaw, err := readRepositoryFile(options.RepositoryRoot, options.GeneratedManifestPath)
	if err != nil {
		return POCCandidatePlan{}, err
	}
	generated, err := parseGeneratedManifest(generatedRaw)
	if err != nil {
		return POCCandidatePlan{}, err
	}
	if err := validateGeneratedManifest(generated, manifest, graph); err != nil {
		return POCCandidatePlan{}, err
	}

	decision := evaluatePOCImpact(manifest, graph.Graph, options.ChangedPaths)
	plan.Reason = "IMPACT_" + decision.Reason
	plan.AffectedProjects = cloneSlice(decision.AffectedProjects)
	plan.OmittedProjects = cloneSlice(decision.OmittedProjects)
	plan.PreservedTestCheckIDs = cloneSlice(decision.PreservedTestCheckIDs)
	if decision.Mode != DecisionShadowAlternative {
		return plan, nil
	}
	plan.Mode = POCCandidateMode
	plan.Reason = "EXPLICIT_OWNER_POC_CANDIDATE"
	plan.Entrypoints = cloneSlice(decision.PredictedEntrypoints)
	plan.AlternativeID = decision.PredictedAlternativeID
	plan.CandidateSelected = true
	return plan, nil
}

func parseGeneratedManifest(raw []byte) (GeneratedManifest, error) {
	if len(raw) == 0 || len(raw) > maximumGeneratedManifestBytes {
		return GeneratedManifest{}, errors.New("generated Build Impact manifest size is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var generated GeneratedManifest
	if err := decoder.Decode(&generated); err != nil {
		return GeneratedManifest{}, fmt.Errorf("decode generated Build Impact manifest: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return GeneratedManifest{}, errors.New("generated Build Impact manifest has trailing content")
	}
	return generated, nil
}

func validateGeneratedManifest(generated GeneratedManifest, manifest LoadedManifest, graph LoadedGraph) error {
	if generated.SchemaVersion != GeneratedManifestSchemaVersion ||
		generated.RepositoryID != manifest.Manifest.RepositoryID ||
		generated.PipelineClass != manifest.Manifest.PipelineClass ||
		generated.ManifestDigest != manifest.Digest ||
		generated.GraphDigest != graph.Digest ||
		generated.AdapterVersion != graph.Graph.AdapterVersion ||
		!sha256Pattern.MatchString(generated.DiscoveryDigest) ||
		!gradleVersionPattern.MatchString(generated.GradleVersion) {
		return errors.New("generated Build Impact manifest binding is invalid")
	}
	if !generated.Complete || len(generated.FallbackReasons) != 0 {
		return errors.New("generated Build Impact state is incomplete")
	}
	return nil
}
