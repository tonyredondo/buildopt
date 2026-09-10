//go:build linux && amd64

// Command history-replay executes and checks the fixed historical viability protocol.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"
)

const manifestSchema = "buildopt.history-replay/manifest/v2"
const observedManifestSchema = "buildopt.history-replay/manifest/v3"
const recordSchema = "buildopt.history-replay/record/v2"
const resultSchema = "buildopt.history-replay/result/v2"
const protocolDigest = "616e2e0cfbc276450cba000fe9afd1132a7c41bf772e5e070bf4547e5feeef41"

type Binding struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type Revision struct {
	Ordinal            int    `json:"ordinal"`
	Commit             string `json:"commit"`
	Parent             string `json:"parent"`
	Tree               string `json:"tree"`
	CommittedUTC       string `json:"committedUTC"`
	ChangedPathsSHA256 string `json:"changedPathsSHA256"`
}

type FilePatch struct {
	Path         string  `json:"path"`
	BeforeSHA256 string  `json:"beforeSHA256"`
	BeforeMode   uint32  `json:"beforeMode"`
	After        Binding `json:"after"`
	AfterMode    uint32  `json:"afterMode"`
}

type Patch struct {
	Identity      string            `json:"identity"`
	Files         []FilePatch       `json:"files"`
	Prerequisites map[string]string `json:"prerequisites"`
}

// A qualification may compare an already installed control against a revision.
// Its control baseline replaces the shared baseline in N; I still uses Baseline
// followed by Candidate. Omission preserves historical manifest semantics.
func baselineForArm(m Manifest, arm string) Patch {
	if arm == "N" && m.ControlBaseline != nil {
		return *m.ControlBaseline
	}
	return m.Baseline
}

func allRecipes(m Manifest) []Patch {
	patches := []Patch{m.Baseline, m.Candidate}
	if m.ControlBaseline != nil {
		patches = append(patches, *m.ControlBaseline)
	}
	return patches
}

type Layer struct {
	Kind        string  `json:"kind"`
	Source      Binding `json:"source"`
	Destination string  `json:"destination"`
}

type OutputRule struct {
	Path     string `json:"path"`
	Producer string `json:"producer"`
	// Transform is exact or an index into the frozen projector registry.
	Transform string `json:"transform"`
}

type Projector struct {
	Identity      string   `json:"identity"`
	Executable    Binding  `json:"executable"`
	Arguments     []string `json:"arguments"`
	Qualification Binding  `json:"qualification"`
}

type OutputPolicy struct {
	Rules        []OutputRule `json:"rules"`
	GraphCapture Binding      `json:"graphCapture"`
	Projectors   []Projector  `json:"projectors"`
	Diagnostics  []OutputRule `json:"diagnostics"`
	Owner        Binding      `json:"owner"`
}

type Limits struct {
	DeadlineUTC             string `json:"deadlineUTC"`
	MaxRequestNS            int64  `json:"maxRequestNS"`
	MaxRunNS                int64  `json:"maxRunNS"`
	MaxBytes                int64  `json:"maxBytes"`
	MinimumFreeBytes        uint64 `json:"minimumFreeBytes"`
	MaxWorkflowStarts       int    `json:"maxWorkflowStarts"`
	MaxGradleStarts         int    `json:"maxGradleStarts"`
	NestedReservePerRequest int    `json:"nestedReservePerRequest"`
	RetryPairs              int    `json:"retryPairs"`
}

