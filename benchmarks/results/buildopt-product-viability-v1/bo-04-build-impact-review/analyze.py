"""Reconstruct BO-04's retained evidence; no subject checkout or build is executed."""

import argparse
import hashlib
import json
import math
import subprocess
from pathlib import Path
from statistics import mean


HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[3]


def load_inputs():
    bindings = json.loads((HERE / "inputs.json").read_text())
    data = {}
    for name, binding in bindings["files"].items():
        raw = (ROOT / binding["path"]).read_bytes()
        assert hashlib.sha256(raw).hexdigest() == binding["sha256"], name
        data[name] = json.loads(raw)
    for name, binding in bindings["source"].items():
        raw = subprocess.check_output(
            ["git", "show", binding["revision"] + ":" + binding["path"]],
            cwd=ROOT, timeout=30,
        )
        assert hashlib.sha256(raw).hexdigest() == binding["sha256"], name
    return data


def reconstruct(data):
    cases = []
    for expected in data["selectedSummary"]["repositories"]:
        name = "ktor" if expected["repositoryId"] == "ktorio/ktor" else "beam"
        qualification = data[name + "Qualification"]
        original = data[name + "Result"]
        observations = qualification["observations"]
        assert [r["pair"] for r in observations] == list(range(1, 9))
        assert qualification["execution"]["mechanisms"] == ["BUILD_IMPACT"]
        assert qualification["subject"]["repositoryRevision"] == expected["targetRevision"]
        assert original["discovery"]["baseRevision"] == expected["baseRevision"]
        assert original["discovery"]["candidateEntrypoints"] == qualification["plan"]["entrypoints"]
        assert original["discovery"]["requiredOutputs"] == qualification["plan"]["requiredOutputs"]
        counts = {}
        for arm in ("control", "candidate"):
            outcomes = [r[arm + "TaskOutcomes"] for r in observations]
            assert all(r == outcomes[0] for r in outcomes), (name, arm)
            counts[arm] = {k: v for k, v in outcomes[0].items() if k != "fingerprintSha256"}
            assert counts[arm]["total"] == sum(v for k, v in counts[arm].items() if k != "total")
        for row in observations:
            assert row["candidateRequiredOutputSha256"] == row["controlRequiredOutputSha256"]
            assert not row["productAttributableFailure"]
            assert row["savedMs"] == row["controlDurationMs"] - row["candidateDurationMs"]
        saving = mean(r["savedMs"] for r in observations)
        assert saving == expected["meanSavedMs"]
        assert math.ceil(expected["calibrationCostMs"] / saving) == expected["breakEvenBuilds"]
        options = qualification["execution"]["gradleOptions"]
        candidate_argv = ["{repo}/gradlew", *options, *qualification["plan"]["entrypoints"]]
        cases.append({
            "repository": expected["repositoryId"],
            "baseRevision": expected["baseRevision"],
            "targetRevision": expected["targetRevision"],
            "gradleVersion": expected["gradleVersion"],
            "originalArgv": ["{repo}/gradlew", *options, *qualification["plan"]["fallbackEntrypoints"]],
            "selectedGradleArgv": candidate_argv,
            "directNativeComparatorArgv": candidate_argv,
            "argvInterpretation": "Same requested task selection and options; not a measured equality of full invocation behavior or duration.",
            "requiredOutputsCompared": qualification["plan"]["requiredOutputs"],
            "allBroadCommandOutputsProved": False,
            "historicalPairs": 8,
            "meanSavedSecondsAgainstBroadCommand": saving / 1000,
            "recordedPreparationSeconds": expected["calibrationCostMs"] / 1000,
            "estimatedApplicableBuildsToPayBack": expected["breakEvenBuilds"],
            "selectedPlanProjects": qualification["plan"]["selectedProjectCount"],
            "declaredGraphProjects": qualification["plan"]["totalProjectCount"],
            "projectCountInterpretation": "Reach in the selected plan; not a measured count of projects whose Gradle configuration was removed.",
            "taskOutcomesIdenticalAcrossEightPairs": counts,
            "resetPolicy": "Both workspaces reset and cleaned at the same target revision; the native cache seed restored before every pair. No persistent chronological evolution in these eight pairs.",
            "matchingReplaysInOriginalResult": original["value"]["matchingReplayCount"],
            "savingAgainstDirectNativeTask": None,
            "exactSetupLifetimeSaving": None,
            "localOriginalCheckoutAndGeneratedProfileRecovered": False,
            "nativeRebuildPerformed": False,
            "decision": "NO_NEW_TRIAL_ADMITTED",
        })
    longitudinal = data["longitudinal"]["summary"]
    assert longitudinal["comparablePairs"] == longitudinal["exactOutputPairs"] == 100
    assert longitudinal["selectedProfiles"] == longitudinal["fragmentActivations"] == 0
    assert longitudinal["cumulativeSignedDeltaNs"] == (
        longitudinal["residualGradleRunnerNs"] - longitudinal["recordedBuildOptCostNs"]
    )
    jetty = data["jettyLifetime"]["economics"]
    assert jetty["grossMatchingSavedMs"] + jetty["fallbackDeltaMs"] == jetty["cumulativeGrossSavedMs"]
    assert data["functionalCoverage"]["decision"] == "STOP_GENERIC_POC"
    return {
        "schemaVersion": "buildopt.research/build-impact-admission/v1",
        "step": "BO-04",
        "status": "verified",
        "decision": "NO_NEW_BUILD_IMPACT_TRIAL_ADMITTED",
        "decisionScope": "Proposed continuations of the known Ktor and Beam selected-plan cases; not a universal impossibility result for build optimization.",
        "newNativeStarts": 0,
        "protectedValidationSourceRead": False,
        "sourceRevisionReviewed": data["selectedSummary"]["package"]["sourceRevision"],
        "cases": cases,
        "priorLimitations": {
            "jettyLifetime": jetty,
            "functionalCoverageFailedCriteria": data["functionalCoverage"]["failedCriteria"],
            "structuralRebinding": {
                "decision": data["rebinding"]["decision"],
                "implementationRevision": data["rebinding"]["implementationRevision"],
                "performanceReplayRun": data["rebinding"]["performanceReplayRun"],
            },
            "longitudinal": {k: longitudinal[k] for k in (
                "comparablePairs", "exactOutputPairs", "selectedProfiles", "fragmentActivations",
                "cumulativeSignedDeltaNs", "recordedBuildOptCostNs", "residualGradleRunnerNs",
            )},
        },
        "followUpDispositions": [
            {"proposal": "Freeze the WebJars or Twitter module manually", "decision": "NOT_ADMITTED", "reason": "The measured selector forwards that module's ordinary Gradle tasks. No additional avoidable work has been identified against the correctly scoped native command."},
            {"proposal": "Replay automatic changed-project selection for longer", "decision": "RETIRED_MECHANISM", "reason": "Longer history or a different repository alone does not address prior selection, compatibility and preparation-cost failures."},
            {"proposal": "Preserve every broad-command output while omitting projects", "decision": "NOT_ADMITTED", "reason": "The selected wins prove only their declared output subset. Prior producer/output and fragment work already addressed broader preservation without sufficient recurring value; no new cause or mechanism is established here."},
            {"proposal": "Remove unnecessary configuration for the same native task request", "decision": "NO_OBSERVED_CAUSE_FOR_ADMISSION", "reason": "Potentially a different mechanism, but the retained project counts and whole-command timings do not identify its avoidable configuration cost or safe correction. It is not an admitted successor."},
        ],
        "nextStep": "BO-05 with applicable early BO-06 readiness for the Checkstyle screen",
        "newTimingProtocol": None,
        "productViabilityEstablished": False,
    }


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    result = reconstruct(load_inputs())
    output = HERE / "assessment.json"
    if args.check:
        assert json.loads(output.read_text()) == result, "Stored assessment differs from bound evidence"
    else:
        output.write_text(json.dumps(result, indent=2) + "\n")
    print("BO-04: both protocols reconstructed from retained records; all 16 pairs preserved; no new trial admitted.")
