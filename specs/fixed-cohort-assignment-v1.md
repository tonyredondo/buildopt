# Fixed cohort assignment v1

This POC contract closes `B-006`. It supplies the pre-bandit A/A and fixed
cohort assignment layer required by the bounded runtime optimizer without
activating an action on an owner repository.

## Assignment boundary

An assignment is keyed by an idempotency ID and binds the repository,
measurement epoch, exact feature-bucket digest, pre-outcome context digest,
seed, policy, catalog, cohort, resource profile, random basis point, and
propensity. The private ledger atomically persists that record before returning
it to the launcher. Redelivery returns the original record; conflicting reuse
of an assignment ID fails closed.

Probabilities are exact integer basis points and must sum to 10,000. At least
500 basis points remain on `STABLE_CONTROL`. Fixed cohorts contain no duplicate
profiles and may use only the four profiles in the golden catalog. An
ineligible or undeclared profile therefore has zero probability.

## A/A and sample ratio

`FIXED_AA` has exactly the declared `A` and `B` cohorts. Both execute
`STABLE_CONTROL`; only the randomized labels differ. The split may be unequal,
and the recorded propensity is the declared probability of the selected
label.

The sample-ratio check compares observations to those declared probabilities,
not to an assumed 50/50 split. It remains `INCONCLUSIVE` below the configured
sample floor, when an expected cell has fewer than five assignments, when an
assignment is duplicated or belongs to another policy, or when its propensity
is missing or inconsistent. Otherwise Pearson's chi-square must not exceed the
versioned policy threshold.

Outcome, duration, cache-hit, and result fields do not exist in the assignment
request. Delayed and exactly-once outcome processing remains in `B-007`.

## Conformance

The executable implementation and checker were retired after Runtime Tuning
failed the POC value gate. This document is retained as historical protocol
evidence only.

This validates private durable state, deterministic/idempotent assignment,
non-50/50 A/A, exact propensities, finite safe arms, stable-control floor, and
valid/mismatched/inconclusive sample-ratio fixtures.
