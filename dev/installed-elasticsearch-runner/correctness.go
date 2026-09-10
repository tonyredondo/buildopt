package main

// correctnessStep describes the state owned by one C/M/T request. Performance
// rows have a different state model and cannot be started by this allocation.
type correctnessStep struct {
	Row          row    `json:"row"`
	InputBefore  string `json:"inputBeforeSha256"`
	InputAfter   string `json:"inputAfterSha256"`
	RemoveMarker bool   `json:"removeMarker"`
	CacheFrom    string `json:"cacheFrom"`
	CompareWith  string `json:"compareWith"`
	TaskOutcome  string `json:"taskOutcome"`
}

type correctnessPlan struct {
	Schema        string            `json:"schemaVersion"`
	Steps         []correctnessStep `json:"steps"`
	OuterStarts   int               `json:"outerStarts"`
	NestedStarts  int               `json:"nestedStarts"`
	ValueAdmitted bool              `json:"valueAdmitted"`
}

func makeCorrectnessPlan() correctnessPlan {
	p := correctnessPlan{Schema: "buildopt.eic/correctness-plan/v1", Steps: []correctnessStep{}}
	inputs := sourceInputs()
	current := map[string]string{"N0": inputs.OwnerInputSHA256, "N1": inputs.OwnerInputSHA256, "W1": inputs.OwnerInputSHA256}
	comparisons := map[string]string{"C002": "C001", "C003": "C001", "C004": "C003", "C005": "C003", "C006": "C005", "C008": "C007", "C010": "C009", "C011": "C001", "C012": "C011"}
	outcomes := map[string]string{"C001": "EXECUTED", "C002": "UP-TO-DATE", "C003": "EXECUTED", "C004": "FROM-CACHE", "C005": "FROM-CACHE", "C006": "UP-TO-DATE", "C007": "EXECUTED", "C008": "EXECUTED", "C009": "FAILED", "C010": "FAILED", "C011": "FROM-CACHE", "C012": "EXECUTED"}
	for _, r := range makeProtocol().Rows {
		if r.Block != "C" && r.Block != "M" && r.Block != "T" {
			continue
		}
		step := correctnessStep{Row: r, CompareWith: comparisons[r.ID], TaskOutcome: outcomes[r.ID]}
		if r.Block != "M" {
			step.InputBefore = current[r.Arm]
			step.InputAfter = step.InputBefore
			switch r.State {
			case "restore-same-root":
				step.RemoveMarker = true
			case "restore-cross-root":
				step.CacheFrom = "N1"
			case "benign-comment":
				step.InputAfter = inputs.BenignSHA256
			case "tab-violation":
				step.InputAfter = inputs.NegativeSHA256
			case "revert":
				step.InputAfter = inputs.OwnerInputSHA256
			}
			current[r.Arm] = step.InputAfter
		}
		p.Steps = append(p.Steps, step)
		p.OuterStarts++
		p.NestedStarts += r.NestedStarts
	}
	return p
}
