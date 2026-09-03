// Package buildlogiccorrection finds explicit state-tracking opt-outs that are
// bound to economically material standard Gradle tasks.
package buildlogiccorrection

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	DecisionProposal   = "EXPLICIT_MATERIAL_STATE_OPTOUT_PROPOSAL"
	DecisionNoAction   = "CONCLUSIVE_NO_MATERIAL_STATE_OPTOUT"
	DecisionAmbiguous  = "INCOMPLETE_AMBIGUOUS_SOURCE_BINDING"
	BindingMaterial    = "MATERIAL_TASK"
	BindingUnrelated   = "UNRELATED_TASK"
	BindingAmbiguous   = "AMBIGUOUS_DYNAMIC_TASK_CONFIGURATION"
	minimumDurationMS  = int64(500)
	minimumSpanPercent = int64(2)
)

type analysis struct {
	Tasks  []analysisTask  `json:"tasks"`
	Builds []analysisBuild `json:"builds"`
}

type analysisTask struct {
	Identity     string `json:"identity"`
	BuildPath    string `json:"buildPath"`
	TaskClass    string `json:"taskClass"`
	DurationMS   int64  `json:"durationMs"`
	CriticalPath bool   `json:"criticalPath"`
}

type analysisBuild struct {
	BuildPath           string `json:"buildPath"`
	TaskExecutionSpanMS int64  `json:"taskExecutionSpanMs"`
}

type MaterialTask struct {
	Identity        string `json:"identity"`
	TaskClass       string `json:"taskClass"`
	DurationMS      int64  `json:"durationMs"`
	BuildSpanMS     int64  `json:"buildSpanMs"`
	DurationPercent string `json:"durationPercent"`
}

type SourceFact struct {
	Path          string   `json:"path"`
	Line          int      `json:"line"`
	SourceSHA256  string   `json:"sourceSha256"`
	Correction    string   `json:"correctionClass"`
	Binding       string   `json:"binding"`
	BoundTasks    []string `json:"boundMaterialTasks"`
	SourceExcerpt string   `json:"sourceExcerpt"`
}

type Report struct {
	SchemaVersion         string         `json:"schemaVersion"`
	Family                string         `json:"family"`
	Revision              string         `json:"revision"`
	SourceTree            string         `json:"sourceTree"`
	AnalysisSHA256        string         `json:"analysisSha256"`
	SourceInventorySHA256 string         `json:"sourceInventorySha256"`
	SourceFileCount       int            `json:"sourceFileCount"`
	MaterialTasks         []MaterialTask `json:"materialTasks"`
	Facts                 []SourceFact   `json:"facts"`
	Decision              string         `json:"decision"`
	Actionable            bool           `json:"actionable"`
}

type sourceFile struct {
	path   string
	sha256 string
	lines  []string
}

var optOutPatterns = []struct {
	re         *regexp.Regexp
	correction string
}{
	{regexp.MustCompile(`\b(?:outputs\.)?cacheIf\s*\{\s*false\s*\}`), "REMOVE_EXPLICIT_CACHE_DISABLE"},
	{regexp.MustCompile(`\b(?:outputs\.)?upToDateWhen\s*\{\s*false\s*\}`), "RESTORE_STATE_TRACKING"},
	{regexp.MustCompile(`\bdoNotCacheIf\s*\(`), "REMOVE_EXPLICIT_CACHE_DISABLE"},
	{regexp.MustCompile(`\bdoNotTrackState\s*\(`), "RESTORE_STATE_TRACKING"},
	{regexp.MustCompile(`\b(?:isIncremental|incremental)\s*=\s*false\b`), "ENABLE_EXPLICIT_INCREMENTAL_MODE"},
}

func Scan(family, revision, sourceTree, sourceRoot, analysisPath string) (Report, error) {
	analysisBytes, err := os.ReadFile(analysisPath)
	if err != nil {
		return Report{}, fmt.Errorf("read analysis: %w", err)
	}
	var input analysis
	if err := json.Unmarshal(analysisBytes, &input); err != nil {
		return Report{}, fmt.Errorf("decode analysis: %w", err)
	}
	material, err := materialTasks(input)
	if err != nil {
		return Report{}, err
	}
	files, inventoryDigest, err := readSourceFiles(sourceRoot)
	if err != nil {
		return Report{}, err
	}
	facts := scanFacts(files, material)
	decision := DecisionNoAction
	actionable := false
	for _, fact := range facts {
		switch fact.Binding {
		case BindingMaterial:
			decision = DecisionProposal
			actionable = true
		case BindingAmbiguous:
			if !actionable {
				decision = DecisionAmbiguous
			}
		}
	}
	return Report{
		SchemaVersion:         "buildopt.evidence/critical-path-build-logic-correction-source/v1",
		Family:                family,
		Revision:              revision,
		SourceTree:            sourceTree,
		AnalysisSHA256:        digestBytes(analysisBytes),
		SourceInventorySHA256: inventoryDigest,
		SourceFileCount:       len(files),
		MaterialTasks:         material,
		Facts:                 facts,
		Decision:              decision,
		Actionable:            actionable,
	}, nil
}

