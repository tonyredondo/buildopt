// Package durablenative detects source-bound Gradle corrections without using
// repository or task names as decision inputs.
package durablenative

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"sort"
	"strings"
)

var (
	classPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?m)(?:public\s+)?(?:abstract\s+)?class\s+([A-Za-z_][A-Za-z0-9_]*)\s+extends\s+DefaultTask\b`),
		regexp.MustCompile(`(?m)(?:abstract\s+)?class\s+([A-Za-z_][A-Za-z0-9_]*)\s*:\s*DefaultTask\s*\(`),
	}
	inputAnnotation  = regexp.MustCompile(`(?m)@(?:get:)?(?:Input|InputFile|InputFiles|InputDirectory|Classpath|CompileClasspath)\b`)
	outputAnnotation = regexp.MustCompile(`(?m)@(?:get:)?(?:OutputFile|OutputFiles|OutputDirectory|OutputDirectories)\b`)
)

// Candidate is a complete custom-task contract eligible for the first DNO
// compiler. SourceSHA256 binds any later patch to the audited bytes.
type Candidate struct {
	Path         string `json:"path"`
	ClassName    string `json:"className"`
	Language     string `json:"language"`
	InputCount   int    `json:"inputCount"`
	OutputCount  int    `json:"outputCount"`
	SourceSHA256 string `json:"sourceSha256"`
}

// ScanSource returns a candidate only when all frozen task-contract evidence
// is present. It intentionally makes no semantic claim beyond that contract.
func ScanSource(path string, source []byte) (Candidate, bool) {
	text := string(source)
	if !strings.Contains(text, "DefaultTask") || !strings.Contains(text, "@TaskAction") ||
		strings.Contains(text, "@CacheableTask") || strings.Contains(text, "@DisableCachingByDefault") {
		return Candidate{}, false
	}
	className := ""
	for _, pattern := range classPatterns {
		if match := pattern.FindStringSubmatch(text); len(match) == 2 {
			className = match[1]
			break
		}
	}
	inputs := inputAnnotation.FindAllString(text, -1)
	outputs := outputAnnotation.FindAllString(text, -1)
	if className == "" || len(inputs) == 0 || len(outputs) == 0 {
		return Candidate{}, false
	}
	digest := sha256.Sum256(source)
	return Candidate{
		Path: path, ClassName: className, Language: language(path),
		InputCount: len(inputs), OutputCount: len(outputs), SourceSHA256: hex.EncodeToString(digest[:]),
	}, true
}

// SortCandidates makes report generation independent of Git traversal order.
func SortCandidates(candidates []Candidate) {
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Path < candidates[j].Path })
}

func language(path string) string {
	switch {
	case strings.HasSuffix(path, ".java"):
		return "JAVA"
	case strings.HasSuffix(path, ".groovy"):
		return "GROOVY"
	case strings.HasSuffix(path, ".kt"):
		return "KOTLIN"
	default:
		return "UNKNOWN"
	}
}
