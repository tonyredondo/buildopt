package buildlogiccorrection

import (
	"os"
	"path/filepath"
	"testing"
)

const testAnalysis = `{
  "tasks":[{"identity":":service:compileJava","buildPath":":","taskClass":"org.gradle.api.tasks.compile.JavaCompile","durationMs":800,"criticalPath":true}],
  "builds":[{"buildPath":":","taskExecutionSpanMs":1000}]
}`

func TestScanBindsExplicitOptOutByTaskName(t *testing.T) {
	report := scanFixture(t, `tasks.named("compileJava") {
  outputs.upToDateWhen { false }
}`)
	if report.Decision != DecisionProposal || !report.Actionable {
		t.Fatalf("expected proposal, got %+v", report)
	}
	if len(report.Facts) != 1 || report.Facts[0].Binding != BindingMaterial {
		t.Fatalf("expected one material fact, got %+v", report.Facts)
	}
}

func TestScanDoesNotBindUnrelatedTask(t *testing.T) {
	report := scanFixture(t, `tasks.register("copyReports") {
  outputs.cacheIf { false }
}`)
	if report.Decision != DecisionNoAction || report.Actionable {
		t.Fatalf("expected conclusive no-action, got %+v", report)
	}
	if report.Facts[0].Binding != BindingUnrelated {
		t.Fatalf("expected unrelated binding, got %+v", report.Facts[0])
	}
}

func TestScanRejectsDynamicBindingAsAmbiguous(t *testing.T) {
	report := scanFixture(t, `tasks.named(taskName) {
  outputs.doNotCacheIf("owner rule") { true }
}`)
	if report.Decision != DecisionAmbiguous || report.Actionable {
		t.Fatalf("expected ambiguous decision, got %+v", report)
	}
}

func TestScanTreatsLiteralUnrelatedReceiverAsConclusive(t *testing.T) {
	report := scanFixture(t, `tasks.addRule("dynamic") { taskName ->
  tasks.register(taskName) { dependsOn(test) }
  test.outputs.upToDateWhen { false }
}`)
	if report.Decision != DecisionNoAction || report.Facts[0].Binding != BindingUnrelated {
		t.Fatalf("expected literal unrelated receiver, got %+v", report)
	}
}

func TestScanBindsExplicitOptOutByTaskType(t *testing.T) {
	report := scanFixture(t, `tasks.withType<JavaCompile>().configureEach {
  options.incremental = false
}`)
	if report.Decision != DecisionProposal || report.Facts[0].Correction != "ENABLE_EXPLICIT_INCREMENTAL_MODE" {
		t.Fatalf("expected incremental proposal, got %+v", report)
	}
}

func TestSourceDriftChangesInventoryDigest(t *testing.T) {
	root := t.TempDir()
	analysisPath := filepath.Join(root, "analysis.json")
	if err := os.WriteFile(analysisPath, []byte(testAnalysis), 0o600); err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(root, "build.gradle.kts")
	if err := os.WriteFile(sourcePath, []byte("tasks.named(\"compileJava\") {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := Scan("fixture", "revision", "tree", root, analysisPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, []byte("tasks.named(\"compileJava\") { outputs.cacheIf { false } }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	after, err := Scan("fixture", "revision", "tree", root, analysisPath)
	if err != nil {
		t.Fatal(err)
	}
	if before.SourceInventorySHA256 == after.SourceInventorySHA256 {
		t.Fatal("source drift did not change the inventory digest")
	}
}

func TestFamilyLabelDoesNotAffectClassification(t *testing.T) {
	root := t.TempDir()
	analysisPath := filepath.Join(root, "analysis.json")
	if err := os.WriteFile(analysisPath, []byte(testAnalysis), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "build.gradle.kts"), []byte(`tasks.named("compileJava") { outputs.cacheIf { false } }`), 0o600); err != nil {
		t.Fatal(err)
	}
	first, err := Scan("first-label", "revision", "tree", root, analysisPath)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Scan("second-label", "revision", "tree", root, analysisPath)
	if err != nil {
		t.Fatal(err)
	}
	first.Family = ""
	second.Family = ""
	if first.Decision != second.Decision || first.Actionable != second.Actionable || len(first.Facts) != len(second.Facts) {
		t.Fatalf("family label changed classification: first=%+v second=%+v", first, second)
	}
}

func scanFixture(t *testing.T, source string) Report {
	t.Helper()
	root := t.TempDir()
	analysisPath := filepath.Join(root, "analysis.json")
	if err := os.WriteFile(analysisPath, []byte(testAnalysis), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "build.gradle.kts"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := Scan("fixture", "0123456789abcdef0123456789abcdef01234567", "tree", root, analysisPath)
	if err != nil {
		t.Fatal(err)
	}
	return report
}
