//go:build linux && amd64

package main

import (
	"errors"
	"path/filepath"
)

type DiskObserverReceipt struct {
	Schema           string                 `json:"schema"`
	Attempt          string                 `json:"attempt"`
	Root             string                 `json:"root"`
	MaxBytes         int64                  `json:"maxBytes"`
	MinimumFreeBytes uint64                 `json:"minimumFreeBytes"`
	Begin            Stamp                  `json:"begin"`
	End              Stamp                  `json:"end"`
	Status           map[string]interface{} `json:"status"`
}

func checkDiskReceipt(m Manifest, dir string, native ProcessReceipt) error {
	var receipt DiskObserverReceipt
	if err := readJSON(filepath.Join(dir, "disk-observer.json"), &receipt); err != nil {
		return err
	}
	if receipt.Schema != "buildopt.disk-observer/v1" || receipt.Attempt != native.Attempt || receipt.Root != m.RunRoot || receipt.MaxBytes != m.Limits.MaxBytes || receipt.MinimumFreeBytes != m.Limits.MinimumFreeBytes || checkStampInterval(receipt.Begin, receipt.End) != nil || receipt.Begin.Boot != native.Start.Boot || receipt.Begin.NS < native.Start.NS || receipt.End.NS < native.End.NS {
		return errors.New("disk observer identity, limits or boundaries differ")
	}
	counts := map[string]int64{}
	for _, key := range []string{"checks", "fullScans", "cachedChecks", "watchedInodes"} {
		n, ok := receipt.Status[key].(float64)
		if !ok || n < 0 || n > 1e12 || n != float64(int64(n)) {
			return errors.New("invalid disk observer counters")
		}
		counts[key] = int64(n)
	}
	reason, ok := receipt.Status["fallbackReason"].(string)
	if !ok || len(receipt.Status) != 5 || counts["cachedChecks"]+counts["fullScans"] > counts["checks"] || reason == "" && counts["fullScans"] != 0 || reason != "" && counts["watchedInodes"] != 0 {
		return errors.New("inconsistent disk observer counters")
	}
	return nil
}
