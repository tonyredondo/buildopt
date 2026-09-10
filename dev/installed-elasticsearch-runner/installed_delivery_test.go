package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/wcncpobserve"
)

// Exactly two real B0 help invocations verify persistent delivery through the
// already-installed package. This is an integration gate, not a value sample.
func TestInstalledPersistentElasticsearchDelivery(t *testing.T) {
	root := os.Getenv("EIC_INSTALLED_DELIVERY_ROOT")
	if root == "" {
		t.Skip("requires a separately allocated two-start public integration probe")
	}
	freeze := fileBinding{os.Getenv("EIC_INSTALLED_DELIVERY_PARENT"), os.Getenv("EIC_INSTALLED_DELIVERY_PARENT_SHA256")}
	parent, err := loadCorrectnessFreeze(freeze)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(root) {
		t.Fatal("absolute task-owned root required")
	}
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	group, err := os.ReadFile("/proc/self/cgroup")
	if err != nil || !strings.HasSuffix(strings.TrimSpace(string(group)), "/buildopt-eic-installed-delivery-v1.service") || os.Getenv("INVOCATION_ID") == "" {
		t.Fatal("owned transient service required")
	}
	if err := writeJSON(filepath.Join(root, "ownership.json"), map[string]string{"cgroup": strings.TrimSpace(string(group)), "invocationId": os.Getenv("INVOCATION_ID")}); err != nil {
		t.Fatal(err)
	}
	endpoint, err := url.Parse(parent.Runtime.BackendURL)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Second)
	defer cancel()
	b, err := startCorrectnessBackendAt(ctx, filepath.Join(root, "backend"), parent.Runtime.Package, endpoint.Host)
	if err != nil {
		t.Fatal(err)
	}
	defer b.close()
	runtime := parent.Runtime
	runtime.Credential = b.Credential
	runtime.CA = fileBinding{Path: b.CA}
	runtime.CA.SHA256, err = hashFile(b.CA)
	if err != nil {
		t.Fatal(err)
	}
	if b.URL != runtime.BackendURL {
		t.Fatal("installed endpoint changed")
	}
	if err := writeJSON(filepath.Join(root, "freeze.json"), map[string]any{"parent": freeze, "maximumGradleStarts": 2, "maximumSeconds": 250, "package": runtime.Package, "credential": runtime.Credential, "ca": runtime.CA, "backendURL": b.URL, "performanceAuthority": false, "workflow": "help", "arms": []string{"N1", "W1"}, "uploadDeadlineMilliseconds": 100}); err != nil {
		t.Fatal(err)
	}
	for i, id := range []string{"C003", "C005"} {
		if err := nativeDiskGuard(runtime.Root); err != nil {
			t.Fatal(err)
		}
		step := correctnessStepForTest(t, id)
		tree, err := verifyCorrectnessSource(ctx, runtime, step, step.InputBefore)
		if err != nil {
			t.Fatal(err)
		}
		request, err := correctnessRequestFor(runtime, step)
		if err != nil {
			t.Fatal(err)
		}
		request.Row.ID = []string{"UI001", "UI002"}[i]
		request.Row.Block = "UI"
		request.Row.Workload = "help"
		request.Row.State = "ordinary-help-delivery-probe"
		request.Row.Pair = 1
		for j, arg := range request.Row.Arguments {
			if arg == ":server:precommit" {
				request.Row.Arguments[j] = "help"
			}
		}
		request.Arguments = append([]string{"/usr/bin/taskset", "--cpu-list", "0-7"}, request.Row.Arguments...)
		request.Diagnostic = nil
		request.TimeoutSeconds = 110
		request.SourceTree = tree
		request.Runtime = parent.JDK
		request.BootID, err = bootID()
		if err != nil {
			t.Fatal(err)
		}
		request.ReservedNS, err = bootNow()
		if err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(root, request.Row.ID)
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		selectors := []string{sourceInputs().TaskPath, sourceInputs().OwnerInputPath, "gradlew"}
		if i == 1 {
			selectors = append(selectors, "buildoptw", "buildoptw.bat", ".buildopt/config.toml", ".buildopt/wrapper.properties")
		}
		before, err := retainSelected(request.Directory, filepath.Join(dir, "inputs"), selectors)
		if err != nil {
			t.Fatal(err)
		}
		if err := writeJSON(filepath.Join(dir, "source-before.json"), before); err != nil {
			t.Fatal(err)
		}
		workflow, _ := json.Marshal(request.Row.Arguments[1:])
		for j, value := range request.Environment {
			switch {
			case strings.HasPrefix(value, "BUILDOPT_EIC_NATIVE_TRACE="):
				request.Environment[j] = "BUILDOPT_EIC_NATIVE_TRACE=" + filepath.Join(dir, "native-supervision.json")
			case strings.HasPrefix(value, "WCNCP_OUTBOX_DIR="):
				request.Environment[j] = "WCNCP_OUTBOX_DIR=" + filepath.Join(root, "outbox")
			case strings.HasPrefix(value, "WCNCP_WORKFLOW_SHA256="):
				request.Environment[j] = "WCNCP_WORKFLOW_SHA256=" + digest(workflow)
			}
		}
		if err := writeJSON(filepath.Join(dir, "request.json"), request); err != nil {
			t.Fatal(err)
		}
		env, err := correctnessPrivateEnvironment(runtime, request.Environment)
		if err != nil {
			t.Fatal(err)
		}
		process, err := runConfiguredProcess(ctx, dir, request.Arguments[0], request.Arguments[1:], request.Directory, env, 110*time.Second)
		if err != nil || !process.Started || process.ExitCode != 0 || process.Signal != 0 || process.Outcome != "SUCCESS" {
			t.Fatalf("native help failed: %+v %v", process, err)
		}
		if i == 1 {
			if err := verifyCorrectnessSupervision(dir, process); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := verifyCorrectnessSource(ctx, runtime, step, step.InputAfter); err != nil {
			t.Fatal(err)
		}
		after, err := inventory(request.Directory, selectors)
		if err != nil || !reflect.DeepEqual(before, after) {
			t.Fatal("installed probe source changed", err)
		}
		if err := writeJSON(filepath.Join(dir, "source-after.json"), after); err != nil {
			t.Fatal(err)
		}
		if err := writeJSON(filepath.Join(dir, "verified.json"), map[string]any{"nativeSuccess": true, "sourceUnchanged": true, "performanceAuthority": false}); err != nil {
			t.Fatal(err)
		}
	}
	if err := b.close(); err != nil {
		t.Fatal(err)
	}
	posts := b.observations()
	if err := writeJSON(filepath.Join(root, "backend-posts.json"), posts); err != nil {
		t.Fatal(err)
	}
	pending, err := (wcncpobserve.Outbox{Dir: filepath.Join(root, "outbox", correctnessScopeDigest())}).Pending(32, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 1 || len(posts[0].Facts) != 1 {
		t.Fatal("missing or extra installed observation", len(posts))
	}
	fact := posts[0].Facts[0]
	if err := verifyInstalledDeliveryFact(fact, runtime.Package); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(filepath.Join(root, "result.json"), map[string]any{"gradleStarts": 2, "nativeSuccesses": 2, "backendHTTPStatus": posts[0].Status, "pendingObservations": len(pending), "acknowledged": len(pending) == 0 && posts[0].Status == 201, "performanceAuthority": false, "backendClosed": true}); err != nil {
		t.Fatal(err)
	}
}

func ownerHelpArguments() []string {
	return []string{"--no-daemon", "--console=plain", "--max-workers=8", "--build-cache", "help"}
}

func verifyInstalledDeliveryFact(fact wcncpobserve.ObservationFacts, pkg fileBinding) error {
	redacted, err := wcncpobserve.RedactArguments(correctnessScope, ownerHelpArguments())
	if err != nil {
		return err
	}
	workflow, _ := json.Marshal(ownerHelpArguments())
	if err := fact.Validate(); err != nil {
		return err
	}
	if fact.Child.ExitCode == nil || *fact.Child.ExitCode != 0 || fact.Child.Outcome != "SUCCESS" || fact.Authority.ProspectiveGateInput || fact.Duration.Classification == "CONTROLLED_VALUE_INPUT" || !reflect.DeepEqual(fact.Arguments, redacted) || fact.Bindings.WorkflowSHA256 != digest(workflow) || fact.Bindings.BuildOptPackageSHA256 != pkg.SHA256 || fact.Bindings.RepositoryRevision != makeProtocol().Revisions[0] {
		return errors.New("installed observation identity, outcome or authority drift")
	}
	return nil
}

func TestRetainedInstalledDelivery(t *testing.T) {
	root := os.Getenv("EIC_INSTALLED_DELIVERY_REPLAY")
	if root == "" {
		t.Skip("requires retained actual installed delivery rows")
	}
	var frozen struct {
		Parent     fileBinding `json:"parent"`
		MaxStarts  int         `json:"maximumGradleStarts"`
		MaxSeconds int         `json:"maximumSeconds"`
		Package    fileBinding `json:"package"`
		Credential fileBinding `json:"credential"`
		CA         fileBinding `json:"ca"`
		BackendURL string      `json:"backendURL"`
		Authority  bool        `json:"performanceAuthority"`
		Workflow   string      `json:"workflow"`
		Arms       []string    `json:"arms"`
		UploadMS   int         `json:"uploadDeadlineMilliseconds"`
	}
	if err := readJSON(filepath.Join(root, "freeze.json"), &frozen); err != nil {
		t.Fatal(err)
	}
	if frozen.MaxStarts != 2 || frozen.MaxSeconds != 250 || frozen.Authority || frozen.Workflow != "help" || frozen.UploadMS != 100 || !reflect.DeepEqual(frozen.Arms, []string{"N1", "W1"}) {
		t.Fatal("integration allocation drift")
	}
	parent, err := loadCorrectnessFreeze(frozen.Parent)
	if err != nil {
		t.Fatal(err)
	}
	for _, binding := range []fileBinding{frozen.Package, frozen.CA, frozen.Credential} {
		if err := checkBinding(binding); err != nil {
			t.Fatal(err)
		}
	}
	last := int64(0)
	for i, id := range []string{"UI001", "UI002"} {
		dir := filepath.Join(root, id)
		var request nativeRequest
		var process processRecord
		var before, after []entry
		for name, target := range map[string]any{"request.json": &request, "process.json": &process, "source-before.json": &before, "source-after.json": &after} {
			if err := readJSON(filepath.Join(dir, name), target); err != nil {
				t.Fatal(err)
			}
		}
		arm := []string{"N1", "W1"}[i]
		entrypoint := []string{"./gradlew", "./buildoptw"}[i]
		wantArgs := append([]string{"/usr/bin/taskset", "--cpu-list", "0-7", entrypoint}, ownerHelpArguments()...)
		if request.Row.ID != id || request.Row.Arm != arm || request.Row.Revision != makeProtocol().Revisions[0] || request.Row.Workload != "help" || request.Row.State != "ordinary-help-delivery-probe" || !reflect.DeepEqual(request.Arguments, wantArgs) || request.Directory != filepath.Join(parent.Runtime.Root, "arms", arm) || request.Root != parent.Runtime.Root || request.TimeoutSeconds != 110 || !reflect.DeepEqual(request.Runtime, parent.JDK) || !process.Started || process.ExitCode != 0 || process.Signal != 0 || process.Outcome != "SUCCESS" || process.StartNS < last || process.EndNS <= process.StartNS || process.EndNS-process.StartNS > int64(110*time.Second) || !reflect.DeepEqual(before, after) {
			t.Fatal("raw native request, source or process drift")
		}
		last = process.EndNS
		selectors := []string{sourceInputs().TaskPath, sourceInputs().OwnerInputPath, "gradlew"}
		if i == 1 {
			selectors = append(selectors, "buildoptw", "buildoptw.bat", ".buildopt/config.toml", ".buildopt/wrapper.properties")
			if err := verifyCorrectnessSupervision(dir, process); err != nil {
				t.Fatal(err)
			}
		}
		retained, err := inventory(filepath.Join(dir, "inputs"), selectors)
		if err != nil || !reflect.DeepEqual(retained, before) {
			t.Fatal("retained input bytes differ", err)
		}
		if err := checkBinding(fileBinding{filepath.Join(dir, "inputs", sourceInputs().TaskPath), sourceInputs().TaskPostimageSHA256}); err != nil {
			t.Fatal(err)
		}
		output, err := os.ReadFile(filepath.Join(dir, "stdout.log"))
		if err != nil || !strings.Contains(string(output), "BUILD SUCCESSFUL") || !strings.Contains(string(output), "21.0.12+8-LTS") || !strings.Contains(string(output), "9.7.1") {
			t.Fatal("actual native runtime or success missing")
		}
	}
	var posts []correctnessPost
	if err := readJSON(filepath.Join(root, "backend-posts.json"), &posts); err != nil {
		t.Fatal(err)
	}
	if len(posts) != 1 || len(posts[0].Facts) != 1 {
		t.Fatal("installed observation missing or duplicated")
	}
	fact := posts[0].Facts[0]
	if err := verifyInstalledDeliveryFact(fact, frozen.Package); err != nil {
		t.Fatal(err)
	}
	bad := fact
	bad.Arguments = ownerHelpArguments()
	if verifyInstalledDeliveryFact(bad, frozen.Package) == nil {
		t.Fatal("unredacted forged observation accepted")
	}
	pending, err := (wcncpobserve.Outbox{Dir: filepath.Join(root, "outbox", correctnessScopeDigest())}).Pending(32, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) == 1 {
		var queued wcncpobserve.ObservationFacts
		if json.Unmarshal(pending[0].Raw, &queued) != nil || !reflect.DeepEqual(queued, fact) {
			t.Fatal("queued fact differs from actual upload")
		}
	}
	var outer processRecord
	var child correctnessSupervision
	if err := readJSON(filepath.Join(root, "UI002/process.json"), &outer); err != nil {
		t.Fatal(err)
	}
	if err := readJSON(filepath.Join(root, "UI002/native-supervision.json"), &child); err != nil {
		t.Fatal(err)
	}
	if posts[0].AtNS < child.End || posts[0].AtNS > outer.EndNS {
		t.Fatal("observation is outside the post-child outer interval")
	}
	if output := os.Getenv("EIC_INSTALLED_DELIVERY_REPLAY_RESULT"); output != "" {
		if err := writeJSON(output, map[string]any{"nativeSuccesses": 2, "sourceAndRuntimeVerified": true, "redactedArgumentsAndWorkflowVerified": true, "pendingObservations": len(pending), "backendHTTPStatus": posts[0].Status, "acknowledged": len(pending) == 0 && posts[0].Status == 201, "postReceiveToOuterEndNanoseconds": outer.EndNS - posts[0].AtNS, "performanceAuthority": false}); err != nil {
			t.Fatal(err)
		}
	}
}
