# BV006 screen completion: reader correction v2

2026-09-09. The8-CPU run completed all10 Gradle starts and both live and
independent output comparisons. Its first post-run analyzer failed on a systemd
startup name: `(history-replay)` for PID1367350, startTicks42904077. The retained
native receipts identify that exact PID/start time in that exact cgroup as the
candidate supervisor. Its observed affinity was correctly CPU8.

Classify supervisor observations using the bound PID, process start time,
cgroup and unit from native receipts. Command text may change across exec and
must not decide process role. Keep every affinity observation and all mask,
process-ownership, output and economics checks. Focused tests reject PID reuse,
a different PID with the worker name, wrong cgroup, wrong mask and missing masks.

The original analyzer, controller failure, freeze and all native inputs remain
unchanged. New `analyze-v2.py` and `complete-v2.py` are separately bound. Reuse the
completed8-CPU run; continue the unchanged4/2-CPU profiles and conditional disjoint
history with the corrected reader. No new Gradle/JVM starts for this correction;
no changes to candidate, thresholds, profile order, warmups or allocation.
