// Command economics-gated-native-patch-audit reconstructs the source-only
// admission rows for Economics-Gated Reviewed Native Patch v1.
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

const (
	subjectSchema = "buildopt.specs/poc-economics-gated-reviewed-native-patch-subjects/v1"
	reusePolicy   = "PUBLIC_REVISIONS_ONLY_NO_PRIOR_RESULT_ROWS"
	familyCount   = 10
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
	SchemaVersion         string         `json:"schemaVersion"`
	ContractSHA256        string         `json:"contractSha256"`
	SubjectsSHA256        string         `json:"subjectsSha256"`
	AnalyzerSHA256        string         `json:"analyzerSha256"`
	EvidenceSource        string         `json:"evidenceSource"`
	Families              []familyReport `json:"families"`
	ConclusiveFamilies    int            `json:"conclusiveFamilies"`
	ActionFamilies        int            `json:"actionFamilies"`
	MinimumActionFamilies int            `json:"minimumActionFamilies"`
	Decision              string         `json:"decision"`
	DiagnosticsAuthorized bool           `json:"diagnosticsAuthorized"`
	CompilationAuthorized bool           `json:"compilationAuthorized"`
	TimingAuthorized      bool           `json:"timingAuthorized"`
}

func main() {
	var subjectsPath, contractPath, sourceRoot, outputPath string
	flag.StringVar(&subjectsPath, "subjects", "specs/poc-economics-gated-reviewed-native-patch-v1.subjects.json", "EGNP subject manifest")
	flag.StringVar(&contractPath, "contract", "specs/poc-economics-gated-reviewed-native-patch-v1.json", "EGNP machine contract")
	flag.StringVar(&sourceRoot, "source-root", "", "directory containing Git repositories named by subject key")
	flag.StringVar(&outputPath, "output", "", "output JSON path, or stdout")
	flag.Parse()
	if sourceRoot == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: economics-gated-native-patch-audit --source-root DIR [--subjects FILE] [--contract FILE] [--output FILE]")
		os.Exit(64)
	}
	if err := run(subjectsPath, contractPath, sourceRoot, outputPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(subjectsPath, contractPath, sourceRoot, outputPath string) error {
	subjectBytes, err := os.ReadFile(subjectsPath)
	if err != nil {
		return err
	}
	manifest, err := decodeManifest(subjectBytes)
	if err != nil {
		return err
	}
	contractBytes, err := os.ReadFile(contractPath)
	if err != nil {
		return err
	}
	result := report{
		SchemaVersion:  "buildopt.evidence/economics-gated-reviewed-native-patch-source-classification/v1",
		ContractSHA256: digest(contractBytes), SubjectsSHA256: digest(subjectBytes),
		EvidenceSource: "FRESH_FROZEN_GIT_SOURCE_ONLY",
		Families:       []familyReport{}, MinimumActionFamilies: 1,
	}
	result.AnalyzerSHA256, err = analyzerDigest()
	if err != nil {
		return err
	}
	for _, family := range manifest.Families {
		repositoryPath := filepath.Join(sourceRoot, family.Key)
		resolved, resolveErr := git(repositoryPath, "rev-parse", family.Revision+"^{commit}")
		if resolveErr != nil || strings.TrimSpace(resolved) != family.Revision {
			return fmt.Errorf("%s: frozen revision unavailable", family.Key)
		}
		paths, pathErr := taskSourcePaths(repositoryPath, family.Revision)
		if pathErr != nil {
			return fmt.Errorf("%s: %w", family.Key, pathErr)
		}
		entry := familyReport{Key: family.Key, Repository: family.Repository,
			Revision: family.Revision, Conclusive: true, Candidates: []normalizationaware.Candidate{}}
		for _, path := range paths {
			source, sourceErr := gitBytes(repositoryPath, "show", family.Revision+":"+path)
			if sourceErr != nil {
				return fmt.Errorf("%s:%s: %w", family.Key, path, sourceErr)
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
	if result.ConclusiveFamilies == familyCount && result.ActionFamilies >= result.MinimumActionFamilies {
		result.Decision = "PASS_SOURCE_DISCOVERY_ACTION_GATE"
		result.DiagnosticsAuthorized = true
	} else {
		result.Decision = "STOP_SOURCE_DISCOVERY_ACTION_GATE"
	}
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

func decodeManifest(raw []byte) (subjectManifest, error) {
	var manifest subjectManifest
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return subjectManifest{}, err
	}
	if manifest.SchemaVersion != subjectSchema || manifest.ReusePolicy != reusePolicy || len(manifest.Families) != familyCount {
		return subjectManifest{}, errors.New("subject manifest must be the exact ten-family EGNP source cohort")
	}
	seen := map[string]bool{}
	for _, family := range manifest.Families {
		if family.Key == "" || family.Repository == "" || len(family.Revision) != 40 || seen[family.Key] {
			return subjectManifest{}, errors.New("subject manifest contains an invalid or duplicate family")
		}
		seen[family.Key] = true
	}
	return manifest, nil
}

func familyHasAction(candidates []normalizationaware.Candidate) bool {
	for _, candidate := range candidates {
		if candidate.Decision == normalizationaware.MarkerOnlyEligible || candidate.Decision == normalizationaware.ReviewedRelativeProofNeeded {
			return true
		}
	}
	return false
}

func taskSourcePaths(repositoryPath, revision string) ([]string, error) {
	out, err := git(repositoryPath, "grep", "-l", "-e", "extends DefaultTask", "-e", ": DefaultTask(", revision, "--", "*.java", "*.groovy", "*.kt")
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

func git(repositoryPath string, arguments ...string) (string, error) {
	out, err := gitBytes(repositoryPath, arguments...)
	return string(out), err
}

func gitBytes(repositoryPath string, arguments ...string) ([]byte, error) {
	command := exec.Command("git", append([]string{"-C", repositoryPath}, arguments...)...)
	command.Env = append(os.Environ(), "GIT_NO_LAZY_FETCH=1")
	out, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w", strings.Join(arguments, " "), err)
	}
	return out, nil
}

func analyzerDigest() (string, error) {
	hash := sha256.New()
	for _, path := range []string{"cmd/economics-gated-native-patch-audit/main.go", "internal/normalizationaware/scanner.go"} {
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

func digest(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
