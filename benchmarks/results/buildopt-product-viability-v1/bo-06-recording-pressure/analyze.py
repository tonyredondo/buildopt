#!/usr/bin/env python3
"""Reconstruct the diagnosis from the published control; start no builds."""
import collections
import gzip
import hashlib
import io
import json
from pathlib import Path
import sys
import tarfile

HERE = Path(__file__).resolve().parent
CONTROL = HERE.parent / "bo-06-qualified-measurement"
MANIFEST = json.loads((CONTROL / "evidence-manifest.json").read_text())
BINDINGS = {row["export"]: row for row in MANIFEST["files"]}
INPUTS = {}


def read(name):
    data = (CONTROL / name).read_bytes()
    digest = hashlib.sha256(data).hexdigest()
    assert digest == BINDINGS[name]["sha256"], f"changed control evidence: {name}"
    INPUTS[name] = digest
    return data


def document(name):
    return json.loads(read(name))


def overlap(a, b, c, d):
    return max(0, min(b, d) - max(a, c))


def prior_pressure(intervals, end):
    start = end - 30_000_000_000
    values = collections.defaultdict(float)
    coverage = 0
    gaps = []
    for row in intervals:
        dt = overlap(start, end, row["startBootNS"], row["endBootNS"])
        if not dt:
            continue
        coverage += dt
        gaps.append(row["endBootNS"] - row["startBootNS"])
        for key, value in row["pressure"].items():
            values[key] += dt * value
    assert coverage == end - start, "missing pressure boundary"
    return {"startNS": start, "endNS": end, "coverageSeconds": coverage / 1e9,
            "maximumSampleGapSeconds": max(gaps) / 1e9,
            "averageSomePressure": {key: value / coverage for key, value in values.items()},
            "method": "overlap-weighted sampled intervals; approximate at window boundaries"}


