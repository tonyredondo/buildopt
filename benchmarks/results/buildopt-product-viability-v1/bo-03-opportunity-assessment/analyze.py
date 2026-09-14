"""Recompute BO-03 from retained observations; never launch a build or read holdout source."""

import argparse
import hashlib
import json
import math
from pathlib import Path
from statistics import mean


HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[3]


def read_inputs():
    bindings = json.loads((HERE / "inputs.json").read_text())
    inputs = {}
    for name, binding in bindings.items():
        data = (ROOT / binding["path"]).read_bytes()
        assert hashlib.sha256(data).hexdigest() == binding["sha256"], name
        if binding["path"].endswith(".json"):
            inputs[name] = json.loads(data)
    return inputs


def paired_summary(rows, control_key, candidate_key):
    assert len(rows) == 8
    assert [row["pair"] for row in rows] == list(range(1, 9))
    controls = [row[control_key] / 1000 for row in rows]
    candidates = [row[candidate_key] / 1000 for row in rows]
    savings = [n - i for n, i in zip(controls, candidates)]
    return {
        "pairs": 8,
        "controlMeanSeconds": mean(controls),
        "candidateMeanSeconds": mean(candidates),
        "savingMeanSeconds": mean(savings),
        "positivePairs": sum(s > 0 for s in savings),
        "savingPercent": 100 * mean(savings) / mean(controls),
        "individualSavingsSeconds": savings,
    }


def sensitivity(native_seconds, saving_seconds, preparation_seconds):
    """An illustration over 80 slots, not a forecast of missing history."""
    assert native_seconds > 0 and saving_seconds > 0 and preparation_seconds >= 0
    count = 80
    required_net = max(count, 0.05 * count * native_seconds)
    required_matches = math.ceil((preparation_seconds + required_net) / saving_seconds)
    return {
        "scheduledValidationSlots": count,
        "assumption": "Every native slot has the historical selected-command mean; each useful slot saves its historical mean; all other deltas and maintenance are zero. No development-prefix saving earns credit.",
        "modeledPreparationSeconds": preparation_seconds,
        "requiredNetSeconds": required_net,
        "applicableSlotsRequired": required_matches,
        "fitsEightySlots": required_matches <= count,
        "netSecondsIfAllEightyApplicable": count * saving_seconds - preparation_seconds,
        "netSecondsPerSlotIfAllApplicable": saving_seconds - preparation_seconds / count,
        "maximumPreparationToMeetBothFloorsIfAllApplicable": count * saving_seconds - required_net,
        "observedPayback": False,
        "wholeWorkflowAdmission": False,
    }


