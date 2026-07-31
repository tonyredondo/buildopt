package runtimeoptimizer

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"sync"
	"time"
)

const (
	fixedCohortSchemaVersion  = 1
	BasisPointTotal           = 10_000
	MinimumControlBasisPoints = 500
	FixedAAMode               = "FIXED_AA"
	FixedCohortMode           = "FIXED_COHORT"
	SampleRatioValid          = "VALID"
	SampleRatioMismatch       = "SAMPLE_RATIO_MISMATCH"
	SampleRatioInconclusive   = "INCONCLUSIVE"
)

// CohortAllocation is one declared pre-outcome allocation probability.
type CohortAllocation struct {
	Cohort                string `json:"cohort"`
	ResourceProfileID     string `json:"resourceProfileId"`
	PropensityBasisPoints int    `json:"propensityBasisPoints"`
}

// FixedCohortPolicy binds assignments to one immutable policy era and bucket.
type FixedCohortPolicy struct {
	PolicyVersion      string             `json:"policyVersion"`
	CatalogVersion     string             `json:"catalogVersion"`
	Mode               string             `json:"mode"`
	MinimumAssignments int                `json:"minimumAssignments"`
	MaximumChiSquare   float64            `json:"maximumChiSquare"`
	Allocations        []CohortAllocation `json:"allocations"`
}

// CohortAssignmentRequest contains only information available before execution.
type CohortAssignmentRequest struct {
	AssignmentID     string            `json:"assignmentId"`
	RepositoryID     string            `json:"repositoryId"`
	MeasurementEpoch string            `json:"measurementEpoch"`
	BucketDigest     string            `json:"bucketDigest"`
	ContextDigest    string            `json:"contextDigest"`
	SeedDigest       string            `json:"seedDigest"`
	Policy           FixedCohortPolicy `json:"policy"`
}

// CohortAssignment is the durable record that must exist before execution.
type CohortAssignment struct {
	Request               CohortAssignmentRequest `json:"request"`
	Cohort                string                  `json:"cohort"`
	ResourceProfileID     string                  `json:"resourceProfileId"`
	PropensityBasisPoints int                     `json:"propensityBasisPoints"`
	RandomBasisPoint      int                     `json:"randomBasisPoint"`
	AssignedAt            time.Time               `json:"assignedAt"`
}

// SampleRatioReport describes goodness of fit against declared propensities.
type SampleRatioReport struct {
	Status           string         `json:"status"`
	AssignmentCount  int            `json:"assignmentCount"`
	ChiSquare        float64        `json:"chiSquare"`
	ObservedByCohort map[string]int `json:"observedByCohort"`
}

type fixedCohortState struct {
	SchemaVersion int                         `json:"schemaVersion"`
	Assignments   map[string]CohortAssignment `json:"assignments"`
}

// CohortLedger durably records an assignment before returning it to a caller.
type CohortLedger struct {
	mutex sync.Mutex
	root  string
	path  string
	now   func() time.Time
	state fixedCohortState
}

// OpenCohortLedger opens a private, bounded assignment store.
func OpenCohortLedger(root string, now func() time.Time) (*CohortLedger, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || now == nil {
		return nil, errors.New("open cohort ledger: invalid configuration")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("open cohort ledger: unsafe state root")
	}
	ledger := &CohortLedger{
		root:  root,
		path:  filepath.Join(root, "cohort-assignments.json"),
		now:   now,
		state: fixedCohortState{SchemaVersion: fixedCohortSchemaVersion, Assignments: map[string]CohortAssignment{}},
	}
	if err := ledger.load(); err != nil {
		return nil, err
	}
	return ledger, nil
}

// Assign deterministically chooses and persists exactly one declared allocation.
func (ledger *CohortLedger) Assign(request CohortAssignmentRequest) (CohortAssignment, bool, error) {
	ledger.mutex.Lock()
	defer ledger.mutex.Unlock()
	if err := validateCohortRequest(request); err != nil {
		return CohortAssignment{}, false, err
	}
	if current, ok := ledger.state.Assignments[request.AssignmentID]; ok {
		if !reflect.DeepEqual(current.Request, request) {
			return CohortAssignment{}, false, ErrConflict
		}
		return current, false, nil
	}
	random := assignmentBasisPoint(request)
	allocation, ok := allocationAt(request.Policy.Allocations, random)
	if !ok {
		return CohortAssignment{}, false, errors.New("assign cohort: incomplete allocation")
	}
	assignment := CohortAssignment{
		Request: request, Cohort: allocation.Cohort, ResourceProfileID: allocation.ResourceProfileID,
		PropensityBasisPoints: allocation.PropensityBasisPoints, RandomBasisPoint: random, AssignedAt: ledger.now().UTC(),
	}
	ledger.state.Assignments[request.AssignmentID] = assignment
	if err := ledger.persist(); err != nil {
		delete(ledger.state.Assignments, request.AssignmentID)
		return CohortAssignment{}, false, err
	}
	return assignment, true, nil
}

