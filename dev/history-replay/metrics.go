//go:build linux && amd64

package main

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
)

const (
	comparable            = "COMPARABLE"
	nativeRetained        = "NATIVE_RETAINED"
	nativeFailure         = "NATIVE_BUILD_FAILURE"
	dependencyUnavailable = "DEPENDENCY_UNAVAILABLE"
	nativeNondeterminism  = "NATIVE_NONDETERMINISM"
	harnessInvalid        = "HARNESS_INVALID"
	candidateFailure      = "CANDIDATE_FAILURE"
	notRunDependency      = "NOT_RUN_DEPENDENCY"
	notRunLimit           = "NOT_RUN_LIMIT"
)

type Cost struct {
	ID             string `json:"id"`
	Class          string `json:"class"`
	Purpose        string `json:"purpose"`
	Replication    int    `json:"replication"`
	Ordinal        int    `json:"ordinal"`
	Arm            string `json:"arm"`
	Attempt        string `json:"attempt"`
	StartNS        int64  `json:"startNS"`
	EndNS          int64  `json:"endNS"`
	DurationNS     int64  `json:"durationNS"`
	InsideEnvelope bool   `json:"insideEnvelope"`
	WorkflowStarts int    `json:"workflowStarts"`
	GradleStarts   int    `json:"gradleStarts"`
	NestedStarts   int    `json:"nestedStarts"`
	Bytes          int64  `json:"bytes"`
}

type Slot struct {
	OwnerMetadataJVMStarts int      `json:"ownerMetadataJVMStarts"`
	Ordinal                int      `json:"ordinal"`
	Class                  string   `json:"class"`
	Reason                 string   `json:"reason"`
	NativeNS               int64    `json:"nativeNS"`
	CandidateNS            int64    `json:"candidateNS"`
	ExtraCandidateNS       int64    `json:"extraCandidateNS"`
	NativeActions          int      `json:"nativeActions"`
	CandidateActions       int      `json:"candidateActions"`
	Attempts               []string `json:"attempts"`
}

type Interval struct {
	Block       int     `json:"block"`
	Samples     int     `json:"samples"`
	Seed        int64   `json:"seed"`
	LowerMeanNS float64 `json:"lowerMeanNS"`
	UpperMeanNS float64 `json:"upperMeanNS"`
}

type Economics struct {
	Replication                 int        `json:"replication"`
	Scheduled                   int        `json:"scheduled"`
	Comparable                  int        `json:"comparable"`
	AdoptionNS                  int64      `json:"adoptionNS"`
	MaintenanceNS               int64      `json:"maintenanceNS"`
	UnpairedCandidateNS         int64      `json:"unpairedCandidateNS"`
	NativeNS                    int64      `json:"nativeNS"`
	NetSavedNS                  int64      `json:"netSavedNS"`
	NetFraction                 float64    `json:"netFraction"`
	NetPerScheduledNS           float64    `json:"netPerScheduledNS"`
	NativeP50NS                 int64      `json:"nativeP50NS"`
	NativeP95NS                 int64      `json:"nativeP95NS"`
	CandidateP50NS              int64      `json:"candidateP50NS"`
	CandidateP95NS              int64      `json:"candidateP95NS"`
	WorstRegressionNS           int64      `json:"worstRegressionNS"`
	NoActionSlots               int        `json:"noActionSlots"`
	NoActionDeltaNS             int64      `json:"noActionDeltaNS"`
	NoActionAttributedSavingNS  int64      `json:"noActionAttributedSavingNS"`
	PaybackOrdinal              int        `json:"paybackOrdinal"`
	TemporaryCrossings          []int      `json:"temporaryCrossings"`
	CurveNS                     []int64    `json:"curveNS"`
	Intervals                   []Interval `json:"intervals"`
	CustomerHumanNS             int64      `json:"customerHumanNS"`
	ResearchNS                  int64      `json:"researchNS"`
	ResearchRecordingAndOtherNS int64      `json:"researchRecordingAndOtherNS"`
	StudyMachineNS              int64      `json:"studyMachineNS"`
	WholeStudySensitivityNetNS  int64      `json:"wholeStudySensitivityNetNS"`
	FullyLoadedNetNS            int64      `json:"fullyLoadedNetNS"`
	Complete                    bool       `json:"complete"`
	Gate                        string     `json:"gate"`
	Reasons                     []string   `json:"reasons"`
}

func nearestRank(v []int64, percent int) int64 {
	if len(v) == 0 {
		return 0
	}
	a := append([]int64{}, v...)
	sort.Slice(a, func(i, j int) bool { return a[i] < a[j] })
	return a[(percent*len(a)+99)/100-1]
}
func percentileFloat(a []float64, numerator, denominator int) float64 {
	sort.Float64s(a)
	return a[(numerator*len(a)+denominator-1)/denominator-1]
}

