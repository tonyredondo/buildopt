# `buildopt`

Go binary installed on CI runners and workstations.

Its boundary includes the launcher, neutral measurement envelope, Local Verifying Cache Gateway, and bypass to the original command.

## Passthrough execution

The initial executable surface is:

```bash
buildopt run -- <command> [args...]
```

The delimiter is mandatory. `buildopt` starts the command directly without a shell, so every token after `--` remains a distinct child-process argument. The child inherits the launcher's working directory, environment, standard input, standard output, and standard error. An ordinary child exit status, including a non-zero status, is returned unchanged.


## Packaged Gradle execution

Installed packages add the user-facing shortcut:

```bash
buildopt gradle [gradle args...]
```

From a Gradle repository root, it discovers the platform Wrapper, resolves the
init script and plugin relative to the installed launcher, and delegates to the
same passthrough/gateway lifecycle. `BUILDOPT_BYPASS=1` keeps Wrapper discovery
but omits the packaged BuildOpt integration. Missing Wrapper or package assets
return configuration exit code `78` before a child starts.
The CLI uses these status conventions; once a child starts, its ordinary exit status always wins:

| Code | Meaning |
|---:|---|
| `0` | Help was printed, or the child completed successfully |
| `64` | Invalid CLI usage |
| `78` | Gradle Wrapper or packaged integration is unavailable |
| `126` | The resolved command could not be executed |
| `127` | The command could not be found |
| `128 + signal` | The child terminated from an unhandled signal; for example, `130` for `SIGINT` and `143` for `SIGTERM` |

Run the real-binary integration suite with:

```bash
./dev/check-buildopt-cli
```

## Explicit Build Impact POC candidate

The installed launcher also exposes the owner-operated candidate path:

```bash
buildopt impact \
  --repository-id owner/repository \
  --pipeline-class pull-request \
  --changes-file .buildopt-changes \
  --gradle-option=--no-daemon
```

The command validates the repository-owned manifest, generated graph and exact
generated binding before evaluating the changed paths. It executes only a
manifest-enumerated alternative; unknown/global changes retain the original
entrypoints. This explicit POC command is not `BIA-002` production promotion
and never selects or removes Test Optimization-owned checks.

## Qualified POC profile

Once a repository has reviewed and committed the Build Impact inputs, the
short recommended POC command is:

```bash
buildopt poc --changes-file .buildopt-changes
```

It reads `buildopt-qualified-profile.json` by default. Before Gradle starts it
prints one machine-readable plan containing the selected/full graph,
entrypoints, expected outputs, preserved Test-owned checks, enabled exact
adapters and disabled mechanisms. Only Build Impact and the standard `Jar`
adapter are accepted. Safe Cache, Runtime Tuning, Hot State, standard `Copy`,
Shared/Edge and session integration are excluded from this command. A fallback
uses the manifest's original entrypoints without the adapter. This remains an
explicit owner-operated POC, not production authorization.

The suite covers empty, quoted, whitespace, wildcard, Unicode, newline, and literal `--` arguments; inherited cwd, environment, and standard streams; success and non-zero child statuses; usage; launch failures; process-group isolation; signal forwarding through a child process tree; and the local bypass described below.

## Process and signal handling

On Linux and macOS, the child becomes the leader of a process group distinct
from `buildopt`. While that child is active, the launcher catches `SIGINT` and
`SIGTERM` and forwards the same signal to the complete child group, including
descendants that remain in it. On Windows, the launcher creates a new process
group and assigns it to a Job Object configured to terminate the complete tree
when the launcher is cancelled or exits unexpectedly.

The launcher then waits for the child to finish and returns the child's ordinary exit status unchanged. It does not impose its own termination deadline or escalate to `SIGKILL`; the CI provider retains control of its grace period and final escalation. If the child does not handle the forwarded signal, the CLI returns the conventional `128 + signal` status without adding a launcher diagnostic to the child's standard error.

The Linux integration fixture verifies an independent process group, a nested
descendant, exact `SIGINT` and `SIGTERM` delivery, delayed cleanup during
cancellation, preserved handled exit statuses, and conventional
unhandled-signal status. The native macOS and Windows workflow additionally
verifies descendant cleanup through their platform contracts.

