//go:build linux && amd64

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type RunBoundary struct {
	Schema      string `json:"schema"`
	Replication int    `json:"replication"`
	LastOrdinal int    `json:"lastOrdinal"`
	Status      string `json:"status"`
	Reason      string `json:"reason"`
	At          Stamp  `json:"at"`
}

func beginReplication(root string, rep int) error {
	path := filepath.Join(root, "lifecycle", fmt.Sprintf("r%d-start.json", rep))
	var existing Stamp
	if err := readJSON(path, &existing); err == nil {
		if existing.Boot != stamp().Boot {
			return errors.New("replication crosses boot identity")
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	return writeExclusive(path, jsonBytes(stamp()), 0600)
}

func endReplication(root string, rep, ordinal int, status, reason string) error {
	files, err := filepath.Glob(filepath.Join(root, "lifecycle", fmt.Sprintf("r%d-end-*.json", rep)))
	if err != nil {
		return err
	}
	return writeExclusive(filepath.Join(root, "lifecycle", fmt.Sprintf("r%d-end-%04d.json", rep, len(files)+1)), jsonBytes(RunBoundary{recordSchema, rep, ordinal, status, reason, stamp()}), 0600)
}

func latestBoundary(root string, rep int) (Stamp, RunBoundary, error) {
	var start Stamp
	var end RunBoundary
	if err := readJSON(filepath.Join(root, "lifecycle", fmt.Sprintf("r%d-start.json", rep)), &start); err != nil {
		return start, end, err
	}
	files, err := filepath.Glob(filepath.Join(root, "lifecycle", fmt.Sprintf("r%d-end-*.json", rep)))
	if err != nil {
		return start, end, err
	}
	sort.Strings(files)
	if len(files) == 0 {
		return start, end, errors.New("replication lacks terminal/paused boundary")
	}
	if err = readJSON(files[len(files)-1], &end); err != nil {
		return start, end, err
	}
	if end.Schema != recordSchema || end.Replication != rep {
		return start, end, errors.New("invalid replication boundary")
	}
	if err = checkStampInterval(start, end.At); err != nil {
		return start, end, err
	}
	return start, end, nil
}

func applyResearchAccounting(m Manifest, rep int, e *Economics, costs []Cost) error {
	start, end, err := latestBoundary(m.RunRoot, rep)
	if os.IsNotExist(err) {
		e.Complete = false
		e.Gate = "INCOMPLETE_EVIDENCE"
		e.Reasons = append(e.Reasons, "replication not started")
		return nil
	}
	if err != nil {
		return err
	}
	customer := []Cost{}
	var externalResearch int64
	for _, c := range costs {
		if c.Replication != rep {
			continue
		}
		if strings.HasPrefix(c.ID, "external-") {
			if c.Class == "research" {
				externalResearch += c.DurationNS
			}
			continue
		}
		if c.StartNS < start.NS || c.EndNS > end.At.NS {
			return errors.New("hidden work outside recorded study envelope")
		}
		if c.Class == "customer-machine" {
			customer = append(customer, c)
		}
	}
	sort.Slice(customer, func(i, j int) bool { return customer[i].StartNS < customer[j].StartNS })
	var active, last int64
	for _, c := range customer {
		if c.StartNS < last {
			return errors.New("overlapping/double-charged customer phases")
		}
		active += c.DurationNS
		last = c.EndNS
	}
	e.StudyMachineNS = end.At.NS - start.NS
	measuredResearch := e.StudyMachineNS - active + externalResearch
	if measuredResearch < e.ResearchNS {
		return errors.New("research ledger exceeds actual study interval")
	}
	e.ResearchRecordingAndOtherNS = measuredResearch - e.ResearchNS
	e.ResearchNS = measuredResearch
	e.FullyLoadedNetNS = e.NetSavedNS - e.ResearchNS
	e.WholeStudySensitivityNetNS = e.NetSavedNS - e.StudyMachineNS - externalResearch
	if end.Status != "COMPLETE" {
		e.Complete = false
		e.Gate = "INCOMPLETE_EVIDENCE"
		e.PaybackOrdinal = -1
		e.Reasons = append(e.Reasons, end.Status+": "+end.Reason)
	}
	return nil
}

func stopClass(err error) string {
	if err == nil {
		return "PAUSED"
	}
	for _, class := range []string{candidateFailure, notRunLimit, dependencyUnavailable} {
		if strings.Contains(err.Error(), class) {
			return class
		}
	}
	return harnessInvalid
}
