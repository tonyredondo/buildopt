# Reviewed cross-date output equivalence

## Question

Can BuildOpt compare qualified outputs across calendar dates without weakening
the default byte-identity boundary or changing any historical performance
claim?

## Frozen method

The prior public replay retained six qualified structural cells. Four Kafka
cells already matched their terminal semantic digest across independent
captures. The two Groovy cells did not: the reviewed JAR contract declared
`BuildTime` volatile, while Groovy's release build also writes `BuildDate` from
the current clock into the same properties entry.

This follow-up adds exactly `BuildDate` beside `BuildTime` in that one entry.
It does not add a generic timestamp heuristic, inspect repository identity, or
ignore arbitrary ZIP metadata. A deterministic boundary probe copies a real
JAR built from the frozen public revision and changes only the declared date.
The old contract must report a mismatch and the reviewed contract must match.
Separate mutations of `ImplementationVersion` and a non-metadata ZIP payload
must still report mismatches.

The controlled date mutation is a reproducible equivalence test, not a second
performance observation and not a claim that two builds were timed on
different days. It complements the four natural Kafka cross-capture matches
and the prior same-checkout native-versus-BuildOpt Groovy equivalence.

## Qualification boundary

All six historical timing qualifications and their evidence digests remain
unchanged. Their profiles keep the exact contracts against which they were
qualified. The broadened reviewed fixture applies to future evidence; this
block does not rewrite a historical contract SHA or re-estimate savings.

The machine-readable preregistration is
[`poc-cross-date-output-equivalence-v1.json`](./poc-cross-date-output-equivalence-v1.json).
