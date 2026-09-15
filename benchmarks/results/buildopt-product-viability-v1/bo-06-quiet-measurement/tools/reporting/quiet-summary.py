"""Summarize retained quiet-start and recorder receipts without launching work."""
from pathlib import Path
import json
import sys

ROOT = Path(__file__).resolve().parents[1]


def read(path):
    return json.loads(path.read_text())


rows = []
resources = []
for profile in sorted((ROOT / "profiles").iterdir()):
    run = profile / "run"
    for path in sorted((run / "attempts").glob("*/quiet-start.json")):
        quiet = read(path)
        cost = read(run / "costs" / (quiet["attempt"] + "-quiet-start.json"))
        assert cost["class"] == "research" and not cost["insideEnvelope"]
        row = {
            "profile": profile.name,
            "attempt": quiet["attempt"],
            "decision": quiet["decision"],
            "reason": quiet["reason"],
            "observationSeconds": (quiet["end"]["ns"] - quiet["begin"]["ns"]) / 1e9,
            "phaseSeconds": cost["durationNS"] / 1e9,
            "samples": len(quiet["samples"]),
            "nativeFinishRecorded": False,
            "launchAgeMS": None,
        }
        native_path = path.parent / "native-finish.json"
        if native_path.exists():
            native = read(native_path)
            age = native["start"]["ns"] - quiet["samples"][-1]["end"]["ns"]
            assert quiet["decision"] == "QUIET_START" and 0 <= age <= 1_000_000_000
            row.update(nativeFinishRecorded=True, launchAgeMS=age / 1e6)
        rows.append(row)
    for path in sorted((run / "recorder-resources").glob("*.json")):
        record = read(path)
        cost_path = run / "costs" / path.name
        if not cost_path.exists():
            raise AssertionError("Resource receipt without completed cost: " + str(path))
        cost = read(cost_path)
        before, after = record["before"], record["after"]
        assert (before["pid"], before["startTicks"]) == (after["pid"], after["startTicks"])
        assert before["end"]["ns"] <= cost["startNS"] <= cost["endNS"] <= after["begin"]["ns"]
        cpu = sum(after["cpu"][key] - before["cpu"][key] for key in ("userNS", "systemNS"))
        io = {key: after["io"][key] - value for key, value in before["io"].items()}
        assert cpu >= 0 and all(value >= 0 for value in io.values())
        resources.append({"profile": profile.name, "phase": record["phase"],
                          "purpose": cost["purpose"], "elapsedSeconds": cost["durationNS"] / 1e9,
                          "outerCPUSeconds": cpu / 1e9, "ioCounterChanges": io})

result = {
    "scope": "Retained observations; no new builds, retries or output comparisons.",
    "quietStarts": rows,
    "totalQuietPhaseSeconds": sum(row["phaseSeconds"] for row in rows),
    "recorderPhases": resources,
    "interpretation": "Waiting is research preparation outside request timing and within the allocation. "
                      "CPU and I/O counters cover only the outer recorder. They are not additional elapsed "
                      "time and do not establish the cause of any build-time difference.",
}
destination = ROOT / "analysis/quiet-start-and-recorder.json"
with destination.open("x") as output:
    json.dump(result, output, indent=2)
    output.write("\n")
print(json.dumps({"quietRecords": len(rows), "recorderRecords": len(resources),
                  "quietPhaseSeconds": result["totalQuietPhaseSeconds"]}))
