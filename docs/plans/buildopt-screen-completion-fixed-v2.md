# BV006 corrected candidate screen v2

2026-09-09. The owner requested necessary repairs and completion without repeated
stops, and previously authorized increasing the budget as needed. This amendment
repairs a demonstrated candidate defect and preserves every earlier attempt.

## Why the original round is superseded

The8-CPU original candidate completed10 native builds and all live/independent
output comparisons. Its measured deltas were-0.647s,+1.186s,+210.796s, giving a
43.8% aggregate signal dominated by one commit. However, only the554-file internal
cluster test task retained successful history. Main/Test did not retain history.

A real Gradle9.7.1 API reproduction proves that the tasks share the convention
`configProperties` map. Inserting a private state key overwrote the earlier tasks'
keys. The admission guard correctly sent Main/Test to native checking. Copying
the map before inserting the private property gives each task its own state.
The actual patched installer compiles and passes task-isolation, property/action
preservation, caller-map and wrong-task-mutation tests. Core checking, state
codec, admission predicates and commit protocol have unchanged source.

The original4-CPU baseline had already started when that round was stopped.
Its one observed direct start and native receipt are retained, along with the
interrupted runner and its incomplete accounting. No original8 run is repeated
or relabelled as a full-mechanism success. The earlier reader failure is retained.

## Fixed procedure and acceptance

| Step | Required evidence and outcome |
| --- | --- |
| Installer repair | Actual patched class compiled and invoked against real Gradle9.7.1. Three independent task state bindings; native actions and owner/caller properties retained; modifying one task cannot change another. |
| Warmup gate | Fresh independent N/I states per profile. Revisions17/18 warm each arm. Every successful I build must retain all three nonempty Checkstyle histories; complete C5 outputs must compare before advancing. Existing core proof remains scoped to unchanged code; owner integration is verified jointly by warmups before measured requests. |
| CPU screen | Profiles8,4,2 in that order; native affinity and workers vary together. Four consecutive revisions17..20;19/20 measured, two pairs/profile.8 builds/profile,24 new total. Alternating N/I order by local ordinal. |
| Materiality | Keep >=5% aggregate net customer saving, >=1s mean saving and at least2 positive measured pairs; require actual eligible-file reuse evidence and all three task histories. No threshold tuning after results. |
| Independent history | First qualifying profile in8,4,2 order executes disjoint0..14.0/1 warm;2..14 measured.30 starts maximum. All13 transitions, including no-action builds, remain in the denominator. |
| Decision | Complete outputs, net request/inverse costs, task actions, candidate state evolution, CPU and lifecycle receipts. Report selected points and medians without a precision/G3/product claim. If history fails materiality, do not promote an adaptive controller. |

This is intentionally an opportunity screen; its two measured commits are not
representative daily development. The independent history checks dilution.
Storage is the existing rotational root filesystem. Keep the same filesystem
and workflow across profiles and do not generalize the result to SSD runners.
Formal validation revisions21..100 remain untouched.

## Bounded continuation

Two older failed CPU starts +11 observed original-round starts +24 corrected
CPU starts =37 total. The one-start extension retains the already-started4-CPU
baseline; it is not hidden or refunded. The new round allows54 starts including
conditional history; both continuations together allow65 new observed Gradle
starts. Preserve20 original charged reservations separately from actual starts.

Up to68 helper JVMs across both continuations covers64 comparator starts and up
to4 API probes. Native runs stay sequential. Deadline is prospectively extended
to2026-09-10 02:00:01UTC to finish the repair and bounded experiment under the
standing budget authorization. Combined disk160GiB, minimum free40GiB. No host
configuration, publication, external services, uploads or deletion of old evidence.

Current state: `.tools/state/buildopt-product-viability-v1/task-state.json`,
`engineeringPrefix.screenCompletionFixed`; artifacts under sibling
`bv006-screen-completion-fixed/`. The previous round remains separately retained
under`bv006-screen-completion/`. New inputs and scripts are bound before launches.
