package main

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

const forbiddenMarker = "server/build/markers/forbiddenPatterns"
const nativeCachePath = "gradle/caches/build-cache-1"

//go:embed testdata/RecoveryState.java.txt
var correctnessOwnerInput []byte

type correctnessTransition struct {
	RowID  string  `json:"rowId"`
	Before []entry `json:"before"`
	After  []entry `json:"after"`
	Cache  []entry `json:"copiedCache,omitempty"`
}

func canonicalExisting(path string) error {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || resolved != path {
		return errors.New("noncanonical or symlinked campaign path")
	}
	return nil
}

func correctnessInput(pin string) ([]byte, error) {
	input := sourceInputs()
	if digest(correctnessOwnerInput) != input.OwnerInputSHA256 {
		return nil, errors.New("embedded owner input drift")
	}
	data := append([]byte{}, correctnessOwnerInput...)
	switch pin {
	case input.OwnerInputSHA256:
	case input.BenignSHA256:
		data = append(data, []byte("// buildopt input invalidation fixture\n")...)
	case input.NegativeSHA256:
		data = append(data, []byte("//\tbuildopt input invalidation fixture\n")...)
	default:
		return nil, errors.New("unallocated owner input")
	}
	if digest(data) != pin {
		return nil, errors.New("owner mutation digest drift")
	}
	return data, nil
}

// applyCorrectnessTransition owns only the selected campaign arm. A durable,
// exclusive transition reservation precedes mutations; partial transitions are
// inspectable and cannot silently be retried. Existing worktrees never move.
func applyCorrectnessTransition(root string, step correctnessStep) (result correctnessTransition, retErr error) {
	matched := false
	for _, expected := range makeCorrectnessPlan().Steps {
		if reflect.DeepEqual(step, expected) {
			matched = true
		}
	}
	if !matched || step.Row.Block == "M" {
		return result, errors.New("unallocated source transition")
	}
	if err := canonicalExisting(root); err != nil {
		return result, err
	}
	work := filepath.Join(root, "arms", step.Row.Arm)
	input := filepath.Join(work, sourceInputs().OwnerInputPath)
	if err := canonicalExisting(input); err != nil {
		return result, err
	}
	if h, err := hashFile(input); err != nil || h != step.InputBefore {
		return result, errors.New("owner input differs before transition")
	}
	desired, err := correctnessInput(step.InputAfter)
	if err != nil {
		return result, err
	}
	selectors := []string{sourceInputs().OwnerInputPath, forbiddenMarker}
	result.RowID = step.Row.ID
	result.Before, err = inventory(work, selectors)
	if err != nil {
		return result, err
	}
	marker := filepath.Join(work, forbiddenMarker)
	if step.RemoveMarker {
		if err = canonicalExisting(marker); err != nil {
			return result, err
		}
		raw, e := os.ReadFile(marker)
		if e != nil || string(raw) != "done" {
			return result, errors.New("expected populated owner marker before removal")
		}
	}
	var sourceCache, targetCache string
	if step.CacheFrom != "" {
		sourceCache = filepath.Join(root, "state", step.CacheFrom, nativeCachePath)
		targetCache = filepath.Join(root, "state", step.Row.Arm, nativeCachePath)
		for _, p := range []string{sourceCache, targetCache} {
			if err = canonicalExisting(p); err != nil {
				return result, err
			}
		}
		result.Cache, err = inventory(filepath.Dir(sourceCache), []string{filepath.Base(sourceCache)})
		if err != nil {
			return result, err
		}
		if len(result.Cache) == 0 {
			return result, errors.New("populated source cache required")
		}
	}
	parent := filepath.Join(root, "transitions")
	if err = os.MkdirAll(parent, 0700); err != nil {
		return result, err
	}
	if err = canonicalExisting(parent); err != nil {
		return result, err
	}
	dir := filepath.Join(parent, step.Row.ID)
	if err = os.Mkdir(dir, 0700); err != nil {
		return result, err
	}
	if err = writeJSON(filepath.Join(dir, "before.json"), result); err != nil {
		return result, err
	}
	defer func() {
		if retErr != nil {
			_ = writeJSON(filepath.Join(dir, "failure.json"), map[string]string{"error": retErr.Error()})
		}
	}()
	if sourceCache != "" {
		if err = os.Rename(targetCache, filepath.Join(dir, "cache-before")); err != nil {
			return result, err
		}
		if err = copyTree(sourceCache, targetCache); err != nil {
			return result, err
		}
		for _, p := range []string{sourceCache, targetCache} {
			actual, e := inventory(filepath.Dir(p), []string{filepath.Base(p)})
			if e != nil || !reflect.DeepEqual(actual, result.Cache) {
				return result, errors.New("native cache changed during transfer")
			}
		}
	}
	if step.RemoveMarker {
		if err = os.Remove(marker); err != nil {
			return result, err
		}
	}
	if step.InputBefore != step.InputAfter {
		info, e := os.Stat(input)
		if e != nil {
			return result, e
		}
		temporary := input + ".eic-" + step.Row.ID
		f, e := os.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
		if e != nil {
			return result, e
		}
		_, e = f.Write(desired)
		if e == nil {
			e = f.Sync()
		}
		e = errors.Join(e, f.Close())
		if e != nil {
			return result, e
		}
		if e = os.Rename(temporary, input); e != nil {
			return result, e
		}
	}
	result.After, err = inventory(work, selectors)
	if err != nil {
		return result, err
	}
	if h, e := hashFile(input); e != nil || h != step.InputAfter {
		return result, fmt.Errorf("owner transition %s failed", step.Row.ID)
	}
	return result, writeJSON(filepath.Join(dir, "after.json"), result)
}
