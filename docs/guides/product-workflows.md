# Product workflows

## Sticky wrapper lifecycle (POC)

The active successor POC makes one committed `./buildoptw <gradle args...>`
command the stable local and CI entrypoint. The wrapper bootstraps an immutable
BuildOpt version, preserves the repository Gradle Wrapper, and uses the
read-only central cache path when a valid connection is available. Decision
consumption is intentionally later, but its canonical state contract is now
implemented:

```text
NATIVE_NOOP -> OBSERVE -> SHADOW -> TRIAL -> QUALIFIED -> ACTIVE
                                              |             |
                                              +------> SUSPENDED -> RETIRED
```

The lifecycle is evidence-gated rather than a user-selected optimization mode.
Expensive trials run only in isolated trusted CI within a 5% natural-compute
budget. Runtime profiles execute only while every exact binding and value lease
matches. Review-required durable patches become ordinary native Gradle behavior
after acceptance. Any ambiguity, expiry, service failure or regression retains
the original Gradle workflow.

Gradle HTTP cache objects are reusable data; they never authorize lifecycle
transitions. Typed decisions, observations and the economic ledger use the
separate BuildOpt state plane. This workflow is planned under the
[Sticky Wrapper Learning POC Tracker](../plans/sticky-wrapper-learning-poc-tracker.md)
and is intentionally still a POC rather than a production rollout path.

`SWL-001..005` fix the committed files, deterministic generator, verified
bootstrap, neutral passthrough and portable connection. `SWL-006` now consumes
the existing Gradle-compatible central cache through an invocation-local
verifying gateway when the private connection is valid. It adds
`--build-cache` unless explicitly disabled, keeps ordinary consumers read-only,
and falls back to native Gradle on missing credentials, corruption or outage.
`SWL-007` defines and tests the typed decision/evidence store, signed
decisions, generation CAS, expiry, revocation and plane separation. `SWL-008`
now consumes that store through a read-only native-retention selector: a
verified local `NATIVE_NOOP` is accepted before Gradle, while active decisions
remain deferred until action execution is proven. Invalid or unavailable state
retains native Gradle and refreshes only outside the critical path. Ordinary
`SWL-009` now records ordinary requested builds in a separate private JSONL
observation plane; it measures wrapper, decision, cache and Gradle boundaries
without changing the command or authorizing an action. Trial orchestration is
available only in trusted CI: `SWL-010` runs four alternating, fully-isolated
candidate/native pairs under a 5% compute ceiling. The first checked-in trial
is exact but negative (7.534 s candidate vs 6.979 s native, 0/4 positive), so
no active optimization is authorized. This is the required fail-closed
behavior: a trial can produce evidence without changing ordinary developer
builds.

### Active runtime boundary

`SWL-011` implements the next control boundary without pretending that the
current negative trial is profitable. A profile is eligible only when its
signed decision is valid, unexpired, unrevoked, exactly bound to the current
workflow and marked `ACTIVE_RUNTIME_PROFILE`. The runner executes direct
candidate/native commands without a shell and hashes the declared outputs.
When the counterfactual cadence is due, native Gradle runs as the authority;
any candidate failure, cancellation, output mismatch or tolerance breach
suspends the profile and retains native execution until a new decision
generation is supplied. Bypass, drift and unavailable state retain native
without attempting the candidate.

The focused evidence is synthetic control-flow coverage, not a repository
speedup claim:

```bash
./dev/check-sticky-wrapper-active
```

The current result contains one positive active scenario (about 24.6 ms saved),
three suspensions and four native retentions. The checked-in SWL-010 report is
still rejected with `TRIAL_NOT_PROFITABLE`; only a future repository-level
paired result can authorize a real action.

### Durable native opportunity catalog

