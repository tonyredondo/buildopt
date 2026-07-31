# Private-beta isolated operations

This runbook is the operator procedure for `A1-005`. It composes only behavior
already exercised by the `PRIVATE_BETA_ISOLATED` profile. It does not claim a
specific pilot deployment, external paging, high availability, RPO/RTO, a
contractual SLO, the external causal-benefit gate, or the eight-hour soak.

## Preflight and startup

Before admitting a pilot build, record the exact release version, deployment
root, data root, runner-slot state root, server loopback address, signed
authority paths, token ID and scope, namespace generations, retention profile,
and any customer-controlled export destinations. One pilot deployment must
have its own private roots and repository/namespace-scoped tokens. Never reuse
another pilot's state or credentials.

Confirm the selected immutable deployment without printing credentials:

```bash
./dev/manage-deployment status --root /exact/private/deployment-root
```

Start `buildopt-server` through the pilot's supervisor with the explicit
`--state-dir`, authority, trust-root, credential, and token-authentication
configuration documented by `cmd/buildopt-server/README.md`. The listener must
remain canonical IPv4 loopback behind any separately managed TLS boundary.
Do not admit a build merely because the process is live.

## Health and readiness

Set `BUILDOPT_OPS_BASE_URL` to the exact loopback origin selected for this
deployment, for example `http://127.0.0.1:8042`, and inspect each surface:

```bash
curl --fail --silent --show-error --head "${BUILDOPT_OPS_BASE_URL:?}/livez"
curl --silent --show-error --head "${BUILDOPT_OPS_BASE_URL:?}/readyz"
curl --fail --silent --show-error "${BUILDOPT_OPS_BASE_URL:?}/ops/v1/alerts" | jq .
```

During reconciliation, `/livez` is `200`, `/readyz` and product routes are
`503`, and the alert endpoint remains available. Admit cache traffic only
after `/readyz` returns `200`. If readiness remains false, preserve the alert
snapshot and server logs, keep new builds on bypass or their normal Gradle
path, and follow the matching triage row. Never route around readiness.

## Alert triage

The local endpoint exposes aggregate state, not a paging integration. Record
the observation time and all firing classes before taking the smallest action:

| Firing class | Immediate containment | Recovery evidence |
|---|---|---|
| `DISK_QUOTA` | Stop adding load; keep affected builds on L1/bypass | Capacity is restored and the class clears |
| `CORRUPTION` | Preserve quarantine/audit state; do not delete blobs or SQLite by hand | Reconciliation completes and verified reads remain safe misses |
| `STUCK_ATTEMPT_OR_LEASE` | Drain new work; do not break leases manually | Expired work is reconciled and the class clears |
| `REVOCATION_LAG` | Keep readiness closed and publish only a higher valid signed authority | Propagation is below 60 seconds and the old route returns `401` |
| `POLICY_FRESHNESS` | Stop admitting new product-path invocations until current signed policy exists | Current unexpired policy is active and readiness returns `200` |
| `CIRCUIT_BREAKER` | Keep the cache route disabled while authority change is unresolved | Verified authority is active and the aggregate class clears |
| `SQLITE_CONTENTION` | Reduce load and verify the data root is on supported local storage | Bounded integrity/probe checks pass |
| `EXPORT_BACKLOG` | Preserve the export root and stop adding diagnostic volume | Pending export falls within the bounded limit |
| `ACCEPTANCE_ERROR_RATE` | Enable the CI kill switch if builds may be affected | A normal-path canary succeeds and the bounded window recovers |
| `ACCEPTANCE_LATENCY` | Enable the CI kill switch if latency exceeds the accepted budget | A normal-path canary and bounded p95 recover |

Escalate using the pilot's external incident channel when configured; this
repository does not claim or provision that paging channel.

## Authority revocation

Publish the complete authority, trust root, and credential update atomically
through the owning secret/configuration mechanism. The authority must carry a
higher monotonic policy or revocation generation and a valid signature. Expect
readiness and cache routing to close while the files are revalidated.

Within 60 seconds, confirm that the old authority returns `401`, the new
read-only authority reaches at most a safe `404` miss for an absent object, and
readiness returns `200`. If verification, freshness, or monotonicity fails,
leave readiness closed, activate bypass for new builds, and correct the signed
source. Never edit `control.sqlite` or persisted generation state.

## Circuit breaker

`FLOOD`, `OBJECT_TOO_LARGE`, and `DISK_PRESSURE` persist a secret-free,
mode-`0600` `circuit-breaker.json` below the exact managed runner slot. The
active L2 binding is removed immediately; during the five-minute cooldown the
next invocation omits Shared, uses writable managed L1, and must preserve the
Gradle result.

Do not delete or rewrite circuit state to force an early retry. Malformed state
fails closed by continuing to suppress L2. After the cooldown, the launcher
removes valid state durably before retrying Shared. If the fallback build is
not preserved, activate the local bypass/CI kill switch and collect the exact
reason, runner slot, invocation result, and non-sensitive alert snapshot.

## Bypass and kill switch

For one affected invocation, use the launcher-owned bypass:

```bash
BUILDOPT_BYPASS=1 buildopt run -- ./gradlew build
```

For every new CI invocation in the affected pilot, set the CI-owned
`BUILDOPT_EMERGENCY_BYPASS=1` mapping described in
[`base-recovery.md`](./base-recovery.md). Confirm the next build runs the
original Gradle argv without gateway, plugin handshake, or server ingest.
Bypass does not revoke an invocation that is already running.

## Shutdown and restart

Stop admission first and wait for active builds to finish. A graceful server
shutdown changes readiness to `503` before draining requests. Stop the process
through its supervisor and verify the Shared writer lease is released before
rollback, uninstall, or any approved data operation.

Restart with the same immutable release and exact configuration. Observe
`/livez` first, keep product routes closed during reconciliation, then require
`/readyz` `200` and no unexplained firing alert before admitting a canary.
Uncertain or unrecoverable control/key/monotonic state remains fail-closed and
requires the generation-rotation recovery owned by the deployment procedure;
never serve prior objects by bypassing reconciliation.

## Rollback and uninstall

Inspect and atomically select an already retained, reverified version without
changing persistent data:

```bash
./dev/manage-deployment status --root /exact/private/deployment-root
./dev/manage-deployment rollback \
  --root /exact/private/deployment-root \
  --key /trusted/buildopt-release.pub
```

Keep the kill switch active until the baseline build and one normal-path canary
produce the expected tasks, outputs, artifacts, and exit status. For removal,
follow the guarded preserve-by-default uninstall in
[`base-recovery.md`](./base-recovery.md). Data purge always requires a separate
explicit retention decision and an exact marked root.

## Recorded exercise

Run the bounded composite exercise from the repository root:

```bash
./dev/check-private-beta-operations
```

It validates this procedure and executes readiness/revocation, all ten local
alert classes, every runner circuit reason with real Kotlin and Groovy Gradle
fallback, and the base bypass/kill-switch/rollback/uninstall drills. It does
not run `./dev/check-beta-soak --qualify`.
