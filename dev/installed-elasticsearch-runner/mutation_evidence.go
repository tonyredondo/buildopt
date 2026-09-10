package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
)

type taskSnapshot struct {
	Hash    string
	Class   string
	Loader  string
	Actions []string
	Values  map[string]string
}
type mutationObservation struct {
	Outcome     string
	Process     processRecord
	Cache       taskSnapshot
	UTF8Failure bool
}

func checkMutationTransition(kind string, before, after mutationObservation) error {
	if before.Outcome != "FROM-CACHE" || before.Process.Outcome != "SUCCESS" || before.Process.ExitCode != 0 || !before.Process.Started || !after.Process.Started {
		return errors.New("mutation requires an observed successful cache restore")
	}
	want := "EXECUTED"
	if kind == "all-source-removal" {
		want = "NO-SOURCE"
	}
	if kind == "malformed-utf8" {
		want = "FAILED"
	}
	if after.Outcome != want {
		return errors.New("mutation did not produce its required Gradle outcome")
	}
	if want == "FAILED" {
		if after.Process.Outcome != "FAILURE" || after.Process.ExitCode == 0 || !after.UTF8Failure {
			return errors.New("missing exact invalid UTF-8 failure")
		}
	} else if after.Process.Outcome != "SUCCESS" || after.Process.ExitCode != 0 {
		return errors.New("mutation process failed")
	}
	if want == "NO-SOURCE" {
		return nil
	}
	a, b := before.Cache, after.Cache
	if a.Hash == "" || b.Hash == "" || a.Hash == b.Hash || a.Class == "" || a.Class != b.Class || a.Loader == "" || a.Loader != b.Loader || len(a.Actions) == 0 || !reflect.DeepEqual(a.Actions, b.Actions) {
		return errors.New("cache key must change with stable owner implementation")
	}
	return nil
}

// Gradle snapshots do not carry task paths. Follow their actual operation
// parent chain instead of associating adjacent log lines with the current task.
func mutationSnapshots(path string) (map[string]taskSnapshot, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 4096), 16<<20)
	parents, owners := map[int64]int64{}, map[int64]string{}
	result := map[string]taskSnapshot{}
	for s.Scan() {
		var op struct {
			ID      int64  `json:"id"`
			Parent  int64  `json:"parentId"`
			Start   *int64 `json:"startTime"`
			Details struct {
				Build string `json:"buildPath"`
				Task  string `json:"taskPath"`
			} `json:"details"`
			ResultClass string `json:"resultClassName"`
			Result      struct {
				Hash    string            `json:"hash"`
				Class   string            `json:"implementationClassName"`
				Loader  string            `json:"classLoaderHash"`
				Actions []string          `json:"actionClassLoaderHashes"`
				Values  map[string]string `json:"inputValueHashes"`
			} `json:"result"`
		}
		if err := json.Unmarshal(s.Bytes(), &op); err != nil {
			return nil, err
		}
		if op.Start != nil {
			if _, exists := parents[op.ID]; exists || op.ID <= 0 || len(parents) >= 100000 {
				return nil, errors.New("invalid operation identity or bound")
			}
			parents[op.ID] = op.Parent
			if op.Details.Task != "" && op.Details.Build != "" {
				owner := op.Details.Task
				if op.Details.Build != ":" {
					owner = op.Details.Build + owner
				}
				owners[op.ID] = owner
			}
		}
		if !strings.Contains(op.ResultClass, "SnapshotTaskInputs") {
			continue
		}
		id := op.ID
		for depth := 0; owners[id] == "" && id != 0 && depth < 256; depth++ {
			id = parents[id]
		}
		owner := owners[id]
		if owner == "" {
			return nil, errors.New("snapshot has no owning task")
		}
		if _, exists := result[owner]; exists {
			return nil, errors.New("duplicate task snapshot")
		}
		result[owner] = taskSnapshot{op.Result.Hash, op.Result.Class, op.Result.Loader, op.Result.Actions, op.Result.Values}
	}
	return result, s.Err()
}
