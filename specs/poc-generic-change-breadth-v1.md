# Generic change-shape breadth

## Purpose

`POC-GENERIC-CHANGE-BREADTH-001` tests whether the reviewed Structural Build
Impact path transfers beyond one exact source edit. It preserves the optimized
native Gradle baseline and the output-equivalence rules already qualified for
Groovy JARs, Kafka Checkstyle reports, and Kafka shadow JARs.

The experiment has ten independent cells across two public repositories and
three workflow families:

- six selective cells cover two distinct source edits per workflow, including
  a Groovy root-library change and a Kafka generator change consumed by the
  requested output;
- two build-logic cells must retain the complete owner workflow; and
- two global-configuration cells must retain the complete owner workflow.

Candidate lifecycle tasks are derived from the reviewed output owners, not a
repository name or a hard-coded project. The generated graph must still prove
that those tasks cover both the changed source and every declared output. An
unsupported relationship therefore retains native Gradle.

## Frozen method

Every selective cell receives two fresh eight-pair captures. Adjacent opposite
orders form eight reciprocal blocks. A cell qualifies only with at least 500 ms
and 2% mean saving, positive median and deterministic 95% lower bound, at least
six positive blocks, non-regressive candidate p95, stable task shape, equivalent
required outputs, two successful full-graph fallbacks, and zero product failure.
No percentage is averaged across cells.

Fallback cells are intentionally not timed. Each receives two independent
installed-path proposals. Both must run and validate the original owner
workflow, return `NATIVE_FULL_GRAPH / GLOBAL_CHANGE_REQUIRES_FULL_GRAPH`, emit
no candidate documents, and preserve the reviewed output contract. Treating a
full-graph fallback as a performance win would be misleading.

All paths, public revisions, original file digests, mutations, workflows,
output contracts, pair orders, and thresholds are frozen in
[`poc-generic-change-breadth-v1.json`](./poc-generic-change-breadth-v1.json)
before terminal timing.

## Boundaries

This is an owner-reviewed POC. It creates no automatic activation, production
authority, repository-name branch, Test Optimization behavior, soak
requirement, or design-partner dependency. Every unfavorable or unavailable
cell is retained and remains on optimized native Gradle.

## Validation

```bash
./dev/check-generic-change-breadth
```

Terminal evidence is assembled and checked separately after all captures
exist.
