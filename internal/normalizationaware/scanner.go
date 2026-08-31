// Package normalizationaware classifies Gradle task source declarations for
// the explicitly versioned normalization-aware cacheability v2 experiment.
package normalizationaware

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"sort"
	"strings"
)

type Decision string

const (
	MarkerOnlyEligible          Decision = "MARKER_ONLY_ELIGIBLE"
	ReviewedRelativeProofNeeded Decision = "REVIEWED_RELATIVE_PROOF_REQUIRED"
	ExplicitNonPortable         Decision = "EXPLICIT_NON_PORTABLE"
	AlreadyCacheable            Decision = "ALREADY_CACHEABLE"
	DisabledCaching             Decision = "DISABLED_CACHING"
	IncompleteAmbiguous         Decision = "INCOMPLETE_OR_AMBIGUOUS"
	NoAction                    Decision = "NO_ACTION"
)

type Span struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn"`
	EndLine     int `json:"endLine"`
	EndColumn   int `json:"endColumn"`
}

type FileInput struct {
	Kind          string   `json:"kind"`
	Binding       string   `json:"binding"`
	Declaration   Span     `json:"declarationSpan"`
	Primary       []string `json:"primaryNormalization"`
	Supplementary []string `json:"supplementaryNormalization"`
}

type Candidate struct {
	Path         string      `json:"path"`
	ClassName    string      `json:"className"`
	Language     string      `json:"language"`
	Declaration  Span        `json:"declarationSpan"`
	SourceSHA256 string      `json:"sourceSha256"`
	FileInputs   []FileInput `json:"fileInputs"`
	Decision     Decision    `json:"decision"`
	Reason       string      `json:"reason"`
}

var (
	classPattern     = regexp.MustCompile(`(?m)(?:public\s+)?(?:abstract\s+)?class\s+([A-Za-z_][A-Za-z0-9_]*)[^\n{]*(?:extends\s+DefaultTask\b|:\s*DefaultTask\s*\()`)
	fileInputPattern = regexp.MustCompile(`@(?:get:)?(InputFile|InputFiles|InputDirectory|InputDirectories)\b`)
	classpathPattern = regexp.MustCompile(`@(?:get:)?(Classpath|CompileClasspath)\b`)
	primaryPatterns  = []struct {
		pattern *regexp.Regexp
		value   string
	}{
		{regexp.MustCompile(`@(?:get:)?PathSensitive\s*\(\s*(?:PathSensitivity\.)?RELATIVE\s*\)`), "PATH_SENSITIVE_RELATIVE"},
		{regexp.MustCompile(`@(?:get:)?PathSensitive\s*\(\s*(?:PathSensitivity\.)?NAME_ONLY\s*\)`), "PATH_SENSITIVE_NAME_ONLY"},
		{regexp.MustCompile(`@(?:get:)?PathSensitive\s*\(\s*(?:PathSensitivity\.)?NONE\s*\)`), "PATH_SENSITIVE_NONE"},
		{regexp.MustCompile(`@(?:get:)?PathSensitive\s*\(\s*(?:PathSensitivity\.)?ABSOLUTE\s*\)`), "PATH_SENSITIVE_ABSOLUTE"},
		{regexp.MustCompile(`@(?:get:)?CompileClasspath\b`), "COMPILE_CLASSPATH"},
		{regexp.MustCompile(`@(?:get:)?Classpath\b`), "CLASSPATH"},
	}
	supplementaryPatterns = []struct {
		pattern *regexp.Regexp
		value   string
	}{
		{regexp.MustCompile(`@(?:get:)?NormalizeLineEndings\b`), "NORMALIZE_LINE_ENDINGS"},
		{regexp.MustCompile(`@(?:get:)?IgnoreEmptyDirectories\b`), "IGNORE_EMPTY_DIRECTORIES"},
	}
	leadingAnnotation = regexp.MustCompile(`^@(?:get:)?[A-Za-z_][A-Za-z0-9_.]*(?:\s*\([^)]*\))?\s*`)
	bindingPatterns   = []*regexp.Regexp{
		regexp.MustCompile(`(?m)\b((?:get|is)[A-Z][A-Za-z0-9_]*)\s*\(`),
		regexp.MustCompile(`(?m)\b(?:val|var)\s+([A-Za-z_][A-Za-z0-9_]*)\b`),
		regexp.MustCompile(`(?m)\b([A-Za-z_][A-Za-z0-9_]*)\s*(?:[;=]|$)`),
	}
)

