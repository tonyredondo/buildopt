package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type provenanceSession struct {
	Root    string
	Seed    comparisonCapture
	Origins []comparisonCapture
}

// Verification is scoped to a single comparison. Prior raw diagnostics are
// replayed once per used origin, and every referenced output is rehashed.
type provenanceView struct {
	session  *provenanceSession
	verified map[fileBinding]comparisonCapture
}

func newProvenanceView(s *provenanceSession) *provenanceView {
	return &provenanceView{session: s, verified: map[fileBinding]comparisonCapture{}}
}

func (s *provenanceSession) identity(c comparisonCapture) error {
	validID := false
	for i := 1; i <= 12; i++ {
		if c.RowID == fmt.Sprintf("C%03d", i) {
			validID = true
		}
	}
	if !validID || c.Campaign != s.Root || c.Revision != makeProtocol().Revisions[0] || c.Binding.Path != filepath.Join(s.Root, "attempts", c.RowID, "diagnostic-complete.json") || c.Root != filepath.Join(s.Root, "attempts", c.RowID, "outputs") {
		return errors.New("provenance campaign, revision or row identity drift")
	}
	return nil
}

// Called only after all row checks and sealing (or independent reconstruction)
// succeed. A failed or merely inspected row never becomes an accepted origin.
func (s *provenanceSession) rememberVerified(binding fileBinding) error {
	c, err := loadComparisonCaptureMetadata(binding)
	if err != nil {
		return err
	}
	if err = s.identity(c); err != nil {
		return err
	}
	if len(s.Origins) > 0 && s.Origins[len(s.Origins)-1].RowID >= c.RowID {
		return errors.New("nonchronological verified provenance row")
	}
	s.Origins = append(s.Origins, c)
	return nil
}

func retainedEntry(c comparisonCapture, path string) (entry, error) {
	var result entry
	count := 0
	for _, e := range c.Inventory {
		if e.Path == path {
			result = e
			count++
		}
	}
	if count != 1 || result.Type != "file" {
		return result, errors.New("exact retained file entry required")
	}
	actual, err := inventory(c.Root, []string{path})
	if err != nil || len(actual) != 1 || actual[0] != result {
		return result, fmt.Errorf("retained provenance file drift: %s", path)
	}
	return result, nil
}

func (v *provenanceView) verify(origin comparisonCapture) (comparisonCapture, error) {
	if c, ok := v.verified[origin.Binding]; ok {
		return c, nil
	}
	c, err := loadComparisonCaptureMetadata(origin.Binding)
	if err != nil {
		return c, err
	}
	if !reflect.DeepEqual(c, origin) {
		return c, errors.New("retained origin metadata drift")
	}
	v.verified[origin.Binding] = c
	return c, nil
}

func (v *provenanceView) earlier(origin, current comparisonCapture) bool {
	return v.session.identity(origin) == nil && v.session.identity(current) == nil && origin.RowID < current.RowID && origin.Window.EndMs < current.Window.StartMs
}

func verifyCurrentPropagation(c comparisonCapture, rule dateOutputRule) error {
	if len(rule.ManifestProvenance) == 0 {
		return errors.New("manifest source provenance required")
	}
	for _, source := range rule.ManifestProvenance {
		if _, err := metadataOwner(c, source.SourceJar, source.SourceProducer, ""); err != nil {
			return err
		}
		if _, err := retainedEntry(c, source.SourceJar); err != nil {
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
			return errors.New("current manifest propagation differs from declared source jar")
		}
	}
	return nil
}

func dateProjectionFrom(c comparisonCapture, rule dateOutputRule) (string, error) {
	if err := c.executedOwner(rule.Path, rule.Producer); err != nil {
		return "", err
	}
	if err := verifyCurrentPropagation(c, rule); err != nil {
		return "", err
	}
	if err := c.verifyManifestSource(rule); err != nil {
		return "", err
	}
	e, err := retainedEntry(c, rule.Path)
	if err != nil {
		return "", err
	}
	if e.Size > maximumDateArchive {
		return "", errors.New("date-bearing output exceeds limit")
	}
	raw, err := os.ReadFile(filepath.Join(c.Root, rule.Path))
	if err != nil {
		return "", err
	}
	if len(rule.AllowedMembers) > 0 {
		return archiveProjection(raw, rule.AllowedMembers, c.Window)
	}
	p, err := manifestProjection(raw, c.Window)
	return digest(p), err
}

