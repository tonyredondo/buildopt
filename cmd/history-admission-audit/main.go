// Command history-admission-audit reconstructs bounded source-history admission
// from one exact, content-bound Gradle discovery snapshot. It never runs Gradle.
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
	"sort"
	"strconv"
	"strings"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
	"github.com/tonyredondo/buildopt/internal/historyadmission"
)

const schemaVersion = "buildopt.history-admission/graph-aware-audit/v1"

type stringList []string

func (values *stringList) String() string { return strings.Join(*values, ",") }
func (values *stringList) Set(value string) error {
	if value == "" {
		return errors.New("value is empty")
	}
	*values = append(*values, value)
	return nil
}

type auditRow struct {
	Commit           string   `json:"commit"`
	ChangedPaths     []string `json:"changedPaths"`
	Owners           []string `json:"owners,omitempty"`
	AffectedProjects []string `json:"affectedProjects,omitempty"`
	Family           string   `json:"family,omitempty"`
	Decision         string   `json:"decision"`
	Reason           string   `json:"reason"`
}

type auditReport struct {
	SchemaVersion      string     `json:"schemaVersion"`
	TargetRevision     string     `json:"targetRevision"`
	SnapshotSHA256     string     `json:"snapshotSha256"`
	Entrypoints        []string   `json:"entrypoints"`
	ExpectedOwners     []string   `json:"expectedOwners"`
	ExpectedFamily     string     `json:"expectedFamily"`
	HistoryWindow      int        `json:"historyWindowCommits"`
	CompatibleCommits  int        `json:"compatibleCommits"`
	MinimumCompatible  int        `json:"minimumCompatibleCommits"`
	Decision           string     `json:"decision"`
	Reason             string     `json:"reason"`
	GradleExecuted     bool       `json:"gradleExecuted"`
	RepositoryNameRule bool       `json:"repositoryNameRule"`
	Rows               []auditRow `json:"rows"`
}

type auditOptions struct {
	repository, snapshotPath, snapshotSHA, target, family string
	entrypoints, owners                                   []string
	maximum, minimum                                      int
}

func main() {
	var options auditOptions
	var entrypoints, owners stringList
	flag.StringVar(&options.repository, "repository", "", "path to the public Git repository")
	flag.StringVar(&options.snapshotPath, "snapshot", "", "path to the retained discovery snapshot")
	flag.StringVar(&options.snapshotSHA, "snapshot-sha256", "", "required SHA-256 binding for the snapshot")
	flag.StringVar(&options.target, "target", "", "target revision")
	flag.StringVar(&options.family, "expected-family", "", "expected change family")
	flag.Var(&entrypoints, "entrypoint", "observed entrypoint (repeatable)")
	flag.Var(&owners, "expected-owner", "expected owner (repeatable)")
	flag.IntVar(&options.maximum, "maximum", 64, "maximum first-parent commits")
	flag.IntVar(&options.minimum, "minimum", 5, "minimum compatible commits")
	flag.Parse()
	options.entrypoints, options.owners = entrypoints, owners
	report, err := runAudit(options)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runAudit(options auditOptions) (auditReport, error) {
	if options.repository == "" || options.snapshotPath == "" || options.target == "" ||
		len(options.entrypoints) == 0 || len(options.owners) == 0 || options.family == "" ||
		options.maximum < 1 || options.maximum > 1024 || options.minimum < 1 || options.minimum > options.maximum {
		return auditReport{}, errors.New("complete bounded audit arguments are required")
	}
	raw, err := os.ReadFile(options.snapshotPath)
	if err != nil {
		return auditReport{}, fmt.Errorf("read snapshot: %w", err)
	}
	digest := sha256.Sum256(raw)
	digestText := hex.EncodeToString(digest[:])
	if options.snapshotSHA != digestText {
		return auditReport{}, errors.New("snapshot SHA-256 binding does not match")
	}
	snapshot, err := buildimpact.ParseObservedDiscoverySnapshot(raw, options.entrypoints)
	if err != nil {
		return auditReport{}, fmt.Errorf("parse snapshot: %w", err)
	}
	sort.Strings(options.owners)
	commitsRaw, err := gitOutput(options.repository, "rev-list", "--first-parent", "--max-count="+strconv.Itoa(options.maximum), options.target)
	if err != nil {
		return auditReport{}, err
	}
	commits := strings.Fields(string(commitsRaw))
	if len(commits) == 0 || len(commits) > options.maximum {
		return auditReport{}, errors.New("bounded first-parent history is unavailable")
	}
	report := auditReport{SchemaVersion: schemaVersion, TargetRevision: options.target, SnapshotSHA256: digestText,
		Entrypoints: append([]string(nil), options.entrypoints...), ExpectedOwners: append([]string(nil), options.owners...),
		ExpectedFamily: options.family, HistoryWindow: len(commits), MinimumCompatible: options.minimum,
		Decision: "REJECT", Reason: "INSUFFICIENT_GRAPH_COMPATIBLE_HISTORY", Rows: make([]auditRow, 0, len(commits))}
	for _, commit := range commits {
		pathsRaw, pathErr := gitOutput(options.repository, "diff-tree", "--root", "--no-commit-id", "--name-only", "--no-renames", "-r", "-z", commit, "--")
		paths := splitNUL(pathsRaw)
		row := auditRow{Commit: commit, ChangedPaths: paths, Decision: "REJECT", Reason: "INCOMPLETE_AMBIGUOUS"}
		if pathErr != nil || len(paths) == 0 {
			report.Rows = append(report.Rows, row)
			continue
		}
		if historyadmission.UnsafeStructuralChange(paths) {
			row.Reason = "UNSAFE_STRUCTURAL_CHANGE"
			report.Rows = append(report.Rows, row)
			continue
		}
		classification, classifyErr := historyadmission.Classify(snapshot, paths)
		if classifyErr != nil {
			report.Rows = append(report.Rows, row)
			continue
		}
		row.Owners, row.AffectedProjects, row.Family = classification.Owners, classification.AffectedProjects, classification.Family
		if !equalStrings(classification.Owners, options.owners) {
			row.Reason = "OWNER_MISMATCH"
		} else if classification.Family != options.family {
			row.Reason = "FAMILY_MISMATCH"
		} else {
			row.Decision, row.Reason = "COMPATIBLE", "EXACT_GRAPH_OWNER_AND_FAMILY"
			report.CompatibleCommits++
		}
		report.Rows = append(report.Rows, row)
	}
	if report.CompatibleCommits >= report.MinimumCompatible {
		report.Decision, report.Reason = "ADMIT", "MINIMUM_GRAPH_COMPATIBLE_HISTORY_MET"
	}
	return report, nil
}

func gitOutput(repository string, arguments ...string) ([]byte, error) {
	command := exec.Command("git", append([]string{"-C", repository}, arguments...)...)
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w", arguments[0], err)
	}
	return output, nil
}

func splitNUL(raw []byte) []string {
	parts := bytes.Split(raw, []byte{0})
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) != 0 {
			result = append(result, string(part))
		}
	}
	sort.Strings(result)
	return result
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
