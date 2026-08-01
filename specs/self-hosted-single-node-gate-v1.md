# Self-hosted single-node gate v1

This contract closes `A2-G01` and the owner-operated `MVP-A2` proof of concept
only by composing the executable current-tree proofs for `A2-001..004`.
Configuration/storage preflight, reproducible signed installation, online
upgrade plus restart, and absent-target manual restore with mandatory authority
rotation must all pass in one invocation.

Run:

```bash
./dev/check-self-hosted-single-node-gate
```

Every constituent uses isolated synthetic private state and real packaged
BuildOpt binaries where the lifecycle crosses a release boundary. The gate
does not start or mutate the operator's systemd service, contact an external
repository, run the deferred eight-hour soak, or claim external validation,
HA, backup RPO/RTO, enterprise identity, or production readiness.
