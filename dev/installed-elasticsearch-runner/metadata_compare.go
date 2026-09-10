package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/gradlecriticalpath"
)

const metadataProposalSHA = "afd73e4d204c6b634de1943379aa2e97bfc29182024551e20e9b9b257017b731"
const seededRATSHA = "7e147521e3ae71aec017fca8c9aff20fc8218fbe26419e81eff41fdc7c3f41aa"

//go:embed metadata-output-contract-v1.json
var acceptedMetadataContract []byte

//go:embed metadata/MetadataProjection.java
var metadataJavaSource []byte

//go:embed metadata/MetadataProjectionTest.java
var metadataJavaTests []byte

type metadataSelector struct {
	Path   string `json:"path"`
	Type   string `json:"type"`
	Owners []struct {
		Identity string `json:"identity"`
		Class    string `json:"class"`
	} `json:"owners"`
}
type metadataOutputContract struct {
	Schema    string             `json:"schemaVersion"`
	Proposal  string             `json:"acceptedProposalSha256"`
	Revision  string             `json:"sourceRevision"`
	Gradle    string             `json:"gradleVersion"`
	Dates     string             `json:"dateContractSha256"`
	Selectors []metadataSelector `json:"compilerSelectors"`
	RAT       struct {
		Path  string `json:"path"`
		Owner string `json:"owner"`
	} `json:"licenseReport"`
}
type metadataTool struct {
	Java   fileBinding   `json:"java"`
	JDK    []entry       `json:"jdkInventory"`
	Helper fileBinding   `json:"helper"`
	Jars   []fileBinding `json:"gradleJars"`
}
type metadataRuntime struct {
	Schema        string       `json:"schemaVersion"`
	ContractSHA   string       `json:"contractSha256"`
	SourceSHA     string       `json:"sourceSha256"`
	TestsSHA      string       `json:"testsSha256"`
	Decision      fileBinding  `json:"ownerDecision"`
	Tool          metadataTool `json:"tool"`
	Qualification fileBinding  `json:"qualification"`
	SeedCapture   fileBinding  `json:"seedCapture"`
}
type metadataSession struct {
	Runtime  metadataRuntime
	Contract metadataOutputContract
	// Scoped to one capture/replay, keyed by already rehashed raw bytes. These
	// memoized projections cannot replace raw inventory or tool binding checks.
	Compiler map[string]string
	Reports  map[string][]byte
}

func loadMetadataContract() (metadataOutputContract, error) {
	var c metadataOutputContract
	if err := json.Unmarshal(acceptedMetadataContract, &c); err != nil {
		return c, err
	}
	if c.Schema != "buildopt.eic/metadata-output-contract/v1" || c.Proposal != metadataProposalSHA || c.Revision != makeProtocol().Revisions[0] || c.Gradle != "9.7.1" || c.Dates != digest(acceptedDateContract) || len(c.Selectors) != 227 || c.RAT.Path != ratOutputPath || c.RAT.Owner != ratOwner {
		return c, errors.New("accepted metadata contract drift")
	}
	seen := map[string]bool{}
	for _, rule := range c.Selectors {
		if !filepath.IsLocal(rule.Path) || filepath.Clean(rule.Path) != rule.Path || strings.ContainsAny(rule.Path, "\\\r\n\t") || !strings.HasSuffix(rule.Path, "/previous-compilation-data.bin") || seen[rule.Path] || (rule.Type != "file" && rule.Type != "absent") || len(rule.Owners) != 1 || rule.Owners[0].Identity == "" || rule.Owners[0].Class != "org.gradle.api.tasks.compile.JavaCompile_Decorated" {
			return c, errors.New("metadata selector drift")
		}
		seen[rule.Path] = true
	}
	return c, nil
}

func (tool metadataTool) verify() error {
	if len(tool.Jars) == 0 || len(tool.Jars) > 1000 || filepath.Base(tool.Java.Path) != "java" || filepath.Ext(tool.Helper.Path) != ".jar" {
		return errors.New("frozen metadata tool required")
	}
	seen := map[string]bool{}
	compiler := false
	for _, pin := range append([]fileBinding{tool.Java, tool.Helper}, tool.Jars...) {
		if seen[pin.Path] {
			return errors.New("duplicate metadata tool input")
		}
		seen[pin.Path] = true
		if err := checkBinding(pin); err != nil {
			return err
		}
	}
	for _, jar := range tool.Jars {
		parent := filepath.Dir(jar.Path)
		if filepath.Ext(jar.Path) != ".jar" || (!strings.HasSuffix(parent, "/gradle-9.7.1/lib") && !strings.HasSuffix(parent, "/gradle-9.7.1/lib/plugins")) {
			return errors.New("unexpected metadata classpath input")
		}
		if filepath.Base(jar.Path) == "gradle-java-compiler-worker-9.7.1.jar" {
			compiler = true
		}
	}
	if !compiler {
		return errors.New("locked Gradle serializer missing")
	}
	actual, err := inventory(filepath.Dir(filepath.Dir(tool.Java.Path)), []string{"bin", "conf", "lib", "release"})
	if err != nil || len(tool.JDK) == 0 || !reflect.DeepEqual(actual, tool.JDK) {
		return errors.New("metadata JDK inventory drift")
	}
	return nil
}

