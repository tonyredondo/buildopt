# Sticky wrapper durable native catalog v1

Status: accepted proof-of-concept contract for `SWL-012`.

The durable catalog is the bridge between learning and an ordinary Gradle
repository change. It can identify a structural opportunity, preserve the
exact source transformation and show that the transformation can be applied
and reverted outside a checkout. It never merges a patch or grants runtime
activation authority.

## Opportunity classes

The first catalog contains two implementation-independent classes:

1. **Task contract:** a repeatable, expensive custom task executes on every
   requested build, has stable input/output snapshots, and is missing the
   Gradle input/output/cache contract needed for native reuse.
2. **Graph breadth:** a declared workflow reaches more projects than are
   required for its declared outputs, while repeated observations show that a
   smaller graph preserves the same required output set.

The detector uses typed facts, digests, counts and chronological observations.
Repository names, task names and checkout paths are provenance only; they do
not select a proposal. Unknown relationships, unstable outputs, failures or
ambiguous source shape reject the proposal and retain native Gradle.

## Review and transaction boundary

Every proposal is `PROPOSED` and has `ownerReviewRequired=true`,
`transactionalValidationRequired=true`, `exactRevertRequired=true`,
`patchAuthorized=false` and `activationAuthorized=false`. A recipe is bound
to:

- an exact version and transformation identifier;
- a target path and SHA-256 preimage/postimage;
- a temporary-workspace apply/revert proof; and
- an explicit `automaticMergeAuthorized=false` boundary.

The proof does not mutate the repository checkout. A reviewer may promote a
patch only through the existing candidate/control and full-relevant-validation
workflow. After a durable patch is accepted, plain Gradle must execute it;
BuildOpt runtime is not required for value credit.

## Current evidence

The checked-in report is generated from current paired native Gradle evidence:
[`sticky-wrapper-durable-catalog-v1.json`](../benchmarks/results/sticky-wrapper-durable-catalog-v1.json).
Two synthetic repository families (Kotlin and Groovy DSL) expose the same
task-contract detector and the reviewed `CUSTOM_TASK_CONTRACT_JAVA_V1` recipe.
Each family has eight alternating native pairs, exact required outputs, zero
product failures and no BuildOpt runtime dependency after the source patch.

The graph detector also emits one review-only proposal per DSL. It records a
3-project full workflow and a 2-project output-preserving candidate, but this
block does not claim durable graph timing yet. That timing is intentionally a
separate experiment because graph selection and a committed dependency change
are not interchangeable.

The task-contract value gate requires:

- at least eight paired builds;
- a positive mean saving and positive lower 95% bound;
- more positive than negative pairs;
- exact required outputs and zero product-attributable failures; and
- no BuildOpt runtime action after acceptance.

The current result is a POC result for synthetic families, not a customer
coverage claim or a production rollout authorization.

## Verification

```bash
./dev/check-sticky-wrapper-durable-catalog
```

The checker compiles the catalog benchmark, validates the Draft 2020-12
schema and fixtures, recomputes both detector classes, verifies exact
apply/revert proofs and checks the checked-in report byte-for-byte. The
benchmark source evidence is immutable once committed; a new measurement must
create a new evidence file and report rather than rewriting history.

## Non-goals

- automatic patch application or merge;
- claiming a generic recipe from two DSL fixtures;
- treating graph reduction as performance evidence without native timing;
- reactivating retired Runtime Tuning, Hot State or Copy mechanisms; and
- Test Optimization behavior.
