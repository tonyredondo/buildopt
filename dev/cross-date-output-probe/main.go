// Command cross-date-output-probe applies controlled mutations to one real
// ZIP output and records which reviewed semantic contracts expose each change.
package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tonyredondo/buildopt/internal/outputequivalence"
)

const releaseInfoEntry = "META-INF/groovy-release-info.properties"

type probeResult struct {
	SchemaVersion            string     `json:"schemaVersion"`
	Artifact                 artifact   `json:"artifact"`
	PreviousContractSHA256   string     `json:"previousContractSha256"`
	ReviewedContractSHA256   string     `json:"reviewedContractSha256"`
	ObservedProperties       properties `json:"observedProperties"`
	ControlledDateBoundary   comparison `json:"controlledDateBoundary"`
	UndeclaredPropertyDrift  comparison `json:"undeclaredPropertyDrift"`
	NonMetadataPayloadDrift  comparison `json:"nonMetadataPayloadDrift"`
	ProductAttributableFails int        `json:"productAttributableFailures"`
}

type artifact struct {
	RelativePath string `json:"relativePath"`
	RawSHA256    string `json:"rawSha256"`
	EntryCount   int    `json:"entryCount"`
}

type properties struct {
	BuildDate             string `json:"buildDate"`
	BuildTime             string `json:"buildTime"`
	ImplementationVersion string `json:"implementationVersion"`
}

type comparison struct {
	Mutation                string `json:"mutation"`
	Entry                   string `json:"entry"`
	OriginalRawSHA256       string `json:"originalRawSha256"`
	MutatedRawSHA256        string `json:"mutatedRawSha256"`
	PreviousContractMatched *bool  `json:"previousContractMatched,omitempty"`
	ReviewedContractMatched bool   `json:"reviewedContractMatched"`
	OriginalSemanticSHA256  string `json:"originalSemanticSha256"`
	MutatedSemanticSHA256   string `json:"mutatedSemanticSha256"`
}

type zipEntry struct {
	header zip.FileHeader
	body   []byte
}

func main() {
	if len(os.Args) != 6 {
		fmt.Fprintln(os.Stderr, "usage: cross-date-output-probe ROOT RELATIVE_ZIP PREVIOUS_CONTRACT REVIEWED_CONTRACT OUTPUT")
		os.Exit(64)
	}
	root, relative, previousPath, reviewedPath, outputPath :=
		os.Args[1], filepath.ToSlash(os.Args[2]), os.Args[3], os.Args[4], os.Args[5]
	result, err := run(root, relative, previousPath, reviewedPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cross-date output probe: %v\n", err)
		os.Exit(1)
	}
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "encode cross-date output probe: %v\n", err)
		os.Exit(1)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(outputPath, raw, 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "write cross-date output probe: %v\n", err)
		os.Exit(1)
	}
}

