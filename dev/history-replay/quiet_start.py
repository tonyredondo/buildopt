#!/usr/bin/env python3
"""Observe a quiet start window; never launch or cancel a build."""
import argparse
import json
from pathlib import Path
import time

RESOURCES = ("cpu", "io", "memory")
POLICY = {
    "schema": "buildopt.quiet-start/v1",
    "windowNS": 30_000_000_000,
    "maximumGapNS": 3_000_000_000,
    "maximumReadNS": 100_000_000,
    "maximumSomePressure": 0.10,
    "sampleIntervalNS": 1_000_000_000,
}


def boot_ns():
    return time.clock_gettime_ns(time.CLOCK_BOOTTIME)


def snapshot():
    start = boot_ns()
    totals = {}
    for resource in RESOURCES:
        lines = Path("/proc/pressure", resource).read_text().splitlines()
        line = next(line for line in lines if line.startswith("some "))
        total = int(dict(item.split("=", 1) for item in line.split()[1:])["total"])
        if total < 0:
            raise ValueError("negative pressure counter")
        totals[resource] = total
    return {"startNS": start, "endNS": boot_ns(), "totalUS": totals}


def assess(samples):
    """Require a complete recent window; busy or missing intervals delay entry."""
    if not samples:
        return {"ready": False, "reason": "no samples"}
    end = samples[-1]["endNS"]
    cutoff = end - POLICY["windowNS"]
    first = next((i for i in range(len(samples) - 1, -1, -1)
                  if samples[i]["endNS"] <= cutoff), None)
    if first is None:
        return {"ready": False, "reason": "window incomplete"}
    window = samples[first:]
    peaks = {resource: 0.0 for resource in RESOURCES}
    for sample in window:
        if not 0 <= sample["endNS"] - sample["startNS"] <= POLICY["maximumReadNS"]:
            return {"ready": False, "reason": "pressure read delayed"}
    for before, after in zip(window, window[1:]):
        gap = after["endNS"] - before["endNS"]
        if not 0 < gap <= POLICY["maximumGapNS"] or after["startNS"] < before["endNS"]:
            return {"ready": False, "reason": "sample gap or clock discontinuity"}
        for resource in RESOURCES:
            delta = after["totalUS"][resource] - before["totalUS"][resource]
            if delta < 0:
                return {"ready": False, "reason": "pressure counter reset"}
            peaks[resource] = max(peaks[resource], delta * 1000 / gap)
    ready = all(value <= POLICY["maximumSomePressure"] for value in peaks.values())
    return {"ready": ready, "reason": "quiet window" if ready else "pressure in window",
            "startNS": window[0]["endNS"], "endNS": end,
            "durationNS": end - window[0]["endNS"], "peakIntervalPressure": peaks}


def observe(timeout_seconds, read=snapshot, clock=boot_ns, sleep=time.sleep):
    start = clock()
    deadline = start + int(timeout_seconds * 1e9)
    cpu_start = time.process_time_ns()
    samples = []
    result = {"decision": "NOT_QUIET_TIMEOUT", "policy": POLICY, "samples": samples,
              "startNS": start, "timeoutSeconds": timeout_seconds}
    try:
        while clock() < deadline:
            samples.append(read())
            result["assessment"] = assess(samples)
            if clock() >= deadline:
                break
            if result["assessment"]["ready"]:
                result["decision"] = "QUIET_START"
                break
            sleep(min(POLICY["sampleIntervalNS"], max(0, deadline - clock())) / 1e9)
    except KeyboardInterrupt:
        result["decision"] = "INTERRUPTED"
    except (OSError, ValueError, KeyError, StopIteration) as error:
        result["decision"] = "OBSERVATION_FAILED"
        result["error"] = f"{type(error).__name__}: {error}"
    result["endNS"] = clock()
    result["observerCPUNS"] = time.process_time_ns() - cpu_start
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--timeout-seconds", type=int, default=180)
    args = parser.parse_args()
    if not 1 <= args.timeout_seconds <= 180:
        parser.error("timeout must be between 1 and 180 seconds")
    # Reserve the output before waiting. Existing evidence is never overwritten.
    with args.output.open("x") as output:
        result = observe(args.timeout_seconds)
        json.dump(result, output, indent=2)
        output.write("\n")
    print(json.dumps({k: v for k, v in result.items() if k != "samples"}))
    return {"QUIET_START": 0, "NOT_QUIET_TIMEOUT": 2,
            "OBSERVATION_FAILED": 1, "INTERRUPTED": 130}[result["decision"]]


if __name__ == "__main__":
    raise SystemExit(main())
