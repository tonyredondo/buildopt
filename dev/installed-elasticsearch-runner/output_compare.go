package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/tonyredondo/buildopt/internal/gradlecriticalpath"
)

type manifestSource struct {
	Member         string `json:"member"`
	SourceJar      string `json:"sourceJar"`
	SourceProducer string `json:"sourceProducer"`
}
type dateOutputRule struct {
	Path               string           `json:"path"`
	Producer           string           `json:"producer"`
	AllowedMembers     []string         `json:"allowedMembers"`
	ManifestProvenance []manifestSource `json:"manifestProvenance"`
}
type dateOutputContract struct {
	Schema           string           `json:"schemaVersion"`
	SourceRevision   string           `json:"sourceRevision"`
	AcceptedProposal string           `json:"acceptedProposalSha256"`
	AllowedFields    []string         `json:"allowedFields"`
	Rules            []string         `json:"rules"`
	Allowlist        []dateOutputRule `json:"allowlist"`
}
type eligibilityResult struct {
	Task                  string `json:"task"`
	Outcome               string `json:"outcome"`
	DurationMs            int64  `json:"durationMs"`
	NativeOuterNS         int64  `json:"nativeOuterNanoseconds"`
	OnHardDependencyChain bool   `json:"onHardDependencyChain"`
	Passed                bool   `json:"passed"`
}
type nativeComparison struct {
	Schema                   string              `json:"schemaVersion"`
	ContractSHA256           string              `json:"contractSha256"`
	Captures                 []fileBinding       `json:"captures"`
	Entries                  int                 `json:"entriesPerCapture"`
	Passed                   bool                `json:"passed"`
	RawEqual                 bool                `json:"rawEqual"`
	DateDifferences          []string            `json:"dateDifferences"`
	Eligibility              []eligibilityResult `json:"eligibility"`
	ValueAdmitted            bool                `json:"valueAdmitted"`
	ProspectiveAdmission     bool                `json:"prospectiveAdmission"`
	MetadataContractSHA256   string              `json:"metadataContractSha256,omitempty"`
	MetadataDifferences      []string            `json:"metadataDifferences,omitempty"`
	ProvenanceContractSHA256 string              `json:"provenanceContractSha256,omitempty"`
	ProvenanceDifferences    []string            `json:"provenanceDifferences,omitempty"`
	Origins                  []outputOrigin      `json:"origins,omitempty"`
}

type outputOrigin struct {
	Path    string      `json:"path"`
	Capture fileBinding `json:"capture"`
	Origin  fileBinding `json:"origin"`
}

//go:embed native-date-contract-v1.json
var acceptedDateContract []byte

func loadAcceptedDateContract() (dateOutputContract, error) {
	var contract dateOutputContract
	decoder := json.NewDecoder(bytes.NewReader(acceptedDateContract))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&contract); err != nil {
		return contract, err
	}
	if contract.Schema != "buildopt.eic/native-date-output-contract/v1" || contract.SourceRevision != makeProtocol().Revisions[0] || contract.AcceptedProposal != "df7b302bdde8c79cfb82f82bb0e6e48825456c78213234d7f694d84ef6f3a17b" || !reflect.DeepEqual(contract.AllowedFields, []string{"Build-Date", "Build-Date-UTC"}) || len(contract.Allowlist) != 64 {
		return contract, errors.New("accepted native date contract identity drift")
	}
	seen := map[string]bool{}
	for _, rule := range contract.Allowlist {
		if !filepath.IsLocal(rule.Path) || filepath.Clean(rule.Path) != rule.Path || strings.Contains(rule.Path, "\\") || seen[rule.Path] || rule.Producer == "" || len(rule.ManifestProvenance) == 0 {
			return contract, errors.New("invalid accepted date output rule")
		}
		seen[rule.Path] = true
		for _, source := range rule.ManifestProvenance {
			if !filepath.IsLocal(source.SourceJar) || filepath.Clean(source.SourceJar) != source.SourceJar || strings.Contains(source.SourceJar, "\\") || source.SourceProducer == "" {
				return contract, errors.New("invalid accepted manifest source")
			}
		}
	}
	return contract, nil
}

