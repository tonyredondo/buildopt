// Package configurationinputsource finds source-bound configuration-time
// process reads. It is deliberately conservative: syntax finds an operation,
// while explicit ownership and phase facts decide whether it can advance.
package configurationinputsource

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

const (
	DecisionPure              = "PURE_CONFIGURATION_PROCESS_READ"
	DecisionSideEffect        = "SIDE_EFFECTING_PROCESS"
	DecisionSecretInteractive = "SECRET_OR_INTERACTIVE_PROCESS"
	DecisionTaskAction        = "TASK_ACTION_ONLY"
	DecisionAlreadyTracked    = "ALREADY_TRACKED"
	DecisionAmbiguous         = "AMBIGUOUS_BINDING"
	DecisionSourceDrifted     = "SOURCE_DRIFTED"
	DecisionNoAction          = "NO_ACTION"

	RecipeProviderExec = "PROVIDER_FACTORY_EXEC_V1"
	RecipeValueSource  = "TYPED_VALUE_SOURCE_V1"
)

var ErrInvalidFacts = errors.New("source-bound configuration-input facts are invalid")

type Span struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn"`
	EndLine     int `json:"endLine"`
	EndColumn   int `json:"endColumn"`
}

type CallSite struct {
	Path string `json:"path"`
	Line int    `json:"line"`
}

// Facts supplies semantic information that lexical syntax cannot prove. Label
// fields are retained in output but never consumed by classification.
type Facts struct {
	SourcePath              string     `json:"sourcePath"`
	ExpectedSHA256          string     `json:"expectedSha256"`
	Language                string     `json:"language"`
	DeclarationSpan         Span       `json:"declarationSpan"`
	CallSites               []CallSite `json:"callSites"`
	SourceOwned             bool       `json:"sourceOwned"`
	GeneratedOrVendor       bool       `json:"generatedOrVendor"`
	ConfigurationBound      bool       `json:"configurationBound"`
	TaskActionBound         bool       `json:"taskActionBound"`
	BindingAmbiguous        bool       `json:"bindingAmbiguous"`
	SecretBearing           bool       `json:"secretBearing"`
	Interactive             bool       `json:"interactive"`
	SemanticSideEffect      bool       `json:"semanticSideEffect"`
	DirectProviderAvailable bool       `json:"directProviderAvailable"`
	RepositoryLabel         string     `json:"repositoryLabel,omitempty"`
	TaskLabel               string     `json:"taskLabel,omitempty"`
	VariableLabel           string     `json:"variableLabel,omitempty"`
}

type Row struct {
	SourcePath      string     `json:"sourcePath"`
	SourceSHA256    string     `json:"sourceSha256"`
	Language        string     `json:"language"`
	DeclarationSpan Span       `json:"declarationSpan"`
	OperationSpan   Span       `json:"operationSpan"`
	CallSites       []CallSite `json:"callSites,omitempty"`
	OperationKind   string     `json:"operationKind,omitempty"`
	Recipe          string     `json:"recipe,omitempty"`
	Decision        string     `json:"decision"`
	Reason          string     `json:"reason"`
}

type operation struct {
	start, end int
	kind       string
	tracked    bool
}

var operationTokens = []struct {
	token   string
	kind    string
	tracked bool
}{
	{"providers.exec", "PROVIDER_FACTORY_EXEC", true},
	{"providerFactory.exec", "PROVIDER_FACTORY_EXEC", true},
	{"ProcessBuilder", "PROCESS_BUILDER", false},
	{"Runtime.getRuntime().exec", "RUNTIME_EXEC", false},
}

var sideEffectFragments = []string{
	"update-index", " tag -f", " tag -d", " remote add", " fetch ",
	" checkout ", " reset ", " commit ", " push ", " add -a", " add -A",
}

