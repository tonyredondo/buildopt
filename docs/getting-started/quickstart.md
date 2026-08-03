# Quickstart

For the shortest product path, use
[product onboarding](./product-onboarding.md): install a published package and
run `buildopt gradle build` in an existing Gradle repository. No source
checkout, Go toolchain, Java compilation, plugin path or service is required.

This page is the source-based evaluation path for contributors and maintainers
who want to prove several components together from the current checkout.

## Source-based product lab

The owner-operated synthetic lab exercises:

1. a real Gradle build and known deliverable;
2. repeated Shared-cache fault handling with tamper-evident evidence;
3. two Edge nodes resolving a collision through Shared;
4. Edge startup, signed authority reload, status, packaging and shutdown.

It does not run the deferred eight-hour soak, require an external design
partner or authorize production deployment.

## Prerequisites

Use Linux AMD64 for this complete source lane. From the repository root:

```bash
./dev/doctor
./dev/bootstrap --toolchain temurin-jdk-21
./dev/bootstrap --toolchain go
```

The repository downloads exact toolchains into ignored `.tools/` state. It
does not replace global tools or use `sudo`.

## Run the lab

```bash
./dev/run-owner-poc-lab --output /tmp/buildopt-poc-lab-result.json
jq '{status, sourceRevision, sourceTreeClean, steps}' \
  /tmp/buildopt-poc-lab-result.json
```

`"status": "PASS"` means every bounded lab step completed against the recorded
Git revision. A failure remains visible in the JSON and returns a non-zero exit
code.

## Build only the launcher

```bash
mkdir -p .tools/bin
./dev/run --toolchain go -- \
  go build -mod=readonly -o .tools/bin/buildopt ./cmd/buildopt
.tools/bin/buildopt doctor
```

A source-built launcher does not have packaged Gradle assets. Either use a
published installation for `buildopt gradle`, or explicitly provide the init
script and plugin paths while developing those assets.

## Bypass

For an installed package:

```bash
BUILDOPT_BYPASS=1 buildopt gradle build
```

For a raw command:

```bash
BUILDOPT_BYPASS=1 buildopt run -- ./gradlew build
```

Only the exact value `1` activates bypass. The original process output and exit
status remain authoritative.

## Cleanup

```bash
./dev/uninstall-toolchains --toolchain temurin-jdk-21
./dev/uninstall-toolchains --toolchain go
rm -f /tmp/buildopt-poc-lab-result.json
```

For product uninstall, run the `uninstall.sh` or `uninstall.ps1` contained in
the downloaded package. Receipts restrict removal to files owned by BuildOpt.

## Next steps

- [Product onboarding](./product-onboarding.md) for installation, first build,
  CI and component rollout.
- [Product workflows](../guides/product-workflows.md) for history, runtime
  policy, Patch Autopilot, Build Impact and Edge.
- [Troubleshooting](../troubleshooting.md) for installation or runtime errors.
