# Next: keep file-write delay out of periodic collection

The live diagnostic is complete and its two Gradle starts are exhausted. The
new, supported mechanism is a 323/341 ms synchronous write holding up sampling
while both heartbeats remain active. The original 2,104 ms event still has no
phase evidence. Do not claim a filesystem fault or change host settings.

| Step | State | Work | Required outcome |
|---|---|---|---|
| R1: freeze the supported scope | verified | Retain both live write delays, cold endpoint gaps and the older rejection | A writer-boundary mechanism is observed; original cause and readiness remain unresolved |
| R2: bounded writer separation | pending | In a fresh task-owned source version, separate periodic collection from file persistence with an explicit byte/record bound; keep phase clocks, payload identity and ownership | A finite writer stall cannot silently stop collection; saturation/errors fail explicitly, all persistence and drain costs remain charged |
| R3: non-Gradle consuming proof | pending | Use live collection with deliberate slow writes, write failure, saturation, cancellation during capture/drain and observer/driver exit | Exact order/count/hash accounting, no silent drops, no unbounded queue or orphan, bounded termination and preserved disk/liveness guards |
| R4: close the screen prerequisite | blocked | Freeze the verified source and qualify its integration with the existing C5 replay/output comparator; explicitly retain cold/warm boundaries | S1 can change only on current, compatible observation/lifecycle proof; success in these cold diagnostics does not pass the original 500 ms gate |
| R5: short CPU screen | blocked | Once S1 is verified, run the existing 8/4/2 plan, at most 36 total starts including warmups/failures | Every row, correction activation, full costs and output comparison; exploratory results only |

Start R2/R3 with a newly recorded non-Gradle wall/disk/process allocation. This
file allocates no native starts, helpers or extra calibration campaign. Do not
repeat the 32/80-build controls or nominal replays hoping for a failure. If any
new actual-owner prerequisite is necessary, identify and register its small
total cost before launch, including preparation/failure starts.

Preserve the original 100 ms capture-balance and 500 ms observation criteria, G0
and lifecycle gates, all rejected rows and 80 held-out transitions. Separating
writes is a research-instrument correction, not a product acceleration claim.
After R3, use its evidence to decide whether R4 is supportable; do not declare
S1 verified from code alone or from an event that did not recur.

Durable locator: `.tools/state/buildopt-product-viability-v1/task-state.json`, key
`engineeringPrefix.liveObserver`; source, bindings and raw records live under
`.tools/state/buildopt-product-viability-v1/bv006-live-observer`. No publication
or host configuration change is part of this continuation.
