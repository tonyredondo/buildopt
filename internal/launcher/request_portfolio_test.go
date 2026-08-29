package launcher

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/changeaware"
	"github.com/tonyredondo/buildopt/internal/contractcrypto"
	"github.com/tonyredondo/buildopt/internal/requestaligned"
	"github.com/tonyredondo/buildopt/internal/requestportfolio"
	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

func TestStickyWrapperPersistsExactRequestPortfolioAfterNativeBuild(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fixture uses a POSIX Gradle Wrapper-shaped script")
	}
	root := writeStickyNativeNoopFixture(t)
	t.Chdir(root)
	clearStickyNativeIntegrationEnvironment(t)
	output := filepath.Join(root, "private", "requests.json")
	evidencePath := filepath.Join(root, "private", "request-evidence.json")
	marker := filepath.Join(root, "child.txt")
	wrapper := filepath.Join(root, "gradlew")
	argumentsSHA := requestportfolio.ArgumentsSHA256([]string{"test", "--tests", "example.A B", ""})
	placeholder := strings.Repeat("0", 64)
	evidence := requestportfolio.Evidence{
		SchemaVersion: requestportfolio.EvidenceSchemaVersion, ObservationID: placeholder, ArgumentsSHA256: argumentsSHA,
		CompatibilityIdentitySHA256: requestportfolio.CompatibilitySHA256("wrapper", "gradle", "jdk", "environment"),
		RequestedTasks:              []string{":test"},
		RequestGraphIdentitySHA256:  requestportfolio.CompatibilitySHA256("request-graph"),
	}
	template := evidencePath + ".template"
	writeRequestPortfolioEvidence(t, template, evidence)
	script := "#!/bin/sh\numask 077\nsed \"s/" + placeholder + "/${" + requestPortfolioObservationIDEnvironment + "}/\" " + strconv.Quote(template) + " > \"${" + requestPortfolioEvidenceEnvironment + "}\"\n" +
		"printf '%s|%s|%s|%s\\n' \"$#\" \"${" + requestPortfolioOutputEnvironment + "-}\" \"${" + requestPortfolioEvidenceEnvironment + "-}\" \"${" + requestPortfolioObservationIDEnvironment + "-}\" > " + strconv.Quote(marker) + "\n"
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(stickyWrapperRootEnvironment, root)
	t.Setenv(stickyObservationModeEnvironment, "0")
	t.Setenv(requestPortfolioOutputEnvironment, output)
	t.Setenv(requestPortfolioEvidenceEnvironment, evidencePath)
	var stdout, stderr bytes.Buffer
	if got := Run([]string{"run", "--", wrapper, "test", "--tests", "example.A B", ""}, strings.NewReader(""), &stdout, &stderr); got != 0 {
		t.Fatalf("exit = %d; stderr=%s", got, stderr.String())
	}
	portfolio, err := requestportfolio.Load(output)
	if err != nil || len(portfolio.Entries) != 1 {
		t.Fatalf("portfolio = %+v/%v", portfolio, err)
	}
	entry := portfolio.Entries[0]
	if entry.ArgumentsSHA256 != argumentsSHA || entry.WorkingDirectoryEvidence != "EXACT" || !entry.CandidateEligible || entry.Lifecycle != "EVIDENCE_COMPLETE" || entry.ObservationCount != 1 || entry.Outcomes.Success != 1 {
		t.Fatalf("entry = %+v; stderr=%s", entry, stderr.String())
	}
	child, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(strings.TrimSpace(string(child)), "|")
	if len(parts) != 4 || parts[0] != "6" || parts[1] != "" || parts[2] != evidencePath || len(parts[3]) != 64 {
		t.Fatalf("child arguments/environment = %q", child)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected diagnostics: %s", stderr.String())
	}
}

