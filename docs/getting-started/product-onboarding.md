# Product onboarding

The default BuildOpt experience has one installation step and one command:

```text
install BuildOpt -> open a Gradle repository -> buildopt gradle build
```

The package contains the launcher, Gradle plugin and init script, Build Impact,
the self-hosted server, Edge Cache, and the JVM agent. Users do not need to
clone this repository, build Go or Java sources, copy JARs, or set internal
plugin paths.

The current public path below separates compatibility, proposal, measurement
and replay explicitly. The target POC experience is the single command
`buildopt optimize build`. Its stable CLI, private state, exact resume, budget,
human/JSON result and exit contract are now implemented. Until automatic
calibration is delivered, the command runs optimized native Gradle and then
automatically produces either a reviewable structural proposal or an exact
native-fallback reason. Use the explicit commands below for measurement and
qualification. Follow the
[one-command onboarding roadmap](../plans/one-command-onboarding-roadmap.md)
for the remaining ordered delivery.

## Install

Published packages currently cover Linux AMD64, macOS Intel and Apple Silicon,
and Windows AMD64. Native runtime source remains portable beyond that matrix,
but the installer fails clearly instead of downloading the wrong architecture.

### Linux and macOS

Download the installer, inspect it if required by your organization, and run
it as your user. It defaults to `~/.local` and never uses `sudo`:

```bash
curl --fail --silent --show-error --location \
  --output buildopt-install.sh \
  https://raw.githubusercontent.com/tonyredondo/buildopt/main/install.sh
bash buildopt-install.sh
export PATH="$HOME/.local/bin:$PATH"
buildopt doctor
```

Use `--version 0.2.0` to pin a release or `--prefix /absolute/path` to choose
another user-owned installation. The installer detects the operating system
and architecture, downloads the matching archive and checksum, verifies the
complete archive, verifies its internal files again, and records only the
files owned by BuildOpt for safe removal.

### Windows

In PowerShell:

```powershell
Invoke-WebRequest `
  https://raw.githubusercontent.com/tonyredondo/buildopt/main/install.ps1 `
  -OutFile buildopt-install.ps1
