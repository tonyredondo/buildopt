// Package outputequivalence implements explicit, owner-reviewed semantic
// comparison for build outputs that cannot be byte-reproducible by design.
// Outputs remain byte-exact unless a narrow rule in a checked contract matches.
package outputequivalence

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	// SchemaVersion identifies the only semantic-output contract accepted by
	// the current POC.
	SchemaVersion = "buildopt.poc/output-equivalence/v1"

	ModeRepositoryRootText = "REPOSITORY_ROOT_TEXT"
	ModeCanonicalZIP       = "CANONICAL_ZIP"

	maximumContractBytes  = 1 << 20
	maximumOutputFiles    = 250000
	maximumTextBytes      = 64 << 20
	maximumZIPEntryBytes  = 512 << 20
	maximumZIPTotalBytes  = 2 << 30
	maximumZIPEntryCount  = 250000
	canonicalRootToken    = "${BUILDOPT_REPOSITORY_ROOT}"
	canonicalPropertyText = "${BUILDOPT_OWNER_VOLATILE_VALUE}"
)

var propertyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)

// Contract contains only explicitly reviewed exceptions to byte identity.
type Contract struct {
	SchemaVersion        string `json:"schemaVersion"`
	Rules                []Rule `json:"rules"`
	ReviewRequired       bool   `json:"reviewRequired"`
	ActivationAutomatic  bool   `json:"activationAutomatic"`
	ProductionAuthorized bool   `json:"productionAuthorized"`
}

// Rule selects required outputs by repository-relative glob. Canonical ZIP
// rules may additionally identify individual Java-properties keys whose value
// is intentionally volatile; every other byte of those entries remains bound.
type Rule struct {
	Pattern            string               `json:"pattern"`
	Mode               string               `json:"mode"`
	VolatileProperties []VolatileProperties `json:"volatileProperties,omitempty"`
}

// VolatileProperties names the only properties that may differ inside one ZIP
// entry. The entry itself and all non-declared lines remain exact.
type VolatileProperties struct {
	Entry string   `json:"entry"`
	Keys  []string `json:"keys"`
}