## Local bypass

Set the launcher-owned bypass to the exact value `1` when the product path must
be removed from an invocation immediately:

```bash
BUILDOPT_BYPASS=1 buildopt run -- ./gradlew build
```

The bypass is evaluated before server/export configuration, gateway startup,
or plugin-handshake startup. The child still uses the WS-001/WS-002 execution
path, but every reserved launcher variable—including `BUILDOPT_BYPASS`
itself—is removed from its environment. The original argv, standard streams,
working directory, process-group signal behavior, and exit status remain
authoritative.

Unset the variable to restore the normal path; only `1` activates bypass. See
the [Phase 0 base recovery runbook](../../runbooks/base-recovery.md) for the CI
kill switch, rollback, and uninstall procedures, and execute its recorded
drills with:

```bash
./dev/check-base-runbooks
```

## Gradle plugin handshake

Before starting the child, the launcher creates a private temporary directory,
a local authenticated event endpoint, and a fresh attempt ID. Linux uses a
current-user-verified Unix socket; macOS and Windows use an
operating-system-assigned `127.0.0.1` TCP endpoint whose mandatory random
credential is verified before any event frame. It replaces any inherited
values of the reserved `BUILDOPT_PLUGIN_*` variables before exposing
invocation-only context to the child.

When the packaged `dev.buildopt` Gradle plugin is present, the launcher accepts
exactly one version-1 `ProducerHello`, validates its attempt and producer
identity, and returns the matching acknowledgement. The socket and temporary
directory are removed after the child exits. A successful handshake or a
protocol failure is reported on standard error, but handshake setup, absence,
or rejection never replaces the child process exit status.

Run the real Gradle Wrapper fixture with:

```bash
./dev/check-gradle-plugin-handshake
```

The checker proves a fresh handshake after Configuration Cache reuse with an
up-to-date task, byte-identical output with and without the plugin, fail-open
behavior when the receiver is missing, and preservation of an intentional
Gradle failure.

## Authenticated local rendezvous

The launcher now starts a neutral HTTP gateway on an operating-system-assigned
`127.0.0.1` port before the child. It injects a minimum-scope Basic credential
and opaque `gatewayConnectionGeneration`; the credential authorizes only the
local readiness endpoint and is never an upstream token. Cache data routes are
deliberately absent in this block.

The plugin requires the complete gateway and event context. It first performs
an authenticated readiness request and verifies the returned generation. It
then connects to the private local endpoint, sends a fixed authentication
preface containing a fresh 256-bit token, and only afterward sends the
length-delimited `ProducerHello`. Linux additionally requires the peer process
to have the launcher's effective user ID. Credentials never enter the Protobuf
payload or diagnostics.

All reserved parent values are removed and the complete context is injected
atomically only when both local services are ready. The gateway and socket are
closed after the child exits, while their setup, authentication, and shutdown
failures remain unable to replace the child status.

Run the restart/concurrency and real Wrapper fixtures with:

```bash
./dev/check-local-gateway
./dev/check-gradle-plugin-handshake
```

The gateway fixture preserves endpoint, credential, and generation across a
restart, rejects cross-slot credentials, and serves concurrent slots without
identity overlap. The Wrapper fixture proves a fresh authenticated rendezvous
after Configuration Cache reuse.

Cache GET/PUT behavior, upstream credentials, retained task-event streaming,
and every optimization remain inactive. The first local JSON export is
described below.

## Managed runner-slot lifecycle

The internal pilot can move the neutral gateway onto its managed process path
by supplying both an absolute private state root and a runner-slot identity:

```bash
BUILDOPT_GATEWAY_STATE_ROOT=/run/user/1000/buildopt \
BUILDOPT_RUNNER_SLOT=runner-01.internal \
buildopt run -- ./gradlew build
```

The orchestrator must include the trust domain in the slot identity until
authenticated policy owns that dimension. Slot identifiers contain 1–128
ASCII letters, digits, dots, underscores, or hyphens and begin with a letter or
digit. Both variables are launcher-only and are removed from the child. With
both absent, the Phase 0 per-invocation gateway remains the compatibility path;
an incomplete or invalid configuration falls back to the original command
without exposing partial rendezvous context.

