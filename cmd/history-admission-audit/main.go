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
	SchemaVersion      string       `json:"schemaVersion"`
	TargetRevision     string       `json:"targetRevision"`
	SnapshotSHA256     string       `json:"snapshotSha256"`
	GraphSHA256        string       `json:"graphSha256,omitempty"`
	ManifestSHA256     string       `json:"manifestSha256,omitempty"`
	BaseRevision       string       `json:"baseRevision,omitempty"`
	HeadChangesSHA256  string       `json:"headChangesSha256,omitempty"`
	Entrypoints        []string     `json:"entrypoints"`
	ExpectedOwners     []string     `json:"expectedOwners"`
	ExpectedFamily     string       `json:"expectedFamily"`
	HistoryWindow      int          `json:"historyWindowCommits"`
	CompatibleCommits  int          `json:"compatibleCommits"`
	MinimumCompatible  int          `json:"minimumCompatibleCommits"`
	Decision           string       `json:"decision"`
	Reason             string       `json:"reason"`
	GradleExecuted     bool         `json:"gradleExecuted"`
	RepositoryNameRule bool         `json:"repositoryNameRule"`
	Mode               string       `json:"mode,omitempty"`
	Groups             []auditGroup `json:"groups,omitempty"`
	Rows               []auditRow   `json:"rows"`
}

type auditGroup struct {
	Owners           []string `json:"owners"`
	Family           string   `json:"family"`
	AffectedProjects int      `json:"affectedProjects"`
	TotalProjects    int      `json:"totalProjects"`
	Commits          []string `json:"commits"`
}

type auditOptions struct {
	repository, snapshotPath, snapshotSHA, graphPath, graphSHA, manifestPath, manifestSHA string
	repositoryID, pipelineClass, target, family                                           string
	baseRevision, headChangesPath, headChangesSHA                                         string
	entrypoints, owners                                                                   []string
	maximum, minimum                                                                      int
	inventory                                                                             bool
}

