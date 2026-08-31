# Troubleshooting

## Normalization-aware evidence

If the compiler checker cannot find a global Go installation, use the checked
toolchain path already embedded through `./dev/run --toolchain go --`. A public
Gradle failure must be classified separately from an intentionally interrupted
bounded run: the NAC v2 terminal ledger records the latter as incomplete, not
as a product failure or a zero-valued timing result.

Start with the narrowest symptom. Preserve the original Gradle command, exit
status, relevant non-secret diagnostics, exact Git SHA, platform, and
`buildopt doctor` output. Never include tokens, credentials, raw authority
secrets, customer source, cache blobs, or unredacted exports in an issue.

## `dev/doctor` reports `FAIL`

`dev/doctor` inspects host prerequisites for source development. Read the
failing `command:<name>` row and install that host command through the
workstation's normal administration process. It intentionally does not use
`sudo` or modify global tools.

Provision missing project-local tools with `./dev/bootstrap --toolchain <id>`.
Do not replace `dev/toolchains.lock.yaml` checksums with locally available
versions.

## `dev/run` says a toolchain is unavailable

Provision the exact requested toolchain first:

```bash
./dev/bootstrap --toolchain go
./dev/run --toolchain go -- go version
```

If provisioning fails, retain the URL, expected digest, actual error, platform,
and `./dev/doctor --json` report. Do not copy another machine's `.tools/`
directory or bypass checksum verification.

## `buildopt run` prints usage

The delimiter is mandatory:

```bash
buildopt run -- ./gradlew build
```

Arguments after `--` are passed directly. Shell operators such as pipes,
redirection, and wildcard expansion are not interpreted by BuildOpt; invoke a
reviewed shell explicitly only when that is genuinely the original command.


## `buildopt gradle` cannot find its setup

Run it from the repository root containing `gradlew` or `gradlew.bat`. A native
installation owns the init script and plugin under its `share/buildopt`
directory; reinstall the package if either is missing. CI integrations expose
equivalent verified paths automatically. Manual
`BUILDOPT_GRADLE_INIT_SCRIPT` and `BUILDOPT_GRADLE_PLUGIN_JAR` overrides are
developer/legacy escape hatches and must be set together.
## Gradle succeeds but BuildOpt reports no handshake

The launcher remains fail-open, so the build can still succeed. Verify that:

- the packaged init script is passed to Gradle;
- `BUILDOPT_GRADLE_PLUGIN_JAR` points to the matching packaged JAR;
- the launcher and plugin came from the same verified release;
- inherited `BUILDOPT_PLUGIN_*` values are not being relied upon;
- Configuration Cache did not preserve a manually constructed stale context.

Run `./dev/check-gradle-plugin-handshake` in the BuildOpt checkout. Do not make
the plugin accept an unauthenticated or mismatched attempt to hide the warning.

## Managed gateway or L1 falls back to baseline

Common causes are an incomplete environment group, a relative/unsafe state
root, an invalid runner slot, an occupied lease, ownership/permission mismatch,
or a stale security generation. Inspect non-secret diagnostics and compare the
complete group with [configuration reference](./reference/configuration.md).

Do not break a lease or delete state while another invocation may be active.
Use different runner-slot identities for concurrent builds.

## Shared cache always misses

A safe miss can mean the key is absent, pending, aborted, expired, quarantined,
revoked, corrupt, outside scope, or bound to another generation. Check:

1. server `/readyz` is `200`;
2. authority and trust root verify with `buildopt-server authority inspect`;
3. the repository, namespace, plane, operation, compatibility class, and
   generations match;
4. the remote token is current and has the required read/write scope;
5. the data root is on a supported local filesystem;
6. no circuit breaker is suppressing L2.

Never turn a miss into a hit by editing SQLite, copying a blob, deleting
anti-rollback state, or loosening checksum/revocation checks.

## Server is live but not ready

This is expected during reconciliation. Inspect:

```bash
curl --fail --head http://127.0.0.1:8042/livez
curl --head http://127.0.0.1:8042/readyz
curl --fail http://127.0.0.1:8042/ops/v1/alerts | jq .
```

Keep new builds on bypass or baseline until readiness is `200`. Use the firing
alert class and [private-beta operations runbook](../runbooks/private-beta-operations.md)
for containment. Do not route around readiness.

## Dashboard returns `401`, `404`, or no data

- `401`: use the independent history read token, not the ingest token.
- `404`: the history token was omitted at startup or the route is disabled.
- Empty list: verify immutable exports exist under the configured private root
  and match the selected redacted repository/outcome filters.
- A search result is missing: client search covers loaded rows only; use API
  filters/pagination for the complete result set.

The token stays in browser memory; do not add it to the URL, source, local
storage, or static assets.

## Build Impact generated state drifts

Run `buildopt-impact generate` with the exact repository ID, pipeline class,
manifest, graph path, generated-manifest path, Gradle command, and compatible
Wrapper used by CI. Review both generated files and commit them only if the
declared graph change is expected.

Unsupported task types, included builds, test-bearing alternatives, unknown
paths, and incomplete discovery intentionally retain `FULL_GRAPH`. Do not
loosen the checker merely to select less work.

## Patch Autopilot rejects a bundle

Treat rejection as repository-unchanged safety behavior. Common reasons are an
untrusted/expired signature, unsupported recipe, base or source-state drift,
unsafe path/symlink/submodule, wrong preimage or mode, conflicting action
branch, or failed relevant validation.

Regenerate a bundle from the current authorized source and recipe. Do not use
fuzzy patching, rebase, force-push, hook execution, or manual branch recovery
to bypass the exact contract. Use
[`runbooks/base-recovery.md`](../runbooks/base-recovery.md) for a partial
delivery.

## Edge reports `NOT_READY` or stale status

Run `buildopt-edge validate --config ...`, then verify the serving process,
status file freshness, Shared reachability, private paths, storage capacity,
and complete current signed authority set. During multi-file authority
replacement, temporary `503`/`NOT_READY` is correct.

Never delete Edge anti-rollback state. Publish a complete higher signed
generation atomically through the owning configuration mechanism.

## macOS or Windows behavior differs from Linux

Run `buildopt doctor` and compare the reported process, resource, storage, and
service capabilities. Native equivalents deliberately differ: process groups
versus Job Objects, `flock` versus `LockFileEx`, launchd versus SCM, and
platform-specific local-filesystem proof.

Run `./dev/check-platform-compatibility` for cross-build/package inventory and
inspect the hosted native workflow for actual macOS/Windows execution. Do not
claim native behavior from a Linux cross-build alone.

## CI fails only on cancellation

Inspect the exact failing SHA and native job logs. Cancellation paths must wait
for descendant cleanup and preserve the child outcome contract. Do not add a
short launcher-owned timeout or detach a process to make the job finish. The CI
provider owns final escalation.

## Immediate safe recovery

For one invocation:

```bash
BUILDOPT_BYPASS=1 buildopt run -- ./gradlew build
```

For broader incidents, follow the [base recovery runbook](../runbooks/base-recovery.md)
and preserve state/log evidence before rollback or removal.

## NAC v2 source classification differs locally

Run `git cat-file -t <frozen-revision>` in each subject repository and rerun
`./dev/check-normalization-aware-cacheability-source-classification SOURCE_ROOT`.
A missing revision, changed source digest or byte-different report is source
drift and must fail closed. Do not substitute a DNO report, patch public source
or run Gradle to repair a source-classification mismatch.
