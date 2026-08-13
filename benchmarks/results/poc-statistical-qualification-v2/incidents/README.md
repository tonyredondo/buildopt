# Preserved Qualification Incidents

These directories preserve every failed or superseded attempt observed while
building the terminal v2 evidence. They are excluded from the terminal timing
matrix for documented methodological reasons; they are not silently deleted or
reclassified as BuildOpt value results.

| Incident | Classification | Why it is excluded from terminal timing |
| --- | --- | --- |
| `spring-framework-capture-1-prior-to-distribution-seed` | External distribution boundary | Valid earlier timing was produced before Gradle distribution isolation was fixed; all subjects restarted on one later exact BuildOpt revision. |
| `spring-framework-capture-2-wrapper-download-eof` | External network failure | Wrapper download ended unexpectedly before a complete capture. |
| `spring-framework-capture-2-candidate-wrapper-download-eof` | External network failure | Candidate warm-up could not obtain the Wrapper distribution; no complete two-capture result existed. |
| `spring-framework-terminal-preflight-wrapper-eof-932e313` | External network failure | Original-workflow preflight failed before measurement; exact Wrapper preparation was moved before proposal. |
| `spring-framework-terminal-pair-85856a1-before-otel-preflight-fix` | Superseded revision | Both captures were valid, but the later generic OpenTelemetry preflight correction required every terminal subject to restart from one identical BuildOpt revision. |
| `opentelemetry-capture-1-output-contract-configuration-cache-85856a1` | Generic preflight defect | The temporary output-contract task accessed project state at execution under Configuration Cache. Proposal now explicitly uses the already frozen no-Configuration-Cache mode. No timed observation existed. |
| `spring-framework-capture-1-duplicate-launch-b8fd0f6` | Orchestration error | A duplicate operator launch was stopped before producing an accepted capture. |
| `spring-framework-capture-1-empty-proposal-b8fd0f6` | Orchestration error | An empty proposal artifact was detected and preserved; no timing was accepted. |
| `opentelemetry-capture-1-interrupted-session-b8fd0f6` | Operator interruption | The interrupted capture was preserved and replaced by a complete fresh capture on the same frozen revision. |
| `apache-kafka-capture-1-invalid-queue-key-b8fd0f6` | Orchestration error | The queue used an invalid repository key and exited before proposal or measurement. |

The terminal matrix uses only the two complete `capture-{1,2}` directories per
repository. Its checker requires the same BuildOpt revision, executable,
repository revision, plan, Gradle options, exact outputs, stable measured task
shape, successful fallbacks, and zero product-attributable failures.
