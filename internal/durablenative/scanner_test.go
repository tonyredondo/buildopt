package durablenative

import "testing"

func TestScanSourceAcceptsCompleteJavaGroovyAndKotlinContracts(t *testing.T) {
	tests := []struct {
		path   string
		source string
	}{
		{"Task.java", `public abstract class Work extends DefaultTask { @Input abstract String getIn(); @OutputFile abstract File getOut(); @TaskAction void run() {} }`},
		{"Task.groovy", `abstract class Work extends DefaultTask { @Input abstract Property<String> getIn(); @OutputDirectory abstract DirectoryProperty getOut(); @TaskAction void run() {} }`},
		{"Task.kt", `abstract class Work : DefaultTask() { @get:Input abstract val input: Property<String>; @get:OutputDirectory abstract val output: DirectoryProperty; @TaskAction fun run() {} }`},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			candidate, ok := ScanSource(test.path, []byte(test.source))
			if !ok || candidate.ClassName != "Work" || candidate.SourceSHA256 == "" {
				t.Fatalf("unexpected scan result: %#v, %v", candidate, ok)
			}
		})
	}
}

func TestScanSourceRejectsIncompleteOrExplicitContracts(t *testing.T) {
	for name, source := range map[string]string{
		"no output":        `class Work extends DefaultTask { @Input String in; @TaskAction void run() {} }`,
		"already cached":   `@CacheableTask class Work extends DefaultTask { @Input String in; @OutputFile File out; @TaskAction void run() {} }`,
		"caching disabled": `@DisableCachingByDefault class Work extends DefaultTask { @Input String in; @OutputFile File out; @TaskAction void run() {} }`,
		"not a task":       `class Work { @Input String in; @OutputFile File out; @TaskAction void run() {} }`,
	} {
		t.Run(name, func(t *testing.T) {
			if candidate, ok := ScanSource("Work.java", []byte(source)); ok {
				t.Fatalf("unexpected candidate: %#v", candidate)
			}
		})
	}
}
