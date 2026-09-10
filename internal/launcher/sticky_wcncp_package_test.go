package launcher

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/sharedcache"
	"github.com/tonyredondo/buildopt/internal/stickywrapper"
)

// This optional integration is mandatory in dev/check-installed-native-correction.
// It consumes the real Linux release package, installs it to an isolated prefix,
// and exercises its generated wrapper against native Gradle and the real TLS API.
func TestStickyWCNCPInstalledPackage(t *testing.T) {
	archive := os.Getenv("EIC_TEST_PACKAGE")
	if archive == "" {
		t.Skip("run dev/check-installed-native-correction for real package proof")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Fatal("package proof requires Linux AMD64")
	}
	sourceRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	archiveBytes, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(archiveBytes))
	t.Logf("package SHA-256 %s (%d bytes)", digest, len(archiveBytes))
	extracted := t.TempDir()
	runEICCommand(t, extracted, nil, "tar", "-xzf", archive, "-C", extracted)
	packageRoot := filepath.Join(extracted, "buildopt-0.0.1-linux-amd64")
	prefix := t.TempDir()
	runEICCommand(t, packageRoot, nil, filepath.Join(packageRoot, "install.sh"), "--prefix", prefix)
	for _, name := range []string{"buildopt", "buildopt-impact", "buildopt-server", "buildopt-edge"} {
		packaged, err := os.ReadFile(filepath.Join(packageRoot, "bin", name))
		if err != nil {
			t.Fatal(err)
		}
		installed, err := os.ReadFile(filepath.Join(prefix, "bin", name))
		if err != nil || !bytes.Equal(packaged, installed) {
			t.Fatalf("installed %s differs: %v", name, err)
		}
	}
	storage, err := sharedcache.Open(context.Background(), filepath.Join(t.TempDir(), "server"))
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	now := time.Now().UTC().Truncate(time.Second)
	issued := issueStickyConnectionToken(t, storage, now, "example/installed-package", "gradle/project", []sharedcache.CentralCapability{sharedcache.CentralStateRead, sharedcache.CentralStateWrite})
	if _, err := storage.GrantWCNCPActor(context.Background(), issued.TokenID, sharedcache.WCNCPActorTrustedObserver, now); err != nil {
		t.Fatal(err)
	}
	handler, err := sharedcache.NewCentralHTTPSHandler(storage)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	marker := filepath.Join(root, "build", "native.txt")
	var downloads, posts, early atomic.Int64
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/0.0.1/") {
			downloads.Add(1)
			_, _ = w.Write(archiveBytes)
			return
		}
		if r.Method == http.MethodPost {
			posts.Add(1)
			if _, err := os.Stat(marker); err != nil {
				early.Add(1)
			}
		}
		handler.ServeHTTP(w, r)
	}))
	server.TLS = &tls.Config{MinVersion: tls.VersionTLS13}
	server.StartTLS()
	defer server.Close()
	ca := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0600); err != nil {
		t.Fatal(err)
	}
	resolver := wcncpPackageResolver{url: server.URL, digest: digest}
	config := stickywrapper.Config{Mode: "observe", ServerURL: server.URL, ProjectScope: "example/installed-package", CredentialEnv: "BUILDOPT_TEAM_TOKEN", TrialBudgetPercent: 5}
	if _, err := (stickywrapper.Generator{Root: root, Resolver: resolver}).Init(context.Background(), config); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"settings.gradle": "rootProject.name = 'installed-native-proof'\n",
		"build.gradle": `tasks.register('nativeProof') {
    doLast {
        assert System.getenv().keySet().findAll { it.startsWith('BUILDOPT_') || it.startsWith('WCNCP_') }.isEmpty()
        def output = layout.buildDirectory.file('native.txt').get().asFile
        output.parentFile.mkdirs()
        output.text = 'native-output\n'
    }
}
`,
		"gradlew": "#!/bin/sh\nexec \"$EIC_NATIVE_WRAPPER\" \"$@\"\n",
	}
	for name, contents := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(contents), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chmod(filepath.Join(root, "gradlew"), 0755); err != nil {
		t.Fatal(err)
	}
	cache := t.TempDir()
	outbox := t.TempDir()
	t.Setenv("EIC_NATIVE_WRAPPER", filepath.Join(sourceRoot, "gradlew"))
	t.Setenv("BUILDOPT_WRAPPER_CACHE_HOME", cache)
	t.Setenv("BUILDOPT_STICKY_OBSERVATION", "wcncp")
	t.Setenv("BUILDOPT_STICKY_CA_FILE", ca)
	t.Setenv("BUILDOPT_TEAM_TOKEN", stickyConnectionTokenJSON(t, issued))
	t.Setenv("WCNCP_OUTBOX_DIR", outbox)
	t.Setenv("WCNCP_BACKEND_TOKEN", "foreign-private-token")
	t.Setenv("WCNCP_REPOSITORY_SCOPE", "foreign/project")
	t.Setenv("CURL_CA_BUNDLE", ca)
	wrapper := filepath.Join(root, "buildoptw")
	args := []string{"--no-daemon", "--offline", "--max-workers=2", "--console=plain", "-q", "nativeProof"}
	native := func() {
		if err := os.Remove(marker); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		runEICCommand(t, root, nil, wrapper, args...)
		raw, err := os.ReadFile(marker)
		if err != nil || string(raw) != "native-output\n" {
			t.Fatalf("native output mismatch: %q %v", raw, err)
		}
	}
	queued := func() int {
		files, err := filepath.Glob(filepath.Join(outbox, "*", "obs-*.json"))
		if err != nil {
			t.Fatal(err)
		}
		return len(files)
	}
	native()
	if downloads.Load() != 1 || posts.Load() != 1 || early.Load() != 0 || queued() != 0 {
		t.Fatalf("cold package downloads=%d posts=%d early=%d queued=%d", downloads.Load(), posts.Load(), early.Load(), queued())
	}
	statusBytes := runEICCommand(t, root, nil, wrapper, "--buildopt", "status", "--json")
	var report stickywrapper.StatusReport
	if err := json.Unmarshal(statusBytes, &report); err != nil || report.NativeCorrection == nil || report.NativeCorrection.State != "OBSERVING" || report.Decision.State != "NATIVE" || report.Economics.NetSavedMs.State != "UNAVAILABLE" {
		t.Fatalf("installed status: %s %v", statusBytes, err)
	}
	server.Close()
	native()
	if downloads.Load() != 1 || queued() != 1 {
		t.Fatal("offline warm reuse lost native observation")
	}
	explanation := runEICCommand(t, root, nil, wrapper, "--buildopt", "explain")
	if !bytes.Contains(explanation, []byte("QUEUED")) {
		t.Fatalf("offline explanation: %s", explanation)
	}
	// A corrupted cached package must never execute. Bootstrap falls back to
	// Gradle, strips private variables, and creates no extra observation.
	cachedBinary := filepath.Join(cache, "buildopt", "wrapper", "distributions", "0.0.1", "linux-amd64", digest, "bin", "buildopt")
	if err := os.WriteFile(cachedBinary+".replacement", []byte("#!/bin/sh\nexit 99\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(cachedBinary+".replacement", cachedBinary); err != nil {
		t.Fatal(err)
	}
	native()
	if queued() != 1 {
		t.Fatal("bootstrap fallback recorded an observation")
	}
	t.Setenv("BUILDOPT_BYPASS", "1")
	native()
	if queued() != 1 {
		t.Fatal("bypass recorded an observation")
	}
}

func runEICCommand(t *testing.T, dir string, env []string, program string, args ...string) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, program, args...)
	command.Dir = dir
	if env != nil {
		command.Env = env
	}
	raw, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s failed: %v\n%s", filepath.Base(program), err, raw)
	}
	return raw
}

type wcncpPackageResolver struct{ url, digest string }

func (r wcncpPackageResolver) Latest(context.Context) (stickywrapper.Release, error) {
	release := stickywrapper.Release{Version: "0.0.1", Distributions: map[string]stickywrapper.Distribution{}}
	for _, platform := range []string{"linux-amd64", "macos-amd64", "macos-arm64", "windows-amd64"} {
		release.Distributions[platform] = stickywrapper.Distribution{URL: r.url + "/0.0.1/" + platform, SHA256: r.digest}
	}
	return release, nil
}
func (r wcncpPackageResolver) Version(ctx context.Context, _ string) (stickywrapper.Release, error) {
	return r.Latest(ctx)
}
