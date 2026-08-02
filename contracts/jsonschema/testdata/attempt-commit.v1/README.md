# Attempt and commit fixtures

Synthetic records for `ATTEMPT_STATE`, `CI_VALIDATION_REQUEST`, and
`COMMIT_DECISION` v1.

- Individual schema fixtures reject a skipped lifecycle transition, shared
  candidate/control writable state, and an inconclusive commit authorization.
- `attempt-lifecycle/valid/happy-commit.json` covers the complete durable
  lifecycle through an exactly covering decision.
- `attempt-lifecycle/valid/abort-before-task.json` proves a pre-task failure
  terminates without a commit.
- Invalid lifecycle vectors exercise a skipped state, a second owner, and a
  decision that does not cover every pending object.

The semantic test treats command IDs as idempotency keys, enforces contiguous
sequence/state versions, immutable source/policy/owner bindings, terminal
states, ordered timestamps, task-action replay boundaries, and exact object
coverage. All identifiers, digests, and signatures are synthetic.
