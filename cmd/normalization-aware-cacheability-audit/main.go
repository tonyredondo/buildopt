package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tonyredondo/buildopt/internal/normalizationaware"
)

type subjectManifest struct {
	SchemaVersion string    `json:"schemaVersion"`
	ReusePolicy   string    `json:"reusePolicy"`
	Families      []subject `json:"families"`
}
type subject struct {
	Key        string `json:"key"`
	Repository string `json:"repository"`
	Revision   string `json:"revision"`
}
type familyReport struct {
	Key        string                         `json:"key"`
	Repository string                         `json:"repository"`
	Revision   string                         `json:"revision"`
	Conclusive bool                           `json:"conclusive"`
	Candidates []normalizationaware.Candidate `json:"candidates"`
}
type report struct {
	SchemaVersion          string         `json:"schemaVersion"`
	ContractRevision       string         `json:"contractRevision"`
	AnalyzerSHA256         string         `json:"analyzerSha256"`
	EvidenceSource         string         `json:"evidenceSource"`
	Families               []familyReport `json:"families"`
	ConclusiveFamilies     int            `json:"conclusiveFamilies"`
	ActionFamilies         int            `json:"actionFamilies"`
	RequiredActionFamilies int            `json:"requiredActionFamilies"`
	Decision               string         `json:"decision"`
	CompilationAuthorized  bool           `json:"compilationAuthorized"`
	TimingAuthorized       bool           `json:"timingAuthorized"`
}

func main() {
	var subjectsPath, sourceRoot, outputPath, contractRevision string
	flag.StringVar(&subjectsPath, "subjects", "specs/poc-normalization-aware-cacheability-v2.subjects.json", "NAC v2 subject manifest")
	flag.StringVar(&sourceRoot, "source-root", "", "directory containing Git repositories named by subject key")
	flag.StringVar(&outputPath, "output", "", "output JSON path, or stdout")
	flag.StringVar(&contractRevision, "contract-revision", "", "commit that froze NAC-001")
	flag.Parse()
	if sourceRoot == "" || contractRevision == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: normalization-aware-cacheability-audit --source-root DIR --contract-revision SHA [--subjects FILE] [--output FILE]")
		os.Exit(64)
	}
	if err := run(subjectsPath, sourceRoot, outputPath, contractRevision); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(subjectsPath, sourceRoot, outputPath, contractRevision string) error {
	raw, err := os.ReadFile(subjectsPath)
	if err != nil {
		return err
	}
	var subjects subjectManifest
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&subjects); err != nil {
		return err
	}
	if subjects.SchemaVersion != "buildopt.specs/poc-normalization-aware-cacheability-subjects/v2" || subjects.ReusePolicy != "SOURCE_REVISIONS_ONLY_NO_DNO_EVIDENCE_ROWS" || len(subjects.Families) != 5 {
		return errors.New("subject manifest must be the exact five-family NAC v2 source cohort")
	}
	result := report{SchemaVersion: "buildopt.evidence/normalization-aware-cacheability-source-classification/v2", ContractRevision: contractRevision, EvidenceSource: "FRESH_FROZEN_GIT_SOURCE_ONLY", RequiredActionFamilies: 3, Families: []familyReport{}}
	result.AnalyzerSHA256, err = analyzerDigest()
	if err != nil {
		return err
	}
	for _, family := range subjects.Families {
		repo := filepath.Join(sourceRoot, family.Key)
		got, err := git(repo, "rev-parse", family.Revision+"^{commit}")
		if err != nil || strings.TrimSpace(got) != family.Revision {
			return fmt.Errorf("%s: frozen revision unavailable", family.Key)
		}
		paths, err := taskSourcePaths(repo, family.Revision)
		if err != nil {
			return fmt.Errorf("%s: %w", family.Key, err)
		}
		entry := familyReport{Key: family.Key, Repository: family.Repository, Revision: family.Revision, Conclusive: true, Candidates: []normalizationaware.Candidate{}}
		for _, path := range paths {
			source, err := gitBytes(repo, "show", family.Revision+":"+path)
			if err != nil {
				return fmt.Errorf("%s:%s: %w", family.Key, path, err)
			}
			entry.Candidates = append(entry.Candidates, normalizationaware.ScanSourceV2(path, source)...)
		}
		normalizationaware.SortCandidates(entry.Candidates)
		result.Families = append(result.Families, entry)
		result.ConclusiveFamilies++
		if familyHasAction(entry.Candidates) {
			result.ActionFamilies++
		}
	}
	if result.ConclusiveFamilies == 5 && result.ActionFamilies >= result.RequiredActionFamilies {
		result.Decision = "PASS_SOURCE_ACTION_BREADTH"
		result.CompilationAuthorized = true
	} else {
		result.Decision = "STOP_SOURCE_ACTION_BREADTH"
	}
	result.TimingAuthorized = false
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if outputPath == "" {
		_, err = os.Stdout.Write(encoded)
		return err
	}
	return os.WriteFile(outputPath, encoded, 0o644)
}

func familyHasAction(candidates []normalizationaware.Candidate) bool {
	for _, candidate := range candidates {
		if candidate.Decision == normalizationaware.MarkerOnlyEligible || candidate.Decision == normalizationaware.ReviewedRelativeProofNeeded {
			return true
		}
	}
	return false
}
func taskSourcePaths(repo, revision string) ([]string, error) {
	out, err := git(repo, "grep", "-l", "-e", "extends DefaultTask", "-e", ": DefaultTask(", revision, "--", "*.java", "*.groovy", "*.kt")
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return []string{}, nil
		}
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	paths := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		path, ok := strings.CutPrefix(line, revision+":")
		if !ok {
			return nil, fmt.Errorf("unexpected git grep row %q", line)
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}
func git(repo string, args ...string) (string, error) {
	out, err := gitBytes(repo, args...)
	return string(out), err
}
func gitBytes(repo string, args ...string) ([]byte, error) {
	command := exec.Command("git", append([]string{"-C", repo}, args...)...)
	command.Env = append(os.Environ(), "GIT_NO_LAZY_FETCH=1")
	out, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}
func analyzerDigest() (string, error) {
	hash := sha256.New()
	for _, path := range []string{"cmd/normalization-aware-cacheability-audit/main.go", "internal/normalizationaware/scanner.go"} {
		content, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		hash.Write([]byte(path))
		hash.Write([]byte{0})
		hash.Write(content)
		hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
