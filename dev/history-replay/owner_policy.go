//go:build linux && amd64

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

// OwnerPolicy binds the previously qualified Elasticsearch output readers.
// The batch comparator consumes complete retained captures outside customer
// request envelopes; it cannot alter task execution or captured output bytes.
type OwnerPolicy struct {
	Schema            string    `json:"schema"`
	Kind              string    `json:"kind"`
	BaseContract      Binding   `json:"baseContract"`
	DateContract      Binding   `json:"dateContract"`
	Interpreter       Binding   `json:"interpreter"`
	Comparator        Binding   `json:"comparator"`
	LexicalProjector  Binding   `json:"lexicalProjector"`
	MetadataJava      Binding   `json:"metadataJava"`
	MetadataClasspath []Binding `json:"metadataClasspath"`
	Qualification     Binding   `json:"qualification"`
	ReusePolicy       string    `json:"reusePolicy"`
}

type OwnerQualification struct {
	Schema           string    `json:"schema"`
	Decision         string    `json:"decision"`
	ComparatorSHA256 string    `json:"comparatorSHA256"`
	ReadersSHA256    string    `json:"readersSHA256"`
	Cases            []Binding `json:"cases"`
	NativeReuse      []Binding `json:"nativeReuse"`
}

// Qualification follows reader bytes and classpath order across relocations.
// A different metadata reader cannot borrow an old comparator's proof.
func ownerReadersDigest(p OwnerPolicy) string {
	identities := []string{p.Schema, p.Kind, p.ReusePolicy, p.BaseContract.SHA256, p.DateContract.SHA256, p.Interpreter.SHA256, p.Comparator.SHA256, p.LexicalProjector.SHA256, p.MetadataJava.SHA256}
	for _, b := range p.MetadataClasspath {
		identities = append(identities, b.SHA256)
	}
	return objectDigest(identities)
}

func readOwnerPolicy(m Manifest) (OwnerPolicy, error) {
	var p OwnerPolicy
	if absentBinding(m.Outputs.Owner) {
		return p, nil
	}
	if m.Driver != "GRADLE" || len(m.Outputs.Projectors) != 0 || len(m.Outputs.Rules) != 0 || len(m.Outputs.Diagnostics) != 0 {
		return p, errors.New("owner policy requires complete native root-build capture without extra output rules")
	}
	if err := checkBinding(m.Outputs.Owner); err != nil {
		return p, err
	}
	if err := readJSON(m.Outputs.Owner.Path, &p); err != nil {
		return p, err
	}
	if p.Schema != "buildopt.history-replay/owner-policy/v1" || p.Kind != "ELASTICSEARCH_CHECKSTYLE_C5" || p.ReusePolicy != "same-arm-earlier-executed-exact-raw-v1" {
		return p, errors.New("unsupported owner output policy")
	}
	if p.BaseContract.SHA256 != "0f78177a1f79ac2d0c909ef4f3c56ac14aaf8158e4d5f0e633891ac7e8f7be34" || p.DateContract.SHA256 != "156bc82fc75d2283b13548813d5e67741d9f6f54cbc15ec7404d717def36b46f" || p.LexicalProjector.SHA256 != "7f66c4d772512669bbd0912da4751712111a18a8b2633ad663e3f2dcd755f7ce" {
		return p, errors.New("owner policy differs from frozen C5 readers/allowlist")
	}
	if len(p.MetadataClasspath) == 0 {
		return p, errors.New("missing complete native compiler-state reader")
	}
	bindings := append([]Binding{p.BaseContract, p.DateContract, p.Interpreter, p.Comparator, p.LexicalProjector, p.MetadataJava}, p.MetadataClasspath...)
	for _, b := range bindings {
		if err := checkBinding(b); err != nil {
			return p, err
		}
	}
	if absentBinding(p.Qualification) && m.Phase == "QUALIFICATION" {
		return p, nil
	}
	if err := checkBinding(p.Qualification); err != nil {
		return p, err
	}
	var q OwnerQualification
	if err := readJSON(p.Qualification.Path, &q); err != nil {
		return p, err
	}
	if q.Schema != "buildopt.history-replay/owner-qualification/v1" || q.Decision != "OWNER_OUTPUT_POLICY_QUALIFIED" || q.ComparatorSHA256 != p.Comparator.SHA256 || q.ReadersSHA256 != ownerReadersDigest(p) || len(q.Cases) == 0 || len(q.NativeReuse) == 0 {
		return p, errors.New("owner comparison/reuse qualification is incomplete")
	}
	for _, b := range append(q.Cases, q.NativeReuse...) {
		if err := checkBinding(b); err != nil {
			return p, err
		}
	}
	return p, nil
}

func compareOwnerCaptures(m Manifest, n, i AttemptEnd) (int, error) {
	p, err := readOwnerPolicy(m)
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, p.Interpreter.Path, "-I", p.Comparator.Path)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C.UTF-8", "TZ=UTC", "PYTHONDONTWRITEBYTECODE=1"}
	cmd.Stdin = bytes.NewReader(jsonBytes(map[string]any{"policy": m.Outputs.Owner, "runRoot": m.RunRoot, "native": n, "candidate": i, "controlAdapted": m.ControlBaseline != nil}))
	var diagnostic bytes.Buffer
	cmd.Stderr = &diagnostic
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("owner output comparison: %w: %s", err, diagnostic.String())
	}
	var report struct {
		Status            string         `json:"status"`
		PolicySHA256      string         `json:"policySHA256"`
		Captures          []Binding      `json:"captures"`
		Counts            map[string]int `json:"counts"`
		MetadataJVMStarts int            `json:"metadataJVMStarts"`
	}
	if err := decodeStrict(output, &report); err != nil {
		return 0, err
	}
	if report.Status != "equivalent" || report.PolicySHA256 != m.Outputs.Owner.SHA256 || !equalJSON(report.Captures, []Binding{n.Capture, i.Capture}) || report.MetadataJVMStarts < 0 || report.MetadataJVMStarts > 1 {
		return 0, errors.New("owner comparator returned an unbound result")
	}
	return report.MetadataJVMStarts, nil
}
