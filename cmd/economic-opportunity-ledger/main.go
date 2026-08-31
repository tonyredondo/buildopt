// Command economic-opportunity-ledger creates the EOF-002 chronological
// source-only recurrence ledger. It invokes Git only; it never starts Gradle.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const reportSchema = "buildopt.evidence/economic-opportunity-source-ledger/v1"

type subjectsFile struct {
	SchemaVersion string          `json:"schemaVersion"`
	ReusePolicy   string          `json:"reusePolicy"`
	History       subjectsHistory `json:"history"`
	Families      []subject       `json:"families"`
}

type subjectsHistory struct {
	Kind                                 string `json:"kind"`
	MinimumPriorCommits                  int    `json:"minimumPriorCommits"`
	MinimumLaterCommitsForInstalledValue int    `json:"minimumLaterCommitsForInstalledValue"`
}

type subject struct {
	Key            string `json:"key"`
	Repository     string `json:"repository"`
	AnchorRevision string `json:"anchorRevision"`
}

type ledgerConfig struct {
	SchemaVersion            string         `json:"schemaVersion"`
	HistoryWindow            int            `json:"historyWindow"`
	MinimumCompatibleMatches int            `json:"minimumCompatibleMatches"`
	GlobalPathPatterns       []string       `json:"globalPathPatterns"`
	Families                 []familyConfig `json:"families"`
}

type familyConfig struct {
	Key             string   `json:"key"`
	Entrypoints     []string `json:"entrypoints,omitempty"`
	EntrypointsFile string   `json:"entrypointsFile,omitempty"`
	Toolchain       string   `json:"toolchain"`
	OwnerRoots      []string `json:"ownerRoots"`
}

type report struct {
	SchemaVersion  string         `json:"schemaVersion"`
	WorkItem       string         `json:"workItem"`
	ContractSHA256 string         `json:"contractSha256"`
	SubjectsSHA256 string         `json:"subjectsSha256"`
	ConfigSHA256   string         `json:"configSha256"`
	Families       []familyResult `json:"families"`
	Rows           []row          `json:"rows"`
	Gate           gate           `json:"gate"`
	SideEffects    sideEffects    `json:"sideEffects"`
	Boundaries     reportBoundary `json:"boundaries"`
}

type familyResult struct {
	Key                 string `json:"key"`
	Repository          string `json:"repository"`
	AnchorRevision      string `json:"anchorRevision"`
	WorkflowSHA256      string `json:"workflowSha256"`
	HistoryRows         int    `json:"historyRows"`
	CompatibleMatches   int    `json:"compatibleMatches"`
	GlobalChanges       int    `json:"globalChanges"`
	MixedChanges        int    `json:"mixedChanges"`
	OutsideOwnerChanges int    `json:"outsideOwnerChanges"`
	Decision            string `json:"decision"`
	Reason              string `json:"reason"`
	Conclusive          bool   `json:"conclusive"`
}

type row struct {
	Family            string   `json:"family"`
	Sequence          int      `json:"sequence"`
	Revision          string   `json:"revision"`
	ParentRevision    string   `json:"parentRevision"`
	ChangedPaths      []string `json:"changedPaths"`
	ChangedPathSHA256 string   `json:"changedPathSha256"`
	SourceFactsSHA256 string   `json:"sourceFactsSha256"`
	WorkflowSHA256    string   `json:"workflowSha256"`
	FeatureSHA256     string   `json:"featureSha256"`
	ChangeClass       string   `json:"changeClass"`
	Decision          string   `json:"decision"`
	Reason            string   `json:"reason"`
}

type gate struct {
	ConclusiveFamilies         int    `json:"conclusiveFamilies"`
	RequiredConclusive         int    `json:"requiredConclusiveFamilies"`
	NativeCeilingProbeFamilies int    `json:"nativeCeilingProbeFamilies"`
	RequiredProbeFamilies      int    `json:"requiredNativeCeilingProbeFamilies"`
	Decision                   string `json:"decision"`
	EOF003Authorized           bool   `json:"eof003Authorized"`
	CandidateBuildAuthorized   bool   `json:"candidateBuildAuthorized"`
	TimingAuthorized           bool   `json:"timingAuthorized"`
}

type sideEffects struct {
	GitCommands        int `json:"gitCommands"`
	GradleStarts       int `json:"gradleStarts"`
	CandidateBuilds    int `json:"candidateBuilds"`
	TimingSamples      int `json:"timingSamples"`
	PublicSourceWrites int `json:"publicSourceWrites"`
}

type reportBoundary struct {
	PredecessorEvidenceRowsConsumed bool   `json:"predecessorEvidenceRowsConsumed"`
	RepositoryOrTaskNameRules       bool   `json:"repositoryOrTaskNameRules"`
	PerformanceClaimed              bool   `json:"performanceClaimed"`
	TestOptimization                string `json:"testOptimization"`
}

