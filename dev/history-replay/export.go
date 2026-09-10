//go:build linux && amd64

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type AttemptLedgerEntry struct {
	Schema   string       `json:"schema"`
	Start    AttemptStart `json:"start"`
	Native   Binding      `json:"native"`
	End      Binding      `json:"end"`
	Evidence Binding      `json:"evidence"`
	Status   string       `json:"status"`
}

func exportBytes(root string, result RunResult) (map[string][]byte, error) {
	out := map[string][]byte{}
	var attempts, phases bytes.Buffer
	paths, err := filepath.Glob(filepath.Join(root, "attempts", "*", "start.json"))
	if err != nil {
		return nil, err
	}
	rows := []AttemptLedgerEntry{}
	for _, path := range paths {
		var a AttemptStart
		if err = readJSON(path, &a); err != nil {
			return nil, err
		}
		dir := filepath.Dir(path)
		row := AttemptLedgerEntry{Schema: recordSchema, Start: a, Status: "PARTIAL"}
		row.Evidence, err = bind(dir)
		if err != nil {
			return nil, err
		}
		for _, item := range []struct {
			name   string
			target *Binding
		}{{"native-finish.json", &row.Native}, {"end.json", &row.End}} {
			b, e := bind(filepath.Join(dir, item.name))
			if e != nil && !os.IsNotExist(e) {
				return nil, e
			}
			if e == nil {
				*item.target = b
			}
		}
		var end AttemptEnd
		if err = readJSON(filepath.Join(dir, "end.json"), &end); err == nil {
			row.Status = end.Class
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Start.At.NS < rows[j].Start.At.NS })
	for _, row := range rows {
		if err = json.NewEncoder(&attempts).Encode(row); err != nil {
			return nil, err
		}
	}
	costs, err := loadCosts(root)
	if err != nil {
		return nil, err
	}
	sort.Slice(costs, func(i, j int) bool { return costs[i].StartNS < costs[j].StartNS })
	for _, cost := range costs {
		if err = json.NewEncoder(&phases).Encode(cost); err != nil {
			return nil, err
		}
	}
	out["attempts.jsonl"] = attempts.Bytes()
	out["costs.jsonl"] = phases.Bytes()
	var manifest Manifest
	if err = readJSON(filepath.Join(root, "manifest.json"), &manifest); err != nil {
		return nil, err
	}
	if !absentBinding(manifest.Observer) {
		var observations bytes.Buffer
		for _, attempt := range rows {
			path := filepath.Join(root, "attempts", attempt.Start.ID, "observation.json")
			var receipt ObservationReceipt
			status := "COMPLETE"
			if err := readJSON(path, &receipt); err != nil {
				if !os.IsNotExist(err) {
					return nil, err
				}
				status = "INCOMPLETE"
			} else if receipt.Error != "" || receipt.ExitCode != 0 {
				status = "FAILED"
			}
			row := struct {
				Attempt string              `json:"attempt"`
				Status  string              `json:"status"`
				Receipt *ObservationReceipt `json:"receipt"`
			}{Attempt: attempt.Start.ID, Status: status}
			if status != "INCOMPLETE" {
				row.Receipt = &receipt
			}
			if err := json.NewEncoder(&observations).Encode(row); err != nil {
				return nil, err
			}
		}
		out["observations.jsonl"] = observations.Bytes()
	}
	extra, err := externalCosts(manifest)
	if err != nil {
		return nil, err
	}
	for _, cost := range extra {
		if err = json.NewEncoder(&phases).Encode(cost); err != nil {
			return nil, err
		}
	}
	out["costs.jsonl"] = phases.Bytes()
	if !absentBinding(manifest.Subjects) {
		if err = checkBinding(manifest.Subjects); err != nil {
			return nil, err
		}
		out["subjects.json"], err = os.ReadFile(manifest.Subjects.Path)
		if err != nil {
			return nil, err
		}
	} else {
		out["subjects.json"] = jsonBytes(struct{ Schema, Subject, Phase, HistorySHA256 string }{recordSchema, manifest.Subject, manifest.Phase, objectDigest(manifest.History)})
	}
	var report bytes.Buffer
	fmt.Fprintf(&report, "# Historical replay result\n\nDecision: `%s`. Phase: `%s`. Subject: `%s`.\n\n", result.Decision, manifest.Phase, manifest.Subject)
	fmt.Fprintf(&report, "Workflows started: %d. Gradle invocations observed: %d, including %d nested. Reserved Gradle starts: %d; unresolved reservations: %d.\n\n", result.WorkflowStarts, result.ActualGradleStarts, result.NestedStarts, result.GradleReservations, result.UnknownGradleReservations)
	fmt.Fprintln(&report, "| Replication | Complete | Comparable / scheduled | Net seconds | Net percent | Payback ordinal | Research seconds | Study elapsed seconds |\n|---|---|---|---|---|---|---|---|")
	for _, e := range result.Replications {
		fmt.Fprintf(&report, "| %d | %t | %d / %d | %.6f | %.3f | %d | %.6f | %.6f |\n", e.Replication, e.Complete, e.Comparable, e.Scheduled, float64(e.NetSavedNS)/1e9, e.NetFraction*100, e.PaybackOrdinal, float64(e.ResearchNS)/1e9, float64(e.StudyMachineNS)/1e9)
	}
	fmt.Fprintln(&report, "\nPrimary requests include candidate admission/application. Research time includes source checkout, evidence capture, synchronization and gaps inside each replication. The final offline checker/export runs after that interval; its command receipt is separate research evidence. Human costs must be reported separately. Qualification and engineering results cannot establish public-repository value.\n\nRaw ledger: [attempts](attempts.jsonl), [costs](costs.jsonl), [manifest](manifest.json), [result](result.json).")
	out["report.md"] = report.Bytes()
	return out, nil
}

func checkExports(root string, result RunResult) error {
	expected, err := exportBytes(root, result)
	if err != nil {
		return err
	}
	for name, bytes := range expected {
		actual, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return err
		}
		if digest(actual) != digest(bytes) {
			return errors.New("exported ledger/report differs from raw evidence: " + name)
		}
	}
	return nil
}

func writeExports(root, version string, result RunResult) error {
	outputs, err := exportBytes(root, result)
	if err != nil {
		return err
	}
	dir := filepath.Join(root, "exports", version)
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	names := []string{}
	for name := range outputs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err = writeExclusive(filepath.Join(dir, name), outputs[name], 0600); err != nil {
			return err
		}
		if err = atomicBytes(filepath.Join(root, name), outputs[name]); err != nil {
			return err
		}
	}
	return nil
}
