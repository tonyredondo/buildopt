package runtimeoptimizer

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"sync"
	"time"
)

const (
	banditSchemaVersion             = 1
	BanditPolicyVersion             = "beta-bandit-v1"
	BanditMode                      = "BANDIT"
	BanditMinimumCandidateOutcomes  = 20
	BanditMinimumEpsilonBasisPoints = 200
	BanditMaximumEpsilonBasisPoints = 1000
	BanditMaximumOutcomeDelay       = 24 * time.Hour
	BanditTrimPercent               = 10
	BanditControlPseudoObservations = 5
)

// BanditBucket is the complete identity boundary across which samples never mix.
type BanditBucket struct {
	RepositoryID     string `json:"repositoryId"`
	MeasurementEpoch string `json:"measurementEpoch"`
	PolicyVersion    string `json:"policyVersion"`
	CatalogVersion   string `json:"catalogVersion"`
	BucketDigest     string `json:"bucketDigest"`
}

// BanditAssignmentRequest contains only pre-outcome selection inputs.
type BanditAssignmentRequest struct {
	AssignmentID       string       `json:"assignmentId"`
	ContextDigest      string       `json:"contextDigest"`
	SeedDigest         string       `json:"seedDigest"`
	Bucket             BanditBucket `json:"bucket"`
	EligibleArms       []string     `json:"eligibleArms"`
	EpsilonBasisPoints int          `json:"epsilonBasisPoints"`
	AARatioStatus      string       `json:"aaRatioStatus"`
	ResetReason        string       `json:"resetReason"`
}

// BanditAssignment is persisted before a BANDIT arm is returned.
type BanditAssignment struct {
	Request               BanditAssignmentRequest `json:"request"`
	Mode                  string                  `json:"mode"`
	ResourceProfileID     string                  `json:"resourceProfileId"`
	PropensityBasisPoints int                     `json:"propensityBasisPoints"`
	RandomBasisPoint      int                     `json:"randomBasisPoint"`
	Disposition           string                  `json:"disposition"`
	AssignedAt            time.Time               `json:"assignedAt"`
}

// RewardComponents materializes the versioned reward without floating inputs.
type RewardComponents struct {
	Complete                         bool  `json:"complete"`
	BaselineCustomerVisibleBuildMS   int64 `json:"baselineCustomerVisibleBuildMs"`
	CustomerVisibleBuildMS           int64 `json:"customerVisibleBuildMs"`
	RunnerOccupiedPenaltyMS          int64 `json:"runnerOccupiedPenaltyMs"`
	AdditionalRunnerPenaltyMS        int64 `json:"additionalRunnerPenaltyMs"`
	CIQueuePenaltyMS                 int64 `json:"ciQueuePenaltyMs"`
	CustomerVisibleFeedbackPenaltyMS int64 `json:"customerVisibleFeedbackPenaltyMs"`
	CostEquivalentPenaltyMS          int64 `json:"costEquivalentPenaltyMs"`
}

// BanditOutcome is an exactly-once outcome for a persisted bandit assignment.
type BanditOutcome struct {
	OutcomeID             string           `json:"outcomeId"`
	AssignmentID          string           `json:"assignmentId"`
	Bucket                BanditBucket     `json:"bucket"`
	PropensityBasisPoints int              `json:"propensityBasisPoints"`
	CompletedAt           time.Time        `json:"completedAt"`
	Reward                RewardComponents `json:"reward"`
	Guardrail             string           `json:"guardrail"`
}

// FixedCohortOutcome imports a B-006 assignment into its exact bandit bucket.
type FixedCohortOutcome struct {
	OutcomeID   string           `json:"outcomeId"`
	CompletedAt time.Time        `json:"completedAt"`
	Reward      RewardComponents `json:"reward"`
	Guardrail   string           `json:"guardrail"`
}

// OutcomeDisposition is the durable result of exactly-once outcome handling.
type OutcomeDisposition struct {
	OutcomeID    string       `json:"outcomeId"`
	AssignmentID string       `json:"assignmentId"`
	Bucket       BanditBucket `json:"bucket"`
	Arm          string       `json:"arm"`
	Status       string       `json:"status"`
	Reward       float64      `json:"reward"`
	RecordedAt   time.Time    `json:"recordedAt"`
}

type banditBucketState struct {
	Identity BanditBucket         `json:"identity"`
	Rewards  map[string][]float64 `json:"rewards"`
}

type banditState struct {
	SchemaVersion       int                           `json:"schemaVersion"`
	Buckets             map[string]banditBucketState  `json:"buckets"`
	Assignments         map[string]BanditAssignment   `json:"assignments"`
	Outcomes            map[string]OutcomeDisposition `json:"outcomes"`
	OutcomeByAssignment map[string]string             `json:"outcomeByAssignment"`
}

