package launcher

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
	"github.com/tonyredondo/buildopt/internal/ordinarylearning"
)

func TestEconomicPrequalificationRequiresEnoughAnalogousCommits(t *testing.T) {
	for _, test := range []struct {
		name     string
		commits  int
		decision string
		reason   string
	}{
		{
			name: "reject four analogous changes", commits: 4,
			decision: optimizePrequalificationReject,
			reason:   optimizePrequalificationReasonInsufficient,
		},
		{
			name: "measure after five analogous changes", commits: 5,
			decision: optimizePrequalificationMeasure,
			reason:   optimizePrequalificationReasonMeasure,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository, target := economicPrequalificationRepository(t, test.commits)
			snapshot := economicPrequalificationSnapshot()
			invocation := optimizeInvocation{
				repositoryRoot:     repository,
				calibrationPairs:   optimizeRequiredCalibrationPairs,
				maxBreakEvenBuilds: 30,
				discovery: optimizeDiscoveryContext{
					TargetRevision: target, Entrypoints: []string{"assemble"},
				},
			}
			result := prequalifyOptimizeDiscovery(
				invocation, snapshot, []string{":library"}, optimizeFamilyDependency,
			)
			if result.Decision != test.decision || result.Reason != test.reason ||
				result.AnalogousCommits != test.commits ||
				result.MinimumPaybackBuilds != ordinarylearning.MaximumPaybackMatches ||
				result.DiscoveryAuthorized != (test.decision == optimizePrequalificationMeasure) ||
				!validOptimizePrequalification(result) {
				t.Fatalf("prequalification = %+v", result)
			}
		})
	}
}

func TestEconomicPrequalificationRejectsAFullGraphCandidate(t *testing.T) {
	repository, target := economicPrequalificationRepository(t, ordinarylearning.MaximumPaybackMatches)
	result := prequalifyOptimizeDiscovery(
		optimizeInvocation{
			repositoryRoot:     repository,
			calibrationPairs:   optimizeRequiredCalibrationPairs,
			maxBreakEvenBuilds: 30,
			discovery: optimizeDiscoveryContext{
				TargetRevision: target, Entrypoints: []string{"assemble"},
			},
		},
		buildimpact.DiscoverySnapshot{
			Projects: []buildimpact.DiscoveredProject{{
				Path: ":library", SourcePaths: []string{"library/**"},
			}},
		},
		[]string{":library"}, optimizeFamilyLeaf,
	)
	if result.Decision != optimizePrequalificationReject ||
		result.Reason != optimizePrequalificationReasonNoReduction ||
		result.OmittedProjects != 0 || !validOptimizePrequalification(result) {
		t.Fatalf("full-graph prequalification = %+v", result)
	}
}

func TestCompatibleLifetimeProjectionCountsOnlyMatchingOrdinaryFamilies(t *testing.T) {
	repository, target := economicPrequalificationRepository(t, ordinarylearning.MaximumPaybackMatches)
	commits, compatible, err := countOptimizeCompatibleCommits(
		repository,
		target,
		optimizePrequalificationMaximumHistoryDepth,
		economicPrequalificationSnapshot(),
		[]string{":library"},
		optimizeFamilyDependency,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != ordinarylearning.MaximumPaybackMatches+1 ||
		compatible != ordinarylearning.MaximumPaybackMatches {
		t.Fatalf("history = %d commits/%d compatible, want %d/%d", len(commits), compatible, ordinarylearning.MaximumPaybackMatches+1, ordinarylearning.MaximumPaybackMatches)
	}
}

func economicPrequalificationRepository(t *testing.T, analogousCommits int) (string, string) {
	t.Helper()
	repository := t.TempDir()
	centralOptimizeGit(t, repository, "init", "-q")
	centralOptimizeGit(t, repository, "config", "user.email", "buildopt@example.invalid")
	centralOptimizeGit(t, repository, "config", "user.name", "BuildOpt fixture")
	centralOptimizeGit(t, repository, "config", "commit.gpgsign", "false")
	writeCentralOptimizeFile(t, repository, "settings.gradle.kts", "rootProject.name = \"economic-prequalification\"\n")
	centralOptimizeGit(t, repository, "add", ".")
	centralOptimizeGit(t, repository, "commit", "-qm", "initial repository")
	for index := 1; index <= analogousCommits; index++ {
		writeCentralOptimizeFile(
			t, repository,
			filepath.ToSlash(filepath.Join("library", "src", "main", "java", "Example.java")),
			fmt.Sprintf("class Example { int revision = %d; }\n", index),
		)
		centralOptimizeGit(t, repository, "add", "library")
		centralOptimizeGit(t, repository, "commit", "-qm", fmt.Sprintf("library change %d", index))
	}
	return repository, strings.TrimSpace(centralOptimizeGit(t, repository, "rev-parse", "HEAD"))
}

func economicPrequalificationSnapshot() buildimpact.DiscoverySnapshot {
	return buildimpact.DiscoverySnapshot{
		Complete: true,
		Projects: []buildimpact.DiscoveredProject{
			{Path: ":library", SourcePaths: []string{"library/**"}},
			{Path: ":consumer", SourcePaths: []string{"consumer/**"}, DependsOn: []string{":library"}},
			{Path: ":unrelated", SourcePaths: []string{"unrelated/**"}},
		},
	}
}
