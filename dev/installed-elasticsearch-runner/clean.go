package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type cleanSeed struct {
	Schema    string `json:"schemaVersion"`
	Workspace string `json:"workspaceDigest"`
	State     string `json:"stateDigest"`
}

// Bind all starting files, modes, symlink targets and file mtimes. Project
// metadata is included: identical bytes alone do not imply identical Gradle state.
func cleanSnapshotDigest(root string) (string, error) {
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil || canonical != root || !filepath.IsAbs(root) {
		return "", errors.New("snapshot root must be canonical")
	}
	h := sha256.New()
	encoder := json.NewEncoder(h)
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		record := struct {
			Path    string
			Mode    uint32
			Mtime   int64
			Size    int64
			Content string
		}{Path: rel, Mode: uint32(info.Mode())}
		switch {
		case info.IsDir():
		case info.Mode().IsRegular():
			record.Mtime, record.Size = info.ModTime().UnixNano(), info.Size()
			record.Content, err = hashFile(path)
		case info.Mode()&os.ModeSymlink != 0:
			var resolved string
			resolved, err = filepath.EvalSymlinks(path)
			if err == nil && !strings.HasPrefix(resolved, root+string(filepath.Separator)) {
				return errors.New("snapshot symlink escapes root")
			}
			if err == nil {
				record.Content, err = os.Readlink(path)
			}
		default:
			return errors.New("unsupported snapshot entry")
		}
		if err != nil {
			return err
		}
		return encoder.Encode(record)
	})
	return hex.EncodeToString(h.Sum(nil)), err
}

func currentCleanSeed(root string) (cleanSeed, error) {
	s := cleanSeed{Schema: "buildopt.eic/clean-seed/v1"}
	var err error
	s.Workspace, err = cleanSnapshotDigest(filepath.Join(root, "arms", "N0"))
	if err != nil {
		return s, err
	}
	s.State, err = cleanSnapshotDigest(filepath.Join(root, "state", "N0"))
	return s, err
}

func verifyCleanSeed(root string, binding fileBinding) error {
	if binding.Path != filepath.Join(root, "clean-seed.json") {
		return errors.New("clean seed path drift")
	}
	if err := checkBinding(binding); err != nil {
		return err
	}
	var expected cleanSeed
	if err := readJSON(binding.Path, &expected); err != nil {
		return err
	}
	actual, err := currentCleanSeed(root)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(actual, expected) {
		return errors.New("clean workspace or populated cache differs from frozen seed")
	}
	return nil
}

func cleanDiagnosticRows() []row {
	rows := append([]row{}, makeProtocol().Rows[3:5]...)
	for i := range rows {
		rows[i].ID = "C" + rows[i].ID
		rows[i].Block = "CD"
		rows[i].State = "clean-workspace-populated-cache"
	}
	return rows
}

func makeCleanDiagnosticRequest(root, runner string, r row) (nativeRequest, error) {
	for _, expected := range cleanDiagnosticRows() {
		if reflect.DeepEqual(r, expected) {
			request, err := makeObservedNativeRequest(root, runner, r)
			if err != nil {
				return request, err
			}
			request.Seed = &fileBinding{Path: filepath.Join(root, "clean-seed.json")}
			return request, nil
		}
	}
	return nativeRequest{}, errors.New("unknown clean diagnostic row")
}

func validateCleanSequence(root, slot string) error {
	rows := cleanDiagnosticRows()
	position := -1
	for i, r := range rows {
		if r.ID == slot {
			position = i
		}
	}
	if position < 0 {
		return errors.New("unknown clean diagnostic slot")
	}
	entries, err := os.ReadDir(filepath.Join(root, "attempts"))
	if err != nil {
		return err
	}
	if len(entries) != position {
		return errors.New("clean starts missing, extra or reserved; no replay")
	}
	for i := 0; i < position; i++ {
		dir := filepath.Join(root, "attempts", rows[i].ID)
		var result nativeResult
		if err := readJSON(filepath.Join(dir, "native-result.json"), &result); err != nil {
			return err
		}
		if result.RowID != rows[i].ID || !result.Process.Started || result.Process.Outcome != "SUCCESS" || result.Process.ExitCode != 0 || result.InputState != "VERIFIED" {
			return errors.New("previous clean diagnostic failed")
		}
		if err := verifyNativeDiagnosticComplete(dir); err != nil {
			return err
		}
	}
	return nil
}
