# Operational runbooks

This directory contains operator-facing recovery procedures that are executable
or paired with executable exercises. The runbooks describe only capabilities
that exist in the current repository state; later deployment and control-plane
work extends them without silently changing the Phase 0 safety boundary.

- [`base-recovery.md`](./base-recovery.md): local bypass, CI kill switch,
  immutable version rollback, uninstall, state preservation or purge, and
  partial patch-branch recovery for `F0-039`.
- [`private-beta-operations.md`](./private-beta-operations.md): preflight,
  health/readiness, alert triage, revocation, circuit fallback, shutdown,
  restart, rollback, and recovery for `A1-005`.
- [`self-hosted-single-node.md`](./self-hosted-single-node.md): signed install,
  private inputs, explicit systemd activation, admission, and removal for
  `A2-002`.

Run the recorded exercises from the repository root:

```bash
./dev/check-base-runbooks
./dev/check-private-beta-operations
./dev/check-self-hosted-service-install
```