The launcher creates mode-`0700`, current-user-owned state directories, hashes
the slot into its on-disk name, and takes a non-blocking exclusive invocation
lease. A second invocation cannot reuse an active slot: it receives no gateway
or plugin context and runs on the baseline path. Concurrent builds must use
different slot identities.

For an available slot, `buildopt` starts or reconnects a detached gateway
process and registers the fresh attempt over a Linux abstract Unix socket.
Both peers verify `SO_PEERCRED` ownership before exchanging a bounded,
versioned registration. The gateway serves readiness only while that
connection is current; an authenticated request between invocations returns
`503`. The original neutral A0-001 registration leaves cache data paths at
`404`; A0-006 supplies an invocation-only authenticated cache binding.

The loopback address, local-only Basic credential, and
`gatewayConnectionGeneration` are atomically stored in a mode-`0600` state
file. A process restart rebinds and reuses the complete identity. If the
address cannot be recovered, the gateway rotates endpoint, credential, and
generation together before admitting the invocation. The detached process
exits after five idle minutes by default; `BUILDOPT_GATEWAY_IDLE_TIMEOUT` can
set a bounded `100ms`–`24h` internal-pilot lifetime and is also removed from the
child. Removing the state root deliberately rotates the identity and must be
done only while its slots are idle.

Validate the real multi-process lifecycle with:

```bash
./dev/check-managed-gateway
```

This path establishes lifecycle and trust boundaries only. L1/L2 cache
behavior, upstream credentials, authenticated policy, revocation, and pending
publication remain owned by their later MVP-A0 blocks. The completed
cross-component restart/rotation and concurrent authenticated-binding gate is
`./dev/check-gateway-rotation`.

## Managed native L1 cache

The launcher can prepare one private native `DirectoryBuildCache` for a
complete scope and security generation:

```bash
BUILDOPT_L1_STATE_ROOT=/run/user/1000/buildopt \
BUILDOPT_L1_TENANT_ID=tenant-7 \
BUILDOPT_L1_REPOSITORY_ID=repository-42 \
BUILDOPT_L1_TRUST_DOMAIN=private-beta \
BUILDOPT_L1_COMPATIBILITY_CLASS=gradle-9.6-java-21-linux-amd64 \
BUILDOPT_L1_SECURITY_GENERATION=42 \
BUILDOPT_L1_L2_WRITE_AUTHORIZED=0 \
buildopt run -- ./gradlew --build-cache build
```

The matching init script must apply `dev.buildopt.managed-l1` from
`beforeSettings`. Raw scope fields remain launcher-only. The child receives an
opaque, generation-segmented directory plus a fixed seven-day native retention
contract; all launcher-owned directories are current-user-owned mode `0700`.
The launcher holds a mode-`0600`, non-blocking exclusive lease for the exact
scope and generation until Gradle exits.

Set `BUILDOPT_L1_L2_WRITE_AUTHORIZED=1` only for an invocation already
authorized by the later L2 policy. That mode exposes no local directory and
the settings plugin disables local load/store, so an aborted pending write
cannot leave a reusable local hit. Incomplete configuration, an unsafe state
root, or an occupied lease produces a diagnostic and preserves the Gradle
baseline. The native-only input remains available without Shared; when an
A0-006 authority is configured, its signed monotonic generation and write
permission replace the parent-supplied generation and writer flag.

Validate the launcher, plugin, Gradle/JDK/DSL matrix, generation rotation, and
real end-to-end handoff with:

```bash
./dev/check-managed-l1
```

## Locally authenticated Shared Cache

The launcher enables the Shared route only from a complete private local
authority configuration:

