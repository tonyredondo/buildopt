# Reviewed Native Patch Delivery v1 Tracker

> **Research disposition: HISTORICAL; closed research (2026-09-08).**
> Original outcomes and evidence are preserved below. Old phase instructions
> and unchecked boxes do not reopen this route. Use the
> [research status register](../research-status.md) and
> [current tracker](./buildopt-product-viability-v1-tracker.md) for new work.

| Block | Outcome | State |
|---|---|---|
| `RNPD-001` | Freeze the two-recipe delivery contract | `DONE` |
| `RNPD-002` | Add exact versioned recipes and fail-closed negatives | `DONE` |
| `RNPD-003` | Materialize both qualified public-source corrections | `DONE` |
| `RNPD-004` | Prove signed draft delivery and exact revert | `DONE` |
| `RNPD-005` | Measure delivery economics and issue the POC decision | `DONE` |

Both accepted RNPP proposals now have closed Patch Autopilot recipe IDs. The
exact public preimages reproduce their frozen postimage digests. The existing
real-Git spike signs and verifies the bundles, creates only isolated action
branches, records draft delivery through an in-memory adapter, and generates
exact signed reversions for both recipe IDs.

Observed materialization plus integrated delivery validation costs 24,150 ms.
Against the signed 7,906.625-ms compatible-build saving, machine-only payback
is four portfolio builds. Human review remains unmeasured, so this qualifies
the controlled owner-operated delivery POC but not commercial economics or
production readiness.
