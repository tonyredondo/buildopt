#!/usr/bin/env python3
"""Compare retained Elasticsearch root-workflow outputs using frozen C5 readers.

No build or source mutation occurs here. Reused output normalization requires an
earlier, byte-identical executed origin in the same physical arm and replication.
The caller independently checks the full source/state/receipt chain.
"""
from collections import Counter
from pathlib import Path
import datetime
import hashlib
import json
import os
import subprocess
import sys
import zipfile


EXTRA = "server/build/checkstyle-content-aware.xml"
EXTRA_OWNER = ":server:copyCheckstyleConf"
REPLACEMENT = ('<module name="org.elasticsearch.gradle.internal.checkstyle.ContentAwareChecker">\n'
               '  <property name="buildoptStateDirectory" value="${config_loc}/${buildopt_state}" />')
CHECKSTYLE = {"server/build/reports/checkstyle/" + name + ".xml": ":server:checkstyle" + task
              for name, task in [("main", "Main"), ("test", "Test"), ("internalClusterTest", "InternalClusterTest")]}


def require(condition, message):
    if not condition:
        raise ValueError(message)


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        require(key not in result, "duplicate JSON key: " + key)
        result[key] = value
    return result


def parse(raw):
    return json.loads(raw, object_pairs_hook=unique_object)


def read(path):
    return parse(Path(path).read_bytes())


def sha(path):
    h = hashlib.sha256()
    with Path(path).open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            h.update(block)
    return h.hexdigest()


def bound(binding):
    path = Path(binding["path"])
    if path.is_dir():
        entries = []
        def visit(directory):
            for child in sorted(directory.iterdir()):
                info = child.lstat()
                entry = {"path": child.relative_to(path).as_posix(), "kind": "", "mode": info.st_mode & 0o777,
                         "size": info.st_size, "sha256": "", "target": ""}
                if child.is_symlink():
                    entry.update(kind="symlink", target=os.readlink(child))
                    entry["sha256"] = hashlib.sha256(entry["target"].encode()).hexdigest()
                elif child.is_dir():
                    entry.update(kind="directory", size=0)
                else:
                    require(child.is_file(), "special file in bound input")
                    entry.update(kind="file", sha256=sha(child))
                entries.append(entry)
                if entry["kind"] == "directory":
                    visit(child)
        visit(path)
        raw = json.dumps(sorted(entries, key=lambda e: e["path"]), indent=2, ensure_ascii=False) + "\n"
        actual = hashlib.sha256(raw.encode()).hexdigest()
    else:
        actual = sha(path)
    require(actual == binding["sha256"], "bound input drift: " + binding["path"])
    return Path(binding["path"])


def milliseconds(value):
    return int(datetime.datetime.fromisoformat(value.replace("Z", "+00:00")).timestamp() * 1000)


def rooted(value, root):
    """Map actual path prefixes, never arbitrary occurrences in diagnostics."""
    require(value.startswith(root + "/"), "path outside producer root: " + value)
    return value[len(root) + 1:]


