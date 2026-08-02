// Command documentation-validator checks repository documentation structure.
package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	markdownLink      = regexp.MustCompile(`!?\[[^\]]*\]\(([^)]+)\)`)
	repositoryCommand = regexp.MustCompile(`\./(?:dev|packaging)/[A-Za-z0-9._/-]+`)
)

var requiredDocuments = []string{
	"README.md",
	"CONTRIBUTING.md",
	"docs/README.md",
	"docs/getting-started/quickstart.md",
	"docs/getting-started/developer-onboarding.md",
	"docs/architecture/overview.md",
	"docs/architecture/repository-map.md",
	"docs/guides/product-workflows.md",
	"docs/guides/ci-integration.md",
	"docs/guides/operations.md",
	"docs/reference/cli.md",
	"docs/reference/configuration.md",
	"docs/reference/validation.md",
	"docs/troubleshooting.md",
	"docs/glossary.md",
}

func main() {
	root, err := repositoryRoot()
	if err != nil {
		fail(err)
	}
	markdown, err := markdownFiles(root)
	if err != nil {
		fail(err)
	}
	var problems []string
	problems = append(problems, checkRequiredDocuments(root)...)
	linkCount := 0
	commandCount := 0
	for _, path := range markdown {
		links, commands, fileProblems := checkMarkdown(root, path)
		linkCount += links
		commandCount += commands
		problems = append(problems, fileProblems...)
	}
	packageCount, packageProblems := checkGoPackageDocs(root)
	problems = append(problems, packageProblems...)
	if len(problems) != 0 {
		sort.Strings(problems)
		for _, problem := range problems {
			fmt.Fprintln(os.Stderr, problem)
		}
		os.Exit(1)
	}
	fmt.Printf(
		"Documentation OK: %d Markdown files, %d local links, %d repository commands, %d Go packages documented\n",
		len(markdown), linkCount, commandCount, packageCount,
	)
}

func repositoryRoot() (string, error) {
	current, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if regularFile(filepath.Join(current, "go.mod")) && regularFile(filepath.Join(current, "README.md")) {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("documentation-validator: repository root not found")
		}
		current = parent
	}
}

func markdownFiles(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".gradle", ".tools", "build", "out", "target":
				if path != root {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".md") {
			paths = append(paths, path)
		}
		return nil
	})
	sort.Strings(paths)
	return paths, err
}

func checkRequiredDocuments(root string) []string {
	var problems []string
	for _, relative := range requiredDocuments {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if !regularFile(path) {
			problems = append(problems, fmt.Sprintf("missing required documentation: %s", relative))
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			problems = append(problems, fmt.Sprintf("read %s: %v", relative, err))
			continue
		}
		if !strings.HasPrefix(string(content), "# ") {
			problems = append(problems, fmt.Sprintf("documentation needs one leading H1: %s", relative))
		}
	}
	return problems
}

func checkMarkdown(root, path string) (int, int, []string) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, []string{fmt.Sprintf("read %s: %v", relative(root, path), err)}
	}
	text := string(content)
	lineStarts := []int{0}
	for index, character := range text {
		if character == '\n' {
			lineStarts = append(lineStarts, index+1)
		}
	}
	var problems []string
	linkCount := 0
	for _, match := range markdownLink.FindAllStringSubmatchIndex(text, -1) {
		target := strings.TrimSpace(text[match[2]:match[3]])
		if space := strings.IndexAny(target, " \t"); space >= 0 {
			target = target[:space]
		}
		target = strings.Trim(target, "<>")
		if target == "" || strings.HasPrefix(target, "#") || externalTarget(target) {
			continue
		}
		linkCount++
		decoded, decodeErr := url.PathUnescape(strings.SplitN(strings.SplitN(target, "#", 2)[0], "?", 2)[0])
		if decodeErr != nil {
			problems = append(problems, location(root, path, lineStarts, match[0])+": invalid escaped link "+target)
			continue
		}
		candidate := filepath.Clean(filepath.Join(filepath.Dir(path), filepath.FromSlash(decoded)))
		if !within(root, candidate) || !exists(candidate) {
			problems = append(problems, location(root, path, lineStarts, match[0])+": broken local link "+target)
		}
	}
	commandCount := 0
	for _, match := range repositoryCommand.FindAllStringIndex(text, -1) {
		if match[1] < len(text) && text[match[1]] == '*' {
			continue
		}
		commandCount++
		command := strings.TrimRight(text[match[0]:match[1]], ".,;:")
		candidate := filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(command, "./")))
		if !regularFile(candidate) {
			problems = append(problems, location(root, path, lineStarts, match[0])+": missing referenced command "+command)
		}
	}
	return linkCount, commandCount, problems
}

func checkGoPackageDocs(root string) (int, []string) {
	base := filepath.Join(root, "internal")
	packages := map[string]bool{}
	documented := map[string]bool{}
	fset := token.NewFileSet()
	var problems []string
	_ = filepath.WalkDir(base, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			problems = append(problems, fmt.Sprintf("walk %s: %v", relative(root, path), walkErr))
			return nil
		}
		if entry.IsDir() {
			if entry.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments|parser.PackageClauseOnly)
		if parseErr != nil {
			problems = append(problems, fmt.Sprintf("parse %s: %v", relative(root, path), parseErr))
			return nil
		}
		directory := filepath.Dir(path)
		packages[directory] = true
		if file.Doc != nil && strings.HasPrefix(strings.TrimSpace(file.Doc.Text()), "Package "+file.Name.Name) {
			documented[directory] = true
		}
		return nil
	})
	for directory := range packages {
		if !documented[directory] {
			problems = append(problems, fmt.Sprintf("Go package needs a Package comment: %s", relative(root, directory)))
		}
	}
	return len(packages), problems
}

func externalTarget(target string) bool {
	lower := strings.ToLower(target)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "mailto:") || strings.HasPrefix(lower, "data:")
}

func location(root, path string, lineStarts []int, offset int) string {
	line := sort.Search(len(lineStarts), func(index int) bool { return lineStarts[index] > offset })
	return fmt.Sprintf("%s:%d", relative(root, path), line)
}

func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func relative(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

func regularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
