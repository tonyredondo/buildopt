package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestCLIRejectsUngatedExecutionAndBudgetReset(t *testing.T) {
	for _, args := range [][]string{nil, {"capture"}, {"run"}, {"fixture", "--root", "relative"}, {"budget", "--charged-seconds", "0"}, {"plan", "extra"}, {"plan", "--unknown"}, {"native-prepare", "--slot", "D001"}, {"native-diagnose", "--slot", "P001"}, {"native-diagnose", "--slot", "V001"}} {
		var out bytes.Buffer
		if err := runCLI(args, &out); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	var out bytes.Buffer
	if err := runCLI([]string{"plan"}, &out); err != nil {
		t.Fatal(err)
	}
	var p protocol
	if err := json.Unmarshal(out.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if err := validateProtocol(p); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := runCLI([]string{"budget", "--charged-seconds", "4440"}, &out); err != nil {
		t.Fatal(err)
	}
	var b budgetReport
	if err := json.Unmarshal(out.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	if b.Decision != "UNPROVED" || b.PublicExecutionAuthorized {
		t.Fatalf("%+v", b)
	}
}