func newMetadataSession(binding *fileBinding) (*metadataSession, error) {
	if binding == nil {
		return nil, nil
	}
	if err := checkBinding(*binding); err != nil {
		return nil, err
	}
	var r metadataRuntime
	if err := readJSON(binding.Path, &r); err != nil {
		return nil, err
	}
	if r.Schema != "buildopt.eic/metadata-runtime/v1" || r.ContractSHA != digest(acceptedMetadataContract) || r.SourceSHA != digest(metadataJavaSource) || r.TestsSHA != digest(metadataJavaTests) {
		return nil, errors.New("metadata implementation identity drift")
	}
	for _, pin := range []fileBinding{r.Decision, r.Qualification, r.SeedCapture} {
		if err := checkBinding(pin); err != nil {
			return nil, err
		}
	}
	// The owner decision is a retained record of the explicit approval, not a
	// proposal whose status can silently be changed during a stopped campaign.
	var decision map[string]any
	if err := readJSON(r.Decision.Path, &decision); err != nil {
		return nil, err
	}
	proposal, ok := decision["proposal"].(map[string]any)
	if !ok || proposal["sha256"] != metadataProposalSHA || decision["decision"] != "ACCEPTED" || decision["ownerMessage"] != "Aprobado" {
		return nil, errors.New("exact metadata owner approval required")
	}
	var q struct {
		Schema       string        `json:"schemaVersion"`
		SourceSHA    string        `json:"sourceSha256"`
		TestsSHA     string        `json:"testsSha256"`
		Tool         metadataTool  `json:"tool"`
		Passed       bool          `json:"passed"`
		Checks       []fileBinding `json:"checks"`
		GradleStarts int           `json:"gradleStarts"`
	}
	if err := readJSON(r.Qualification.Path, &q); err != nil {
		return nil, err
	}
	if q.Schema != "buildopt.eic/metadata-qualification/v1" || !q.Passed || q.SourceSHA != r.SourceSHA || q.TestsSHA != r.TestsSHA || !reflect.DeepEqual(q.Tool, r.Tool) || len(q.Checks) < 3 || q.GradleStarts != 0 {
		return nil, errors.New("metadata comparator qualification required")
	}
	for _, pin := range q.Checks {
		if err := checkBinding(pin); err != nil {
			return nil, err
		}
	}
	if err := r.Tool.verify(); err != nil {
		return nil, err
	}
	c, err := loadMetadataContract()
	if err != nil {
		return nil, err
	}
	s := &metadataSession{Runtime: r, Contract: c, Compiler: map[string]string{}, Reports: map[string][]byte{}}
	if r.SeedCapture.SHA256 != "17120c2cc69e1387f3676ef547b5a376b61ec6bc3a8cbe1ec9eb1901e560f03e" {
		return nil, errors.New("approved cached RAT origin drift")
	}
	seed, err := loadComparisonCapture(r.SeedCapture)
	if err != nil {
		return nil, err
	}
	if _, err := metadataOwner(seed, ratOutputPath, ratOwner, ""); err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(filepath.Join(seed.Root, ratOutputPath))
	if err != nil {
		return nil, err
	}
	if digest(raw) != seededRATSHA {
		return nil, errors.New("cached RAT seed bytes drift")
	}
	// This exact report is the approved prior-capture trust anchor. D001 retained
	// it UP-TO-DATE; do not pretend its timestamp came from D001's execution.
	projection, err := projectRAT(raw, seed.Workspace, nativeTimeWindow{StartMs: 1788780168000, EndMs: 1788780168999})
	if err != nil {
		return nil, err
	}
	s.Reports[seededRATSHA] = projection
	return s, nil
}