func main() {
	var options auditOptions
	var entrypoints, owners stringList
	flag.StringVar(&options.repository, "repository", "", "path to the public Git repository")
	flag.StringVar(&options.snapshotPath, "snapshot", "", "path to the retained discovery snapshot")
	flag.StringVar(&options.snapshotSHA, "snapshot-sha256", "", "required SHA-256 binding for the snapshot")
	flag.StringVar(&options.graphPath, "graph", "", "path to a reviewed declared graph")
	flag.StringVar(&options.graphSHA, "graph-sha256", "", "required raw SHA-256 binding for the graph")
	flag.StringVar(&options.manifestPath, "manifest", "", "path to the graph's manifest")
	flag.StringVar(&options.manifestSHA, "manifest-sha256", "", "required raw SHA-256 binding for the manifest")
	flag.StringVar(&options.repositoryID, "repository-id", "", "manifest repository identity")
	flag.StringVar(&options.pipelineClass, "pipeline-class", "", "manifest pipeline identity")
	flag.StringVar(&options.target, "target", "", "target revision")
	flag.StringVar(&options.baseRevision, "base", "", "public base revision for a retained synthetic head")
	flag.StringVar(&options.headChangesPath, "head-changes", "", "newline-delimited paths for a retained synthetic head")
	flag.StringVar(&options.headChangesSHA, "head-changes-sha256", "", "required raw SHA-256 binding for head changes")
	flag.StringVar(&options.family, "expected-family", "", "expected change family")
	flag.Var(&entrypoints, "entrypoint", "observed entrypoint (repeatable)")
	flag.Var(&owners, "expected-owner", "expected owner (repeatable)")
	flag.IntVar(&options.maximum, "maximum", 64, "maximum first-parent commits")
	flag.IntVar(&options.minimum, "minimum", 5, "minimum compatible commits")
	flag.BoolVar(&options.inventory, "inventory", false, "inventory every exact owner/family group")
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
	usesSnapshot := options.snapshotPath != "" && options.snapshotSHA != ""
	usesGraph := options.graphPath != "" && options.graphSHA != "" && options.manifestPath != "" &&
		options.manifestSHA != "" && options.repositoryID != "" && options.pipelineClass != ""
	usesSyntheticHead := options.baseRevision != "" && options.headChangesPath != "" && options.headChangesSHA != ""
	if options.repository == "" || usesSnapshot == usesGraph || options.target == "" ||
		(options.baseRevision != "" || options.headChangesPath != "" || options.headChangesSHA != "") != usesSyntheticHead ||
		(usesSnapshot && len(options.entrypoints) == 0) ||
		(!options.inventory && (len(options.owners) == 0 || options.family == "")) ||
		(options.inventory && (len(options.owners) != 0 || options.family != "")) ||
		options.maximum < 1 || options.maximum > 1024 || options.minimum < 1 || options.minimum > options.maximum {
		return auditReport{}, errors.New("complete bounded audit arguments are required")
	}
	snapshot, snapshotSHA, graphSHA, manifestSHA, err := loadGraphFacts(options)
	if err != nil {
		return auditReport{}, err
	}
	if len(options.entrypoints) == 0 {
		for _, entrypoint := range snapshot.Entrypoints {
			options.entrypoints = append(options.entrypoints, entrypoint.Name)
		}
	}
	sort.Strings(options.owners)
	historyTarget := options.target
	historyMaximum := options.maximum
	var headPaths []string
	var headChangesDigest string
	if usesSyntheticHead {
		historyTarget = options.baseRevision
		historyMaximum--
		if historyMaximum < 1 {
			return auditReport{}, errors.New("synthetic-head history window is too small")
		}
		raw, digestText, readErr := readBoundFile(options.headChangesPath, options.headChangesSHA, "head changes")
		if readErr != nil {
			return auditReport{}, readErr
		}
		headPaths = splitLines(raw)
		headChangesDigest = digestText
		if len(headPaths) == 0 {
			return auditReport{}, errors.New("synthetic head changes are empty")
		}
	}
	commitsRaw, err := gitOutput(options.repository, "rev-list", "--first-parent", "--max-count="+strconv.Itoa(historyMaximum), historyTarget)
	if err != nil {
		return auditReport{}, err
	}
	commits := strings.Fields(string(commitsRaw))
	if usesSyntheticHead {
		commits = append([]string{options.target}, commits...)
	}
	if len(commits) == 0 || len(commits) > options.maximum {
		return auditReport{}, errors.New("bounded first-parent history is unavailable")
	}
	report := auditReport{SchemaVersion: schemaVersion, TargetRevision: options.target, SnapshotSHA256: snapshotSHA,
		GraphSHA256: graphSHA, ManifestSHA256: manifestSHA,
		BaseRevision: options.baseRevision, HeadChangesSHA256: headChangesDigest,
		Entrypoints: append([]string(nil), options.entrypoints...), ExpectedOwners: append([]string(nil), options.owners...),
		ExpectedFamily: options.family, HistoryWindow: len(commits), MinimumCompatible: options.minimum,
		Decision: "REJECT", Reason: "INSUFFICIENT_GRAPH_COMPATIBLE_HISTORY", Rows: make([]auditRow, 0, len(commits))}
	groups := map[string]*auditGroup{}
	if options.inventory {
		report.Mode, report.Decision, report.Reason = "INVENTORY", "INVENTORY_COMPLETE", "ALL_EXACT_GRAPH_GROUPS_CLASSIFIED"
	}
	for _, commit := range commits {
		paths := headPaths
		var pathErr error
		if commit != options.target || !usesSyntheticHead {
			var pathsRaw []byte
			pathsRaw, pathErr = gitOutput(options.repository, "diff-tree", "--root", "--no-commit-id", "--name-only", "--no-renames", "-r", "-z", commit, "--")
			paths = splitNUL(pathsRaw)
		}
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
		if options.inventory {
			row.Decision, row.Reason = "CLASSIFIED", "EXACT_GRAPH_OWNER_AND_FAMILY"
			key := strings.Join(classification.Owners, "\x00") + "\x01" + classification.Family
			group := groups[key]
			if group == nil {
				group = &auditGroup{Owners: append([]string(nil), classification.Owners...), Family: classification.Family,
					AffectedProjects: len(classification.AffectedProjects), TotalProjects: len(snapshot.Projects)}
				groups[key] = group
			} else if group.AffectedProjects != len(classification.AffectedProjects) || group.TotalProjects != len(snapshot.Projects) {
				return auditReport{}, errors.New("one structural group produced inconsistent affected closures")
			}
			group.Commits = append(group.Commits, commit)
		} else if !equalStrings(classification.Owners, options.owners) {
			row.Reason = "OWNER_MISMATCH"
		} else if classification.Family != options.family {
			row.Reason = "FAMILY_MISMATCH"
		} else {
			row.Decision, row.Reason = "COMPATIBLE", "EXACT_GRAPH_OWNER_AND_FAMILY"
			report.CompatibleCommits++
		}
		report.Rows = append(report.Rows, row)
	}
	if options.inventory {
		for _, group := range groups {
			report.Groups = append(report.Groups, *group)
		}
		sort.Slice(report.Groups, func(left, right int) bool {
			leftKey := strings.Join(report.Groups[left].Owners, "\x00") + report.Groups[left].Family
			rightKey := strings.Join(report.Groups[right].Owners, "\x00") + report.Groups[right].Family
			return leftKey < rightKey
		})
	} else if report.CompatibleCommits >= report.MinimumCompatible {
		report.Decision, report.Reason = "ADMIT", "MINIMUM_GRAPH_COMPATIBLE_HISTORY_MET"
	}
	return report, nil
}

