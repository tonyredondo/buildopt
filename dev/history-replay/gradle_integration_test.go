//go:build linux && amd64 && replay_integration && replay_gradle

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func gradleFixture(t *testing.T, count int, script string) Manifest {
	t.Helper()
	m := runnableFixture(t, count, "success")
	gradleHome := os.Getenv("BUILDOPT_REPLAY_GRADLE_HOME")
	javaHome := os.Getenv("BUILDOPT_REPLAY_JAVA_HOME")
	if !filepath.IsAbs(gradleHome) || !filepath.IsAbs(javaHome) {
		t.Fatal("native qualification requires explicit pinned BUILDOPT_REPLAY_GRADLE_HOME and BUILDOPT_REPLAY_JAVA_HOME")
	}
	// Replace only this newly created fixture's unreferenced object history.
	// All workflow source is present in every immutable first-parent revision.
	base := map[string]string{"settings.gradle": "rootProject.name = 'replay-native-fixture'\n", "build.gradle": script}
	blobs := map[string]string{}
	for name, body := range base {
		blobs[name] = testGit(t, m.CommonGit, []byte(body), "hash-object", "-w", "--stdin")
	}
	parent := ""
	for i := 0; i < count; i++ {
		blob := testGit(t, m.CommonGit, []byte(fmt.Sprintf("revision %d\n", i)), "hash-object", "-w", "--stdin")
		treeText := "100644 blob " + blob + "\tsource.txt\n"
		for name, id := range blobs {
			treeText += "100644 blob " + id + "\t" + name + "\n"
		}
		tree := testGit(t, m.CommonGit, []byte(treeText), "mktree")
		args := []string{"commit-tree", tree}
		if parent != "" {
			args = append(args, "-p", parent)
		}
		parent = testGit(t, m.CommonGit, []byte(fmt.Sprintf("native fixture %d\n", i)), args...)
	}
	var err error
	m.History, err = reconstructHistory(m.CommonGit, parent, count)
	if err != nil {
		t.Fatal(err)
	}
	capture, err := filepath.Abs("capture.init.gradle")
	if err != nil {
		t.Fatal(err)
	}
	m.Driver = "GRADLE"
	m.DaemonPolicy = "REPLICATION"
	m.Limits.RetryPairs = 0
	m.Command = []string{filepath.Join(gradleHome, "bin/gradle"), "--offline", "--max-workers=2", "replay"}
	m.Environment = map[string]string{"PATH": filepath.Join(javaHome, "bin") + ":/usr/bin:/bin", "JAVA_HOME": javaHome, "LANG": "C.UTF-8", "LC_ALL": "C.UTF-8", "TZ": "UTC", "QUAL_GRADLE_HOME": gradleHome, "QUAL_JAVA_HOME": javaHome}
	m.Runtime = []Binding{bound(t, gradleHome), bound(t, javaHome)}
	m.Outputs = OutputPolicy{Rules: []OutputRule{}, GraphCapture: bound(t, capture), Projectors: []Projector{}, Diagnostics: []OutputRule{}}
	m.GeneratedPaths = []string{".gradle/**", "build/**", "out/**"}
	m.Limits.MaxRequestNS = int64(120 * time.Second)
	m.Limits.MaxRunNS = int64(10 * time.Minute)
	m.Limits.MaxBytes = 3 << 30
	m.Limits.MaxWorkflowStarts = count * 2
	m.Limits.MaxGradleStarts = count * 2
	m.Limits.NestedReservePerRequest = 0
	return m
}

func TestGradleNestedTestKitAccounting(t *testing.T) {
	script := `def nested = layout.buildDirectory.dir('nested-project').get().asFile
def testKit = layout.buildDirectory.dir('native-testkit-home').get().asFile
def result = layout.projectDirectory.file('out/nested.txt').asFile
tasks.register('replay', Exec) {
    outputs.file(result)
    commandLine(System.getenv('QUAL_JAVA_HOME') + '/bin/java', '-cp',
        System.getenv('QUAL_HELPER') + ':' + System.getenv('QUAL_GRADLE_HOME') + '/lib/*:' +
        System.getenv('QUAL_GRADLE_HOME') + '/lib/plugins/*', 'NativeNested',
        System.getenv('QUAL_GRADLE_HOME'), nested.absolutePath, testKit.absolutePath)
    doFirst {
        nested.mkdirs()
        new File(nested, 'settings.gradle').text = "rootProject.name = 'native-nested'\n"
        new File(nested, 'build.gradle').text = "tasks.register('hello') { doLast { println 'native TestKit request' } }\n"
    }
    doLast {
        result.parentFile.mkdirs()
        result.text = 'two native TestKit requests completed\n'
    }
}
`
	m := gradleFixture(t, 1, script)
	helper := filepath.Join(filepath.Dir(m.RunRoot), "native-helper")
	if err := os.Mkdir(helper, 0700); err != nil {
		t.Fatal(err)
	}
	source := `import java.io.File;
import org.gradle.testkit.runner.GradleRunner;
public class NativeNested {
    public static void main(String[] args) {
        for (int i = 0; i < 2; i++) {
            GradleRunner.create().withGradleInstallation(new File(args[0]))
                .withProjectDir(new File(args[1])).withTestKitDir(new File(args[2]))
                .withArguments("hello", "--console=plain", "--no-configuration-cache")
                .forwardOutput().build();
        }
    }
}
`
	java := filepath.Join(helper, "NativeNested.java")
	mustWrite(t, java, []byte(source))
	args := []string{"-cp", filepath.Join(m.Environment["QUAL_GRADLE_HOME"], "lib/plugins/gradle-test-kit-9.7.1.jar"), "-d", helper, java}
	compile := exec.Command(filepath.Join(m.Environment["QUAL_JAVA_HOME"], "bin/javac"), args...)
	compile.Env = []string{"PATH=/usr/bin:/bin", "LANG=C.UTF-8"}
	start := stamp()
	output, err := compile.CombinedOutput()
	mustWrite(t, filepath.Join(helper, "javac.log"), output)
	mustWrite(t, filepath.Join(helper, "javac.json"), jsonBytes(struct {
		Command []string `json:"command"`
		Start   Stamp    `json:"start"`
		End     Stamp    `json:"end"`
		Success bool     `json:"success"`
	}{append([]string{compile.Path}, args...), start, stamp(), err == nil}))
	if err != nil {
		t.Fatalf("native TestKit helper compile: %v %s", err, output)
	}
	m.Environment["QUAL_HELPER"] = helper
	m.Runtime = append(m.Runtime, bound(t, helper))
	m.Limits.NestedReservePerRequest = 2
	m.Limits.MaxGradleStarts = 6
	r := runFixture(t, m)
	if err = r.execute(1, 0, -1); err != nil {
		t.Fatal(err)
	}
	result, err := writeResult(m.RunRoot)
	if err != nil {
		t.Fatal(err)
	}
	if result.WorkflowStarts != 2 || result.ActualGradleStarts != 6 || result.NestedStarts != 4 || result.Slots[0][0].Class != comparable {
		t.Fatalf("nested invocations miscounted: %+v", result)
	}
	for _, arm := range []string{"N", "I"} {
		var capture Capture
		if err = readJSON(filepath.Join(m.RunRoot, "attempts", "r1-000-g0-"+arm, "capture.json"), &capture); err != nil {
			t.Fatal(err)
		}
		pids := map[int]int{}
		for _, build := range capture.GradleBuilds {
			pids[build.PID]++
		}
		reused := false
		for _, count := range pids {
			if count == 2 {
				reused = true
			}
		}
		if !reused {
			t.Fatal("fixture did not exercise two requests in one TestKit daemon")
		}
	}
}

