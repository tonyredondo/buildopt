package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func correctnessStepForTest(t *testing.T, id string) correctnessStep {
	t.Helper()
	for _, s := range makeCorrectnessPlan().Steps {
		if s.Row.ID == id {
			return s
		}
	}
	t.Fatal("unknown test row")
	return correctnessStep{}
}
func correctnessStateFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	raw, err := os.ReadFile("testdata/RecoveryState.java.txt")
	if err != nil {
		t.Fatal(err)
	}
	for _, arm := range []string{"N0", "N1", "W1"} {
		for path, data := range map[string][]byte{filepath.Join("arms", arm, sourceInputs().OwnerInputPath): raw, filepath.Join("arms", arm, "server/build/markers/forbiddenPatterns"): []byte("done"), filepath.Join("arms", arm, "server/build/unrelated"): []byte("keep"), filepath.Join("state", arm, "gradle/caches/build-cache-1/entry"): []byte(arm), filepath.Join("state", arm, "gradle/daemon/private.txt"): []byte(arm)} {
			p := filepath.Join(root, path)
			if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, data, 0644); err != nil {
				t.Fatal(err)
			}
		}
	}
	return root
}

func TestCorrectnessTransitionMutatesOnlyTheFrozenInput(t *testing.T) {
	root := correctnessStateFixture(t)
	for _, id := range []string{"C007", "C009", "C012"} {
		step := correctnessStepForTest(t, id)
		r, err := applyCorrectnessTransition(root, step)
		if err != nil {
			t.Fatal(id, err)
		}
		if r.RowID != id {
			t.Fatal("transition identity lost")
		}
		h, err := hashFile(filepath.Join(root, "arms/N0", sourceInputs().OwnerInputPath))
		if err != nil || h != step.InputAfter {
			t.Fatal("input not changed to frozen bytes")
		}
	}
	raw, _ := os.ReadFile(filepath.Join(root, "arms/N0/server/build/unrelated"))
	if string(raw) != "keep" {
		t.Fatal("unrelated output modified")
	}
	h, _ := hashFile(filepath.Join(root, "arms/W1", sourceInputs().OwnerInputPath))
	if h != "09c15178d0ebecb3da1d2efccd73d2e40689d060409c196a7ec2ca2f18c4d886" {
		t.Fatal("other arm modified")
	}
}

func TestCorrectnessTransitionRefusesInputDriftBeforeCacheOrMarkerChanges(t *testing.T) {
	root := correctnessStateFixture(t)
	input := filepath.Join(root, "arms/N1", sourceInputs().OwnerInputPath)
	if err := os.WriteFile(input, []byte("user drift"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := applyCorrectnessTransition(root, correctnessStepForTest(t, "C004")); err == nil {
		t.Fatal("drift accepted")
	}
	raw, err := os.ReadFile(filepath.Join(root, "arms/N1/server/build/markers/forbiddenPatterns"))
	if err != nil || string(raw) != "done" {
		t.Fatal("marker removed before input verification")
	}
}

func TestCorrectnessTransitionTransfersOnlyTheNativeCacheAndPreservesOtherState(t *testing.T) {
	root := correctnessStateFixture(t)
	if _, err := applyCorrectnessTransition(root, correctnessStepForTest(t, "C004")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "arms/N1/server/build/markers/forbiddenPatterns")); !os.IsNotExist(err) {
		t.Fatal("same-root output not removed")
	}
	r, err := applyCorrectnessTransition(root, correctnessStepForTest(t, "C005"))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Cache) == 0 {
		t.Fatal("cache transfer lacks inventory")
	}
	for path, want := range map[string]string{"state/W1/gradle/caches/build-cache-1/entry": "N1", "state/W1/gradle/daemon/private.txt": "W1", "state/N1/gradle/caches/build-cache-1/entry": "N1", "arms/N1/server/build/unrelated": "keep"} {
		raw, err := os.ReadFile(filepath.Join(root, path))
		if err != nil || !bytes.Equal(raw, []byte(want)) {
			t.Fatalf("%s changed incorrectly: %q %v", path, raw, err)
		}
	}
}

func TestCorrectnessTransitionRefusesSymlinkedPrivateState(t *testing.T) {
	root := correctnessStateFixture(t)
	outside := t.TempDir()
	cache := filepath.Join(root, "state/W1/gradle/caches/build-cache-1")
	if err := os.Rename(cache, filepath.Join(root, "old-cache")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "sentinel"), []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, cache); err != nil {
		t.Fatal(err)
	}
	if _, err := applyCorrectnessTransition(root, correctnessStepForTest(t, "C005")); err == nil {
		t.Fatal("symlinked destination accepted")
	}
	raw, _ := os.ReadFile(filepath.Join(outside, "sentinel"))
	if string(raw) != "outside" {
		t.Fatal("outside state changed")
	}
}