// BanditEngine owns exact-bucket learning and durable bandit assignments.
type BanditEngine struct {
	mutex sync.Mutex
	root  string
	path  string
	now   func() time.Time
	state banditState
}

// OpenBanditEngine opens a private durable beta-bandit store.
func OpenBanditEngine(root string, now func() time.Time) (*BanditEngine, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || now == nil {
		return nil, errors.New("open bandit engine: invalid configuration")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("open bandit engine: unsafe state root")
	}
	engine := &BanditEngine{root: root, path: filepath.Join(root, "bandit.json"), now: now, state: newBanditState()}
	if err := engine.load(); err != nil {
		return nil, err
	}
	return engine, nil
}

// Assign selects from the finite safe catalog and persists before returning.
func (engine *BanditEngine) Assign(request BanditAssignmentRequest) (BanditAssignment, bool, error) {
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	if err := validateBanditRequest(request); err != nil {
		return BanditAssignment{}, false, err
	}
	if current, ok := engine.state.Assignments[request.AssignmentID]; ok {
		if !reflect.DeepEqual(current.Request, request) {
			return BanditAssignment{}, false, ErrConflict
		}
		return current, false, nil
	}
	random := assignmentBasisPoint(CohortAssignmentRequest{AssignmentID: request.AssignmentID, RepositoryID: request.Bucket.RepositoryID, MeasurementEpoch: request.Bucket.MeasurementEpoch, BucketDigest: request.Bucket.BucketDigest, SeedDigest: request.SeedDigest})
	assignment := BanditAssignment{Request: request, Mode: FixedCohortMode, ResourceProfileID: "STABLE_CONTROL", PropensityBasisPoints: BasisPointTotal, RandomBasisPoint: random, Disposition: "INCONCLUSIVE", AssignedAt: engine.now().UTC()}
	if request.ResetReason != "NONE" {
		assignment.Mode, assignment.Disposition = FixedAAMode, "RESET"
	} else if request.AARatioStatus != SampleRatioValid {
		assignment.Mode = FixedAAMode
	} else {
		bucket := engine.bucket(request.Bucket)
		if candidatesReady(bucket.Rewards, request.EligibleArms) {
			assignment.Mode = BanditMode
			assignment.ResourceProfileID, assignment.PropensityBasisPoints, assignment.Disposition = selectBanditArm(bucket.Rewards, request.EligibleArms, request.EpsilonBasisPoints, random)
		} else {
			assignment.Disposition = "PENDING_SAMPLE"
		}
	}
	engine.state.Assignments[request.AssignmentID] = assignment
	if err := engine.persist(); err != nil {
		delete(engine.state.Assignments, request.AssignmentID)
		return BanditAssignment{}, false, err
	}
	return assignment, true, nil
}

// RecordOutcome applies a valid bandit outcome no more than once.
func (engine *BanditEngine) RecordOutcome(outcome BanditOutcome) (OutcomeDisposition, bool, error) {
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	assignment, ok := engine.state.Assignments[outcome.AssignmentID]
	if !ok {
		return OutcomeDisposition{}, false, os.ErrNotExist
	}
	if assignment.Mode != BanditMode || !reflect.DeepEqual(outcome.Bucket, assignment.Request.Bucket) {
		return OutcomeDisposition{}, false, errors.New("record bandit outcome: assignment binding mismatch")
	}
	return engine.recordOutcome(outcome.OutcomeID, outcome.AssignmentID, outcome.Bucket, assignment.ResourceProfileID, outcome.PropensityBasisPoints, assignment.PropensityBasisPoints, assignment.AssignedAt, outcome.CompletedAt, outcome.Reward, outcome.Guardrail)
}

// RecordFixedOutcome imports a persisted B-006 fixed-cohort observation.
func (engine *BanditEngine) RecordFixedOutcome(assignment CohortAssignment, outcome FixedCohortOutcome) (OutcomeDisposition, bool, error) {
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	if err := validateCohortRequest(assignment.Request); err != nil {
		return OutcomeDisposition{}, false, err
	}
	allocation, ok := allocationAt(assignment.Request.Policy.Allocations, assignment.RandomBasisPoint)
	if !ok || assignment.Cohort != allocation.Cohort || assignment.ResourceProfileID != allocation.ResourceProfileID || assignment.PropensityBasisPoints != allocation.PropensityBasisPoints || assignment.RandomBasisPoint != assignmentBasisPoint(assignment.Request) {
		return OutcomeDisposition{}, false, errors.New("record fixed outcome: inconsistent assignment")
	}
	bucket := BanditBucket{RepositoryID: assignment.Request.RepositoryID, MeasurementEpoch: assignment.Request.MeasurementEpoch, PolicyVersion: BanditPolicyVersion, CatalogVersion: assignment.Request.Policy.CatalogVersion, BucketDigest: assignment.Request.BucketDigest}
	return engine.recordOutcome(outcome.OutcomeID, assignment.Request.AssignmentID, bucket, assignment.ResourceProfileID, assignment.PropensityBasisPoints, assignment.PropensityBasisPoints, assignment.AssignedAt, outcome.CompletedAt, outcome.Reward, outcome.Guardrail)
}

