package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tonyredondo/buildopt/internal/normalizationaware"
)

func main() {
	var root, record, operation string
	flag.StringVar(&root, "root", "", "checkout root")
	flag.StringVar(&record, "patch", "", "patch JSON")
	flag.StringVar(&operation, "operation", "", "apply or revert")
	flag.Parse()
	if root == "" || record == "" || (operation != "apply" && operation != "revert") {
		fmt.Fprintln(os.Stderr, "usage: normalization-aware-cacheability-patch --root DIR --patch FILE --operation apply|revert")
		os.Exit(64)
	}
	raw, err := os.ReadFile(record)
	if err != nil {
		fail(err)
	}
	var patch normalizationaware.Patch
	if err = json.Unmarshal(raw, &patch); err != nil {
		fail(err)
	}
	target := filepath.Join(root, filepath.FromSlash(patch.Path))
	source, err := os.ReadFile(target)
	if err != nil {
		fail(err)
	}
	var next []byte
	if operation == "apply" {
		next, err = normalizationaware.ApplyPatchV2(source, patch)
	} else {
		next, err = normalizationaware.RevertPatchV2(source, patch)
	}
	if err != nil {
		fail(err)
	}
	if err = os.WriteFile(target, next, 0644); err != nil {
		fail(err)
	}
}
func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
