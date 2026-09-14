"""Source-bound Checkstyle ceiling on all 20 already consumed engineering transitions."""
import functools
import hashlib
import json
from pathlib import Path

HERE = Path(__file__).resolve().parent
STATE = HERE.parent
TARGETS = [":server:checkstyleMain", ":server:checkstyleTest", ":server:checkstyleInternalClusterTest"]
rows = []


def key(identity):
    return identity["buildPath"], identity["taskPath"]


for ordinal in range(21):
    directory = STATE / "bv002/native-prefix-nocc" / f"{ordinal:03d}"
    diag = json.loads((directory / "diagnostics.json").read_text())
    receipt = json.loads((directory / "receipt.json").read_text())
    assert receipt["exitCode"] == 0 and diag["completeExecutedGraph"]
    trace = directory / "operations-log.txt"
    with trace.open("rb") as raw:
        assert hashlib.file_digest(raw, "sha256").hexdigest() == diag["sha256"]
    ops = {}; snapshots = {}
    for line in trace.open():
        event = json.loads(line)
        if "startTime" in event:
            ops[event["id"]] = {"name": event["displayName"], "parent": event.get("parentId"), "start": event["startTime"]}
        elif "endTime" in event:
            op = ops[event["id"]]
            op.update(end=event["endTime"], milliseconds=event["endTime"] - op["start"])
            for target in TARGETS:
                if op["name"] == "Snapshot task inputs for " + target:
                    snapshots[target] = event.get("result", {})
    selected = {}
    for target in TARGETS:
        identifier = next(i for i, o in ops.items() if o["name"] == "Task " + target)
        descendants = []
        for i, op in ops.items():
            parent = op["parent"]
            while parent is not None and parent != identifier:
                parent = ops[parent]["parent"]
            if parent == identifier:
                descendants.append(op)
        actions = [o for o in descendants if o["name"] == "Execute run for " + target]
        workers = [o for o in descendants if o["name"] == "org.gradle.api.plugins.quality.internal.CheckstyleInvoker"]
        selected[target] = {"taskMilliseconds": ops[identifier]["milliseconds"],
                            "actionMilliseconds": sum(o["milliseconds"] for o in actions),
                            "workerMilliseconds": sum(o["milliseconds"] for o in workers),
                            "snapshotMilliseconds": sum(o["milliseconds"] for o in descendants if o["name"].startswith("Snapshot task inputs")),
                            "actions": actions, "workers": workers}
    tasks = {key(t["details"]): t for t in diag["tasks"]}
    plans = {key(n["task"]): n for g in diag["graphs"] for n in g["taskPlan"]}

    def path(remove_actions):
        @functools.lru_cache(None)
        def walk(identity):
            if identity not in tasks:
                return 0
            duration = tasks[identity]["milliseconds"]
            if remove_actions and identity[0] == ":" and identity[1] in selected:
                duration -= selected[identity[1]]["actionMilliseconds"]
            deps = [key(d) for d in plans[identity]["nodeDependencies"] if d.get("nodeType") == "TASK"]
            return duration + max((walk(d) for d in deps), default=0)
        return walk((":", ":server:precommit"))

    intervals = sorted((o["start"], o["end"]) for data in selected.values() for o in data["actions"])
    union = []
    for start, end in intervals:
        if union and start <= union[-1][1]:
            union[-1][1] = max(union[-1][1], end)
        else:
            union.append([start, end])
    row = {"ordinal": ordinal, "role": "anchor" if ordinal == 0 else "engineering",
           "workflowMilliseconds": diag["workflow"]["milliseconds"], "commandSeconds": receipt["elapsedSeconds"],
           "selected": selected, "actionUnionMilliseconds": sum(b-a for a,b in union),
           "nativeDependencyPathMilliseconds": path(False), "zeroActionDependencyPathMilliseconds": path(True),
           "dependencyModelSavingMilliseconds": path(False)-path(True), "rawTraceSHA256": diag["sha256"]}
    rows.append(row)
    if ordinal == 20:
        (HERE / "native-source-snapshots.json").write_text(json.dumps(snapshots, indent=2) + "\n")
    print(f"ordinal={ordinal:02d} actionUnionMs={row['actionUnionMilliseconds']} dependencyModelSavingMs={row['dependencyModelSavingMilliseconds']}", flush=True)
transitions = rows[1:]
workflow = sum(r["workflowMilliseconds"] for r in transitions)
model = sum(r["dependencyModelSavingMilliseconds"] for r in transitions)
result = {"step": "C3", "status": "source-and-component-review-required",
          "nativeCacheDecision": "INCOMPATIBLE; native baseline remains full Checkstyle with ordinary Gradle avoidance/build cache",
          "rows": rows, "totals": {"scheduledTransitions": 20, "workflowMilliseconds": workflow,
            "actionUnionMilliseconds": sum(r["actionUnionMilliseconds"] for r in transitions),
            "sumOverlappingActionMilliseconds": sum(v["actionMilliseconds"] for r in transitions for v in r["selected"].values()),
            "sumOverlappingWorkerMilliseconds": sum(v["workerMilliseconds"] for r in transitions for v in r["selected"].values()),
            "dependencyModelSavingMilliseconds": model, "dependencyModelSecondsPerTransition": model/20000,
            "dependencyModelPercent": model/workflow*100, "ceilingReachesFloors": model >= 20000 and model/workflow >= .05},
          "interpretation": "Optimistic fixed-duration explicit-dependency model, not measured savings. Retains task overhead; removes all selected actions including mandatory reports and changed-file work. Shared-resource effects and actual report cost are not derived from this trace.",
          "validationOrdinalsConsumed": [], "candidateImplemented": False}
(HERE / "residual-opportunity.json").write_text(json.dumps(result, indent=2) + "\n")
print(json.dumps(result["totals"], indent=2))
