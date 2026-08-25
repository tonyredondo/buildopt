// Command current-longitudinal-cohorts freezes and independently validates the
// AF-014B public first-parent cohorts before any timing is observed.
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
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	contractSchema = "buildopt.specs/current-longitudinal-cohorts/v1"
	resultSchema   = "buildopt.poc/current-longitudinal-cohorts/v1"
)

var revisionPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)
var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type contract struct {
	SchemaVersion  string             `json:"schemaVersion"`
	WorkItem       string             `json:"workItem"`
	Selection      selection          `json:"selection"`
	Repositories   []repositoryPolicy `json:"repositories"`
	Exclusions     []string           `json:"exclusions"`
	ChangeShapes   []string           `json:"changeShapes"`
	ResourceBudget resourceBudget     `json:"resourceBudget"`
	Boundaries     boundaries         `json:"boundaries"`
}

type selection struct {
	History                 string `json:"history"`
	PrimaryCount            int    `json:"primaryCount"`
	ReserveCount            int    `json:"reserveCount"`
	TimingAllowedAtFreeze   bool   `json:"timingAllowedAtFreeze"`
	Replacement             string `json:"replacement"`
	MinimumComparableTarget int    `json:"minimumComparableTarget"`
}

type repositoryPolicy struct {
	Key             string   `json:"key"`
	RepositoryID    string   `json:"repositoryId"`
	RemoteURL       string   `json:"remoteUrl"`
	Branch          string   `json:"branch"`
	JDKToolchain    string   `json:"jdkToolchain"`
	Workflow        []string `json:"workflow"`
	RequiredOutputs []string `json:"requiredOutputs"`
}

type resourceBudget struct {
	Platform                             string `json:"platform"`
	MinimumFreeDiskBytesBeforeRepository int64  `json:"minimumFreeDiskBytesBeforeRepository"`
	SequentialRepositories               bool   `json:"sequentialRepositories"`
	RemoveTemporaryState                 bool   `json:"removeReproducibleTemporaryStateAfterEvidence"`
}

type boundaries struct {
	ProofOfConcept        bool   `json:"proofOfConcept"`
	PerformanceClaim      bool   `json:"performanceClaim"`
	ProductionAuthorized  bool   `json:"productionAuthorized"`
	SoakRequired          bool   `json:"soakRequired"`
	DesignPartnerRequired bool   `json:"designPartnerRequired"`
	TestOptimization      string `json:"testOptimization"`
}

type result struct {
	SchemaVersion  string             `json:"schemaVersion"`
	WorkItem       string             `json:"workItem"`
	FrozenAt       string             `json:"frozenAt"`
	Outcome        string             `json:"outcome"`
	Contract       contractReference  `json:"contract"`
	Selection      selection          `json:"selection"`
	Repositories   []repositoryCohort `json:"repositories"`
	Exclusions     []string           `json:"exclusions"`
	ResourceBudget resourceBudget     `json:"resourceBudget"`
	Boundaries     boundaries         `json:"boundaries"`
}

type contractReference struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type repositoryCohort struct {
	Policy     repositoryPolicy `json:"policy"`
	RemoteHead string           `json:"remoteHead"`
	Anchor     commit           `json:"anchor"`
	Primary    []commit         `json:"primary"`
	Reserves   []commit         `json:"reserves"`
}

type commit struct {
	Ordinal            int      `json:"ordinal"`
	Revision           string   `json:"revision"`
	ParentRevision     string   `json:"parentRevision"`
	TreeRevision       string   `json:"treeRevision"`
	CommittedAt        string   `json:"committedAt"`
	ChangedPaths       []string `json:"changedPaths"`
	ChangedPathsSHA256 string   `json:"changedPathsSha256"`
	ChangeShape        string   `json:"changeShape"`
}

