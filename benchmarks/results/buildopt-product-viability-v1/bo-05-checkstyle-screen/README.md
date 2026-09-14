# Checkstyle comparison on four changes

BO-05 · 14 September 2026 · **INCOMPLETE_SCREEN**.

Eight builds finished and all four live comparisons of their outputs passed.
A permissions error in the measurement script stopped the experiment before the
second sequence's first build. BO-05 remains incomplete. This result neither admits the
correction to long validation nor rejects it for lack of savings.

## What ran

Checkstyle checks source formatting and coding rules. The correction remembers
successful checks on unchanged files while preserving complete reports.
Both versions ran Elasticsearch's `:server:precommit --continue` command.
N used native Checkstyle; I used the supported V2 correction. Both retained
the earlier ForbiddenPatterns correction, which checks for disallowed text,
so this comparison concerns Checkstyle alone.

The [frozen protocol](./protocol.md) specifies two independent sequences of
changes 17–20, sixteen builds in total. Each starts from cold private state and
preserves that state through the following changes. The [manifest](./manifest.json)
fixes the revisions, eight-worker CPU profile, tools and output checks. Native
build caching is enabled; Configuration Cache remains disabled for this workflow.

## Every completed build

These are all eight builds in the first sequence. Whole-request time also
includes required machine work recorded outside the native command.

| Change | Role | Version | Native seconds | Whole-request seconds | Host flag |
|---|---|---|---:|---:|---|
| 17 | Cold | N | 241.893 | 242.241 | FLAGGED |
| 17 | Cold | I | 205.750 | 205.951 | FLAGGED |
| 18 | Measured | N | 29.162 | 29.246 | BELOW_REGISTERED_FLAG |
| 18 | Measured | I | 28.928 | 28.995 | BELOW_REGISTERED_FLAG |
| 19 | Measured | N | 80.552 | 83.462 | FLAGGED |
| 19 | Measured | I | 35.497 | 36.118 | FLAGGED |
| 20 | Measured | N | 92.991 | 93.239 | FLAGGED |
| 20 | Measured | I | 86.722 | 87.296 | FLAGGED |

Across changes 18–20, the observed mean fell from
68.649 to 50.803 seconds:
26.00% less time, or 17.846
seconds per build. All three differences favored the correction, including the
change with little opportunity. Including the cold build, the first sequence's
request times differed by 89.827 seconds in favor of the correction.

These observations are provisional. The required second sequence and independent
output reconstruction are missing. No passing screen decision is assigned.
The [result](./result.json) retains every difference and its components; the
[task records](./task-execution.json) retain the work performed and reused.

The shared workstation showed pressure during several builds. All
[host flags and sampling gaps](./host-pressure.json) are retained. They do not
identify the cause of an individual timing, and no observation was dropped.
Sampled CPU masks do not establish uninterrupted isolation or general precision.

## What did not run

| Second sequence change | Native build | Corrected build |
|---|---|---|
| 17 | Not run | Preparation started; native command not started |
| 18 | Not run | Not run |
| 19 | Not run | Not run |
| 20 | Not run | Not run |

The [verification record](./verification.json) retains all sixteen scheduled
statuses. The independent check that reconstructs comparisons from their saved
files was not reached. The four live output checks remain evidence for their
original scope; the complete correctness requirement has not passed.

## Why it stopped and what was corrected

The controller recorded `PermissionError(13, 'Permission denied')` while
observing processes during preparation of the second sequence. It then stopped
the runner. Its error record did not retain the failing PID or file, so the
exact original read cannot be identified.

A [focused Linux test](./receipts/observer-repair-tests.json) reproduces the same
failure class: a process can remain visible while access to its disk counters
is denied. The original collector aborts on that read. The revised collector
records inaccessible disk counters and wait information as explicit gaps while
retaining process identity and CPU affinity. Other IO errors and failures to
read required identity or command information still fail. Future fatal errors
retain their file and traceback.

The resource report now keeps unavailable disk measurements empty instead of
inventing zeroes. The status helper also distinguishes a stopped controller
from an unfinished phase marker. Fourteen focused repair cases passed, along
with the [fourteen decision and eight collection checks](./receipts/revised-controller-tool-tests.json)
on the revised controller. One test-fixture correction is retained with its
explanation. No Gradle build or comparator JVM was started by these tests.

The revised collector has **not** run this build experiment. Its exact changes
are available in the [collector diff](./tools/run-screen-v2.py.diff) and
[resource-report diff](./tools/analyze-phases-v2.py.diff). The original frozen
files remain unchanged. Recovery analyses were added after the interruption;
they preserve the original timing, cost and host-flag rules and assign no
performance pass.

## Costs and closure

Sixteen build reservations remain charged. Eight builds and four live-comparison
JVMs actually ran; eight builds and all eight independent-comparison JVMs did
not. There were no retries. The allocation is closed and its unused reservations
are not reusable. The [cost and order record](./order-and-costs.json) retains all
44 completed cost rows. The unfinished preparation phase remains explicit in
the verification record, with no invented completion time.

| Recorded work and location | Seconds |
|---|---:|
| Research preparation and checking / outside request | 3218.408 |
| Required machine work / inside request | 806.538 |
| Required machine work / outside request | 0.011 |

Rows inside a request are already included in its time and must not be added
again. Preparation and checking costs are retained separately. The observations
do not establish that future savings would recover preparation or maintenance.

The state uses 19.82 GiB within the 64 GiB limit. All three created services
are inactive, and the controller and runner have ended.
Two services have their original closure records; the third is verified through
an external service-state observation. No missing owner record was fabricated.
All four worktrees and raw evidence remain available.

## Next step

Checkstyle remains the candidate awaiting a complete short comparison. Micronaut's
whole-workflow opportunity remains unmeasured; Spring and the reviewed Ktor/Beam
continuations have no admitted timing allocation. This interruption changes
none of those decisions.

Before another measured attempt, register a separate allocation and freeze the
revised observation sources. The proposed attempt retains the same correction,
commands, changes and criteria: two fresh sequences, sixteen builds, at most
sixteen comparator JVMs, one CPU profile, three hours and 64 GiB. Preserve this
interrupted attempt separately and do not choose the better result from repeated
sequences. The frozen protocol permits no automatic replacement trial, so none
has started. Protected changes 21–100 remain untouched.

Follow the [execution tracker](../../../../docs/plans/buildopt-research-execution-plan-2026-09-14.md).
The [evidence index](./evidence-index.json) identifies exact copies and the local
raw files needed for further verification. Full generated outputs, inventories
and process samples remain local. Product viability remains **NOT_ESTABLISHED**.
