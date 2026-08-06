# Installed Build Impact value experiment

This experiment answers whether the Build Impact value already qualified on
Spring Framework survives the actual installed `buildopt impact` path. It does
not repeat the mechanism search and does not widen production authority.

The repository revision, `spring-jms` source mutation, optimized native Gradle
control, 12-worker runner, required class outputs, common `buildSrc` test,
eight alternating pairs, and 500-ms/2%/positive-lower-bound gate are inherited
unchanged from [`poc-spring-test-preparation-v2`](./poc-spring-test-preparation-v2.json).

The candidate differs in one deliberate way: BuildOpt is built as the native
Linux package, installed into an isolated prefix, and invoked as:

```bash
buildopt impact \
  --repository-id spring-projects/spring-framework \
  --pipeline-class poc-spring-test-preparation \
  --changes-file .buildopt-changes \
  --gradle-option=--daemon \
  --gradle-option=--offline \
  --gradle-option=--build-cache \
  --gradle-option=--parallel \
  --gradle-option=--no-configuration-cache \
  --gradle-option=--console=plain \
  --gradle-option=--max-workers=12 \
  --gradle-option=--no-scan \
  --gradle-option=--stacktrace
```

The wall-clock measurement therefore includes launcher argument parsing,
manifest/graph/generated-binding validation, changed-path evaluation, process
setup, and Gradle execution. The control remains direct optimized native
Gradle. Both arms receive the same offline dependency state and native build
cache seed; no remote cache, Shared Cache, Edge Cache, network access, or Test
Optimization behavior is introduced.

The experiment qualifies only if all eight observations complete, every
required affected output is byte-identical and non-empty, the common build
logic test outcome remains unchanged, no root-build `Test` executes, and the
unchanged value gate passes. A failed pair is terminal and cannot be retried or
discarded.

The first invocation stopped before warmups and before any accepted pair because
the runner read the manifest from the repository-discovery specification rather
than the bound test-preparation protocol, producing JSON `null`. The runner now
reads only the manifest from the bound protocol; source URL and original-file
checksum remain sourced from the unchanged repository specification. This
pre-timing correction changes neither the measurement nor the value gate.

Run the preregistered experiment with:

```bash
./dev/run-poc-spring-installed-impact \
  /absolute/path/poc-spring-installed-impact-v1.json
```

Validate the result with:

```bash
./dev/check-poc-spring-installed-impact \
  /absolute/path/poc-spring-installed-impact-v1.json
```

This is POC evidence only. It does not activate `BIA-002`, require a public
release, claim production readiness, add soak/design-partner work, or modify
Test Optimization.
