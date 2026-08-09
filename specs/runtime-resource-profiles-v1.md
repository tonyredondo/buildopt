# Runtime resource profiles v1

This POC contract closes `B-003`. The initial catalog is finite and contains
exactly `STABLE_CONTROL`, `W2_H3G`, `W3_H4G`, and `W4_H6G` for the existing
`golden-linux-amd64-4c-16g-v1` runner catalog. No values are synthesized or
scaled for a different machine.

All four arms retain the exact runner, build, Gradle/JDK, compiler, process,
and cgroup identity from the normative resource-profile fixtures. They vary
only `--max-workers` and Gradle daemon heap. Selection requires the signed
profile ID, digest, and catalog version plus exact runner/build/compatibility,
JDK, and cgroup identity. Insufficient live memory headroom makes the arm
ineligible with zero opportunity to alter argv.

The launcher applies an explicitly authorized `RESOURCE_PROFILE` before Gradle
starts. It replaces only caller worker and daemon-heap arguments. Unknown
catalogs, arbitrary arms, identity drift, malformed capacity, or insufficient
headroom preserve the original command and report the unavailable profile.

This block proves finite selection, materialization, startup/memory/rollback
eligibility, and launcher fallback. Real causal improvement and p95/p99/queue/
OOM evidence remain with `B-G03` and owner-operated evaluation.

The executable implementation and checker were retired after Runtime Tuning
failed the POC value gate. This document is retained as historical protocol
evidence only.
