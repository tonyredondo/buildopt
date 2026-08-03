# Product workflows

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

BuildOpt composes Gradle's native local cache with an authenticated loopback
HTTP cache:

```text
Gradle -> managed DirectoryBuildCache (L1)
       -> Local Verifying Cache Gateway (L2 endpoint)
       -> optional Edge
       -> Shared
```

The full configuration consists of a private signed authority, trust root,
cache credential or scoped token, Shared URL, managed L1 scope, and compatible
Gradle init script. Treat it as one atomic configuration group; do not invent
partial values. The detailed environment contract is in
[configuration reference](../reference/configuration.md#managed-l1-and-shared-cache).

Reads require a committed object, matching scope and generation, current
revocation state, and complete-byte checksum verification. Writes remain
pending until the control path authorizes commit. Any unknown, invalid,
expired, conflicting, or unavailable state falls back to a miss or baseline.

## Runtime Optimizer

The runtime optimizer implements bounded policies for:

- Configuration Cache adoption;
- clean-task removal under an explicit workspace lifecycle;
- four resource profiles;
- allowlisted invocation merging and policy prefetch;
- fixed control cohorts and contextual bandit selection;
- budgets, canaries, rollback, and a kill switch.

These are policy/evaluation surfaces in the current POC, not arbitrary CLI
switches. Evidence remains isolated by repository, compatibility class,
policy, and cohort. Validate the complete implemented model with:

```bash
./dev/check-runtime-resource-profiles
./dev/check-runtime-rollout-control
./dev/check-runtime-validation-isolation
./dev/check-runtime-owner-evaluation
```

Read [the Runtime Optimizer section of the RFC](../../gradle-build-optimization-platform.md#14-resource-autotuning)
for the model and use the tracker to inspect exact evidence. A policy that
cannot be bound or validated does not activate.

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
```

See the [patcher README](../../jvm/patcher/README.md) and
[customer workflow specification](../../specs/customer-patch-workflow-v1.md).

## Build Impact

Build Impact can discover a conventional Gradle project graph while keeping
the repository-owned manifest as authority. Generate reviewable state:

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

Active selection remains separate from discovery. It requires current exact
digests, explicit enablement, and qualified validation evidence. Included
builds, test-bearing alternatives, unsupported task types, missing projects,
drift, unknown paths, and global changes use the original full entrypoints.
Test Optimization-owned checks are preserved separately.

Run `./dev/check-build-impact-automatic` for the bounded synthetic example and
read [Automatic Build Impact](../../specs/build-impact-automatic-v1.md).

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
