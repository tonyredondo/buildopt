# Change-Scoped Candidate Correctness v1 Tracker

**Status:** closed — `STOP_CHANGE_SCOPED_CANDIDATE_CORRECTNESS`<br>
**Current block:** `CSCPC-004`

| Block | Deliverable | State |
|---|---|---|
| `CSCPC-001` | Freeze four-family untimed correctness contract | `DONE` |
| `CSCPC-002` | Execute native/native/candidate rows | `DONE_STOP` |
| `CSCPC-003` | Independently reconstruct exact-output gate | `DONE` |
| `CSCPC-004` | Record terminal correctness decision | `DONE` |

The route consumes the 4/5 actionable CSCPN result without moving its gate.
It permits twelve requested Gradle starts and four candidates, but no timing
sample or speedup claim. A 4/4 exact, zero-failure result is required before a
separate paired-value contract may be considered.

Groovy stopped after its first successful native invocation because current
ordinary-learning economics returned `COMPATIBLE_LIFETIME_INSUFFICIENT` with
one historical compatible match. The independent checker reconstructs that
typed result from the compressed invocation. Totals are 1/4 attempted
families, one Gradle start, zero candidates, zero timing samples and zero
product failures. Three families and eleven starts remain deliberately unrun.
