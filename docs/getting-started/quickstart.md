# Quickstart

This path produces a trustworthy BuildOpt result from the current source tree
without requiring a private repository, a remote cache, or external services.
It is the recommended first evaluation of the POC.

## What you will prove

The owner-operated synthetic lab exercises four existing product paths:

1. a real Gradle build and known deliverable;
2. repeated Shared-cache fault handling with tamper-evident evidence;
3. two Edge nodes resolving a collision through Shared;
4. Edge process startup, signed authority reload, status, packaging, and
   shutdown.

The output is a private JSON result bound to the checked-out Git revision. The
lab does not run the deferred eight-hour soak, use an external design partner,
or authorize production deployment.

## Prerequisites

Use a Linux AMD64 checkout for the complete source-driven lab. The native
runtime is also supported on macOS and Windows, but the repository-local
bootstrap and full lab are currently the Linux development lane.

Required host commands are Bash, Git, `curl`, `jq`, `tar`, `unzip`,
`sha256sum`, `awk`, `df`, `getconf`, `stat`, `uname`, and `xz`. The repository
downloads exact Go and Java toolchains; it does not replace global tools or use
`sudo`.

From the repository root, inspect the host without changing it:

```bash
./dev/doctor
```

`FAIL` identifies a required host dependency. `WARN` identifies an optional
capability or a project-local toolchain that has not been provisioned yet.

## Run the complete synthetic lab

Provision the two required toolchains. Downloads are checksum-verified and
stored under ignored `.tools/` state:

```bash
./dev/bootstrap --toolchain temurin-jdk-21
./dev/bootstrap --toolchain go
```

Run the lab and keep the machine-readable result outside the checkout:

```bash
./dev/run-owner-poc-lab --output /tmp/buildopt-poc-lab-result.json
```

Inspect the outcome:

```bash
jq '{status, sourceRevision, sourceTreeClean, steps}' \
  /tmp/buildopt-poc-lab-result.json
```

Expected shape:

```json
{
  "status": "PASS",
  "sourceRevision": "<40-character Git SHA>",
  "sourceTreeClean": true,
  "steps": [
    {"id": "O2-001", "status": "PASS", "exitCode": 0},
    {"id": "O2-002", "status": "PASS", "exitCode": 0},
    {"id": "O2-003", "status": "PASS", "exitCode": 0},
    {"id": "O2-004", "status": "PASS", "exitCode": 0}
  ]
}
```

Additional fields include exact commands and start/completion timestamps. A
failed step stops the lab, remains visible in the JSON result, and returns a
non-zero exit code.

## Try the launcher in isolation

For a much shorter smoke test, build only the main CLI:

```bash
mkdir -p .tools/bin
./dev/run --toolchain go -- \
  go build -mod=readonly -o .tools/bin/buildopt ./cmd/buildopt
.tools/bin/buildopt doctor
.tools/bin/buildopt run -- ./gradlew --version
```

`buildopt doctor` prints the runtime's platform capability report as JSON.
`buildopt run --` executes the exact command after `--` without a shell and
returns its exit code. This short path validates the launcher but does not by
itself configure Shared cache, build history, or an optimization policy.

## Run BuildOpt around another Gradle repository

Use an absolute path to the binary or add its directory to `PATH`, change to
the target repository, and preserve that repository's Wrapper:

```bash
cd /path/to/gradle-repository
/path/to/buildopt/.tools/bin/buildopt run -- ./gradlew build
```

This is a safe baseline-compatible invocation. To obtain session exports,
cache reuse, runtime policy, or Build Impact results, configure the
corresponding workflow in [Product workflows](../guides/product-workflows.md)
or [CI integration](../guides/ci-integration.md). Do not copy internal
credentials into the target repository.

## Immediate bypass

If a configured BuildOpt path is unavailable or suspect, run the original
argv through the launcher bypass:

```bash
BUILDOPT_BYPASS=1 buildopt run -- ./gradlew build
```

Only the exact value `1` activates bypass. BuildOpt removes its reserved
configuration from the child, does not start the plugin or gateway, and keeps
the original command's output and exit status authoritative.

## Cleanup

Remove only the two provisioned toolchains while retaining verified downloads
and build state:

```bash
./dev/uninstall-toolchains --toolchain temurin-jdk-21
./dev/uninstall-toolchains --toolchain go
```

The lab uses temporary directories and does not modify tracked source files.
Delete `/tmp/buildopt-poc-lab-result.json` when its evidence is no longer
needed. For broader cleanup choices, read
[Development tools](../../dev/README.md#cleanup-and-uninstall).

## Next steps

- Use the [build history workflow](../guides/product-workflows.md#build-history-and-dashboard)
  to inspect real session exports.
- Add BuildOpt to [GitHub Actions or GitLab CI](../guides/ci-integration.md).
- Read the [architecture overview](../architecture/overview.md) before
  configuring Shared or Edge storage.
- Use [troubleshooting](../troubleshooting.md) if the doctor, bootstrap, or lab
  reports a failure.
