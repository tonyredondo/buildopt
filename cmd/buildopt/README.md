# `buildopt`

Go binary installed on CI runners and workstations.

Its boundary includes the launcher, neutral measurement envelope, Local Verifying Cache Gateway, and bypass to the original command.

## WS-001 passthrough

The initial executable surface is:

```bash
buildopt run -- <command> [args...]
```

The delimiter is mandatory. `buildopt` starts the command directly without a shell, so every token after `--` remains a distinct child-process argument. The child inherits the launcher's working directory, environment, standard input, standard output, and standard error. An ordinary child exit status, including a non-zero status, is returned unchanged.

The CLI uses these status conventions; once a child starts, its ordinary exit status always wins:

| Code | Meaning |
|---:|---|
| `0` | Help was printed, or the child completed successfully |
| `64` | Invalid CLI usage |
| `126` | The resolved command could not be executed |
| `127` | The command could not be found |
| `128 + signal` | The child terminated from an unhandled signal; for example, `130` for `SIGINT` and `143` for `SIGTERM` |

Run the real-binary integration suite with:

```bash
./dev/check-buildopt-cli
```

The suite covers empty, quoted, whitespace, wildcard, Unicode, newline, and literal `--` arguments; inherited cwd, environment, and standard streams; success and non-zero child statuses; usage; launch failures; process-group isolation; and signal forwarding through a child process tree.

## WS-002 process and signal contract

On the Linux x86-64 acceptance platform, the child becomes the leader of a process group distinct from `buildopt`. While that child is active, the launcher catches `SIGINT` and `SIGTERM` and forwards the same signal to the complete child group, including descendants that remain in it.

The launcher then waits for the child to finish and returns the child's ordinary exit status unchanged. It does not impose its own termination deadline or escalate to `SIGKILL`; the CI provider retains control of its grace period and final escalation. If the child does not handle the forwarded signal, the CLI returns the conventional `128 + signal` status without adding a launcher diagnostic to the child's standard error.

The integration fixture verifies an independent process group, a nested descendant, exact `SIGINT` and `SIGTERM` delivery, delayed cleanup during cancellation, preserved handled exit statuses, and conventional unhandled-signal status. Platform expansion remains deferred until its own compatibility fixtures.

## WS-003 Gradle plugin handshake

Before starting the child, the launcher creates a private temporary directory,
a Unix socket, and a fresh attempt ID. It replaces any inherited values of the
reserved `BUILDOPT_PLUGIN_*` variables before exposing invocation-only context
to the child.

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

## WS-004 authenticated local rendezvous

The launcher now starts a neutral HTTP gateway on an operating-system-assigned
`127.0.0.1` port before the child. It injects a minimum-scope Basic credential
and opaque `gatewayConnectionGeneration`; the credential authorizes only the
local readiness endpoint and is never an upstream token. Cache data routes are
deliberately absent in this block.

The plugin requires the complete gateway and event context. It first performs
an authenticated readiness request and verifies the returned generation. It
then connects to the private Unix socket, sends a fixed authentication preface
containing a fresh 256-bit token, and only afterward sends the length-delimited
`ProducerHello`. The receiver also requires the peer process to have the
launcher's effective user ID. Credentials never enter the Protobuf payload or
diagnostics.

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
observability export, and every optimization remain inactive.

## WS-005 server ingest

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

This block carries only session identity, gateway generation, neutral
timestamps/duration, outcome, and exit code. `WS-006` owns construction and
schema validation of the complete immutable `BUILD_SESSION v1`; durable
storage, JSONL export, retries/spooling, cancellation classification, cache
payloads, and optimization remain inactive.