`SWL-012` adds a review-only catalog for opportunities that can become ordinary
native Gradle behavior after acceptance. It currently recognizes two generic
shapes: an expensive custom task missing a stable input/output contract, and a
declared workflow whose dependency breadth exceeds the observed required
outputs. The current strict POC evidence accepts the task-contract detector in
both Kotlin and Groovy (64.1% and 74.7% mean savings over eight exact pairs per
family). Every recipe is digest-bound, validated and exactly reversible outside
the checkout; no proposal is merged or activated automatically. Graph breadth
is still structural-only until its durable timing is measured. See the
[durable catalog specification](../../specs/poc-sticky-wrapper-durable-catalog-v1.md),
[current report](../../benchmarks/results/sticky-wrapper-durable-catalog-v1.json)
and `./dev/check-sticky-wrapper-durable-catalog`.

### Ordinary-build observation

When observation is enabled, the launcher appends one canonical record per
ordinary Wrapper invocation. The record includes exact provenance hashes,
outcome and exit code, Configuration Cache state, and phase timing. A phase is
`EXACT`, `APPROXIMATED` or `UNAVAILABLE`; unavailable data is never represented
as zero. Records are private, append-only and independently loadable, so the
learning loop can charge its own overhead before considering a trial.

Set `BUILDOPT_STICKY_OBSERVATION=0` to disable this diagnostic plane. The
default output is under the user cache in a repository-scope directory; set
`BUILDOPT_STICKY_OBSERVATION_OUTPUT` only for a private test destination.
Validate the contract with `./dev/check-sticky-wrapper-observation`.

Validate the bounded trial harness with `./dev/check-sticky-wrapper-trial`.

This guide describes what a user or evaluator can do with the current POC and
which interface owns each result. It is intentionally honest about surfaces
that remain developer/operator workflows rather than a hosted product.

## Launcher and immediate recovery

Run any existing Gradle command without changing its argv:

```bash
buildopt run -- ./gradlew --build-cache build
```

The `--` delimiter is mandatory. BuildOpt does not invoke a shell, so quoting,
wildcards, streams, working directory, and the child exit status keep their
normal process semantics.

Remove all configured optimization behavior immediately:

```bash
BUILDOPT_BYPASS=1 buildopt run -- ./gradlew --build-cache build
```

Use the [base recovery runbook](../../runbooks/base-recovery.md) for CI kill
switches, immutable rollback, uninstall, state preservation, and partial patch
recovery.

## Evaluate the adaptive one-command path

The current customer-facing POC remains:

```bash
buildopt optimize build
```

Unsupported, drifted or non-economic state executes optimized native Gradle;
`BUILDOPT_BYPASS=1` removes optimize state and reporting entirely. AF-014A has
validated the installed command with separate persistent control/candidate
arms, forward-only learning, exact selected/native-retained/bypass behavior and
reconciled phase timing. Its controlled fixture is measurement-apparatus
evidence, not a speedup. Public-repository value must come from the cohorts
frozen by AF-014B and measured by AF-014C/D.

## Build history and dashboard

`buildopt-server` can export immutable redacted `BUILD_SESSION v1` documents
and expose them through an authenticated loopback-only API and embedded UI.
Use independent write and read credentials:

```bash
BUILDOPT_SERVER_INGEST_TOKEN='<write-only-token>' \
BUILDOPT_HISTORY_API_TOKEN='<independent-read-token>' \
  buildopt-server serve \
    --listen 127.0.0.1:8042 \
    --export-dir /absolute/private/buildopt-exports
```

Then open `http://127.0.0.1:8042/buildopt/` and enter only the history token.
The page stores it in memory, makes same-origin requests, and displays fields
from real redacted session documents. The server does not embed credentials or
history in the static assets.

The corresponding API routes are:

```text
GET /api/v1/build-sessions?repository=TOKEN&outcome=SUCCESS&limit=25&cursor=OPAQUE
GET /api/v1/build-session?id=SESSION_ID
```

Both require `Authorization: Bearer <history-token>` and return
`Cache-Control: no-store`. Omitting `BUILDOPT_HISTORY_API_TOKEN` leaves the
history API and dashboard route unavailable.

To copy the validated stream as a private CI artifact:

```bash
buildopt-server export \
  --export-dir /absolute/private/buildopt-exports \
  --format jsonl
```

The launcher-to-server context and complete server options are documented in
[`cmd/buildopt-server/README.md`](../../cmd/buildopt-server/README.md).

## Managed cache

The normal `buildopt gradle` command requires no cache configuration. It
creates a private repository- and Wrapper-compatible native L1 below the user
cache directory, enables Gradle's build cache, and applies the safe Tier 1 task
policy. `--no-build-cache` is the cache-only opt-out.

For an operated Shared deployment, BuildOpt composes that native local cache
with an authenticated loopback HTTP cache:

```text
Gradle -> managed DirectoryBuildCache (L1)
       -> Local Verifying Cache Gateway (L2 endpoint)
       -> optional Edge
       -> Shared
```

The advanced Shared configuration consists of a private signed authority,
trust root, cache credential or scoped token, Shared URL, managed L1 scope,
and compatible Gradle init script. Treat it as one atomic group; do not invent
partial values. The detailed environment contract is in
[configuration reference](../reference/configuration.md#managed-l1-and-shared-cache).

Reads require a committed object, matching scope and generation, current
revocation state, and complete-byte checksum verification. Writes remain
pending until the control path authorizes commit. Any unknown, invalid,
expired, conflicting, or unavailable state falls back to a miss or baseline.

## Retired optimization experiments

Runtime Tuning, exact-bound Hot State, and the standard-Copy adapter are no
longer product workflows. Repeated optimized-native comparisons found no
stable incremental value, so their launchers, plugins, evaluators, and hosted
workflows were removed. Historical contracts and immutable benchmark results
remain available for audit; they are not instructions for activation.

Configuration Cache support remains because it delegates to Gradle's native
feature and is independent of the retired resource tuner.

## Task Intelligence

Task Intelligence models the lifecycle from observation through contractual
qualification, quarantine validation, activation, and suspension. Exact trace
coverage is required where a decision depends on producer behavior. The JVM
Agent and Linux helper are supporting experiments and cannot independently
qualify a task.

Run the owner-controlled evaluation:

```bash
./dev/check-task-intelligence-poc
```

The fallback for incomplete or unavailable evidence is no publication or
`ABORT_PENDING`. BuildOpt does not reinterpret unavailable evidence as a task
with no inputs or outputs.

## Patch Autopilot

Patch Autopilot is a signed, exact, PR-only workflow. The current recipe
registry covers bounded archive reproducibility, Groovy DSL archive
reproducibility, root build-cache properties, and custom task contracts.

For the adaptive POC, ordinary requested builds may first create a review-only
task-contract opportunity when the same expensive task repeatedly executes
with stable inputs/outputs while neither cacheable nor up-to-date. Detection is
repository-name independent and cannot apply a patch. An owner must accept an
exact recipe; BuildOpt then validates it in an isolated workspace, checks exact
revert and measures native Gradle before/after. A rejected proposal leaves the
checkout unchanged.

The Java patcher:

1. verifies canonical JSON, signature, expiry, repository/base/source-state
   binding, recipe version, paths, modes, and every pre/postimage;
2. creates a private detached Git worktree without running checkout hooks or
   content filters;
3. applies only exact `ADD` and `MODIFY` operations;
4. runs the required validation boundary;
5. creates or reuses only an exact action branch and draft delivery;
6. can generate an exact signed inverse bundle for a proven regression.

It never executes bundle content, fuzz-applies a patch, rebases, force-pushes,
merges, or uploads customer source by itself. Remote draft-PR delivery is an
adapter owned by the caller; the repository proof uses an in-memory adapter
and no credentials.

Exercise the implemented workflow with:

```bash
./dev/check-patch-bundle-applier
./dev/check-full-relevant-validation
./dev/check-customer-patch-workflow
./dev/check-patch-delivery-recovery
./dev/check-post-merge-patch-monitor
./dev/check-patch-autopilot-recipes
./dev/check-patch-autopilot-validation-revert
./dev/check-adaptive-fragment-patch-opportunity
```

See the [patcher README](../../jvm/patcher/README.md) and
[customer workflow specification](../../specs/customer-patch-workflow-v1.md).

## Build Impact

Build Impact can discover a conventional Gradle project graph while keeping
the repository-owned manifest as authority. Generate reviewable state:

The adaptive-fragment POC now also has a repository-internal activation proof.
It pairs a qualified producer subgraph with its exact output materialization,
restores only verified unaffected bytes and rebuilds only producers whose
bindings or output revision changed. Global, ambiguous or incomplete state
returns to the complete original workflow before Gradle starts. This is not
yet an installed customer workflow and it has no wall-time claim; run
`./dev/check-adaptive-fragment-activation` from a BuildOpt checkout to inspect
the six-scenario Gradle proof.

```bash
buildopt-impact generate \
  --repository . \
  --manifest buildopt-impact-manifest.json \
  --repository-id owner/repository \
  --pipeline-class pull-request \
  --graph buildopt-impact-graph.generated.json \
  --generated-manifest buildopt-impact.generated.json
```

Commit the customer manifest and reviewed generated files. CI verifies drift
with the same options and `check` instead of `generate`:

```bash
buildopt-impact check \
  --repository . \
  --manifest buildopt-impact-manifest.json \
  --repository-id owner/repository \
  --pipeline-class pull-request \
  --graph buildopt-impact-graph.generated.json \
  --generated-manifest buildopt-impact.generated.json
```

Before adopting a candidate, confirm the workflow and outputs into one owner
file, then collect exact workload evidence instead of assuming graph reduction
equals wall-clock value:

```bash
mkdir -p .buildopt
buildopt profile outputs \
  --repository-id owner/repository \
  --pipeline-class pull-request \
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

`--entrypoint` is repeatable. Multi-entrypoint workflows retain every declared
native task as fallback while the generic proposal derives candidate task
selectors from their terminal names and the exact owners of the change.
`GIT_DIFF_BASE_TO_HEAD` is stored in the owner file, so local and CI invocations
derive the same exact no-rename change set.

Before graph discovery, the proposal executes the exact owner workflow once
and writes `buildopt-output-contract.json`. Review the non-empty candidate
paths, Gradle project owners and producer tasks. An empty or ambiguously owned
declaration stops on native Gradle before any warm-up or timing. Use
`buildopt profile outputs` when output ownership needs review before preparing
the full change-bound proposal. `buildopt profile input --check
.buildopt/profile.json` verifies the checked owner file without running Gradle.

After output validation, the proposal command produces the manifest, graph,
generated binding and fallback input used below without hand-authored JSON.
Review the selected task set, omitted projects, required output scope and native
fallback before running the emitted measurement command.

The same owner-input flow has been checked for packaging (`Jar`), typed
verification (`VerificationTask`), distribution (`AbstractArchiveTask`), and
build-owned test preparation (`testClasses`). In each case the proposal selects
the exact changed-project task and preserves the declared output bytes without
running a Gradle `Test` task. Custom executable tasks without one of the
supported public semantics remain on the original native workflow before
timing. This capability proof does not imply that every supported candidate is
faster; each reviewed candidate still needs paired installed-path evidence.

The same packaged proposal path has been checked from clean Apache Groovy and
Micronaut Core revisions. It rediscovered their previously qualified 37-to-2
and 75-to-22 project plans without retained BuildOpt files or repository-name
rules. See the [public-repository replay evidence](../../benchmarks/results/poc-generic-profile-realworld-v1/README.md).

```bash
buildopt profile measure \
  --manifest buildopt-impact-manifest.json \
  --graph buildopt-impact-graph.generated.json \
  --generated-manifest buildopt-impact.generated.json \
  --changes-file buildopt-changes.txt \
  --fallback-changes-file buildopt-fallback-changes.txt \
  --base-revision "$BASE_SHA" \
  --buildopt-revision "$BUILDOPT_REVISION" \
  --evidence-output buildopt-structural-evidence.json
```

The target tracked tree must be clean and the declared changes must equal the
Git diff. The command uses independent clones, Gradle homes and cache seeds for
eight alternating pairs, requires stable byte-identical outputs and executes a
full-graph fallback. Feed the resulting evidence to `buildopt profile
evaluate`; do not commit or activate an inconclusive result.

For an owner-operated POC, commit `buildopt-qualified-profile.json` and execute
the reviewed clean profile without wiring an internal selection harness:

```bash
git diff --name-only --diff-filter=ACMR BASE_SHA HEAD_SHA > .buildopt-changes
buildopt poc --changes-file .buildopt-changes
```

This explicit command is not autonomous production selection. It requires the
current exact generated binding, reports its plan before execution and may
choose only a manifest alternative. Its standard `Jar` adapter is disabled on
the native/full-graph fallback path.
Included builds, test-bearing alternatives, unsupported task types, missing
projects, drift, unknown paths, and global changes use the original full
entrypoints or reject invalid state before Gradle. Test Optimization-owned
checks are preserved separately and stay in their existing workflow.

For the qualified Kafka composition, commit the v2 profile and pass the
already-running local Edge origin explicitly:

```bash
buildopt poc --changes-file .buildopt-changes \
  --edge-url http://127.0.0.1:<PORT>
```

The reported plan includes the normalized source precondition and marks Edge
as read-only. Drift, global scope, missing Edge and local bypass execute the
native full graph without Edge; an HTTP failure executes the selected graph
locally. This is a bounded POC replay of existing evidence, not a general
remote-cache policy.

To audit how the retained profile was derived, run the read-only discovery
command with the checked manifest, graph, generated state, terminal matrix
summary, qualifying cell evidence, and profile contract. Review the embedded
profile and every enabled/disabled explanation; do not copy a profile when the
decision is `NATIVE_FULL_GRAPH`.

```bash
buildopt profile discover \
  --manifest buildopt-impact-manifest.json \
  --graph buildopt-impact-graph.generated.json \
  --generated-manifest buildopt-impact.generated.json \
  --matrix-summary path/to/summary.json \
  --cell-evidence path/to/qualified-cell.json \
  --profile-contract path/to/profile-contract.json
```

Run `./dev/check-build-impact-automatic` for the bounded synthetic example and
`./dev/run --toolchain temurin-jdk-21 -- ./dev/check-build-impact-poc-onboarding`
for the installed candidate/fallback proof. Read
[Automatic Build Impact](../../specs/build-impact-automatic-v1.md),
[Build Impact POC onboarding](../../specs/build-impact-poc-onboarding-v1.md),
and the [generic measurement contract](../../specs/poc-generic-measurement-v1.md).

The later adaptive-fragment successor is not an onboarding workflow. Its
current 100-pair campaign closed as `STOP_ADAPTIVE_FRAGMENT_POC` after zero of
71 eligible builds activated a fragment and all generic value gates failed.
Do not configure or copy its research state into a customer repository; the
specifications and scorecard are retained for audit and future materially
different hypotheses only.

## Edge Cache

Edge is optional. Validate its strict private configuration before serving:

```bash
buildopt-edge validate --config /absolute/private/edge.json
buildopt-edge serve --config /absolute/private/edge.json
buildopt-edge status --config /absolute/private/edge.json
```

Edge may return a committed object only under current revocation authority. An
offline write is visible only to its exact pending attempt. Shared remains the
only commit and collision authority. See the
[operations guide](./operations.md#edge-cache) and
[Edge runbook](../../runbooks/edge-cache.md).

## CI installation

The GitHub Action and GitLab component resolve a published native package,
verify its complete SHA-256 and then verify the packaged files before exposing
the binaries and Gradle integration. Pin the Action/component source revision
and set `version` when reproducible native bits are required. Continue with
[CI integration](./ci-integration.md).
