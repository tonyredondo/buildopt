#!/usr/bin/env python3
"""Recompute the storage diagnosis from published receipts; never launch a build."""

import argparse
import gzip
import hashlib
import json
from collections import Counter, defaultdict
from pathlib import Path


HERE = Path(__file__).resolve().parent
EVIDENCE = HERE.parent / "bo-06-quiet-measurement"
MANIFEST_SHA = "a429e8b827e56c8654eec431f63252cc4376ae76e53028006fedde21d2f98262"


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def analyze():
    manifest_path = EVIDENCE / "evidence-manifest.json"
    assert sha(manifest_path) == MANIFEST_SHA, "predecessor manifest changed"
    manifest = json.loads(manifest_path.read_text())
    bindings = {row["export"]: row["sha256"] for row in manifest["files"]}
    used = {}

    def checked(path):
        rel = path.relative_to(EVIDENCE).as_posix()
        digest = sha(path)
        assert digest == bindings[rel], f"input changed: {rel}"
        used[rel] = digest
        return path

    def read(path):
        return json.loads(checked(path).read_text())

    for name in ("runner-source.tar.gz", "source-archive-members.json",
                 "inputs/quiet-start-policy.json"):
        checked(EVIDENCE / name)

    supervisors, phases, quiets = [], [], []
    for profile in ("control", "prefix"):
        root = EVIDENCE / "profiles" / profile / "run"
        for path in sorted((root / "attempts").glob("*/quiet-start.json")):
            data = read(path)
            quiets.append({"profile": profile, "attempt": path.parent.name,
                           "decision": data["decision"], "samples": len(data["samples"]),
                           "seconds": (data["end"]["ns"] - data["begin"]["ns"]) / 1e9})
        for path in sorted((root / "attempts").glob("*/supervisor-cpu.json")):
            cpu, native = read(path), read(path.with_name("native-finish.json"))
            seconds = (native["end"]["ns"] - native["start"]["ns"]) / 1e9
            cpu_seconds = sum(cpu["end"][key] - cpu["begin"][key]
                              for key in ("userNS", "systemNS")) / 1e9
            supervisors.append({"profile": profile, "attempt": path.parent.name,
                                "nativeSeconds": seconds, "supervisorCPUSeconds": cpu_seconds,
                                "cpuSecondsPerNativeSecond": cpu_seconds / seconds})
        for path in sorted((root / "recorder-resources").glob("*.json")):
            resource = read(path)
            cost = read(root / "costs" / path.name)
            before, after = resource["before"], resource["after"]
            assert (before["pid"], before["startTicks"]) == (after["pid"], after["startTicks"])
            phases.append({"profile": profile, "phase": resource["phase"],
                           "purpose": cost["purpose"], "seconds": cost["durationNS"] / 1e9,
                           "beginUTC": before["begin"]["utc"], "endUTC": after["end"]["utc"],
                           "cpuSeconds": sum(after["cpu"][k] - before["cpu"][k]
                                             for k in ("userNS", "systemNS")) / 1e9,
                           **{k: after["io"][k] - before["io"][k]
                              for k in ("read_bytes", "write_bytes", "cancelled_write_bytes")}})
    assert len(supervisors) == 25 and len(quiets) == 26
    assert Counter(q["decision"] for q in quiets) == {"QUIET_START": 25, "NOT_QUIET_TIMEOUT": 1}

    refused = read(EVIDENCE / "profiles/prefix/run/attempts/r1-008-g0-I/quiet-start.json")
    samples = refused["samples"]
    intervals = [(b["end"]["ns"] - a["end"]["ns"]) / 1e9 for a, b in zip(samples, samples[1:])]
    assert all(dt > 0 for dt in intervals)
    pressure = {}
    for kind in ("cpu", "io", "memory"):
        fractions = [(b["totalsUS"][kind] - a["totalsUS"][kind]) / (dt * 1e6)
                     for a, b, dt in zip(samples, samples[1:], intervals)]
        pressure[kind] = {"intervalsAboveTenPercent": sum(f > .1 for f in fractions),
                          "weightedFraction": sum(f * dt for f, dt in zip(fractions, intervals)) / sum(intervals),
                          "minFraction": min(fractions), "maxFraction": max(fractions)}

    selected = []
    with gzip.open(checked(EVIDENCE / "profiles/prefix/process-samples.jsonl.gz"), "rt") as stream:
        for line in stream:
            row = json.loads(line)
            if refused["begin"]["ns"] <= row["beginBootNS"] and row["endBootNS"] <= refused["end"]["ns"]:
                selected.append(row)
    assert len(selected) >= 2

    def processes(row):
        result = {}
        for group in row["groups"]:
            for process in group["processes"]:
                ident = process["identity"]
                key = (ident["pid"], ident["startTicks"])
                assert key not in result
                result[key] = process
        return result

    observed = [processes(row) for row in selected]
    identities = set(observed[0])
    assert all(set(row) == identities for row in observed), "process coverage changed"
    assert all(not process["unavailableObservations"] for row in observed for process in row.values())
    assert all(row["host"]["ticksPerSecond"] == 100 for row in selected)
    owned = []
    for key in sorted(identities):
        sequence = [row[key] for row in observed]
        first, last = sequence[0], sequence[-1]
        assert all(p["command"] == first["command"] for p in sequence)
        command = first["command"]
        role = ("supervisor" if "history-replay-quiet worker " in command else
                "Gradle daemon" if "org.gradle.launcher.daemon.bootstrap.GradleDaemon" in command else
                "Gradle worker" if "GradleWorkerMain" in command else "other")
        deltas = {k: last["io"][k] - first["io"][k] for k in ("read_bytes", "write_bytes")}
        cpu_ticks = last["identity"]["cpuTicks"] - first["identity"]["cpuTicks"]
        assert cpu_ticks >= 0 and all(v >= 0 for v in deltas.values())
        owned.append({"pid": key[0], "startTicks": key[1], "role": role,
                      "cpuTicks": cpu_ticks, "cpuSeconds": cpu_ticks / 100, **deltas,
                      "mainThreadStates": sorted({p["identity"]["state"] for p in sequence})})

    groups = defaultdict(lambda: {"phases": 0, "seconds": 0, "cpuSeconds": 0,
                                  "read_bytes": 0, "write_bytes": 0, "cancelled_write_bytes": 0})
    for phase in phases:
        total = groups[(phase["profile"], phase["purpose"])]
        total["phases"] += 1
        for k in total.keys() - {"phases"}:
            total[k] += phase[k]
    span = (selected[-1]["endBootNS"] - selected[0]["endBootNS"]) / 1e9
    result = {
        "schema": "buildopt.storage-diagnosis/v1", "ownerStarts": 0, "comparatorStarts": 0,
        "sourceManifestSHA256": MANIFEST_SHA,
        "quietDecisions": quiets, "supervisors": supervisors,
        "supervisorRatioRange": [min(r["cpuSecondsPerNativeSecond"] for r in supervisors),
                                 max(r["cpuSecondsPerNativeSecond"] for r in supervisors)],
        "recorderPhaseTotals": [{"profile": k[0], "purpose": k[1], **v} for k, v in sorted(groups.items())],
        "recorderPhases": phases,
        "failedWait": {"beginUTC": refused["begin"]["utc"], "endUTC": refused["end"]["utc"],
                       "quietIntervals": len(intervals), "pressure": pressure,
                       "processSamples": len(selected), "processSampleSpanSeconds": span,
                       "processSampleBoundaryGapsSeconds": [
                           (selected[0]["beginBootNS"] - refused["begin"]["ns"]) / 1e9,
                           (refused["end"]["ns"] - selected[-1]["endBootNS"]) / 1e9],
                       "largestProcessSampleGapSeconds": max(
                           (b["endBootNS"] - a["endBootNS"]) / 1e9 for a, b in zip(selected, selected[1:])),
                       "sameProcessIdentitiesInEverySample": True,
                       "ownedProcesses": owned,
                       "ownedCPUSeconds": sum(p["cpuTicks"] for p in owned) / 100,
                       "ownedReadBytes": sum(p["read_bytes"] for p in owned),
                       "ownedWriteBytes": sum(p["write_bytes"] for p in owned),
                       "hostBlockedProcessRange": [min(r["host"]["cpu"]["procs_blocked"] for r in selected),
                                                   max(r["host"]["cpu"]["procs_blocked"] for r in selected)]},
        "limits": ["Sample timestamps precede sequential host reads; alignment is approximate.",
                   "Process coverage is sampled; short-lived processes between samples are not excluded.",
                   "Process state and wait channel describe the main thread only.",
                   "The outer recorder is measured by phase; the Python observer and external processes are not.",
                   "No historical device counters, Dirty/Writeback or group IO pressure establish the storage cause.",
                   "CPU and IO counters are not elapsed-time savings or proof of causal interference."],
        "verifiedInputs": dict(sorted(used.items())),
    }
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="compare with the published derivation")
    args = parser.parse_args()
    if args.check:
        manifest = json.loads((HERE / "evidence-manifest.json").read_text())
        for row in manifest["files"]:
            path = HERE / row["path"]
            assert path.stat().st_size == row["bytes"] and sha(path) == row["sha256"], path
    result = analyze()
    target = HERE / "analysis.json"
    if args.check:
        assert json.loads(target.read_text()) == result, "derived analysis changed"
        print(f"Verified {len(result['verifiedInputs'])} inputs, 26 quiet decisions and 25 builds; no builds launched.")
    else:
        target.write_text(json.dumps(result, indent=2, sort_keys=True) + "\n")


if __name__ == "__main__":
    main()