type Manifest struct {
	Schema          string            `json:"schema"`
	Phase           string            `json:"phase"`
	Experiment      string            `json:"experiment"`
	Mode            string            `json:"mode"`
	FrozenUTC       string            `json:"frozenUTC"`
	Protocol        Binding           `json:"protocol"`
	Executable      Binding           `json:"executable"`
	Package         []Binding         `json:"package"`
	Subject         string            `json:"subject"`
	CommonGit       string            `json:"commonGit"`
	History         []Revision        `json:"history"`
	RunRoot         string            `json:"runRoot"`
	Replications    int               `json:"replications"`
	PrefixEnd       int               `json:"prefixEnd"`
	ExecutionEnd    int               `json:"executionEnd"`
	DaemonPolicy    string            `json:"daemonPolicy"`
	Driver          string            `json:"driver"`
	Command         []string          `json:"command"`
	Environment     map[string]string `json:"environment"`
	Runtime         []Binding         `json:"runtime"`
	GeneratedPaths  []string          `json:"generatedPaths"`
	Acquisition     []Layer           `json:"acquisition"`
	Baseline        Patch             `json:"baseline"`
	ControlBaseline *Patch            `json:"controlBaseline,omitempty"`
	Candidate       Patch             `json:"candidate"`
	Outputs         OutputPolicy      `json:"outputs"`
	Affinity        string            `json:"affinity"`
	Limits          Limits            `json:"limits"`
	Correctness     Binding           `json:"correctness"`
	Overhead        Binding           `json:"overhead"`
	Subjects        Binding           `json:"subjects"`
	ExternalCosts   []Binding         `json:"externalCosts"`
	Observer        Binding           `json:"observer,omitzero"`
}

// parseJSON rejects duplicate keys before decoding: encoding/json alone keeps
// the last value. Required fields (including explicit empty collections) are
// checked recursively, so Go zero values cannot invent a missing contract.
func parseJSON(d *json.Decoder) (any, error) {
	t, err := d.Token()
	if err != nil {
		return nil, err
	}
	switch t {
	case json.Delim('{'):
		m := map[string]any{}
		for d.More() {
			k, e := d.Token()
			if e != nil {
				return nil, e
			}
			name, ok := k.(string)
			if !ok {
				return nil, errors.New("non-string object key")
			}
			if _, ok := m[name]; ok {
				return nil, fmt.Errorf("duplicate field %q", name)
			}
			v, e := parseJSON(d)
			if e != nil {
				return nil, e
			}
			m[name] = v
		}
		_, err = d.Token()
		return m, err
	case json.Delim('['):
		a := []any{}
		for d.More() {
			v, e := parseJSON(d)
			if e != nil {
				return nil, e
			}
			a = append(a, v)
		}
		_, err = d.Token()
		return a, err
	default:
		if _, ok := t.(json.Delim); ok {
			return nil, errors.New("unexpected delimiter")
		}
		return t, nil
	}
}

func requiredJSON(v any, t reflect.Type, path string) error {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if v == nil {
		return fmt.Errorf("null is not allowed at %s", path)
	}
	switch t.Kind() {
	case reflect.Struct:
		m, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("expected object at %s", path)
		}
		allowed := map[string]bool{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			n := strings.Split(f.Tag.Get("json"), ",")[0]
			if n == "-" {
				continue
			}
			if n == "" {
				n = f.Name
			}
			allowed[n] = true
			x, ok := m[n]
			if !ok {
				// Only the v2 manifest predates the explicit observation binding.
				// The control baseline override is optional; other fields stay required.
				if t == reflect.TypeOf(Manifest{}) && n == "observer" && m["schema"] == manifestSchema {
					continue
				}
				if t == reflect.TypeOf(Manifest{}) && n == "controlBaseline" {
					continue
				}
				return fmt.Errorf("missing %s.%s", path, n)
			}
			if err := requiredJSON(x, f.Type, path+"."+n); err != nil {
				return err
			}
		}
		for k := range m {
			if !allowed[k] {
				return fmt.Errorf("unknown %s.%s", path, k)
			}
		}
	case reflect.Slice, reflect.Array:
		a, ok := v.([]any)
		if !ok {
			return fmt.Errorf("expected array at %s", path)
		}
		for i, x := range a {
			if err := requiredJSON(x, t.Elem(), fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	case reflect.Map:
		m, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("expected map at %s", path)
		}
		for k, x := range m {
			if err := requiredJSON(x, t.Elem(), path+"."+k); err != nil {
				return err
			}
		}
	}
	return nil
}