```bash
BUILDOPT_LOCAL_AUTHORITY_PATH=/run/buildopt/authority.json \
BUILDOPT_LOCAL_TRUST_ROOT_PATH=/run/buildopt/trust-root.json \
BUILDOPT_LOCAL_CACHE_CREDENTIAL_PATH=/run/buildopt/cache-credential \
BUILDOPT_SHARED_CACHE_URL=http://127.0.0.1:8042 \
BUILDOPT_L1_STATE_ROOT=/run/user/1000/buildopt \
BUILDOPT_L1_TENANT_ID=tenant-7 \
BUILDOPT_L1_REPOSITORY_ID=repository-42 \
BUILDOPT_L1_TRUST_DOMAIN=private-beta \
BUILDOPT_L1_COMPATIBILITY_CLASS=gradle-9.6-java-21-linux-amd64 \
CI=true \
buildopt run -- ./gradlew --build-cache build
```

All authority files must be absolute, current-user-owned mode-`0600` regular
files. The launcher verifies canonical JCS, Ed25519 signatures, repository and
component binding, expiration, and durable anti-rollback state before starting
Gradle. It sends the upstream credential only to the gateway's same-UID
control channel. Gradle receives a distinct local Basic credential plus
non-secret authority/configuration generation markers; the managed settings
plugin configures `HttpBuildCache` at the loopback gateway. Invalid or partial
authority disables the authenticated cache for that invocation without
changing the child exit status.

For `A1-002`, set `BUILDOPT_SHARED_CACHE_TOKEN_PATH` to a separate absolute,
current-user-owned mode-`0600` file containing the provisioned opaque remote
token. The signed local credential continues to bind policy and attempt, while
the gateway substitutes only the remote token on upstream requests and removes
both token paths before Gradle starts. `BUILDOPT_SHARED_CACHE_URL` may use
canonical HTTPS for the beta backend; plaintext HTTP remains accepted only for
canonical `127.0.0.1` loopback.

Validate this complete boundary with:

```bash
./dev/check-local-authority
```

## Server ingest

When both `BUILDOPT_SERVER_URL` and `BUILDOPT_SERVER_INGEST_TOKEN` are present,
the launcher captures the neutral session boundaries and child outcome, then
asks the active local gateway to deliver one provisional internal record to
`buildopt-server`. The walking skeleton accepts only a validated
`http://127.0.0.1:<port>` server endpoint, disables proxies and redirects, uses
a Bearer token plus the session ID as the idempotency key, and binds the record
to the active `gatewayConnectionGeneration`.

The server URL and token are launcher-only inputs. They are removed from the
child environment and never enter Gradle, logs, or the session body. Missing,
invalid, rejected, or unavailable server ingest remains fail-open and never
replaces the child exit code. With both variables absent, local bypass does not
contact the server.

Run the real-binary ingest fixture with:

```bash
./dev/check-session-ingest
```

Without additional context this block carries only session identity, gateway
generation, neutral timestamps/duration, outcome, and exit code.

## BUILD_SESSION export context

`BUILDOPT_BUILD_SESSION_CONTEXT` may predeclare the non-secret, tokenized facts
that cannot be inferred safely from the process:

```json
{
  "repositoryId": "repository-42",
  "revision": "revision-abc",
  "requestedTasks": ["build"],
  "sourceStateDigest": "hmac-sha256:<64-lowercase-hex>",
  "workUnitsFingerprint": "hmac-sha256:<64-lowercase-hex>",
  "tokenKeyVersion": "local-token-v1",
  "trustDomain": "developer-local"
}
```

The context is strict JSON, limited to 32 KiB, parsed before the outcome, and
removed from the child environment. It contains no HMAC key or source content.
The current walking skeleton exports only when this context, valid server
configuration, and one authenticated Gradle `ProducerHello` are all present.
The launcher then attaches the producer identity/version and exact child
process interval to the provisional ingest record.

`buildopt-server --export-dir` derives requested-work and empty-deliverables
manifest digests, preserves neutral-envelope timestamps and monotonic
durations, and writes the immutable `BUILD_SESSION v1`. Invalid or incomplete
context and export failure remain fail-open and cannot replace the Gradle exit
code.

Validate real Gradle success/failure exports against the normative schema with:

```bash
./dev/check-build-session-export
```

Durable storage, JSONL, retries/spooling, cancellation and infrastructure
classification, CI workload metadata, cache payloads, and optimization remain
inactive.
