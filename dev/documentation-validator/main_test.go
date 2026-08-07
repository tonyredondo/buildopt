package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckMarkdown(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "README.md"), "# Root\n")
	mustWrite(t, filepath.Join(root, "dev", "check-example"), "#!/bin/sh\n")
	guide := filepath.Join(root, "docs", "guide.md")
	mustWrite(t, guide, "# Guide\n\n[Root](../README.md)\n\n```bash\n./dev/check-example\n```\n")

	links, commands, problems := checkMarkdown(root, guide)
	if links != 1 || commands != 1 || len(problems) != 0 {
		t.Fatalf("checkMarkdown() = links %d, commands %d, problems %v", links, commands, problems)
	}

	mustWrite(t, guide, "# Guide\n\n[Missing](missing.md)\n\n./dev/check-missing\n")
	_, _, problems = checkMarkdown(root, guide)
	if len(problems) != 2 || !strings.Contains(strings.Join(problems, "\n"), "broken local link") ||
		!strings.Contains(strings.Join(problems, "\n"), "missing referenced command") {
		t.Fatalf("unexpected problems: %v", problems)
	}
}

func TestCheckRepositoryLandingPage(t *testing.T) {
	root := t.TempDir()
	if problems := checkRepositoryLandingPage(root); len(problems) != 0 {
		t.Fatalf("empty repository has presentation problems: %v", problems)
	}
	mustWrite(t, filepath.Join(root, ".github", "README.md"), "# Automation\n")
	problems := checkRepositoryLandingPage(root)
	if len(problems) != 1 || !strings.Contains(problems[0], "shadows") {
		t.Fatalf("unexpected presentation problems: %v", problems)
	}
}

func TestReadmeHeadingRejectsTrackerID(t *testing.T) {
	root := t.TempDir()
	readme := filepath.Join(root, "component", "README.md")
	mustWrite(t, readme, "# Component\n\n## WS-001 passthrough\n")

	_, _, problems := checkMarkdown(root, readme)
	if len(problems) != 1 || !strings.Contains(problems[0], "internal tracker ID") {
		t.Fatalf("unexpected README problems: %v", problems)
	}

	guide := filepath.Join(root, "docs", "guide.md")
	mustWrite(t, guide, "# Guide\n\n## WS-001 traceability\n")
	_, _, problems = checkMarkdown(root, guide)
	if len(problems) != 0 {
		t.Fatalf("non-README traceability heading rejected: %v", problems)
	}
}

func TestCheckEnglishLanguage(t *testing.T) {
	root := t.TempDir()
	english := filepath.Join(root, "docs", "english.md")
	mustWrite(t, english, "# Guide\n\nRun the validation command and review the results.\n")
	if problems := checkEnglishLanguage(root, english); len(problems) != 0 {
		t.Fatalf("English documentation rejected: %v", problems)
	}

	accentedSpanish := filepath.Join(root, "docs", "accented.md")
	mustWrite(t, accentedSpanish, "# Guía\n\nEjecuta la validación.\n")
	problems := checkEnglishLanguage(root, accentedSpanish)
	if len(problems) != 1 || !strings.Contains(problems[0], "must be written in English") {
		t.Fatalf("accented Spanish documentation accepted: %v", problems)
	}

	unaccentedSpanish := filepath.Join(root, "evidence.json")
	mustWrite(t, unaccentedSpanish, "{\"summary\":\"resultados para el equipo\"}\n")
	problems = checkEnglishLanguage(root, unaccentedSpanish)
	if len(problems) != 1 || !strings.Contains(problems[0], "possible Spanish text") {
		t.Fatalf("unaccented Spanish documentation accepted: %v", problems)
	}
}

func TestCheckGoPackageDocs(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "internal", "good", "doc.go"), "// Package good is documented.\npackage good\n")
	mustWrite(t, filepath.Join(root, "internal", "bad", "bad.go"), "package bad\n")

	packages, problems := checkGoPackageDocs(root)
	if packages != 2 || len(problems) != 1 || !strings.Contains(problems[0], "internal/bad") {
		t.Fatalf("checkGoPackageDocs() = packages %d, problems %v", packages, problems)
	}
}

func TestExternalTarget(t *testing.T) {
	for _, target := range []string{"https://example.test", "http://example.test", "mailto:owner@example.test", "data:text/plain,ok"} {
		if !externalTarget(target) {
			t.Fatalf("externalTarget(%q) = false", target)
		}
	}
	if externalTarget("../README.md") {
		t.Fatal("relative repository link classified as external")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
