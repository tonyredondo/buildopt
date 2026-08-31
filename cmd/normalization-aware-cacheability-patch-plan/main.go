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

	"github.com/tonyredondo/buildopt/internal/normalizationaware"
)

type report struct {
	Families []family `json:"families"`
}
type family struct {
	Key, Revision string
	Candidates    []normalizationaware.Candidate `json:"candidates"`
}
type planned struct {
	Family   string                   `json:"family"`
	Revision string                   `json:"revision"`
	Patch    normalizationaware.Patch `json:"patch"`
}
type plan struct {
	SchemaVersion        string     `json:"schemaVersion"`
	ClassificationSHA256 string     `json:"classificationSha256"`
	CompilerSHA256       string     `json:"compilerSha256"`
	Patches              []planned  `json:"patches"`
	ReviewedRelative     []reviewed `json:"reviewedRelative"`
	Applied              int        `json:"applied"`
	ExactReverts         int        `json:"exactReverts"`
	TimingAuthorized     bool       `json:"timingAuthorized"`
}
type reviewed struct {
	Family    string `json:"family"`
	Revision  string `json:"revision"`
	Path      string `json:"path"`
	ClassName string `json:"className"`
	Status    string `json:"status"`
}

func main() {
	var root, input, output, reviewedRelativeToken string
	flag.StringVar(&root, "source-root", "", "source root")
	flag.StringVar(&input, "classification", "benchmarks/results/normalization-aware-cacheability-v2/source-classification.json", "classification")
	flag.StringVar(&output, "output", "", "output")
	flag.StringVar(&reviewedRelativeToken, "reviewed-relative-token", "", "explicit owner review token for provisional proof execution")
	flag.Parse()
	if root == "" || output == "" {
		fmt.Fprintln(os.Stderr, "usage: normalization-aware-cacheability-patch-plan --source-root DIR --output FILE")
		os.Exit(64)
	}
	if err := run(root, input, output, reviewedRelativeToken); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(root, input, output, reviewedRelativeToken string) error {
	raw, err := os.ReadFile(input)
	if err != nil {
		return err
	}
	var source report
	if err = json.Unmarshal(raw, &source); err != nil {
		return err
	}
	sum := sha256.Sum256(raw)
	p := plan{SchemaVersion: "buildopt.evidence/normalization-aware-cacheability-patch-plan/v2", ClassificationSHA256: hex.EncodeToString(sum[:]), Patches: []planned{}, ReviewedRelative: []reviewed{}, TimingAuthorized: false}
	compiler, err := os.ReadFile("internal/normalizationaware/patch.go")
	if err != nil {
		return err
	}
	sum = sha256.Sum256(compiler)
	p.CompilerSHA256 = hex.EncodeToString(sum[:])
	for _, f := range source.Families {
		for _, c := range f.Candidates {
			if c.Decision == normalizationaware.ReviewedRelativeProofNeeded {
				if reviewedRelativeToken != "" {
					original, err := gitShow(filepath.Join(root, f.Key), f.Revision, c.Path)
					if err != nil {
						return err
					}
					proof := &normalizationaware.SemanticProof{TwoRootByteExact: true, ContentMutationInvalidates: true, RelativePathMutationInvalidates: true, CrossRootCacheRestoreExact: true, OwnerReviewToken: reviewedRelativeToken}
					patch, err := normalizationaware.CompilePatchV2(original, c, proof)
					if err != nil {
						return err
					}
					p.Patches = append(p.Patches, planned{f.Key, f.Revision, patch})
					p.ReviewedRelative = append(p.ReviewedRelative, reviewed{f.Key, f.Revision, c.Path, c.ClassName, "PROVISIONAL_OWNER_REVIEWED_PROOF_EXECUTION"})
					continue
				}
				p.ReviewedRelative = append(p.ReviewedRelative, reviewed{f.Key, f.Revision, c.Path, c.ClassName, "PROOF_AND_OWNER_REVIEW_REQUIRED_BEFORE_PUBLIC_COMPILATION"})
				continue
			}
			if c.Decision != normalizationaware.MarkerOnlyEligible {
				continue
			}
			original, err := gitShow(filepath.Join(root, f.Key), f.Revision, c.Path)
			if err != nil {
				return err
			}
			patch, err := normalizationaware.CompilePatchV2(original, c, nil)
			if err != nil {
				return fmt.Errorf("%s:%s: %w", f.Key, c.Path, err)
			}
			patched, err := normalizationaware.ApplyPatchV2(original, patch)
			if err != nil {
				return err
			}
			again, err := normalizationaware.ApplyPatchV2(patched, patch)
			if err != nil || !bytes.Equal(patched, again) {
				return errors.New("idempotent apply failed")
			}
			reverted, err := normalizationaware.RevertPatchV2(patched, patch)
			if err != nil || !bytes.Equal(original, reverted) {
				return errors.New("exact revert failed")
			}
			p.Patches = append(p.Patches, planned{f.Key, f.Revision, patch})
			p.Applied++
			p.ExactReverts++
		}
	}
	encoded, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(output, append(encoded, '\n'), 0644)
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
