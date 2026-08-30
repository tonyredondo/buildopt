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

	"github.com/tonyredondo/buildopt/internal/durablenative"
)

type subjectManifest struct {
	SchemaVersion string    `json:"schemaVersion"`
	Families      []subject `json:"families"`
}

type subject struct {
	Key        string `json:"key"`
	Repository string `json:"repository"`
	Revision   string `json:"revision"`
}

type familyReport struct {
	Key         string                    `json:"key"`
	Repository  string                    `json:"repository"`
	Revision    string                    `json:"revision"`
	Complete    bool                      `json:"complete"`
	Candidates  []durablenative.Candidate `json:"candidates"`
	Unavailable []string                  `json:"unavailableDetectors"`
}

type report struct {
	SchemaVersion       string         `json:"schemaVersion"`
	ContractRevision    string         `json:"contractRevision"`
	AnalyzerSHA256      string         `json:"analyzerSha256"`
	Families            []familyReport `json:"families"`
	CompleteFamilies    int            `json:"completeFamilies"`
	ActionFamilies      int            `json:"actionFamilies"`
	RequiredActionCount int            `json:"requiredActionFamilies"`
	Decision            string         `json:"decision"`
	TimingAuthorized    bool           `json:"timingAuthorized"`
}

func main() {
	var subjectsPath, sourceRoot, outputPath, contractRevision string
	flag.StringVar(&subjectsPath, "subjects", "specs/poc-durable-native-optimization-v1.subjects.json", "subject manifest")
	flag.StringVar(&sourceRoot, "source-root", "", "directory containing repositories named by subject key")
	flag.StringVar(&outputPath, "output", "", "output JSON path, or stdout when empty")
	flag.StringVar(&contractRevision, "contract-revision", "", "commit that froze DNO-001")
	flag.Parse()
	if sourceRoot == "" || contractRevision == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: durable-native-audit --source-root DIR --contract-revision SHA [--subjects FILE] [--output FILE]")
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
	if subjects.SchemaVersion != "buildopt.specs/poc-durable-native-optimization-subjects/v1" || len(subjects.Families) != 5 {
		return errors.New("subject manifest must contain exactly five DNO v1 families")
	}

	result := report{SchemaVersion: "buildopt.evidence/durable-native-opportunity/v1", ContractRevision: contractRevision, RequiredActionCount: 3}
	analyzer, err := analyzerDigest()
	if err != nil {
		return err
	}
	result.AnalyzerSHA256 = analyzer
	for _, family := range subjects.Families {
		repo := filepath.Join(sourceRoot, family.Key)
		if got, err := git(repo, "rev-parse", family.Revision+"^{commit}"); err != nil || strings.TrimSpace(got) != family.Revision {
			return fmt.Errorf("%s: frozen revision unavailable", family.Key)
		}
		paths, err := taskSourcePaths(repo, family.Revision)
		if err != nil {
			return fmt.Errorf("%s: %w", family.Key, err)
		}
		entry := familyReport{Key: family.Key, Repository: family.Repository, Revision: family.Revision, Complete: true, Candidates: []durablenative.Candidate{}, Unavailable: []string{"RECURRENT_CLEAN_ELISION", "DECLARED_GRAPH_SCOPE", "INTERNAL_COMPILE_CLASSPATH"}}
		for _, path := range paths {
			source, err := gitBytes(repo, "show", family.Revision+":"+path)
			if err != nil {
				return fmt.Errorf("%s:%s: %w", family.Key, path, err)
			}
			if candidate, ok := durablenative.ScanSource(path, source); ok {
				entry.Candidates = append(entry.Candidates, candidate)
			}
		}
		durablenative.SortCandidates(entry.Candidates)
		result.Families = append(result.Families, entry)
		result.CompleteFamilies++
		if len(entry.Candidates) > 0 {
			result.ActionFamilies++
		}
	}
	if result.CompleteFamilies == 5 && result.ActionFamilies >= result.RequiredActionCount {
		result.Decision = "PASS_SOURCE_ACTION_BREADTH"
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
	for _, path := range []string{"cmd/durable-native-audit/main.go", "internal/durablenative/scanner.go"} {
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