// AnalyzeSampleRatio compares observations with the exact declared allocation.
func AnalyzeSampleRatio(assignments []CohortAssignment, policy FixedCohortPolicy) SampleRatioReport {
	report := SampleRatioReport{Status: SampleRatioInconclusive, AssignmentCount: len(assignments), ObservedByCohort: map[string]int{}}
	if err := validateCohortPolicy(policy); err != nil || len(assignments) < policy.MinimumAssignments {
		return report
	}
	allocationByCohort := make(map[string]CohortAllocation, len(policy.Allocations))
	for _, allocation := range policy.Allocations {
		allocationByCohort[allocation.Cohort] = allocation
	}
	seen := make(map[string]struct{}, len(assignments))
	for _, assignment := range assignments {
		allocation, ok := allocationByCohort[assignment.Cohort]
		if !ok || !reflect.DeepEqual(assignment.Request.Policy, policy) || assignment.ResourceProfileID != allocation.ResourceProfileID || assignment.PropensityBasisPoints != allocation.PropensityBasisPoints || assignment.PropensityBasisPoints <= 0 {
			return report
		}
		if _, duplicate := seen[assignment.Request.AssignmentID]; duplicate {
			return report
		}
		seen[assignment.Request.AssignmentID] = struct{}{}
		report.ObservedByCohort[assignment.Cohort]++
	}
	for _, allocation := range policy.Allocations {
		expected := float64(len(assignments)*allocation.PropensityBasisPoints) / BasisPointTotal
		if expected < 5 {
			return report
		}
		difference := float64(report.ObservedByCohort[allocation.Cohort]) - expected
		report.ChiSquare += difference * difference / expected
	}
	report.ChiSquare = math.Round(report.ChiSquare*1_000_000) / 1_000_000
	if report.ChiSquare > policy.MaximumChiSquare {
		report.Status = SampleRatioMismatch
	} else {
		report.Status = SampleRatioValid
	}
	return report
}

func validateCohortRequest(request CohortAssignmentRequest) error {
	if !identifierPattern.MatchString(request.AssignmentID) || !identifierPattern.MatchString(request.RepositoryID) || !identifierPattern.MatchString(request.MeasurementEpoch) || !validDigest(request.BucketDigest) || !validDigest(request.ContextDigest) || !validDigest(request.SeedDigest) {
		return errors.New("assign cohort: invalid pre-outcome identity")
	}
	return validateCohortPolicy(request.Policy)
}

