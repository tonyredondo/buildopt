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
    if len(sys.argv) != 2 or sys.argv[1] not in ("--unit", "--integration", "--disk-unit", "--disk-integration"):
        raise SystemExit("usage: frozen runner check --unit|--integration|--disk-unit|--disk-integration")
    disk = sys.argv[1].startswith("--disk-")
    integration = sys.argv[1].endswith("integration")
    root = Path(__file__).resolve().parent.parent
    evidence_name = "bo-06-disk-accounting" if disk else "bo-06-quiet-start-integration"
    evidence = root / "benchmarks/results/buildopt-product-viability-v1" / evidence_name
    manifest = json.loads((evidence / "evidence-manifest.json").read_text())
    for name, expected in manifest["files"].items():
        path = Path(name)
        if path.is_absolute() or ".." in path.parts:
            raise ValueError(f"evidence path leaves the result directory: {name}")
        if hashlib.sha256((evidence / path).read_bytes()).hexdigest() != expected:
            raise ValueError(f"retained evidence differs: {name}")
    if disk:
        subprocess.run([sys.executable, "-B", "-I", str(evidence / "verify-evidence.py")], check=True, timeout=60)
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
        if integration:
            # Two (quiet) or seven (disk + quiet) possible native fixture
            # starts; no Gradle. Requires Linux amd64 and user systemd.
            pattern = "^(TestQuietRunner|TestLiveDiskRunnerIntegration)" if disk else "^TestQuietRunner"
            args += ["-tags=replay_integration", "-run=" + pattern, "-timeout=" + ("700s" if disk else "610s")]
            limit = 720 if disk else 630
        else:
            args += ["-timeout=160s"]
            limit = 180
        args += ["-v", "./" + str(source.relative_to(root))]
        print(f"Frozen runner {frozen['archiveSHA256']}: {sys.argv[1]}", flush=True)
        result = subprocess.run(["timeout", "--signal=TERM", "--kill-after=10s", str(limit) + "s", *args], cwd=root)
        if result.returncode:
            raise SystemExit(result.returncode)
        print(f"BO-06 {evidence_name} checks passed; no owner timing was performed.")


if __name__ == "__main__":
    main()
