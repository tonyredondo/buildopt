# Patch delivery recovery v1

This POC contract closes C4-008 by fixing one idempotent recovery state machine
for the customer-owned C4-007 workflow. Its immutable identity is repositoryId,
actionId, and bundleDigest; its only branch is buildopt/<actionId>.

An absent identity creates the exact action branch and then one draft PR. An
exact branch without a PR preserves the branch and creates only the missing
draft PR. An exact branch plus matching draft returns the existing delivery.
A different head/tree, marker, identity, or non-draft PR returns PROPOSED
without changing remote state. A lookup or PR-creation failure leaves the
exact branch intact so workflow_dispatch can retry the same identity.

Each run attempts at most one missing PR creation. Recovery never rebases,
force-pushes, overwrites or deletes a branch, writes the default branch, marks
a PR ready, or merges. It never turns a changed idempotency key into a retry.

Run `./dev/check-patch-delivery-recovery`.

The checker requires all six terminal states, composes the protected customer
workflow, and executes the production Java patcher matrix. Its real temporary
Git repositories prove initial creation, exact replay, branch-without-PR
recovery, divergent-head rejection, conflicting-PR rejection, and retryable
PR failure without remote credentials or mutation.
