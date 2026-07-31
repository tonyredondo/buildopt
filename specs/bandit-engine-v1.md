# Bounded bandit engine v1

This POC contract closes `B-007` and the finite-safe-arm gate `B-G05`. It turns
the Phase 0 replay policy into a private durable runtime engine; no repository
or action is activated by this implementation.

## Exact bucket and readiness

Samples are keyed by repository, measurement epoch, policy version, catalog
version, and pre-outcome feature-bucket digest. A different epoch or bucket
starts empty, while an explicit measurement/drift reset returns to `FIXED_AA`.
The engine enters `BANDIT` only after every eligible candidate has at least 20
valid outcomes in that exact bucket and the A/A sample-ratio status is `VALID`.

Fixed-cohort outcomes enter through their complete B-006 assignment record, so
the arm, seed-derived random point, propensity, policy, epoch, and bucket are
revalidated before learning. Bandit assignments bind the same identity and are
atomically stored in a mode-`0600` file before being returned.

## Selection and propensity

The selector accepts only `STABLE_CONTROL`, `W2_H3G`, `W3_H4G`, and `W4_H6G`.
Stable control owns a permanent 500-basis-point floor. Exploration is bounded
to 200–1,000 basis points and split exactly across eligible candidates; any
integer remainder is assigned in catalog order. The remaining probability is
assigned to the greedy arm. The recorded propensity is the sum of all regions
that can choose the selected arm.

Prediction uses the 10% trimmed mean and five control pseudo-observations.
Strict ties retain control. The conformance test exhaustively selects all
10,000 random points and verifies that observed arm counts equal every recorded
propensity.

## Exactly-once outcomes

An outcome must match the persisted assignment's exact bucket and propensity,
carry complete nonnegative reward components, and arrive from assignment time
through 24 hours inclusive. The reward is baseline customer-visible time minus
observed time and the five explicit penalties. An assignment is terminal after
its first outcome disposition: duplicates, late, partial, missing-propensity,
or cross-binding records cannot update learning. A guardrail result is
`SUSPENDED_ROLLBACK` and never contributes a reward.

## Conformance

Run:

```bash
./dev/check-bandit-engine
```

The checker composes the original 15-scenario replay with the production
selector, durable reopen, exact propensity, 20-sample entry, era reset,
fixed-cohort ingestion, delayed/duplicate/partial outcomes, and safe-arm tests.