def assess(inputs):
    residual = inputs["checkstyleOpportunity"]
    assert [r["ordinal"] for r in residual["rows"]] == list(range(21))
    assert residual["validationOrdinalsConsumed"] == []
    rows = residual["rows"][1:]
    command = sum(r["commandSeconds"] for r in rows)
    workflow_ms = sum(r["workflowMilliseconds"] for r in rows)
    saving_ms = sum(r["dependencyModelSavingMilliseconds"] for r in rows)
    for row in rows:
        assert row["dependencyModelSavingMilliseconds"] == (
            row["nativeDependencyPathMilliseconds"] - row["zeroActionDependencyPathMilliseconds"]
        )
    assert workflow_ms == residual["totals"]["workflowMilliseconds"]
    assert saving_ms == residual["totals"]["dependencyModelSavingMilliseconds"]
    assert math.isclose(command, inputs["nativeOpportunity"]["totals"]["commandSeconds"])
    saving = saving_ms / 1000
    required = max(len(rows), 0.05 * command)
    active = [r["ordinal"] for r in rows if any(t["actionMilliseconds"] for t in r["selected"].values())]
    checkstyle = {
        "id": "elasticsearch-checkstyle-supported-v2",
        "disposition": "ELIGIBLE_FOR_REGISTERED_SHORT_SCREEN_AFTER_READINESS",
        "reason": "Native whole-command opportunity narrowly clears the floors in the existing fixed-duration model; V2 correctness and A/A do not establish native-versus-V2 value.",
        "wholeWorkflowAdmission": False,
        "developmentTransitions": len(rows),
        "nativeActionOrdinals": active,
        "nativeActionFrequency": len(active) / len(rows),
        "commandSeconds": command,
        "gradleRunBuildSpanSeconds": workflow_ms / 1000,
        "fixedDependencyModelSavingSeconds": saving,
        "modelSecondsPerTransition": saving / len(rows),
        "modelWholeCommandPercent": 100 * saving / command,
        "modelRequiredNetSeconds": required,
        "modelMarginForResidualWorkAndCostsSeconds": saving - required,
        "modelMarginSecondsPerTransition": (saving - required) / len(rows),
        "modelLimitation": "Zero all Checkstyle actions, retain other task durations and overhead. Not an absolute ceiling: contention and scheduling can change other durations. Not a measured candidate result.",
        "actionUnionSeconds": sum(r["actionUnionMilliseconds"] for r in rows) / 1000,
        "overlappingActionSumSeconds": sum(t["actionMilliseconds"] for r in rows for t in r["selected"].values()) / 1000,
        "nonzeroModelRows": [{"ordinal": r["ordinal"], "savingSeconds": r["dependencyModelSavingMilliseconds"] / 1000} for r in rows if r["dependencyModelSavingMilliseconds"]],
        "adoptionSeconds": None,
        "maintenanceSeconds": None,
        "earlierImplementationDisjointHistory": {k: inputs["oldScreen"]["profiles"]["history-8"][k] for k in ("measuredOriginalOrdinals", "meanNetSavingSeconds", "aggregateNetSavingPercent")},
        "identicalV2Control": {k: inputs["aa"][k] for k in ("decision", "absoluteDifferenceMS", "differencePercentOfFaster")},
    }
    results = [checkstyle]
    inventory = {c["id"]: c for c in inputs["inventory"]["candidates"]}
    prefix = {c["id"]: c for c in inputs["sourceReview"]["repositories"]}
    for proposal in inputs["portfolio"]["proposals"]:
        identity = proposal["id"]
        if identity not in prefix:
            continue
        source = prefix[identity]
        assert [r["ordinal"] for r in source["rows"]] == list(range(21))
        assert all(r["taskSourceSHA256"] == inventory[identity]["patch"]["expectedSourceSha256"] for r in source["rows"])
        summary = paired_summary(proposal["observations"], "controlMs", "candidateMs")
        assert math.isclose(summary["savingMeanSeconds"] * 1000, proposal["summary"]["meanSavedMs"])
        assert math.isclose(summary["controlMeanSeconds"] * 1000, proposal["summary"]["controlMeanMs"])
        cost = proposal["economics"]["chargedMachineCostMs"] / 1000
        assert cost == sum(c["chargedMs"] for c in proposal["economics"]["costComponents"]) / 1000
        assert math.ceil(cost / summary["savingMeanSeconds"]) == proposal["economics"]["machinePaybackBuilds"]
        is_spring = identity.startswith("spring")
        results.append({
            "id": identity,
            "disposition": "DO_NOT_TIME_SELECTED_CASE_ON_CURRENT_EVIDENCE" if is_spring else "DEFER_TIMING_UNTIL_RECURRING_CACHE_RESTORE_WORKFLOW_IS_ESTABLISHED",
            "reason": "Recorded mean saving misses the current one-second floor even at every build before costs; no defensible full-workflow ceiling has been measured." if is_spring else "Selected savings could matter, but a useful cache restore is not implied by a code change. The whole command, frequency and costs remain unmeasured.",
            "historicalPairs": summary,
            "historicalPreparationChargeSeconds": cost,
            "historicalPreparationPolicy": inputs["portfolio"]["costPolicy"],
            "historicalEstimatedPaybackApplicableBuilds": proposal["economics"]["machinePaybackBuilds"],
            "wholeWorkflowCeilingSeconds": None,
            "taskExecutionFrequency": None,
            "usefulCacheRestoreFrequency": None,
            "adoptionSeconds": None,
            "maintenanceSeconds": None,
            "matchingDevelopmentTaskPreimages": source["matchingPreimages"],
            "sourceTouchesNotTaskExecutions": source["touchedOrdinals"],
            "zeroCostSensitivity": sensitivity(summary["controlMeanSeconds"], summary["savingMeanSeconds"], 0),
            "historicalChargeSensitivity": sensitivity(summary["controlMeanSeconds"], summary["savingMeanSeconds"], cost),
            "wholeWorkflowAdmission": False,
        })
    for historical in inputs["impact"]["repositories"]:
        name = "ktor" if historical["repositoryId"] == "ktorio/ktor" else "beam"
        qualification = inputs[name + "Qualification"]
        summary = paired_summary(qualification["observations"], "controlDurationMs", "candidateDurationMs")
        assert math.isclose(summary["savingMeanSeconds"] * 1000, historical["meanSavedMs"])
        for row in qualification["observations"]:
            assert row["controlRequiredOutputSha256"] == row["candidateRequiredOutputSha256"]
            assert not row["productAttributableFailure"]
        cost = historical["calibrationCostMs"] / 1000
        assert math.ceil(cost / summary["savingMeanSeconds"]) == historical["breakEvenBuilds"]
        results.append({
            "id": name + "-selected-build-impact",
            "disposition": "BO04_ADMISSION_REVIEW_ONLY",
            "reason": "Large selected savings, but required-output scope, advantage over a direct module command and lifetime must be resolved before another trial.",
            "historicalPairs": summary,
            "originalEntrypoints": qualification["plan"]["fallbackEntrypoints"],
            "candidateEntrypoints": qualification["plan"]["entrypoints"],
            "comparedRequiredOutputs": qualification["plan"]["requiredOutputs"],
            "fullOriginalCommandOutputEquivalenceEstablished": False,
            "historicalPreparationChargeSeconds": cost,
            "historicalEstimatedPaybackApplicableBuilds": historical["breakEvenBuilds"],
            "wholeWorkflowCeilingSeconds": None,
            "usefulPlanFrequency": None,
            "adoptionSeconds": None,
            "maintenanceSeconds": None,
            "zeroCostSensitivity": sensitivity(summary["controlMeanSeconds"], summary["savingMeanSeconds"], 0),
            "historicalChargeSensitivity": sensitivity(summary["controlMeanSeconds"], summary["savingMeanSeconds"], cost),
            "wholeWorkflowAdmission": False,
        })
    assert {r["id"] for r in results} == set(inventory)
    return {
        "schemaVersion": "buildopt.research/opportunity-assessment/v1",
        "step": "BO-03",
        "status": "verified",
        "meaningOfVerified": "Assessment and disposition complete; unknown workflow evidence is explicit, not qualified or treated as zero.",
        "newNativeBuilds": 0,
        "validationSourceRead": False,
        "productViabilityEstablished": False,
        "longReplaysAdmitted": [],
        "nextStep": "BO-04",
        "costRule": "80 validation slots; no development savings credited. Historical campaign charges are a sensitivity, not measured future adoption cost. Unknown required work remains unknown and cannot be waived.",
        "candidates": results,
    }


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="Check stored calculations without writing")
    args = parser.parse_args()
    result = assess(read_inputs())
    path = HERE / "assessment.json"
    if args.check:
        assert json.loads(path.read_text()) == result, "Stored assessment differs from its bound inputs"
    else:
        path.write_text(json.dumps(result, indent=2) + "\n")
    print("BO-03: five assessments reproduced; 20 native development rows and 32 historical pairs retained; no builds started.")