func materialTasks(input analysis) ([]MaterialTask, error) {
	spans := make(map[string]int64, len(input.Builds))
	for _, build := range input.Builds {
		if build.TaskExecutionSpanMS <= 0 {
			return nil, fmt.Errorf("analysis build %q has no positive task span", build.BuildPath)
		}
		spans[build.BuildPath] = build.TaskExecutionSpanMS
	}
	var result []MaterialTask
	for _, task := range input.Tasks {
		span, ok := spans[task.BuildPath]
		if !ok {
			return nil, fmt.Errorf("task %q references an unknown build", task.Identity)
		}
		if !task.CriticalPath || task.DurationMS < minimumDurationMS || task.DurationMS*100 < span*minimumSpanPercent {
			continue
		}
		result = append(result, MaterialTask{
			Identity:        task.Identity,
			TaskClass:       task.TaskClass,
			DurationMS:      task.DurationMS,
			BuildSpanMS:     span,
			DurationPercent: fmt.Sprintf("%.6f", float64(task.DurationMS)*100/float64(span)),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Identity < result[j].Identity })
	return result, nil
}

func readSourceFiles(root string) ([]sourceFile, string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.IsDir() {
			if relative != "." && excludedDirectory(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if sourcePath(relative) {
			paths = append(paths, relative)
		}
		return nil
	})
	if err != nil {
		return nil, "", fmt.Errorf("walk source root: %w", err)
	}
	sort.Strings(paths)
	h := sha256.New()
	files := make([]sourceFile, 0, len(paths))
	for _, relative := range paths {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			return nil, "", fmt.Errorf("read %s: %w", relative, err)
		}
		digest := digestBytes(content)
		fmt.Fprintf(h, "%s\t%s\n", relative, digest)
		files = append(files, sourceFile{path: relative, sha256: digest, lines: splitLines(content)})
	}
	return files, hex.EncodeToString(h.Sum(nil)), nil
}

func sourcePath(path string) bool {
	name := filepath.Base(path)
	if name == "settings.gradle" || name == "settings.gradle.kts" || name == "build.gradle" || name == "build.gradle.kts" {
		return true
	}
	extension := filepath.Ext(path)
	if extension == ".gradle" || extension == ".kts" {
		return true
	}
	if extension != ".kt" && extension != ".groovy" && extension != ".java" {
		return false
	}
	return strings.HasPrefix(path, "buildSrc/") || strings.HasPrefix(path, "build-logic/") ||
		strings.HasPrefix(path, "gradle-plugins/") || strings.HasPrefix(path, "conventions/")
}

func excludedDirectory(name string) bool {
	switch name {
	case ".git", ".gradle", "build", "node_modules", ".tools":
		return true
	default:
		return false
	}
}

func splitLines(content []byte) []string {
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	buffer := make([]byte, 64*1024)
	scanner.Buffer(buffer, 1024*1024)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

func scanFacts(files []sourceFile, tasks []MaterialTask) []SourceFact {
	facts := make([]SourceFact, 0)
	for _, file := range files {
		for lineIndex, line := range file.lines {
			for _, pattern := range optOutPatterns {
				if !pattern.re.MatchString(line) {
					continue
				}
				start := lineIndex - 14
				if start < 0 {
					start = 0
				}
				end := lineIndex + 4
				if end > len(file.lines) {
					end = len(file.lines)
				}
				context := strings.Join(file.lines[start:end], "\n")
				bound := boundTasks(context, line, tasks)
				binding := BindingUnrelated
				if len(bound) > 0 {
					binding = BindingMaterial
				} else if dynamicConfiguration(context) && !explicitReceiver(line) {
					binding = BindingAmbiguous
				}
				facts = append(facts, SourceFact{
					Path:          file.path,
					Line:          lineIndex + 1,
					SourceSHA256:  file.sha256,
					Correction:    pattern.correction,
					Binding:       binding,
					BoundTasks:    bound,
					SourceExcerpt: strings.TrimSpace(line),
				})
				break
			}
		}
	}
	sort.Slice(facts, func(i, j int) bool {
		if facts[i].Path == facts[j].Path {
			return facts[i].Line < facts[j].Line
		}
		return facts[i].Path < facts[j].Path
	})
	return facts
}

func boundTasks(context, line string, tasks []MaterialTask) []string {
	var bound []string
	for _, task := range tasks {
		name := taskName(task.Identity)
		typeName := taskTypeName(task.TaskClass)
		namePattern := regexp.MustCompile(`(?:named|register|getByName)\s*(?:<[^>]+>)?\s*\(\s*["']` + regexp.QuoteMeta(name) + `["']`)
		receiverPattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\s*\.\s*(?:outputs|options)\b`)
		typePattern := regexp.MustCompile(`(?:withType\s*(?:<\s*` + regexp.QuoteMeta(typeName) + `\s*>|\(\s*` + regexp.QuoteMeta(typeName) + `\b)|(?:named|register)\s*<\s*` + regexp.QuoteMeta(typeName) + `\s*>)`)
		if namePattern.MatchString(context) || receiverPattern.MatchString(line) || typePattern.MatchString(context) {
			bound = append(bound, task.Identity)
		}
	}
	sort.Strings(bound)
	return bound
}

func dynamicConfiguration(context string) bool {
	dynamic := regexp.MustCompile(`(?:named|register|getByName)\s*\(\s*[A-Za-z_$][A-Za-z0-9_.$]*\s*\)`)
	return dynamic.MatchString(context)
}

func explicitReceiver(line string) bool {
	receiver := regexp.MustCompile(`\b[A-Za-z_$][A-Za-z0-9_.$]*\s*\.\s*(?:outputs|options)\b`)
	return receiver.MatchString(line)
}

func taskName(identity string) string {
	if index := strings.LastIndex(identity, ":"); index >= 0 {
		return identity[index+1:]
	}
	return identity
}

func taskTypeName(className string) string {
	if index := strings.LastIndex(className, "."); index >= 0 {
		return className[index+1:]
	}
	return className
}

func digestBytes(content []byte) string {
	digest := sha256.Sum256(content)
	return hex.EncodeToString(digest[:])
}

func WriteJSON(writer io.Writer, report Report) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(report)
}