func TestRequestPortfolioMaterializesSameInvocationCapture(t *testing.T) {
	root := writeStickyNativeNoopFixture(t)
	t.Chdir(root)
	private := filepath.Join(root, "private")
	evidencePath := filepath.Join(private, "request-evidence.json")
	capturePath := filepath.Join(private, "request-capture.json")
	t.Setenv(requestPortfolioEvidenceEnvironment, evidencePath)
	t.Setenv(requestPortfolioCaptureEnvironment, capturePath)
	started := time.Date(2026, 8, 28, 18, 0, 0, 0, time.UTC)
	args := []string{filepath.Join(root, gradleWrapperName(runtime.GOOS)), ":bundle", "--no-daemon"}
	state := newRequestPortfolioStateAt(root, args, started)
	prepared := state.prepareChild(args)
	if len(prepared) != len(args)+2 || prepared[len(args)] != "--init-script" ||
		prepared[len(args)+1] != evidencePath+".init.gradle" ||
		state.argumentsSHA != requestportfolio.ArgumentsSHA256(args[1:]) {
		t.Fatalf("prepared arguments/state = %q/%+v", prepared, state)
	}
	digest := func(value string) string { return strings.Repeat(value, 64) }
	capture := requestaligned.Capture{
		SchemaVersion: requestaligned.CaptureSchemaVersion, GeneratedAt: started.Format(time.RFC3339Nano),
		Status: requestaligned.CaptureComplete, GradleArguments: args[1:], RequestedTasks: []string{":bundle"},
		GradleVersion: "9.6.1", JavaRuntime: requestaligned.JavaRuntime{
			Version: "21.0.12", Vendor: "Eclipse Adoptium", RuntimeName: "OpenJDK Runtime Environment",
			VMName: "OpenJDK 64-Bit Server VM", Architecture: "amd64",
		},
		EnvironmentBindingSHA256: digest("a"),
		WrapperFiles:             []requestaligned.FileBinding{{Path: "gradle/wrapper/gradle-wrapper.properties", SHA256: digest("b")}},
		BuildLogicFiles:          []requestaligned.FileBinding{{Path: "build.gradle", SHA256: digest("c")}},
		Tasks: []changeaware.TaskEvidence{{Path: ":bundle", Outputs: []changeaware.OutputEvidence{{
			Path: "build/bundle.bin", Kind: "FILE", SHA256: digest("d"), Exists: true,
		}}}},
	}
	raw, err := json.Marshal(capture)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(capturePath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := state.materializeEvidence(); err != nil {
		t.Fatal(err)
	}
	evidence, err := requestportfolio.LoadEvidence(evidencePath, state.observationID, state.argumentsSHA)
	if err != nil || len(evidence.RequestedTasks) != 1 || evidence.RequestedTasks[0] != ":bundle" {
		t.Fatalf("evidence = %+v/%v", evidence, err)
	}
	state.cleanupCaptureArtifacts()
	if _, err := os.Lstat(state.initScriptPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("init script survived cleanup: %v", err)
	}
}

func TestRequestPortfolioKeepsBypassAndFailureIneligible(t *testing.T) {
	root := writeStickyNativeNoopFixture(t)
	t.Chdir(root)
	clearStickyNativeIntegrationEnvironment(t)
	output := filepath.Join(root, "private", "requests.json")
	evidencePath := filepath.Join(root, "private", "request-evidence.json")
	args := []string{filepath.Join(root, gradleWrapperName(runtime.GOOS)), "help"}
	t.Setenv(requestPortfolioOutputEnvironment, output)
	t.Setenv(requestPortfolioEvidenceEnvironment, evidencePath)
	t.Setenv(bypassEnvironment, "1")
	started := time.Date(2026, 8, 28, 17, 0, 0, 0, time.UTC)
	state := newRequestPortfolioStateAt(root, args, started)
	evidence := requestportfolio.Evidence{
		SchemaVersion:               requestportfolio.EvidenceSchemaVersion,
		ObservationID:               state.observationID,
		ArgumentsSHA256:             requestportfolio.ArgumentsSHA256(args[1:]),
		CompatibilityIdentitySHA256: requestportfolio.CompatibilitySHA256("compatibility"),
		RequestedTasks:              []string{":help"}, RequestGraphIdentitySHA256: requestportfolio.CompatibilitySHA256("graph"),
	}
	writeRequestPortfolioEvidence(t, evidencePath, evidence)
	state.finishGradle(childExecution{started: true, startedAt: started, completedAt: started.Add(time.Second)})
	if err := state.finish(0, started.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	portfolio, err := requestportfolio.Load(output)
	if err != nil || len(portfolio.Entries) != 1 {
		t.Fatalf("portfolio = %+v/%v", portfolio, err)
	}
	entry := portfolio.Entries[0]
	if entry.CandidateEligible || entry.Lifecycle != "INELIGIBLE_OUTCOME" || entry.Outcomes.Bypassed != 1 || entry.EligibleObservationCount != 0 {
		t.Fatalf("bypass entry = %+v", entry)
	}
}

func TestRequestPortfolioPersistsWhenConfiguredServerIsUnavailable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fixture uses a POSIX Gradle Wrapper-shaped script")
	}
	now := time.Now().UTC().Truncate(time.Second)
	storage, err := sharedcache.Open(context.Background(), filepath.Join(t.TempDir(), "server-state"))
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	issued := issueStickyConnectionToken(
		t, storage, now, "example/outage-project", "gradle-9.6.1/linux-amd64/jdk-21/project",
		[]sharedcache.CentralCapability{sharedcache.CentralCacheRead, sharedcache.CentralStateRead},
	)
	root := writeStickyConnectionRepository(t, "https://127.0.0.1:1", "example/outage-project", "BUILDOPT_TEAM_TOKEN")
	t.Chdir(root)
	gradle := stickyConnectionGradleCommand(root)
	if err := os.WriteFile(gradle, []byte("#!/bin/sh\nprintf 'native-command-ran\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	clearStickyNativeIntegrationEnvironment(t)
	output := filepath.Join(root, "private", "requests.json")
	t.Setenv(stickyWrapperRootEnvironment, root)
	t.Setenv(stickyObservationModeEnvironment, "0")
	t.Setenv(requestPortfolioOutputEnvironment, output)
	t.Setenv("BUILDOPT_TEAM_TOKEN", stickyConnectionTokenJSON(t, issued))
	var stdout, stderr bytes.Buffer
	if got := Run([]string{"run", "--", gradle, "help"}, strings.NewReader(""), &stdout, &stderr); got != 0 {
		t.Fatalf("exit = %d; stdout=%s stderr=%s", got, stdout.String(), stderr.String())
	}
	if stdout.String() != "native-command-ran\n" || !strings.Contains(stderr.String(), "connection unavailable; retaining native Gradle") {
		t.Fatalf("outage result stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	portfolio, err := requestportfolio.Load(output)
	if err != nil || len(portfolio.Entries) != 1 {
		t.Fatalf("outage portfolio = %+v/%v", portfolio, err)
	}
	entry := portfolio.Entries[0]
	if entry.Outcomes.Success != 1 || entry.Lifecycle != "OBSERVED_INCOMPLETE" || entry.CandidateEligible {
		t.Fatalf("outage entry = %+v", entry)
	}
}

func writeRequestPortfolioEvidence(t *testing.T, path string, evidence requestportfolio.Evidence) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = contractcrypto.CanonicalizeJCS(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}
