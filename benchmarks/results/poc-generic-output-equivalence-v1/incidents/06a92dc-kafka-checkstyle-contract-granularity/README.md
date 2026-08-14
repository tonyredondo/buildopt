# Kafka Checkstyle output-contract granularity incident

BuildOpt revision `06a92dce02cb70b635142b1b38f4db934e1b551c` produced two
complete diagnostic Groovy captures, then stopped the first Kafka Checkstyle
capture before pair 1. The reviewed contract applied
`REPOSITORY_ROOT_TEXT` independently to both `main.html` and `main.xml`.
Kafka's HTML report contains no isolated checkout root, so the strict
canonicalizer rejected the unused relocation rule. This was a contract-shape
error, not output drift or a negative performance result.

The superseded Groovy captures are retained because they demonstrate why the
whole matrix must restart after a preregistration correction. They recorded
16/16 positive pairs, semantic output equivalence, and two valid fallbacks,
but they are not terminal evidence for the revised protocol.

The correction leaves `main.html` unruled and therefore byte-exact, while
`main.xml` alone receives the narrow repository-root relocation. No workflow,
task, threshold, pair, timing boundary, payload rule, or fallback changed.
All terminal captures restart from zero on one later immutable BuildOpt SHA.