func metadataOwner(c comparisonCapture, path, owner, class string) (gradlecriticalpath.Task, error) {
	owners := 0
	for _, task := range c.Graph.Tasks {
		for _, output := range task.Outputs {
			if path == output || strings.HasPrefix(path, output+"/") {
				owners++
				if task.Identity != owner || (class != "" && task.Class != class) {
					return gradlecriticalpath.Task{}, errors.New("metadata owner identity drift")
				}
			}
		}
	}
	var found gradlecriticalpath.Task
	count := 0
	for _, task := range c.Report.Tasks {
		if task.Identity == owner {
			found = task
			count++
		}
	}
	if owners != 1 || count != 1 {
		return found, errors.New("missing or ambiguous metadata owner")
	}
	return found, nil
}

func (s *metadataSession) projectCompiler(paths []string, entries []entry) error {
	if len(paths) == 0 {
		return nil
	}
	if err := s.Runtime.Tool.verify(); err != nil {
		return err
	}
	cp := []string{s.Runtime.Tool.Helper.Path}
	for _, jar := range s.Runtime.Tool.Jars {
		cp = append(cp, jar.Path)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, s.Runtime.Tool.Java.Path, "-Xmx512m", "-cp", strings.Join(cp, string(os.PathListSeparator)), "MetadataProjection")
	cmd.Env = []string{"LANG=C.UTF-8", "LC_ALL=C.UTF-8"}
	cmd.Stdin = strings.NewReader(strings.Join(paths, "\n") + "\n")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	raw, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("metadata decoder failed: %w: %.2000s", err, stderr.String())
	}
	if len(raw) > 128<<10 || stderr.Len() != 0 {
		return errors.New("unexpected metadata decoder output")
	}
	lines := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
	if len(lines) != len(paths) {
		return errors.New("metadata batch result count differs")
	}
	for i, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) != 2 || parts[0] != entries[i].SHA256 || !validSHA256(parts[1]) {
			return errors.New("metadata decoder raw binding differs")
		}
		if err := checkBinding(fileBinding{paths[i], entries[i].SHA256}); err != nil {
			return err
		}
		s.Compiler[parts[0]] = parts[1]
	}
	return s.Runtime.Tool.verify()
}

func (s *metadataSession) projectCapture(c comparisonCapture) (map[string]string, error) {
	byPath := map[string]entry{}
	for _, e := range c.Inventory {
		byPath[e.Path] = e
	}
	var paths []string
	var entries []entry
	for _, rule := range s.Contract.Selectors {
		if _, err := metadataOwner(c, rule.Path, rule.Owners[0].Identity, rule.Owners[0].Class); err != nil {
			return nil, fmt.Errorf("%s: %w", rule.Path, err)
		}
		e, ok := byPath[rule.Path]
		if !ok || e.Type != rule.Type {
			return nil, errors.New("metadata file identity drift")
		}
		if e.Type == "file" && s.Compiler[e.SHA256] == "" {
			paths = append(paths, filepath.Join(c.Root, e.Path))
			entries = append(entries, e)
		}
	}
	if err := s.projectCompiler(paths, entries); err != nil {
		return nil, err
	}
	result := map[string]string{}
	for _, rule := range s.Contract.Selectors {
		e := byPath[rule.Path]
		if e.Type == "file" {
			result[e.Path] = s.Compiler[e.SHA256]
		}
	}
	task, err := metadataOwner(c, ratOutputPath, ratOwner, "")
	if err != nil {
		return nil, err
	}
	e := byPath[ratOutputPath]
	if e.Type != "file" || e.Size > 16<<20 {
		return nil, errors.New("bounded RAT report required")
	}
	raw, err := os.ReadFile(filepath.Join(c.Root, ratOutputPath))
	if err != nil {
		return nil, err
	}
	if digest(raw) != e.SHA256 {
		return nil, errors.New("RAT raw binding differs")
	}
	projected, err := s.projectReport(raw, c.Workspace, task)
	if err != nil {
		return nil, err
	}
	result[ratOutputPath] = digest(projected)
	return result, nil
}

func (s *metadataSession) projectReport(raw []byte, workspace string, task gradlecriticalpath.Task) ([]byte, error) {
	var projected []byte
	var err error
	switch task.Outcome {
	case "EXECUTED":
		projected, err = projectRAT(raw, workspace, nativeTimeWindow{StartMs: task.StartTimeMs, EndMs: task.EndTimeMs})
	case "FROM-CACHE", "UP-TO-DATE":
		projected = s.Reports[digest(raw)]
		if projected == nil {
			err = errors.New("cached RAT lacks byte-identical prior capture provenance")
		}
	default:
		err = errors.New("unsupported RAT task outcome")
	}
	if err != nil {
		return nil, err
	}
	s.Reports[digest(raw)] = projected
	return projected, nil
}
