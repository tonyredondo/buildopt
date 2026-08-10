# Product onboarding

The default BuildOpt experience has one installation step and one command:

```text
install BuildOpt -> open a Gradle repository -> buildopt gradle build
```

The package contains the launcher, Gradle plugin and init script, Build Impact,
the self-hosted server, Edge Cache, and the JVM agent. Users do not need to
clone this repository, build Go or Java sources, copy JARs, or set internal
plugin paths.

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

## Try the Build Impact accelerator

The default Gradle command establishes compatibility; Build Impact is the POC
feature that has demonstrated incremental value against optimized native
Gradle. Adopt it only after reviewing and committing the repository manifest
and generated graph described in the
[Build Impact workflow](../guides/product-workflows.md#build-impact).

Before committing a qualified profile, current `main` can create the exact POC
inputs without hand-authoring Build Impact JSON:

```bash
git diff --name-only --no-renames "$BASE_SHA" HEAD > buildopt-changes.txt
buildopt profile propose \
  --repository-id owner/repository \
  --pipeline-class classes \
  --entrypoint classes \
  --changes-file buildopt-changes.txt \
  --base-revision "$BASE_SHA" \
  --required-output 'module/build/classes/**'
```

Review `buildopt-profile-proposal.json` first. A
`MEASURE_STRUCTURAL_CANDIDATE` decision includes the follow-up argument vector
for `buildopt profile measure`; fill the immutable BuildOpt revision if it was
not supplied to `profile propose`. Measurement compares eight isolated
optimized-native and BuildOpt pairs and writes evidence only after every build
and output check succeeds. Then `buildopt profile evaluate` either writes a
reviewable profile or retains native full graph. See the [generic onboarding
contract](../../specs/poc-generic-profile-onboarding-v1.md) and [measurement
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

This short command is available in source-built packages from current `main`.
The published `v0.2.0` package uses the equivalent explicit `buildopt impact`
form until a later tagged release.

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

The current scorecard measures mechanisms separately so one feature cannot
hide another's cost:

| Mechanism | Measured result | What the comparison proves |
|---|---:|---|
| Safe Cache | `NO_VALUE_NO_ACTION`; explicit-only | The default delegates to Gradle native cache, removing product overhead without claiming acceleration |
| Installed Spring Build Impact | 15.76% faster; 1,260.125 ms saved; 8/8 positive pairs | The packaged command beats optimized native Gradle for one reviewed Spring output scope with identical declared outputs |
| Installed Spring breadth | `spring-webmvc` 13.50% faster and qualified; shared `spring-core` scope averaged 10.89% faster but failed 4/4 stability | The POC remains output-scope-specific and does not promise acceleration for every change |
| Runtime Tuning | `RETIRED` | `W4_H6G` regressed 54.7%; `W3_H4G` regressed 4.3%; the final Spring worker cap was 2.00% slower |
| Build Impact | 73.5–76.0% faster in the strict bounded Kotlin/Groovy workloads | Avoided unrelated non-cacheable work while required outputs stayed unchanged |
| Reviewed Task/Patch | 67.3–68.0% faster for the exact reviewed custom-task recipe | Restored all eight qualified task outputs; this does not generalize to other recipes |
| Combined public path | 63.5–84.1% faster across four strict Kotlin/Groovy workload cells | The actual packaged CLIs and plugin beat optimized native Gradle with unproven mechanisms disabled |

These are controlled POC workloads, not universal predictions, and the
mechanism percentages must not be added; the combined row is measured directly.
A separate realistic five-project matrix initially qualified 2/8 cells. After
installed-path attribution and removal of one candidate-only environment
difference, the unchanged repeat qualified leaf Kotlin/Groovy and shared Kotlin
acceleration plus no-change Kotlin parity (4/8 cells). Global build-logic,
no-change Groovy, and shared Groovy remain outside the claim. The isolated-arm
experiment then exposed four order-dependent classifications, so those cells
remain outside the claim while a temporally paired, state-isolated measurement
removes runner drift.
A repository that needs the full graph receives
no Build Impact saving; an already warm native cache is a parity baseline, not
the cache-off comparison. Run `./dev/check-poc-value-validation` to validate
and print the `CONTINUE` decision without rerunning Gradle. This means the idea
merits further POC work for the qualified synthetic classes, not that it is
production-ready. The
[benchmark index](../../benchmarks/README.md#build-optimization-scorecard)
links every raw observation and measurement contract.

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

1. Run `buildopt doctor` and one local `buildopt gradle` build.
2. Put the same command in one CI job and keep `BUILDOPT_BYPASS=1` as rollback.
3. Collect build history before changing policy so improvements have a
   baseline.
4. Add Build Impact independently and compare it against the history baseline.
5. Evaluate Task Intelligence and Patch Autopilot through their owner workflows; they are not first-run package switches.
6. Operate Shared or Edge only when local/CI reuse justifies a persistent
   service and its credential lifecycle.

This ordering keeps the first result local and reversible. Persistent storage,
network access, credentials and repository mutation are never prerequisites
for the first build.

Runtime Tuning, exact-bound Hot State, and standard Copy are not onboarding
options. They were retired after failing incremental end-to-end value gates;
their historical evidence remains available only for audit.

## Develop BuildOpt itself

Source bootstrap, the synthetic product lab and repository validation belong
to contributors, not first-time product users. Follow
[developer onboarding](./developer-onboarding.md) when changing BuildOpt, or
the [source evaluation path](./quickstart.md#source-based-product-lab) when
you need the complete project-owned synthetic proof.