func validateCohortPolicy(policy FixedCohortPolicy) error {
	if !identifierPattern.MatchString(policy.PolicyVersion) || policy.CatalogVersion != GoldenResourceCatalogVersion || policy.MinimumAssignments < 20 || policy.MinimumAssignments > 1_000_000 || math.IsNaN(policy.MaximumChiSquare) || math.IsInf(policy.MaximumChiSquare, 0) || policy.MaximumChiSquare <= 0 || len(policy.Allocations) < 2 || len(policy.Allocations) > len(GoldenResourceProfiles()) {
		return errors.New("cohort policy: invalid bounds")
	}
	known := make(map[string]struct{}, len(GoldenResourceProfiles()))
	for _, profile := range GoldenResourceProfiles() {
		known[profile.ProfileID] = struct{}{}
	}
	total, control := 0, 0
	cohorts := map[string]struct{}{}
	profiles := map[string]struct{}{}
	for _, allocation := range policy.Allocations {
		if !identifierPattern.MatchString(allocation.Cohort) || allocation.PropensityBasisPoints <= 0 || allocation.PropensityBasisPoints > BasisPointTotal {
			return errors.New("cohort policy: invalid allocation")
		}
		if _, ok := known[allocation.ResourceProfileID]; !ok {
			return errors.New("cohort policy: arm outside finite catalog")
		}
		if _, duplicate := cohorts[allocation.Cohort]; duplicate {
			return errors.New("cohort policy: duplicate cohort")
		}
		cohorts[allocation.Cohort] = struct{}{}
		total += allocation.PropensityBasisPoints
		if allocation.ResourceProfileID == "STABLE_CONTROL" {
			control += allocation.PropensityBasisPoints
		}
		if policy.Mode == FixedCohortMode {
			if _, duplicate := profiles[allocation.ResourceProfileID]; duplicate {
				return errors.New("cohort policy: duplicate resource profile")
			}
			profiles[allocation.ResourceProfileID] = struct{}{}
		}
	}
	if total != BasisPointTotal || control < MinimumControlBasisPoints {
		return errors.New("cohort policy: allocation or stable control floor is invalid")
	}
	if policy.Mode == FixedAAMode {
		if len(policy.Allocations) != 2 || policy.Allocations[0].Cohort != "A" || policy.Allocations[1].Cohort != "B" || policy.Allocations[0].ResourceProfileID != "STABLE_CONTROL" || policy.Allocations[1].ResourceProfileID != "STABLE_CONTROL" {
			return errors.New("cohort policy: A/A must compare identical stable-control arms")
		}
	} else if policy.Mode != FixedCohortMode {
		return errors.New("cohort policy: unsupported mode")
	}
	return nil
}

func assignmentBasisPoint(request CohortAssignmentRequest) int {
	digest := sha256.Sum256([]byte(request.SeedDigest + "\x00" + request.AssignmentID + "\x00" + request.RepositoryID + "\x00" + request.MeasurementEpoch + "\x00" + request.BucketDigest))
	return int(binary.BigEndian.Uint64(digest[:8]) % BasisPointTotal)
}

func allocationAt(allocations []CohortAllocation, point int) (CohortAllocation, bool) {
	boundary := 0
	for _, allocation := range allocations {
		boundary += allocation.PropensityBasisPoints
		if point < boundary {
			return allocation, true
		}
	}
	return CohortAllocation{}, false
}

func (ledger *CohortLedger) load() error {
	file, err := os.Open(ledger.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || info.Size() > 8<<20 {
		return errors.New("open cohort ledger: unsafe state file")
	}
	decoder := json.NewDecoder(io.LimitReader(file, 8<<20))
	decoder.DisallowUnknownFields()
	var state fixedCohortState
	if err := decoder.Decode(&state); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("open cohort ledger: trailing state")
	}
	if state.SchemaVersion != fixedCohortSchemaVersion || state.Assignments == nil {
		return errors.New("open cohort ledger: unsupported state")
	}
	for id, assignment := range state.Assignments {
		if id != assignment.Request.AssignmentID || validateCohortRequest(assignment.Request) != nil {
			return errors.New("open cohort ledger: invalid assignment")
		}
		expected, ok := allocationAt(assignment.Request.Policy.Allocations, assignment.RandomBasisPoint)
		if !ok || assignment.Cohort != expected.Cohort || assignment.ResourceProfileID != expected.ResourceProfileID || assignment.PropensityBasisPoints != expected.PropensityBasisPoints || assignment.RandomBasisPoint != assignmentBasisPoint(assignment.Request) || assignment.AssignedAt.IsZero() {
			return errors.New("open cohort ledger: inconsistent assignment")
		}
	}
	ledger.state = state
	return nil
}

func (ledger *CohortLedger) persist() error {
	keys := make([]string, 0, len(ledger.state.Assignments))
	for key := range ledger.state.Assignments {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	ordered := fixedCohortState{SchemaVersion: fixedCohortSchemaVersion, Assignments: make(map[string]CohortAssignment, len(keys))}
	for _, key := range keys {
		ordered.Assignments[key] = ledger.state.Assignments[key]
	}
	data, err := json.Marshal(ordered)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(ledger.root, ".cohort-*.tmp")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err = file.Chmod(0o600); err != nil {
		_ = file.Close()
		return err
	}
	if _, err = file.Write(append(data, '\n')); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err := os.Rename(temporary, ledger.path); err != nil {
		return err
	}
	directory, err := os.Open(ledger.root)
	if err != nil {
		return err
	}
	err = directory.Sync()
	return errors.Join(err, directory.Close())
}