class Capture:
    def __init__(self, end, verify_all_outputs=True):
        self.end = end
        self.path = bound(end["capture"])
        self.raw = read(self.path)
        self.native = read(bound(end["native"]))
        self.state = read(bound(end["after"]))
        self.start = read(self.path.parent / "start.json")
        self.root = None
        require(self.start["ordinal"] == self.state["ordinal"], "capture/state ordinal differs")
        require(self.start["arm"] == self.state["arm"] and self.start["replication"] == self.state["replication"], "capture/state arm differs")
        self.outputs = {}
        self.verified_outputs = set()
        for output in self.raw["outputs"]:
            path = output["entry"]["path"]
            require(path not in self.outputs, "duplicate captured output: " + path)
            require(output["transform"] == "exact", "owner capture must retain exact bytes")
            self.outputs[path] = output
            if output["entry"]["kind"] == "file":
                artifact = Path(output["raw"]["path"])
                require(output["raw"]["sha256"] == output["entry"]["sha256"], "raw output identity differs: " + path)
                require(artifact.is_relative_to(self.path.parent / "outputs"), "output borrowed from another attempt")
                if verify_all_outputs:
                    self.verify_output(path)
        self.tasks, self.outcomes = {}, {}
        require(len(self.raw["graph"]) == 1, "complete graph missing")
        for line in bound(self.raw["graph"][0]).read_text().splitlines():
            event = parse(line)
            if event["buildPath"] != ":":
                continue
            if self.root is None:
                require(event["kind"] == "graph" and str(Path(event["root"]).parent) == self.state["root"], "graph outside bound arm state")
                self.root = event["root"]
            require(event["root"] == self.root, "producer root differs")
            if event["kind"] == "graph":
                for task in event["tasks"]:
                    require(task["identity"] not in self.tasks, "duplicate graph task")
                    self.tasks[task["identity"]] = task
            else:
                task = event["task"]["identity"]
                require(event["kind"] == "task" and task in self.tasks and task not in self.outcomes, "invalid outcome identity")
                start, end = milliseconds(event["startedUTC"]), milliseconds(event["utc"])
                require(milliseconds(self.native["start"]["utc"]) - 1 <= start <= end <= milliseconds(self.native["end"]["utc"]) + 1, "producer outside native request")
                self.outcomes[task] = {**event["task"], "start": start, "end": end}
        require(bool(self.tasks), "empty owner graph")
        require([{k: value[k] for k in ("identity", "outcome", "action")} for value in self.outcomes.values()] == self.raw["tasks"], "captured outcomes differ from graph")

    def verify_output(self, path):
        require(path in self.outputs and self.outputs[path]["entry"]["kind"] == "file", "required file missing: " + path)
        if path not in self.verified_outputs:
            output = self.outputs[path]
            artifact = bound(output["raw"])
            require(artifact.stat().st_size == output["entry"]["size"], "raw output identity differs: " + path)
            self.verified_outputs.add(path)

    def data(self, path):
        self.verify_output(path)
        return Path(self.outputs[path]["raw"]["path"]).read_bytes()

    def owner(self, path, owner):
        require(path in self.outputs and owner in self.outputs[path]["producer"].split(","), "unproven producer for " + path)
        require(owner in self.tasks and owner in self.outcomes, "missing producer outcome: " + owner)
        return self.outcomes[owner]


class Origins:
    def __init__(self, run_root, captures):
        self.root = Path(run_root)
        self.current = captures
        self.loaded = {}
        self.indexed = {}
        self.donors = {}
        self.reused = 0

    def executed(self, current, path, owner):
        event = current.owner(path, owner)
        if event["outcome"] == "EXECUTED" and event["action"]:
            return current, event
        require(event["outcome"] in ("UP-TO-DATE", "FROM-CACHE"), "producer did not execute or reuse qualified output: " + owner)
        # Search only earlier sealed successful attempts. No other root, arm,
        # replication, manifest, future ordinal or partial attempt may donate.
        key = (current.start["replication"], current.start["arm"], current.start["ordinal"])
        if key not in self.loaded:
            candidates = []
            for start_path in sorted((self.root / "attempts").glob("*/start.json")):
                start = read(start_path)
                if start["replication"] != key[0] or start["arm"] != key[1] or start["ordinal"] >= key[2]:
                    continue
                if start["manifestSHA256"] != current.start["manifestSHA256"]:
                    continue
                end_path = start_path.parent / "end.json"
                if not end_path.is_file():
                    continue
                end = read(end_path)
                if end["class"] not in ("COMPARABLE", "NATIVE_RETAINED"):
                    continue
                candidates.append((start["ordinal"], end))
            self.loaded[key] = sorted(candidates, key=lambda item: item[0], reverse=True)
        target = current.outputs[path]
        for _, end in self.loaded[key]:
            identity = (end["capture"]["path"], end["capture"]["sha256"])
            if identity not in self.indexed:
                raw = read(bound(end["capture"]))
                executed = {task["identity"] for task in raw["tasks"] if task["outcome"] == "EXECUTED" and task["action"]}
                # Intermediate UP-TO-DATE snapshots cannot donate an executed
                # origin. Keep only a small source index before loading a donor.
                self.indexed[identity] = {output["entry"]["path"]: output for output in raw["outputs"]
                                          if executed.intersection(output["producer"].split(","))}
            old = self.indexed[identity].get(path)
            if old is None or old["entry"] != target["entry"] or old["producer"] != target["producer"]:
                continue
            if identity not in self.donors:
                # All current outputs are verified above. The caller/checker
                # independently verifies every retained attempt. Here verify
                # the actual origin's graph/state plus each donated raw file,
                # without rereading unrelated historical output bytes per pair.
                self.donors[identity] = Capture(end, verify_all_outputs=False)
            prior = self.donors[identity]
            if (prior.root != current.root or prior.native["exitCode"] != 0
                    or prior.start["replication"] != key[0] or prior.start["arm"] != key[1]
                    or prior.start["ordinal"] >= key[2]
                    or prior.start["manifestSHA256"] != current.start["manifestSHA256"]):
                continue
            event = prior.owner(path, owner)
            if event["outcome"] == "EXECUTED" and event["action"]:
                prior.verify_output(path)
                self.reused += 1
                return prior, event
        raise ValueError("no earlier executed origin for reused output: " + path)