func main() {
	root := flag.String("repo-root", ".", "BuildOpt repository root")
	histories := flag.String("history-root", "", "directory containing the five public Git histories")
	output := flag.String("output", "", "output JSON path")
	flag.Parse()
	if flag.NArg() != 0 || *histories == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "usage: economic-opportunity-ledger --history-root DIR --output FILE [--repo-root DIR]")
		os.Exit(64)
	}
	candidate, err := generate(*root, *histories)
	if err == nil {
		err = writeJSON(*output, candidate)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "economic opportunity ledger failed: %v\n", err)
		os.Exit(1)
	}
}

func generate(root, historyRoot string) (report, error) {
	contractPath := filepath.Join(root, "specs/poc-economic-opportunity-first-v1.json")
	subjectsPath := filepath.Join(root, "specs/poc-economic-opportunity-first-v1.subjects.json")
	configPath := filepath.Join(root, "specs/poc-economic-opportunity-first-v1.ledger.json")
	contractBytes, err := os.ReadFile(contractPath)
	if err != nil {
		return report{}, err
	}
	subjectBytes, err := os.ReadFile(subjectsPath)
	if err != nil {
		return report{}, err
	}
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return report{}, err
	}
	var subjects subjectsFile
	var config ledgerConfig
	if err := strictJSON(subjectBytes, &subjects); err != nil {
		return report{}, fmt.Errorf("subjects: %w", err)
	}
	if err := strictJSON(configBytes, &config); err != nil {
		return report{}, fmt.Errorf("config: %w", err)
	}
	if config.HistoryWindow != 64 || config.MinimumCompatibleMatches != 5 {
		return report{}, errors.New("unexpected frozen ledger thresholds")
	}

	result := report{
		SchemaVersion: reportSchema, WorkItem: "EOF-002",
		ContractSHA256: digest(contractBytes), SubjectsSHA256: digest(subjectBytes), ConfigSHA256: digest(configBytes),
		Gate:       gate{RequiredConclusive: 5, RequiredProbeFamilies: 3},
		Boundaries: reportBoundary{TestOptimization: "OUT_OF_SCOPE"},
	}
	configs := map[string]familyConfig{}
	for _, value := range config.Families {
		configs[value.Key] = value
	}
	for _, current := range subjects.Families {
		familyConfig, ok := configs[current.Key]
		if !ok {
			return report{}, fmt.Errorf("missing config for %s", current.Key)
		}
		family, rows, gitCommands, err := scanFamily(root, historyRoot, current, familyConfig, config)
		if err != nil {
			return report{}, err
		}
		result.SideEffects.GitCommands += gitCommands
		result.Families = append(result.Families, family)
		result.Rows = append(result.Rows, rows...)
		if family.Conclusive {
			result.Gate.ConclusiveFamilies++
		}
		if family.Decision == "ADMIT_NATIVE_CEILING_PROBE" {
			result.Gate.NativeCeilingProbeFamilies++
		}
	}
	sort.Slice(result.Families, func(i, j int) bool { return result.Families[i].Key < result.Families[j].Key })
	sort.Slice(result.Rows, func(i, j int) bool {
		if result.Rows[i].Family == result.Rows[j].Family {
			return result.Rows[i].Sequence < result.Rows[j].Sequence
		}
		return result.Rows[i].Family < result.Rows[j].Family
	})
	result.Gate.EOF003Authorized = result.Gate.ConclusiveFamilies == 5 && result.Gate.NativeCeilingProbeFamilies >= 3
	if result.Gate.EOF003Authorized {
		result.Gate.Decision = "PASS_SOURCE_RECURRENCE_BREADTH"
	} else {
		result.Gate.Decision = "STOP_INSUFFICIENT_SOURCE_RECURRENCE_BREADTH"
	}
	return result, nil
}

