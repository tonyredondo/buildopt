//go:build linux && amd64

// Native daemon lifecycle parser reused from installed-elasticsearch-runner/testkit_logs.go.
// Original source SHA256: fd578975e10359d2b5ef3e2681bec761a78c60ceb595bd2b7a2cfad5472000d9
// The old command and schema are unchanged; this copy uses replay binding types.
package main

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type GradleBuild struct {
	ID               string  `json:"buildId"`
	PID              int     `json:"pid"`
	DaemonUID        string  `json:"daemonUid"`
	JavaHome         string  `json:"javaHome"`
	ClientDirectory  string  `json:"clientWorkingDirectory"`
	ReceivedMs       int64   `json:"receivedTimeMs"`
	StartedMs        int64   `json:"commandStartTimeMs"`
	ExecutionStartMs int64   `json:"executionStartTimeMs"`
	ExecutionEndMs   int64   `json:"executionEndTimeMs"`
	FinishedMs       int64   `json:"commandFinishTimeMs"`
	Log              Binding `json:"log"`
}

var GradleBuildIdentity = regexp.MustCompile(`Build\{id=([a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}), currentDir=([^\r\n}]+)\}`)
var daemonContextIdentity = regexp.MustCompile(`DefaultDaemonContext\[uid=([a-f0-9-]{36}),javaHome=([^,]+),.*?,pid=([0-9]+),`)

// A TestKit daemon can execute multiple requests. Count its complete command
// lifecycle, not JVM processes or init-script callbacks (which CC can skip).
// Wall-clock fields correlate lifecycle evidence; they are not value timings.
func readDaemonBuilds(binding Binding) ([]GradleBuild, error) {
	if err := checkBinding(binding); err != nil {
		return nil, err
	}
	name := filepath.Base(binding.Path)
	if !strings.HasPrefix(name, "daemon-") || !strings.HasSuffix(name, ".out.log") {
		return nil, errors.New("invalid daemon log name")
	}
	pid, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(name, "daemon-"), ".out.log"))
	if err != nil || pid <= 0 {
		return nil, errors.New("invalid daemon PID")
	}
	f, err := os.Open(binding.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.Size() > 64<<20 {
		return nil, errors.New("daemon log exceeds bound")
	}
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 4096), 1<<20)
	builds := []GradleBuild{}
	var active *GradleBuild
	seen := map[string]bool{}
	const handler = "[org.gradle.launcher.daemon.server.DefaultIncomingConnectionHandler] "
	const execute = "[org.gradle.launcher.daemon.server.exec.ExecuteBuild] "
	for s.Scan() {
		line := s.Text()
		action := ""
		for _, candidate := range []string{"Received command: Build{", "Starting executing command: Build{", "Finishing executing command: Build{"} {
			if strings.Contains(line, handler+candidate) {
				action = candidate
				break
			}
		}
		if action == "" {
			for _, candidate := range []string{"The daemon has started executing the build.", "Executing build with daemon context:", "The daemon has finished executing the build."} {
				if strings.Contains(line, execute+candidate) {
					action = candidate
					break
				}
			}
		}
		if action == "" {
			continue
		}
		stamp, _, ok := strings.Cut(line, " ")
		if !ok {
			return builds, errors.New("missing daemon timestamp")
		}
		parsed, err := time.Parse("2006-01-02T15:04:05.000-0700", stamp)
		if err != nil {
			return builds, err
		}
		at := parsed.UnixMilli()
		if strings.Contains(action, "command:") {
			match := GradleBuildIdentity.FindStringSubmatch(line)
			if len(match) != 3 || !filepath.IsAbs(match[2]) {
				return builds, errors.New("invalid daemon build identity")
			}
			if action == "Received command: Build{" {
				if active != nil || seen[match[1]] || len(builds) >= 1000 {
					return builds, errors.New("duplicate or overlapping daemon build")
				}
				active = &GradleBuild{ID: match[1], PID: pid, ClientDirectory: match[2], ReceivedMs: at, Log: binding}
				seen[active.ID] = true
			} else {
				if active == nil || active.ID != match[1] || active.ClientDirectory != match[2] {
					return builds, errors.New("unmatched daemon command event")
				}
				if action == "Starting executing command: Build{" {
					if active.StartedMs != 0 || at < active.ReceivedMs {
						return builds, errors.New("invalid daemon command start")
					}
					active.StartedMs = at
				} else {
					if active.ExecutionEndMs == 0 || active.DaemonUID == "" || at < active.ExecutionEndMs {
						return builds, errors.New("incomplete daemon execution")
					}
					active.FinishedMs = at
					builds = append(builds, *active)
					active = nil
				}
			}
			continue
		}
		if active == nil || active.StartedMs == 0 {
			return builds, errors.New("execution outside a started daemon command")
		}
		switch action {
		case "The daemon has started executing the build.":
			if active.ExecutionStartMs != 0 || at < active.StartedMs {
				return builds, errors.New("duplicate or reversed execution start")
			}
			active.ExecutionStartMs = at
		case "Executing build with daemon context:":
			match := daemonContextIdentity.FindStringSubmatch(line)
			if len(match) != 4 || match[3] != strconv.Itoa(pid) || active.ExecutionStartMs == 0 || active.JavaHome != "" || !filepath.IsAbs(match[2]) {
				return builds, errors.New("daemon runtime identity drift")
			}
			active.DaemonUID, active.JavaHome = match[1], match[2]
		case "The daemon has finished executing the build.":
			if active.ExecutionStartMs == 0 || active.ExecutionEndMs != 0 || at < active.ExecutionStartMs {
				return builds, errors.New("missing, duplicate or reversed execution finish")
			}
			active.ExecutionEndMs = at
		}
	}
	if err := s.Err(); err != nil {
		return builds, err
	}
	if active != nil {
		return append(builds, *active), errors.New("unfinished daemon build retained")
	}
	if len(builds) == 0 {
		return builds, errors.New("no observed daemon builds")
	}
	return builds, nil
}
