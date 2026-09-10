# Spring Messaging Candidate Correctness v1 Tracker

> **Research disposition: HISTORICAL; closed research (2026-09-08).**
> Original outcomes and evidence are preserved below. Old phase instructions
> and unchecked boxes do not reopen this route. Use the
> [research status register](../research-status.md) and
> [current tracker](./buildopt-product-viability-v1-tracker.md) for new work.

| Block | Outcome | State |
|---|---|---|
| `SMCC-001` | Freeze the exact untimed three-request correctness contract | `DONE` |
| `SMCC-002` | Execute native/native/candidate from empty state | `DONE` |
| `SMCC-003` | Reconstruct exact outputs and authorize value or stop | `DONE` |

The sequence naturally produced native/native/candidate with 11 matches,
14,406 outputs and zero failures on every request. Its fresh digest is stable
across all three requests but differs from the SMGC digest frozen by the
contract.

Terminal decision: `STOP_SPRING_MESSAGING_CANDIDATE_CORRECTNESS_OUTPUT_DRIFT`.
The candidate is not accepted as correct and paired timing remains
unauthorized. Any successor must isolate the exact cross-run output drift
without weakening equality after seeing the result.
