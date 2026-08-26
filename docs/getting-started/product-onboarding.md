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
human/JSON result and exit contract are now implemented. The command runs
optimized native Gradle, discovers a structural proposal and calibrates it
under one bounded deadline. A later invocation with the same exact bindings
automatically selects the qualified profile before Gradle; any drift retains
optimized native Gradle. Use the explicit commands below to inspect the
lower-level stages. Follow the
[one-command onboarding roadmap](../plans/one-command-onboarding-roadmap.md)
for the remaining ordered delivery.

## Planned sticky onboarding experiment

The next POC removes the global installation step as well. A maintainer will
generate and commit four portable files once:

```text
buildoptw
buildoptw.bat
.buildopt/wrapper.properties
.buildopt/config.toml
```

The repeated developer and CI command will be:

```bash
./buildoptw build
```

The wrapper will checksum-verify and cache the pinned BuildOpt distribution,
invoke the repository's existing Gradle Wrapper and choose native, observe,
shadow, bounded trial or exact active behavior. The committed configuration may
name an HTTPS BuildOpt Server and project scope but never contains a token.
Credentials remain in an environment variable, CI secret/OIDC exchange or
private file and are not passed to Gradle.

This command is not implemented yet. The current installation and commands
below remain the usable path until `SWL-002..004` close. The complete sequence,
scorecard and status are in the
[Sticky Wrapper Learning POC Tracker](../plans/sticky-wrapper-learning-poc-tracker.md).

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

Use `--version <published-version>` to pin a release or `--prefix /absolute/path` to choose
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

Open a new terminal after `-UpdatePath`. `-Version <published-version>` pins a release and
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

Today it executes the optimized native Gradle baseline, derives the exact Git
change, Gradle-owned outputs and complete structural graph, then measures a
safe candidate through eight order-balanced pairs. Qualification requires
equivalent outputs, a successful full-graph fallback, a positive paired bound,
at least 500 ms and 2% mean saving, at least 6/8 positive pairs, a positive
paired 95% lower bound, non-regressive p95, and repayment within
`--max-break-even-builds`. A positive candidate reports
`LEARNING / QUALIFIED_PROFILE_STORED`; ambiguity, global changes,
unsupported relationships, insufficient evidence, poor payback and full test
execution retain native Gradle or make no claim with an exact reason. It writes
private generated state/evidence under `.buildopt/optimize/v1`, requires no
hand-authored BuildOpt files and stores qualifying candidates in a private
portfolio keyed by structural change family. Repeating the exact command on the
same revision automatically validates and selects that profile before Gradle;
any repository, Wrapper, executable, option, graph, output, evidence or profile
drift runs the original optimized native graph instead. Selection remains
POC-only and grants no production authority. Machine-readable mode reserves
stdout for one result document:

The adaptive-fragment successor is not yet part of the installed command. Its
first runtime proof is complete in the repository: two independent producers
can restore exact unaffected outputs, rebuild only a changed producer and fall
back to the original complete workflow when change or stored state is unsafe.
This closes activation correctness only. The current customer command remains
the whole-profile POC above. AF-011 measured direct net wall-time value for the
adaptive composition, but retained independently qualified fragments because
one constituent missed its frozen repeatability gate. AF-012 now preserves the
same typed portfolio and ledger locally and through the HTTPS state plane,
including clean-machine restore and offline/native fallback. AF-013 now remains
an immutable historical audit rather than the current scorecard. AF-014A has
now proved the public installed command, separate persistent arms, forward-only
learning, exact selected/native-retained/bypass paths and reconciled phase
timing. AF-014B..D subsequently froze larger commit cohorts, executed the
current implementation chronologically and attributed its value before the
terminal decision. AF-014C completed 100 exact-output current pairs
with zero product failures, but all 100 invocations retained native Gradle and
the signed total is -368.623 seconds. AF-014D attributes 179.029 seconds to the
recorded BuildOpt path and -189.593 seconds to residual Gradle/runner
variation. The native-retention path is 0.531 seconds p50 and 8.656 seconds
p95; discovery/learning alone records 98.385 seconds. Because no mechanism
activated, attributable saving is zero. This adaptive successor therefore
remains outside customer onboarding. AF-015 closes it as
`STOP_ADAPTIVE_FRAGMENT_POC`: 9/15 frozen criteria pass, while activation
(0/71), breadth (0/5), positive confidence (0/5), portfolio value (-368.623
seconds), payback (0 families) and native-retention cost fail. With no
activated saving, bounded specialization is not authorized. The controlled
AF-014A fixture remains apparatus evidence only.

```bash
buildopt optimize --json -- build
```

Every completed invocation writes two customer-facing files beside the exact
state:

