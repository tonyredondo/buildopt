# Hermetic helper spike v1

`SPIKE-HERMETIC-001` answers `SPK-003` as `UNAVAILABLE` on the supported Linux
x86-64 development runner.

## Probe result and boundary

The host exposes unprivileged user, mount, PID, and network namespaces and
advertises seccomp actions. It does not expose a Landlock securityfs
interface or delegated cgroup. Availability alone is not enforcement: the
prototype intentionally installs no partial policy.

The complete required coverage vector cannot be established:

- filesystem, process-tree, and network namespaces are probed but not backed
  by installed mount/seccomp/Landlock/cgroup policy;
- environment remains unmediated;
- clock access may use vDSO;
- randomness may use `getrandom`.

Consequently `traceComplete=false`, qualification is `UNAVAILABLE`, and the
helper cannot create a `HERMETIC_PRODUCER_PROFILE`.

## Task-specific producer

The helper accepts only a closed manifest naming one `DEDICATED_TASK`
producer, exact task/producer identities, one canonical command beneath its
workspace, read-only input, and disjoint writable output/temporary
directories. Network, clock, and randomness must all request `DENY`.

Even a valid manifest does not override the capability probe. Incomplete
coverage prevents command execution, discards the candidate, and aborts
pending publication. The same producer is then run only through an
uninstrumented baseline, whose exit code and output remain authoritative.
A complete Gradle-invocation sandbox would still be build-level evidence and
would not satisfy this task-specific contract.

Run:

```bash
./dev/check-hermetic-helper-spike
```

This result closes the bounded spike and the hermetic qualification route as
`UNAVAILABLE`; it does not block official contracts, reviewed adapters, or
source patches in C1.