func (v *provenanceView) date(c comparisonCapture, rule dateOutputRule) (string, fileBinding, error) {
	if err := v.session.identity(c); err != nil {
		return "", fileBinding{}, err
	}
	task, err := metadataOwner(c, rule.Path, rule.Producer, "")
	if err != nil {
		return "", fileBinding{}, err
	}
	ownerClass := ""
	for _, g := range c.Graph.Tasks {
		for _, output := range g.Outputs {
			if output == rule.Path || strings.HasPrefix(rule.Path, output+"/") {
				ownerClass = g.Class
			}
		}
	}
	if ownerClass == "" {
		return "", fileBinding{}, errors.New("missing date producer class")
	}
	current, err := retainedEntry(c, rule.Path)
	if err != nil {
		return "", fileBinding{}, err
	}
	if err = verifyCurrentPropagation(c, rule); err != nil {
		return "", fileBinding{}, err
	}
	if task.Outcome == "EXECUTED" {
		p, err := dateProjectionFrom(c, rule)
		return p, c.Binding, err
	}
	if task.Outcome != "FROM-CACHE" && task.Outcome != "UP-TO-DATE" {
		return "", fileBinding{}, errors.New("unsupported date producer outcome")
	}
	for _, origin := range v.session.Origins {
		if !v.earlier(origin, c) {
			continue
		}
		// Only exact raw identity can select an origin; a projected digest cannot.
		match := false
		for _, e := range origin.Inventory {
			if e == current {
				match = true
				break
			}
		}
		if !match {
			continue
		}
		o, err := v.verify(origin)
		if err != nil {
			return "", fileBinding{}, err
		}
		ot, err := metadataOwner(o, rule.Path, rule.Producer, ownerClass)
		if err != nil {
			return "", fileBinding{}, err
		}
		if ot.Outcome != "EXECUTED" {
			continue
		}
		if ot.TaskClass != task.TaskClass {
			return "", fileBinding{}, errors.New("date producer operation class drift")
		}
		p, err := dateProjectionFrom(o, rule)
		return p, o.Binding, err
	}
	return "", fileBinding{}, fmt.Errorf("cached date output lacks exact earlier executed origin: %s", rule.Path)
}

func (v *provenanceView) checkstyle(c comparisonCapture) (string, fileBinding, error) {
	if err := v.session.identity(c); err != nil {
		return "", fileBinding{}, err
	}
	task, err := metadataOwner(c, checkstyleOutputPath, checkstyleOwner, checkstyleClass)
	if err != nil {
		return "", fileBinding{}, err
	}
	current, err := retainedEntry(c, checkstyleOutputPath)
	if err != nil {
		return "", fileBinding{}, err
	}
	if current.Size > 16<<20 {
		return "", fileBinding{}, errors.New("Checkstyle report exceeds limit")
	}
	project := func(o comparisonCapture) (string, fileBinding, error) {
		raw, err := os.ReadFile(filepath.Join(o.Root, checkstyleOutputPath))
		if err != nil {
			return "", fileBinding{}, err
		}
		p, err := projectCheckstyle(raw, o.Workspace)
		return digest(p), o.Binding, err
	}
	if task.Outcome == "EXECUTED" {
		return project(c)
	}
	if task.Outcome != "FROM-CACHE" && task.Outcome != "UP-TO-DATE" {
		return "", fileBinding{}, errors.New("unsupported Checkstyle producer outcome")
	}
	for _, origin := range v.session.Origins {
		if !v.earlier(origin, c) {
			continue
		}
		match := false
		for _, e := range origin.Inventory {
			if e == current {
				match = true
				break
			}
		}
		if !match {
			continue
		}
		o, err := v.verify(origin)
		if err != nil {
			return "", fileBinding{}, err
		}
		ot, err := metadataOwner(o, checkstyleOutputPath, checkstyleOwner, checkstyleClass)
		if err != nil {
			return "", fileBinding{}, err
		}
		if ot.Outcome != "EXECUTED" {
			continue
		}
		if _, err = retainedEntry(o, checkstyleOutputPath); err != nil {
			return "", fileBinding{}, err
		}
		return project(o)
	}
	// The one explicitly approved historical seed is the only campaign exception.
	if current.SHA256 == seededCheckstyleSHA {
		o, err := v.verify(v.session.Seed)
		if err != nil {
			return "", fileBinding{}, err
		}
		if o.Binding.SHA256 != provenanceSeedSHA {
			return "", fileBinding{}, errors.New("Checkstyle seed capture drift")
		}
		if _, err = metadataOwner(o, checkstyleOutputPath, checkstyleOwner, checkstyleClass); err != nil {
			return "", fileBinding{}, err
		}
		e, err := retainedEntry(o, checkstyleOutputPath)
		if err != nil {
			return "", fileBinding{}, err
		}
		if e != current {
			return "", fileBinding{}, errors.New("Checkstyle seed identity drift")
		}
		return project(o)
	}
	return "", fileBinding{}, errors.New("cached Checkstyle lacks byte-identical prior capture provenance")
}