def graph_contract(captures, candidate_applied, control_adapted=False):
    n, i = captures
    require(n.tasks.keys() == i.tasks.keys(), "owner task identity set differs")
    for identity in n.tasks:
        left, right = n.tasks[identity], i.tasks[identity]
        require(left["taskClass"] == right["taskClass"] and left["dependencies"] == right["dependencies"] and left["path"] == right["path"], "required task contract differs: " + identity)
        output_sets = [set(rooted(path, c.root) for path in c.tasks[identity]["outputs"]) for c in captures]
        if EXTRA in output_sets[0]:
            require(control_adapted and identity == EXTRA_OWNER, "native declares candidate-only output")
        if EXTRA in output_sets[1]:
            require(candidate_applied and identity == EXTRA_OWNER, "unapproved extra output producer")
        require(output_sets[0] - {EXTRA} == output_sets[1] - {EXTRA}, "declared output roots differ: " + identity)
        contracts = []
        for c in captures:
            inputs = c.tasks[identity]["checkstyle"]
            require(inputs["classpath"] == [], "unqualified Checkstyle analysis classpath")
            contracts.append({"source": [rooted(path, c.root) for path in inputs["source"]],
                              "reports": [{**report, "path": rooted(report["path"], c.root)} for report in inputs["reports"]]})
        require(contracts[0] == contracts[1], "Checkstyle source/report contract differs: " + identity)
    failures = [{identity: event["outcome"] == "FAILED" for identity, event in c.outcomes.items()} for c in captures]
    require(failures[0] == failures[1], "task failure coverage differs")


def extra_contract(captures, candidate_applied, control_adapted=False):
    for capture, adapted in zip(captures, (control_adapted, candidate_applied)):
        if not adapted:
            require(EXTRA not in capture.outputs, "native has candidate-only output")
            continue
        require(EXTRA in capture.outputs, "candidate config declaration missing")
        extra = capture.outputs[EXTRA]
        require(extra["producer"] == EXTRA_OWNER, "unapproved candidate-only output")
        if extra["entry"]["kind"] == "absent":
            require(capture.native["exitCode"] != 0, "successful candidate is missing generated config")
            continue
        original = capture.data("server/build/checkstyle/checkstyle.xml")
        require(original.count(b'<module name="Checker">') == 1, "ambiguous original Checkstyle config")
        require(capture.data(EXTRA) == original.replace(b'<module name="Checker">', REPLACEMENT.encode()), "generated candidate config differs")
        require(extra["entry"]["mode"] == capture.outputs["server/build/checkstyle/checkstyle.xml"]["entry"]["mode"], "generated config mode differs")


def member(capture, path, name):
    if not name:
        return capture.data(path)
    with zipfile.ZipFile(capture.outputs[path]["raw"]["path"]) as archive:
        require(archive.namelist().count(name) == 1, "missing/duplicate provenance member")
        return archive.read(name)


