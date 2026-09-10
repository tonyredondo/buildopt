# Next: explain the ordinal 7 Checkstyle Main regression

State: **pending**, not executed or allocated by this document. The completed
screen is negative on disjoint history. This is one proposed causal investigation
before further historical validation, repository replication or adaptive delivery.
Follow the current user authorization and tracker; this file grants none.

## Question and known evidence

Why does `:server:checkstyleMain` take 117.340 s with the candidate versus 72.718 s
natively at ordinal 7, despite preserved prior state and 4,954 same-content files?
Whole-request loss is 41.094 s. Test checking improves 60.568 -> 9.284 s, while other
native work also varies. The data does not establish an engine defect, a fallback,
worker waiting, hashing cost, JIT/GC or I/O as the cause.

Use the [task spans](./corrected/history-task-diagnostic.json) and
[state-continuity proof](./corrected/history-state-continuity.json). Pin predecessor
`b653442570bc16718c8b75b6bd318dd98e5599b8` and target
`3a00a9167b54dc298e015cf92ef692d6ddacab77`, the corrected candidate, exact engine,
owner policy and strongest compatible baseline. Do not reuse the old shared-map
installer or consume validation 21–100.

## Steps and required outcomes

| Step | State | Work and proof | Required outcome |
| --- | --- | --- | --- |
| N1 | verified | Bound task spans, current/prior content and all three prior success/XML records | Regression localized to a concrete request; state was preserved before launch; actual engine path remains unknown |
| N2 | pending | Prepare minimal, bounded diagnostic receipts for actual admission/refusal, reused/processed counts and separate preparation, hashing, native-checking and commit durations. Keep unaccounted outer-task time explicit. Apply equivalent measurement to the control and prove unchanged complete outputs/failure behavior on focused cases | Distinguish full native fallback, useful reuse with expensive work, and time outside the measured engine phases. No inferred file count is called an observed skip |
| N3 | pending | Freeze one 8-CPU/8-worker diagnostic: at most two independent N/I comparisons of 6 -> 7, each with a warm 6 build per arm; eight Gradle starts total including all warmups and failures. Preserve full C5 checks and ordering. Freeze a four-hour wall limit, disk/free-space limits and helper budget before launch | Phase/count evidence locates a reproducible cause or explicitly returns NOT_REPRODUCED/UNRESOLVED. This two-revision setup does not recreate all prior daemon age/cache history and cannot count as chronological validation |
| N4 | pending | If a concrete cause supports a small repair, freeze it separately and prove semantics plus whole-request benefit before admitting more history. Otherwise close this diagnostic without additional tuning/sampling | An evidence-backed corrected mechanism, or a documented stop/defer decision for the current candidate. No unlimited diagnostic loop or automatic retries |
| N5 | deferred | Only after mechanism and measurement prerequisites pass, return to the frozen lifecycle protocol at a single CPU profile, then breadth and adaptive N/F/A comparison | Savings across every scheduled transition, full adoption/maintenance costs and formal product gates; selected wins alone remain insufficient |

N2 must budget its fixture/helper/compiler work explicitly. N3's eight starts are
a diagnostic proposal, not a new 80-request pilot or three-profile grid. Do not
repeat a failed run outside the frozen budget. A positive task-only result cannot
replace the workflow floor; a non-reproduced regression is not a repaired one.

Keep the observed worker-affinity failure separate. It remains unqualified and
cannot be waived by a positive economic point estimate. Any measurement design
used for formal value must satisfy its prerequisites without reclassifying older
failed gates. H2 still needs a source-bound unnecessary-invalidation cause;
the longer `collectTransportVersionReferences` span alone does not admit it.
