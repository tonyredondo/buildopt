# EXPERIMENT_RESULT / ACTION_RECORD v1 lifecycle vectors

Each vector links standalone schema fixtures by repository-relative path. The
test first validates both documents against their own Draft 2020-12 schemas,
then enforces the invariants JSON Schema cannot express across documents.

Valid vectors cover policy-only shadow entry, final-result promotion, and
rollback after invalidation. Invalid vectors prove that an action cannot:

- relabel a preliminary result as final;
- reference a different immutable result version; or
- activate from a final result whose decision is `INCONCLUSIVE`.

The checker also verifies timestamp and interval order, immediate result
ancestry, exclusion totals, action/result scope, policy version, action
membership, state preconditions, and authorization time.
