//go:build linux && amd64

package main

import (
	"errors"
	"fmt"
	"path/filepath"
)

type ProjectionCase struct {
	Name           string  `json:"name"`
	Kind           string  `json:"kind"`
	Input          Binding `json:"input"`
	Root           string  `json:"root"`
	Start          Stamp   `json:"start"`
	End            Stamp   `json:"end"`
	ExpectedSHA256 string  `json:"expectedSHA256"`
	ExpectedError  bool    `json:"expectedError"`
}
type ProjectorProof struct {
	Schema     string           `json:"schema"`
	Identity   string           `json:"identity"`
	Executable Binding          `json:"executable"`
	Arguments  []string         `json:"arguments"`
	Cases      []ProjectionCase `json:"cases"`
}

// A proof has independent equivalent, semantic-change and malformed inputs.
// Exact expected hashes are frozen before use. A constant/drop-all projector
// cannot satisfy the semantic counterexample. Public owner-specific projection
// still needs its BV-006 output contract; this is only executable admission.
func checkProjectorProof(p Projector, execute bool) error {
	if err := checkBinding(p.Qualification); err != nil {
		return err
	}
	var proof ProjectorProof
	if err := readJSON(p.Qualification.Path, &proof); err != nil {
		return err
	}
	if proof.Schema != "buildopt.history-replay/projector-proof/v1" || proof.Identity != p.Identity || proof.Executable != p.Executable || !equalJSON(proof.Arguments, p.Arguments) {
		return errors.New("projector qualification uses different executable or arguments")
	}
	names := map[string]bool{}
	equivalentHashes := map[string]bool{}
	equivalentInputs := map[string]bool{}
	semanticHashes := map[string]bool{}
	negative := 0
	for _, test := range proof.Cases {
		if !namePattern.MatchString(test.Name) || names[test.Name] || !filepath.IsAbs(test.Root) {
			return errors.New("invalid projector test identity")
		}
		names[test.Name] = true
		if err := validateBinding(test.Input); err != nil {
			return err
		}
		if err := checkStampInterval(test.Start, test.End); err != nil {
			return err
		}
		if test.ExpectedError {
			if test.Kind != "malformed" || test.ExpectedSHA256 != "" {
				return errors.New("invalid negative projection case")
			}
			negative++
		} else {
			if !shaPattern.MatchString(test.ExpectedSHA256) {
				return errors.New("missing independently expected projection hash")
			}
			switch test.Kind {
			case "equivalent":
				equivalentHashes[test.ExpectedSHA256] = true
				equivalentInputs[test.Input.SHA256] = true
			case "semantic-change":
				semanticHashes[test.ExpectedSHA256] = true
			default:
				return errors.New("unknown projection case purpose")
			}
		}
		if execute {
			m := Manifest{Outputs: OutputPolicy{Projectors: []Projector{p}}}
			got, err := projection(m, OutputRule{Transform: p.Identity}, test.Input.Path, test.Root, ProcessReceipt{Start: test.Start, End: test.End})
			if test.ExpectedError {
				if err == nil {
					return fmt.Errorf("projector accepted malformed input %s", test.Name)
				}
			} else if err != nil || got != test.ExpectedSHA256 {
				return fmt.Errorf("projector %s failed independent case %s: %v", p.Identity, test.Name, err)
			}
		}
	}
	if len(equivalentHashes) != 1 || len(equivalentInputs) < 2 || len(semanticHashes) == 0 || negative == 0 {
		return errors.New("projector lacks equivalent, semantic and malformed counterexamples")
	}
	for hash := range semanticHashes {
		if equivalentHashes[hash] {
			return errors.New("projector erases a declared semantic change")
		}
	}
	return nil
}
