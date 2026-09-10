//go:build linux && amd64

package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type CheckstyleReport struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Path     string `json:"path"`
}
type CheckstyleInputs struct {
	Source    []string           `json:"source"`
	Classpath []string           `json:"classpath"`
	Reports   []CheckstyleReport `json:"reports"`
}
type GraphTask struct {
	Checkstyle   CheckstyleInputs `json:"checkstyle"`
	Identity     string           `json:"identity"`
	Path         string           `json:"path"`
	TaskClass    string           `json:"taskClass"`
	Dependencies []string         `json:"dependencies"`
	Outputs      []string         `json:"outputs"`
}
type GraphEvent struct {
	Kind       string      `json:"kind"`
	Root       string      `json:"root"`
	BuildPath  string      `json:"buildPath"`
	Tasks      []GraphTask `json:"tasks"`
	Task       TaskOutcome `json:"task"`
	UTC        string      `json:"utc"`
	StartedUTC string      `json:"startedUTC"`
}

func readGraph(path, repo string) ([]GraphTask, []TaskOutcome, error) {
	return readGraphScope(path, repo, false)
}

func readGraphScope(path, repo string, rootOnly bool) ([]GraphTask, []TaskOutcome, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 4096), 16<<20)
	tasks := []GraphTask{}
	outcomes := []TaskOutcome{}
	identities := map[string]bool{}
	finished := map[string]bool{}
	for s.Scan() {
		var event GraphEvent
		if err = decodeStrict(s.Bytes(), &event); err != nil {
			return nil, nil, err
		}
		if !inside(repo, event.Root) {
			return nil, nil, errors.New("captured build outside owner workspace")
		}
		if _, err = time.Parse(time.RFC3339Nano, event.UTC); err != nil {
			return nil, nil, err
		}
		if rootOnly && event.BuildPath != ":" {
			continue
		}
		switch event.Kind {
		case "graph":
			for _, task := range event.Tasks {
				if task.Identity == "" || identities[task.Identity] {
					return nil, nil, errors.New("duplicate or missing task identity")
				}
				identities[task.Identity] = true
				for _, path := range task.Outputs {
					if !inside(repo, path) || path == repo {
						return nil, nil, errors.New("unbounded/outside owner output")
					}
				}
				tasks = append(tasks, task)
			}
		case "task":
			if !identities[event.Task.Identity] || finished[event.Task.Identity] {
				return nil, nil, errors.New("task outcome outside graph or duplicated")
			}
			finished[event.Task.Identity] = true
			outcomes = append(outcomes, event.Task)
		default:
			return nil, nil, errors.New("unknown capture event")
		}
	}
	if err = s.Err(); err != nil {
		return nil, nil, err
	}
	if len(tasks) == 0 {
		return nil, nil, errors.New("missing complete workflow graph")
	}
	return tasks, outcomes, nil
}

func resolveTokens(value, repo, armRoot, evidence string) string {
	return strings.NewReplacer("{repo}", repo, "{armRoot}", armRoot, "{home}", filepath.Join(armRoot, "home"), "{gradleHome}", filepath.Join(armRoot, "gradle"), "{tmp}", filepath.Join(armRoot, "tmp"), "{evidence}", evidence).Replace(value)
}
func workflow(m Manifest, rep int, arm, evidence string) ([]string, map[string]string) {
	root := filepath.Join(m.RunRoot, fmt.Sprintf("r%d", rep), arm)
	repo := filepath.Join(root, "repo")
	cmd := []string{}
	env := map[string]string{}
	for _, v := range m.Command {
		cmd = append(cmd, resolveTokens(v, repo, root, evidence))
	}
	for k, v := range m.Environment {
		env[k] = resolveTokens(v, repo, root, evidence)
	}
	env["HOME"] = filepath.Join(root, "home")
	env["GRADLE_USER_HOME"] = filepath.Join(root, "gradle")
	env["TMPDIR"] = filepath.Join(root, "tmp")
	env["XDG_CACHE_HOME"] = filepath.Join(root, "cache")
	env["BUILDOPT_REPLAY_CAPTURE"] = filepath.Join(evidence, "graph.jsonl")
	if m.Driver == "GRADLE" {
		env["JAVA_TOOL_OPTIONS"] = strings.TrimSpace(env["JAVA_TOOL_OPTIONS"] + " -Duser.home=" + strconv.Quote(filepath.Join(root, "home")) + " -Dmaven.repo.local=" + strconv.Quote(filepath.Join(root, "home", ".m2", "repository")))
		cmd = append(cmd, "--no-configuration-cache", "--build-cache", "--console=plain", "--init-script", m.Outputs.GraphCapture.Path)
		if m.DaemonPolicy == "REPLICATION" {
			cmd = append(cmd, "--daemon")
		} else {
			cmd = append(cmd, "--no-daemon")
		}
	}
	return cmd, env
}