def analyze():
    pressure = document("analysis/control-host-pressure.json")
    cost_rows = []
    for name in sorted(BINDINGS):
        if name.startswith("profiles/control/run/costs/"):
            row = document(name)
            if "durationNS" in row:
                assert row["durationNS"] == row["endNS"] - row["startNS"]
                cost_rows.append(row)
    captures = [row for row in cost_rows if row["purpose"] == "complete-output-state-and-invocation-capture"]
    phase_totals = collections.Counter()
    for row in cost_rows:
        phase_totals[row["purpose"]] += row["durationNS"]
    rows = []
    finishes = {}
    for request in pressure["requests"]:
        attempt = request["attempt"]
        prefix = f"profiles/control/run/attempts/{attempt}/"
        finish = document(prefix + "native-finish.json")
        cpu = document(prefix + "supervisor-cpu.json")
        start, end = finish["start"]["ns"], finish["end"]["ns"]
        duration = end - start
        cpu_ns = sum(cpu["end"].values()) - sum(cpu["begin"].values())
        own_capture = next(row for row in captures if row["attempt"] == attempt)
        preceding = [row for row in captures if row["endNS"] <= start]
        previous = max(preceding, key=lambda row: row["endNS"]) if preceding else None
        overlaps = sum(overlap(start, end, row["startNS"], row["endNS"]) for row in captures)
        assert overlaps == 0 and own_capture["startNS"] > end
        rows.append({"attempt": attempt, "originalOrdinal": request["originalOrdinal"],
                     "arm": request["arm"], "nativeStartNS": start, "nativeEndNS": end,
                     "nativeSeconds": duration / 1e9, "supervisorCPUSeconds": cpu_ns / 1e9,
                     "supervisorCoreEquivalent": cpu_ns / duration,
                     "bulkCaptureOverlapSeconds": overlaps / 1e9,
                     "ownCaptureSeconds": own_capture["durationNS"] / 1e9,
                     "previousCapture": None if previous is None else {
                         "attempt": previous["attempt"], "endNS": previous["endNS"],
                         "gapToNativeSeconds": (start - previous["endNS"]) / 1e9},
                     "prior30Seconds": prior_pressure(pressure["intervals"], start),
                     "duringNative": {key: request[key] for key in (
                         "averagePressure", "samplingQualified", "maximumSampleGapSeconds")}})
        finishes[attempt] = finish
    assert len(rows) == len(captures) == 8

    # First/last observations within the last native pair. These omit process
    # lifetime boundaries and the outer recorder, and are not full I/O totals.
    selected = [row for row in rows if row["originalOrdinal"] == 20]
    observations = collections.defaultdict(list)
    ticks = set()
    with gzip.GzipFile(fileobj=io.BytesIO(read("profiles/control/process-samples.jsonl.gz"))) as stream:
        for line in stream:
            sample = json.loads(line)
            for row in selected:
                if not row["nativeStartNS"] <= sample["beginBootNS"] <= sample["endBootNS"] <= row["nativeEndNS"]:
                    continue
                ticks.add(sample["host"]["ticksPerSecond"])
                finish = finishes[row["attempt"]]
                for group in sample.get("groups", []):
                    if group["unit"] != finish["unit"]:
                        continue
                    for process in group["processes"]:
                        identity = process["identity"]
                        role = "supervisor" if (identity["pid"], identity["startTicks"]) == (
                            finish["supervisor"]["pid"], finish["supervisor"]["startTicks"]) else (
                            "Gradle daemon" if "org.gradle.launcher.daemon.bootstrap.GradleDaemon" in process["command"] else None)
                        if role:
                            key = row["attempt"], role, identity["pid"], identity["startTicks"]
                            observations[key].append((sample["endBootNS"], process))
    assert len(ticks) == 1 and len(observations) == 4
    process_rows = []
    for (attempt, role, pid, start_ticks), samples in sorted(observations.items()):
        first_ns, first = samples[0]
        last_ns, last = samples[-1]
        counters = {key: last["io"][key] - first["io"][key] for key in first["io"]}
        assert all(value >= 0 for value in counters.values())
        request = next(row for row in selected if row["attempt"] == attempt)
        process_rows.append({"attempt": attempt, "role": role, "pid": pid, "startTicks": start_ticks,
                             "sampleCount": len(samples), "firstSampleNS": first_ns, "lastSampleNS": last_ns,
                             "missingStartSeconds": (first_ns - request["nativeStartNS"]) / 1e9,
                             "missingEndSeconds": (request["nativeEndNS"] - last_ns) / 1e9,
                             "maximumSampleGapSeconds": max(b[0] - a[0] for a, b in zip(samples, samples[1:])) / 1e9,
                             "observedThreadMasks": sorted({mask for _, sample in samples for mask in sample["threadMasks"]}),
                             "sampledCPUSeconds": (last["identity"]["cpuTicks"] - first["identity"]["cpuTicks"]) / next(iter(ticks)),
                             "sampledMajorFaults": last["identity"]["majorFaults"] - first["identity"]["majorFaults"],
                             "sampledIOCounterDeltas": counters})

    manifest = document("profiles/control/manifest.json")
    source = {}
    with tarfile.open(fileobj=io.BytesIO(read("runner-source.tar.gz"))) as archive:
        for name in ("runner.go", "process_linux.go", "watch_linux.go", "files.go", "capture.init.gradle"):
            data = archive.extractfile(name).read()
            source[name] = hashlib.sha256(data).hexdigest()
    assert source["capture.init.gradle"] == manifest["outputs"]["graphCapture"]["sha256"]
    return {"schema": "buildopt.recording-pressure/v1", "decision": "CAUSE_NOT_FULLY_ATTRIBUTED",
            "originalDecisionUnchanged": "CONTROL_MATERIAL_DIFFERENCE", "newOwnerStarts": 0,
            "inputs": dict(sorted(INPUTS.items())), "measuredSourceFiles": source,
            "requests": rows, "phaseSeconds": {key: value / 1e9 for key, value in sorted(phase_totals.items())},
            "lastPairSampledProcesses": process_rows,
            "limits": ["Pre-build pressure is descriptive, not retrospective admission or sample filtering.",
                       "Supervisor CPU is measured; CPU spent in each observer function was not profiled.",
                       "Bulk capture follows native execution; lingering effects on later builds are unproven.",
                       "Process I/O covers sampled intervals, with missing boundaries and sampling gaps.",
                       "Outer recorder CPU/I/O, per-thread waits and per-cgroup I/O pressure were not captured.",
                       "Host pressure identifies stalls, not their responsible process."]}


if __name__ == "__main__":
    result = analyze()
    if sys.argv[1:] == ["--check"]:
        assert result == json.loads((HERE / "analysis.json").read_text()), "derived analysis differs"
        bindings = json.loads((HERE / "file-bindings.json").read_text())
        for row in bindings:
            path = HERE / row["path"]
            assert hashlib.sha256(path.read_bytes()).hexdigest() == row["sha256"], f"changed file: {path}"
        print(f"Verified {len(result['requests'])} existing build records, source bindings and {len(bindings)} new files; no builds started")
    elif not sys.argv[1:]:
        print(json.dumps(result, indent=2))
    else:
        raise SystemExit("usage: analyze.py [--check]")
