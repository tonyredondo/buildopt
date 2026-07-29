# CI orchestration v1

This specification materializes `F0-030` and refines `CI-ORCH-001` from
RFC section 18.1.2. Its machine-readable companion is
[`ci-orchestration-v1.json`](./ci-orchestration-v1.json), and
[`dev/check-ci-orchestration`](../dev/check-ci-orchestration) executes every
scheduling, isolation, budget, cancellation, and recovery vector.

## Authority boundary

`buildopt run -- <original argv>` remains the only authoritative normal job.
It executes exactly one pre-outcome assigned arm, preserves the original
command's exit status, and publishes that job's deliverables. A validation job
never changes the originating job's status or deliverables.

High-correction validation is a separate, idempotent request. The protected
default-branch workflow may lease at most one request for a repository. It
runs only trusted revisions and requires a reservation before a lease is
issued. Candidate and control are executed sequentially in randomized order
unless a future adapter proves equivalent runner pools.

## GitHub binding

The beta workflow contract is:

- protected default-branch source and trusted revisions only;
- `schedule` every 15 minutes plus `workflow_dispatch` recovery;
- one non-cancelling `concurrency` group per repository;
- read-only repository permission while leasing and validating;
- at most one validation lease per workflow run;
- no GitHub App, backend-trigger token, fork secret, or
  `pull_request_target`.

The declarative fixture is
[`fixtures/ci-orchestration/github-validation.yml`](../fixtures/ci-orchestration/github-validation.yml).
It is not installed as an active repository workflow; consumers copy the
validated shape into their protected default branch and bind the two explicit
adapter commands.

## Isolation and lifecycle

Candidate and control have distinct fresh worktrees, writable output roots,
`GRADLE_USER_HOME`, Configuration Cache, daemon identity, L1 namespace, and
write credentials. Comparison artifacts cross the boundary only by digest.

The durable attempt lifecycle is the `F0-023` machine:

```text
CREATED
  -> POLICY_BOUND
  -> GRADLE_STARTED
  -> TASK_ACTION_STARTED
  -> VALIDATED
  -> COMMITTED | ABORTED
```

Every transition is compare-and-set and every command is idempotent. An
unknown first-task boundary, changed source/policy, skipped state, or second
owner aborts the attempt and prevents pending publication.

## Budget and recovery

Additional validation reserves runner time before leasing. Without explicit
pilot authorization, a repository may consume no more than 5% of eligible
natural runner-minutes in a rolling seven-day window, no more than 10% in any
24-hour window, and one concurrent validation. Natural cohort assignment is
not additional compute.

Completion, cancellation, expiration, or timeout returns unused reservation.
A dead-owner reconciler aborts the attempt and releases its lease and
reservation; it never starts a duplicate. Infrastructure failure remains
`ABORTED`/`INCONCLUSIVE`, preserves the normal job, and leaves the action
disabled.

## Queue metrics

Natural-job creation, start, finish, labels/pool, and cancellation come from
the CI provider and must be marked `EXACT`. Sequential validation timestamps
must not be reported as customer queue time. Missing exact provider fields
keeps B in fixed cohorts and prevents configuration promotion.

## Conformance

Run:

```bash
./dev/check-ci-orchestration
```

The checker validates the GitHub fixture, cross-checks the lifecycle states,
and interprets all cases in the JSON companion. A case passes only when its
expected decision, reservation disposition, attempt disposition, and
normal-job authority are exact.
