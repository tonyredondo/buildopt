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
measured accelerator is the qualified POC profile: after committing a reviewed
Build Impact manifest, generated graph and `buildopt-qualified-profile.json`,
run `buildopt poc --changes-file .buildopt-changes`. It reports the selected or
full graph, exact adapters and expected outputs before Gradle starts. Only
Build Impact plus the exact standard-`Jar` adapter can activate; unknown/global
changes and `BUILDOPT_BYPASS=1` restore native full-graph execution. On the pinned Spring
Framework workload, direct discovery saved 28.72%; the installed command still
saved 15.76% after package, launcher, manifest and graph-validation overhead,
with 8/8 positive pairs and identical declared outputs. See the
[Build Impact workflow](./docs/guides/product-workflows.md#build-impact).
An additional installed Spring matrix qualified `spring-webmvc` at 13.50%
faster, while a shared `spring-core` to `spring-jms` scope averaged 10.89%
faster but failed the preregistered 4/4-positive stability rule. BuildOpt
therefore keeps a bounded output-scope claim rather than generalizing to every
change.

The latest fresh-family check used Micronaut Core. A generic ownership fix now
separates each project's direct source roots from the larger conservative
boundary retained for cyclic dependencies. Discovery reduced the fixed
75-project `assemble` reach to 22 projects without a repository-name rule. In
eight alternating installed-path pairs, optimized native Gradle averaged
24.067 s and BuildOpt averaged 6.506 s: **17.562 s/72.97% faster**, with 8/8
positive pairs, three byte-identical JARs and full-graph fallback for a global
change. This qualifies only the fixed Micronaut structural scope, not every
repository or change.

`buildopt poc` is available in source-built packages from the current `main`.
The published `v0.2.0` package continues to use the longer `buildopt impact`
form until a later explicitly authorized release.

The checked scorecard measures each optimization separately and then measures
the complete public path without adding unrelated percentages. Safe Cache and
the tested Runtime Tuning profiles did not add defensible value over optimized
native Gradle, so neither is active on the default path. The final combined
path saved 63.5–84.1% across four Kotlin/Groovy synthetic workload cells, with
identical required outputs and zero product-attributable failures. The POC
decision is therefore `CONTINUE`, qualified only for those controlled workload
classes. The repeated realistic breadth gate retained that narrow claim after
4/8 cells qualified. A substantial Spring `testClasses` workload then saved
28.72% across 8/8 pairs. After task attribution and a conservative
standard-`Jar` cache adapter, the clean installed OpenTelemetry Spring-family
path saved 50.40% or 5,361.25 ms across 4/4 pairs, with a positive paired
interval, 125 identical outputs, Hot State disabled, and safe full-graph
fallback. This is qualified POC value for that workload, not a universal-savings
or production-readiness claim. A separate unchanged Spring test workflow
rejected direct Test-fixture JAR reuse after it regressed by 11.31%, so that
diagnostic switch is not part of the recommended path. A subsequent three-arm
ablation narrowed plugin registration but still found the complete adapter
612.25 ms/9.53% slower than native; BuildOpt therefore keeps native Gradle for
that unqualified workflow instead of treating cache hits as value. The next
bounded Runtime Tuning hypothesis also failed: capping the same Spring
`testClasses` workload from 12 to 6 workers made it 191.5 ms/2.00% slower, with
only 2/4 favorable pairs and an interval crossing zero. BuildOpt therefore
retains the native 12-worker control and performs no further parameter search
for that trace. The controlled remote-cache experiment then isolated Edge
locality: the same eight committed Shared objects and Gradle HTTP client took
6,911.25 ms directly over an 80-ms/20-MiB/s modeled WAN and 4,510 ms through a
prewarmed loopback Edge. That is 2,401.25 ms/**34.74% faster**, with 4/4
positive pairs, identical 32-MiB outputs and zero measured upstream Edge
requests. This qualifies Edge locality only under that frozen profile; it does
not claim that Shared alone outperforms another remote-cache origin. Finally,
the unchanged clean profile transferred to Apache Kafka 4.3.1: native root
`testClasses` averaged 4,609.5 ms and installed BuildOpt 2,070 ms, saving
2,539.5 ms/**55.09%** with 4/4 positive pairs, 4,062 byte-identical outputs and
full 64-project fallback. This broadens the POC evidence to a Java/Scala and
generated-source workload without adding Kafka-specific product logic; it is
still not a universal claim. See the [POC value
contract](./specs/poc-value-validation-v1.md) and
[raw scorecard](./benchmarks/README.md#build-optimization-scorecard).

The same repository-owned profile has also been replayed through an installed
package on those fixed OpenTelemetry and Kafka revisions. Both candidates
reproduced their historical outputs; OpenTelemetry restored its standard JAR
and Kafka restored `:generator:jar`. Global changes completed the native full
graph. This is usability and fallback evidence only, so the earlier
percentages remain unchanged.

Fresh packaging evidence now extends the Kafka claim beyond test preparation.
For the fixed `Metadata.java` change, native root `assemble` averaged 8,054 ms
and installed `buildopt poc` averaged 3,416.5 ms, saving 4,637.5 ms/**57.58%**.
All 4/4 pairs were positive, the smallest saving was 4,050 ms, the 10.2-MB
client JAR was byte-identical, and the global fallback passed. This qualifies
only the declared Kafka client-packaging scope. A subsequent composition seed
proved that the required shaded artifact is produced by `:clients:shadowJar`
while `:clients:jar` is skipped. It stopped before any timing, so the 57.58%
result is Build Impact evidence and is not attributed to the standard-`Jar`
adapter. The first Build Impact + Edge composition produced a strong diagnostic
signal but remained unqualified because forced Edge failure rebuilt custom
`shadowJar` with different ZIP metadata. After qualifying reproducible archive
settings independently, a fresh preregistered composition used those settings
equally in both arms. Native full `assemble` through Shared averaged
42,992.75 ms; installed Build Impact through prewarmed Edge averaged
7,587.25 ms, saving 35,405.5 ms/**82.35%** with 4/4 positive pairs and interval
+30,162..+42,487.75 ms. Every arm produced exact `3ffd994e...3349` bytes,
global changes restored the full graph, and HTTP 503 disabled Edge and rebuilt
the same output locally. This qualifies the combined POC only for the fixed
Kafka change and modeled network profile.

The same composition is now available through the repository-owned v2 POC
profile instead of experiment-only cache variables:

```bash
buildopt poc --changes-file .buildopt-changes \
  --edge-url http://127.0.0.1:<PORT>
```

The plan exposes the exact normalized `build.gradle` SHA-256 and read-only
Edge endpoint. Global/unknown changes, precondition drift, missing/invalid
Edge, and bypass select the native full graph without Edge. HTTP 503 executes
the selected graph locally and preserves the exact output. This is usability
evidence; it does not widen or recompute the 82.35% result.

The profile can now be reproduced by the read-only
`buildopt profile discover` command from the checked matrix evidence, Build
Impact manifest/graph/generated state, trace digests, and reviewed profile
contract. The generated Kafka profile is exactly equal to the reviewed v2
profile. Spring and OpenTelemetry emit `NATIVE_FULL_GRAPH`, as do evidence
drift, incomplete graphs, unknown relationships, selected Test tasks, and
precondition drift. Discovery never writes or activates a profile.

The subsequent trace-gated decision found no new generic optimization worth
implementing. Across the retained installed synthetic and Spring traces,
BuildOpt-specific setup peaks at 1.238233 ms, startup at 364.875 ms,
finalization at 97 ms, and teardown at 87 ms. Configuration reaches 682 ms in
Spring but is neither causally attributed to BuildOpt nor reproduced above the
500-ms threshold in a second workload family. The checked result is therefore
`NO_ACTIONABLE_HYPOTHESIS`; no new timing or mechanism activation follows.

The terminal portfolio decision is now
`SPECIALIZE_BOUNDED_KAFKA_PROFILE`. Only Kafka qualified through the installed
path: **28,523.25 ms / 81.85%** saved with 8/8 positive pairs and complete
fallback. Spring and OpenTelemetry remain on optimized native Gradle, the
general accelerator claim is withdrawn, and deterministic discovery stays
read-only and review-required. This is terminal POC evidence, not a production
or universal-value claim.

A new generalization foundation now separates structural opportunity from
activation. `buildopt profile analyze` detects a complete smaller graph without
matching repository names and emits a measurement proposal, never a predicted
speedup. The checked whole-profile scorecard then evaluates every mechanism
supported by exact evidence for each target: Spring Build Impact was **30.86%**
faster, clean OpenTelemetry Impact + standard `Jar` was **50.40%** faster, and
Kafka Impact + read-only Edge was **82.35%** faster. These are direct independent
compositions, not added percentages. Only Kafka also passed the later strict
installed replication; Spring and OpenTelemetry therefore remain native by
default while fresh generalization experiments continue. See the [general
build-value contract](./specs/poc-general-build-value-v1.md).

## Choose what to do next

| Goal | Start here |
|---|---|
| Install and run BuildOpt | [Product onboarding](./docs/getting-started/product-onboarding.md) |
| Run the source-based POC lab | [Quickstart](./docs/getting-started/quickstart.md) |
| Add it to GitHub Actions or GitLab CI | [CI integration](./docs/guides/ci-integration.md) |
| Understand the design | [Architecture overview](./docs/architecture/overview.md) |
| Review measured value and next priorities | [Performance findings](./docs/findings/build-optimization-performance.md) |
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
