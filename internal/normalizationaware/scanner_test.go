package normalizationaware

import "testing"

func TestScanSourceV2NormalizationDecisions(t *testing.T) {
	tests := []struct {
		name, annotation string
		want             Decision
	}{
		{"relative", "@PathSensitive(PathSensitivity.RELATIVE)", MarkerOnlyEligible},
		{"name only", "@PathSensitive(PathSensitivity.NAME_ONLY)", MarkerOnlyEligible},
		{"none", "@PathSensitive(PathSensitivity.NONE)", MarkerOnlyEligible},
		{"classpath", "@Classpath", MarkerOnlyEligible},
		{"compile classpath", "@CompileClasspath", MarkerOnlyEligible},
		{"missing", "", ReviewedRelativeProofNeeded},
		{"absolute", "@PathSensitive(PathSensitivity.ABSOLUTE)", ExplicitNonPortable},
		{"supplementary only", "@NormalizeLineEndings @IgnoreEmptyDirectories", ReviewedRelativeProofNeeded},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := []byte(`abstract class Work extends DefaultTask {` + "\n" + test.annotation + "\n@InputDirectory\nabstract File getSourceDirectory()\n@OutputDirectory abstract File getOutputDirectory()\n@TaskAction void run() {}\n}")
			got := ScanSourceV2("src/Work.groovy", source)
			if len(got) != 1 || got[0].Decision != test.want {
				t.Fatalf("got %#v, want %s", got, test.want)
			}
			if len(got) == 1 && (got[0].SourceSHA256 == "" || got[0].FileInputs[0].Binding != "getSourceDirectory" || got[0].FileInputs[0].Declaration.StartLine == 0) {
				t.Fatalf("missing source binding: %#v", got[0])
			}
		})
	}
	pureClasspath := ScanSourceV2("Work.java", []byte(`class Work extends DefaultTask {
@Classpath FileCollection getRuntimeClasspath()
@OutputFile File getOutput()
@TaskAction void run() {}
}`))
	if len(pureClasspath) != 1 || len(pureClasspath[0].FileInputs) != 1 || pureClasspath[0].FileInputs[0].Kind != "INPUT_FILES" || pureClasspath[0].Decision != MarkerOnlyEligible {
		t.Fatalf("pure classpath was not classified: %#v", pureClasspath)
	}
}

func TestScanSourceV2TypedNonActionsAndAmbiguity(t *testing.T) {
	base := func(marker, input string) []byte {
		return []byte(marker + ` abstract class Work extends DefaultTask {
` + input + `
@OutputFile abstract File getOutput()
@TaskAction void run() {}
}`)
	}
	tests := []struct {
		name   string
		source []byte
		want   Decision
	}{
		{"already cacheable", base("@CacheableTask", "@Input String getValue()"), AlreadyCacheable},
		{"disabled", base("@DisableCachingByDefault", "@Input String getValue()"), DisabledCaching},
		{"no file input", base("", "@Input String getValue()"), MarkerOnlyEligible},
		{"no declared io", []byte(`class Work extends DefaultTask { @TaskAction void run() {} }`), NoAction},
		{"ambiguous primary", base("", "@InputFile @Classpath @PathSensitive(PathSensitivity.RELATIVE) File getInput()"), IncompleteAmbiguous},
		{"incomplete", []byte(`class Work extends DefaultTask { @InputFile File getInput() }`), IncompleteAmbiguous},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ScanSourceV2("Work.java", test.source)
			if len(got) != 1 || got[0].Decision != test.want {
				t.Fatalf("got %#v, want %s", got, test.want)
			}
		})
	}
}

func TestScanSourceV2DoesNotUseNames(t *testing.T) {
	source := []byte(`class ForbiddenRepositoryTask extends DefaultTask {
@InputDirectory File getInput()
@OutputDirectory File getOutput()
@TaskAction void run() {}
}`)
	got := ScanSourceV2("special/repository/name/Task.java", source)
	if len(got) != 1 || got[0].Decision != ReviewedRelativeProofNeeded {
		t.Fatalf("name affected decision: %#v", got)
	}
}
