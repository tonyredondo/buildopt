package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tonyredondo/buildopt/internal/durablenative"
)

type plan struct {
	Patches []plannedPatch `json:"patches"`
}
type plannedPatch struct {
	Family string              `json:"family"`
	Patch  durablenative.Patch `json:"patch"`
}

func main() {
	var planPath, sourceRoot, family, mode string
	flag.StringVar(&planPath, "plan", "benchmarks/results/durable-native-optimization-v1/patch-plan.json", "patch plan")
	flag.StringVar(&sourceRoot, "source-root", "", "checkout to mutate")
	flag.StringVar(&family, "family", "", "family key")
	flag.StringVar(&mode, "mode", "", "apply or revert")
	flag.Parse()
	if sourceRoot == "" || family == "" || (mode != "apply" && mode != "revert") || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: durable-native-patch --source-root DIR --family KEY --mode apply|revert [--plan FILE]")
		os.Exit(64)
	}
	if err := run(planPath, sourceRoot, family, mode); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(planPath, sourceRoot, family, mode string) error {
	root, err := filepath.EvalSymlinks(sourceRoot)
	if err != nil {
		return err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(planPath)
	if err != nil {
		return err
	}
	var input plan
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&input); err != nil {
		return err
	}
	matched := 0
	for _, planned := range input.Patches {
		if planned.Family != family {
			continue
		}
		matched++
		if err := durablenative.ValidatePatch(planned.Patch); err != nil {
			return fmt.Errorf("%s: %w", planned.Patch.Path, err)
		}
		path, err := confinedPath(root, planned.Patch.Path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var next []byte
		if mode == "apply" {
			next, err = durablenative.ApplyPatch(content, planned.Patch)
		} else {
			next, err = durablenative.RevertPatch(content, planned.Patch)
		}
		if err != nil {
			return fmt.Errorf("%s: %w", planned.Patch.Path, err)
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, next, info.Mode().Perm()); err != nil {
			return err
		}
	}
	if matched == 0 {
		return errors.New("family has no planned patches")
	}
	return nil
}

func confinedPath(root, slashPath string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(slashPath))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("patch path escapes source root: %s", slashPath)
	}
	path := filepath.Join(root, clean)
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(root, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("patch path escapes source root: %s", slashPath)
	}
	return resolved, nil
}
