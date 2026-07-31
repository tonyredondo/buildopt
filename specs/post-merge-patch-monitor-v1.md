# Post-merge patch monitor v1

This POC contract closes C4-009 and C4-G05 without pretending that Git offers
percentage rollout after merge. Natural builds of the patched revision are
retained. When budget permits, an isolated control executes the exact inverse
on the same repository, revision, epoch, work units, runner, and policy.
Missing or inexact controls are CONTEXTUAL; fewer than four complete exact
pairs or an interval crossing zero is INCONCLUSIVE.

The effect is patchedNaturalMs - inverseControlMs. Four or more exact pairs use
4,096 deterministic paired-bootstrap replicates and a separate p95 guardrail.
A positive lower 95% bound, positive mean plus p95 regression, patched build
failure/cancellation with a successful inverse, or required-artifact divergence
requests a draft revert PR. Failed and cancelled outcomes remain retained.
Only a strictly negative upper bound with a non-regressed p95 is classified as
an improvement.

The exact inverse path currently supports the proven archive-reproducibility
recipe and MODIFY operations only. It reads the signed original preimages from
the original base commit without checkout, requires every file at the merged
revision to equal the original postimage, swaps pre/postimage digests, and
creates a new JCS/Ed25519 PatchBundle whose action ID is derived from the complete original bundle SHA-256. That
bundle must pass the production
verifier and path-safe applier before delivery through the existing immutable
branch/draft-PR workflow. ADD, later customer edits, unsupported recipes, stale
validation, or any binding mismatch produce a precise revert instruction or
PROPOSED state instead of deleting, rebasing, overwriting, or pushing silently.

Run `./dev/check-post-merge-patch-monitor`.

The checker executes causal improvement, regression, crossing-interval,
contextual, insufficient, failure, cancellation, artifact-divergence, and
inexact-control cases. A real temporary Git repository then applies an archive
patch, simulates its merge, generates and independently verifies the signed
inverse, creates only a draft revert branch/PR, restores exact original bytes,
and proves the default branch remains unchanged. Diverged postimages and ADD
operations fail closed. No credentials or remote mutation are used.
