# BuildOpt

BuildOpt makes Gradle builds faster without changing their expected outputs.
It runs the existing Gradle command, observes what happened, and activates
only optimizations that have enough evidence. If an optimization is unavailable
or cannot be proven safe, the original build remains authoritative.

> **New here?** You do not need to read the implementation tracker, contracts,
> or architecture documents first. On Linux, follow **Get your first result**
> below. It uses a synthetic repository, needs no external service, and does
> not modify one of your projects.

## How it fits into a build

```text
your Gradle command
        |
        v
BuildOpt launcher -----> local verification and cache gateway
        |                            |
        v                            v
      Gradle              optional Shared or Edge Cache
```

The launcher preserves the command, output, and exit status. BuildOpt records
evidence around that execution and uses conservative fallbacks: a rejected
cache entry becomes a normal cache miss, an unqualified optimization is not
applied, and `BUILDOPT_BYPASS=1` removes the optimization path immediately.

> **Project status:** this repository contains a functionally complete,
> owner-operated proof of concept. It is not a production-ready hosted service.
> Long-duration soak testing, external validation, multi-tenant identity, high
> availability, and production operations are intentionally deferred. Test
> Optimization is a separate product and is not implemented here.

## Get your first result

The recommended first run exercises the complete POC against project-owned
synthetic repositories. From a Linux AMD64 checkout:

```bash
./dev/doctor
./dev/bootstrap --toolchain temurin-jdk-21
./dev/bootstrap --toolchain go
./dev/run-owner-poc-lab --output /tmp/buildopt-poc-lab-result.json
jq '{status, sourceRevision, steps}' /tmp/buildopt-poc-lab-result.json
```

You are looking for `"status": "PASS"`. That result proves the checked-out
revision completed a real Gradle build, Shared-cache fault handling, a two-node
Edge scenario, and the packaged Edge lifecycle. It does not run the deferred
eight-hour soak or claim a speedup for one of your own repositories.

The [step-by-step quickstart](./docs/getting-started/quickstart.md) explains
prerequisites, expected output, a shorter launcher-only test, running BuildOpt
around another Gradle repository, and cleanup.

## Choose what to do next

| Goal | Start here |
|---|---|
| See the POC work | [Quickstart](./docs/getting-started/quickstart.md) |
| Try BuildOpt on a Gradle repository | [Run around another repository](./docs/getting-started/quickstart.md#run-buildopt-around-another-gradle-repository) |
| Add it to GitHub Actions or GitLab CI | [CI integration](./docs/guides/ci-integration.md) |
| Understand the design | [Architecture overview](./docs/architecture/overview.md) |
| Make a code change | [Developer onboarding](./docs/getting-started/developer-onboarding.md) |
| Find a command or setting | [CLI](./docs/reference/cli.md) and [configuration](./docs/reference/configuration.md) references |
| Diagnose a failure | [Troubleshooting](./docs/troubleshooting.md) |

## What BuildOpt changes

| Capability | What it does | Safe fallback |
|---|---|---|
| Launcher | Runs the original argv without a shell and preserves its exit code | Original command |
| Managed L1 and Shared Cache | Reuses only authenticated, verified, committed Gradle outputs | Cache miss and normal execution |
| Runtime Optimizer | Applies bounded resource, invocation, Configuration Cache, and lifecycle policies | Baseline policy |
| Task Intelligence | Qualifies only tasks with sufficient exact evidence | No publication or optimization |
| Patch Autopilot | Creates exact, signed, reviewable draft changes and exact revert bundles | Repository remains unchanged |
| Build Impact | Chooses only repository-authorized Gradle entrypoint alternatives | Full original graph |
| Build history | Exposes redacted immutable sessions through a loopback API and embedded dashboard | No history endpoint |
| Edge Cache | Provides an optional nearby cache while Shared remains commit authority | Shared or ordinary miss |

Set `BUILDOPT_BYPASS=1` to remove the optimization path immediately while
preserving the original command and process behavior:

```bash
BUILDOPT_BYPASS=1 buildopt run -- ./gradlew build
```

## Documentation

Start at the [documentation portal](./docs/README.md). Common routes are:

- [Quickstart](./docs/getting-started/quickstart.md) — obtain a first result.
- [Developer onboarding](./docs/getting-started/developer-onboarding.md) — set
  up a reproducible development environment and make a first change.
- [Architecture](./docs/architecture/overview.md) — components, data flow,
  trust boundaries, and failure behavior.
- [Repository map](./docs/architecture/repository-map.md) — the architecture's
  exact correspondence with folders and artifacts.
- [Product workflows](./docs/guides/product-workflows.md) — history, runtime
  optimization, Patch Autopilot, Build Impact, and Edge.
- [CI integration](./docs/guides/ci-integration.md) — GitHub Actions and GitLab
  CI.
- [CLI reference](./docs/reference/cli.md) and
  [configuration reference](./docs/reference/configuration.md).
- [Validation guide](./docs/reference/validation.md) — choose the smallest
  useful proof instead of running every check.
- [Troubleshooting](./docs/troubleshooting.md) and
  [glossary](./docs/glossary.md).

Operator recovery procedures live in [runbooks](./runbooks/README.md).
Executable cross-component behavior lives in [specifications](./specs/README.md).

## Supported environments

| Surface | Linux | macOS | Windows |
|---|---:|---:|---:|
| Native launcher and Build Impact | Yes | Yes | Yes |
| Persistent gateway and managed L1 | Yes | Yes | Yes |
| Server and Edge storage | Yes | Yes | Yes |
| Process-tree cancellation | Process group | Process group | Job Object |
| Background services | systemd | launchd user agent | Windows SCM |
| Source bootstrap and full development lane | Primary | Native CI/package lane | Native CI/package lane |

The Gradle compatibility target and evidence levels are defined in the
[capability matrix](./specs/capability-matrix-v1.md). Run `buildopt doctor` on
an installed binary to see exact platform capabilities as JSON.

## Build and contribute

Repository-owned toolchains avoid dependence on global Go, Java, or lint
versions:

```bash
./dev/doctor
./dev/bootstrap --toolchain temurin-jdk-21
./dev/bootstrap --toolchain go
./dev/run --toolchain go -- go test ./internal/launcher ./cmd/buildopt
./dev/run -- ./gradlew --no-daemon check
```

Read [CONTRIBUTING.md](./CONTRIBUTING.md) before changing contracts, generated
code, tests, or product boundaries. Documentation is validated with:

```bash
./dev/check-documentation
```

## Sources of truth

Use this precedence when documents answer different questions:

1. [Master RFC](./gradle-build-optimization-platform.md) — product intent,
   safety invariants, and accepted decisions.
2. `contracts/` — normative wire and document formats.
3. `specs/` and `adr/` — executable behavior and architectural decisions.
4. [Implementation tracker](./implementation-tracker.md) — current status and
   evidence.
5. `docs/`, component READMEs, and runbooks — explanatory and operational
   guidance.

An example in documentation never overrides an executable contract. If the
implementation, a guide, and a normative contract disagree, stop and reconcile
the contract and its consumers before activating the behavior.