func loadGraphFacts(options auditOptions) (buildimpact.DiscoverySnapshot, string, string, string, error) {
	if options.snapshotPath != "" {
		raw, digestText, err := readBoundFile(options.snapshotPath, options.snapshotSHA, "snapshot")
		if err != nil {
			return buildimpact.DiscoverySnapshot{}, "", "", "", err
		}
		snapshot, err := buildimpact.ParseObservedDiscoverySnapshot(raw, options.entrypoints)
		if err != nil {
			return buildimpact.DiscoverySnapshot{}, "", "", "", fmt.Errorf("parse snapshot: %w", err)
		}
		return snapshot, digestText, "", "", nil
	}
	manifestRaw, manifestDigest, err := readBoundFile(options.manifestPath, options.manifestSHA, "manifest")
	if err != nil {
		return buildimpact.DiscoverySnapshot{}, "", "", "", err
	}
	manifest, err := buildimpact.ParseManifest(manifestRaw, options.repositoryID, options.pipelineClass)
	if err != nil {
		return buildimpact.DiscoverySnapshot{}, "", "", "", fmt.Errorf("parse manifest: %w", err)
	}
	graphRaw, graphDigest, err := readBoundFile(options.graphPath, options.graphSHA, "graph")
	if err != nil {
		return buildimpact.DiscoverySnapshot{}, "", "", "", err
	}
	graph, err := buildimpact.ParseDeclaredGraph(graphRaw, manifest)
	if err != nil {
		return buildimpact.DiscoverySnapshot{}, "", "", "", fmt.Errorf("parse graph: %w", err)
	}
	return historyadmission.SnapshotFromDeclaredGraph(graph.Graph), "", graphDigest, manifestDigest, nil
}

func readBoundFile(path, expected, label string) ([]byte, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read %s: %w", label, err)
	}
	digest := sha256.Sum256(raw)
	digestText := hex.EncodeToString(digest[:])
	if expected != digestText {
		return nil, "", fmt.Errorf("%s SHA-256 binding does not match", label)
	}
	return raw, digestText, nil
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

func splitLines(raw []byte) []string {
	result := strings.Fields(string(raw))
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
