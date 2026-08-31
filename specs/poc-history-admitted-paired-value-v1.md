# History-Admitted Paired Value v1

Status: `HAPV-001..004` complete;
`QUALIFY_EXACT_HISTORY_ADMITTED_KAFKA_VALUE`.

The exact HACC Kafka row runs through installed BuildOpt from empty state for
seventeen ordinary requests: one discovery baseline and eight balanced
alternating optimized-native/candidate pairs. The complete BuildOpt duration,
including discovery, materialization and verification, is charged. Workspaces
are cleaned before each request while private Gradle and BuildOpt state remain.

Every pair must reproduce the same 4,440 required outputs. Qualification needs
8/8 positive pairs, at least 500 ms and 2% mean saving, positive paired 95%
lower bound, non-regressive candidate p95, payback within five compatible
matches and zero product failures. Historical timings supply no result. The
route is capped at seventeen Gradle starts, 30 minutes each and 12 workers.

The fresh result passes every gate: 8/8 positive pairs, 6,866.125 ms mean
saving (19.09%), paired 95% interval +5,717.25..+8,204.5 ms, candidate p95
31,310 ms versus native 38,695 ms, and two-match payback after charging 7,242
ms of learning cost. Every arm reproduced 4,440 outputs and product failures
were zero. This qualifies the exact history-admitted Kafka class, not generic
repository breadth or production activation.