func nativeEligibility(report gradlecriticalpath.Report, p processRecord) (eligibilityResult, error) {
	r := eligibilityResult{Task: ":server:forbiddenPatterns", NativeOuterNS: p.EndNS - p.StartNS}
	if !p.Started || p.Outcome != "SUCCESS" || p.ExitCode != 0 || p.Signal != 0 || r.NativeOuterNS <= 0 || r.NativeOuterNS > 3600000000000 {
		return r, errors.New("successful bounded native process required")
	}
	count := 0
	for _, task := range report.Tasks {
		if task.Identity == r.Task {
			count++
			r.Outcome = task.Outcome
			r.DurationMs = task.DurationMs
			r.OnHardDependencyChain = task.CriticalPath
		}
	}
	if count != 1 || r.DurationMs < 0 || r.DurationMs > r.NativeOuterNS/1000000+1 {
		return r, errors.New("invalid or missing native task duration")
	}
	r.Passed = r.Outcome == "EXECUTED" && r.DurationMs >= 500 && r.DurationMs*100000000 >= r.NativeOuterNS*2
	return r, nil
}

type comparisonCapture struct {
	Binding     fileBinding
	Campaign    string
	Revision    string
	RowID       string
	Root        string
	Workspace   string
	Inventory   []entry
	Graph       nativeGraph
	Report      gradlecriticalpath.Report
	Window      nativeTimeWindow
	Eligibility eligibilityResult
}

func loadComparisonCapture(binding fileBinding) (comparisonCapture, error) {
	c, err := loadComparisonCaptureMetadata(binding)
	if err != nil {
		return c, err
	}
	selectors, err := nativeOutputSelectors(filepath.Join(filepath.Dir(binding.Path), "task-graph.jsonl"))
	if err != nil {
		return c, err
	}
	actual, err := inventory(c.Root, selectors)
	if err != nil || !reflect.DeepEqual(actual, c.Inventory) {
		return c, errors.New("retained native output inventory differs")
	}
	return c, rejectExtraRetainedOutputs(c.Root, c.Inventory)
}

// Earlier rows already passed full inventory verification. Reconstruct their
// sealed raw metadata here; callers must also rehash every artifact they use.
func loadComparisonCaptureMetadata(binding fileBinding) (comparisonCapture, error) {
	var c comparisonCapture
	if !filepath.IsAbs(binding.Path) || filepath.Base(binding.Path) != "diagnostic-complete.json" {
		return c, errors.New("absolute diagnostic completion binding required")
	}
	if err := checkBinding(binding); err != nil {
		return c, err
	}
	dir := filepath.Dir(binding.Path)
	if err := verifyNativeDiagnosticComplete(dir); err != nil {
		return c, err
	}
	var request nativeRequest
	var result nativeResult
	var process processRecord
	for name, target := range map[string]any{"native-request.json": &request, "native-result.json": &result, "process.json": &process} {
		if err := readJSON(filepath.Join(dir, name), target); err != nil {
			return c, err
		}
	}
	if request.Row.Revision != makeProtocol().Revisions[0] || request.Row.ID != result.RowID || result.InputState != "VERIFIED" || !reflect.DeepEqual(process, result.Process) {
		return c, errors.New("native source, input or process identity drift")
	}
	c.Binding, c.Campaign, c.Revision, c.RowID = binding, request.Root, request.Row.Revision, request.Row.ID
	c.Root = filepath.Join(dir, "outputs")
	c.Workspace = request.Directory
	if err := readJSONLimit(filepath.Join(dir, "output-inventory.json"), &c.Inventory, 64<<20); err != nil {
		return c, err
	}
	if len(c.Inventory) > 100000 {
		return c, errors.New("retained output inventory limit")
	}
	graphFile, err := os.Open(filepath.Join(dir, "task-graph.jsonl"))
	if err != nil {
		return c, err
	}
	defer graphFile.Close()
	scanner := bufio.NewScanner(graphFile)
	scanner.Buffer(make([]byte, 4096), 4<<20)
	for scanner.Scan() {
		var g nativeGraph
		if err = json.Unmarshal(scanner.Bytes(), &g); err != nil {
			return c, err
		}
		if g.Build == ":" {
			c.Graph = g
		}
	}
	if err = scanner.Err(); err != nil {
		return c, err
	}
	if err = readJSONLimit(filepath.Join(dir, "critical-path.json"), &c.Report, 16<<20); err != nil {
		return c, err
	}
	replay, err := gradlecriticalpath.Analyze(filepath.Join(dir, "operations-log.txt"), filepath.Join(dir, "task-graph.jsonl"), c.Report.Arm)
	if err != nil || !reflect.DeepEqual(replay, c.Report) {
		return c, errors.New("native task report differs from raw replay")
	}
	c.Eligibility, err = nativeEligibility(replay, process)
	if err != nil {
		return c, err
	}
	c.Window, err = nativeInvocationWindow(filepath.Join(dir, "operations-log.txt"))
	return c, err
}

