# Next step after the Checkstyle regression investigation

Status: pending design and proof; no new native trial is started or scheduled.
The prior disjoint-history negative result and formal validation 21–100 remain
unchanged. This follows the [investigation](README.md), not a new CPU profile grid.

| Step | Work and check | Required outcome |
|---|---|---|
|1|Find a finalization point that preserves the native final-action lock behavior. Keep before/after context validation, exact completed-report binding, task-local history and fail-closed publication.|A concrete design or a documented refusal; simply deleting Commit is unacceptable.|
|2|Exercise actual native/adapter errors, cancellation, incomplete reports, changed context/files and separate task histories using the existing correctness procedures.|All affected G2 invariants pass; no success can be published from partial or invalid work.|
|3|Compare current optimized finalization against the revised optimized finalization with equivalent inputs and full C5 outputs. Freeze a small balanced6→7 comparison and exact limits before launch; eight actual native starts would cover two independent paired replays.|Actual processed/reused counts stay equivalent. Phase capture distinguishes worker completion, finalization and request completion. Bound or flush daemon traces per request so the target cannot be evicted.|
|4|Measure whole customer-request time, selected-task spans, checking CPU and overlap with compilation. Preserve failures and all setup/validation costs.|A change in reported task duration alone cannot establish value. Keep the proposed mechanism only if its benefit or required behavioral correction is supported by the actual workflow.|
|5|Reconcile the result with sparse Checkstyle activity in the disjoint history before implementing an adaptive controller or expanding repositories.|A justified continue/defer decision. Cache-health signals and whole-build value remain separate; a single transient regression does not automatically invalidate reusable state.|

The exact cause of the old 47s collect span is not recoverable from the retained
old measurements. If a comparable spike recurs, preserve daemon/task JFR intervals,
input-fingerprinting phases, process I/O and host pressure immediately, before
trying a cache or GC correction. Do not call that old cause resolved by a new
synthetic service test or by non-reproduction.