func scanFamily(root, historyRoot string, current subject, cfg familyConfig, all ledgerConfig) (familyResult, []row, int, error) {
	repo := filepath.Join(historyRoot, current.Key)
	if _, err := os.Stat(filepath.Join(repo, ".git")); err != nil {
		return familyResult{}, nil, 0, fmt.Errorf("%s Git history: %w", current.Key, err)
	}
	entrypoints := append([]string(nil), cfg.Entrypoints...)
	if cfg.EntrypointsFile != "" {
		data, err := os.ReadFile(filepath.Join(root, cfg.EntrypointsFile))
		if err != nil {
			return familyResult{}, nil, 0, err
		}
		entrypoints = nonemptyLines(string(data))
	}
	workflowSHA := digest([]byte(strings.Join(append(entrypoints, cfg.Toolchain), "\x00")))
	commitsText, err := git(repo, "rev-list", "--first-parent", fmt.Sprintf("--max-count=%d", all.HistoryWindow), current.AnchorRevision+"^")
	if err != nil {
		return familyResult{}, nil, 1, fmt.Errorf("%s history: %w", current.Key, err)
	}
	commits := nonemptyLines(commitsText)
	if len(commits) != all.HistoryWindow {
		return familyResult{}, nil, 1, fmt.Errorf("%s history rows = %d, want %d", current.Key, len(commits), all.HistoryWindow)
	}
	result := familyResult{Key: current.Key, Repository: current.Repository, AnchorRevision: current.AnchorRevision, WorkflowSHA256: workflowSHA, HistoryRows: len(commits), Conclusive: true}
	rows := make([]row, 0, len(commits))
	gitCommands := 1
	for index, revision := range commits {
		parent, err := git(repo, "rev-parse", revision+"^")
		gitCommands++
		if err != nil {
			return familyResult{}, nil, gitCommands, err
		}
		pathText, err := git(repo, "diff-tree", "--no-commit-id", "--name-only", "-r", revision)
		gitCommands++
		if err != nil {
			return familyResult{}, nil, gitCommands, err
		}
		paths := uniqueSorted(nonemptyLines(pathText))
		pathDigest := digest([]byte(strings.Join(paths, "\n") + "\n"))
		changeClass, decision, reason := classify(paths, cfg.OwnerRoots, all.GlobalPathPatterns)
		featureSHA := digest([]byte(strings.Join([]string{workflowSHA, strings.Join(cfg.OwnerRoots, "\x00"), pathDigest, changeClass}, "\x00")))
		sourceSHA := digest([]byte(revision + "\n" + strings.TrimSpace(parent) + "\n" + strings.Join(paths, "\n") + "\n"))
		rows = append(rows, row{Family: current.Key, Sequence: index + 1, Revision: revision, ParentRevision: strings.TrimSpace(parent), ChangedPaths: paths, ChangedPathSHA256: pathDigest, SourceFactsSHA256: sourceSHA, WorkflowSHA256: workflowSHA, FeatureSHA256: featureSHA, ChangeClass: changeClass, Decision: decision, Reason: reason})
		switch changeClass {
		case "OWNER_ONLY":
			result.CompatibleMatches++
		case "GLOBAL_OR_BUILD_LOGIC":
			result.GlobalChanges++
		case "MIXED_OWNER_BOUNDARY":
			result.MixedChanges++
		case "OUTSIDE_OWNER":
			result.OutsideOwnerChanges++
		}
	}
	if result.CompatibleMatches >= all.MinimumCompatibleMatches {
		result.Decision = "ADMIT_NATIVE_CEILING_PROBE"
		result.Reason = "SOURCE_RECURRENCE_THRESHOLD_MET"
	} else {
		result.Decision = "REJECT_INSUFFICIENT_RECURRENCE"
		result.Reason = "SOURCE_RECURRENCE_BELOW_FIVE_MATCHES"
	}
	return result, rows, gitCommands, nil
}

func classify(paths, roots, globalPatterns []string) (string, string, string) {
	if len(paths) == 0 {
		return "INCOMPLETE", "REJECT_INCOMPLETE_OR_AMBIGUOUS", "EMPTY_CHANGED_PATH_SET"
	}
	for _, path := range paths {
		if matchesAny(path, globalPatterns) {
			return "GLOBAL_OR_BUILD_LOGIC", "NO_ACTION", "GLOBAL_OR_BUILD_LOGIC_CHANGE"
		}
	}
	inside, outside := false, false
	for _, path := range paths {
		if matchesAny(path, roots) {
			inside = true
		} else {
			outside = true
		}
	}
	switch {
	case inside && !outside:
		return "OWNER_ONLY", "ADMIT_NATIVE_CEILING_PROBE", "MATCHING_OWNER_AND_CHANGE_FAMILY"
	case inside:
		return "MIXED_OWNER_BOUNDARY", "REJECT_INCOMPLETE_OR_AMBIGUOUS", "MIXED_OWNER_BOUNDARY"
	default:
		return "OUTSIDE_OWNER", "NO_ACTION", "DIFFERENT_OWNER_OR_CHANGE_FAMILY"
	}
}

func matchesAny(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.HasSuffix(pattern, "/") && strings.HasPrefix(path, pattern) || path == pattern {
			return true
		}
	}
	return false
}
func git(repo string, args ...string) (string, error) {
	command := exec.Command("git", append([]string{"-C", repo}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, bytes.TrimSpace(output))
	}
	return string(output), nil
}
func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func nonemptyLines(value string) []string {
	var result []string
	for _, line := range strings.Split(value, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			result = append(result, line)
		}
	}
	return result
}
func uniqueSorted(values []string) []string {
	sort.Strings(values)
	result := values[:0]
	for _, value := range values {
		if len(result) == 0 || result[len(result)-1] != value {
			result = append(result, value)
		}
	}
	return result
}
func strictJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON value")
	}
	return nil
}
func writeJSON(path string, value report) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
