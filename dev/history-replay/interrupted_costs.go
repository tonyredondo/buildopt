//go:build linux && amd64

package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func closeInterruptedCosts(root string, at Stamp) ([]string, error) {
	closed := []string{}
	files, err := filepath.Glob(filepath.Join(root, "costs", "*.start.json"))
	if err != nil {
		return nil, err
	}
	for _, path := range files {
		base := strings.TrimSuffix(path, ".start.json")
		if _, err = os.Stat(base + ".json"); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		var start phaseStart
		if err = readJSON(path, &start); err != nil {
			return nil, err
		}
		if start.At.Boot != at.Boot {
			return nil, errors.New("cannot close interrupted cost across boots")
		}
		end := at
		if err = readJSON(base+".end.json", &end); os.IsNotExist(err) {
			end = at
			if err = writeExclusive(base+".end.json", jsonBytes(end), 0600); err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}
		work := start.At
		if err = readJSON(base+".work-start.json", &work); os.IsNotExist(err) {
			work = start.At
			if err = writeExclusive(base+".work-start.json", jsonBytes(work), 0600); err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}
		c := start.Cost
		c.StartNS = work.NS
		c.EndNS = end.NS
		c.DurationNS = end.NS - work.NS
		if err = writeExclusive(base+".json", jsonBytes(c), 0600); err != nil {
			return nil, err
		}
		closed = append(closed, c.ID)
	}
	return closed, nil
}