- `.buildopt/optimize/v1/value-report.md` explains graph reduction, observed
  installed-path saving, uncertainty, p95, learning cost, break-even and the
  exact fallback reason;
- `.buildopt/optimize/v1/value-report.json` contains the source metrics and
  formulas needed to recompute every derived number.

Cumulative value in an ordinary invocation is clearly labeled as a projection
over successful exact replays, not observed cumulative wall time. Expected
useful lifetime remains `UNAVAILABLE` unless that profile has its own
cross-commit evidence; the POC never applies one repository's lifetime to
another profile.

`BUILDOPT_BYPASS=1 buildopt optimize build` skips optimize state and reporting
entirely. The [command contract](../../specs/poc-magic-onboarding-contract-v1.md)
defines resume bindings, budgets, outcomes and exit behavior; the
[automatic-discovery contract](../../specs/poc-magic-auto-discovery-v1.md)
defines derived inputs, evidence and native fallbacks. The
[automatic-calibration contract](../../specs/poc-magic-calibration-v1.md)
defines measurement, value, payback and exact evidence-reuse gates. The
[profile-portfolio contract](../../specs/poc-magic-profile-portfolio-v1.md)
defines generic family classification, digest bindings, bounded coexistence
and tamper recovery. The
[automatic-replay contract](../../specs/poc-magic-auto-replay-v1.md) defines
exact pre-Gradle matching, measured decision overhead and native fallback.
The [one-input CI contract](../../specs/poc-magic-ci-onboarding-v1.md) makes
that same exact command portable between clean runners of one provider
repository without trusting restored files or requiring a BuildOpt service.
The [value-report contract](../../specs/poc-magic-wow-report-v1.md) defines the
human explanation, recomputation formulas and POC boundary.

No server is required for this flow. The optional cross-machine design now has
an executable [storage contract](../../specs/poc-central-storage-contract-v1.md),
and its [local typed store](../../specs/poc-central-state-storage-v1.md)
persists portfolios, evidence and checkpoints safely inside `buildopt-server`.
Its [central HTTPS boundary](../../specs/poc-central-https-auth-v1.md) accepts
trusted TLS 1.3 clients with separately scoped cache/state tokens. The Gradle
gateway proof and [state-sync contract](../../specs/poc-central-state-sync-v1.md)
now add exact cache reuse plus one-time `buildopt connect` and explicit
`buildopt sync`. The
[central optimize integration](../../specs/poc-central-optimize-integration-v1.md)
then makes state lookup/publication automatic around the normal optimize
invocation and permits only locally revalidated source-commit reuse. The
[isolated two-machine proof](../../specs/poc-central-two-machine-v1.md) now
shows the complete connected path: a clean consumer automatically selects the
remote profile and read-only cache, while outage rebuilds with verified local
state. The subsequent
[central value result](../../benchmarks/results/poc-central-end-to-end-value-v1/README.md)
compares equal committed cache opportunity and records **82.45% lower wall
time on Ktor** and **56.41% on Beam**, with exact outputs and 8/8 positive
pairs. Running the service remains optional and operator-owned; this is bounded
POC evidence, not production qualification.

The separate
[Ktor lifetime result](../../benchmarks/results/poc-profile-lifetime-v1/README.md)
shows the economic limitation behind this onboarding. One matching replay
saved 112.198 seconds, but an unrelated-owner fallback added 220.761 seconds
and the 1,443.324-second learning cost did not repay in the observed public
window. The subsequent
[economic prequalification result](../../benchmarks/results/poc-economic-prequalification-v1/README.md)
uses direct graph ownership and bounded recent history to reject learning for
that unrelated CORS owner in 192.442 ms. It performs no discovery or
calibration, reducing the observed fallback penalty to 13.896 seconds while a
matching Jetty replay still saves 100.744 seconds. Qualification still needs
31 matching builds and remains unpaid in the observed window, so a large
calibrated speedup is technical potential rather than guaranteed cumulative
customer value.

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

Pin the Action source to a reviewed 40-character commit SHA. The only ordinary
BuildOpt input is the existing Gradle workflow:

```yaml
- name: Optimize the build
  uses: tonyredondo/buildopt@<40-character-commit-sha>
  with:
    command: optimize build
```

The Action installs the verified release, derives GitHub base/head revisions,
restores only provider-bound exact state, and uploads
`buildopt-optimize-result`. A cache miss or drift runs optimized native Gradle.
Run it after a checkout with the comparison base available (`fetch-depth: 0`
for `actions/checkout`).
Set `version` when the workflow must pin a particular binary release. Existing
install-only and legacy archive consumers remain supported.

### GitLab CI

The component is self-contained; the consuming repository does not copy
BuildOpt scripts:

