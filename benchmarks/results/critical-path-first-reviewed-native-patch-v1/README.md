# Critical-path-first reviewed native patch v1

This evidence closes `CPFRNP-001..007` at the diagnostic gate. The frozen
campaign ran one owner-workflow native diagnostic for each of ten previously
unused public Gradle families. Four workflows completed and six produced typed
environment or workflow failures. The four complete task-DAG traces contain 19
tasks above both 500 ms and 2% of their build span; every one is a standard
Gradle or Kotlin task, so no repository-owned task qualifies for source review.

The gate therefore records 4/10 conclusive families and 0/10 proposal families,
below the required 10/10 and 3/10. No public source was inspected or modified,
no candidate build or timing sample ran, and correctness, value, owner review,
and delivery remain closed.

`diagnostic-summary.json` contains the typed family rows, gate counts, protocol
corrections, and complete bounded campaign charge. `terminal-decision.json`
binds the terminal stop. `diagnostics/` retains each raw capture, compressed
operation trace and log, task graph, and the four replayable analyses.

Run `./dev/check-critical-path-first-reviewed-native-patch-evidence` to verify
the frozen revisions and arguments, reconstruct the family/count decisions,
replay all successful critical-path analyses, and enforce the zero-candidate
boundary.
