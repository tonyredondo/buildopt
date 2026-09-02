# Reviewed Native Patch Delivery v1

`REVIEWED_NATIVE_PATCH_DELIVERY_V1` tests whether the two qualified RNPP
corrections can enter the existing Patch Autopilot product path without a
special public-repository runtime. The fixed inputs are the Micronaut reviewed
relative-normalization correction and the Spring marker-only correction from
RNPP-005.

Each recipe is a closed ID/version pair bound to one relative source path, one
preimage SHA-256, exact insertion offsets and one postimage SHA-256. Source
drift, unknown recipes, ambiguous edits and non-idempotent output fail closed.
Both recipes require an owner-reviewed adapter, full relevant validation and
an exact inverse before PatchBundle delivery.

The proof must materialize both exact public preimages, reproduce the accepted
postimage digests, reject negative fixtures, and pass the existing signed
PatchBundle verification, isolated Git branch, draft-delivery and exact-revert
matrix. Delivery uses an in-memory draft-PR adapter; no public remote is
modified.

Machine economics include observed recipe materialization and the complete
integrated delivery/revert validation. Human review remains `NOT_MEASURED` and
is excluded from payback. Qualification requires two delivered recipes, exact
digests, reversible delivery, zero product failures and finite machine payback
against the RNPP signed saving of 7,906.625 ms per compatible portfolio build.

This owner-operated POC does not authorize production promotion, automatic
application, automatic merge, Test Optimization or repository/task-name
selection rules.