func (engine *BanditEngine) recordOutcome(outcomeID, assignmentID string, bucket BanditBucket, arm string, actualPropensity, expectedPropensity int, assignedAt, completedAt time.Time, reward RewardComponents, guardrail string) (OutcomeDisposition, bool, error) {
	if !identifierPattern.MatchString(outcomeID) || !identifierPattern.MatchString(assignmentID) || validateBanditBucket(bucket) != nil {
		return OutcomeDisposition{}, false, errors.New("record bandit outcome: invalid identity")
	}
	if currentID, duplicate := engine.state.OutcomeByAssignment[assignmentID]; duplicate {
		return engine.state.Outcomes[currentID], false, nil
	}
	if current, duplicate := engine.state.Outcomes[outcomeID]; duplicate {
		if current.AssignmentID != assignmentID {
			return OutcomeDisposition{}, false, ErrConflict
		}
		return current, false, nil
	}
	previousBucket, bucketExisted := engine.state.Buckets[bucketKey(bucket)]
	disposition := OutcomeDisposition{OutcomeID: outcomeID, AssignmentID: assignmentID, Bucket: bucket, Arm: arm, Status: "INCONCLUSIVE", RecordedAt: engine.now().UTC()}
	validTiming := !assignedAt.IsZero() && !completedAt.Before(assignedAt) && completedAt.Sub(assignedAt) <= BanditMaximumOutcomeDelay
	if guardrail != "NONE" {
		disposition.Status = "SUSPENDED_ROLLBACK"
	} else if actualPropensity == expectedPropensity && actualPropensity > 0 && validTiming && validReward(reward) {
		disposition.Status = "UPDATED"
		disposition.Reward = calculateReward(reward)
		state := engine.bucket(bucket)
		state.Rewards[arm] = append(state.Rewards[arm], disposition.Reward)
		engine.state.Buckets[bucketKey(bucket)] = state
	}
	engine.state.Outcomes[outcomeID] = disposition
	engine.state.OutcomeByAssignment[assignmentID] = outcomeID
	if err := engine.persist(); err != nil {
		delete(engine.state.Outcomes, outcomeID)
		delete(engine.state.OutcomeByAssignment, assignmentID)
		if bucketExisted {
			engine.state.Buckets[bucketKey(bucket)] = previousBucket
		} else {
			delete(engine.state.Buckets, bucketKey(bucket))
		}
		return OutcomeDisposition{}, false, err
	}
	return disposition, disposition.Status == "UPDATED", nil
}

func selectBanditArm(rewards map[string][]float64, eligible []string, epsilon, random int) (string, int, string) {
	candidates := make([]string, 0, len(eligible)-1)
	for _, arm := range eligible {
		if arm != "STABLE_CONTROL" {
			candidates = append(candidates, arm)
		}
	}
	greedy := greedyArm(rewards, eligible)
	exploration := explorationAllocations(candidates, epsilon)
	greedyMass := BasisPointTotal - MinimumControlBasisPoints - epsilon
	if random < MinimumControlBasisPoints {
		propensity := MinimumControlBasisPoints
		if greedy == "STABLE_CONTROL" {
			propensity += greedyMass
		}
		return "STABLE_CONTROL", propensity, "ASSIGNED_CONTROL"
	}
	boundary := MinimumControlBasisPoints
	for _, arm := range candidates {
		boundary += exploration[arm]
		if random < boundary {
			propensity := exploration[arm]
			if greedy == arm {
				propensity += greedyMass
			}
			return arm, propensity, "ASSIGNED_EXPLORATION"
		}
	}
	propensity := greedyMass + exploration[greedy]
	if greedy == "STABLE_CONTROL" {
		propensity += MinimumControlBasisPoints
	}
	return greedy, propensity, "ASSIGNED"
}

