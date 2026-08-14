package outputequivalence

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestExactBytesRemainDefault(t *testing.T) {
	left, right := t.TempDir(), t.TempDir()
	writeFile(t, left, "build/report.txt", "same\n")
	writeFile(t, right, "build/report.txt", "same\n")
	leftSHA, _, err := HashOutputs(left, []string{"build/**"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	rightSHA, _, err := HashOutputs(right, []string{"build/**"}, nil)
	if err != nil || leftSHA != rightSHA {
		t.Fatalf("exact outputs differ: %s %s %v", leftSHA, rightSHA, err)
	}
	writeFile(t, right, "build/report.txt", "changed\n")
	rightSHA, _, err = HashOutputs(right, []string{"build/**"}, nil)
	if err != nil || leftSHA == rightSHA {
		t.Fatal("payload drift was hidden by exact-byte mode")
	}
}

func TestRepositoryRootTextCanonicalizesOnlyTheRoot(t *testing.T) {
	left, right := t.TempDir(), t.TempDir()
	writeFile(t, left, "build/report.xml", "<file>"+left+"/src/Main.java</file>\n")
	writeFile(t, right, "build/report.xml", "<file>"+right+"/src/Main.java</file>\n")
	contract := Contract{SchemaVersion: SchemaVersion, Rules: []Rule{{
		Pattern: "build/*.xml", Mode: ModeRepositoryRootText,
	}}, ReviewRequired: true}
	leftSHA, _, err := HashOutputs(left, []string{"build/*.xml"}, &contract)
	if err != nil {
		t.Fatal(err)
	}
	rightSHA, _, err := HashOutputs(right, []string{"build/*.xml"}, &contract)
	if err != nil || leftSHA != rightSHA {
		t.Fatalf("relocatable outputs differ: %s %s %v", leftSHA, rightSHA, err)
	}
	writeFile(t, right, "build/report.xml", "<file>"+right+"/src/Other.java</file>\n")
	rightSHA, _, err = HashOutputs(right, []string{"build/*.xml"}, &contract)
	if err != nil || leftSHA == rightSHA {
		t.Fatal("finding-level payload drift was hidden")
	}
}

func TestUnruledRequiredOutputRemainsExactBesideRelocatableText(t *testing.T) {
	left, right := t.TempDir(), t.TempDir()
	writeFile(t, left, "build/report.html", "<p>same</p>\n")
	writeFile(t, right, "build/report.html", "<p>same</p>\n")
	writeFile(t, left, "build/report.xml", "<file>"+left+"/src/Main.java</file>\n")
	writeFile(t, right, "build/report.xml", "<file>"+right+"/src/Main.java</file>\n")
	contract := Contract{SchemaVersion: SchemaVersion, Rules: []Rule{{
		Pattern: "build/*.xml", Mode: ModeRepositoryRootText,
	}}, ReviewRequired: true}
	patterns := []string{"build/*.html", "build/*.xml"}
	leftSHA, _, err := HashOutputs(left, patterns, &contract)
	if err != nil {
		t.Fatal(err)
	}
	rightSHA, _, err := HashOutputs(right, patterns, &contract)
	if err != nil || leftSHA != rightSHA {
		t.Fatalf("mixed exact and relocatable outputs differ: %s %s %v", leftSHA, rightSHA, err)
	}
	writeFile(t, right, "build/report.html", "<p>changed</p>\n")
	rightSHA, _, err = HashOutputs(right, patterns, &contract)
	if err != nil || leftSHA == rightSHA {
		t.Fatal("exact output drift was hidden beside a relocatable output")
	}
}

func TestCanonicalZIPIgnoresOrderTimestampsAndDeclaredPropertyValue(t *testing.T) {
	left, right := t.TempDir(), t.TempDir()
	writeZIP(t, filepath.Join(left, "build/library.jar"), []zipFixtureEntry{
		{name: "payload.txt", body: "payload"},
		{name: "META-INF/release.properties", body: "Version=1\nBuildTime=one\n"},
	})
	writeZIP(t, filepath.Join(right, "build/library.jar"), []zipFixtureEntry{
		{name: "META-INF/release.properties", body: "Version=1\nBuildTime=two\n"},
		{name: "payload.txt", body: "payload"},
	})
	contract := Contract{SchemaVersion: SchemaVersion, Rules: []Rule{{
		Pattern: "build/*.jar", Mode: ModeCanonicalZIP,
		VolatileProperties: []VolatileProperties{{Entry: "META-INF/release.properties", Keys: []string{"BuildTime"}}},
	}}, ReviewRequired: true}
	leftSHA, _, err := HashOutputs(left, []string{"build/*.jar"}, &contract)
	if err != nil {
		t.Fatal(err)
	}
	rightSHA, _, err := HashOutputs(right, []string{"build/*.jar"}, &contract)
	if err != nil || leftSHA != rightSHA {
		t.Fatalf("canonical ZIP outputs differ: %s %s %v", leftSHA, rightSHA, err)
	}
	writeZIP(t, filepath.Join(right, "build/library.jar"), []zipFixtureEntry{
		{name: "META-INF/release.properties", body: "Version=2\nBuildTime=three\n"},
		{name: "payload.txt", body: "payload"},
	})
	rightSHA, _, err = HashOutputs(right, []string{"build/*.jar"}, &contract)
	if err != nil || leftSHA == rightSHA {
		t.Fatal("non-volatile archive payload drift was hidden")
	}
}

func TestContractFailsClosed(t *testing.T) {
	valid := []byte(`{"schemaVersion":"buildopt.poc/output-equivalence/v1","rules":[{"pattern":"build/*.zip","mode":"CANONICAL_ZIP"}],"reviewRequired":true,"activationAutomatic":false,"productionAuthorized":false}`)
	if _, err := Parse(valid); err != nil {
		t.Fatal(err)
	}
	for _, raw := range [][]byte{
		[]byte(`{"schemaVersion":"buildopt.poc/output-equivalence/v1","rules":[],"reviewRequired":true}`),
		[]byte(`{"schemaVersion":"buildopt.poc/output-equivalence/v1","rules":[{"pattern":"../build/**","mode":"CANONICAL_ZIP"}],"reviewRequired":true}`),
		[]byte(`{"schemaVersion":"buildopt.poc/output-equivalence/v1","rules":[{"pattern":"build/[broken.zip","mode":"CANONICAL_ZIP"}],"reviewRequired":true}`),
		[]byte(`{"schemaVersion":"buildopt.poc/output-equivalence/v1","rules":[{"pattern":"build/**","mode":"ANYTHING"}],"reviewRequired":true}`),
		[]byte(`{"schemaVersion":"buildopt.poc/output-equivalence/v1","rules":[{"pattern":"build/**","mode":"REPOSITORY_ROOT_TEXT","volatileProperties":[{"entry":"x","keys":["BuildTime"]}]}],"reviewRequired":true}`),
	} {
		if _, err := Parse(raw); err == nil {
			t.Fatalf("invalid contract was accepted: %s", raw)
		}
	}
	root := t.TempDir()
	writeFile(t, root, "build/report.txt", "no absolute path\n")
	contract := Contract{SchemaVersion: SchemaVersion, Rules: []Rule{{Pattern: "build/*.txt", Mode: ModeRepositoryRootText}}, ReviewRequired: true}
	if _, _, err := HashOutputs(root, []string{"build/*.txt"}, &contract); err == nil {
		t.Fatal("unused relocation rule was accepted")
	}
}

type zipFixtureEntry struct{ name, body string }

func writeZIP(t *testing.T, filename string, entries []zipFixtureEntry) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		header.SetMode(0o644)
		output, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := output.Write([]byte(entry.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, root, relative, content string) {
	t.Helper()
	filename := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCanonicalPropertyOutputIsStable(t *testing.T) {
	first, err := canonicalizeProperties([]byte("A=1\nBuildTime=first\n"), []string{"BuildTime"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := canonicalizeProperties([]byte("A=1\nBuildTime=second\n"), []string{"BuildTime"})
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("canonical properties differ: %q %q %v", first, second, err)
	}
}