func TestGradlePersistentNoActionReplay(t *testing.T) {
	m := gradleFixture(t, 3, "tasks.register('replay')\n")
	r := runFixture(t, m)
	if err := r.execute(1, 0, -1); err != nil {
		t.Fatal(err)
	}
	result, err := writeResult(m.RunRoot)
	if err != nil {
		t.Fatal(err)
	}
	if result.WorkflowStarts != 6 || result.ActualGradleStarts != 6 || result.NestedStarts != 0 || result.Decision != "FIXTURE_VERIFIED" {
		t.Fatalf("native replay count/classification differs: %+v", result)
	}
	for _, s := range result.Slots[0] {
		if s.Class != comparable || s.NativeActions != 0 || s.CandidateActions != 0 {
			t.Fatalf("native no-action behavior changed: %+v", s)
		}
	}
	if result.Replications[0].NoActionAttributedSavingNS != 0 {
		t.Fatal("no-action fixture earned mechanism credit")
	}
	if err = checkResult(m.RunRoot, filepath.Join(m.RunRoot, "result.json")); err != nil {
		t.Fatal(err)
	}
}

func TestGradleNativeUpToDateAndCacheRestore(t *testing.T) {
	script := `import org.gradle.api.tasks.*
import org.gradle.api.file.RegularFileProperty
@CacheableTask
abstract class NativeReplayCopy extends DefaultTask {
    @InputFile @PathSensitive(PathSensitivity.RELATIVE)
    abstract RegularFileProperty getSourceFile()
    @OutputFile abstract RegularFileProperty getResultFile()
    @TaskAction void copyInput() {
        def result = resultFile.get().asFile
        result.parentFile.mkdirs()
        result.bytes = sourceFile.get().asFile.bytes
    }
}
tasks.register('nativeRemove', Delete) {
    onlyIf { file('source.txt').text.trim() == 'revision 2' }
    delete(layout.buildDirectory.file('replay.txt'))
}
tasks.register('replay', NativeReplayCopy) {
    dependsOn('nativeRemove')
    sourceFile = layout.projectDirectory.file('settings.gradle')
    resultFile = layout.buildDirectory.file('replay.txt')
}
`
	m := gradleFixture(t, 3, script)
	r := runFixture(t, m)
	if err := r.execute(1, 0, -1); err != nil {
		t.Fatal(err)
	}
	result, err := writeResult(m.RunRoot)
	if err != nil {
		t.Fatal(err)
	}
	if result.Decision != "FIXTURE_VERIFIED" || result.ActualGradleStarts != 6 || result.UnknownGradleReservations != 0 {
		t.Fatal("native cache lifecycle not fully accounted")
	}
	for ordinal, want := range []string{"EXECUTED", "UP-TO-DATE", "FROM-CACHE"} {
		for _, arm := range []string{"N", "I"} {
			var c Capture
			path := filepath.Join(m.RunRoot, "attempts", fmt.Sprintf("r1-%03d-g0-%s", ordinal, arm), "capture.json")
			if err = readJSON(path, &c); err != nil {
				t.Fatal(err)
			}
			got := ""
			for _, task := range c.Tasks {
				if task.Identity == ":replay" {
					got = task.Outcome
				}
			}
			if got != want {
				t.Fatalf("ordinal %d arm %s: got %s want %s", ordinal, arm, got, want)
			}
		}
	}
	if err = checkResult(m.RunRoot, filepath.Join(m.RunRoot, "result.json")); err != nil {
		t.Fatal(err)
	}
}