func rejectExtraRetainedOutputs(root string, entries []entry) error {
	known := map[string]bool{".": true}
	for _, e := range entries {
		if e.Type != "absent" {
			known[e.Path] = true
			for parent := filepath.Dir(e.Path); parent != "."; parent = filepath.Dir(parent) {
				known[parent] = true
			}
		}
	}
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, e := filepath.Rel(root, path)
		if e != nil {
			return e
		}
		if !known[rel] || d.Type()&os.ModeSymlink != 0 {
			return errors.New("unexpected retained output entry")
		}
		return nil
	})
}

func nativeInvocationWindow(path string) (nativeTimeWindow, error) {
	var w nativeTimeWindow
	f, err := os.Open(path)
	if err != nil {
		return w, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4096), 4<<20)
	for scanner.Scan() {
		var record struct {
			Start *int64 `json:"startTime"`
			End   *int64 `json:"endTime"`
		}
		if err = json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return w, err
		}
		if record.Start != nil && (w.StartMs == 0 || *record.Start < w.StartMs) {
			w.StartMs = *record.Start
		}
		if record.End != nil && *record.End > w.EndMs {
			w.EndMs = *record.End
		}
	}
	if err = scanner.Err(); err != nil {
		return w, err
	}
	if w.StartMs <= 0 || w.EndMs < w.StartMs || w.EndMs-w.StartMs > 3600000 {
		return w, errors.New("invalid native wall-clock window")
	}
	return w, nil
}

func (c comparisonCapture) executedOwner(path string, expected string) error {
	owners := []string{}
	for _, t := range c.Graph.Tasks {
		for _, output := range t.Outputs {
			if path == output || strings.HasPrefix(path, output+"/") {
				owners = append(owners, t.Identity)
			}
		}
	}
	if len(owners) != 1 || owners[0] != expected {
		return errors.New("missing or ambiguous output producer")
	}
	for _, task := range c.Report.Tasks {
		if task.Identity == expected && task.Outcome == "EXECUTED" {
			return nil
		}
	}
	return errors.New("changed manifest requires an executed producer")
}

func (c comparisonCapture) verifyManifestSource(rule dateOutputRule) error {
	if len(rule.ManifestProvenance) == 0 {
		return errors.New("manifest source provenance required")
	}
	for _, source := range rule.ManifestProvenance {
		if err := c.executedOwner(source.SourceJar, source.SourceProducer); err != nil {
			return err
		}
		original, err := readJarManifest(filepath.Join(c.Root, source.SourceJar), "META-INF/MANIFEST.MF")
		if err != nil {
			return err
		}
		var actual []byte
		if source.Member == "" {
			actual, err = os.ReadFile(filepath.Join(c.Root, rule.Path))
		} else {
			actual, err = readJarManifest(filepath.Join(c.Root, rule.Path), source.Member)
		}
		if err != nil {
			return err
		}
		if !bytes.Equal(original, actual) {
			return errors.New("manifest propagation differs from declared source jar")
		}
	}
	return nil
}

func readJarManifest(path, member string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maximumDateArchive {
		return nil, errors.New("invalid manifest source archive")
	}
	z, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer z.Close()
	var found *zip.File
	for _, f := range z.File {
		if f.Name == member {
			if found != nil {
				return nil, errors.New("duplicate source manifest")
			}
			found = f
		}
	}
	if found == nil || found.UncompressedSize64 > 1<<20 {
		return nil, errors.New("missing or oversized source manifest")
	}
	f, err := found.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > 1<<20 {
		return nil, errors.New("oversized source manifest")
	}
	return raw, nil
}

// Compare only retained, externally pinned captures. Eligibility is a candidate
// screen; neither a retrospective comparison nor an eligible task admits value.
func compareNativeOutputs(first, second fileBinding, c dateOutputContract) (nativeComparison, error) {
	return compareOutputs(first, second, c, nil)
}

func compareOutputs(first, second fileBinding, c dateOutputContract, metadata *metadataSession) (nativeComparison, error) {
	return compareOutputsWithProvenance(first, second, c, metadata, nil)
}

