#!/usr/bin/env python3
"""Check the frozen BO-06 quiet-start runner without changing older runners."""

import hashlib
import json
from pathlib import Path
import subprocess
import sys
import tarfile
import tempfile


def main():
    if sys.argv[1:] not in (["--unit"], ["--integration"]):
        raise SystemExit("usage: dev/check-quiet-start-integration --unit|--integration")
    root = Path(__file__).resolve().parent.parent
    evidence = root / "benchmarks/results/buildopt-product-viability-v1/bo-06-quiet-start-integration"
    manifest = json.loads((evidence / "evidence-manifest.json").read_text())
    for name, expected in manifest["files"].items():
        path = Path(name)
        if path.is_absolute() or ".." in path.parts:
            raise ValueError(f"evidence path leaves the result directory: {name}")
        if hashlib.sha256((evidence / path).read_bytes()).hexdigest() != expected:
            raise ValueError(f"retained evidence differs: {name}")
    frozen = json.loads((evidence / "source-freeze.json").read_text())

    def checked(name, expected):
        raw = (evidence / name).read_bytes()
        if hashlib.sha256(raw).hexdigest() != expected:
            raise ValueError(f"frozen input differs: {name}")
        return raw

    checked("runner-source.tar.gz", frozen["archiveSHA256"])
    protocol = checked("poc-product-viability-v1.json", frozen["protocolSHA256"])
    parent = root / ".tools/state/quiet-start-verification"
    parent.mkdir(parents=True, exist_ok=True)
    # TemporaryDirectory removes only the directory this invocation creates.
    with tempfile.TemporaryDirectory(prefix="check-", dir=parent) as directory:
        target = Path(directory)
        source = target / "source"
        source.mkdir()
        with tarfile.open(evidence / "runner-source.tar.gz", "r:gz") as archive:
            seen = set()
            for member in archive.getmembers():
                name = member.name
                if not member.isfile() or Path(name).name != name or name in seen or name not in frozen["members"]:
                    raise ValueError(f"unexpected source member: {name}")
                seen.add(name)
                raw = archive.extractfile(member).read()
                if hashlib.sha256(raw).hexdigest() != frozen["members"][name]:
                    raise ValueError(f"source member differs: {name}")
                (source / name).write_bytes(raw)
            if seen != set(frozen["members"]):
                raise ValueError("source archive is incomplete")
        (target / "specs").mkdir()
        (target / "specs/poc-product-viability-v1.json").write_bytes(protocol)
        args = [str(root / "dev/run"), "--toolchain", "go", "--", "go", "test", "-mod=readonly", "-count=1"]
        if sys.argv[1] == "--integration":
            # Exactly two possible native fixture starts, no Gradle or owner
            # builds. Requires Linux amd64 and a working user systemd manager.
            args += ["-tags=replay_integration", "-run=^TestQuietRunner", "-timeout=610s"]
            limit = 630
        else:
            args += ["-timeout=160s"]
            limit = 180
        args += ["-v", "./" + str(source.relative_to(root))]
        print(f"Frozen runner {frozen['archiveSHA256']}: {sys.argv[1]}", flush=True)
        result = subprocess.run(["timeout", "--signal=TERM", "--kill-after=10s", str(limit) + "s", *args], cwd=root)
        if result.returncode:
            raise SystemExit(result.returncode)
        print("BO-06 quiet-start checks passed; no owner timing was performed.")


if __name__ == "__main__":
    main()
