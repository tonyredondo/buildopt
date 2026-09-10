#!/usr/bin/env python3
"""Owned executable rejection fixtures for the C5 replay adapter."""
from pathlib import Path
import importlib.util
import json
import os
import tempfile
import unittest

spec = importlib.util.spec_from_file_location("owner_compare", Path(__file__).with_name("owner_compare.py"))
owner = importlib.util.module_from_spec(spec)
spec.loader.exec_module(owner)


def save(path, value):
    path.write_text(json.dumps(value, indent=2) + "\n")
    return binding(path)


def binding(path):
    return {"path": str(path), "sha256": owner.sha(path)}


class OwnerTests(unittest.TestCase):
    def setUp(self):
        base = Path(os.environ["BUILDOPT_REPLAY_TEST_ROOT"])
        base.mkdir(parents=True, exist_ok=True)
        self.root = Path(tempfile.mkdtemp(prefix=self._testMethodName + "-", dir=base))
        (self.root / "attempts").mkdir()
        self.policy = owner.read(os.environ["BUILDOPT_REPLAY_OWNER_POLICY"])
        self.policy["comparator"] = binding(Path(__file__).with_name("owner_compare.py").resolve())
        self.policy_binding = save(self.root / "policy.json", self.policy)
        print("retained owner fixture:", self.root, flush=True)

    def arm(self, arm, ordinal=0, outcome="EXECUTED", control_adapted=False):
        directory = self.root / "attempts" / f"r1-{ordinal:03d}-{arm}"
        directory.mkdir()
        (directory / "outputs").mkdir()
        root = str(self.root / "r1" / arm / "repo")
        task = {"identity": ":server:checkstyleMain", "path": ":server:checkstyleMain", "taskClass": "org.gradle.api.plugins.quality.Checkstyle_Decorated", "dependencies": [],
                "outputs": [root + "/server/build/reports/checkstyle/main.xml"],
                "checkstyle": {"source": [root + "/src/Main.java"], "classpath": [], "reports": [{"name": "xml", "required": True, "path": root + "/server/build/reports/checkstyle/main.xml"}]}}
        config = {"identity": owner.EXTRA_OWNER, "path": owner.EXTRA_OWNER, "taskClass": "org.gradle.api.tasks.Copy_Decorated", "dependencies": [],
                  "outputs": [root + "/server/build/checkstyle/checkstyle.xml"] + ([root + "/" + owner.EXTRA] if arm == "I" or control_adapted else []), "checkstyle": {"source": [], "classpath": [], "reports": []}}
        tasks = [config, task]
        results = [{"identity": t["identity"], "outcome": outcome, "action": outcome == "EXECUTED"} for t in tasks]
        start = f"2026-09-08T12:{ordinal:02d}:00Z"
        end = f"2026-09-08T12:{ordinal:02d}:10Z"
        events = [{"kind": "graph", "root": root, "buildPath": ":", "tasks": tasks, "task": {"identity": "", "outcome": "", "action": False}, "utc": start, "startedUTC": ""}]
        events += [{"kind": "task", "root": root, "buildPath": ":", "tasks": [], "task": result, "utc": end, "startedUTC": start} for result in results]
        (directory / "graph.jsonl").write_text("".join(json.dumps(event) + "\n" for event in events))
        raw = {"server/build/reports/checkstyle/main.xml": (f'<checkstyle version="10.26.1"><file name="{root}/src/Main.java"></file></checkstyle>\n'.encode(), ":server:checkstyleMain"),
               "server/build/checkstyle/checkstyle.xml": (b'<module name="Checker">\n</module>\n', owner.EXTRA_OWNER)}
        if arm == "I" or control_adapted:
            raw[owner.EXTRA] = (raw["server/build/checkstyle/checkstyle.xml"][0].replace(b'<module name="Checker">', owner.REPLACEMENT.encode()), owner.EXTRA_OWNER)
        outputs = []
        for index, (path, (data, producer)) in enumerate(sorted(raw.items())):
            destination = directory / "outputs" / str(index)
            destination.write_bytes(data)
            b = binding(destination)
            outputs.append({"entry": {"path": path, "kind": "file", "mode": 420, "size": len(data), "sha256": b["sha256"], "target": ""}, "producer": producer, "transform": "exact", "raw": b, "projectionSHA256": b["sha256"]})
        capture = {"schema": "buildopt.history-replay/record/v2", "outputs": outputs, "tasks": results, "graph": [binding(directory / "graph.jsonl")], "daemonLogs": [], "gradleBuilds": [], "error": ""}
        end_record = {"id": directory.name, "capture": save(directory / "capture.json", capture), "native": save(directory / "native.json", {"start": {"utc": start}, "end": {"utc": end}, "exitCode": 0}),
                      "after": save(directory / "state.json", {"root": str(Path(root).parent), "ordinal": ordinal, "arm": arm, "replication": 1}), "candidateApplied": arm == "I", "class": "COMPARABLE"}
        save(directory / "start.json", {"ordinal": ordinal, "arm": arm, "replication": 1, "manifestSHA256": "fixture-manifest"})
        save(directory / "end.json", end_record)
        return end_record

    def pair(self, ordinal=0, outcome="EXECUTED"):
        return {"policy": self.policy_binding, "runRoot": str(self.root), "native": self.arm("N", ordinal, outcome), "candidate": self.arm("I", ordinal, outcome)}

    def mutate(self, end, path, change):
        capture = owner.read(end["capture"]["path"])
        output = next(o for o in capture["outputs"] if o["entry"]["path"] == path)
        raw = Path(output["raw"]["path"])
        raw.write_bytes(change(raw.read_bytes()))
        output["raw"] = binding(raw)
        output["entry"].update(size=raw.stat().st_size, sha256=output["raw"]["sha256"])
        output["projectionSHA256"] = output["raw"]["sha256"]
        end["capture"] = save(Path(end["capture"]["path"]), capture)
        save(Path(end["capture"]["path"]).parent / "end.json", end)

    def add_real_metadata(self, request):
        retained_request = owner.read(Path(os.environ["BUILDOPT_REPLAY_OWNER_POLICY"]).parent / "request.json")
        originals = [owner.read(retained_request[arm]["capture"]["path"]) for arm in ("native", "candidate")]
        indexed = [{o["entry"]["path"]: o for o in capture["outputs"]} for capture in originals]
        choices = [path for path, o in indexed[0].items() if path.endswith("/previous-compilation-data.bin") and o["entry"]["kind"] == "file" and o["entry"]["sha256"] != indexed[1][path]["entry"]["sha256"]]
        original_path = min(choices, key=lambda path: indexed[0][path]["entry"]["size"])
        path, identity = "server/build/tmp/compileFixture/previous-compilation-data.bin", ":server:compileFixture"
        for index, arm in enumerate(("native", "candidate")):
            end = request[arm]
            directory = Path(end["capture"]["path"]).parent
            raw = directory / "outputs" / "compiler-state"
            raw.write_bytes(Path(indexed[index][original_path]["raw"]["path"]).read_bytes())
            b = binding(raw)
            capture = owner.read(end["capture"]["path"])
            capture["outputs"].append({"entry": {"path": path, "kind": "file", "mode": 420, "size": raw.stat().st_size, "sha256": b["sha256"], "target": ""}, "producer": identity, "transform": "exact", "raw": b, "projectionSHA256": b["sha256"]})
            graph_path = Path(capture["graph"][0]["path"])
            events = [owner.parse(line) for line in graph_path.read_text().splitlines()]
            events[0]["tasks"].append({"identity": identity, "path": identity, "taskClass": "org.gradle.api.tasks.compile.JavaCompile_Decorated", "dependencies": [], "outputs": [events[0]["root"] + "/" + path], "checkstyle": {"source": [], "classpath": [], "reports": []}})
            result = {"identity": identity, "outcome": "EXECUTED", "action": True}
            events.append({**events[-1], "task": result})
            graph_path.write_text("".join(json.dumps(event) + "\n" for event in events))
            capture["tasks"].append(result)
            capture["graph"] = [binding(graph_path)]
            end["capture"] = save(Path(end["capture"]["path"]), capture)
            save(directory / "end.json", end)
        return path

    def test_equivalent_complete_reports_and_exact_config(self):
        result = owner.compare(self.pair())
        self.assertEqual(result["counts"], {"exact": 1, "checkstyle": 1, "causalOrigins": 0, "candidateOnly": 1})
        self.assertEqual(result["metadataJVMStarts"], 0)

    def adapted_pair(self):
        return {"policy": self.policy_binding, "runRoot": str(self.root),
                "native": self.arm("N", control_adapted=True), "candidate": self.arm("I"), "controlAdapted": True}

    def test_adapted_control_complete_reports_and_config(self):
        result = owner.compare(self.adapted_pair())
        self.assertEqual(result["counts"], {"exact": 2, "checkstyle": 1, "causalOrigins": 0, "candidateOnly": 0})
        self.assertEqual(result["metadataJVMStarts"], 0)

    def test_adapted_control_requires_explicit_declaration(self):
        request = self.adapted_pair()
        del request["controlAdapted"]
        with self.assertRaisesRegex(ValueError, "native declares candidate-only output"):
            owner.compare(request)

    def test_adapted_control_config_corruption_is_rejected(self):
        request = self.adapted_pair()
        self.mutate(request["native"], owner.EXTRA, lambda raw: raw+b"tampered")
        with self.assertRaisesRegex(ValueError, "generated candidate config differs"):
            owner.compare(request)

    def test_adapted_control_missing_config_is_rejected(self):
        request = self.pair()
        request["controlAdapted"] = True
        with self.assertRaisesRegex(ValueError, "candidate config declaration missing"):
            owner.compare(request)

    def test_adapted_control_keeps_full_finding_checks(self):
        request = self.adapted_pair()
        self.mutate(request["native"], "server/build/reports/checkstyle/main.xml", lambda raw: raw.replace(b"</file>", b'<error line="1" message="violation"/></file>'))
        with self.assertRaisesRegex(ValueError, "non-permitted projected difference"):
            owner.compare(request)

    def test_finding_is_not_normalized_away(self):
        request = self.pair()
        self.mutate(request["candidate"], "server/build/reports/checkstyle/main.xml", lambda raw: raw.replace(b"</file>", b'<error line="1" message="violation"/></file>'))
        with self.assertRaisesRegex(ValueError, "non-permitted projected difference"):
            owner.compare(request)

    def test_malformed_report_is_rejected(self):
        request = self.pair()
        self.mutate(request["candidate"], "server/build/reports/checkstyle/main.xml", lambda raw: raw + b"<")
        with self.assertRaisesRegex(ValueError, "rejected/malformed lexical projection"):
            owner.compare(request)

    def test_duplicate_file_is_rejected(self):
        request = self.pair()
        self.mutate(request["candidate"], "server/build/reports/checkstyle/main.xml", lambda raw: raw.replace(b"</checkstyle>", raw[raw.index(b"<file"):raw.index(b"</checkstyle>")] + b"</checkstyle>"))
        with self.assertRaisesRegex(ValueError, "rejected/malformed lexical projection"):
            owner.compare(request)

    def test_changed_config_is_rejected(self):
        request = self.pair()
        self.mutate(request["candidate"], owner.EXTRA, lambda raw: raw + b" ")
        with self.assertRaisesRegex(ValueError, "generated candidate config differs"):
            owner.compare(request)

    def test_unlisted_exact_difference_is_rejected(self):
        request = self.pair()
        # Both original and generated candidate config must still agree locally.
        self.mutate(request["candidate"], "server/build/checkstyle/checkstyle.xml", lambda raw: raw + b" ")
        self.mutate(request["candidate"], owner.EXTRA, lambda raw: raw + b" ")
        with self.assertRaisesRegex(ValueError, "unapproved output difference"):
            owner.compare(request)

    def test_missing_config_is_rejected(self):
        request = self.pair()
        end = request["candidate"]
        capture = owner.read(end["capture"]["path"])
        capture["outputs"] = [o for o in capture["outputs"] if o["entry"]["path"] != owner.EXTRA]
        end["capture"] = save(Path(end["capture"]["path"]), capture)
        with self.assertRaisesRegex(ValueError, "candidate config declaration missing"):
            owner.compare(request)

    def test_mode_difference_is_rejected(self):
        request = self.pair()
        end = request["candidate"]
        capture = owner.read(end["capture"]["path"])
        next(o for o in capture["outputs"] if o["entry"]["path"] == owner.EXTRA)["entry"]["mode"] = 493
        end["capture"] = save(Path(end["capture"]["path"]), capture)
        with self.assertRaisesRegex(ValueError, "generated config mode differs"):
            owner.compare(request)

    def test_causal_up_to_date_and_cache_restore(self):
        self.pair()
        for ordinal, outcome in [(1, "UP-TO-DATE"), (2, "FROM-CACHE")]:
            result = owner.compare(self.pair(ordinal, outcome))
            self.assertEqual(result["counts"]["causalOrigins"], 2)

    def test_missing_origin_is_rejected(self):
        with self.assertRaisesRegex(ValueError, "no earlier executed origin"):
            owner.compare(self.pair(1, "UP-TO-DATE"))

    def test_changed_reused_output_is_rejected(self):
        self.pair()
        request = self.pair(1, "FROM-CACHE")
        self.mutate(request["candidate"], "server/build/reports/checkstyle/main.xml", lambda raw: raw + b" ")
        with self.assertRaisesRegex(ValueError, "no earlier executed origin"):
            owner.compare(request)

    def test_changed_origin_raw_is_rejected(self):
        origin = self.pair()
        request = self.pair(1, "UP-TO-DATE")
        capture = owner.read(origin["native"]["capture"]["path"])
        output = next(o for o in capture["outputs"] if o["entry"]["path"] == "server/build/reports/checkstyle/main.xml")
        Path(output["raw"]["path"]).write_bytes(b"unrecorded origin change")
        with self.assertRaisesRegex(ValueError, "bound input drift"):
            owner.compare(request)

    def test_future_origin_is_rejected(self):
        self.pair(2)
        with self.assertRaisesRegex(ValueError, "no earlier executed origin"):
            owner.compare(self.pair(1, "UP-TO-DATE"))

    def test_other_arm_origin_is_rejected(self):
        self.arm("I")
        with self.assertRaisesRegex(ValueError, "no earlier executed origin"):
            owner.compare(self.pair(1, "UP-TO-DATE"))

    def test_other_manifest_origin_is_rejected(self):
        self.pair()
        for path in (self.root / "attempts").glob("*/start.json"):
            start = owner.read(path)
            start["manifestSHA256"] = "other-manifest"
            save(path, start)
        with self.assertRaisesRegex(ValueError, "no earlier executed origin"):
            owner.compare(self.pair(1, "UP-TO-DATE"))

    def test_unqualified_skip_is_rejected(self):
        self.pair()
        with self.assertRaisesRegex(ValueError, "producer did not execute or reuse"):
            owner.compare(self.pair(1, "SKIPPED"))

    def test_raw_hash_drift_is_rejected(self):
        request = self.pair()
        capture = owner.read(request["candidate"]["capture"]["path"])
        Path(capture["outputs"][0]["raw"]["path"]).write_bytes(b"unrecorded change")
        with self.assertRaisesRegex(ValueError, "bound input drift"):
            owner.compare(request)

    def test_real_metadata_order_difference_is_equivalent(self):
        request = self.pair()
        self.add_real_metadata(request)
        result = owner.compare(request)
        self.assertEqual(result["counts"]["compilationMetadata"], 1)
        self.assertEqual(result["metadataJVMStarts"], 1)

    def test_real_metadata_trailing_bytes_are_rejected(self):
        request = self.pair()
        path = self.add_real_metadata(request)
        self.mutate(request["candidate"], path, lambda raw: raw + b"\x00")
        with self.assertRaisesRegex(ValueError, "complete compiler-state projector failed"):
            owner.compare(request)

    def test_metadata_requires_java_compile_producer(self):
        request = self.pair()
        self.add_real_metadata(request)
        for arm in ("native", "candidate"):
            end = request[arm]
            capture = owner.read(end["capture"]["path"])
            path = Path(capture["graph"][0]["path"])
            events = [owner.parse(line) for line in path.read_text().splitlines()]
            events[0]["tasks"][-1]["taskClass"] = "unqualified.Task"
            path.write_text("".join(json.dumps(event) + "\n" for event in events))
            capture["graph"] = [binding(path)]
            end["capture"] = save(Path(end["capture"]["path"]), capture)
        with self.assertRaisesRegex(ValueError, "unqualified compiler-state producer"):
            owner.compare(request)


if __name__ == "__main__":
    unittest.main(verbosity=2)