func decodeStrict(raw []byte, out any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	v, err := parseJSON(d)
	if err != nil {
		return err
	}
	if _, err = d.Token(); err != io.EOF {
		return errors.New("trailing JSON data")
	}
	if err = requiredJSON(v, reflect.TypeOf(out), "$"); err != nil {
		return err
	}
	d = json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	return d.Decode(out)
}

func readJSON(path string, out any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return decodeStrict(raw, out)
}

var shaPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
var gitPattern = regexp.MustCompile(`^[a-f0-9]{40}$`)
var namePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,95}$`)

func relativePath(p string) bool {
	return p != "" && p != "." && !filepath.IsAbs(p) && filepath.Clean(p) == p && p != ".." && !strings.HasPrefix(p, "../") && !strings.ContainsAny(p, "\x00\r\n\\")
}
func absentBinding(b Binding) bool { return b.Path == "" && b.SHA256 == "" }
func validateBinding(b Binding) error {
	if !filepath.IsAbs(b.Path) || filepath.Clean(b.Path) != b.Path || !shaPattern.MatchString(b.SHA256) {
		return fmt.Errorf("invalid binding %q", b.Path)
	}
	return checkBinding(b)
}

func validateManifest(m Manifest) error {
	if m.Protocol.SHA256 != protocolDigest {
		return errors.New("protocol bytes are not the implemented v2 contract")
	}
	if (m.Schema != manifestSchema && m.Schema != observedManifestSchema) || m.Experiment != "FIXED_NI" || m.Mode != "P" {
		return errors.New("unsupported version, experiment or mode")
	}
	if _, err := readObserverPolicy(m); err != nil {
		return err
	}
	if m.Phase != "QUALIFICATION" && m.Phase != "ENGINEERING" && m.Phase != "CONFIRMATION" {
		return errors.New("unsupported phase")
	}
	if m.ControlBaseline != nil && m.Phase != "QUALIFICATION" {
		return errors.New("control baseline override is qualification-only")
	}
	if m.DaemonPolicy != "REQUEST" && m.DaemonPolicy != "REPLICATION" {
		return errors.New("unsupported daemon policy")
	}
	if m.Driver != "FIXTURE" && m.Driver != "GRADLE" {
		return errors.New("unsupported driver")
	}
	if m.Driver == "FIXTURE" && m.Phase != "QUALIFICATION" {
		return errors.New("fixture cannot run public experiment")
	}
	if m.Replications < 1 || m.Replications > 2 || m.PrefixEnd < 0 || m.ExecutionEnd < 0 || m.ExecutionEnd >= len(m.History) || len(m.History) > 101 {
		return errors.New("invalid horizon")
	}
	if m.Phase != "QUALIFICATION" && (len(m.History) != 101 || m.PrefixEnd != 20) {
		return errors.New("public history must preserve the 100-edge split")
	}
	if m.Phase == "ENGINEERING" && m.ExecutionEnd > 20 {
		return errors.New("engineering cannot consume validation")
	}
	if m.Phase == "CONFIRMATION" && (m.ExecutionEnd != 100 || m.Replications != 2) {
		return errors.New("confirmation requires two complete fresh replications")
	}
	if !namePattern.MatchString(m.Subject) || !filepath.IsAbs(m.CommonGit) || !filepath.IsAbs(m.RunRoot) || filepath.Clean(m.RunRoot) != m.RunRoot {
		return errors.New("invalid repository/run identity")
	}
	if inside(m.CommonGit, m.RunRoot) || inside(m.RunRoot, m.CommonGit) {
		return errors.New("run root overlaps repository")
	}
	freeze, err := time.Parse(time.RFC3339Nano, m.FrozenUTC)
	if err != nil {
		return fmt.Errorf("freeze: %w", err)
	}
	deadline, err := time.Parse(time.RFC3339Nano, m.Limits.DeadlineUTC)
	if err != nil || !deadline.After(freeze) {
		return errors.New("invalid deadline")
	}
	if m.Limits.MaxRequestNS <= 0 || m.Limits.MaxRunNS < m.Limits.MaxRequestNS || m.Limits.MaxBytes <= 0 || m.Limits.MinimumFreeBytes == 0 || m.Limits.MaxWorkflowStarts <= 0 || m.Limits.MaxGradleStarts < 0 || m.Limits.NestedReservePerRequest < 0 || m.Limits.RetryPairs < 0 {
		return errors.New("invalid allocation")
	}
	if m.Limits.MaxRunNS > int64(366*24*time.Hour) || m.Limits.MaxWorkflowStarts > 10000 || m.Limits.MaxGradleStarts > 100000 || m.Limits.RetryPairs > 100 {
		return errors.New("allocation exceeds supported bounded replay")
	}
	if m.DaemonPolicy == "REPLICATION" && m.Limits.RetryPairs != 0 {
		return errors.New("warm daemon state cannot be restored from disk")
	}
	if len(m.Command) == 0 || !(filepath.IsAbs(m.Command[0]) || strings.HasPrefix(m.Command[0], "{repo}/") && relativePath(strings.TrimPrefix(m.Command[0], "{repo}/"))) || len(m.Runtime) == 0 || len(m.Package) == 0 || m.Affinity == "" {
		return errors.New("missing workflow/runtime/package/resources")
	}
	launcherBound := strings.HasPrefix(m.Command[0], "{repo}/") || m.Command[0] == m.Executable.Path
	for _, runtime := range m.Runtime {
		if inside(runtime.Path, m.Command[0]) {
			launcherBound = true
		}
	}
	if !launcherBound {
		return errors.New("native launcher is outside frozen source/runtime bindings")
	}
	if m.Driver == "GRADLE" {
		if filepath.Base(m.Command[0]) != "gradle" && filepath.Base(m.Command[0]) != "gradlew" {
			return errors.New("Gradle driver requires a bound native launcher")
		}
		for _, arg := range m.Command[1:] {
			for _, forbidden := range []string{"--gradle-user-home", "--project-cache-dir", "--project-dir", "--configuration-cache", "--no-build-cache", "--init-script", "-Duser.home", "-Dgradle.user.home", "-Dmaven.repo.local"} {
				if arg == forbidden || strings.HasPrefix(arg, forbidden+"=") {
					return errors.New("workflow overrides owned state or capture policy")
				}
			}
			if arg == "-g" || arg == "-p" || arg == "-I" {
				return errors.New("workflow overrides owned state or capture policy")
			}
		}
		for _, value := range m.Environment {
			if strings.Contains(value, "-Duser.home") || strings.Contains(value, "-Dgradle.user.home") || strings.Contains(value, "-Dmaven.repo.local") {
				return errors.New("environment overrides private JVM state")
			}
		}
	}
	for k, v := range m.Environment {
		if !regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`).MatchString(k) || strings.ContainsAny(v, "\x00\r\n") {
			return errors.New("invalid environment allowlist")
		}
		switch k {
		case "HOME", "GRADLE_USER_HOME", "TMPDIR", "XDG_CACHE_HOME", "BUILDOPT_REPLAY_CAPTURE":
			return fmt.Errorf("owned environment key %s", k)
		}
	}
	bindings := append([]Binding{m.Protocol, m.Executable}, m.Package...)
	bindings = append(bindings, m.Runtime...)
	for _, b := range []Binding{m.Correctness, m.Overhead, m.Subjects} {
		if !absentBinding(b) {
			bindings = append(bindings, b)
		} else if m.Phase == "CONFIRMATION" {
			return errors.New("confirmation missing qualification/subjects binding")
		}
	}
	seen := map[string]string{}
	for _, b := range bindings {
		if err := validateBinding(b); err != nil {
			return err
		}
		if s, ok := seen[b.Path]; ok && s != b.SHA256 {
			return errors.New("conflicting input pins")
		}
		seen[b.Path] = b.SHA256
		if inside(m.RunRoot, b.Path) {
			return errors.New("input cannot derive from run state")
		}
	}
	for _, p := range allRecipes(m) {
		if !namePattern.MatchString(p.Identity) {
			return errors.New("invalid recipe identity")
		}
		paths := map[string]bool{}
		for path, hash := range p.Prerequisites {
			if !relativePath(path) || path == ".git" || strings.HasPrefix(path, ".git/") || !shaPattern.MatchString(hash) {
				return errors.New("invalid source prerequisite")
			}
		}
		for _, f := range p.Files {
			if !relativePath(f.Path) || paths[f.Path] || strings.HasPrefix(f.Path, ".git/") || f.Path == ".git" {
				return errors.New("invalid/duplicate patch path")
			}
			paths[f.Path] = true
			if f.BeforeSHA256 != "" && !shaPattern.MatchString(f.BeforeSHA256) {
				return errors.New("invalid preimage")
			}
			if f.AfterMode != 0644 && f.AfterMode != 0755 {
				return errors.New("unsupported patch mode")
			}
			if f.BeforeSHA256 == "" && f.BeforeMode != 0 {
				return errors.New("absent preimage has mode")
			}
			if f.BeforeSHA256 != "" && f.BeforeMode != 0644 && f.BeforeMode != 0755 {
				return errors.New("unsupported preimage mode")
			}
			if err := validateBinding(f.After); err != nil {
				return err
			}
		}
	}
	for _, n := range m.Baseline.Files {
		for _, i := range m.Candidate.Files {
			if n.Path == i.Path {
				return errors.New("baseline/candidate recipe overlap")
			}
		}
	}
	for _, p := range m.GeneratedPaths {
		if !relativePath(p) || p == ".git" || strings.HasPrefix(p, ".git/") {
			return errors.New("invalid generated path")
		}
	}
	for _, l := range m.Acquisition {
		allowed := map[string]string{"dependencies": "gradle/caches/modules-2", "wrapper": "gradle/wrapper/dists", "jdks": "gradle/jdks"}
		if allowed[l.Kind] == "" || l.Destination != allowed[l.Kind] || inside(m.RunRoot, l.Source.Path) {
			return errors.New("unapproved acquisition layer or future state")
		}
		if err := validateBinding(l.Source); err != nil {
			return err
		}
		entries, err := inventory(l.Source.Path)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			// Inspect the layer, not its parent checkout name. A repository
			// called buildopt may legitimately hold downloaded dependencies.
			for _, component := range strings.Split(filepath.ToSlash(entry.Path), "/") {
				if strings.HasPrefix(component, "build-cache-") || component == "buildopt" || component == "executionHistory" {
					return errors.New("acquisition contains task/product state")
				}
			}
		}
	}
	if _, err := readOwnerPolicy(m); err != nil {
		return err
	}
	transforms := map[string]bool{"exact": true}
	for _, p := range m.Outputs.Projectors {
		if !namePattern.MatchString(p.Identity) || transforms[p.Identity] {
			return errors.New("invalid/duplicate projector")
		}
		if err := validateBinding(p.Executable); err != nil {
			return err
		}
		if err := validateBinding(p.Qualification); err != nil {
			return err
		}
		if err := checkProjectorProof(p, false); err != nil {
			return err
		}
		transforms[p.Identity] = true
	}
	for _, r := range append(append([]OutputRule{}, m.Outputs.Rules...), m.Outputs.Diagnostics...) {
		if !relativePath(r.Path) || r.Producer == "" || !transforms[r.Transform] {
			return errors.New("invalid output/normalization rule")
		}
	}
	if m.Driver == "GRADLE" {
		if err := validateBinding(m.Outputs.GraphCapture); err != nil {
			return err
		}
		if m.Limits.MaxGradleStarts == 0 {
			return errors.New("unbudgeted Gradle driver")
		}
	} else if len(m.Outputs.Rules) == 0 {
		return errors.New("fixture needs complete output declaration")
	}
	if _, err := externalCosts(m); err != nil {
		return err
	}
	return validateHistory(m.CommonGit, m.History)
}
