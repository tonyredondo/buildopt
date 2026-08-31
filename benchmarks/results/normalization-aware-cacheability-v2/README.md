# Normalization-aware cacheability v2 evidence

`NAC-002` completed the fresh source-only classification in
[`source-classification.json`](./source-classification.json). All 5/5 frozen
revisions are conclusive and 4/5 families expose an action, passing the frozen
3/5 breadth threshold. The 18 typed rows contain eight marker-only candidates,
one reviewed-relative-proof-required candidate, seven already-cacheable tasks
and two incomplete/ambiguous tasks.

The report was regenerated from the five frozen Git source trees by the
versioned v2 classifier. The independent checker derives family counts from
rows, verifies source digests, reruns the classifier and rejects any DNO report
dependency.

`NAC-003` adds the checked [`patch-plan.json`](./patch-plan.json), a
digest-bound compiler and a real Gradle 8.14.3/9.6.1 × Groovy/Kotlin fixture
matrix. All four fixture rows restore exact outputs from cache and all eight
marker-only transactions apply idempotently and revert byte-exact.

[`public-correctness.json`](./public-correctness.json) records the bounded
`NAC-004` result. Micronaut's reviewed-relative action passes native fallback,
content/path invalidation, cross-root restore and exact output comparison;
OpenTelemetry and both Spring candidate classes also restore in a second root.
That is 4/9 fully proved candidates. Groovy's performance-summary inputs require
four versions with 350 compilation iterations each; the bounded run stopped
during the first version without a product failure, leaving the frozen
every-candidate gate incomplete.

[`terminal-decision.json`](./terminal-decision.json) therefore closes
`NAC-005` and `NAC-006` as `NOT_AUTHORIZED` and records
`STOP_NORMALIZATION_AWARE_CACHEABILITY_POC`. No paired or longitudinal timing,
speedup, installed proposal UX, automatic merge, or successor is claimed.
