# General build value and qualified composition

## Purpose

This POC contract answers a product-level question: does the complete BuildOpt
path improve a real build after every mechanism with relevant evidence for that
exact target is enabled together? The control is the target's optimized native
Gradle path. A fast isolated component is not treated as an additive saving,
and a faster terminal arm cannot hide a regressive included mechanism.

The contract does not rerun unchanged experiments. The product execution path
did not change after the source measurements, so rerunning the same revisions,
mutations, tasks and mechanisms would spend compute without testing a new
hypothesis. Instead, the checked scorecard binds the three direct end-to-end
composition results by SHA-256 and validates their calculations, outputs,
fallbacks and later replication status.

## Target compositions

| Target | Enabled mechanisms | Direct whole-path result | Later installed replication |
| --- | --- | ---: | --- |
| Spring Framework | Build Impact | 2,492.375 ms / 30.86% faster, 8/8 positive | 1,895 ms / 14.33% faster, 7/8 positive; retained native under the frozen all-positive gate |
| OpenTelemetry Java Instrumentation | Build Impact + exact standard `Jar` adapter | 5,361.25 ms / 50.40% faster, 4/4 positive | preparation failed before accepted timing; no replication claim |
| Apache Kafka | Build Impact + read-only Edge | 35,405.5 ms / 82.35% faster, 4/4 positive | 28,523.25 ms / 81.85% faster, 8/8 positive; explicit reviewed POC profile retained |

These percentages are independent target results. They are never averaged or
added. OpenTelemetry deliberately excludes Hot State because its direct arm
regressed by 892 ms / 7.68%. Runtime Tuning, Safe Cache, standard `Copy` and
Test Optimization are absent because they did not qualify for these exact
workloads. OpenTelemetry Build Impact was positive but did not independently
clear its stability gate; it remains the required graph-selection layer inside
the directly qualified Impact-plus-Jar composition, not a separately credited
effect. The Kafka composition includes source normalization as an input
precondition, not as an independently credited accelerator.

## Structural opportunity analysis

The installed command below inspects a complete digest-bound Build Impact
manifest, graph and generated-state document:

```bash
buildopt profile analyze \
  --manifest buildopt-impact-manifest.json \
  --graph buildopt-impact-graph.generated.json \
  --generated-manifest buildopt-impact.generated.json
```

It emits `MEASURE_STRUCTURAL_CANDIDATE` only when a reviewed alternative reaches
fewer projects than the original build, the graph is complete, every relation
is known, generated state is bound, and the selected entrypoint contains no
Gradle `Test` task. The output quantifies selected and omitted projects but does
not predict a percentage. Only Build Impact becomes a measurement candidate;
every other mechanism requires separate task-shape, locality or runtime
evidence. The command is read-only, review-required and cannot activate a
profile.

This separates a general product capability from repository-specific value:
BuildOpt can find structural opportunities across repositories, while direct
measurement decides whether the opportunity is worth using on each workload.
Unknown or incomplete state always returns `NATIVE_FULL_GRAPH`.

## Decision

All three exact compositions have a positive direct result, but only Kafka has
also passed the terminal installed-profile replication gate. The correct POC
decision is therefore to continue structural generalization and fresh
cross-repository experiments while retaining optimized native Gradle by
default. The data supports promising exact scopes, not a universal accelerator
claim or automatic activation.

Validate the command, scorecard, source digests and calculations with:

```bash
./dev/check-poc-opportunity-analysis
./dev/check-poc-general-build-value
```

Production rollout, soak testing, design-partner operation and Test
Optimization remain outside this block.
