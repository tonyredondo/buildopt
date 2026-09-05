// Package strictdiagnostic selects the Configuration Cache report owned by a
// Gradle wrapper invocation without treating nested-build report inventory as
// root-build authority.
package strictdiagnostic

import (
	"bufio"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const reportReferencePrefix = "See the complete report at "

// Outcome is a typed root-report selection result.
type Outcome string

const (
	OutcomeCaptured           Outcome = "ROOT_REPORT_CAPTURED"
	OutcomeReferenceMissing   Outcome = "ROOT_REPORT_REFERENCE_MISSING"
	OutcomeReferenceAmbiguous Outcome = "ROOT_REPORT_REFERENCE_AMBIGUOUS"
	OutcomeOutsideRoot        Outcome = "ROOT_REPORT_OUTSIDE_EXPECTED_ROOT"
	OutcomeHarnessFailure     Outcome = "HARNESS_FAILURE"
)

// Selection records only portable paths relative to the expected report root.
type Selection struct {
	Outcome            Outcome  `json:"outcome"`
	ReferenceCount     int      `json:"referenceCount"`
	ReferenceDigests   []string `json:"referenceSha256"`
	References         []string `json:"-"`
	SelectedRelative   string   `json:"selectedRelativePath,omitempty"`
	SelectedAbsolute   string   `json:"selectedAbsolutePath,omitempty"`
	FailureDescription string   `json:"failureDescription,omitempty"`
}

// SelectRootReport parses the child log and validates its one report reference.
// Inventory discovered elsewhere is deliberately not an input to selection.
func SelectRootReport(log []byte, expectedRoot string) Selection {
	return selectRootReport(log, expectedRoot, false)
}

func selectRootReport(log []byte, expectedRoot string, deduplicate bool) Selection {
	references, err := reportReferences(log)
	if err != nil {
		return failure(OutcomeHarnessFailure, 0, nil, err)
	}
	if deduplicate {
		references = uniqueReferences(references)
	}
	if len(references) == 0 {
		return Selection{Outcome: OutcomeReferenceMissing, ReferenceDigests: []string{}, References: []string{}}
	}
	if len(references) != 1 {
		return Selection{
			Outcome:            OutcomeReferenceAmbiguous,
			ReferenceCount:     len(references),
			ReferenceDigests:   referenceDigests(references),
			References:         references,
			FailureDescription: "child log must contain exactly one root report reference",
		}
	}

	selected, err := reportPath(references[0])
	if err != nil {
		return failure(OutcomeOutsideRoot, 1, references, err)
	}
	relative, canonical, err := validateContainedRegularFile(expectedRoot, selected)
	if err != nil {
		return failure(OutcomeOutsideRoot, 1, references, err)
	}
	return Selection{
		Outcome:          OutcomeCaptured,
		ReferenceCount:   1,
		ReferenceDigests: referenceDigests(references),
		References:       references,
		SelectedRelative: filepath.ToSlash(relative),
		SelectedAbsolute: canonical,
	}
}

func uniqueReferences(references []string) []string {
	unique := make([]string, 0, len(references))
	seen := make(map[string]struct{}, len(references))
	for _, reference := range references {
		if _, exists := seen[reference]; exists {
			continue
		}
		seen[reference] = struct{}{}
		unique = append(unique, reference)
	}
	return unique
}

// AddProjectDirectory reports whether the runner must add its own -p argument.
func AddProjectDirectory(ownerArguments []string) bool {
	for _, argument := range ownerArguments {
		if argument == "-p" || argument == "--project-dir" || strings.HasPrefix(argument, "--project-dir=") {
			return false
		}
	}
	return true
}

func reportReferences(log []byte) ([]string, error) {
	scanner := bufio.NewScanner(strings.NewReader(string(log)))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	references := make([]string, 0, 1)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if index := strings.Index(line, reportReferencePrefix); index >= 0 {
			reference := strings.TrimSpace(line[index+len(reportReferencePrefix):])
			if reference != "" {
				references = append(references, reference)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan child log: %w", err)
	}
	return references, nil
}

func reportPath(reference string) (string, error) {
	parsed, err := url.Parse(reference)
	if err != nil || parsed.Scheme != "file" || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("report reference is not a local file URI")
	}
	path, err := url.PathUnescape(parsed.EscapedPath())
	if err != nil || !filepath.IsAbs(path) {
		return "", fmt.Errorf("report reference has no absolute local path")
	}
	return filepath.FromSlash(path), nil
}

func validateContainedRegularFile(root, selected string) (string, string, error) {
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", "", errors.New("expected report root cannot be resolved")
	}
	canonicalSelected, err := filepath.EvalSymlinks(selected)
	if err != nil {
		return "", "", errors.New("selected report cannot be resolved")
	}
	relative, err := filepath.Rel(canonicalRoot, canonicalSelected)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", "", errors.New("selected report is outside expected root")
	}
	if err := rejectSymlinkComponents(canonicalRoot, selected); err != nil {
		return "", "", err
	}
	info, err := os.Lstat(selected)
	if err != nil || !info.Mode().IsRegular() {
		return "", "", errors.New("selected report is not a regular file")
	}
	if filepath.Base(selected) != "configuration-cache-report.html" {
		return "", "", errors.New("selected file is not a Configuration Cache report")
	}
	return relative, canonicalSelected, nil
}

func rejectSymlinkComponents(root, selected string) error {
	relative, err := filepath.Rel(root, selected)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("selected report path is outside expected root")
	}
	current := root
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, inspectErr := os.Lstat(current)
		if inspectErr != nil {
			return errors.New("selected report path cannot be inspected")
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("selected report path contains a symlink")
		}
	}
	return nil
}

func failure(outcome Outcome, count int, references []string, err error) Selection {
	if references == nil {
		references = []string{}
	}
	return Selection{
		Outcome:            outcome,
		ReferenceCount:     count,
		ReferenceDigests:   referenceDigests(references),
		References:         references,
		FailureDescription: err.Error(),
	}
}

func referenceDigests(references []string) []string {
	digests := make([]string, 0, len(references))
	for _, reference := range references {
		digests = append(digests, fmt.Sprintf("%x", sha256.Sum256([]byte(reference))))
	}
	return digests
}
