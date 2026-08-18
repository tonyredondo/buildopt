# Automatic qualified-profile replay

## Decision

`buildopt optimize` may execute a smaller qualified graph only when the same
owner invokes the POC command again and every stored binding still matches
before Gradle starts. This is POC-only execution authority. It never grants
production promotion or changes an ordinary `gradlew` invocation.

## Exact replay boundary

Automatic replay deliberately starts with exact revisions. A later invocation
must accept the previous checkpoint and validate all of these bindings:

| Binding | Fail-closed meaning |
| --- | --- |
| Repository identity and revision | A different repository or commit runs the original workflow. |
| BuildOpt executable | A different launcher cannot inherit an earlier decision. |
| Wrapper properties | A Gradle distribution change disables the profile. |
| Workflow and Gradle options | Added, removed or reordered execution options retain native Gradle. |
| Structural family | Changed owners, candidate entrypoints or required outputs must still identify the same family. |
| Discovery documents | Changes, manifest, graph, generated manifest and output contract remain digest-bound. |
| Qualification evidence | The eight-pair evidence and its recomputed summary remain exact. |
| Profile artifacts and preconditions | Every private portfolio artifact and SHA-256 precondition still passes. |

Cross-commit structural matching is intentionally not inferred in this block.
It requires a later contract that can prove current graph and output semantics
without spending more time than the saved build work.

## Execution and reporting

Selection completes before the target Gradle process starts. The result records
nanosecond selection overhead, original and selected entrypoints, every checked
binding, any failed binding, and `source=LOCAL_PORTFOLIO` for this exact local
replay contract. Central portfolio reuse has its own stricter contract and
reports a distinct source. A selected execution reports
`SELECTIVE_PROFILE`; the `native` object explicitly records that the full graph
did not start. A rejected or unavailable selection reports
`OPTIMIZED_NATIVE` and preserves the original workflow and exit status.
An invocation with no qualified checkpoint reports
`SKIPPED / NO_QUALIFIED_CHECKPOINT`; it is not mislabeled as a failed replay.

The first qualifying invocation still runs optimized native Gradle and creates
the portfolio entry. An exact subsequent invocation selects it without an
extra flag or another calibration. A corrupt artifact is rejected before
Gradle, the current invocation runs native, and still-valid evidence may repair
the portfolio for the following invocation.

## POC boundary

The replay is useful only inside the explicitly invoked `buildopt optimize`
command, keeps `productionAuthorized=false`, and leaves Test Optimization out
of scope. It proves safe automatic activation and measurable decision overhead;
it does not claim that the synthetic calibration percentage transfers to real
repositories, require a soak, or depend on a design partner.

The machine-readable contract is
[`poc-magic-auto-replay-v1.json`](./poc-magic-auto-replay-v1.json).
