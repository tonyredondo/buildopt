# Operational runbooks

This directory contains operator-facing recovery procedures that are executable
or paired with executable exercises. The runbooks describe only capabilities
that exist in the current repository state; later deployment and control-plane
work extends them without silently changing the Phase 0 safety boundary.

- [`base-recovery.md`](./base-recovery.md): local bypass, CI kill switch,
  immutable version rollback, uninstall, state preservation or purge, and
  partial patch-branch recovery for `F0-039`.

Run the recorded Phase 0 exercises from the repository root:

```bash
./dev/check-base-runbooks
```
