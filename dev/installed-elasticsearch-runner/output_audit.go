package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type nativeOutputDifference struct {
	First  entry    `json:"first"`
	Second entry    `json:"second"`
	Owners []string `json:"owners"`
}

type nativeOutputAudit struct {
	Schema        string                   `json:"schemaVersion"`
	Captures      []fileBinding            `json:"captures"`
	Entries       int                      `json:"entriesPerCapture"`
	EqualEntries  int                      `json:"rawEqualEntries"`
	EqualClasses  int                      `json:"rawEqualClassFiles"`
	AcceptedDates []string                 `json:"verifiedAcceptedDateDifferences"`
	Unapproved    []nativeOutputDifference `json:"unapprovedDifferences"`
	GatePassed    bool                     `json:"gatePassed"`
	ValueAdmitted bool                     `json:"valueAdmitted"`
}

// Exhaustive read-only diagnosis after a strict comparison stops. Unapproved
// differences remain failures; this does not introduce an output projection.
func auditNativeOutputs(first, second fileBinding) (nativeOutputAudit, error) {
	out := nativeOutputAudit{Schema: "buildopt.eic/native-output-audit/v1", Captures: []fileBinding{first, second}, AcceptedDates: []string{}, Unapproved: []nativeOutputDifference{}}
	a, err := loadComparisonCapture(first)
	if err != nil {
		return out, err
	}
	b, err := loadComparisonCapture(second)
	if err != nil {
		return out, err
	}
	if !reflect.DeepEqual(a.Graph, b.Graph) || len(a.Inventory) != len(b.Inventory) {
		return out, errors.New("native graph or output path set differs")
	}
	out.Entries = len(a.Inventory)
	contract, err := loadAcceptedDateContract()
	if err != nil {
		return out, err
	}
	rules := map[string]dateOutputRule{}
	for _, rule := range contract.Allowlist {
		rules[rule.Path] = rule
	}
	for i, x := range a.Inventory {
		y := b.Inventory[i]
		if x == y {
			out.EqualEntries++
			if x.Type == "file" && strings.HasSuffix(x.Path, ".class") {
				out.EqualClasses++
			}
			continue
		}
		if x.Path != y.Path || x.Type != "file" || y.Type != "file" || x.Mode != y.Mode {
			return out, errors.New("native output path, type or mode differs")
		}
		rule, ok := rules[x.Path]
		if !ok {
			difference := nativeOutputDifference{First: x, Second: y, Owners: []string{}}
			for _, task := range a.Graph.Tasks {
				for _, output := range task.Outputs {
					if output == x.Path || strings.HasPrefix(x.Path, output+"/") {
						difference.Owners = append(difference.Owners, task.Identity)
						break
					}
				}
			}
			out.Unapproved = append(out.Unapproved, difference)
			continue
		}
		projections := []string{}
		for _, capture := range []comparisonCapture{a, b} {
			if err = capture.executedOwner(x.Path, rule.Producer); err != nil {
				return out, err
			}
			if err = capture.verifyManifestSource(rule); err != nil {
				return out, err
			}
			path := filepath.Join(capture.Root, x.Path)
			info, err := os.Stat(path)
			if err != nil || info.Size() > maximumDateArchive {
				return out, errors.New("invalid date archive size")
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return out, err
			}
			var projection string
			if len(rule.AllowedMembers) > 0 {
				projection, err = archiveProjection(raw, rule.AllowedMembers, capture.Window)
			} else {
				var projected []byte
				projected, err = manifestProjection(raw, capture.Window)
				projection = digest(projected)
			}
			if err != nil {
				return out, err
			}
			projections = append(projections, projection)
		}
		if projections[0] != projections[1] {
			return out, errors.New("output differs beyond accepted dates")
		}
		out.AcceptedDates = append(out.AcceptedDates, x.Path)
	}
	out.GatePassed = len(out.Unapproved) == 0
	return out, nil
}
