# Private-beta operations v1

This specification closes `A1-005` by composing the already implemented
health/readiness, local alert, runner circuit-breaker, and recovery boundaries
into one operator-facing procedure for a `PRIVATE_BETA_ISOLATED` deployment.
It does not replace or weaken any of the component contracts.

## Operational boundary

The server is live before it is safe to serve cache traffic. Readiness and all
product routes remain closed until Shared reconciliation and signed authority
loading complete. The bounded loopback alert endpoint remains observable while
readiness is false. A signed authority change disables serving until the new
monotonic state is verified, and a runner-side flood, oversized object, or disk
pressure opens the per-slot circuit so the next Gradle invocation omits Shared
and continues through writable managed L1.

Operators use the independent local bypass and CI kill switch when the product
path itself may affect a build. They use the signed deployment manager for
rollback or uninstall and never repair SQLite, authority, or circuit state by
hand. The exact procedures and stop conditions are in
[`runbooks/private-beta-operations.md`](../runbooks/private-beta-operations.md).

## Recorded exercise

Run:

```bash
./dev/check-private-beta-operations
```

The composite check strictly validates the machine-readable contract and
runbook, then executes the real readiness/revocation, ten-class alert,
circuit-breaker/Gradle-preservation, and base recovery drills. Every child
exercise must leave the repository unchanged.

This closes only `A1-005`. The per-pilot deployment, external design-partner
operation, external paging integration, causal-benefit gate, exact eight-hour
soak, `A1-G02`, `A1-G06`, and the complete `OPS-001/A1` profile remain open.