func main() {
	freeze := flag.Bool("freeze", false, "freeze public cohorts into the result path")
	tempRoot := flag.String("temp-root", "", "temporary root used only while freezing")
	flag.Parse()
	if flag.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "usage: current-longitudinal-cohorts [--freeze --temp-root PATH] CONTRACT RESULT")
		os.Exit(64)
	}
	contractPath, resultPath := flag.Arg(0), flag.Arg(1)
	specRaw, spec, err := readContract(contractPath)
	if err != nil {
		fatal(err)
	}
	if *freeze {
		if *tempRoot == "" {
			fatal(errors.New("--temp-root is required with --freeze"))
		}
		captured, err := freezeCohorts(specRaw, spec, *tempRoot)
		if err != nil {
			fatal(err)
		}
		encoded, err := json.MarshalIndent(captured, "", "  ")
		if err != nil {
			fatal(err)
		}
		encoded = append(encoded, '\n')
		if err := os.WriteFile(resultPath, encoded, 0o600); err != nil {
			fatal(err)
		}
	}
	reportRaw, report, err := readResult(resultPath)
	if err != nil {
		fatal(err)
	}
	if err := validate(specRaw, spec, reportRaw, report); err != nil {
		fatal(err)
	}
	fmt.Printf("AF-014B cohorts frozen: %d repositories, %d primary revisions, %d ordered reserves\n", len(report.Repositories), len(report.Repositories)*spec.Selection.PrimaryCount, len(report.Repositories)*spec.Selection.ReserveCount)
}

func readContract(path string) ([]byte, contract, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, contract{}, err
	}
	var value contract
	if err := decodeStrict(raw, &value); err != nil {
		return nil, contract{}, fmt.Errorf("decode contract: %w", err)
	}
	return raw, value, nil
}

func readResult(path string) ([]byte, result, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, result{}, err
	}
	var value result
	if err := decodeStrict(raw, &value); err != nil {
		return nil, result{}, fmt.Errorf("decode result: %w", err)
	}
	return raw, value, nil
}

func decodeStrict(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("multiple JSON values")
	}
	return nil
}

func freezeCohorts(specRaw []byte, spec contract, root string) (result, error) {
	if err := validateContract(spec); err != nil {
		return result{}, err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return result{}, err
	}
	digest := sha256.Sum256(specRaw)
	report := result{
		SchemaVersion:  resultSchema,
		WorkItem:       "AF-014B",
		FrozenAt:       time.Now().UTC().Format(time.RFC3339),
		Outcome:        "CURRENT_LONGITUDINAL_COHORTS_FROZEN",
		Contract:       contractReference{Path: "specs/poc-current-longitudinal-cohorts-v1.json", SHA256: hex.EncodeToString(digest[:])},
		Selection:      spec.Selection,
		Exclusions:     append([]string(nil), spec.Exclusions...),
		ResourceBudget: spec.ResourceBudget,
		Boundaries:     spec.Boundaries,
	}
	for _, policy := range spec.Repositories {
		cohort, err := freezeRepository(root, policy, spec.Selection)
		if err != nil {
			return result{}, fmt.Errorf("freeze %s: %w", policy.Key, err)
		}
		report.Repositories = append(report.Repositories, cohort)
	}
	return report, nil
}

func freezeRepository(root string, policy repositoryPolicy, limits selection) (repositoryCohort, error) {
	bare := filepath.Join(root, policy.Key+".git")
	if err := os.RemoveAll(bare); err != nil {
		return repositoryCohort{}, err
	}
	if _, err := runGit("init", "--bare", bare); err != nil {
		return repositoryCohort{}, err
	}
	ref := "refs/remotes/origin/" + policy.Branch
	depth := limits.PrimaryCount + limits.ReserveCount + 32
	if _, err := runGit("-C", bare, "fetch", "--quiet", "--filter=blob:none", fmt.Sprintf("--depth=%d", depth), policy.RemoteURL, "+refs/heads/"+policy.Branch+":"+ref); err != nil {
		return repositoryCohort{}, err
	}
	want := limits.PrimaryCount + limits.ReserveCount + 1
	output, err := runGit("-C", bare, "rev-list", "--first-parent", fmt.Sprintf("--max-count=%d", want), ref)
	if err != nil {
		return repositoryCohort{}, err
	}
	revisions := nonemptyLines(output)
	if len(revisions) != want {
		return repositoryCohort{}, fmt.Errorf("need %d first-parent revisions, found %d", want, len(revisions))
	}
	for left, right := 0, len(revisions)-1; left < right; left, right = left+1, right-1 {
		revisions[left], revisions[right] = revisions[right], revisions[left]
	}
	commits := make([]commit, 0, want)
	for index, revision := range revisions {
		item, err := inspectCommit(bare, index, revision)
		if err != nil {
			return repositoryCohort{}, err
		}
		commits = append(commits, item)
	}
	return repositoryCohort{
		Policy:     policy,
		RemoteHead: commits[len(commits)-1].Revision,
		Anchor:     commits[0],
		Primary:    append([]commit(nil), commits[1:1+limits.PrimaryCount]...),
		Reserves:   append([]commit(nil), commits[1+limits.PrimaryCount:]...),
	}, nil
}

