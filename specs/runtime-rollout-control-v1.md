# Runtime rollout control v1

This POC contract closes `B-008`, the intentional isolation-contamination gate
`B-G02`, and the permanent-control/revalidation gate `B-G06`. It does not
activate an owner repository or replace the still-required owner-operated A/A
and causal performance evidence.

## Learning budget

Additional validation reserves runner milliseconds before scheduling. The
default hard ceilings are 5% of eligible natural runner time in the preceding
seven days and 10% in the preceding 24 hours, with no borrowing and one active
reservation per repository. A repository may only reduce either percentage,
including to zero. Natural candidate/control assignment consumes no additional
budget. Completion charges actual time; cancellation releases the reservation.
Exhaustion rejects the reservation without relaxing any gate.

## Progressive rollout

Direct reversible actions advance exactly `5 → 25 → 50 → 95%` after their
smoke gate. Proof-gated actions advance exactly
`shadow → 1 → 5 → 25 → 50 → 95%` and cannot leave shadow without complete
contract and quarantine evidence. Every transition also requires correctness,
sample, budget, complete telemetry, and revalidation evidence. No call advances
more than one stage.

At every visible stage, the candidate receives exactly its declared integer
basis-point probability. The remaining probability selects
`STABLE_CONTROL`; the first 500 basis points are permanently labeled control.
The 95% terminal stage is therefore exactly 95% candidate and 5% control.
Local, release, urgent, external-effect, malformed, or undeclared-arm contexts
always select the conservative profile.

## Suspension, fallback, and rollback

Incomplete telemetry, drift, artifact divergence, attributable failure, OOM,
sustained swapping, p95 regression, or queue regression immediately suspends
the action and sets candidate probability to zero. Explicit rollback is
terminal for that generation. Candidate/control namespaces continue to reject
any stable write target.

Before task actions start, a failed candidate may retry immutable original argv
once. After task actions start, only an already isolated baseline or a manifest
that guarantees no effects permits baseline execution. Otherwise the failure is
preserved, the action is suspended, and a signed kill switch is required for
subsequent builds.

The repository kill switch is Ed25519-authenticated against pinned keys,
time-bounded, and strictly monotonic by generation. Enabling it durably
suspends every active action before another selection can return a candidate.
A later signed disable does not reactivate actions; fresh revalidation is
required.

## Conformance

Run:

```bash
./dev/check-runtime-rollout-control
```

The checker composes the existing CI budget and validation-isolation contracts
with focused race tests for daily/weekly boundaries, zero budget, concurrency,
stage gates, exhaustive 95/5 selection, exclusions, fallback, explicit
rollback, all suspension causes, intentional stable contamination, signature
tampering, stale generations, and durable kill-switch reopen.
