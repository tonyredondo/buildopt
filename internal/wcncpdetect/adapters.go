package wcncpdetect

// Catalog adapters re-express the current reviewed-native recipes as
// versioned WCNCP classifiers over fresh prospective evidence only.
// Historical candidate lists never seed the cohort; each adapter independently
// classifies fresh inputs and returns a typed DetectorRow.

// NormalizationInput carries fresh file-input normalization facts for one
// task. Names are labels; classification reads only typed normalization and
// ownership fields.
type NormalizationInput struct {
	TaskOwned            bool
	FileInputsDeclared   bool
	NormalizationDeclared bool
	NormalizationSound   bool
	PortableCacheable    bool
	SourceDrifted        bool
	BindingAmbiguous     bool
	RequiresOwnerSemantics bool
	CriticalPathMs       int64
	WorkflowMs           int64
	EnvironmentClass     string
	SourcePath           string
}

// NormalizationAwareCacheabilityV2 classifies one task for a portable
// owner-reviewed cacheability correction with explicit file-input
// normalization. Only exact, reversible, semantics-safe rows become actionable.
func NormalizationAwareCacheabilityV2(input NormalizationInput) DetectorRow {
	base := DetectorRow{DetectorID: "NORMALIZATION_AWARE_CACHEABILITY", DetectorVersion: "v2"}
	if input.BindingAmbiguous || input.SourceDrifted {
		base.Decision = "SOURCE_OR_BINDING_AMBIGUOUS"
		if input.SourceDrifted {
			base.Decision = "SOURCE_DRIFTED"
		}
		base.Reason = "exact binding required"
		return base
	}
	if input.RequiresOwnerSemantics {
		base.Decision = "OWNER_SEMANTICS_REQUIRED"
		base.Reason = "normalization semantics need an owner answer"
		return base
	}
	if !input.TaskOwned || !input.FileInputsDeclared || !input.NormalizationDeclared || !input.NormalizationSound || !input.PortableCacheable {
		base.Decision = "UNSAFE_OR_NON_REVERSIBLE"
		if !input.FileInputsDeclared || !input.NormalizationDeclared {
			base.Decision = "INCOMPLETE_OBSERVATION"
			base.Reason = "missing normalization declaration"
			base.RequiredDiagnostics = []string{"file-input-normalization-declaration"}
			return base
		}
		base.Reason = "sound portable normalization required"
		return base
	}
	materiality, _, percent := Materiality(input.EnvironmentClass, input.CriticalPathMs, input.WorkflowMs)
	if materiality == "REQUIRES_CONTROLLED_MEASUREMENT" {
		base.Decision = "INCOMPLETE_OBSERVATION"
		base.Reason = "materiality requires controlled measurement"
		base.RequiredDiagnostics = []string{"controlled-critical-path-measurement"}
		return base
	}
	if materiality == "FAILED" {
		base.Decision = "NON_MATERIAL_BLOCKER"
		base.Reason = "below 500 ms and 2 percent materiality"
		return base
	}
	base.Decision = "ACTIONABLE_MATERIAL_CORRECTION"
	base.SourcePath = input.SourcePath
	base.RecipeClass = "NORMALIZATION_AWARE_CACHEABILITY_V2"
	base.CriticalPathMs = input.CriticalPathMs
	base.WorkflowPercentMilli = percent
	return base
}

// TaskContractInput carries fresh durable task-contract facts: stable inputs,
// outputs, and incremental behavior for one owned task.
type TaskContractInput struct {
	TaskOwned              bool
	DeclaresInputsOutputs  bool
	IncrementalSafe        bool
	ContractStable         bool
	SourceDrifted          bool
	BindingAmbiguous       bool
	RequiresOwnerSemantics bool
	CriticalPathMs         int64
	WorkflowMs             int64
	EnvironmentClass       string
	SourcePath             string
}

// DurableTaskContractCurrent classifies one task for a durable contract
// correction. Behavior-changing or unstable contracts stay out.
func DurableTaskContractCurrent(input TaskContractInput) DetectorRow {
	base := DetectorRow{DetectorID: "DURABLE_TASK_CONTRACT", DetectorVersion: "current"}
	if input.BindingAmbiguous || input.SourceDrifted {
		base.Decision = "SOURCE_OR_BINDING_AMBIGUOUS"
		if input.SourceDrifted {
			base.Decision = "SOURCE_DRIFTED"
		}
		base.Reason = "exact binding required"
		return base
	}
	if input.RequiresOwnerSemantics {
		base.Decision = "OWNER_SEMANTICS_REQUIRED"
		base.Reason = "contract semantics need an owner answer"
		return base
	}
	if !input.TaskOwned || !input.DeclaresInputsOutputs || !input.IncrementalSafe || !input.ContractStable {
		base.Decision = "UNSAFE_OR_NON_REVERSIBLE"
		if !input.DeclaresInputsOutputs {
			base.Decision = "INCOMPLETE_OBSERVATION"
			base.Reason = "missing input/output declaration"
			base.RequiredDiagnostics = []string{"task-contract-declaration"}
			return base
		}
		base.Reason = "stable incremental contract required"
		return base
	}
	materiality, _, percent := Materiality(input.EnvironmentClass, input.CriticalPathMs, input.WorkflowMs)
	if materiality == "REQUIRES_CONTROLLED_MEASUREMENT" {
		base.Decision = "INCOMPLETE_OBSERVATION"
		base.Reason = "materiality requires controlled measurement"
		base.RequiredDiagnostics = []string{"controlled-critical-path-measurement"}
		return base
	}
	if materiality == "FAILED" {
		base.Decision = "NON_MATERIAL_BLOCKER"
		base.Reason = "below 500 ms and 2 percent materiality"
		return base
	}
	base.Decision = "ACTIONABLE_MATERIAL_CORRECTION"
	base.SourcePath = input.SourcePath
	base.RecipeClass = "DURABLE_TASK_CONTRACT_CURRENT"
	base.CriticalPathMs = input.CriticalPathMs
	base.WorkflowPercentMilli = percent
	return base
}
