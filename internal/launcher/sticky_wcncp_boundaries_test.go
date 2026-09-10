//go:build linux

package launcher

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/sharedcache"
	"github.com/tonyredondo/buildopt/internal/stickywrapper"
	"github.com/tonyredondo/buildopt/internal/wcncpobserve"
	"golang.org/x/sys/unix"
)

type eicBoundary struct {
	StartNS  int64  `json:"startBootNanoseconds"`
	EndNS    int64  `json:"endBootNanoseconds"`
	PID      int    `json:"pid"`
	ExitCode int    `json:"exitCode"`
	Cgroup   string `json:"cgroup"`
}

func eicBootTime() int64 {
	var ts unix.Timespec
	if unix.ClockGettime(unix.CLOCK_BOOTTIME, &ts) != nil {
		panic("CLOCK_BOOTTIME unavailable")
	}
	return ts.Nano()
}

func eicWriteNew(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err = f.Write(append(raw, '\n')); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// This helper brackets a real command on both sides of the installed wrapper.
// The instrumentation is local qualification only, never a value observation.
func TestStickyWCNCPBoundaryHelper(t *testing.T) {
	for i, arg := range os.Args {
		if arg != "--eic-owned-boundary" {
			continue
		}
		args := os.Args[i+1:]
		if len(args) < 2 {
			os.Exit(64)
		}
		p := eicBoundary{ExitCode: -1, StartNS: eicBootTime()}
		cgroup, e := os.ReadFile("/proc/self/cgroup")
		if e != nil {
			os.Exit(65)
		}
		p.Cgroup = strings.TrimSpace(string(cgroup))
		cmd := exec.Command(args[1], args[2:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Start()
		if err == nil {
			p.PID = cmd.Process.Pid
			err = cmd.Wait()
			p.ExitCode = cmd.ProcessState.ExitCode()
		}
		p.EndNS = eicBootTime()
		if eicWriteNew(args[0], p) != nil {
			os.Exit(65)
		}
		if err != nil {
			if p.ExitCode > 0 {
				os.Exit(p.ExitCode)
			}
			os.Exit(66)
		}
		os.Exit(0)
	}
}

// Opt-in retained package qualification. Four actual Gradle starts are frozen
// before execution; all services and credentials belong to this local fixture.
func TestStickyWCNCPInstalledBoundaries(t *testing.T) {
	root, pkg, gradle, jdk := os.Getenv("EIC_BOUNDARY_ROOT"), os.Getenv("EIC_TEST_PACKAGE"), os.Getenv("EIC_BOUNDARY_GRADLE"), os.Getenv("EIC_BOUNDARY_JDK")
	if root == "" {
		t.Skip("retained installed boundary qualification not selected")
	}
	for _, p := range []string{root, pkg, gradle, jdk} {
		if !filepath.IsAbs(p) || filepath.Clean(p) != p {
			t.Fatal("absolute frozen paths required")
		}
	}
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"project", "cache", "outbox", "gradle-home", "tmp", "user", "rows", "server"} {
		if err := os.Mkdir(filepath.Join(root, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	packageBytes, err := os.ReadFile(pkg)
	if err != nil {
		t.Fatal(err)
	}
	packageSHA := fmt.Sprintf("%x", sha256.Sum256(packageBytes))
	if packageSHA != "3eb84a331caf2b78346ff1001053f23b5243b05943c298e5380a569a893ae456" {
		t.Fatal("retained installed package drift")
	}
	helper, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	helperBytes, err := os.ReadFile(helper)
	if err != nil {
		t.Fatal(err)
	}
	runtimeSHA := eicVerifyBoundaryRuntime(t, jdk, gradle)
	serverRoot := os.Getenv("EIC_BOUNDARY_SERVER_ROOT")
	if serverRoot == "" {
		serverRoot = filepath.Join(root, "server")
	} else {
		if !filepath.IsAbs(serverRoot) || filepath.Clean(serverRoot) != serverRoot {
			t.Fatal("canonical local backend directory required")
		}
		if err = os.Mkdir(serverRoot, 0700); err != nil {
			t.Fatal(err)
		}
	}
	storage, err := sharedcache.Open(context.Background(), serverRoot)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	now := time.Now().UTC().Truncate(time.Second)
	issued := issueStickyConnectionToken(t, storage, now, "example/installed-boundaries", "gradle/project", []sharedcache.CentralCapability{sharedcache.CentralStateRead, sharedcache.CentralStateWrite})
	if _, err = storage.GrantWCNCPActor(context.Background(), issued.TokenID, sharedcache.WCNCPActorTrustedObserver, now); err != nil {
		t.Fatal(err)
	}
	handler, err := sharedcache.NewCentralHTTPSHandler(storage)
	if err != nil {
		t.Fatal(err)
	}
	type observedBatch struct {
		AtNS  int64                           `json:"receivedBootNanoseconds"`
		Facts []wcncpobserve.ObservationFacts `json:"facts"`
	}
	var mutex sync.Mutex
	batches := []observedBatch{}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/0.0.1/") {
			_, _ = w.Write(packageBytes)
			return
		}
		if r.Method == http.MethodPost {
			at := eicBootTime()
			raw, e := io.ReadAll(io.LimitReader(r.Body, 1<<20))
			if e != nil {
				http.Error(w, "read failed", 400)
				return
			}
			r.Body.Close()
			r.Body = io.NopCloser(bytes.NewReader(raw))
			var facts []wcncpobserve.ObservationFacts
			if json.Unmarshal(raw, &facts) != nil {
				http.Error(w, "invalid observations", 400)
				return
			}
			mutex.Lock()
			batches = append(batches, observedBatch{at, facts})
			mutex.Unlock()
		}
		handler.ServeHTTP(w, r)
	}))
	server.TLS = &tls.Config{MinVersion: tls.VersionTLS13}
	server.StartTLS()
	defer server.Close()
	ca := filepath.Join(root, "ca.pem")
	if err = os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0600); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(root, "project")
	config := stickywrapper.Config{Mode: "observe", ServerURL: server.URL, ProjectScope: "example/installed-boundaries", CredentialEnv: "BUILDOPT_TEAM_TOKEN", TrialBudgetPercent: 5}
	if _, err = (stickywrapper.Generator{Root: project, Resolver: wcncpPackageResolver{url: server.URL, digest: packageSHA}}).Init(context.Background(), config); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"settings.gradle": "rootProject.name = 'installed-boundary-qualification'\n",
		"build.gradle": `tasks.register('nativeProof') {
    def fail = providers.gradleProperty('eicFail').isPresent()
    def marker = layout.buildDirectory.file('native.txt')
    doLast {
        assert System.getenv().keySet().findAll { it.startsWith('BUILDOPT_') || it.startsWith('WCNCP_') }.isEmpty()
        if (fail) { throw new GradleException('eic expected native failure') }
        marker.get().asFile.parentFile.mkdirs()
        marker.get().asFile.text = 'native-output\n'
    }
}
`,
		"gradlew": "#!/bin/sh\nexec \"$EIC_BOUNDARY_HELPER\" -test.run='^TestStickyWCNCPBoundaryHelper$' -- --eic-owned-boundary \"$EIC_BOUNDARY_CHILD_RECORD\" \"$EIC_BOUNDARY_GRADLE\" \"$@\"\n",
	}
	for name, contents := range files {
		if err = os.WriteFile(filepath.Join(project, name), []byte(contents), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err = os.Chmod(filepath.Join(project, "gradlew"), 0755); err != nil {
		t.Fatal(err)
	}
	sourceRaw, _ := json.Marshal(files)
	sourceSHA := fmt.Sprintf("%x", sha256.Sum256(sourceRaw))
	wrapperBytes, err := os.ReadFile(filepath.Join(project, "buildoptw"))
	if err != nil {
		t.Fatal(err)
	}
	gradleBytes, err := os.ReadFile(gradle)
	if err != nil {
		t.Fatal(err)
	}
	release, err := os.ReadFile(filepath.Join(jdk, "release"))
	if err != nil || !bytes.Contains(release, []byte("21.0.12+8-LTS")) {
		t.Fatal("locked runtime drift", err)
	}
	baseArgs := []string{"--no-daemon", "--offline", "--console=plain", "--max-workers=2", "nativeProof"}
	workflowRaw, _ := json.Marshal(baseArgs)
	environment := map[string]string{
		"PATH": filepath.Join(jdk, "bin") + ":/usr/bin:/bin", "JAVA_HOME": jdk, "JAVA_TOOL_OPTIONS": "-Dfile.encoding=UTF-8 -Duser.home=" + filepath.Join(root, "user"), "GRADLE_USER_HOME": filepath.Join(root, "gradle-home"), "TMPDIR": filepath.Join(root, "tmp"), "LANG": "C.UTF-8", "LC_ALL": "C.UTF-8", "TZ": "UTC",
		"EIC_BOUNDARY_HELPER": helper, "EIC_BOUNDARY_GRADLE": gradle,
		"BUILDOPT_WRAPPER_CACHE_HOME": filepath.Join(root, "cache"), "BUILDOPT_STICKY_OBSERVATION": "wcncp", "BUILDOPT_STICKY_CA_FILE": ca, "BUILDOPT_TEAM_TOKEN": stickyConnectionTokenJSON(t, issued), "CURL_CA_BUNDLE": ca,
		"WCNCP_OUTBOX_DIR": filepath.Join(root, "outbox"), "WCNCP_ENVIRONMENT_CLASS": "CONTROLLED_PERFORMANCE", "WCNCP_PROSPECTIVE_GATE_INPUT": "1",
		"WCNCP_REPOSITORY_REVISION": "b76ded08c952ebb386576fafce4ae2d8fdcc09f1", "WCNCP_SOURCE_TREE_SHA256": sourceSHA, "WCNCP_WRAPPER_SHA256": fmt.Sprintf("%x", sha256.Sum256([]byte(files["gradlew"]))), "WCNCP_PACKAGE_SHA256": packageSHA, "WCNCP_GRADLE_VERSION": "9.7.1", "WCNCP_JDK_SHA256": runtimeSHA, "WCNCP_WORKFLOW_SHA256": fmt.Sprintf("%x", sha256.Sum256(workflowRaw)), "WCNCP_ENVIRONMENT_SHA256": fmt.Sprintf("%x", sha256.Sum256([]byte(jdk+"\x00"+gradle))), "WCNCP_OUTPUT_CONTRACT_SHA256": fmt.Sprintf("%x", sha256.Sum256([]byte("native.txt must contain native-output LF; preserve failure"))), "WCNCP_OUTPUT_MANIFEST_SHA256": fmt.Sprintf("%x", sha256.Sum256([]byte("native-output\n"))),
	}
	// These typed facts exercise the installed recorder's controlled-input
	// branch against a frozen local fixture. The enclosing evidence explicitly
	// has no prospective or Elasticsearch value authority.
	frozenEnv := map[string]string{}
	for k, v := range environment {
		frozenEnv[k] = v
	}
	frozenEnv["BUILDOPT_TEAM_TOKEN"] = "EPHEMERAL_LOCAL_FIXTURE_CREDENTIAL"
	freeze := map[string]any{"schemaVersion": "buildopt.eic/installed-boundary-qualification/v1", "environmentClass": "LOCAL_INSTALLED_BOUNDARY_QUALIFICATION", "maximumGradleStarts": 4, "maximumServiceSeconds": 120, "maximumBytes": 1 << 30, "packageSha256": packageSHA, "helperSha256": fmt.Sprintf("%x", sha256.Sum256(helperBytes)), "wrapperSha256": fmt.Sprintf("%x", sha256.Sum256(wrapperBytes)), "gradleLauncherSha256": fmt.Sprintf("%x", sha256.Sum256(gradleBytes)), "sourceSha256": sourceSHA, "arguments": baseArgs, "environment": frozenEnv, "performanceAuthority": false}
	freeze["localBackendRoot"] = serverRoot
	freeze["maximumBackendBytes"] = 64 << 20
	if err = eicWriteNew(filepath.Join(root, "freeze.json"), freeze); err != nil {
		t.Fatal(err)
	}
	for ordinal := 1; ordinal <= 4; ordinal++ {
		if err = eicBoundaryDiskGuard(root); err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(root, "rows", fmt.Sprintf("QW%03d", ordinal))
		if err = os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		environment["EIC_BOUNDARY_CHILD_RECORD"] = filepath.Join(dir, "child.json")
		entry := filepath.Join(project, "buildoptw")
		if ordinal == 1 {
			entry = filepath.Join(project, "gradlew")
		}
		args := append([]string{}, baseArgs...)
		if ordinal == 4 {
			args = append(args, "-PeicFail=true")
		}
		actualWorkflow, _ := json.Marshal(args)
		environment["WCNCP_WORKFLOW_SHA256"] = fmt.Sprintf("%x", sha256.Sum256(actualWorkflow))
		unit := "buildopt-eic-boundary-" + fmt.Sprintf("%x", sha256.Sum256([]byte(dir)))[:20] + ".service"
		service := []string{"--user", "--quiet", "--wait", "--pipe", "--collect", "--service-type=exec", "--unit=" + unit, "--property=KillMode=control-group", "--property=RuntimeMaxSec=120s", "--property=TimeoutStopSec=2s", "--property=Restart=no", "--working-directory=" + project}
		keys := []string{}
		for k := range environment {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		env := []string{}
		for _, k := range keys {
			if ordinal == 1 && (strings.HasPrefix(k, "BUILDOPT_") || strings.HasPrefix(k, "WCNCP_")) {
				continue
			}
			service = append(service, "--setenv="+k)
			env = append(env, k+"="+environment[k])
		}
		// systemd-run needs the user's existing bus/runtime address, but those
		// variables are not forwarded to the Gradle workload.
		for _, k := range []string{"XDG_RUNTIME_DIR", "DBUS_SESSION_BUS_ADDRESS"} {
			if v := os.Getenv(k); v != "" {
				env = append(env, k+"="+v)
			}
		}
		service = append(service, "--", "/usr/bin/taskset", "--cpu-list", "0-3", helper, "-test.run=^TestStickyWCNCPBoundaryHelper$", "--", "--eic-owned-boundary", filepath.Join(dir, "outer.json"), entry)
		service = append(service, args...)
		if err = eicWriteNew(filepath.Join(dir, "service-request.json"), service); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 130*time.Second)
		command := exec.CommandContext(ctx, "/usr/bin/systemd-run", service...)
		command.Env = env
		raw, runErr := command.CombinedOutput()
		cancel()
		if err = os.WriteFile(filepath.Join(dir, "service.log"), raw, 0600); err != nil {
			t.Fatal(err)
		}
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = exec.CommandContext(stopCtx, "/usr/bin/systemctl", "--user", "stop", unit).Run()
		stopCancel()
		if (runErr != nil) != (ordinal == 4) {
			t.Fatalf("row %d: %v\n%s", ordinal, runErr, raw)
		}
		var outer, child eicBoundary
		for path, target := range map[string]*eicBoundary{"outer.json": &outer, "child.json": &child} {
			b, e := os.ReadFile(filepath.Join(dir, path))
			if e != nil || json.Unmarshal(b, target) != nil {
				t.Fatal("missing boundary", path, e)
			}
		}
		wantExit := 0
		if ordinal == 4 {
			wantExit = 1
		}
		if outer.StartNS <= 0 || outer.StartNS > child.StartNS || child.EndNS >= outer.EndNS || child.StartNS >= child.EndNS || outer.ExitCode != wantExit || child.ExitCode != wantExit || outer.PID <= 0 || child.PID <= 0 {
			t.Fatalf("invalid boundaries: outer=%+v child=%+v", outer, child)
		}
		if !strings.HasPrefix(outer.Cgroup, "0::/") || !strings.HasSuffix(outer.Cgroup, "/"+unit) || child.Cgroup != outer.Cgroup {
			t.Fatal("owned cgroup identity drift")
		}
		cgroupPath := filepath.Join("/sys/fs/cgroup", strings.TrimPrefix(outer.Cgroup, "0::"))
		if _, e := os.Stat(cgroupPath); !os.IsNotExist(e) {
			t.Fatal("owned service cgroup remains", e)
		}
		mutex.Lock()
		captured := append([]observedBatch{}, batches...)
		mutex.Unlock()
		if len(captured) != max(0, ordinal-1) {
			t.Fatalf("row %d: expected %d observed HTTP batches, got %d", ordinal, ordinal-1, len(captured))
		}
		if ordinal > 1 {
			queued, e := filepath.Glob(filepath.Join(root, "outbox", "*", "obs-*.json"))
			if e != nil || len(queued) != 0 {
				t.Fatalf("fast local backend did not acknowledge upload; queued=%d err=%v", len(queued), e)
			}
			batch := captured[len(captured)-1]
			if len(batch.Facts) != 1 {
				t.Fatal("duplicate observations")
			}
			fact := batch.Facts[0]
			if fact.Bindings.WorkflowSHA256 != environment["WCNCP_WORKFLOW_SHA256"] || fact.Bindings.BuildOptPackageSHA256 != packageSHA || fact.Bindings.JDKSHA256 != runtimeSHA {
				t.Fatal("actual invocation binding drift")
			}
			if fact.Duration.ValueMs == nil || fact.Duration.Classification != "CONTROLLED_VALUE_INPUT" || fact.Completeness != "COMPLETE" {
				t.Fatalf("missing measured child duration: %+v", fact.Duration)
			}
			duration := *fact.Duration.ValueMs * int64(time.Millisecond)
			if duration+int64(time.Millisecond) < child.EndNS-child.StartNS || duration > outer.EndNS-outer.StartNS || batch.AtNS < child.EndNS || batch.AtNS > outer.EndNS {
				t.Fatalf("reported child timing outside actual nested boundaries: %d %+v %+v", duration, outer, child)
			}
			if fact.Child.ExitCode == nil || *fact.Child.ExitCode != wantExit {
				t.Fatal("lost native outcome")
			}
			if err = eicWriteNew(filepath.Join(dir, "observation.json"), batch); err != nil {
				t.Fatal(err)
			}
		}
		marker, err := os.ReadFile(filepath.Join(project, "build/native.txt"))
		if err != nil || string(marker) != "native-output\n" {
			t.Fatal("native output changed", err)
		}
		if ordinal == 4 && !bytes.Contains(raw, []byte("eic expected native failure")) {
			t.Fatal("unrelated native failure")
		}
		if err = eicWriteNew(filepath.Join(dir, "verified.json"), map[string]any{"unit": unit, "nativeExitCode": wantExit, "boundariesVerified": true, "performanceAuthority": false}); err != nil {
			t.Fatal(err)
		}
		if err = eicBoundaryDiskGuard(root); err != nil {
			t.Fatal(err)
		}
		var backendBytes int64
		if e := filepath.Walk(serverRoot, func(_ string, info os.FileInfo, e error) error {
			if e != nil {
				return e
			}
			backendBytes += info.Size()
			if backendBytes > 64<<20 {
				return fmt.Errorf("local backend exceeds 64 MiB")
			}
			return nil
		}); e != nil {
			t.Fatal(e)
		}
	}
	if after := eicVerifyBoundaryRuntime(t, jdk, gradle); after != runtimeSHA {
		t.Fatal("runtime changed during capture")
	}
	if err = eicWriteNew(filepath.Join(root, "verified.json"), map[string]any{"environmentClass": "LOCAL_INSTALLED_BOUNDARY_QUALIFICATION", "gradleStarts": 4, "installedObservations": 3, "expectedNativeFailures": 1, "actualNestedBoundariesVerified": true, "performanceAuthority": false}); err != nil {
		t.Fatal(err)
	}
}

func eicVerifyBoundaryRuntime(t *testing.T, jdk, gradle string) string {
	t.Helper()
	path, pin := os.Getenv("EIC_BOUNDARY_RUNTIME_MANIFEST"), os.Getenv("EIC_BOUNDARY_RUNTIME_SHA256")
	raw, err := os.ReadFile(path)
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(raw)) != pin {
		t.Fatal("frozen runtime manifest drift", err)
	}
	type item struct {
		Path   string `json:"path"`
		Type   string `json:"type"`
		Mode   uint32 `json:"mode"`
		Size   int64  `json:"size"`
		SHA256 string `json:"sha256"`
	}
	var manifest struct {
		JDK             string `json:"jdk"`
		Gradle          string `json:"gradle"`
		JDKInventory    []item `json:"jdkInventory"`
		GradleInventory []item `json:"gradleInventory"`
	}
	if json.Unmarshal(raw, &manifest) != nil || manifest.JDK != jdk || filepath.Join(manifest.Gradle, "bin/gradle") != gradle || len(manifest.JDKInventory) == 0 || len(manifest.GradleInventory) == 0 {
		t.Fatal("frozen runtime identity drift")
	}
	for base, entries := range map[string][]item{jdk: manifest.JDKInventory, manifest.Gradle: manifest.GradleInventory} {
		for _, entry := range entries {
			if !filepath.IsLocal(entry.Path) {
				t.Fatal("unsafe runtime member")
			}
			p := filepath.Join(base, entry.Path)
			info, e := os.Lstat(p)
			if e != nil || uint32(info.Mode().Perm()) != entry.Mode {
				t.Fatal("runtime member drift", entry.Path, e)
			}
			if entry.Type == "directory" {
				if !info.IsDir() {
					t.Fatal("runtime directory drift")
				}
				continue
			}
			if entry.Type != "file" || !info.Mode().IsRegular() || info.Size() != entry.Size {
				t.Fatal("runtime file drift")
			}
			f, e := os.Open(p)
			if e != nil {
				t.Fatal(e)
			}
			hash := sha256.New()
			_, e = io.Copy(hash, f)
			f.Close()
			if e != nil || fmt.Sprintf("%x", hash.Sum(nil)) != entry.SHA256 {
				t.Fatal("runtime bytes drift", entry.Path, e)
			}
		}
	}
	encoded, _ := json.Marshal(manifest.JDKInventory)
	return fmt.Sprintf("%x", sha256.Sum256(encoded))
}

