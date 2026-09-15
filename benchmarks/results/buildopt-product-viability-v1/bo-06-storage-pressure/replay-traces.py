#!/usr/bin/env python3
"""Replay quiet-window decisions with the frozen runner. Linux amd64 only."""

import hashlib
import json
import os
from pathlib import Path
import subprocess
import tarfile
import tempfile

from analyze import EVIDENCE, HERE, analyze


def main():
    analyze()  # Bind every input before compiling the historical runner.
    repo = HERE.parents[3]
    scratch = repo / ".tools" / "state"
    members = json.loads((EVIDENCE / "source-archive-members.json").read_text())
    expected = {row["path"]: row for row in members}
    with tempfile.TemporaryDirectory(prefix="bo06-trace-", dir=scratch) as directory:
        source = Path(directory)
        with tarfile.open(EVIDENCE / "runner-source.tar.gz") as archive:
            assert len(archive.getmembers()) == len(expected)
            assert {m.name for m in archive.getmembers()} == set(expected)
            for member in archive.getmembers():
                path = Path(member.name)
                assert member.isfile() and not path.is_absolute() and ".." not in path.parts
                data = archive.extractfile(member).read()
                assert len(data) == expected[member.name]["bytes"]
                assert hashlib.sha256(data).hexdigest() == expected[member.name]["sha256"]
                target = source / path
                target.parent.mkdir(parents=True, exist_ok=True)
                target.write_bytes(data)
        (source / "recorded_pressure_test.go").write_bytes((HERE / "recorded_pressure_test.go.txt").read_bytes())
        env = dict(os.environ, BUILDOPT_DIAG_EVIDENCE_ROOT=str(EVIDENCE))
        package = "./" + source.relative_to(repo).as_posix()
        for test, expected_exit, message in (
            ("TestRecordedWindowCanAdmit", 1, "recorded refusal reproduced: no quiet window in 180 observations"),
            ("TestRecordedAdmissionDecisions", 0, "25 admitted and one refused trace reproduced"),
        ):
            command = [str(repo / "dev/run"), "--toolchain", "go", "--", "go", "test",
                       "-count=1", "-v", "-run", f"^{test}$", package]
            run = subprocess.run(command, cwd=repo, env=env, text=True,
                                 stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=120)
            print(run.stdout, end="")
            assert run.returncode == expected_exit and message in run.stdout, test
    print("Both trace checks behaved as expected; no owner build or comparator started.")


if __name__ == "__main__":
    main()