// ScanSourceV2 emits one source-bound decision for every DefaultTask class.
// It uses only declaration facts and deliberately ignores repository and task labels.
func ScanSourceV2(path string, source []byte) []Candidate {
	text := string(source)
	matches := classPattern.FindAllStringSubmatchIndex(text, -1)
	result := make([]Candidate, 0, len(matches))
	digest := sha256.Sum256(source)
	for i, match := range matches {
		classStart := annotationBlockStart(text, match[0], 0)
		end := len(text)
		if i+1 < len(matches) {
			end = annotationBlockStart(text, matches[i+1][0], match[0])
		}
		body := text[classStart:end]
		candidate := Candidate{
			Path: path, ClassName: text[match[2]:match[3]], Language: language(path),
			Declaration: span(text, match[0], match[1]), SourceSHA256: hex.EncodeToString(digest[:]),
			FileInputs: []FileInput{},
		}
		candidate.FileInputs = scanFileInputs(text, match[0], end)
		candidate.Decision, candidate.Reason = decide(body, candidate.FileInputs)
		result = append(result, candidate)
	}
	return result
}

func scanFileInputs(text string, start, end int) []FileInput {
	body := text[start:end]
	matches := fileInputPattern.FindAllStringSubmatchIndex(body, -1)
	inputs := make([]FileInput, 0, len(matches))
	for i, match := range matches {
		annotationStart := start + match[0]
		declarationStart := annotationBlockStart(text, annotationStart, start)
		declarationEnd := end
		if i+1 < len(matches) {
			declarationEnd = start + matches[i+1][0]
		}
		if limit := strings.IndexByte(text[declarationStart:declarationEnd], '\n'); limit >= 0 {
			// Include a bounded declaration window for stacked annotations and the binding.
			windowEnd := declarationStart
			lines := 0
			for windowEnd < declarationEnd && lines < 8 {
				n := strings.IndexByte(text[windowEnd:declarationEnd], '\n')
				if n < 0 {
					windowEnd = declarationEnd
					break
				}
				windowEnd += n + 1
				lines++
				if lines > 1 && binding(text[declarationStart:windowEnd]) != "" {
					break
				}
			}
			declarationEnd = windowEnd
		}
		declaration := text[declarationStart:declarationEnd]
		input := FileInput{Kind: snakeUpper(body[match[2]:match[3]]), Binding: binding(declaration), Declaration: span(text, declarationStart, declarationEnd), Primary: []string{}, Supplementary: []string{}}
		for _, p := range primaryPatterns {
			if p.pattern.MatchString(declaration) {
				input.Primary = append(input.Primary, p.value)
			}
		}
		for _, p := range supplementaryPatterns {
			if p.pattern.MatchString(declaration) {
				input.Supplementary = append(input.Supplementary, p.value)
			}
		}
		sort.Strings(input.Primary)
		sort.Strings(input.Supplementary)
		inputs = append(inputs, input)
	}
	for _, match := range classpathPattern.FindAllStringSubmatchIndex(body, -1) {
		annotationStart := start + match[0]
		declarationStart := annotationBlockStart(text, annotationStart, start)
		declarationEnd := end
		windowEnd := declarationStart
		for lines := 0; windowEnd < declarationEnd && lines < 8; lines++ {
			n := strings.IndexByte(text[windowEnd:declarationEnd], '\n')
			if n < 0 {
				windowEnd = declarationEnd
				break
			}
			windowEnd += n + 1
			if lines > 0 && binding(text[declarationStart:windowEnd]) != "" {
				break
			}
		}
		declarationEnd = windowEnd
		declaration := text[declarationStart:declarationEnd]
		if fileInputPattern.MatchString(declaration) {
			continue
		}
		input := FileInput{Kind: "INPUT_FILES", Binding: binding(declaration), Declaration: span(text, declarationStart, declarationEnd), Primary: []string{}, Supplementary: []string{}}
		for _, p := range primaryPatterns {
			if p.pattern.MatchString(declaration) {
				input.Primary = append(input.Primary, p.value)
			}
		}
		for _, p := range supplementaryPatterns {
			if p.pattern.MatchString(declaration) {
				input.Supplementary = append(input.Supplementary, p.value)
			}
		}
		sort.Strings(input.Primary)
		sort.Strings(input.Supplementary)
		inputs = append(inputs, input)
	}
	sort.Slice(inputs, func(i, j int) bool { return inputs[i].Declaration.StartLine < inputs[j].Declaration.StartLine })
	return inputs
}