func greedyArm(rewards map[string][]float64, eligible []string) string {
	controlMean := average(rewards["STABLE_CONTROL"])
	bestArm, bestPrediction := "STABLE_CONTROL", controlMean
	for _, profile := range GoldenResourceProfiles() {
		arm := profile.ProfileID
		if arm == "STABLE_CONTROL" || !slices.Contains(eligible, arm) {
			continue
		}
		values := slices.Clone(rewards[arm])
		sort.Float64s(values)
		trim := len(values) * BanditTrimPercent / 100
		if trim > 0 && trim*2 < len(values) {
			values = values[trim : len(values)-trim]
		}
		prediction := (sumFloat(values) + BanditControlPseudoObservations*controlMean) / float64(len(values)+BanditControlPseudoObservations)
		if prediction > bestPrediction {
			bestArm, bestPrediction = arm, prediction
		}
	}
	return bestArm
}

func candidatesReady(rewards map[string][]float64, eligible []string) bool {
	candidates := 0
	for _, arm := range eligible {
		if arm == "STABLE_CONTROL" {
			continue
		}
		candidates++
		if len(rewards[arm]) < BanditMinimumCandidateOutcomes {
			return false
		}
	}
	return candidates > 0
}

func explorationAllocations(candidates []string, epsilon int) map[string]int {
	result := make(map[string]int, len(candidates))
	for index, arm := range candidates {
		result[arm] = epsilon / len(candidates)
		if index < epsilon%len(candidates) {
			result[arm]++
		}
	}
	return result
}

func validateBanditRequest(request BanditAssignmentRequest) error {
	if !identifierPattern.MatchString(request.AssignmentID) || !validDigest(request.ContextDigest) || !validDigest(request.SeedDigest) || validateBanditBucket(request.Bucket) != nil || request.EpsilonBasisPoints < BanditMinimumEpsilonBasisPoints || request.EpsilonBasisPoints > BanditMaximumEpsilonBasisPoints || (request.AARatioStatus != SampleRatioValid && request.AARatioStatus != SampleRatioMismatch && request.AARatioStatus != SampleRatioInconclusive) || (request.ResetReason != "NONE" && request.ResetReason != "MEASUREMENT_EPOCH_CHANGED" && request.ResetReason != "DRIFT_THRESHOLD_EXCEEDED") {
		return errors.New("assign bandit: invalid request")
	}
	known := map[string]bool{}
	for _, profile := range GoldenResourceProfiles() {
		known[profile.ProfileID] = true
	}
	seen, control := map[string]bool{}, false
	for _, arm := range request.EligibleArms {
		if !known[arm] || seen[arm] {
			return errors.New("assign bandit: arm outside finite eligible set")
		}
		seen[arm] = true
		control = control || arm == "STABLE_CONTROL"
	}
	if !control || len(request.EligibleArms) < 2 {
		return errors.New("assign bandit: control and candidate required")
	}
	return nil
}

func validateBanditBucket(bucket BanditBucket) error {
	if !identifierPattern.MatchString(bucket.RepositoryID) || !identifierPattern.MatchString(bucket.MeasurementEpoch) || bucket.PolicyVersion != BanditPolicyVersion || bucket.CatalogVersion != GoldenResourceCatalogVersion || !validDigest(bucket.BucketDigest) {
		return errors.New("bandit bucket: invalid identity")
	}
	return nil
}

func validReward(reward RewardComponents) bool {
	return reward.Complete && reward.BaselineCustomerVisibleBuildMS >= 0 && reward.CustomerVisibleBuildMS >= 0 && reward.RunnerOccupiedPenaltyMS >= 0 && reward.AdditionalRunnerPenaltyMS >= 0 && reward.CIQueuePenaltyMS >= 0 && reward.CustomerVisibleFeedbackPenaltyMS >= 0 && reward.CostEquivalentPenaltyMS >= 0
}

func calculateReward(reward RewardComponents) float64 {
	return float64(reward.BaselineCustomerVisibleBuildMS - reward.CustomerVisibleBuildMS - reward.RunnerOccupiedPenaltyMS - reward.AdditionalRunnerPenaltyMS - reward.CIQueuePenaltyMS - reward.CustomerVisibleFeedbackPenaltyMS - reward.CostEquivalentPenaltyMS)
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	return sumFloat(values) / float64(len(values))
}

func sumFloat(values []float64) float64 {
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total
}

func bucketKey(bucket BanditBucket) string {
	return bucket.RepositoryID + "\x00" + bucket.MeasurementEpoch + "\x00" + bucket.PolicyVersion + "\x00" + bucket.CatalogVersion + "\x00" + bucket.BucketDigest
}

