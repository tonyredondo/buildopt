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
BuildOpt launcher -----> optional verification and cache gateway
        |                            |
        v                            v
      Gradle              optional Shared or Edge Cache
```

The launcher preserves the command, output, and exit status. BuildOpt records
evidence around that execution and uses conservative fallbacks: a rejected
cache entry becomes a normal cache miss, an unqualified optimization is not
applied, and `BUILDOPT_BYPASS=1` removes the optimization path immediately.

> **Project status:** this is an owner-operated proof of concept. The combined
> public path has beaten a well-configured native Gradle baseline across four
> qualified synthetic Kotlin/Groovy workload cells, so the bounded decision is
> `CONTINUE`. After removing one attributable benchmark-environment asymmetry,
> the repeated realistic five-project matrix qualified 4/8 change/DSL cells,
> up from 2/8. The claim has still not been broadened. This does not prove
> universal savings or production readiness.
> Soak, design partners, HA, enterprise identity, multi-tenancy, and production
> operations remain outside this phase. Test Optimization is a separate product
> and is not implemented here.

## Get your first result

Install a published package; you do not need this source checkout or a Go
toolchain. On Linux or macOS:

```bash
curl --fail --silent --show-error --location \
  --output buildopt-install.sh \
  https://raw.githubusercontent.com/tonyredondo/buildopt/main/install.sh
bash buildopt-install.sh
export PATH="$HOME/.local/bin:$PATH"
buildopt doctor
```

Then open any Gradle repository that has its Wrapper:

```bash
cd /path/to/your-gradle-repository
buildopt gradle clean build
buildopt gradle clean build
```

BuildOpt discovers the Wrapper and packaged Gradle integration automatically.
The default path enables Gradle's native local Build Cache; the second clean
build can therefore restore compatible outputs without a plugin path, service,
credential, or `--build-cache` flag. BuildOpt's stricter Safe Cache remains an
explicit POC experiment because it has not demonstrated incremental build-time
value over that native baseline.
The [product onboarding guide](./docs/getting-started/product-onboarding.md)
contains Windows installation, CI snippets, component ownership and the
recommended rollout order. Contributors who want the complete synthetic lab
can use the [source quickstart](./docs/getting-started/quickstart.md).

The cache-compatible command is the zero-configuration starting point. The
measured accelerator is Build Impact: after committing a reviewed manifest and
generated graph, run the explicit POC candidate with `buildopt impact`. It
selects only a repository-authorized alternative and restores the full graph
for unknown/global changes or `BUILDOPT_BYPASS=1`. On the pinned Spring
Framework workload, direct discovery saved 28.72%; the installed command still
saved 15.76% after package, launcher, manifest and graph-validation overhead,
with 8/8 positive pairs and identical declared outputs. See the
[Build Impact workflow](./docs/guides/product-workflows.md#build-impact).

The checked scorecard measures each optimization separately and then measures
the complete public path without adding unrelated percentages. Safe Cache and
the tested Runtime Tuning profiles did not add defensible value over optimized
native Gradle, so neither is active on the default path. The final combined
path saved 63.5–84.1% across four Kotlin/Groovy synthetic workload cells, with
identical required outputs and zero product-attributable failures. The POC
decision is therefore `CONTINUE`, qualified only for those controlled workload
classes. The repeated realistic breadth gate retained that narrow claim after
4/8 cells qualified. A substantial Spring `testClasses` workload then saved
28.72% across 8/8 pairs, but its unchanged OpenTelemetry transfer stopped when
the optimized native control failed in pair 7; the six positive completed
pairs were not promoted. It is not a universal-savings or
production-readiness claim. See the [POC value
contract](./specs/poc-value-validation-v1.md) and
[raw scorecard](./benchmarks/README.md#build-optimization-scorecard).

## Choose what to do next

| Goal | Start here |
|---|---|
| Install and run BuildOpt | [Product onboarding](./docs/getting-started/product-onboarding.md) |
| Run the source-based POC lab | [Quickstart](./docs/getting-started/quickstart.md) |
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
BUILDOPT_BYPASS=1 buildopt gradle build
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