// Scan emits one decision per operation found inside the frozen declaration.
func Scan(source []byte, facts Facts) ([]Row, error) {
	if err := validate(source, facts); err != nil {
		return nil, err
	}
	actualDigest := digest(source)
	base := Row{SourcePath: facts.SourcePath, SourceSHA256: actualDigest, Language: facts.Language, DeclarationSpan: facts.DeclarationSpan, CallSites: append([]CallSite(nil), facts.CallSites...)}
	if actualDigest != facts.ExpectedSHA256 {
		base.Decision = DecisionSourceDrifted
		base.Reason = "frozen source digest does not match current source"
		return []Row{base}, nil
	}
	start, end, ok := byteRange(source, facts.DeclarationSpan)
	if !ok {
		return nil, ErrInvalidFacts
	}
	region := source[start:end]
	operations := findOperations(region, start)
	if len(operations) == 0 {
		base.Decision = DecisionNoAction
		base.Reason = "no supported process operation is present in the bound declaration"
		return []Row{base}, nil
	}
	rows := make([]Row, 0, len(operations))
	for _, op := range operations {
		row := base
		row.OperationKind = op.kind
		row.OperationSpan = spanForOffsets(source, op.start, op.end)
		switch {
		case !facts.SourceOwned || facts.GeneratedOrVendor || facts.BindingAmbiguous || (facts.ConfigurationBound == facts.TaskActionBound):
			row.Decision, row.Reason = DecisionAmbiguous, "ownership, phase, or call-site binding is incomplete"
		case op.tracked:
			row.Decision, row.Reason = DecisionAlreadyTracked, "operation already uses a tracked Gradle provider"
		case facts.TaskActionBound:
			row.Decision, row.Reason = DecisionTaskAction, "process execution is deferred to a task action"
		case facts.SecretBearing || facts.Interactive:
			row.Decision, row.Reason = DecisionSecretInteractive, "secret-bearing or interactive processes are outside v1"
		case facts.SemanticSideEffect || hasSideEffect(region):
			row.Decision, row.Reason = DecisionSideEffect, "explicit command semantics may mutate persistent state"
		default:
			row.Decision, row.Reason = DecisionPure, "bounded repository-owned configuration-time process read"
			if facts.DirectProviderAvailable {
				row.Recipe = RecipeProviderExec
			} else {
				row.Recipe = RecipeValueSource
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func validate(source []byte, facts Facts) error {
	if len(source) == 0 || facts.SourcePath == "" || facts.Language == "" || !validDigest(facts.ExpectedSHA256) ||
		facts.DeclarationSpan.StartLine < 1 || facts.DeclarationSpan.EndLine < facts.DeclarationSpan.StartLine ||
		facts.DeclarationSpan.StartColumn < 1 || facts.DeclarationSpan.EndColumn < 1 {
		return ErrInvalidFacts
	}
	for _, call := range facts.CallSites {
		if call.Path == "" || call.Line < 1 {
			return ErrInvalidFacts
		}
	}
	return nil
}

func findOperations(region []byte, base int) []operation {
	var found []operation
	text := string(region)
	for _, candidate := range operationTokens {
		for cursor := 0; ; {
			index := strings.Index(text[cursor:], candidate.token)
			if index < 0 {
				break
			}
			index += cursor
			if identifierBoundary(text, index, len(candidate.token)) && followedByInvocation(text, index+len(candidate.token), candidate.tracked) {
				end := callEnd(text, index+len(candidate.token))
				found = append(found, operation{start: base + index, end: base + end, kind: candidate.kind, tracked: candidate.tracked})
			}
			cursor = index + len(candidate.token)
		}
	}
	for left := 0; ; {
		index := strings.Index(text[left:], ".execute(")
		if index < 0 {
			break
		}
		index += left
		end := callEnd(text, index+len(".execute"))
		found = append(found, operation{start: base + index + 1, end: base + end, kind: "GROOVY_EXECUTE"})
		left = index + len(".execute(")
	}
	for i := 1; i < len(found); i++ {
		for j := i; j > 0 && found[j].start < found[j-1].start; j-- {
			found[j], found[j-1] = found[j-1], found[j]
		}
	}
	return found
}

func followedByInvocation(text string, afterToken int, lambdaAllowed bool) bool {
	for afterToken < len(text) && unicode.IsSpace(rune(text[afterToken])) {
		afterToken++
	}
	return afterToken < len(text) && (text[afterToken] == '(' || lambdaAllowed && text[afterToken] == '{')
}

func identifierBoundary(text string, start, length int) bool {
	beforeOK := start == 0 || !(unicode.IsLetter(rune(text[start-1])) || unicode.IsDigit(rune(text[start-1])) || text[start-1] == '_')
	end := start + length
	afterOK := end == len(text) || !(unicode.IsLetter(rune(text[end])) || unicode.IsDigit(rune(text[end])) || text[end] == '_')
	return beforeOK && afterOK
}

func callEnd(text string, afterToken int) int {
	open := strings.IndexByte(text[afterToken:], '(')
	if open < 0 {
		return afterToken
	}
	open += afterToken
	depth := 0
	quote := byte(0)
	escaped := false
	for i := open; i < len(text); i++ {
		char := text[i]
		if quote != 0 {
			if escaped {
				escaped = false
			} else if char == '\\' {
				escaped = true
			} else if char == quote {
				quote = 0
			}
			continue
		}
		if char == '\'' || char == '"' || char == '`' {
			quote = char
			continue
		}
		switch char {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return len(text)
}

func hasSideEffect(region []byte) bool {
	normalized := strings.ToLower(strings.ReplaceAll(string(region), "\"", ""))
	for _, fragment := range sideEffectFragments {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}

func byteRange(source []byte, span Span) (int, int, bool) {
	start := offset(source, span.StartLine, span.StartColumn)
	end := offset(source, span.EndLine, span.EndColumn)
	if start < 0 || end < start {
		return 0, 0, false
	}
	if end < len(source) {
		end++
	}
	return start, end, true
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

func spanForOffsets(source []byte, start, end int) Span {
	startLine, startColumn := lineColumn(source, start)
	endLine, endColumn := lineColumn(source, max(start, end-1))
	return Span{StartLine: startLine, StartColumn: startColumn, EndLine: endLine, EndColumn: endColumn}
}

func lineColumn(source []byte, target int) (int, int) {
	line, column := 1, 1
	for index, char := range source {
		if index == target {
			break
		}
		if char == '\n' {
			line, column = line+1, 1
		} else {
			column++
		}
	}
	return line, column
}

func validDigest(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && strings.ToLower(value) == value
}

func digest(content []byte) string {
	sum := sha256.Sum256(content)
	return fmt.Sprintf("%x", sum)
}