./buildopt-install.ps1 -UpdatePath
buildopt doctor
```

Open a new terminal after `-UpdatePath`. `-Version 0.2.0` pins a release and
`-Prefix C:\absolute\path` changes the installation root.

## Run the first build

Change to a Gradle repository that contains its Wrapper and run:

```bash
cd /path/to/your-gradle-repository
buildopt gradle clean build
buildopt gradle clean build
```

On Windows the same command discovers `gradlew.bat`; on Linux and macOS it
discovers `gradlew`. BuildOpt enables Gradle's native local Build Cache,
preserves Gradle output and returns the Gradle exit code. The first clean build
populates native entries. The second removes project outputs again and should
show reusable tasks such as `compileJava FROM-CACHE`.

The default uses the same cache semantics as `./gradlew --build-cache`; this is
intentional because the stricter BuildOpt Safe Cache did not prove additional
speed in `POC-VALUE-002`. It remains available only for explicit POC evaluation
with `BUILDOPT_SAFE_CACHE=1`. Any ordinary Gradle argument can follow
`buildopt gradle`:

```bash
buildopt gradle --no-daemon clean assemble
```

Disable only the automatic native build cache while retaining the launcher:

```bash
buildopt gradle --no-build-cache build
```

Immediate bypass keeps the same entrypoint while removing BuildOpt's plugin
and optimization path:

```bash
BUILDOPT_BYPASS=1 buildopt gradle build
```

In PowerShell use `$env:BUILDOPT_BYPASS = '1'` for that invocation.

## Preview the one-command POC

The final entrypoint is already safe to use:

```bash
buildopt optimize build
```

Today it executes the optimized native Gradle baseline and then automatically
derives the exact Git change, Gradle-owned outputs and complete structural
graph for supported build-owned workflows. A safe candidate reports
`LEARNING / STRUCTURAL_CANDIDATE_DISCOVERED`; ambiguity, global changes,
unsupported relationships and full test execution retain native Gradle with an
exact reason. It writes private generated state/evidence under
`.buildopt/optimize/v1`, requires no hand-authored BuildOpt files, performs no
calibration or selection and grants no production authority. Machine-readable
mode reserves stdout for one result document:

```bash
buildopt optimize --json -- build
```

`BUILDOPT_BYPASS=1 buildopt optimize build` skips optimize state and reporting
entirely. The [command contract](../../specs/poc-magic-onboarding-contract-v1.md)
defines resume bindings, budgets, outcomes and exit behavior; the
[automatic-discovery contract](../../specs/poc-magic-auto-discovery-v1.md)
defines derived inputs, evidence and native fallbacks.

## Try the Build Impact accelerator

The default Gradle command establishes compatibility; Build Impact is the POC
feature that has demonstrated incremental value against optimized native
Gradle. Adopt it only after reviewing and committing the repository manifest
and generated graph described in the
[Build Impact workflow](../guides/product-workflows.md#build-impact).

The easiest first step is review-only CI. Commit the owner input described by
the [owner-input contract](../../specs/poc-generic-owner-input-v1.md)
and run the repository-root Action in `profile-proposal` mode. It derives the
exact Git change and uploads a deterministic proposal or an explicit native
fallback. It does not time the candidate or activate a profile, so a team can
inspect BuildOpt's reasoning before trusting it with any execution change.

Before committing a qualified profile, current `main` can create one checked
owner input without hand-authoring Build Impact JSON:

```bash
mkdir -p .buildopt
buildopt profile outputs \
  --repository-id owner/repository \
  --pipeline-class classes \
  --entrypoint classes \
  --required-output 'module/build/classes/**' \
  --output .buildopt/output-contract.json
buildopt profile input \
  --output-contract .buildopt/output-contract.json \
  --confirm \
  --gradle-command ./gradlew \
  --output .buildopt/profile.json
buildopt profile propose \
  --owner-input .buildopt/profile.json \
  --base-revision "$BASE_SHA"
```

Repeat `--entrypoint` for a workflow that invokes several Gradle task paths.
BuildOpt preserves the complete native workflow as fallback and maps the unique
terminal selectors to the exact changed project owners; it does not require a
repository-specific profile or an artificial aggregate task. The owner file
declares `GIT_DIFF_BASE_TO_HEAD`, so local and CI proposal paths use the same
change source. BuildOpt revalidates the output contract on every target and
retains native Gradle with concrete candidates if those outputs drift.

Review `buildopt-profile-proposal.json` first. A
`MEASURE_STRUCTURAL_CANDIDATE` decision includes the follow-up argument vector
for `buildopt profile measure`; fill the immutable BuildOpt revision if it was
not supplied to `profile propose`. Measurement compares eight isolated
optimized-native and BuildOpt pairs and writes evidence only after every build
and output check succeeds. Then `buildopt profile evaluate` either writes a
reviewable profile or retains native full graph. See the [owner-input
contract](../../specs/poc-generic-owner-input-v1.md), [generic onboarding
contract](../../specs/poc-generic-profile-onboarding-v1.md), and [measurement
contract](../../specs/poc-generic-measurement-v1.md).

This workflow is not limited to the small conformance fixture. Starting from
fresh public checkouts, the installed command independently reproduced Apache
Groovy's qualified 37-to-2 project plan and Micronaut Core's 75-to-22 plan
without copying retained BuildOpt JSON. The replay intentionally did not repeat
unchanged timings; review the [captured onboarding evidence](../../benchmarks/results/poc-generic-profile-realworld-v1/README.md)
before adapting the command to another repository.

For a pull request, commit the qualified profile described in the
[configuration reference](../reference/configuration.md#qualified-poc-profile),
create an exact changed-path input and execute the reviewable profile:

```bash
git diff --name-only --diff-filter=ACMR BASE_SHA HEAD_SHA > .buildopt-changes
buildopt-impact check \
  --repository . \
  --repository-id owner/repository \
  --pipeline-class pull-request