func run(root, relative, previousPath, reviewedPath string) (probeResult, error) {
	if !filepath.IsAbs(root) || filepath.IsAbs(relative) || relative == "." || strings.HasPrefix(relative, "../") {
		return probeResult{}, errors.New("artifact location is unsafe")
	}
	previousRaw, err := os.ReadFile(previousPath)
	if err != nil {
		return probeResult{}, fmt.Errorf("read previous contract: %w", err)
	}
	reviewedRaw, err := os.ReadFile(reviewedPath)
	if err != nil {
		return probeResult{}, fmt.Errorf("read reviewed contract: %w", err)
	}
	previous, err := outputequivalence.Parse(previousRaw)
	if err != nil {
		return probeResult{}, fmt.Errorf("parse previous contract: %w", err)
	}
	reviewed, err := outputequivalence.Parse(reviewedRaw)
	if err != nil {
		return probeResult{}, fmt.Errorf("parse reviewed contract: %w", err)
	}
	if err := validateExtension(previous, reviewed); err != nil {
		return probeResult{}, err
	}

	filename := filepath.Join(root, filepath.FromSlash(relative))
	entries, err := readZIP(filename)
	if err != nil {
		return probeResult{}, err
	}
	metadataIndex := entryIndex(entries, releaseInfoEntry)
	if metadataIndex < 0 {
		return probeResult{}, errors.New("Groovy release-properties entry is unavailable")
	}
	values, err := requiredProperties(entries[metadataIndex].body)
	if err != nil {
		return probeResult{}, err
	}
	payloadIndex := -1
	for index := range entries {
		if strings.HasSuffix(entries[index].header.Name, ".class") && !entries[index].header.FileInfo().IsDir() {
			payloadIndex = index
			break
		}
	}
	if payloadIndex < 0 {
		return probeResult{}, errors.New("non-metadata class payload is unavailable")
	}

	originalRawSHA, err := fileSHA(filename)
	if err != nil {
		return probeResult{}, err
	}
	originalSemantic, err := semanticSHA(root, relative, reviewed)
	if err != nil {
		return probeResult{}, err
	}
	previousOriginal, err := semanticSHA(root, relative, previous)
	if err != nil {
		return probeResult{}, err
	}

	dateValue := "01-Jan-2000"
	if values.BuildDate == dateValue {
		dateValue = "02-Jan-2000"
	}
	dateEntries := cloneEntries(entries)
	dateEntries[metadataIndex].body, err = replaceProperty(dateEntries[metadataIndex].body, "BuildDate", dateValue)
	if err != nil {
		return probeResult{}, err
	}
	dateComparison, err := compareVariant(root, relative, dateEntries, reviewed, originalRawSHA, originalSemantic)
	if err != nil {
		return probeResult{}, err
	}
	dateComparison.Mutation = "REPLACE_DECLARED_BUILDDATE"
	dateComparison.Entry = releaseInfoEntry
	previousDate, err := variantSemanticSHA(root, relative, dateEntries, previous)
	if err != nil {
		return probeResult{}, err
	}
	previousMatched := previousOriginal == previousDate
	dateComparison.PreviousContractMatched = &previousMatched

	propertyEntries := cloneEntries(entries)
	propertyEntries[metadataIndex].body, err = replaceProperty(
		propertyEntries[metadataIndex].body, "ImplementationVersion", values.ImplementationVersion+"-buildopt-drift",
	)
	if err != nil {
		return probeResult{}, err
	}
	propertyComparison, err := compareVariant(root, relative, propertyEntries, reviewed, originalRawSHA, originalSemantic)
	if err != nil {
		return probeResult{}, err
	}
	propertyComparison.Mutation = "CHANGE_UNDECLARED_IMPLEMENTATIONVERSION"
	propertyComparison.Entry = releaseInfoEntry

	payloadEntries := cloneEntries(entries)
	payloadEntries[payloadIndex].body = append(append([]byte(nil), payloadEntries[payloadIndex].body...), 0)
	payloadComparison, err := compareVariant(root, relative, payloadEntries, reviewed, originalRawSHA, originalSemantic)
	if err != nil {
		return probeResult{}, err
	}
	payloadComparison.Mutation = "CHANGE_NON_METADATA_CLASS_PAYLOAD"
	payloadComparison.Entry = payloadEntries[payloadIndex].header.Name

	return probeResult{
		SchemaVersion:           "buildopt.evidence/cross-date-output-probe/v1",
		Artifact:                artifact{RelativePath: relative, RawSHA256: originalRawSHA, EntryCount: len(entries)},
		PreviousContractSHA256:  outputequivalence.SHA256(previousRaw),
		ReviewedContractSHA256:  outputequivalence.SHA256(reviewedRaw),
		ObservedProperties:      values,
		ControlledDateBoundary:  dateComparison,
		UndeclaredPropertyDrift: propertyComparison,
		NonMetadataPayloadDrift: payloadComparison,
	}, nil
}

func validateExtension(previous, reviewed outputequivalence.Contract) error {
	if len(previous.Rules) != 1 || len(reviewed.Rules) != 1 ||
		previous.Rules[0].Pattern != reviewed.Rules[0].Pattern ||
		previous.Rules[0].Mode != outputequivalence.ModeCanonicalZIP ||
		reviewed.Rules[0].Mode != outputequivalence.ModeCanonicalZIP ||
		len(previous.Rules[0].VolatileProperties) != 1 ||
		len(reviewed.Rules[0].VolatileProperties) != 1 ||
		previous.Rules[0].VolatileProperties[0].Entry != releaseInfoEntry ||
		reviewed.Rules[0].VolatileProperties[0].Entry != releaseInfoEntry ||
		!equalStrings(previous.Rules[0].VolatileProperties[0].Keys, []string{"BuildTime"}) ||
		!equalStrings(reviewed.Rules[0].VolatileProperties[0].Keys, []string{"BuildDate", "BuildTime"}) {
		return errors.New("reviewed contract is not the exact BuildDate extension")
	}
	return nil
}

func compareVariant(root, relative string, entries []zipEntry, contract outputequivalence.Contract, originalRaw, originalSemantic string) (comparison, error) {
	mutatedRaw, mutatedSemantic, err := variantDigests(root, relative, entries, contract)
	if err != nil {
		return comparison{}, err
	}
	return comparison{
		OriginalRawSHA256: originalRaw, MutatedRawSHA256: mutatedRaw,
		ReviewedContractMatched: originalSemantic == mutatedSemantic,
		OriginalSemanticSHA256:  originalSemantic, MutatedSemanticSHA256: mutatedSemantic,
	}, nil
}

