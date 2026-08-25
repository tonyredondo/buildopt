package adaptivefragment

import (
	"reflect"
	"strings"
	"testing"
)

func TestPriorRankingIgnoresRepositoryNamesAndInputOrder(t *testing.T) {
	query := priorQuery("holdout-a")
	observations := priorObservations()
	got, err := RankHypotheses(query, observations)
	if err != nil {
		t.Fatal(err)
	}

	renamedQuery := priorQuery("renamed-holdout")
	renamed := append([]PriorObservation{}, observations...)
	repositoryRenames := map[string]string{}
	for index := range renamed {
		original := renamed[index].RepositoryScopeSHA256
		if repositoryRenames[original] == "" {
			repositoryRenames[original] = priorSHA("renamed-source-" + string(rune('a'+len(repositoryRenames))))
		}
		renamed[index].RepositoryScopeSHA256 = repositoryRenames[original]
		renamed[index].EvidenceSHA256 = priorSHA("renamed-evidence-" + string(rune('a'+index)))
	}
	for left, right := 0, len(renamed)-1; left < right; left, right = left+1, right-1 {
		renamed[left], renamed[right] = renamed[right], renamed[left]
	}
	renamedGot, err := RankHypotheses(renamedQuery, renamed)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, renamedGot) {
		t.Fatalf("repository rename/input order changed ranking:\n%+v\n%+v", got, renamedGot)
	}
	if got.RepositoryIdentityUsedForScoring || got.ActivationAuthorized || len(got.Candidates) != 3 {
		t.Fatalf("unsafe or incomplete ranking: %+v", got)
	}
	for _, candidate := range got.Candidates {
		if !candidate.LocalCorrectnessRequired || !candidate.LocalValueRequired || candidate.ActivationAuthorized {
			t.Fatalf("candidate transferred authority: %+v", candidate)
		}
	}
}

func TestPriorEvidenceChangesPriorityOnly(t *testing.T) {
	query := priorQuery("holdout-a")
	baseline, err := RankHypotheses(query, priorObservations())
	if err != nil {
		t.Fatal(err)
	}
	changed := priorObservations()
	for index := range changed {
		switch changed[index].Hypothesis {
		case "TASK_CONTRACT_PATCH":
			changed[index].Outcome = PriorOutcomeNonPositive
			changed[index].LocalValuePassed = false
		case "SUBGRAPH_REDUCTION":
			changed[index].Outcome = PriorOutcomePositive
			changed[index].LocalValuePassed = true
		}
	}
	reordered, err := RankHypotheses(query, changed)
	if err != nil {
		t.Fatal(err)
	}
	if baseline.Candidates[0].Hypothesis == reordered.Candidates[0].Hypothesis {
		t.Fatalf("source evidence did not change exploration priority: %s", baseline.Candidates[0].Hypothesis)
	}
	if reordered.ActivationAuthorized || reordered.Candidates[0].ActivationAuthorized ||
		!reordered.Candidates[0].LocalCorrectnessRequired || !reordered.Candidates[0].LocalValueRequired {
		t.Fatalf("priority change transferred authority: %+v", reordered)
	}
}

func TestPriorReturnsNoCandidateWithoutTaskOrPluginMatch(t *testing.T) {
	query := priorQuery("holdout-a")
	query.Features.TaskImplementationSHA256 = strings.Repeat("e", 64)
	query.Features.PluginVersionSHA256 = strings.Repeat("f", 64)
	got, err := RankHypotheses(query, priorObservations())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Candidates) != 0 || got.ActivationAuthorized {
		t.Fatalf("unmatched prior returned candidates: %+v", got)
	}
}

func TestPriorRejectsUnsafeOrLocalEvidence(t *testing.T) {
	query := priorQuery("holdout-a")
	valid := priorObservations()
	cases := map[string]func([]PriorObservation) []PriorObservation{
		"same repository": func(observations []PriorObservation) []PriorObservation {
			observations[0].RepositoryScopeSHA256 = query.RepositoryScopeSHA256
			return observations
		},
		"inexact outputs": func(observations []PriorObservation) []PriorObservation {
			observations[0].ExactOutputs = false
			return observations
		},
		"product failure": func(observations []PriorObservation) []PriorObservation {
			observations[0].ProductAttributableFail = true
			return observations
		},
		"false positive value": func(observations []PriorObservation) []PriorObservation {
			observations[0].LocalValuePassed = false
			return observations
		},
		"duplicate evidence": func(observations []PriorObservation) []PriorObservation {
			observations[1].EvidenceSHA256 = observations[0].EvidenceSHA256
			return observations
		},
		"duplicate repository hypothesis": func(observations []PriorObservation) []PriorObservation {
			observations[1].RepositoryScopeSHA256 = observations[0].RepositoryScopeSHA256
			return observations
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			observations := append([]PriorObservation{}, valid...)
			if _, err := RankHypotheses(query, mutate(observations)); err == nil {
				t.Fatal("unsafe prior evidence was accepted")
			}
		})
	}
}

func priorQuery(repository string) PriorQuery {
	return PriorQuery{RepositoryScopeSHA256: priorSHA(repository), Features: priorFeatures()}
}

func priorFeatures() PriorFeatures {
	return PriorFeatures{
		TaskImplementationSHA256: strings.Repeat("1", 64), PluginVersionSHA256: strings.Repeat("2", 64),
		GradleMajor: 9, TaskClass: "ARCHIVE", GraphShape: "MULTI_PROJECT", OutputShape: "MULTI_ARTIFACT",
	}
}

func priorObservations() []PriorObservation {
	features := priorFeatures()
	return []PriorObservation{
		priorObservation("source-a", "TASK_CONTRACT_PATCH", PriorOutcomePositive, true, features),
		priorObservation("source-b", "TASK_CONTRACT_PATCH", PriorOutcomePositive, true, features),
		priorObservation("source-c", "SUBGRAPH_REDUCTION", PriorOutcomePositive, true, features),
		priorObservation("source-d", "SUBGRAPH_REDUCTION", PriorOutcomeNonPositive, false, features),
		priorObservation("source-a", "OUTPUT_MATERIALIZATION", PriorOutcomeNonPositive, false, features),
		priorObservation("source-b", "OUTPUT_MATERIALIZATION", PriorOutcomeNonPositive, false, features),
	}
}

func priorObservation(repository, hypothesis, outcome string, valuePassed bool, features PriorFeatures) PriorObservation {
	return PriorObservation{
		EvidenceSHA256: priorSHA(repository + hypothesis), RepositoryScopeSHA256: priorSHA(repository),
		Hypothesis: hypothesis, Features: features, Outcome: outcome, ExactOutputs: true,
		LocalCorrectnessPassed: true, LocalValuePassed: valuePassed,
	}
}

func priorSHA(value string) string {
	return sha("prior-" + value)
}