func projection(m Manifest, rule OutputRule, path, repo string, native ProcessReceipt) (string, error) {
	if rule.Transform == "exact" {
		return fileDigest(path)
	}
	for _, p := range m.Outputs.Projectors {
		if p.Identity != rule.Transform {
			continue
		}
		if err := checkBinding(p.Executable); err != nil {
			return "", err
		}
		if err := checkBinding(p.Qualification); err != nil {
			return "", err
		}
		args := []string{}
		for _, a := range p.Arguments {
			args = append(args, strings.NewReplacer("{path}", path, "{root}", repo, "{startUTC}", native.Start.UTC, "{endUTC}", native.End.UTC).Replace(a))
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		c := exec.CommandContext(ctx, p.Executable.Path, args...)
		c.Env = []string{"PATH=/usr/bin:/bin", "LANG=C.UTF-8", "TZ=UTC"}
		b, err := c.Output()
		if err != nil {
			return "", err
		}
		value := strings.TrimSpace(string(b))
		if !shaPattern.MatchString(value) {
			return "", errors.New("projector must return one SHA256")
		}
		return value, nil
	}
	return "", errors.New("unqualified normalization")
}

func captureOutputs(m Manifest, repo, evidence string, native ProcessReceipt, previousBuilds map[string]bool) (c Capture, resultErr error) {
	c = Capture{Schema: recordSchema, Outputs: []OutputEntry{}, Tasks: []TaskOutcome{}, Graph: []Binding{}, DaemonLogs: []Binding{}, GradleBuilds: []GradleBuild{}}
	// Invocation evidence survives missing output/graph evidence and native failures.
	if m.Driver == "GRADLE" {
		defer func() {
			var err error
			c.DaemonLogs, c.GradleBuilds, err = collectGradleLogs(filepath.Dir(repo), evidence, native, previousBuilds)
			resultErr = errors.Join(resultErr, err)
		}()
	}
	rules := append([]OutputRule{}, m.Outputs.Rules...)
	if m.Driver == "GRADLE" {
		graph := filepath.Join(evidence, "graph.jsonl")
		tasks, outcomes, err := readGraphScope(graph, repo, !absentBinding(m.Outputs.Owner))
		if err != nil {
			return c, err
		}
		c.Tasks = outcomes
		b, err := bind(graph)
		if err != nil {
			return c, err
		}
		c.Graph = append(c.Graph, b)
		for _, t := range tasks {
			for _, path := range t.Outputs {
				rel, _ := filepath.Rel(repo, path)
				rules = append(rules, OutputRule{rel, t.Identity, "exact"})
			}
		}
	} else {
		path := filepath.Join(evidence, "fixture-tasks.json")
		if err := readJSON(path, &c.Tasks); err != nil {
			return c, err
		}
	}
	owners := map[string]map[string]bool{}
	transforms := map[string]string{}
	for _, rule := range append(rules, m.Outputs.Diagnostics...) {
		if strings.ContainsAny(rule.Path, "*?[") {
			continue
		} // Selectors qualify existing producer outputs; they never invent roots.
		path := filepath.Join(repo, rule.Path)
		if !inside(repo, path) {
			return c, errors.New("output escapes owner")
		}
		entries := []Entry{}
		e, err := entryAt(repo, path)
		if err != nil {
			return c, err
		}
		entries = append(entries, e)
		if e.Kind == "directory" {
			children, err := inventory(path)
			if err != nil {
				return c, err
			}
			for _, child := range children {
				child.Path = filepath.Join(rule.Path, child.Path)
				entries = append(entries, child)
			}
		}
		for _, entry := range entries {
			if owners[entry.Path] == nil {
				owners[entry.Path] = map[string]bool{}
			}
			owners[entry.Path][rule.Producer] = true
			if transforms[entry.Path] == "" {
				transforms[entry.Path] = "exact"
			}
			if rule.Transform != "exact" {
				if transforms[entry.Path] != "exact" && transforms[entry.Path] != rule.Transform {
					return c, errors.New("ambiguous normalization")
				}
				transforms[entry.Path] = rule.Transform
			}
		}
	}
	// A specific declared rule may qualify a file within a captured output dir.
	for path := range owners {
		for _, r := range m.Outputs.Rules {
			if pathPattern(r.Path, path) && r.Transform != "exact" {
				if !owners[path][r.Producer] {
					return c, errors.New("normalizer selector names a different producer")
				}
				if transforms[path] != "exact" && transforms[path] != r.Transform {
					return c, errors.New("ambiguous normalization selector")
				}
				transforms[path] = r.Transform
			}
		}
	}
	paths := []string{}
	for path := range owners {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	if err := os.Mkdir(filepath.Join(evidence, "outputs"), 0700); err != nil {
		return c, err
	}
	copies := []fileCopy{}
	planned := []OutputEntry{}
	for i, path := range paths {
		entry, err := entryAt(repo, filepath.Join(repo, path))
		if err != nil {
			return c, err
		}
		names := []string{}
		for owner := range owners[path] {
			names = append(names, owner)
		}
		sort.Strings(names)
		o := OutputEntry{Entry: entry, Producer: strings.Join(names, ","), Transform: transforms[path]}
		if entry.Kind == "file" {
			dst := filepath.Join(evidence, "outputs", fmt.Sprintf("%06d", i))
			copies = append(copies, fileCopy{filepath.Join(repo, path), dst})
			o.Raw.Path = dst
		}
		planned = append(planned, o)
	}
	if err := retainOutputFiles(copies, filepath.Join(m.RunRoot, "output-objects")); err != nil {
		return c, err
	}
	// Persist output names before the capture/attempt can be sealed.
	if err := syncDir(filepath.Join(evidence, "outputs")); err != nil {
		return c, err
	}
	for _, o := range planned {
		if o.Entry.Kind == "file" {
			var err error
			o.Raw, err = bind(o.Raw.Path)
			if err != nil {
				return c, err
			}
			if o.Raw.SHA256 != o.Entry.SHA256 {
				return c, errors.New("output changed while retaining evidence: " + o.Entry.Path)
			}
			o.ProjectionSHA256, err = projection(m, OutputRule{o.Entry.Path, o.Producer, o.Transform}, o.Raw.Path, repo, native)
			if err != nil {
				return c, err
			}
		} else {
			o.ProjectionSHA256 = objectDigest(o.Entry)
		}
		c.Outputs = append(c.Outputs, o)
	}
	return c, nil
}

func collectGradleLogs(armRoot, evidence string, native ProcessReceipt, previous map[string]bool) ([]Binding, []GradleBuild, error) {
	logs := []Binding{}
	builds := []GradleBuild{}
	paths := []string{}
	err := filepath.WalkDir(armRoot, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if !d.IsDir() && strings.HasPrefix(d.Name(), "daemon-") && strings.HasSuffix(d.Name(), ".out.log") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return logs, builds, err
	}
	if len(paths) > 100 {
		return logs, builds, errors.New("too many daemon logs")
	}
	if err = os.Mkdir(filepath.Join(evidence, "daemon-logs"), 0700); err != nil {
		return logs, builds, err
	}
	seen := map[string]bool{}
	start, err := time.Parse(time.RFC3339Nano, native.Start.UTC)
	if err != nil {
		return logs, builds, err
	}
	end, err := time.Parse(time.RFC3339Nano, native.End.UTC)
	if err != nil {
		return logs, builds, err
	}
	for i, path := range paths {
		dir := filepath.Join(evidence, "daemon-logs", fmt.Sprintf("%03d", i))
		if err = os.Mkdir(dir, 0700); err != nil {
			return logs, builds, err
		}
		dst := filepath.Join(dir, filepath.Base(path))
		if err = copyTree(path, dst); err != nil {
			return logs, builds, err
		}
		b, err := bind(dst)
		if err != nil {
			return logs, builds, err
		}
		logs = append(logs, b)
		observed, parseErr := readDaemonBuilds(b)
		for _, build := range observed {
			if previous[build.ID] {
				continue
			}
			if seen[build.ID] {
				return logs, builds, errors.New("duplicate Gradle command across daemon logs")
			}
			seen[build.ID] = true
			builds = append(builds, build)
			if !inside(armRoot, build.ClientDirectory) || build.ReceivedMs < start.UnixMilli()-1 || build.FinishedMs > end.UnixMilli()+1000 {
				return logs, builds, errors.New("Gradle command outside request/arm")
			}
		}
		if parseErr != nil {
			return logs, builds, parseErr
		}
	}
	if len(builds) == 0 {
		return logs, builds, errors.New("missing outer Gradle command")
	}
	return logs, builds, nil
}

func compareCaptures(n, i Capture) error {
	if n.Schema != recordSchema || i.Schema != recordSchema || len(n.Outputs) != len(i.Outputs) {
		return errors.New("required output coverage differs")
	}
	tasks := func(c Capture) map[string]bool {
		m := map[string]bool{}
		for _, task := range c.Tasks {
			m[task.Identity] = task.Outcome == "FAILED"
		}
		return m
	}
	if !equalJSON(tasks(n), tasks(i)) {
		return errors.New("task identity/failure coverage differs")
	}
	for index, a := range n.Outputs {
		b := i.Outputs[index]
		if a.Entry.Path != b.Entry.Path || a.Entry.Kind != b.Entry.Kind || a.Entry.Mode != b.Entry.Mode || a.Entry.Target != b.Entry.Target || a.Producer != b.Producer || a.Transform != b.Transform || a.ProjectionSHA256 != b.ProjectionSHA256 {
			return fmt.Errorf("required output differs: %s", a.Entry.Path)
		}
	}
	return nil
}

func compareWorkflowGraphs(n, i Capture, nRoot, iRoot string) error {
	if len(n.Graph) != len(i.Graph) {
		return errors.New("workflow graph coverage differs")
	}
	normalized := func(c Capture, root string) ([]GraphTask, error) {
		all := []GraphTask{}
		for _, b := range c.Graph {
			tasks, _, err := readGraph(b.Path, root)
			if err != nil {
				return nil, err
			}
			for _, task := range tasks {
				for j, path := range task.Outputs {
					rel, err := filepath.Rel(root, path)
					if err != nil || !relativePath(rel) {
						return nil, errors.New("graph output outside owner")
					}
					task.Outputs[j] = rel
				}
				all = append(all, task)
			}
		}
		sort.Slice(all, func(i, j int) bool { return all[i].Identity < all[j].Identity })
		return all, nil
	}
	a, err := normalized(n, nRoot)
	if err != nil {
		return err
	}
	b, err := normalized(i, iRoot)
	if err != nil {
		return err
	}
	if !equalJSON(a, b) {
		return errors.New("candidate changes required workflow graph")
	}
	return nil
}

func checkCapture(m Manifest, c Capture, repo, evidence string, native ProcessReceipt) error {
	if c.Schema != recordSchema {
		return errors.New("unknown capture version")
	}
	seen := map[string]bool{}
	for _, o := range c.Outputs {
		if !relativePath(o.Entry.Path) || seen[o.Entry.Path] {
			return errors.New("duplicate/invalid output path")
		}
		seen[o.Entry.Path] = true
		if o.Entry.Kind == "file" {
			if !inside(filepath.Join(evidence, "outputs"), o.Raw.Path) {
				return errors.New("output bytes borrowed from another attempt")
			}
			if err := checkBinding(o.Raw); err != nil {
				return err
			}
			if o.Raw.SHA256 != o.Entry.SHA256 {
				return errors.New("forged output content digest")
			}
			s, err := os.Stat(o.Raw.Path)
			if err != nil {
				return err
			}
			if s.Size() != o.Entry.Size {
				return errors.New("forged output length")
			}
			h, err := projection(m, OutputRule{o.Entry.Path, o.Producer, o.Transform}, o.Raw.Path, repo, native)
			if err != nil {
				return err
			}
			if h != o.ProjectionSHA256 {
				return errors.New("forged normalized output digest")
			}
		} else {
			if o.Entry.Kind != "directory" && o.Entry.Kind != "symlink" && o.Entry.Kind != "absent" {
				return errors.New("unsupported output kind")
			}
			if o.ProjectionSHA256 != objectDigest(o.Entry) {
				return errors.New("forged output metadata")
			}
		}
	}
	for _, b := range append(append([]Binding{}, c.Graph...), c.DaemonLogs...) {
		if !inside(evidence, b.Path) {
			return errors.New("capture artifact outside its attempt")
		}
		if err := checkBinding(b); err != nil {
			return err
		}
	}
	if m.Driver == "GRADLE" {
		actual := []Binding{}
		err := filepath.WalkDir(filepath.Join(evidence, "daemon-logs"), func(path string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if !d.IsDir() {
				b, e := bind(path)
				if e != nil {
					return e
				}
				actual = append(actual, b)
			}
			return nil
		})
		if err != nil {
			return err
		}
		if !equalJSON(actual, c.DaemonLogs) {
			return errors.New("hidden, reordered or missing native daemon log")
		}
	}
	return nil
}

func checkGradleCommands(m Manifest, c Capture, native ProcessReceipt, armRoot string, previous map[string]bool) ([]GradleBuild, error) {
	builds := []GradleBuild{}
	seen := map[string]bool{}
	start, err := time.Parse(time.RFC3339Nano, native.Start.UTC)
	if err != nil {
		return nil, err
	}
	end, err := time.Parse(time.RFC3339Nano, native.End.UTC)
	if err != nil {
		return nil, err
	}
	for _, log := range c.DaemonLogs {
		all, err := readDaemonBuilds(log)
		if err != nil {
			return nil, err
		}
		for _, build := range all {
			if previous[build.ID] {
				continue
			}
			if seen[build.ID] {
				return nil, errors.New("duplicate native command in raw daemon logs")
			}
			seen[build.ID] = true
			if !inside(armRoot, build.ClientDirectory) || build.ReceivedMs < start.UnixMilli()-1 || build.FinishedMs > end.UnixMilli()+1000 {
				return nil, errors.New("raw Gradle command belongs to another request/arm")
			}
			approved := false
			for _, runtime := range m.Runtime {
				if inside(runtime.Path, build.JavaHome) {
					approved = true
				}
			}
			for _, layer := range m.Acquisition {
				if layer.Kind == "jdks" && inside(filepath.Join(armRoot, layer.Destination), build.JavaHome) {
					approved = true
				}
			}
			if !approved {
				return nil, errors.New("Gradle daemon runtime outside frozen toolchain inventory")
			}
			builds = append(builds, build)
		}
	}
	if !equalJSON(builds, c.GradleBuilds) {
		return nil, errors.New("Gradle start summary differs from native command lifecycles")
	}
	if len(builds) == 0 {
		return nil, errors.New("missing outer native Gradle invocation")
	}
	first := builds[0]
	for _, build := range builds {
		if build.ReceivedMs < first.ReceivedMs {
			first = build
		}
	}
	if first.ClientDirectory != filepath.Join(armRoot, "repo") {
		return nil, errors.New("first Gradle invocation is not the ordinary owner workflow")
	}
	return builds, nil
}

func checkOutputCoverage(m Manifest, c Capture, state StatePin) error {
	rules := append(append([]OutputRule{}, m.Outputs.Rules...), m.Outputs.Diagnostics...)
	if m.Driver == "GRADLE" {
		if len(c.Graph) != 1 {
			return errors.New("missing complete native graph")
		}
		tasks, outcomes, err := readGraphScope(c.Graph[0].Path, filepath.Join(state.Root, "repo"), !absentBinding(m.Outputs.Owner))
		if err != nil {
			return err
		}
		if !equalJSON(outcomes, c.Tasks) {
			return errors.New("forged native task outcomes")
		}
		for _, task := range tasks {
			for _, path := range task.Outputs {
				rel, _ := filepath.Rel(filepath.Join(state.Root, "repo"), path)
				rules = append(rules, OutputRule{rel, task.Identity, "exact"})
			}
		}
	} else {
		// The fixture child owns this separate raw task receipt, not the driver.
		if len(c.Tasks) == 0 {
			return errors.New("fixture outcome missing")
		}
	}
	entries := map[string]Entry{}
	for _, e := range state.Entries {
		if strings.HasPrefix(e.Path, "repo/") {
			e.Path = strings.TrimPrefix(e.Path, "repo/")
			entries[e.Path] = e
		}
	}
	owners := map[string]map[string]bool{}
	transforms := map[string]string{}
	for _, r := range rules {
		if strings.ContainsAny(r.Path, "*?[") {
			continue
		}
		paths := []string{r.Path}
		if e, ok := entries[r.Path]; ok && e.Kind == "directory" {
			for path := range entries {
				if strings.HasPrefix(path, r.Path+"/") {
					paths = append(paths, path)
				}
			}
		}
		for _, path := range paths {
			if owners[path] == nil {
				owners[path] = map[string]bool{}
			}
			owners[path][r.Producer] = true
			if transforms[path] == "" {
				transforms[path] = "exact"
			}
			if r.Transform != "exact" {
				if transforms[path] != "exact" && transforms[path] != r.Transform {
					return errors.New("ambiguous recorded output normalization")
				}
				transforms[path] = r.Transform
			}
		}
	}
	if len(owners) != len(c.Outputs) {
		return errors.New("missing/extra output records relative to complete graph and after-state")
	}
	for _, o := range c.Outputs {
		names, ok := owners[o.Entry.Path]
		if !ok {
			return errors.New("output absent from required graph")
		}
		list := []string{}
		for name := range names {
			list = append(list, name)
		}
		sort.Strings(list)
		transform := transforms[o.Entry.Path]
		for _, r := range m.Outputs.Rules {
			if pathPattern(r.Path, o.Entry.Path) && r.Transform != "exact" {
				if !names[r.Producer] {
					return errors.New("recorded normalizer names a different producer")
				}
				if transform != "exact" && transform != r.Transform {
					return errors.New("ambiguous recorded normalization selector")
				}
				transform = r.Transform
			}
		}
		if o.Producer != strings.Join(list, ",") || o.Transform != transform {
			return errors.New("output producer or unqualified normalization differs")
		}
		expected, exists := entries[o.Entry.Path]
		if !exists {
			expected = Entry{Path: o.Entry.Path, Kind: "absent"}
		}
		if !equalJSON(o.Entry, expected) {
			return errors.New("output raw metadata differs from independent after-state inventory")
		}
	}
	return nil
}

// Diagnostic stdout is retained verbatim. Only frozen structured diagnostics
// participate in equivalence; replacing paths in arbitrary console text would
// silently normalize real failures.
func equalJSON(a, b any) bool { return bytes.Equal(jsonBytes(a), jsonBytes(b)) }