func variantSemanticSHA(root, relative string, entries []zipEntry, contract outputequivalence.Contract) (string, error) {
	_, semantic, err := variantDigests(root, relative, entries, contract)
	return semantic, err
}

func variantDigests(root, relative string, entries []zipEntry, contract outputequivalence.Contract) (string, string, error) {
	temp, err := os.MkdirTemp("", "buildopt-cross-date-variant.")
	if err != nil {
		return "", "", err
	}
	defer os.RemoveAll(temp)
	filename := filepath.Join(temp, filepath.FromSlash(relative))
	if err := writeZIP(filename, entries); err != nil {
		return "", "", err
	}
	raw, err := fileSHA(filename)
	if err != nil {
		return "", "", err
	}
	semantic, err := semanticSHA(temp, relative, contract)
	return raw, semantic, err
}

func semanticSHA(root, relative string, contract outputequivalence.Contract) (string, error) {
	digest, count, err := outputequivalence.HashOutputs(root, []string{relative}, &contract)
	if err != nil {
		return "", err
	}
	if count != 1 {
		return "", fmt.Errorf("expected one required output, got %d", count)
	}
	return digest, nil
}

func readZIP(filename string) ([]zipEntry, error) {
	reader, err := zip.OpenReader(filename)
	if err != nil {
		return nil, fmt.Errorf("open real ZIP output: %w", err)
	}
	defer reader.Close()
	entries := make([]zipEntry, 0, len(reader.File))
	for _, file := range reader.File {
		input, err := file.Open()
		if err != nil {
			return nil, err
		}
		body, readErr := io.ReadAll(input)
		closeErr := input.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		entries = append(entries, zipEntry{header: file.FileHeader, body: body})
	}
	return entries, nil
}

func writeZIP(filename string, entries []zipEntry) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return err
	}
	output, err := os.Create(filename)
	if err != nil {
		return err
	}
	writer := zip.NewWriter(output)
	for _, entry := range entries {
		header := entry.header
		header.CRC32, header.CompressedSize, header.CompressedSize64 = 0, 0, 0
		header.UncompressedSize, header.UncompressedSize64 = 0, 0
		target, err := writer.CreateHeader(&header)
		if err != nil {
			_ = output.Close()
			return err
		}
		if _, err := target.Write(entry.body); err != nil {
			_ = output.Close()
			return err
		}
	}
	if err := writer.Close(); err != nil {
		_ = output.Close()
		return err
	}
	return output.Close()
}

func requiredProperties(raw []byte) (properties, error) {
	result := properties{}
	var err error
	if result.BuildDate, err = propertyValue(raw, "BuildDate"); err != nil {
		return properties{}, err
	}
	if result.BuildTime, err = propertyValue(raw, "BuildTime"); err != nil {
		return properties{}, err
	}
	if result.ImplementationVersion, err = propertyValue(raw, "ImplementationVersion"); err != nil {
		return properties{}, err
	}
	return result, nil
}

func propertyValue(raw []byte, key string) (string, error) {
	prefix := []byte(key + "=")
	found := ""
	count := 0
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		if bytes.HasPrefix(line, prefix) {
			found = string(bytes.TrimSuffix(line[len(prefix):], []byte{'\r'}))
			count++
		}
	}
	if count != 1 || found == "" {
		return "", fmt.Errorf("property %s is missing or ambiguous", key)
	}
	return found, nil
}

func replaceProperty(raw []byte, key, value string) ([]byte, error) {
	prefix := []byte(key + "=")
	lines := bytes.Split(raw, []byte{'\n'})
	count := 0
	for index, line := range lines {
		if bytes.HasPrefix(line, prefix) {
			ending := []byte(nil)
			if bytes.HasSuffix(line, []byte{'\r'}) {
				ending = []byte{'\r'}
			}
			lines[index] = append(append(append([]byte(nil), prefix...), value...), ending...)
			count++
		}
	}
	if count != 1 {
		return nil, fmt.Errorf("property %s is missing or ambiguous", key)
	}
	return bytes.Join(lines, []byte{'\n'}), nil
}

func entryIndex(entries []zipEntry, name string) int {
	for index := range entries {
		if entries[index].header.Name == name {
			return index
		}
	}
	return -1
}

func cloneEntries(entries []zipEntry) []zipEntry {
	cloned := make([]zipEntry, len(entries))
	for index := range entries {
		cloned[index] = zipEntry{header: entries[index].header, body: append([]byte(nil), entries[index].body...)}
	}
	return cloned
}

func fileSHA(filename string) (string, error) {
	raw, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}

func equalStrings(left, right []string) bool {
	leftCopy, rightCopy := append([]string(nil), left...), append([]string(nil), right...)
	sort.Strings(leftCopy)
	sort.Strings(rightCopy)
	return strings.Join(leftCopy, "\x00") == strings.Join(rightCopy, "\x00")
}
