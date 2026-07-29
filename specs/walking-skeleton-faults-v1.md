# Walking-skeleton fault contract v1

**Status:** normative Phase 0 fixture contract
**Tracker item:** `WS-008`
**Decision dependencies:** `GOLDEN-LANE-001`, `F0-039`

## Scope

This contract fixes the failure semantics of the optimization-off walking
skeleton. It covers the launcher, its child process group, the private plugin
attempt socket, the loopback gateway, and the provisional session handoff. It
does not create the managed-cache attempt or lease protocols owned by A0.

The active walking skeleton has no cache data route and cannot create a cache
lease. Its only attempt-scoped resource is the invocation-owned plugin
rendezvous. Consequently, the expected active cache-attempt and lease counts
are zero before, during, and after every fixture. The checker must not invent a
durable cache lifecycle merely to satisfy this pre-cache gate.

## Required cases

The same locked launcher and server binaries used by the golden lane execute
these cases:

| Case | Child result | Session result | Required cleanup |
|---|---:|---|---|
| ordinary failure | exit `37` | `BUILD_FAILURE`, exit `37` | plugin socket and gateway closed |
| handled cancellation | receives `SIGTERM`, cleans descendants, exits `42` | `CANCELLED`, exit `42` | child process group gone; plugin directory/socket and gateway closed |
| local bypass | exit `38` | no session and no control-plane contact | no product rendezvous created |

The launcher forwards `SIGINT` and `SIGTERM` to the isolated child process
group and waits for its cleanup. A forwarded cancellation signal determines
the session class even when the child handles the signal and returns its own
nonzero cleanup status. An unhandled signal retains the conventional
`128 + signal` process status and is also `CANCELLED`.

The launcher must preserve the child's status in every case. Session delivery,
shutdown diagnostics, and missing product services remain fail-open for the
build result.

## Completion evidence

[`dev/check-walking-skeleton-faults`](../dev/check-walking-skeleton-faults)
must pass both on the development host and inside the digest-pinned strict
4-CPU/16-GiB golden container. It observes the live socket, gateway, and
process tree before cancellation, then proves that all are absent afterward.
The server log must contain exactly one `BUILD_FAILURE` and one `CANCELLED`
record, while the bypass contributes no record and no credential appears in
diagnostics.