func annotationBlockStart(text string, offset, lowerBound int) int {
	start := offset
	lineStart := strings.LastIndexByte(text[lowerBound:start], '\n') + lowerBound + 1
	if strings.Contains(text[lineStart:offset], "@") {
		start = lineStart
	}
	for lineStart > lowerBound {
		previousEnd := lineStart - 1
		previousStart := strings.LastIndexByte(text[lowerBound:previousEnd], '\n') + lowerBound + 1
		line := strings.TrimSpace(text[previousStart:previousEnd])
		if !strings.HasPrefix(line, "@") {
			break
		}
		start = previousStart
		lineStart = previousStart
	}
	return start
}

func decide(body string, inputs []FileInput) (Decision, string) {
	if strings.Contains(body, "@CacheableTask") {
		return AlreadyCacheable, "CACHEABLE_TASK_MARKER_PRESENT"
	}
	if strings.Contains(body, "@DisableCachingByDefault") {
		return DisabledCaching, "DISABLE_CACHING_MARKER_PRESENT"
	}
	hasInput := regexp.MustCompile(`@(?:get:)?Input\b|@(?:get:)?(?:InputFile|InputFiles|InputDirectory|InputDirectories|Classpath|CompileClasspath)\b`).MatchString(body)
	hasOutput := regexp.MustCompile(`@(?:get:)?Output(?:File|Files|Directory|Directories)\b`).MatchString(body)
	if strings.Contains(body, "@TaskAction") && !hasInput && !hasOutput {
		return NoAction, "NO_DECLARED_TASK_INPUTS_OR_OUTPUTS"
	}
	if !strings.Contains(body, "@TaskAction") || !hasOutput || !hasInput {
		return IncompleteAmbiguous, "INCOMPLETE_CUSTOM_TASK_CONTRACT"
	}
	if len(inputs) == 0 {
		return MarkerOnlyEligible, "NO_FILE_INPUT_REQUIRES_NORMALIZATION"
	}
	missing := false
	for _, input := range inputs {
		if input.Binding == "" || len(input.Primary) > 1 {
			return IncompleteAmbiguous, "AMBIGUOUS_FILE_INPUT_DECLARATION"
		}
		if len(input.Primary) == 0 {
			missing = true
		}
		if len(input.Primary) == 1 && input.Primary[0] == "PATH_SENSITIVE_ABSOLUTE" {
			return ExplicitNonPortable, "PATH_SENSITIVE_ABSOLUTE"
		}
	}
	if missing {
		return ReviewedRelativeProofNeeded, "MISSING_PRIMARY_NORMALIZATION"
	}
	return MarkerOnlyEligible, "EVERY_FILE_INPUT_HAS_PORTABLE_PRIMARY_NORMALIZATION"
}

func SortCandidates(candidates []Candidate) {
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Path == candidates[j].Path {
			return candidates[i].ClassName < candidates[j].ClassName
		}
		return candidates[i].Path < candidates[j].Path
	})
}

func binding(declaration string) string {
	for _, line := range strings.Split(declaration, "\n") {
		trimmed := strings.TrimSpace(line)
		for strings.HasPrefix(trimmed, "@") {
			next := leadingAnnotation.ReplaceAllString(trimmed, "")
			if next == trimmed {
				break
			}
			trimmed = strings.TrimSpace(next)
		}
		if trimmed == "" {
			continue
		}
		for _, p := range bindingPatterns {
			if m := p.FindStringSubmatch(trimmed); len(m) == 2 {
				return m[1]
			}
		}
	}
	return ""
}
func snakeUpper(value string) string {
	var b strings.Builder
	for i, r := range value {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return strings.ToUpper(b.String())
}
func language(path string) string {
	if strings.HasSuffix(path, ".java") {
		return "JAVA"
	}
	if strings.HasSuffix(path, ".groovy") {
		return "GROOVY"
	}
	if strings.HasSuffix(path, ".kt") {
		return "KOTLIN"
	}
	return "UNKNOWN"
}
func span(text string, start, end int) Span {
	line, col := position(text, start)
	endLine, endCol := position(text, end)
	return Span{line, col, endLine, endCol}
}
func position(text string, offset int) (int, int) {
	before := text[:offset]
	line := strings.Count(before, "\n") + 1
	last := strings.LastIndexByte(before, '\n')
	return line, offset - last
}
