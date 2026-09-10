//go:build linux && amd64

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
)

func command(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: history-replay validate MANIFEST | run MANIFEST | resume RUN_ROOT | check RUN_ROOT | history COMMON_GIT ENDPOINT COUNT OUTPUT")
	}
	switch args[0] {
	case "observer":
		if len(args) != 3 {
			return errors.New("invalid private observer invocation")
		}
		return serveProcessObserver(args[1], args[2])
	case "observer-writer":
		if len(args) != 2 {
			return errors.New("invalid private writer invocation")
		}
		return serveBufferedWriter(args[1])
	case "observer-heartbeat":
		if len(args) != 3 {
			return errors.New("invalid private heartbeat invocation")
		}
		return serveObserverHeartbeat(args[1], args[2])
	case "worker":
		if len(args) != 3 {
			return errors.New("invalid private worker invocation")
		}
		return serveWorker(args[1], args[2])
	case "validate":
		if len(args) != 2 {
			return errors.New("validate requires one manifest")
		}
		var m Manifest
		if err := readJSON(args[1], &m); err != nil {
			return err
		}
		if err := validateManifest(m); err != nil {
			return err
		}
		fmt.Println("manifest valid; no workflow started")
		return nil
	case "history":
		if len(args) != 5 {
			return errors.New("history requires common Git directory, frozen endpoint, count and output")
		}
		count, err := strconv.Atoi(args[3])
		if err != nil {
			return err
		}
		rows, err := reconstructHistory(args[1], args[2], count)
		if err != nil {
			return err
		}
		return writeExclusive(args[4], jsonBytes(rows), 0600)
	case "run":
		if len(args) != 2 {
			return errors.New("run requires one manifest")
		}
		r, err := newRunner(args[1])
		if err != nil {
			return err
		}
		runErr := r.execute(1, 0, -1)
		result, checkErr := writeResult(r.manifest.RunRoot)
		if checkErr != nil {
			return errors.Join(runErr, checkErr)
		}
		if err = json.NewEncoder(os.Stdout).Encode(result); err != nil {
			return err
		}
		return runErr
	case "check":
		if len(args) != 2 {
			return errors.New("check requires one run root")
		}
		if err := checkResult(args[1], args[1]+"/result.json"); err != nil {
			return err
		}
		fmt.Println("raw evidence and claimed result agree")
		return nil
	case "resume":
		if len(args) != 2 {
			return errors.New("resume requires one owned run root")
		}
		r, rep, ordinal, err := resumeRunner(args[1])
		if err != nil {
			return err
		}
		runErr := r.execute(rep, ordinal, -1)
		result, checkErr := writeResult(args[1])
		if checkErr != nil {
			return errors.Join(runErr, checkErr)
		}
		if err = json.NewEncoder(os.Stdout).Encode(result); err != nil {
			return err
		}
		return runErr
	default:
		return fmt.Errorf("unsupported command %q", args[0])
	}
}

func main() {
	if err := command(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