```yaml
include:
  - project: tonyredondo/buildopt
    ref: <40-character-commit-sha>
    file: /.gitlab/buildopt-component.yml
    inputs:
      command: optimize build
```

The component installs the matching public release into `.buildopt/runtime`,
runs the same `buildopt optimize` command, uses the native project cache only
for exact state, and retains the normalized job event plus review result. Add
`version` to pin the native package. Neither provider requires a BuildOpt
server, credential, hand-authored profile or internal state option.

## What improvement should you expect?

Do not use one headline percentage as an expectation for a new repository.
The latest zero-manual-file evidence uses the public package and the exact
`buildopt optimize` command described above:

| Repository | Native mean | BuildOpt mean | Result | Current decision |
| --- | ---: | ---: | ---: | --- |
| Ktor `jvmJar` | 38.810 s | 7.830 s | **79.82% faster**, 8/8 pairs | Qualify; 26-build payback. |
| Apache Beam `classes` | 65.081 s | 24.958 s | **61.65% faster**, 8/8 pairs | Qualify; 28-build payback. |

Both rows preserve the required outputs byte for byte, lower p95, pass
full-graph fallback and repay calibration within the declared 30 matching
builds. A Ktor root build-logic change also proves the negative path: the
complete native build succeeds and no candidate or timing claim is created.

The newest generalization run applies that unchanged command to five more
public repositories after incremental learning, verified output
materialization and aggregate partitioning are composed. All five candidates
are faster with exact outputs: Spring 12.71%, OpenTelemetry 14.97%, Kafka
54.92%, Micronaut 66.24% and Groovy 75.97%. The last four qualify with 8/8
pairs and 19-, 14-, 8- and 2-build payback. Spring safely remains native at
7/8 and 67-build payback. This is the current onboarding behavior: measurable
automatic value where proven, optimized native Gradle otherwise.

An earlier reviewed-profile matrix reports 14.97% to 87.35% savings across
Spring, OpenTelemetry, Kafka, Micronaut and Groovy. It demonstrates broader
structural potential, but those owner-reviewed inputs are not substituted for
the current zero-configuration result. A new repository must establish its own
output semantics, measured value and economics. Safe Cache remains at
native-cache parity; Runtime Tuning, Hot State and standard Copy are retired.
Mechanism and repository percentages are not averaged or added. See the
[current POC one-pager](../findings/buildopt-poc-handoff.md) for the
interpretation and roadmap, and the
[benchmark index](../../benchmarks/README.md#build-optimization-scorecard) for
the raw evidence.

The current package result is reproducible with
`./dev/check-magic-end-to-end-value-v2`.
The latest breadth decisions are reproducible with
`./dev/check-automatic-breadth-transfer`.

## Component ownership and configuration

Start with only the launcher. Add another component when a concrete need
justifies operating it.

| Component | Installed with | Configured in | Orchestrated by | First proof |
|---|---|---|---|---|
| Launcher and native Gradle cache | Native package or CI integration | No configuration for the default path | `buildopt gradle` | Run the same build twice and inspect normal Gradle output |
| Safe Cache experiment | Included in the native package | `BUILDOPT_SAFE_CACHE=1`; not a recommended default | `buildopt gradle` | Use only when collecting new paired POC evidence |
| Task Intelligence | No separate installed CLI; owner evaluation in this source repository | Qualification evidence and trace contracts | `task-intelligence-evaluation` and owner checks | Prove qualification without publishing an optimization |
| Build history | `buildopt-server` in the package | Server config and export directory on the server host | Service manager plus server API/dashboard | Start loopback server and inspect a redacted session |
| Structural Build Impact | `buildopt` in the native package | No target-repository file for the automatic POC path; private state lives under `.buildopt/optimize` | `buildopt optimize <workflow>` discovers, calibrates and replays; the older explicit `buildopt impact` flow remains available for owner-reviewed experiments | Verify the generated value report, exact outputs, full fallback and payback before trusting a profile |
| Patch Autopilot | Java patcher and owner workflow in this source repository; not yet a native-package CLI | Recipe registry, signing/trust material and repository policy | Owner-controlled candidate/validation workflow | Produce a draft bundle; applying remains explicit and reversible |
| Shared Cache and state | `buildopt-server` | Private server state, trusted certificate/key and owner-issued scoped token; `buildopt connect` stores a private repository connection | A connected `buildopt optimize` automatically reads/publishes verified optimization memory and activates read-only central cache reuse when the token grants `CACHE_READ`; `buildopt sync` remains an explicit diagnostic | Validate paired value with `check-central-end-to-end-value` and isolated restart/outage behavior with `check-central-two-machine`; lower-level TLS, cache, state and profile checks remain available for diagnosis |
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