def compare(request):
    policy_path = bound(request["policy"])
    policy = read(policy_path)
    for field in ("baseContract", "dateContract", "interpreter", "comparator", "lexicalProjector", "metadataJava"):
        bound(policy[field])
    for binding in policy["metadataClasspath"]:
        bound(binding)
    require(policy["reusePolicy"] == "same-arm-earlier-executed-exact-raw-v1", "unsupported reuse policy")
    date_rules = {rule["path"]: rule for rule in read(policy["dateContract"]["path"])["allowlist"]}
    captures = [Capture(request[arm]) for arm in ("native", "candidate")]
    n, i = captures
    require(n.native["exitCode"] == i.native["exitCode"], "workflow exit differs")
    control_adapted = request.get("controlAdapted", False)
    require(type(control_adapted) is bool, "invalid control adapter declaration")
    graph_contract(captures, request["candidate"]["candidateApplied"], control_adapted)
    extra_contract(captures, request["candidate"]["candidateApplied"], control_adapted)
    require(set(n.outputs) - {EXTRA} == set(i.outputs) - {EXTRA}, "required output coverage differs")
    origins = Origins(request["runRoot"], captures)
    counts = Counter()
    requests, metadata = [], []
    for path, left in n.outputs.items():
        right = i.outputs[path]
        for field in ("kind", "mode", "target"):
            require(left["entry"][field] == right["entry"][field], "output metadata differs: " + path)
        require(left["producer"] == right["producer"], "output owners differ: " + path)
        if left["entry"]["kind"] != "file" or left["entry"]["sha256"] == right["entry"]["sha256"]:
            counts["exact"] += 1
            continue
        if path in date_rules:
            rule = date_rules[path]
            for c in captures:
                origin, _ = origins.executed(c, path, rule["producer"])
                for source in rule["manifestProvenance"]:
                    origins.executed(c, source["sourceJar"], source["sourceProducer"])
                    require(member(c, source["sourceJar"], "META-INF/MANIFEST.MF") == member(c, path, source["member"]), "manifest provenance differs: " + path)
                requests.append({"kind": "archive" if rule["allowedMembers"] else "manifest", "path": c.outputs[path]["raw"]["path"], "members": rule["allowedMembers"],
                                 "start": milliseconds(origin.native["start"]["utc"]), "end": milliseconds(origin.native["end"]["utc"])})
            counts["date"] += 1
        elif path == "server/build/reports/licenseHeaders/rat.xml":
            for c in captures:
                _, window = origins.executed(c, path, ":server:licenseHeaders")
                requests.append({"kind": "rat", "path": c.outputs[path]["raw"]["path"], "root": c.root, "start": window["start"], "end": window["end"]})
            counts["rat"] += 1
        elif path in CHECKSTYLE:
            for c in captures:
                require(c.outputs[path]["producer"] == CHECKSTYLE[path], "Checkstyle report producer differs")
                origins.executed(c, path, CHECKSTYLE[path])
                requests.append({"kind": "checkstyle", "path": c.outputs[path]["raw"]["path"], "root": c.root})
            counts["checkstyle"] += 1
        elif path.endswith("/previous-compilation-data.bin"):
            owner = left["producer"]
            require(owner in n.tasks and n.tasks[owner]["taskClass"] == "org.gradle.api.tasks.compile.JavaCompile_Decorated", "unqualified compiler-state producer: " + path)
            metadata.extend(c.outputs[path]["raw"]["path"] for c in captures)
            counts["compilationMetadata"] += 1
        else:
            raise ValueError("unapproved output difference: " + path)
    if requests:
        result = subprocess.run([policy["lexicalProjector"]["path"]], input="".join(json.dumps(r) + "\n" for r in requests), text=True, capture_output=True, timeout=90)
        require(result.returncode == 0, "lexical projector failed: " + result.stderr[-1000:])
        responses = [parse(line) for line in result.stdout.splitlines()]
        require(len(responses) == len(requests), "missing lexical projection")
        for query, response in zip(requests, responses):
            require(set(response) == {"path", "projection"} and response["path"] == query["path"], "rejected/malformed lexical projection: " + str(response))
        for left, right in zip(responses[::2], responses[1::2]):
            require(left["projection"] == right["projection"], "non-permitted projected difference: " + left["path"])
    if metadata:
        command = [policy["metadataJava"]["path"], "-Xmx512m", "-cp", ":".join(b["path"] for b in policy["metadataClasspath"]), "MetadataProjection"]
        result = subprocess.run(command, input="".join(path + "\n" for path in metadata), text=True, capture_output=True, timeout=90)
        require(result.returncode == 0, "complete compiler-state projector failed: " + result.stderr[-1000:])
        values = [line.split("\t") for line in result.stdout.splitlines()]
        require(len(values) == len(metadata), "missing compilation-state projection")
        for path, row in zip(metadata, values):
            require(len(row) == 2 and row[0] == sha(path) and len(row[1]) == 64, "unbound metadata projection")
        for left, right in zip(values[::2], values[1::2]):
            require(left[1] == right[1], "complete compilation-state values differ")
    counts["causalOrigins"] = origins.reused
    counts["candidateOnly"] = int(EXTRA in i.outputs and EXTRA not in n.outputs)
    return {"status": "equivalent", "policySHA256": request["policy"]["sha256"],
            "captures": [c.end["capture"] for c in captures], "counts": dict(counts), "metadataJVMStarts": int(bool(metadata))}


if __name__ == "__main__":
    try:
        print(json.dumps(compare(parse(sys.stdin.read()))))
    except (ValueError, KeyError, OSError, subprocess.SubprocessError, zipfile.BadZipFile) as error:
        print(str(error), file=sys.stderr)
        sys.exit(1)
