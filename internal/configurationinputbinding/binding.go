// Package configurationinputbinding binds strict Gradle diagnostics to exact
// source operations without using repository, task, or executable names as
// classification rules.
package configurationinputbinding

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/tonyredondo/buildopt/internal/configurationinput"
	"github.com/tonyredondo/buildopt/internal/configurationinputsource"
)

const (
	DecisionBound     = "DIAGNOSTIC_BOUND"
	DecisionUnbound   = "DIAGNOSTIC_NOT_OBSERVED"
	DecisionAmbiguous = "DIAGNOSTIC_BINDING_AMBIGUOUS"
)

var quotedString = regexp.MustCompile(`"(?:\\.|[^"\\])*"`)

// Binding records one exact source-operation and diagnostic pair.
type Binding struct {
	SourceRowIndex int    `json:"sourceRowIndex"`
	ProblemIndex   int    `json:"problemIndex"`
	Command        string `json:"command"`
}

// Result is the fail-closed family-level binding decision.
type Result struct {
	Decision                string    `json:"decision"`
	Reason                  string    `json:"reason"`
	SourceRows              int       `json:"sourceRows"`
	Problems                int       `json:"problems"`
	ExternalProcessProblems int       `json:"externalProcessProblems"`
	Bindings                []Binding `json:"bindings"`
}

// Bind matches quoted command arguments in pure source rows to exact commands
// reported by Gradle. One problem matching multiple source rows is ambiguous.
func Bind(source []byte, rows []configurationinputsource.Row, problems []configurationinput.Problem) Result {
	result := Result{SourceRows: len(rows), Problems: len(problems), Bindings: []Binding{}}
	commands := make([]string, len(rows))
	for index, row := range rows {
		if row.Decision == configurationinputsource.DecisionPure {
			commands[index] = sourceCommand(source, row.OperationSpan)
		}
	}
	for _, problem := range problems {
		if problem.Kind != configurationinput.ProblemExternalProcess {
			continue
		}
		result.ExternalProcessProblems++
		matches := 0
		for rowIndex, command := range commands {
			if command == "" || !matchesCommand(problem.Message, command) {
				continue
			}
			matches++
			result.Bindings = append(result.Bindings, Binding{SourceRowIndex: rowIndex, ProblemIndex: problem.Index, Command: command})
		}
		if matches > 1 {
			result.Decision = DecisionAmbiguous
			result.Reason = "one diagnostic matches multiple source operations"
			return result
		}
	}
	if len(result.Bindings) == 0 {
		result.Decision = DecisionUnbound
		result.Reason = "no strict external-process diagnostic matches a pure source operation"
		return result
	}
	result.Decision = DecisionBound
	result.Reason = "strict external-process diagnostic matches one exact source command"
	return result
}

func matchesCommand(message, command string) bool {
	normalized := strings.TrimSpace(message)
	return normalized == "external process started "+command ||
		normalized == "external process started '"+command+"'" ||
		normalized == "Starting an external process '"+command+"' during configuration time is unsupported."
}

func sourceCommand(source []byte, span configurationinputsource.Span) string {
	start := offset(source, span.StartLine, span.StartColumn)
	end := offset(source, span.EndLine, span.EndColumn)
	if start < 0 || end < start || end > len(source) {
		return ""
	}
	if end < len(source) {
		end++
	}
	matches := quotedString.FindAll(source[start:end], -1)
	parts := make([]string, 0, len(matches))
	for _, match := range matches {
		value, err := strconv.Unquote(string(match))
		if err != nil || value == "" {
			return ""
		}
		parts = append(parts, value)
	}
	return strings.Join(parts, " ")
}

func offset(source []byte, line, column int) int {
	currentLine, currentColumn := 1, 1
	for index, char := range source {
		if currentLine == line && currentColumn == column {
			return index
		}
		if char == '\n' {
			currentLine, currentColumn = currentLine+1, 1
		} else {
			currentColumn++
		}
	}
	if currentLine == line && currentColumn == column {
		return len(source)
	}
	return -1
}
