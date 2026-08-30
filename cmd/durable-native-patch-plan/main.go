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

	"github.com/tonyredondo/buildopt/internal/durablenative"
)

type opportunityReport struct {
	Families []family `json:"families"`
}
type family struct {
	Key, Revision string
	Candidates    []durablenative.Candidate `json:"candidates"`
}
type plannedPatch struct {
	Family   string              `json:"family"`
	Revision string              `json:"revision"`
	Patch    durablenative.Patch `json:"patch"`
}
type plan struct {
	SchemaVersion     string         `json:"schemaVersion"`
	OpportunitySHA256 string         `json:"opportunitySha256"`
	CompilerSHA256    string         `json:"compilerSha256"`
	Patches           []plannedPatch `json:"patches"`
	Applied           int            `json:"applied"`
	ExactReverts      int            `json:"exactReverts"`
	TimingAuthorized  bool           `json:"timingAuthorized"`
}

func main() {
	var sourceRoot, opportunities, output string
	flag.StringVar(&sourceRoot, "source-root", "", "directory containing repositories")
	flag.StringVar(&opportunities, "opportunities", "benchmarks/results/durable-native-optimization-v1/opportunity-report.json", "DNO-002 report")
	flag.StringVar(&output, "output", "", "output JSON")
	flag.Parse()
	if sourceRoot == "" || output == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: durable-native-patch-plan --source-root DIR --output FILE")
		os.Exit(64)
	}
	if err := run(sourceRoot, opportunities, output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(sourceRoot, opportunities, output string) error {
	raw, err := os.ReadFile(opportunities)
	if err != nil {
		return err
	}
	var source opportunityReport
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&source); err != nil {
		return err
	}
	opportunityDigest := sha256.Sum256(raw)
	result := plan{SchemaVersion: "buildopt.evidence/durable-native-patch-plan/v1", OpportunitySHA256: hex.EncodeToString(opportunityDigest[:]), Patches: []plannedPatch{}}
	compilerRaw, err := os.ReadFile("internal/durablenative/patch.go")
	if err != nil {
		return err
	}
	compilerDigest := sha256.Sum256(compilerRaw)
	result.CompilerSHA256 = hex.EncodeToString(compilerDigest[:])
	for _, family := range source.Families {
		for _, candidate := range family.Candidates {
			original, err := gitShow(filepath.Join(sourceRoot, family.Key), family.Revision, candidate.Path)
			if err != nil {
				return err
			}
			patch, err := durablenative.CompilePatch(original, candidate)
			if err != nil {
				return fmt.Errorf("%s:%s: %w", family.Key, candidate.Path, err)
			}
			if err := durablenative.ValidatePatch(patch); err != nil {
				return err
			}
			patched, err := durablenative.ApplyPatch(original, patch)
			if err != nil {
				return err
			}
			patchedAgain, err := durablenative.ApplyPatch(patched, patch)
			if err != nil || !bytes.Equal(patched, patchedAgain) {
				return errors.New("idempotent apply failed")
			}
			reverted, err := durablenative.RevertPatch(patched, patch)
			if err != nil || !bytes.Equal(original, reverted) {
				return errors.New("exact revert failed")
			}
			result.Patches = append(result.Patches, plannedPatch{Family: family.Key, Revision: family.Revision, Patch: patch})
			result.Applied++
			result.ExactReverts++
		}
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(output, append(encoded, '\n'), 0o644)
}

func gitShow(repo, revision, path string) ([]byte, error) {
	command := exec.Command("git", "-C", repo, "show", revision+":"+path)
	command.Env = append(os.Environ(), "GIT_NO_LAZY_FETCH=1")
	out, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("git show %s:%s: %w", revision, path, err)
	}
	return out, nil
}