func bootstrap(slots []int64, adopt int64, block int) (Interval, error) {
	b := Interval{Block: block, Samples: 10000, Seed: 20260908}
	if len(slots) == 0 || len(slots)%block != 0 {
		return b, errors.New("bootstrap needs complete non-overlapping blocks")
	}
	blocks := make([]int64, len(slots)/block)
	for i, n := range slots {
		blocks[i/block] += n
	}
	r := rand.New(rand.NewSource(b.Seed))
	samples := make([]float64, b.Samples)
	for i := range samples {
		sum := -adopt
		for range blocks {
			sum += blocks[r.Intn(len(blocks))]
		}
		samples[i] = float64(sum) / float64(len(slots))
	}
	b.LowerMeanNS = percentileFloat(samples, 25, 1000)
	b.UpperMeanNS = percentileFloat(samples, 975, 1000)
	return b, nil
}

func validClass(c string) bool {
	switch c {
	case comparable, nativeRetained, nativeFailure, dependencyUnavailable, nativeNondeterminism, harnessInvalid, candidateFailure, notRunDependency, notRunLimit:
		return true
	}
	return false
}
func isComparable(s Slot) bool { return s.Class == comparable || s.Class == nativeRetained }

func calculate(m Manifest, rep int, slots []Slot, costs []Cost) (Economics, error) {
	e := Economics{Replication: rep, PaybackOrdinal: -1, TemporaryCrossings: []int{}, CurveNS: []int64{}, Intervals: []Interval{}, Reasons: []string{}, Complete: true, Gate: "NOT_QUALIFIED"}
	if len(slots) != m.ExecutionEnd+1 {
		return e, errors.New("missing scheduled ordinal")
	}
	// The bounded contract prevents integer accumulation overflow even for all
	// failed attempts; accepting an unbounded receipt would defeat raw metrics.
	maxDuration := int64(366*24*60*60) * 1e9
	maintenance := make([]int64, len(slots))
	var prefixExcess, externalAdopt int64
	ids := map[string]bool{}
	var allCosts int64
	for _, c := range costs {
		if ids[c.ID] || c.ID == "" || c.DurationNS != c.EndNS-c.StartNS || c.StartNS < 0 || c.DurationNS < 0 || c.DurationNS > maxDuration {
			return e, errors.New("invalid or duplicate cost phase")
		}
		if c.DurationNS > maxDuration-allCosts {
			return e, errors.New("cost accumulation exceeds supported one-year bound")
		}
		allCosts += c.DurationNS
		ids[c.ID] = true
		if c.Replication != rep {
			continue
		}
		if c.Ordinal < 0 || c.Ordinal >= len(slots) {
			return e, errors.New("cost outside scheduled horizon")
		}
		switch c.Class {
		case "customer-human":
			e.CustomerHumanNS += c.DurationNS
		case "research":
			if !c.InsideEnvelope {
				e.ResearchNS += c.DurationNS
			}
		case "customer-machine":
			if !c.InsideEnvelope {
				if c.Ordinal <= m.PrefixEnd {
					externalAdopt += c.DurationNS
				} else {
					maintenance[c.Ordinal] += c.DurationNS
				}
			}
		default:
			return e, errors.New("unknown cost classification")
		}
	}
	adjusted := []int64{}
	nValues := []int64{}
	iValues := []int64{}
	for i, s := range slots {
		if s.Ordinal != i || !validClass(s.Class) {
			return e, errors.New("duplicate/reordered/invalid slot")
		}
		if s.NativeNS < 0 || s.CandidateNS < 0 || s.ExtraCandidateNS < 0 || s.NativeNS > maxDuration || s.CandidateNS > maxDuration || s.ExtraCandidateNS > maxDuration {
			return e, errors.New("invalid request duration")
		}
		if s.Class == notRunLimit || s.Class == notRunDependency || s.Class == harnessInvalid || s.Class == candidateFailure || s.Class == nativeNondeterminism {
			e.Complete = false
		}
		if isComparable(s) && (s.NativeNS == 0 || s.CandidateNS == 0) {
			return e, errors.New("comparable slot lacks request")
		}
		if i <= m.PrefixEnd {
			if isComparable(s) {
				prefixExcess += s.CandidateNS - s.NativeNS
			} else {
				externalAdopt += s.CandidateNS
			}
			externalAdopt += s.ExtraCandidateNS
			continue
		}
		e.Scheduled++
		e.MaintenanceNS += maintenance[i]
		charge := s.ExtraCandidateNS
		delta := int64(0)
		if isComparable(s) {
			e.Comparable++
			e.NativeNS += s.NativeNS
			delta = s.NativeNS - s.CandidateNS
			nValues = append(nValues, s.NativeNS)
			iValues = append(iValues, s.CandidateNS)
			e.WorstRegressionNS = max(e.WorstRegressionNS, -delta)
			if s.NativeActions == 0 && s.CandidateActions == 0 {
				e.NoActionSlots++
				e.NoActionDeltaNS += delta
			}
		} else {
			charge += s.CandidateNS
		}
		e.UnpairedCandidateNS += charge
		adjusted = append(adjusted, delta-charge-maintenance[i])
	}
	e.AdoptionNS = externalAdopt + max(int64(0), prefixExcess)
	running := -e.AdoptionNS
	for _, n := range adjusted {
		running += n
		e.CurveNS = append(e.CurveNS, running)
	}
	e.NetSavedNS = running
	e.FullyLoadedNetNS = running - e.ResearchNS
	if e.Scheduled > 0 {
		e.NetPerScheduledNS = float64(running) / float64(e.Scheduled)
	}
	if e.NativeNS > 0 {
		e.NetFraction = float64(running) / float64(e.NativeNS)
	}
	e.NativeP50NS = nearestRank(nValues, 50)
	e.NativeP95NS = nearestRank(nValues, 95)
	e.CandidateP50NS = nearestRank(iValues, 50)
	e.CandidateP95NS = nearestRank(iValues, 95)
	// A crossing counts only if the entire remaining *scheduled* suffix repays.
	suffixNonnegative := true
	for i := len(e.CurveNS) - 1; i >= 0; i-- {
		suffixNonnegative = suffixNonnegative && e.CurveNS[i] >= 0
		if suffixNonnegative {
			e.PaybackOrdinal = m.PrefixEnd + 1 + i
		}
	}
	for i, v := range e.CurveNS {
		if v >= 0 && (i == 0 || e.CurveNS[i-1] < 0) {
			ordinal := m.PrefixEnd + 1 + i
			if e.PaybackOrdinal < 0 || ordinal < e.PaybackOrdinal {
				e.TemporaryCrossings = append(e.TemporaryCrossings, ordinal)
			}
		}
	}
	if !e.Complete {
		e.PaybackOrdinal = -1
	}
	for _, block := range []int{5, 10} {
		if len(adjusted) > 0 && len(adjusted)%block == 0 {
			b, err := bootstrap(adjusted, e.AdoptionNS, block)
			if err != nil {
				return e, err
			}
			e.Intervals = append(e.Intervals, b)
		}
	}
	if m.Phase != "CONFIRMATION" {
		e.Reasons = append(e.Reasons, "qualification/engineering is not confirmation")
		return e, nil
	}
	if !e.Complete || e.Scheduled != 80 {
		e.Reasons = append(e.Reasons, "incomplete or unsafe horizon")
	}
	if e.Comparable < 76 {
		e.Reasons = append(e.Reasons, "coverage below 76/80")
	}
	if e.NetFraction < 0.05 {
		e.Reasons = append(e.Reasons, "net reduction below 5 percent")
	}
	if e.NetPerScheduledNS < 1e9 {
		e.Reasons = append(e.Reasons, "net saving below one second per scheduled transition")
	}
	if e.PaybackOrdinal < 21 {
		e.Reasons = append(e.Reasons, "no durable observed payback")
	}
	if float64(e.CandidateP95NS) > float64(e.NativeP95NS)+math.Max(1e9, 0.05*float64(e.NativeP95NS)) {
		e.Reasons = append(e.Reasons, "candidate p95 regression")
	}
	if len(e.Intervals) != 2 {
		return e, errors.New("confirmation bootstrap missing")
	}
	if e.Intervals[0].LowerMeanNS <= 0 {
		e.Reasons = append(e.Reasons, "nonpositive lower uncertainty bound")
	}
	if (e.Intervals[0].LowerMeanNS > 0) != (e.Intervals[1].LowerMeanNS > 0) {
		e.Gate = "INCONCLUSIVE"
		e.Reasons = append(e.Reasons, "block sensitivity changes lower-bound sign")
		return e, nil
	}
	e.Gate = "G3_NEGATIVE"
	if len(e.Reasons) == 0 {
		e.Gate = "G3_PASS"
	}
	return e, nil
}

func armOrder(rep, ordinal int) []string {
	if (rep+ordinal)%2 == 1 {
		return []string{"N", "I"}
	}
	return []string{"I", "N"}
}
func slotKey(rep, ordinal int) string { return fmt.Sprintf("r%d-%03d", rep, ordinal) }
