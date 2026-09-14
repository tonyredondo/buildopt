# Checkstyle on four consecutive changes

BO-05 · 14 September 2026 · **`EXPLORATORY_MATERIAL_SIGNAL`**.

The Checkstyle correction reduced total request time by 34.14% and 28.17% in
the two sequences. Both passed the rule set before the experiment. Checkstyle
can proceed to BO-06, where we need to settle measurement readiness and freeze
the longer validation. Savings over a longer history remain unproved.

## Why this is a separate attempt

The [previous attempt](../bo-05-checkstyle-screen/README.md) stopped on a
permissions error in the observation script. The failing read was not recorded;
focused tests reproduced the same error when optional process data was
inaccessible. Its completed sequence and missing outcomes remain recorded there.
This comparison started both
sequences from fresh state and uses the repaired observer; no earlier timing
contributes to its decision. The source correction and decision thresholds
are unchanged.

The observer repair passed 14 focused checks, including a real permission
denial. Another 22 checks covered the decision rule and actual observation
callback. None started Gradle or a comparator JVM. Missing optional data stays
explicit; identity and CPU assignment checks remain required.

The [completed observer record](./receipts/observer-completion.json) contains
5,155 samples with no observation failure or recorded permission denial. The
denial handling was exercised in the separate focused tests. The measured run
used `run-screen-v2.py`; the original collector is retained only as the failing
case in those tests.

## What we compared

Checkstyle checks source formatting and coding rules. The correction remembers
successful checks on unchanged files while preserving complete reports. Both
arms ran `:server:precommit --continue`, including the same dependencies. N used
native Checkstyle; I used the supported V2 correction. Both retained the earlier
ForbiddenPatterns fix, so the measured difference concerns Checkstyle alone.

Two independent runs followed Elasticsearch changes 17–20. Change 17 started
from cold private state; changes 18–20 retained each arm's state. The command,
native cache policy, tools, eight-worker CPU profile and output rules were fixed.
Configuration Cache remained disabled for this workflow. The [manifest](./manifest.json)
records exact revisions and settings; the [protocol](./protocol.md) was fixed
before timing. No protected validation change was executed.

## Decision across all measured changes

Each replication had to save at least one second per measured pair and 5% in
total, with at least two positive pairs, actual native checking and candidate
reuse. Both replications had to pass. The inactive change remains included.

| Replication | Native total seconds | Corrected total seconds | Mean seconds saved | Saving | Positive pairs | Result |
|---|---:|---:|---:|---:|---:|---|
| 1 | 195.564 | 128.803 | 22.254 | 34.14% | 2/3 | Pass |
| 2 | 303.005 | 217.639 | 28.455 | 28.17% | 3/3 | Pass |

These totals cover changes 18–20. Request time includes required machine work
recorded outside the native command. The [result](./result.json) retains each
delta and its component costs. The two replications are repeated runs of one
selected development window, not independent repositories or a precision study.

## Every build

N is native Checkstyle; I is the supported correction. No timing was removed,
replaced or repeated because it was unfavorable.

| Replication | Change | Role | Arm | Build seconds | Whole-request seconds | Host flag |
|---|---|---|---|---:|---:|---|
| 1 | 17 | Cold | N | 212.489 | 212.797 | FLAGGED |
| 1 | 17 | Cold | I | 201.714 | 201.877 | FLAGGED |
| 1 | 18 | Measured | N | 29.227 | 29.291 | BELOW_REGISTERED_FLAG |
| 1 | 18 | Measured | I | 29.215 | 29.294 | BELOW_REGISTERED_FLAG |
| 1 | 19 | Measured | N | 75.546 | 75.669 | BELOW_REGISTERED_FLAG |
| 1 | 19 | Measured | I | 35.309 | 35.393 | BELOW_REGISTERED_FLAG |
| 1 | 20 | Measured | N | 90.512 | 90.603 | FLAGGED |
| 1 | 20 | Measured | I | 64.040 | 64.117 | FLAGGED |
| 2 | 17 | Cold | N | 207.461 | 207.565 | FLAGGED |
| 2 | 17 | Cold | I | 200.242 | 200.321 | FLAGGED |
| 2 | 18 | Measured | N | 29.046 | 29.113 | BELOW_REGISTERED_FLAG |
| 2 | 18 | Measured | I | 28.803 | 28.883 | BELOW_REGISTERED_FLAG |
| 2 | 19 | Measured | N | 76.241 | 76.326 | BELOW_REGISTERED_FLAG |
| 2 | 19 | Measured | I | 34.986 | 35.238 | BELOW_REGISTERED_FLAG |
| 2 | 20 | Measured | N | 197.110 | 197.566 | FLAGGED |
| 2 | 20 | Measured | I | 152.996 | 153.518 | FLAGGED |

