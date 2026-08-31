package normalizationaware

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const cacheableMarker = "@org.gradle.api.tasks.CacheableTask"
const relativeMarker = "@org.gradle.api.tasks.PathSensitive(org.gradle.api.tasks.PathSensitivity.RELATIVE)"

type SemanticProof struct {
	TwoRootByteExact                bool   `json:"twoRootByteExact"`
	ContentMutationInvalidates      bool   `json:"contentMutationInvalidates"`
	RelativePathMutationInvalidates bool   `json:"relativePathMutationInvalidates"`
	CrossRootCacheRestoreExact      bool   `json:"crossRootCacheRestoreExact"`
	OwnerReviewToken                string `json:"ownerReviewToken"`
}

type Edit struct {
	Offset       int    `json:"offset"`
	InsertedText string `json:"insertedText"`
}
type Patch struct {
	SchemaVersion        string `json:"schemaVersion"`
	Action               string `json:"action"`
	Path                 string `json:"path"`
	ClassName            string `json:"className"`
	ExpectedSourceSHA256 string `json:"expectedSourceSha256"`
	PatchedSourceSHA256  string `json:"patchedSourceSha256"`
	Edits                []Edit `json:"edits"`
}

func CompilePatchV2(source []byte, candidate Candidate, proof *SemanticProof) (Patch, error) {
	if sourceDigest(source) != candidate.SourceSHA256 {
		return Patch{}, errors.New("source digest drift")
	}
	rescanned := ScanSourceV2(candidate.Path, source)
	var current *Candidate
	for i := range rescanned {
		if rescanned[i].ClassName == candidate.ClassName {
			if current != nil {
				return Patch{}, errors.New("ambiguous class declaration")
			}
			current = &rescanned[i]
		}
	}
	if current == nil || current.Decision != candidate.Decision {
		return Patch{}, errors.New("candidate contract drift")
	}
	classOffset, indent, err := classInsertion(source, candidate.ClassName)
	if err != nil {
		return Patch{}, err
	}
	patch := Patch{SchemaVersion: "buildopt.patch/normalization-aware-cacheability/v2", Path: candidate.Path, ClassName: candidate.ClassName, ExpectedSourceSHA256: candidate.SourceSHA256, Edits: []Edit{}}
	switch candidate.Decision {
	case MarkerOnlyEligible:
		patch.Action = "ADD_CACHEABLE_TASK_MARKER_V2"
	case ReviewedRelativeProofNeeded:
		if proof == nil || !proof.TwoRootByteExact || !proof.ContentMutationInvalidates || !proof.RelativePathMutationInvalidates || !proof.CrossRootCacheRestoreExact || strings.TrimSpace(proof.OwnerReviewToken) == "" {
			return Patch{}, errors.New("complete reviewed-relative semantic proof required")
		}
		missing := make([]FileInput, 0, 1)
		for _, input := range candidate.FileInputs {
			if len(input.Primary) == 0 {
				missing = append(missing, input)
			}
		}
		if len(missing) != 1 {
			return Patch{}, errors.New("reviewed-relative v1 requires exactly one unnormalized file input")
		}
		lineOffset, inputIndent, err := lineInsertion(source, missing[0].Declaration.StartLine)
		if err != nil {
			return Patch{}, err
		}
		patch.Action = "ADD_RELATIVE_PATH_NORMALIZATION_AND_CACHEABLE_MARKER_V1"
		patch.Edits = append(patch.Edits, Edit{Offset: lineOffset, InsertedText: inputIndent + relativeMarker + "\n"})
	default:
		return Patch{}, fmt.Errorf("decision %s is not compilable", candidate.Decision)
	}
	patch.Edits = append(patch.Edits, Edit{Offset: classOffset, InsertedText: indent + cacheableMarker + "\n"})
	sort.Slice(patch.Edits, func(i, j int) bool { return patch.Edits[i].Offset > patch.Edits[j].Offset })
	patched, err := applyEdits(source, patch.Edits)
	if err != nil {
		return Patch{}, err
	}
	patch.PatchedSourceSHA256 = sourceDigest(patched)
	return patch, nil
}

func ApplyPatchV2(source []byte, patch Patch) ([]byte, error) {
	d := sourceDigest(source)
	if d == patch.PatchedSourceSHA256 {
		return append([]byte(nil), source...), nil
	}
	if d != patch.ExpectedSourceSHA256 {
		return nil, errors.New("source digest drift")
	}
	patched, err := applyEdits(source, patch.Edits)
	if err != nil {
		return nil, err
	}
	if sourceDigest(patched) != patch.PatchedSourceSHA256 {
		return nil, errors.New("patched digest mismatch")
	}
	return patched, nil
}

func RevertPatchV2(source []byte, patch Patch) ([]byte, error) {
	d := sourceDigest(source)
	if d == patch.ExpectedSourceSHA256 {
		return append([]byte(nil), source...), nil
	}
	if d != patch.PatchedSourceSHA256 {
		return nil, errors.New("patched source drift")
	}
	result := append([]byte(nil), source...)
	edits := append([]Edit(nil), patch.Edits...)
	sort.Slice(edits, func(i, j int) bool { return edits[i].Offset < edits[j].Offset })
	for _, edit := range edits {
		start := edit.Offset
		end := start + len(edit.InsertedText)
		if start < 0 || end > len(result) || string(result[start:end]) != edit.InsertedText {
			return nil, errors.New("compiled insertion missing")
		}
		result = append(append([]byte(nil), result[:start]...), result[end:]...)
	}
	if sourceDigest(result) != patch.ExpectedSourceSHA256 {
		return nil, errors.New("reverted digest mismatch")
	}
	return result, nil
}

func applyEdits(source []byte, edits []Edit) ([]byte, error) {
	result := append([]byte(nil), source...)
	last := len(source) + 1
	for _, edit := range edits {
		if edit.Offset < 0 || edit.Offset > len(source) || edit.Offset >= last {
			return nil, errors.New("invalid or unordered edit")
		}
		result = append(append(append([]byte(nil), result[:edit.Offset]...), []byte(edit.InsertedText)...), result[edit.Offset:]...)
		last = edit.Offset
	}
	return result, nil
}
func classInsertion(source []byte, name string) (int, string, error) {
	p := regexp.MustCompile(`(?m)^(?P<i>[ \t]*)(?:(?:public|protected|private|abstract|final|open)\s+)*class\s+` + regexp.QuoteMeta(name) + `(?:\s+extends\s+DefaultTask\b|\s*:\s*DefaultTask\s*\()`)
	m := p.FindSubmatchIndex(source)
	if m == nil {
		return 0, "", errors.New("unique class declaration not found")
	}
	if len(p.FindAllIndex(source, -1)) != 1 {
		return 0, "", errors.New("ambiguous class declaration")
	}
	return m[0], string(source[m[2]:m[3]]), nil
}
func lineInsertion(source []byte, line int) (int, string, error) {
	if line < 1 {
		return 0, "", errors.New("invalid declaration line")
	}
	offset := 0
	for current := 1; current < line; current++ {
		n := strings.IndexByte(string(source[offset:]), '\n')
		if n < 0 {
			return 0, "", errors.New("declaration line unavailable")
		}
		offset += n + 1
	}
	end := offset
	for end < len(source) && (source[end] == ' ' || source[end] == '\t') {
		end++
	}
	return offset, string(source[offset:end]), nil
}
func sourceDigest(source []byte) string {
	sum := sha256.Sum256(source)
	return hex.EncodeToString(sum[:])
}