func compareOutputsWithProvenance(first, second fileBinding, c dateOutputContract, metadata *metadataSession, provenance *provenanceSession) (nativeComparison, error) {
	rawContract, err := json.Marshal(c)
	if err != nil {
		return nativeComparison{}, err
	}
	r := nativeComparison{Schema: "buildopt.eic/native-output-comparison/v1", ContractSHA256: digest(rawContract), Captures: []fileBinding{first, second}, DateDifferences: []string{}, Eligibility: []eligibilityResult{}}
	a, err := loadComparisonCapture(first)
	if err != nil {
		return r, err
	}
	b, err := loadComparisonCapture(second)
	if err != nil {
		return r, err
	}
	if !reflect.DeepEqual(a.Graph, b.Graph) || len(a.Inventory) != len(b.Inventory) {
		return r, errors.New("native owner graph or output paths differ")
	}
	r.Entries = len(a.Inventory)
	r.Eligibility = []eligibilityResult{a.Eligibility, b.Eligibility}
	var ma, mb map[string]string
	if metadata != nil {
		ma, err = metadata.projectCapture(a)
		if err != nil {
			return r, err
		}
		mb, err = metadata.projectCapture(b)
		if err != nil {
			return r, err
		}
		r.MetadataContractSHA256 = digest(acceptedMetadataContract)
	}
	var view *provenanceView
	var reports [2]string
	var reportOrigins [2]fileBinding
	if provenance != nil {
		view = newProvenanceView(provenance)
		r.ProvenanceContractSHA256 = digest(acceptedProvenanceContract)
		for i, capture := range []comparisonCapture{a, b} {
			reports[i], reportOrigins[i], err = view.checkstyle(capture)
			if err != nil {
				return r, err
			}
		}
	}
	rules := map[string]dateOutputRule{}
	for _, rule := range c.Allowlist {
		if _, ok := rules[rule.Path]; ok {
			return r, errors.New("duplicate output date rule")
		}
		rules[rule.Path] = rule
	}
	for i, x := range a.Inventory {
		y := b.Inventory[i]
		if x == y {
			continue
		}
		if x.Path != y.Path || x.Type != "file" || y.Type != "file" || x.Mode != y.Mode {
			return r, errors.New("native output path, type or mode differs")
		}
		if view != nil && x.Path == checkstyleOutputPath {
			if reports[0] != reports[1] {
				return r, errors.New("Checkstyle differs beyond approved producer prefix")
			}
			r.ProvenanceDifferences = append(r.ProvenanceDifferences, x.Path)
			r.Origins = append(r.Origins, outputOrigin{x.Path, first, reportOrigins[0]}, outputOrigin{x.Path, second, reportOrigins[1]})
			continue
		}
		if projected, ok := ma[x.Path]; ok {
			if projected != mb[x.Path] {
				return r, fmt.Errorf("metadata semantic content differs: %s", x.Path)
			}
			r.MetadataDifferences = append(r.MetadataDifferences, x.Path)
			continue
		}
		rule, ok := rules[x.Path]
		if !ok {
			return r, fmt.Errorf("unapproved output differs: %s", x.Path)
		}
		var projections []string
		for _, capture := range []comparisonCapture{a, b} {
			if view != nil {
				projected, origin, e := view.date(capture, rule)
				if e != nil {
					return r, e
				}
				projections = append(projections, projected)
				r.Origins = append(r.Origins, outputOrigin{x.Path, capture.Binding, origin})
				continue
			}
			if err = capture.executedOwner(x.Path, rule.Producer); err != nil {
				return r, err
			}
			if err = capture.verifyManifestSource(rule); err != nil {
				return r, err
			}
			path := filepath.Join(capture.Root, x.Path)
			info, e := os.Stat(path)
			if e != nil {
				return r, e
			}
			if info.Size() > maximumDateArchive {
				return r, errors.New("date-bearing output exceeds limit")
			}
			raw, e := os.ReadFile(path)
			if e != nil {
				return r, e
			}
			var projected string
			if len(rule.AllowedMembers) > 0 {
				projected, e = archiveProjection(raw, rule.AllowedMembers, capture.Window)
			} else {
				var p []byte
				p, e = manifestProjection(raw, capture.Window)
				projected = digest(p)
			}
			if e != nil {
				return r, e
			}
			projections = append(projections, projected)
		}
		if projections[0] != projections[1] {
			return r, fmt.Errorf("native output differs beyond accepted dates: %s", x.Path)
		}
		r.DateDifferences = append(r.DateDifferences, x.Path)
	}
	r.RawEqual = len(r.DateDifferences) == 0 && len(r.MetadataDifferences) == 0 && len(r.ProvenanceDifferences) == 0
	r.Passed = true
	return r, nil
}
