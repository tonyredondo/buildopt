# Checkstyle comparison on changes 17–20

BO-05, fixed before execution on 14 September 2026.

The question is whether the supported Checkstyle correction warrants further
validation against native Checkstyle on the same complete build command. This
is the short comparison admitted by BO-03 after the BO-02 identical-code control.
BO-04 admitted no new Build Impact trial.

Run two independent sequences of Elasticsearch changes 17, 18, 19 and 20. Each
has native and corrected arms, with private state. Change 17 is the cold anchor.
Changes 18–20 are the six measured pairs across the two sequences. Keep the
existing runner's alternating order, reversed in the second sequence. Preserve
state within each arm; do not reset it between commits or substitute a commit.

Both arms request `:server:precommit --continue` with the retained
ForbiddenPatterns correction, native build caching, eight workers and
Configuration Cache disabled. Only the corrected arm receives the supported
Checkstyle V2 change. The manifest records every flag, tool, input and source
revision. CPUs 0–7 run native work, CPU 8 observes it and CPU 9 controls the run.
These assignments do not isolate the shared machine.

Reuse the unchanged runner, output comparison, phase instrumentation and
supported lifecycle proofs. Verify source preconditions at all four revisions.
The complete live output comparison and independent reconstruction must pass
for every pair, including the cold builds. Do not change output normalization.
Check preserved success state against the captured reports; native Checkstyle
must not acquire the candidate's local state. Missing evidence or a build or
correctness failure makes the comparison incomplete, not a measured rejection.

Before timing, fix this exploratory continuation rule: **each sequence** must
save at least one second per measured pair and 5% of total native request time
over all three measured pairs, with at least two positive pairs, observed native
checking and candidate reuse. Both sequences must pass. Inactive changes remain
in the denominator. A pass is `EXPLORATORY_MATERIAL_SIGNAL`; otherwise a complete
comparison is `NO_MATERIAL_SIGNAL_IN_THIS_SCREEN`. Never pool a failed sequence
into a favorable average or select only the active changes.

The primary time includes the full request and recorded required machine work
outside that request. Keep every cost row, both cold pairs, each cold excess,
and totals including the cold builds. Show research-only costs separately.
The short sequence does not establish recovery of preparation costs, recurring
net savings, statistical precision or G0/G3. Report individual regressions and
all host-pressure flags without dropping observations or identifying a cause
that the records cannot establish.

The allocation is **16 owner builds**, including four cold builds, and at most
**16 comparator JVMs**: eight live and eight independent comparisons. Reuse
prior valid correctness proof; preparation starts no Gradle or helper JVM.
There are no timing retries or extra comparator passes. Stop within three hours
from allocation, 64 GiB of new state and at least 40 GiB free disk. The controller
has a 128 MiB diagnostic limit; the runner receives the remaining state limit
after reserving 256 MiB for other task files. Keep failures and unrun rows.

A positive result permits the remaining BO-06 readiness and protocol work. It
does not start the protected changes 21–100. A negative result closes this
candidate selection under the current hypothesis. An incomplete result needs
its cause and remaining proof recorded; no automatic replacement trial follows.

## Separate attempt with the repaired observer

The owner authorized this new allocation after the interrupted attempt. Use
fresh state for both sequences. Keep the interrupted attempt and its costs
separate; none of its rows contribute to this decision. Candidate code, native
command, history, output comparison and decision thresholds are unchanged.

Only optional process I/O counters and wait-channel reads tolerate a permission
denial. Record the missing observation and its path explicitly. Process identity,
command and CPU masks remain required. Missing I/O must never be reported as zero.
The corrected observer and phase analyzer are frozen before execution.
CPU sampling describes the observations; it does not prove continuous isolation.
