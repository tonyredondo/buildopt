package main

import (
	"embed"
	"errors"
	"path/filepath"
	"reflect"
)

const provenanceProposalSHA = "449e56c3924704129ae90d498004392fdfeb2ce51e02f11e6da578d15422a14d"
const provenanceSeedSHA = "17120c2cc69e1387f3676ef547b5a376b61ec6bc3a8cbe1ec9eb1901e560f03e"

//go:embed provenance-runtime-contract-v1.json
var acceptedProvenanceContract []byte

// Bind qualification to the implementation actually compiled into this runner.
// Historical runners retain their own source bytes and original contracts.
//
//go:embed provenance_*.go output_compare.go correctness_capture.go correctness_run.go correctness_request.go
var provenanceSources embed.FS

func provenanceSourceHashes() map[string]string {
	result := map[string]string{}
	files, _ := provenanceSources.ReadDir(".")
	for _, file := range files {
		raw, _ := provenanceSources.ReadFile(file.Name())
		result[file.Name()] = digest(raw)
	}
	return result
}

type provenanceRuntime struct {
	Schema        string      `json:"schemaVersion"`
	Root          string      `json:"campaignRoot"`
	ContractSHA   string      `json:"contractSha256"`
	Decision      fileBinding `json:"ownerDecision"`
	Qualification fileBinding `json:"qualification"`
	Seed          fileBinding `json:"seedCapture"`
}

func newProvenanceSession(binding *fileBinding, root string) (*provenanceSession, error) {
	if binding == nil {
		return nil, nil
	}
	if err := checkBinding(*binding); err != nil {
		return nil, err
	}
	var r provenanceRuntime
	if err := readJSON(binding.Path, &r); err != nil {
		return nil, err
	}
	if r.Schema != "buildopt.eic/reused-output-provenance-runtime/v1" || r.Root != root || !filepath.IsAbs(root) || filepath.Clean(root) != root || r.ContractSHA != digest(acceptedProvenanceContract) || r.Seed.SHA256 != provenanceSeedSHA {
		return nil, errors.New("approved provenance runtime identity drift")
	}
	for _, pin := range []fileBinding{r.Decision, r.Qualification, r.Seed} {
		if err := checkBinding(pin); err != nil {
			return nil, err
		}
	}
	var decision map[string]any
	if err := readJSON(r.Decision.Path, &decision); err != nil {
		return nil, err
	}
	proposal, ok := decision["proposal"].(map[string]any)
	if !ok || proposal["sha256"] != provenanceProposalSHA || decision["decision"] != "ACCEPTED" || decision["ownerMessage"] != "Aprobado" || decision["campaignRoot"] != root {
		return nil, errors.New("exact reused-output owner approval required")
	}
	var q struct {
		Schema       string        `json:"schemaVersion"`
		ContractSHA  string        `json:"contractSha256"`
		Sources      []fileBinding `json:"sources"`
		Checks       []fileBinding `json:"checks"`
		Passed       bool          `json:"passed"`
		GradleStarts int           `json:"gradleStarts"`
	}
	if err := readJSON(r.Qualification.Path, &q); err != nil {
		return nil, err
	}
	if q.Schema != "buildopt.eic/reused-output-provenance-qualification/v1" || q.ContractSHA != r.ContractSHA || !q.Passed || q.GradleStarts != 0 || len(q.Checks) < 3 || len(q.Sources) < 8 {
		return nil, errors.New("provenance qualification required before starts")
	}
	expected := provenanceSourceHashes()
	for _, pin := range q.Sources {
		name := filepath.Base(pin.Path)
		if expected[name] != pin.SHA256 {
			return nil, errors.New("qualification differs from compiled provenance implementation")
		}
		delete(expected, name)
	}
	if len(expected) != 0 {
		return nil, errors.New("incomplete provenance source qualification")
	}
	for _, pin := range append(q.Sources, q.Checks...) {
		if err := checkBinding(pin); err != nil {
			return nil, err
		}
	}
	seed, err := loadComparisonCaptureMetadata(r.Seed)
	if err != nil {
		return nil, err
	}
	if _, err = metadataOwner(seed, checkstyleOutputPath, checkstyleOwner, checkstyleClass); err != nil {
		return nil, err
	}
	e, err := retainedEntry(seed, checkstyleOutputPath)
	if err != nil {
		return nil, err
	}
	if e.SHA256 != seededCheckstyleSHA || !reflect.DeepEqual(seed.Binding, r.Seed) {
		return nil, errors.New("exact approved Checkstyle seed required")
	}
	return &provenanceSession{Root: root, Seed: seed}, nil
}