func inspectCommit(bare string, ordinal int, revision string) (commit, error) {
	meta, err := runGit("-C", bare, "show", "-s", "--format=%P%n%T%n%cI", revision)
	if err != nil {
		return commit{}, err
	}
	lines := strings.Split(strings.TrimSpace(meta), "\n")
	if len(lines) != 3 {
		return commit{}, errors.New("unexpected commit metadata")
	}
	parents := strings.Fields(lines[0])
	if len(parents) == 0 {
		return commit{}, errors.New("cohort commit has no parent")
	}
	pathsRaw, err := runGit("-C", bare, "diff-tree", "--no-commit-id", "--name-only", "-r", parents[0], revision)
	if err != nil {
		return commit{}, err
	}
	paths := nonemptyLines(pathsRaw)
	sort.Strings(paths)
	pathDigest := sha256.Sum256([]byte(strings.Join(paths, "\n") + "\n"))
	return commit{
		Ordinal:            ordinal,
		Revision:           revision,
		ParentRevision:     parents[0],
		TreeRevision:       lines[1],
		CommittedAt:        lines[2],
		ChangedPaths:       paths,
		ChangedPathsSHA256: hex.EncodeToString(pathDigest[:]),
		ChangeShape:        classify(paths),
	}, nil
}

func validate(specRaw []byte, spec contract, reportRaw []byte, report result) error {
	if err := validateContract(spec); err != nil {
		return err
	}
	digest := sha256.Sum256(specRaw)
	if report.SchemaVersion != resultSchema || report.WorkItem != "AF-014B" || report.Outcome != "CURRENT_LONGITUDINAL_COHORTS_FROZEN" ||
		report.Contract.Path != "specs/poc-current-longitudinal-cohorts-v1.json" || report.Contract.SHA256 != hex.EncodeToString(digest[:]) ||
		report.Selection != spec.Selection || !equalStrings(report.Exclusions, spec.Exclusions) || report.ResourceBudget != spec.ResourceBudget || report.Boundaries != spec.Boundaries ||
		len(report.Repositories) != len(spec.Repositories) {
		return errors.New("result metadata differs from the frozen contract")
	}
	if _, err := time.Parse(time.RFC3339, report.FrozenAt); err != nil {
		return errors.New("invalid freeze timestamp")
	}
	if bytes.Contains(bytes.ToLower(reportRaw), []byte("wallns")) || bytes.Contains(bytes.ToLower(reportRaw), []byte("saving")) || bytes.Contains(bytes.ToLower(reportRaw), []byte("duration")) {
		return errors.New("cohort manifest contains forbidden timing or value evidence")
	}
	for index, policy := range spec.Repositories {
		cohort := report.Repositories[index]
		if !equalPolicy(cohort.Policy, policy) || len(cohort.Primary) != spec.Selection.PrimaryCount || len(cohort.Reserves) != spec.Selection.ReserveCount || cohort.RemoteHead != cohort.Reserves[len(cohort.Reserves)-1].Revision {
			return fmt.Errorf("invalid cohort layout for %s", policy.Key)
		}
		chain := append([]commit{cohort.Anchor}, cohort.Primary...)
		chain = append(chain, cohort.Reserves...)
		seen := map[string]bool{}
		for ordinal, item := range chain {
			if item.Ordinal != ordinal || !revisionPattern.MatchString(item.Revision) || !revisionPattern.MatchString(item.ParentRevision) || !revisionPattern.MatchString(item.TreeRevision) || seen[item.Revision] {
				return fmt.Errorf("invalid revision identity at %s ordinal %d", policy.Key, ordinal)
			}
			seen[item.Revision] = true
			if ordinal > 0 && item.ParentRevision != chain[ordinal-1].Revision {
				return fmt.Errorf("non-contiguous first-parent chain at %s ordinal %d", policy.Key, ordinal)
			}
			if _, err := time.Parse(time.RFC3339, item.CommittedAt); err != nil {
				return fmt.Errorf("invalid commit timestamp at %s ordinal %d", policy.Key, ordinal)
			}
			paths := append([]string(nil), item.ChangedPaths...)
			sort.Strings(paths)
			if !equalStrings(paths, item.ChangedPaths) || item.ChangeShape != classify(paths) || !digestPattern.MatchString(item.ChangedPathsSHA256) {
				return fmt.Errorf("invalid changed-path evidence at %s ordinal %d", policy.Key, ordinal)
			}
			pathDigest := sha256.Sum256([]byte(strings.Join(paths, "\n") + "\n"))
			if item.ChangedPathsSHA256 != hex.EncodeToString(pathDigest[:]) {
				return fmt.Errorf("changed-path digest mismatch at %s ordinal %d", policy.Key, ordinal)
			}
		}
	}
	return nil
}