buildopt poc --changes-file .buildopt-changes
```

This command is available in public `v0.3.1`. The six retained Groovy and
Kafka profiles have been replayed through that exact package: each exact
profile selected its reviewed graph, each digest-drift probe retained the
native full graph, and every candidate matched its same-run native fallback
under the owner-reviewed output contract. Review the
[installed replay bundle](../../benchmarks/results/poc-installed-profile-replay-v1/README.md).

The profile remains repository-owned and review-required. BuildOpt does not
discover, qualify or activate an optimization automatically during this
command, and any binding drift returns to the original Gradle entrypoints.

The command prints the complete selected/fallback plan, exact adapters and
expected outputs before Gradle begins. It does not select tests or grant
production promotion. Keep the
existing Test-owned commands in the workflow, compare required outputs between
candidate and full builds, and treat the measured result as POC evidence.

## Add CI

### GitHub Actions

Pin the Action source to a reviewed 40-character commit SHA. No release inputs
are required for the normal path:

```yaml
- name: Install BuildOpt
  uses: tonyredondo/buildopt@<40-character-commit-sha>

- name: Build
  run: buildopt gradle --no-daemon build
```

Set `version` when the workflow must pin a particular binary release. The
Action resolves the published archive checksum, verifies the archive and adds
the installed commands to `PATH`. Existing consumers that provide the legacy
`archive-url` and `archive-sha256` pair remain supported.

### GitLab CI

The component is self-contained; the consuming repository does not copy
BuildOpt scripts:

```yaml
include:
  - project: tonyredondo/buildopt
    ref: <40-character-commit-sha>
    file: /.gitlab/buildopt-component.yml
    inputs:
      gradle-tasks: build
