package buildimpact

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

const (
	SelectionPlanSchemaVersion   = "buildopt.build-impact/selection-plan/v1"
	SelectionFullGraph           = "FULL_GRAPH"
	SelectionCustomerAlternative = "CUSTOMER_ALTERNATIVE"
)

type SelectionControls struct {
	Enabled          bool
	KillSwitchActive bool
	LocalBypass      bool
}

type SelectionPlan struct {
	SchemaVersion         string   `json:"schemaVersion"`
	Mode                  string   `json:"mode"`
	Reason                string   `json:"reason"`
	Entrypoints           []string `json:"entrypoints"`
	AlternativeID         string   `json:"alternativeId,omitempty"`
	AffectedProjects      []string `json:"affectedProjects"`
	OmittedProjects       []string `json:"omittedProjects"`
	PreservedTestCheckIDs []string `json:"preservedTestCheckIds"`
	PromotionState        string   `json:"promotionState,omitempty"`
	SelectionAuthorized   bool     `json:"selectionAuthorized"`
}

// PlanSelection is the only active Build Impact selection boundary. It
// reevaluates BIA-002 from bound observations instead of trusting a caller-
// constructed report and returns the original customer entrypoints on every
// disabled, invalid, unknown, global, drifted, or unqualified path.
func PlanSelection(
	manifest LoadedManifest,
	graph LoadedGraph,
	changedPaths []string,
	promotion PromotionInput,
	controls SelectionControls,
) SelectionPlan {
	plan := SelectionPlan{
		SchemaVersion:       SelectionPlanSchemaVersion,
		Mode:                SelectionFullGraph,
		Reason:              "MANIFEST_INVALID",
		SelectionAuthorized: false,
	}
	if !validLoadedManifest(manifest) {
		return plan
	}
	plan.Entrypoints = append([]string(nil), manifest.Manifest.OriginalEntrypoints...)
	_, plan.PreservedTestCheckIDs = manifestCheckIDs(manifest.Manifest)
	if !validLoadedGraph(manifest, graph) {
		plan.Reason = "GRAPH_INVALID"
		return plan
	}
	if controls.LocalBypass {
		plan.Reason = "LOCAL_BYPASS"
		return plan
	}
	if controls.KillSwitchActive {
		plan.Reason = "KILL_SWITCH_ACTIVE"
		return plan
	}
	if !controls.Enabled {
		plan.Reason = "SELECTION_DISABLED"
		return plan
	}
	if promotion.RepositoryID != manifest.Manifest.RepositoryID ||
		promotion.PipelineClass != manifest.Manifest.PipelineClass ||
		promotion.ManifestDigest != manifest.Digest ||
		promotion.GraphDigest != graph.Digest ||
		promotion.AdapterVersion != graph.Graph.AdapterVersion {
		plan.Reason = "PROMOTION_BINDING_MISMATCH"
		return plan
	}
	report := EvaluatePromotion(promotion)
	plan.PromotionState = report.State
	if report.State != PromotionQualified {
		plan.Reason = "PROMOTION_" + report.Reason
		return plan
	}
	decision := EvaluateImpact(manifest, graph.Graph, changedPaths)
	plan.AffectedProjects = append([]string(nil), decision.AffectedProjects...)
	plan.OmittedProjects = append([]string(nil), decision.OmittedProjects...)
	plan.PreservedTestCheckIDs = append([]string(nil), decision.PreservedTestCheckIDs...)
	if decision.Mode != DecisionShadowAlternative {
		plan.Reason = "IMPACT_" + decision.Reason
		return plan
	}
	plan.Mode = SelectionCustomerAlternative
	plan.Reason = "PROMOTED_CUSTOMER_ALTERNATIVE"
	plan.Entrypoints = append([]string(nil), decision.PredictedEntrypoints...)
	plan.AlternativeID = decision.PredictedAlternativeID
	plan.SelectionAuthorized = true
	return plan
}

func validLoadedManifest(manifest LoadedManifest) bool {
	if err := validateManifest(
		manifest.Manifest,
		manifest.Manifest.RepositoryID,
		manifest.Manifest.PipelineClass,
	); err != nil {
		return false
	}
	digest, err := canonicalBuildImpactDigest(manifest.Manifest)
	return err == nil && manifest.Digest == digest
}

func validLoadedGraph(manifest LoadedManifest, graph LoadedGraph) bool {
	if err := validateDeclaredGraph(manifest, graph.Graph); err != nil {
		return false
	}
	digest, err := canonicalBuildImpactDigest(graph.Graph)
	return err == nil && graph.Digest == digest
}

func canonicalBuildImpactDigest(value any) (string, error) {
	canonical, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}