// Parse decodes a bounded strict JSON contract and rejects broad or ambiguous
// rules before any build output is inspected.
func Parse(raw []byte) (Contract, error) {
	if len(raw) == 0 || len(raw) > maximumContractBytes {
		return Contract{}, errors.New("output-equivalence contract size is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var contract Contract
	if err := decoder.Decode(&contract); err != nil {
		return Contract{}, fmt.Errorf("decode output-equivalence contract: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Contract{}, errors.New("output-equivalence contract has trailing content")
	}
	if err := Validate(contract); err != nil {
		return Contract{}, err
	}
	return contract, nil
}

// Validate checks the repository-independent contract shape. A later output
// traversal also requires every rule to match at least one declared output.
func Validate(contract Contract) error {
	if contract.SchemaVersion != SchemaVersion || !contract.ReviewRequired ||
		contract.ActivationAutomatic || contract.ProductionAuthorized ||
		len(contract.Rules) == 0 || len(contract.Rules) > 256 {
		return errors.New("output-equivalence contract boundary is invalid")
	}
	seenPatterns := map[string]bool{}
	for _, rule := range contract.Rules {
		if !safeGlob(rule.Pattern) || seenPatterns[rule.Pattern] {
			return errors.New("output-equivalence rule pattern is unsafe or repeated")
		}
		seenPatterns[rule.Pattern] = true
		switch rule.Mode {
		case ModeRepositoryRootText:
			if len(rule.VolatileProperties) != 0 {
				return errors.New("text relocation cannot declare archive properties")
			}
		case ModeCanonicalZIP:
			if err := validateVolatileProperties(rule.VolatileProperties); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported output-equivalence mode %q", rule.Mode)
		}
	}
	return nil
}

func validateVolatileProperties(properties []VolatileProperties) error {
	if len(properties) > 64 {
		return errors.New("too many volatile archive-property declarations")
	}
	seenEntries := map[string]bool{}
	for _, declaration := range properties {
		if !safeArchivePath(declaration.Entry) || seenEntries[declaration.Entry] ||
			len(declaration.Keys) == 0 || len(declaration.Keys) > 64 {
			return errors.New("volatile archive-property declaration is invalid")
		}
		seenEntries[declaration.Entry] = true
		seenKeys := map[string]bool{}
		for _, key := range declaration.Keys {
			if !propertyKeyPattern.MatchString(key) || seenKeys[key] {
				return errors.New("volatile archive-property key is invalid")
			}
			seenKeys[key] = true
		}
	}
	return nil
}

// SHA256 returns the lowercase digest of the exact reviewed contract bytes.
func SHA256(raw []byte) string {
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

// HashOutputs computes a stable manifest of all required outputs. Files not
// matched by a rule remain byte-exact. Every rule must be exercised exactly
// once per matching file set, and overlapping rules are rejected.
func HashOutputs(repositoryRoot string, requiredPatterns []string, contract *Contract) (string, int, error) {
	files, err := collectRequiredFiles(repositoryRoot, requiredPatterns)
	if err != nil {
		return "", 0, err
	}
	ruleMatches := make([]int, 0)
	if contract != nil {
		if err := Validate(*contract); err != nil {
			return "", 0, err
		}
		ruleMatches = make([]int, len(contract.Rules))
	}
	manifest := sha256.New()
	for _, relative := range files {
		ruleIndex := -1
		if contract != nil {
			for index, rule := range contract.Rules {
				if matchGlob(rule.Pattern, relative) {
					if ruleIndex >= 0 {
						return "", 0, fmt.Errorf("required output %s matches overlapping equivalence rules", relative)
					}
					ruleIndex = index
					ruleMatches[index]++
				}
			}
		}
		mode := "EXACT_BYTES"
		var digest string
		if ruleIndex < 0 {
			digest, err = hashRegularFile(filepath.Join(repositoryRoot, filepath.FromSlash(relative)))
		} else {
			rule := contract.Rules[ruleIndex]
			mode = rule.Mode
			digest, err = canonicalFileDigest(repositoryRoot, relative, rule)
		}
		if err != nil {
			return "", 0, fmt.Errorf("canonicalize required output %s: %w", relative, err)
		}
		_, _ = fmt.Fprintf(manifest, "%s  %s  %s\n", digest, mode, relative)
	}
	for index, count := range ruleMatches {
		if count == 0 {
			return "", 0, fmt.Errorf("output-equivalence rule %q matched no required output", contract.Rules[index].Pattern)
		}
	}
	return hex.EncodeToString(manifest.Sum(nil)), len(files), nil
}

func collectRequiredFiles(repositoryRoot string, patterns []string) ([]string, error) {
	if len(patterns) == 0 || len(patterns) > 256 {
		return nil, errors.New("required output patterns are missing or unbounded")
	}
	for _, pattern := range patterns {
		if !safeGlob(pattern) {
			return nil, fmt.Errorf("required output pattern %q is unsafe", pattern)
		}
	}
	files := make([]string, 0)
	err := filepath.WalkDir(repositoryRoot, func(candidate string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(repositoryRoot, candidate)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.IsDir() {
			if relative == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !matchesAny(patterns, relative) {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("required output is not a regular file: %s", relative)
		}
		files = append(files, relative)
		if len(files) > maximumOutputFiles {
			return errors.New("required output file count exceeds the safety bound")
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("inspect required outputs: %w", err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil, errors.New("measurement produced no required outputs")
	}
	return files, nil
}

func canonicalFileDigest(repositoryRoot, relative string, rule Rule) (string, error) {
	filename := filepath.Join(repositoryRoot, filepath.FromSlash(relative))
	switch rule.Mode {
	case ModeRepositoryRootText:
		return hashRelocatableText(filename, repositoryRoot)
	case ModeCanonicalZIP:
		return hashCanonicalZIP(filename, rule.VolatileProperties)
	default:
		return "", errors.New("unsupported output-equivalence mode")
	}
}

func hashRelocatableText(filename, repositoryRoot string) (string, error) {
	info, err := os.Stat(filename)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maximumTextBytes {
		return "", errors.New("relocatable text must be a bounded regular file")
	}
	raw, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(raw) || bytes.IndexByte(raw, 0) >= 0 {
		return "", errors.New("relocatable text must be NUL-free UTF-8")
	}
	rootForms := [][]byte{[]byte(repositoryRoot)}
	forwardRoot := filepath.ToSlash(repositoryRoot)
	if forwardRoot != repositoryRoot {
		rootForms = append(rootForms, []byte(forwardRoot))
	}
	replacements := 0
	canonical := append([]byte(nil), raw...)
	for _, root := range rootForms {
		count := bytes.Count(canonical, root)
		if count == 0 {
			continue
		}
		replacements += count
		canonical = bytes.ReplaceAll(canonical, root, []byte(canonicalRootToken))
	}
	if replacements == 0 {
		return "", errors.New("relocatable text does not contain the isolated repository root")
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}

func hashCanonicalZIP(filename string, properties []VolatileProperties) (string, error) {
	archive, err := zip.OpenReader(filename)
	if err != nil {
		return "", fmt.Errorf("open ZIP: %w", err)
	}
	defer archive.Close()
	if len(archive.File) == 0 || len(archive.File) > maximumZIPEntryCount {
		return "", errors.New("ZIP entry count is invalid")
	}
	propertyRules := map[string][]string{}
	for _, declaration := range properties {
		propertyRules[declaration.Entry] = append([]string(nil), declaration.Keys...)
	}
	type entryDigest struct {
		name, digest string
		mode         fs.FileMode
		directory    bool
		size         uint64
	}
	entries := make([]entryDigest, 0, len(archive.File))
	seen := map[string]bool{}
	matchedProperties := map[string]bool{}
	var total uint64
	for _, entry := range archive.File {
		if !safeArchivePath(entry.Name) || seen[entry.Name] || entry.Flags&1 != 0 || entry.UncompressedSize64 > maximumZIPEntryBytes {
			return "", fmt.Errorf("ZIP entry %q is unsafe, repeated, encrypted, or too large", entry.Name)
		}
		seen[entry.Name] = true
		total += entry.UncompressedSize64
		if total > maximumZIPTotalBytes {
			return "", errors.New("ZIP uncompressed size exceeds the safety bound")
		}
		reader, err := entry.Open()
		if err != nil {
			return "", err
		}
		payload, readErr := io.ReadAll(io.LimitReader(reader, maximumZIPEntryBytes+1))
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil || len(payload) > maximumZIPEntryBytes || uint64(len(payload)) != entry.UncompressedSize64 {
			return "", errors.New("ZIP entry payload is incomplete or oversized")
		}
		if keys, ok := propertyRules[entry.Name]; ok {
			payload, err = canonicalizeProperties(payload, keys)
			if err != nil {
				return "", fmt.Errorf("canonicalize ZIP entry %s: %w", entry.Name, err)
			}
			matchedProperties[entry.Name] = true
		}
		digest := sha256.Sum256(payload)
		entries = append(entries, entryDigest{
			name: entry.Name, digest: hex.EncodeToString(digest[:]),
			mode: entry.Mode().Perm(), directory: entry.FileInfo().IsDir(), size: uint64(len(payload)),
		})
	}
	for entry := range propertyRules {
		if !matchedProperties[entry] {
			return "", fmt.Errorf("declared volatile-properties entry %q is absent", entry)
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
	manifest := sha256.New()
	for _, entry := range entries {
		_, _ = fmt.Fprintf(manifest, "%s  %06o  %t  %d  %s\n", entry.digest, entry.mode, entry.directory, entry.size, entry.name)
	}
	return hex.EncodeToString(manifest.Sum(nil)), nil
}

func canonicalizeProperties(raw []byte, keys []string) ([]byte, error) {
	if !utf8.Valid(raw) || bytes.IndexByte(raw, 0) >= 0 {
		return nil, errors.New("properties payload must be NUL-free UTF-8")
	}
	wanted := map[string]bool{}
	for _, key := range keys {
		wanted[key] = true
	}
	seen := map[string]bool{}
	lines := bytes.SplitAfter(raw, []byte("\n"))
	for index, lineWithEnding := range lines {
		ending := []byte{}
		line := lineWithEnding
		if bytes.HasSuffix(line, []byte("\n")) {
			ending = []byte("\n")
			line = line[:len(line)-1]
		}
		if bytes.HasSuffix(line, []byte("\r")) {
			ending = []byte("\r\n")
			line = line[:len(line)-1]
		}
		for key := range wanted {
			prefixes := [][]byte{[]byte(key + "="), []byte(key + ":")}
			for _, prefix := range prefixes {
				if !bytes.HasPrefix(line, prefix) {
					continue
				}
				if seen[key] || bytes.HasSuffix(line, []byte{'\\'}) {
					return nil, fmt.Errorf("volatile property %q is repeated or continued", key)
				}
				seen[key] = true
				line = append(append([]byte(nil), prefix...), []byte(canonicalPropertyText)...)
			}
		}
		lines[index] = append(line, ending...)
	}
	for key := range wanted {
		if !seen[key] {
			return nil, fmt.Errorf("volatile property %q is absent", key)
		}
	}
	return bytes.Join(lines, nil), nil
}

func hashRegularFile(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	digest := sha256.New()
	_, copyErr := io.Copy(digest, file)
	closeErr := file.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func safeGlob(candidate string) bool {
	if candidate == "" || strings.Contains(candidate, `\`) || strings.Contains(candidate, "//") ||
		strings.ContainsAny(candidate, "\r\n\x00") || strings.HasPrefix(candidate, "/") ||
		path.Clean(candidate) != candidate || candidate == "." || candidate == ".." ||
		strings.HasPrefix(candidate, "../") {
		return false
	}
	for _, segment := range strings.Split(candidate, "/") {
		if segment == "**" {
			continue
		}
		if _, err := path.Match(segment, ""); err != nil {
			return false
		}
	}
	return true
}

func safeArchivePath(candidate string) bool {
	trimmed := strings.TrimSuffix(candidate, "/")
	return trimmed != "" && !strings.Contains(candidate, `\`) && !strings.ContainsAny(candidate, "\r\n\x00") &&
		!strings.HasPrefix(candidate, "/") && path.Clean(trimmed) == trimmed && trimmed != "." &&
		trimmed != ".." && !strings.HasPrefix(trimmed, "../")
}

func matchesAny(patterns []string, candidate string) bool {
	for _, pattern := range patterns {
		if matchGlob(pattern, candidate) {
			return true
		}
	}
	return false
}

func matchGlob(pattern, candidate string) bool {
	patternParts := strings.Split(pattern, "/")
	candidateParts := strings.Split(candidate, "/")
	var match func(int, int) bool
	match = func(patternIndex, candidateIndex int) bool {
		if patternIndex == len(patternParts) {
			return candidateIndex == len(candidateParts)
		}
		if patternParts[patternIndex] == "**" {
			for next := candidateIndex; next <= len(candidateParts); next++ {
				if match(patternIndex+1, next) {
					return true
				}
			}
			return false
		}
		if candidateIndex == len(candidateParts) {
			return false
		}
		matched, err := path.Match(patternParts[patternIndex], candidateParts[candidateIndex])
		return err == nil && matched && match(patternIndex+1, candidateIndex+1)
	}
	return match(0, 0)
}