func eicBoundaryDiskGuard(root string) error {
	var size int64
	return filepath.Walk(root, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		size += info.Size()
		if size > 1<<30 {
			return fmt.Errorf("installed boundary fixture exceeds 1 GiB")
		}
		return nil
	})
}

// Diagnose transport against retained facts without spending another Gradle
// start or changing the production 100 ms upload deadline.
func TestStickyWCNCPBoundaryTransportReplay(t *testing.T) {
	path := os.Getenv("EIC_BOUNDARY_REPLAY")
	if path == "" {
		t.Skip("no retained transport input")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != os.Getenv("EIC_BOUNDARY_REPLAY_SHA256") {
		t.Fatal("retained observation drift")
	}
	var record struct {
		Facts []json.RawMessage `json:"facts"`
	}
	if json.Unmarshal(raw, &record) != nil || len(record.Facts) != 1 {
		t.Fatal("one retained fact required")
	}
	root := os.Getenv("EIC_BOUNDARY_REPLAY_ROOT")
	if !filepath.IsAbs(root) {
		t.Fatal("new replay root required")
	}
	if err = os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	storage, err := sharedcache.Open(context.Background(), filepath.Join(root, "server"))
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	now := time.Now().UTC().Truncate(time.Second)
	issued := issueStickyConnectionToken(t, storage, now, "example/installed-boundaries", "gradle/project", []sharedcache.CentralCapability{sharedcache.CentralStateRead, sharedcache.CentralStateWrite})
	if _, err = storage.GrantWCNCPActor(context.Background(), issued.TokenID, sharedcache.WCNCPActorTrustedObserver, now); err != nil {
		t.Fatal(err)
	}
	handler, err := sharedcache.NewCentralHTTPSHandler(storage)
	if err != nil {
		t.Fatal(err)
	}
	type response struct {
		Status     int    `json:"status"`
		Body       string `json:"body"`
		DurationNs int64  `json:"durationNanoseconds"`
	}
	var mutex sync.Mutex
	responses := []response{}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		capture := httptest.NewRecorder()
		handler.ServeHTTP(capture, r)
		mutex.Lock()
		responses = append(responses, response{capture.Code, capture.Body.String(), time.Since(start).Nanoseconds()})
		mutex.Unlock()
		for k, v := range capture.Header() {
			w.Header()[k] = v
		}
		w.WriteHeader(capture.Code)
		_, _ = w.Write(capture.Body.Bytes())
	}))
	defer server.Close()
	url, err := wcncpobserve.EndpointForScope(server.URL, optimizePortfolioRepositoryScope("example/installed-boundaries"), "WCNCP_OBSERVATION/batch")
	if err != nil {
		t.Fatal(err)
	}
	results := []map[string]any{}
	for _, deadline := range []time.Duration{100 * time.Millisecond, 2 * time.Second} {
		start := time.Now()
		outcome := wcncpobserve.UploadBatch(context.Background(), server.Client(), url, issued.Token, record.Facts, deadline)
		results = append(results, map[string]any{"deadlineMilliseconds": deadline.Milliseconds(), "durationNanoseconds": time.Since(start).Nanoseconds(), "outcome": outcome})
	}
	server.Close()
	mutex.Lock()
	captured := append([]response{}, responses...)
	mutex.Unlock()
	if err = eicWriteNew(filepath.Join(root, "diagnosis.json"), map[string]any{"responses": captured, "uploads": results, "gradleStarts": 0, "productionDeadlineUnchanged": true}); err != nil {
		t.Fatal(err)
	}
	t.Logf("responses=%+v uploads=%+v", captured, results)
}
