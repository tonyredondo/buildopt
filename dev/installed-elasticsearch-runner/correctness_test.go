package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

// Losing the cache transfer or restoring the input from the wrong prior bytes
// would make the apparent cross-root/cache-invalidation proof meaningless.
func TestCorrectnessPlanPreservesStateAndNestedStarts(t *testing.T) {
	var out bytes.Buffer
	if err := runCLI([]string{"correctness-plan"}, &out); err != nil {
		t.Fatal(err)
	}
	var plan struct {
		Steps []struct {
			Row          row    `json:"row"`
			InputBefore  string `json:"inputBeforeSha256"`
			InputAfter   string `json:"inputAfterSha256"`
			RemoveMarker bool   `json:"removeMarker"`
			CacheFrom    string `json:"cacheFrom"`
			CompareWith  string `json:"compareWith"`
		} `json:"steps"`
		Outer  int  `json:"outerStarts"`
		Nested int  `json:"nestedStarts"`
		Value  bool `json:"valueAdmitted"`
	}
	if err := json.Unmarshal(out.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.Steps) != 40 || plan.Outer != 40 || plan.Nested != 12 || plan.Value {
		t.Fatalf("wrong complete correctness allocation: %+v", plan)
	}
	if !plan.Steps[3].RemoveMarker || plan.Steps[4].CacheFrom != "N1" || plan.Steps[4].Row.Arm != "W1" || plan.Steps[4].CompareWith != "C003" {
		t.Fatal("same-root or cross-root cache restoration missing")
	}
	if plan.Steps[8].InputBefore != "1a35aa9d75c9e922c783ce3bde85e700828440d4ccd0286944575547091269c0" || plan.Steps[8].InputAfter != "9ca0190189574e0aeac1aa08111bf5c20e5f2c5246d02af99a91205ab6ba70f1" || !plan.Steps[8].Row.ExpectedFailure {
		t.Fatal("negative probe did not replace the benign input")
	}
	if plan.Steps[10].InputBefore != "9ca0190189574e0aeac1aa08111bf5c20e5f2c5246d02af99a91205ab6ba70f1" || plan.Steps[10].InputAfter != "09c15178d0ebecb3da1d2efccd73d2e40689d060409c196a7ec2ca2f18c4d886" {
		t.Fatal("revert does not restore exact original input")
	}
	if plan.Steps[38].Row.NestedStarts != 6 || plan.Steps[39].Row.NestedStarts != 6 {
		t.Fatal("functional TestKit requests are not charged")
	}
	for _, s := range plan.Steps {
		if s.Row.Block != "C" && s.Row.Block != "M" && s.Row.Block != "T" {
			t.Fatal("value or reserve request admitted")
		}
	}
}
