package durablenative

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const cacheableAnnotation = "@org.gradle.api.tasks.CacheableTask"

// Patch is the complete byte transaction for one additive cacheability marker.
// It carries no repository-specific selector and can only apply to the audited
// source digest and unique class declaration.
type Patch struct {
	SchemaVersion        string `json:"schemaVersion"`
	Path                 string `json:"path"`
	ClassName            string `json:"className"`
	ExpectedSourceSHA256 string `json:"expectedSourceSha256"`
	PatchedSourceSHA256  string `json:"patchedSourceSha256"`
	InsertionOffset      int    `json:"insertionOffset"`
	InsertedText         string `json:"insertedText"`
}

// CompilePatch creates a digest-bound additive patch without mutating source.
func CompilePatch(source []byte, candidate Candidate) (Patch, error) {
	if digest(source) != candidate.SourceSHA256 {
		return Patch{}, errors.New("source digest drift")
	}
	rescanned, ok := ScanSource(candidate.Path, source)
	if !ok || rescanned.ClassName != candidate.ClassName {
		return Patch{}, errors.New("candidate contract no longer matches source")
	}
	declaration := declarationPattern(candidate.ClassName)
	matches := declaration.FindAllIndex(source, -1)
	if len(matches) != 1 {
		return Patch{}, fmt.Errorf("class declaration must be unique, got %d", len(matches))
	}
	lineStart := matches[0][0]
	for lineStart > 0 && source[lineStart-1] != '\n' {
		lineStart--
	}
	indentEnd := lineStart
	for indentEnd < len(source) && (source[indentEnd] == ' ' || source[indentEnd] == '\t') {
		indentEnd++
	}
	indent := string(source[lineStart:indentEnd])
	inserted := indent + cacheableAnnotation + "\n"
	patched := insert(source, lineStart, inserted)
	return Patch{
		SchemaVersion: "buildopt.patch/add-cacheable-task-marker/v1",
		Path:          candidate.Path, ClassName: candidate.ClassName,
		ExpectedSourceSHA256: candidate.SourceSHA256, PatchedSourceSHA256: digest(patched),
		InsertionOffset: lineStart, InsertedText: inserted,
	}, nil
}

// ApplyPatch applies once and returns identical bytes on repeated application.
func ApplyPatch(source []byte, patch Patch) ([]byte, error) {
	sourceDigest := digest(source)
	if sourceDigest == patch.PatchedSourceSHA256 {
		return append([]byte(nil), source...), nil
	}
	if sourceDigest != patch.ExpectedSourceSHA256 {
		return nil, errors.New("source digest drift")
	}
	if patch.InsertionOffset < 0 || patch.InsertionOffset > len(source) {
		return nil, errors.New("invalid insertion offset")
	}
	patched := insert(source, patch.InsertionOffset, patch.InsertedText)
	if digest(patched) != patch.PatchedSourceSHA256 {
		return nil, errors.New("patched digest mismatch")
	}
	return patched, nil
}

// RevertPatch removes only the compiled insertion and is also idempotent.
func RevertPatch(source []byte, patch Patch) ([]byte, error) {
	sourceDigest := digest(source)
	if sourceDigest == patch.ExpectedSourceSHA256 {
		return append([]byte(nil), source...), nil
	}
	if sourceDigest != patch.PatchedSourceSHA256 {
		return nil, errors.New("patched source drift")
	}
	start := patch.InsertionOffset
	end := start + len(patch.InsertedText)
	if start < 0 || end > len(source) || string(source[start:end]) != patch.InsertedText {
		return nil, errors.New("compiled insertion missing")
	}
	original := append(append([]byte(nil), source[:start]...), source[end:]...)
	if digest(original) != patch.ExpectedSourceSHA256 {
		return nil, errors.New("reverted digest mismatch")
	}
	return original, nil
}

func declarationPattern(className string) *regexp.Regexp {
	name := regexp.QuoteMeta(className)
	return regexp.MustCompile(`(?m)^(?:[ \t]*)(?:(?:public|protected|private|abstract|final|open)\s+)*class\s+` + name + `(?:\s+extends\s+DefaultTask\b|\s*:\s*DefaultTask\s*\()`)
}

func insert(source []byte, offset int, text string) []byte {
	result := make([]byte, 0, len(source)+len(text))
	result = append(result, source[:offset]...)
	result = append(result, text...)
	return append(result, source[offset:]...)
}

func digest(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// ValidatePatch rejects incomplete or non-canonical patch records.
func ValidatePatch(patch Patch) error {
	if patch.SchemaVersion != "buildopt.patch/add-cacheable-task-marker/v1" || patch.Path == "" || patch.ClassName == "" {
		return errors.New("incomplete patch identity")
	}
	if !strings.Contains(patch.InsertedText, cacheableAnnotation) || len(patch.ExpectedSourceSHA256) != 64 || len(patch.PatchedSourceSHA256) != 64 {
		return errors.New("invalid patch contract")
	}
	return nil
}
