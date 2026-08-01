# Patch candidate validation v1

This POC contract closes C4-005 with a production Java 17 validator for the
isolated candidate/control and artifact boundary. It does not close C4-G02
until C4-006 supplies FULL_RELEVANT_VALIDATION where applicable.

Each validation contains exactly six runs: candidate and control for CLEAN,
INCREMENTAL, and RELOCATED phases. Every run must use an identical repository,
action, revision, source-state, work-units, deliverables, policy, toolchain,
and runner context; each must have a distinct isolation key and a successful
exit. Clean and relocated runs must store Configuration Cache state, while
incremental runs must reuse it.

The required deliverables manifest is closed: every run must contain exactly
the declared relative paths. Inputs are bounded and content is digested inside
the validator. The finite adapters are:

- ARCHIVE_CONTENTS_V1 for both archive-reproducibility DSL recipes and the
  build-cache properties recipe. ZIP-family entry names, directory markers,
  uncompressed sizes, and content digests are compared independent of
  timestamp, compression, and entry order. Unsafe, duplicate, empty,
  oversized, or over-expanded archives fail.
- EXACT_BYTES for CUSTOM_TASK_CONTRACT_JAVA_V1. It remains available to the
  validator, but does not implement or qualify the C1-gated recipe.

All six logical artifact sets must equal control-clean. In addition, all three
candidate raw artifact sets must be byte-identical, proving clean,
incremental, and relocated reproducibility. A failed run, Configuration Cache
failure, invalid artifact, logical divergence, or candidate byte divergence
returns FAILED. Missing, duplicated, context-drifted, non-isolated, or
unsupported observations return INCONCLUSIVE. Only PASSED exposes the three
content-addressed result digests.

Run ./dev/check-patch-candidate-validation.

The checker verifies the machine-readable contract and executes the production
validator cases plus the complete signed PatchBundle and real-Git recovery
matrix. It performs no customer build, remote mutation, or test selection.