The workstation was shared. [Host observations](./host-pressure.json) and
[order and idle intervals](./order-and-costs.json) retain all flags and coverage
limits. They do not identify the cause of an individual slow build. A flag does
not remove a build from the comparison.

Change 20 was much slower in the second sequence on both sides: native request
time rose from 90.60 to 197.57 seconds, and corrected time from 64.12 to 153.52.
Both versions were flagged for host pressure in both sequences. The records
do not establish the cause of the increase. The correction still saved time
in each comparison. Change 18 was effectively unchanged, including a small
regression retained in the first sequence.

## Work avoided and correctness

| Replication | Change | Arm | Files passed to the checker | Reused checks |
|---|---|---|---:|---:|
| 1 | 18 | I | 0 | 0 |
| 1 | 18 | N | 0 | 0 |
| 1 | 19 | I | 2 | 4957 |
| 1 | 19 | N | 4959 | 0 |
| 1 | 20 | I | 7 | 8332 |
| 1 | 20 | N | 8339 | 0 |
| 2 | 18 | I | 0 | 0 |
| 2 | 18 | N | 0 | 0 |
| 2 | 19 | I | 2 | 4957 |
| 2 | 19 | N | 4959 | 0 |
| 2 | 20 | I | 7 | 8332 |
| 2 | 20 | N | 8339 | 0 |

The [task records](./task-execution.json) retain cold builds, all task outcomes
and exact engine/adapter counts. Task durations are not added together to claim
elapsed savings. All eight live output comparisons and eight independent
reconstructions passed under the existing complete-output contract. All 16
builds succeeded. Source inventories verified that only the declared correction
differed; native Checkstyle acquired no candidate success state.

The [verification](./verification.json) also checks all frozen inputs, four
private worktrees, ownership cleanup and limits. Earlier lifecycle proof was
reused only after its sources and fixture records matched. Its Gradle version
and Configuration Cache limits remain unchanged.

All sampled native and observer CPU assignments matched their expected sets.
Sampling does not prove continuous isolation. Formal timing qualification
(G0/G3 in the replay contract) remains unfinished.

## Cold and preparation costs

| Replication | Extra corrected cold seconds, if any | Seconds saved across all four requests, including cold |
|---|---:|---:|
| 1 | 0.000 | 77.681 |
| 2 | 0.000 | 92.611 |

The following ledger totals preserve the runner's existing classification.
`customer-machine` names required machine work in that contract; it does not
refer to a customer trial. Inside-request rows are already counted in request
time and must not be added again. Research preparation is retained separately.

| Cost class and location | Seconds |
|---|---:|
| research / outside request | 2871.223 |
| customer-machine / inside request | 1667.553 |
| customer-machine / outside request | 0.018 |

These figures do not establish recovery of adoption or maintenance costs. The
screen used 16 owner builds and 16 comparator JVMs, with no retry or extra
comparison pass. State used 25.67 GiB within
the 64 GiB allowance. The allocation and all four owned service scopes closed.

This attempt finished in about 93 minutes. Across both BO-05 attempts, 32 build
reservations were charged, 24 builds actually ran and 20 comparator JVMs started.
The interrupted attempt remains separate and contributes no savings to this
decision.

## Candidate selection and next step

| Candidate | Current disposition |
|---|---|
| Elasticsearch Checkstyle | Short screen passed; remaining BO-06 work is next. |
| Micronaut Python task | Whole-workflow frequency and useful recurring cache restores remain unmeasured. BO-03 did not admit timing. |
| Spring architecture checks | The retained opportunity does not justify another timing allocation under the one-second floor. |
| Ktor and Apache Beam Build Impact | BO-04 admitted no distinct continuation of the selected-plan experiments. |

Continue with the remaining BO-06 readiness and confirmation protocol. Formal G0/G3, preparation recovery and the protected validation history remain unqualified or unrun. A positive short screen does not authorize that replay.

Follow the [governing tracker](../../../../docs/plans/buildopt-research-execution-plan-2026-09-14.md).
The [evidence index](./evidence-index.json) identifies exact report copies and
retained local raw files. This export includes every timing row and its analysis;
full generated outputs and inventories remain local and are required to rerun
the complete output reconstruction. Product viability remains **NOT_ESTABLISHED**.
