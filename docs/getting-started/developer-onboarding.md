# Developer onboarding

This guide takes a contributor from a clean checkout to a focused, reproducible
change. Repository content, code, comments, examples, commits, and pull requests
are written in English.

## 1. Understand the boundaries

Before changing code, read:

1. [CONTRIBUTING.md](../../CONTRIBUTING.md) for repository rules;
2. the [architecture overview](../architecture/overview.md) for system
   boundaries;
3. the [repository map](../architecture/repository-map.md) to find the owning
   layer;
4. the closest specification, contract, ADR, or tracker item for the behavior.

Build Optimization must not select, shard, retry, prioritize, or otherwise
implement Test Optimization behavior. Unknown evidence, unsupported Gradle
combinations, invalid authority, and incomplete configuration must retain the
documented conservative fallback.

## 2. Inspect the host

From the repository root:

```bash
./dev/doctor
./dev/doctor --json | jq '.summary, .checks[] | select(.status != "PASS")'
```

The first command is human-readable. The JSON form is useful in issue reports.
The doctor does not install or modify anything.

## 3. Provision repository-owned tools

The common Go and JVM development path needs:

```bash
./dev/bootstrap --toolchain temurin-jdk-21
./dev/bootstrap --toolchain go
```

Add tools only when the owning check requires them:

```bash
./dev/bootstrap --toolchain protoc
./dev/bootstrap --toolchain buf
./dev/bootstrap --toolchain shellcheck
./dev/bootstrap --toolchain actionlint
./dev/bootstrap --toolchain cosign
./dev/bootstrap --toolchain syft
```

Tool versions, immutable URLs, checksums, and adoption status live in
[`dev/toolchains.lock.yaml`](../../dev/toolchains.lock.yaml). `dev/run` exposes
one verified toolchain only to its child process and keeps module/build caches
inside `.tools/`.

## 4. Build the code

Build Go packages and binaries:

```bash
./dev/run --toolchain go -- go test ./...
./dev/run --toolchain go -- go build ./cmd/...
```

Build JVM artifacts as Java 17-compatible bytecode:

```bash
./dev/run -- ./gradlew --no-daemon assemble
```

The optional Rust helper has an independent pinned toolchain and is not needed
for core development. See [its README](../../rust/hermetic-helper/README.md).

## 5. Make a focused change

Follow the dependency direction for the type of change:

```text
decision or invariant
  -> contract/specification
  -> producer and consumers
  -> focused tests and fixture
  -> composition check
  -> explanatory documentation
  -> tracker evidence
```

For an implementation-only correction that does not change a public contract,
start in the owning package and add the smallest reproduction. Do not revise an
RFC or schema merely because the implementation is inconvenient.

Keep entrypoints under `cmd/` thin. Reusable Go code belongs under `internal/`.
Gradle behavior belongs in `jvm/gradle-plugin/`; exact repository transforms
belong in `jvm/patcher/`. Cross-process shapes are defined in `contracts/`
before clients or handlers copy them.

## 6. Validate at the owning layer

Examples:

```bash
# Go launcher change
./dev/run --toolchain go -- go test -count=1 ./internal/launcher ./cmd/buildopt
./dev/check-buildopt-cli

# Gradle plugin change
./dev/run -- ./gradlew --no-daemon :jvm:gradle-plugin:check
./dev/check-gradle-plugin-handshake

# Shared storage change
./dev/run --toolchain go -- go test -count=1 ./internal/sharedcache
./dev/check-shared-storage

# Build Impact change
./dev/run --toolchain go -- go test -count=1 ./internal/buildimpact ./cmd/buildopt-impact
./dev/check-build-impact-automatic

# Documentation or repository navigation change
./dev/check-documentation
```

Use [the validation reference](../reference/validation.md) to find other gates.
Run `./dev/check-base-ci --static` for a fast workflow/lock inspection. The full
base lane is intentionally broader and should be reserved for shared-contract
or release-level changes.

## 7. Generated code

Never edit files listed in [`GENERATED_CODE.md`](../../GENERATED_CODE.md)
directly. Change their source schema or IDL, then run the manifest-owned
generator. For example:

```bash
./dev/generate-code --artifact openapi-go-client-v1
./dev/check-generated-code
./dev/check-generated-clients
```

Inspect source and generated diffs together. A changed snapshot, descriptor,
or vector needs a semantic explanation; it is not accepted simply because a
generator produced it.

## 8. Documentation and code comments

Add comments for package boundaries, public/exported behavior, side effects,
security assumptions, lifecycle, failure semantics, and non-obvious platform
differences. Avoid comments that merely repeat syntax.

Update the closest guide when a command, option, default, configuration group,
folder ownership, or user-visible fallback changes. Update the architecture
documents when responsibility or dependency direction changes. Run:

```bash
./dev/check-documentation
```

## 9. Final review

Before committing:

```bash
git status --short
git diff --check
git diff --stat
```

Verify that every changed line belongs to the objective, generated output is
paired with its source, the smallest useful tests passed after the final edit,
and no `.tools/`, runtime state, credentials, logs, or customer source entered
the diff.

## Platform notes

- Linux AMD64 is the complete repository-local development and synthetic-lab
  lane.
- macOS packages four native binaries and launchd user-agent helpers under
  `packaging/macos/`.
- Windows packages five executables, including the SCM wrapper, under
  `packaging/windows/`.
- Native behavior is verified by `.github/workflows/platform-ci.yml`; a Linux
  cross-build alone is not proof of macOS or Windows lifecycle behavior.

Run `./dev/check-platform-compatibility` after touching portable file, process,
storage, CLI, packaging, or service code.
