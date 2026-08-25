# Adaptive-fragment patch-opportunity learning v1

Status: accepted POC contract for `AF-008`.

Machine policy:
[`poc-adaptive-fragment-patch-opportunity-v1.json`](./poc-adaptive-fragment-patch-opportunity-v1.json).

## Purpose

This block tests whether ordinary-build evidence can identify a durable source
improvement rather than requiring BuildOpt on every later execution. It detects
one bounded missing Gradle custom-task contract, emits a review-only Patch
Autopilot proposal and accepts that patch only after transactional correctness
and native Gradle value validation.

The detector is generic; the accepted recipe is not. Detection uses opaque
repository provenance, task-implementation identity, repeated requested-build
observations and bounded source facts. It contains no repository name, remote,
project allowlist or task-name rule. The current Java recipe remains restricted
to the exact reviewed source shape and cannot be generalized to arbitrary task
implementations.

## Detection gate

A proposal requires at least three ordinary requested builds with:

- a median repeated cost of at least 500 ms;
- the task executing every time while neither cacheable nor up-to-date;
- stable input and output snapshots;
- no product-attributable failure; and
- one bounded Java `DefaultTask` shape with internal input/output properties,
  exactly one task action and no known side effect.

These facts identify an opportunity for review; they do not prove that any
annotation is semantically correct. Ten unsafe or ambiguous mutations are
rejected by the executable tests. Repository/path renaming preserves the
classification.

Every detector result remains `PROPOSED` and carries:

- `ownerReviewRequired=true`;
- `transactionalValidationRequired=true`;
- `exactRevertRequired=true`;
- `patchAuthorized=false`; and
- `activationAuthorized=false`.

## Review, transaction and revert

The owner-reviewed fixture uses the existing
`CUSTOM_TASK_CONTRACT_JAVA_V1@1.0` recipe. Its exact preimage and postimage are
content-bound. The proof applies the replacement only in a temporary workspace,
verifies the postimage, restores the exact preimage and confirms that a rejected
proposal performs zero mutations. The existing signed bundle, six-run candidate
validator and real-Git exact-revert checks remain authoritative for delivery.

## Independent native value

AF-008 does not generate a second favourable timing sample. It imports and
revalidates the frozen paired measurements from
[`poc-value-coverage-v1.json`](../benchmarks/results/poc-value-coverage-v1.json),
where the accepted source patch was measured using ordinary Gradle directly on
the qualified 4-CPU/16-GiB runner:

| DSL | Pairs | Native before | Native after | Mean saved | Reduction | 95% interval | Positive pairs |
|---|---:|---:|---:|---:|---:|---:|---:|
| Kotlin | 8 | 2,035.250 ms | 666.000 ms | 1,369.250 ms | 67.28% | [1,142.125, 1,624.375] ms | 8/8 |
| Groovy | 8 | 3,454.125 ms | 1,105.000 ms | 2,349.125 ms | 68.01% | [1,245.125, 3,420.750] ms | 7/8 |

All 16 paired comparisons preserve exact required outputs, candidate runs
restore all eight tasks from the native Gradle build cache, and product-
attributable failures remain zero. BuildOpt is not required after the source
patch is accepted.

This proves durable value for one reviewed synthetic task-contract shape. It
does not prove a generic patch recipe, automatic patch application, production
promotion or arbitrary customer-repository coverage.

Run:

```bash
./dev/check-adaptive-fragment-patch-opportunity
```