func validateContract(spec contract) error {
	if spec.SchemaVersion != contractSchema || spec.WorkItem != "AF-014B" || spec.Selection.History != "PUBLIC_FIRST_PARENT_CONTIGUOUS" || spec.Selection.PrimaryCount != 20 || spec.Selection.ReserveCount != 10 || spec.Selection.TimingAllowedAtFreeze || spec.Selection.Replacement != "NEXT_UNUSED_RESERVE_ONLY" || spec.Selection.MinimumComparableTarget != 15 || len(spec.Repositories) != 5 || len(spec.Exclusions) != 5 || len(spec.ChangeShapes) != 6 || !spec.ResourceBudget.SequentialRepositories || !spec.ResourceBudget.RemoveTemporaryState || spec.ResourceBudget.MinimumFreeDiskBytesBeforeRepository < 8<<30 || !spec.Boundaries.ProofOfConcept || spec.Boundaries.PerformanceClaim || spec.Boundaries.ProductionAuthorized || spec.Boundaries.SoakRequired || spec.Boundaries.DesignPartnerRequired || spec.Boundaries.TestOptimization != "OUT_OF_SCOPE" {
		return errors.New("invalid AF-014B contract")
	}
	for _, policy := range spec.Repositories {
		if policy.Key == "" || policy.RepositoryID == "" || !strings.HasPrefix(policy.RemoteURL, "https://github.com/") || policy.Branch == "" || policy.JDKToolchain == "" || len(policy.Workflow) == 0 || len(policy.RequiredOutputs) == 0 {
			return errors.New("incomplete repository policy")
		}
	}
	return nil
}

func classify(paths []string) string {
	var production, tests, build, docs, other bool
	for _, raw := range paths {
		path := strings.ToLower(raw)
		switch {
		case strings.HasSuffix(path, ".md") || strings.HasPrefix(path, "docs/") || strings.Contains(path, "/docs/"):
			docs = true
		case strings.Contains(path, "build.gradle") || strings.Contains(path, "settings.gradle") || strings.HasPrefix(path, "gradle/") || strings.Contains(path, "/gradle/") || strings.HasPrefix(path, "buildsrc/") || strings.Contains(path, "/buildsrc/") || strings.HasSuffix(path, ".versions.toml") || strings.HasSuffix(path, ".lockfile"):
			build = true
		case strings.Contains(path, "/src/test/") || strings.Contains(path, "/src/testfixtures/") || strings.HasPrefix(path, "src/test/"):
			tests = true
		case strings.Contains(path, "/src/main/") || strings.HasPrefix(path, "src/main/") || strings.HasSuffix(path, ".java") || strings.HasSuffix(path, ".kt") || strings.HasSuffix(path, ".groovy") || strings.HasSuffix(path, ".scala"):
			production = true
		default:
			other = true
		}
	}
	count := 0
	for _, present := range []bool{production, tests, build, docs, other} {
		if present {
			count++
		}
	}
	if count > 1 {
		return "MIXED"
	}
	if production {
		return "PRODUCTION_SOURCE"
	}
	if tests {
		return "TEST_SOURCE"
	}
	if build {
		return "BUILD_LOGIC_OR_DEPENDENCY"
	}
	if docs {
		return "DOCUMENTATION_ONLY"
	}
	return "OTHER"
}

func equalPolicy(first, second repositoryPolicy) bool {
	return first.Key == second.Key && first.RepositoryID == second.RepositoryID && first.RemoteURL == second.RemoteURL && first.Branch == second.Branch && first.JDKToolchain == second.JDKToolchain && equalStrings(first.Workflow, second.Workflow) && equalStrings(first.RequiredOutputs, second.RequiredOutputs)
}

func equalStrings(first, second []string) bool {
	if len(first) != len(second) {
		return false
	}
	for index := range first {
		if first[index] != second[index] {
			return false
		}
	}
	return true
}

func nonemptyLines(value string) []string {
	var result []string
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func runGit(args ...string) (string, error) {
	command := exec.Command("git", args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