```

The component installs the matching public release into `.buildopt/runtime`,
runs `buildopt gradle`, and retains the normalized job event and redacted
exports. Add `version: 0.2.0` to pin the native package.

## What improvement should you expect?

Do not use one headline percentage as an expectation for a new repository.
The current comparable evidence measures the same installed structural-only
method against optimized native Gradle:

| Repository | Native mean | BuildOpt mean | Result | Current decision |
| --- | ---: | ---: | ---: | --- |
| Spring Framework | 13.311 s | 11.183 s | **15.99% faster**, 8/8 balanced blocks | Qualify. |
| OpenTelemetry | 87.869 s | 74.713 s | **14.97% faster**, 8/8 balanced blocks | Qualify. |
| Apache Kafka | 113.381 s | 14.341 s | **87.35% faster**, 8/8 balanced blocks | Qualify. |
| Micronaut Core | 30.411 s | 18.418 s | **39.44% faster**, 8/8 balanced blocks | Qualify. |
| Apache Groovy | 79.868 s | 20.767 s | **74.00% faster**, 8/8 balanced blocks | Qualify. |
| Ktor JVM JAR workflow | 103.724 s | 14.308 s | **86.21% faster**, 8/8 balanced blocks | Qualify. |

Every accepted row preserved the declared outputs byte for byte and proved
full-graph fallback. These results show that structural reduction can create
material value, not that a new repository will match one of these percentages.
The propose -> measure -> evaluate workflow must establish that repository's
own result. Safe Cache remains at native-cache parity; Runtime Tuning, Hot
State, and standard Copy are retired. Mechanism and repository percentages are
not averaged or added. See the [current POC one-pager](../findings/buildopt-poc-handoff.md)
for the interpretation and roadmap, and the
[benchmark index](../../benchmarks/README.md#build-optimization-scorecard) for
the raw evidence.

Ktor also has three change-breadth results under the same selector:
dependency source is **85.80% faster**, a JVM resource is **86.51% faster**,
and a two-module mixed-source change is **77.98% faster**. Each result has
16/16 positive pairs, exact required JARs and both fallbacks; root
configuration retains native Gradle without a timing claim.

## Component ownership and configuration

Start with only the launcher. Add another component when a concrete need
justifies operating it.

| Component | Installed with | Configured in | Orchestrated by | First proof |
|---|---|---|---|---|
| Launcher and native Gradle cache | Native package or CI integration | No configuration for the default path | `buildopt gradle` | Run the same build twice and inspect normal Gradle output |
| Safe Cache experiment | Included in the native package | `BUILDOPT_SAFE_CACHE=1`; not a recommended default | `buildopt gradle` | Use only when collecting new paired POC evidence |
| Task Intelligence | No separate installed CLI; owner evaluation in this source repository | Qualification evidence and trace contracts | `task-intelligence-evaluation` and owner checks | Prove qualification without publishing an optimization |
| Build history | `buildopt-server` in the package | Server config and export directory on the server host | Service manager plus server API/dashboard | Start loopback server and inspect a redacted session |
| Build Impact | `buildopt` and `buildopt-impact` in the package | Manifest and generated graph committed in the target repository | CI calls `generate` during adoption, `check` for drift, then explicit `buildopt impact` with a changed-path file | Verify candidate/full fallback and compare required outputs before interpreting timings |
| Patch Autopilot | Java patcher and owner workflow in this source repository; not yet a native-package CLI | Recipe registry, signing/trust material and repository policy | Owner-controlled candidate/validation workflow | Produce a draft bundle; applying remains explicit and reversible |
| Shared Cache | `buildopt-server` | Private server config, authority and scoped credentials on operator hosts | Server service plus launcher gateway | Validate config, start loopback service, run an authorized repository |
| Edge Cache | `buildopt-edge` | Private Edge config and authority on the Edge host | launchd, Windows SCM, or foreground process | `buildopt-edge validate`, then `serve` and `status` |
| JVM agent | Native package | Enabled only by the relevant policy | Launcher/plugin path | Use the owning runtime workflow; no separate installation |

The [product workflows](../guides/product-workflows.md) contain the detailed
commands for Build Impact, history, runtime policy, Patch Autopilot and Edge.
The [configuration reference](../reference/configuration.md) separates safe
defaults from operator-owned credentials and private configuration.

## Recommended adoption order

1. Run `buildopt doctor` and capture the real optimized-native Gradle command
   and wall-time baseline.
2. Declare the original Gradle entrypoint, exact Git change, and required
   output globs; these are repository-owner decisions rather than BuildOpt
   guesses.
3. Run `buildopt profile propose` and review the selected graph or explicit
   native-fallback reason.
4. Run the isolated `profile measure -> profile evaluate` flow. Accept no
   profile unless required outputs, repeatability, wall-time value, and the
   full-graph fallback all pass.
5. Commit the reviewed profile and exercise `buildopt poc` in one CI job with
   `BUILDOPT_BYPASS=1` available as immediate rollback.
6. Consider Patch Autopilot or Edge only after structural value is established
   and their own workload-specific evidence justifies the extra mechanism.

This ordering keeps the first result local and reversible. Persistent storage,
network access, credentials and repository mutation are never prerequisites
for the first build, and a new repository remains on optimized native Gradle
until its own evidence qualifies.

Runtime Tuning, exact-bound Hot State, and standard Copy are not onboarding
options. They were retired after failing incremental end-to-end value gates;
their historical evidence remains available only for audit.

## Develop BuildOpt itself

Source bootstrap, the synthetic product lab and repository validation belong
to contributors, not first-time product users. Follow
[developer onboarding](./developer-onboarding.md) when changing BuildOpt, or
the [source evaluation path](./quickstart.md#source-based-product-lab) when
you need the complete project-owned synthetic proof.