func newBanditState() banditState {
	return banditState{SchemaVersion: banditSchemaVersion, Buckets: map[string]banditBucketState{}, Assignments: map[string]BanditAssignment{}, Outcomes: map[string]OutcomeDisposition{}, OutcomeByAssignment: map[string]string{}}
}

func (engine *BanditEngine) bucket(identity BanditBucket) banditBucketState {
	state, ok := engine.state.Buckets[bucketKey(identity)]
	if !ok {
		state = banditBucketState{Identity: identity, Rewards: map[string][]float64{}}
	}
	return state
}

func (engine *BanditEngine) load() error {
	file, err := os.Open(engine.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || info.Size() > 16<<20 {
		return errors.New("open bandit engine: unsafe state file")
	}
	decoder := json.NewDecoder(io.LimitReader(file, 16<<20))
	decoder.DisallowUnknownFields()
	state := newBanditState()
	if err := decoder.Decode(&state); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("open bandit engine: trailing state")
	}
	if state.SchemaVersion != banditSchemaVersion || state.Buckets == nil || state.Assignments == nil || state.Outcomes == nil || state.OutcomeByAssignment == nil {
		return errors.New("open bandit engine: unsupported state")
	}
	engine.state = state
	return nil
}

func (engine *BanditEngine) persist() error {
	data, err := json.Marshal(engine.state)
	if err != nil || len(data) > 16<<20 {
		return errors.New("persist bandit engine: state exceeds bound")
	}
	file, err := os.CreateTemp(engine.root, ".bandit-*.tmp")
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
	if err := os.Rename(temporary, engine.path); err != nil {
		return err
	}
	directory, err := os.Open(engine.root)
	if err != nil {
		return err
	}
	err = directory.Sync()
	return errors.Join(err, directory.Close())
}

// BanditReplayCase is the deterministic Phase 0 replay input.
type BanditReplayCase struct {
	Trigger               string               `json:"trigger"`
	AAValid               bool                 `json:"aaValid"`
	CandidateSamplesReady bool                 `json:"candidateSamplesReady"`
	PropensityPresent     bool                 `json:"propensityPresent"`
	DuplicateOutcome      bool                 `json:"duplicateOutcome"`
	OutcomeDelayHours     int                  `json:"outcomeDelayHours"`
	EpsilonPercent        int                  `json:"epsilonPercent"`
	RandomPercent         int                  `json:"randomPercent"`
	ResetReason           string               `json:"resetReason"`
	Guardrail             string               `json:"guardrail"`
	EligibleArms          []string             `json:"eligibleArms"`
	Rewards               map[string][]float64 `json:"rewards"`
}

// BanditReplayResult is the deterministic Phase 0 replay result.
type BanditReplayResult struct {
	Mode    string
	Arm     string
	Update  bool
	Outcome string
}

// ReplayBoundedBanditCase executes Phase 0 cases through the production selector.
func ReplayBoundedBanditCase(testCase BanditReplayCase) BanditReplayResult {
	control := BanditReplayResult{Mode: FixedCohortMode, Arm: "STABLE_CONTROL", Outcome: "INCONCLUSIVE"}
	if testCase.Guardrail != "NONE" {
		control.Outcome = "SUSPENDED_ROLLBACK"
		return control
	}
	if testCase.ResetReason != "NONE" {
		control.Mode, control.Outcome = FixedAAMode, "RESET"
		return control
	}
	if !testCase.AAValid {
		control.Mode = FixedAAMode
		return control
	}
	if !testCase.PropensityPresent {
		return control
	}
	if testCase.Trigger == "OUTCOME" {
		result := BanditReplayResult{Mode: BanditMode, Arm: firstEligibleCandidate(testCase.EligibleArms), Outcome: "INCONCLUSIVE"}
		if !testCase.DuplicateOutcome && testCase.OutcomeDelayHours >= 0 && testCase.OutcomeDelayHours <= 24 {
			result.Update, result.Outcome = true, "UPDATED"
		}
		return result
	}
	if !testCase.CandidateSamplesReady {
		control.Outcome = "PENDING_SAMPLE"
		return control
	}
	if testCase.EpsilonPercent < 2 || testCase.EpsilonPercent > 10 || testCase.RandomPercent < 0 || testCase.RandomPercent >= 100 {
		return control
	}
	arm, _, disposition := selectBanditArm(testCase.Rewards, testCase.EligibleArms, testCase.EpsilonPercent*100, testCase.RandomPercent*100)
	return BanditReplayResult{Mode: BanditMode, Arm: arm, Outcome: disposition}
}

func firstEligibleCandidate(eligible []string) string {
	for _, arm := range eligible {
		if arm != "STABLE_CONTROL" {
			return arm
		}
	}
	return "STABLE_CONTROL"
}
