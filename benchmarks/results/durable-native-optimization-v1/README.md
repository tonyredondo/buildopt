# Durable native optimization evidence

`opportunity-report.json` is the fresh `DNO-002` source audit over the five
public revisions frozen in
`specs/poc-durable-native-optimization-v1.subjects.json`. The analyzer uses
only the source structure required by the contract; repository and task names
are labels, not decision inputs.

The result is 5/5 conclusive revisions, four action-bearing families and eight
digest-bound custom-task candidates. Kafka is a conclusive no-action family
because its custom tasks already carry cacheability markers. The three catalog
classes without a complete compiler remain explicitly unavailable.

This report contains no Gradle execution, candidate timing or speedup claim.
Recompute it from prepared public histories with:

```bash
./dev/check-durable-native-opportunity-audit \
  --source-root /path/to/five-public-histories
```

Without `--source-root`, the checker validates the checked evidence, analyzer
digest, unit tests and negative breadth fixture without requiring local public
clones.

`patch-plan.json` is the `DNO-003` transaction evidence. All eight candidates
compile to the same generic fully qualified annotation, apply idempotently and
revert to the exact original source bytes. Drift and ambiguous declarations
fail closed. The plan still contains no candidate execution or timing.

`correctness.json` is the terminal `DNO-004` correctness result. Spring and
OpenTelemetry preserve their required outputs exactly and restore the patched
tasks from Gradle's build cache. Micronaut fails Gradle validation because an
`@InputDirectory` lacks a normalization strategy. The marker-only compiler is
not authorized to infer that missing semantic contract, so the frozen
zero-product-failure gate fails at one failure. Groovy and the remaining
fixture matrix were not run after the stop; they are not represented as zero.
`DNO-005` and `DNO-006` are consequently not authorized, and this evidence
makes no wall-time claim.

Validate the checked result and the external-checkout patch driver with:

```bash
./dev/check-durable-native-correctness
```
