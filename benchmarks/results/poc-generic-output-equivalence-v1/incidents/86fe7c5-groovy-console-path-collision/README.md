# Groovy console task-path collision

The first Groovy capture on preregistration revision `86fe7c5` completed the
native target warm-up but produced no timed observation. Gradle's plain console
reported the local task identifier `:compileJava` with both `EXECUTED` and
`FROM-CACHE` outcomes in the same build tree. The original parser treated one
path with multiple outcomes as malformed exact evidence and stopped before the
first pair.

This is a generic composite/included-build representation boundary, not an
optimization result. Gradle console task paths are local to a build and are not
globally unique across the build tree. The correction preserves the sorted set
of observed outcomes on that path in the evidence fingerprint and normalizes
its counter to the conservative outcome (`EXECUTED`). Repeated identical lines
remain deduplicated. The output-equivalence contract, workflows, thresholds,
pairing, and fallbacks did not change.

All terminal captures restart from zero on the correction revision. The failed
result and its logs remain here; it is not counted as a performance sample.
