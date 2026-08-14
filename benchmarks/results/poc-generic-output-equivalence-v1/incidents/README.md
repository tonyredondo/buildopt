# Generic Output-Equivalence Incident Index

The terminal result uses only captures produced by BuildOpt revision
`55ad6bdd4dbd0320cac6102dd0b86be250d518f6`. Earlier attempts are retained here
so collection failures and diagnostic corrections cannot disappear from the
record.

| BuildOpt revision | Workflow | Observation | Disposition |
| --- | --- | --- | --- |
| [`86fe7c5`](./86fe7c5-groovy-console-path-collision/README.md) | Groovy `jar` | Gradle emitted one local task path from more than one build tree. | No timed pair; generalized task outcome identity to include the build-tree-local observation set. |
| [`bfb7578`](./bfb7578-groovy-console-transition-fingerprint/README.md) | Groovy `jar` | One task printed a transient line before its terminal `FROM-CACHE` outcome. | Retained all eight diagnostic pairs; terminal outcomes, not transient rendering, now define the fingerprint. |
| [`06a92dc`](./06a92dc-kafka-checkstyle-contract-granularity/README.md) | Kafka Checkstyle | A relocation rule incorrectly required the repository root in byte-exact `main.html`. | Retained superseded Groovy captures and the zero-pair Kafka stop; narrowed relocation to `main.xml`. |
| [`99d7785`](./99d7785-groovy-target-warmup-convergence/README.md) | Groovy `jar` | One changed-target warm-up had not reached the stable measured shape. | Retained all diagnostic pairs; added two bounded confirmations and fail-closed convergence. |
| [`eb873f5`](./eb873f5-groovy-candidate-target-convergence/README.md) | Groovy `jar` | The final two candidate confirmations differed by one executed versus cached task. | No timed pair; retained the stop and added bounded task-level convergence diagnostics. |

No incident timing contributes to `summary.json`. No correction changed a
workflow, output exception, pair order, value threshold, tail guard, fallback,
or product boundary. Every terminal capture restarted from zero on the final
revision.
