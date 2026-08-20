# BuildOpt Generalization Audit

## Audit question

Can the retained BuildOpt POC run unchanged on an arbitrary Gradle repository,
find a safe optimization and beat optimized native Gradle without
repository-name rules?

The current answer is: **yes for four of five tested repository/workflow
families, with safe native retention for the fifth**. The latest transfer runs
one exact binary on five different public repositories with zero manual
BuildOpt files and zero product failures. All five candidates beat optimized
native Gradle and preserve exact outputs; OpenTelemetry, Kafka, Micronaut and
Groovy pass 8/8 pairs and repay learning within 30 matching builds. Spring
improves but remains native because it reaches only 7/8 and 67-build payback.

## What is generalized today

| Layer | Generic behavior | Current boundary |
| --- | --- | --- |
| Installation / launcher | Native packages locate the Wrapper and preserve argv, exit and signal behavior. | Linux, macOS and Windows lifecycle is tested; comparable wall-time breadth evidence is Linux. |
| Change and workflow discovery | Derives provider/local revisions, exact changed paths and requested Gradle entrypoints. | Global/build-logic ambiguity retains native. |
| Output discovery | Reads Gradle-owned outputs and rejects missing, external, symlinked or ambiguous declarations. | A root aggregate workflow can legitimately declare a very broad output surface. |
| Structural proposal | Uses typed project/task relationships and changed-project ownership; no repository-name branch is allowed. | Unknown relationships, excessive candidate task sets and no reduction retain native. |
| Measurement / decision | Alternating native/candidate observations verify outputs, execution shape, interval, fallback and payback. | Observations now accumulate across useful invocations with zero measurement-only workflows; weak value still retains native. |
| Verified output materialization | Captures required outputs omitted by a candidate in digest-bound private state, then restores only exact missing bytes before candidate execution. | Composed and timed on all five public subjects; stale, missing or corrupt payloads cannot authorize candidate output. |
| Aggregate workflow partition | Groups directly changed output producers by generic lifecycle selector and variant, while exact unaffected outputs remain materializable. | Transfers to public workflows: Kafka selects 3/64 projects, Micronaut 22/75 and Groovy 2/37. |
| Portfolio / central state | Reuses exact compatible evidence across checkouts or machines. | Reuse cannot infer lifetime or value from another profile/repository. |
| Gradle-compatible cache | Supports local and optional HTTP/HTTPS reuse with safe miss/outage behavior. | Cache is supporting infrastructure; native-cache parity is not a speed advantage. |

Runtime Tuning, Hot State and standard Copy remain retired. The standard `Jar`
adapter and Patch Autopilot retain only their exact qualified scopes.

## Latest unchanged breadth evidence

| Repository | Automatic graph | Direct timed effect | Economics / decision |
| --- | ---: | ---: | --- |
| Spring Framework | 27 -> 10 | 12.71% faster, 7/8, positive interval | 88.668-s learning; 67-build payback; native retained. |
| OpenTelemetry Java Instrumentation | 1,024 -> 34 | 14.97% faster, 8/8, positive interval | 201.913-s learning; 19-build payback; qualified. |
| Apache Kafka | 64 -> 3 | 54.92% faster, 8/8, positive interval | 70.808-s learning; 14-build payback; qualified. |
| Micronaut Core | 75 -> 22 | 66.24% faster, 8/8, positive interval | 114.284-s learning; 8-build payback; qualified. |
| Apache Groovy | 37 -> 2 | 75.97% faster, 8/8, positive interval | 73.857-s learning; 2-build payback; qualified. |

All 85 ordinary invocations preserve exact required outputs and pass full-graph
fallback. This is stronger generalization evidence than a manually reviewed
profile matrix because the target repositories contain no BuildOpt files and
the same binary makes all five decisions. Percentages remain row-specific.

## Why graph reduction alone is insufficient

Three distinct constraints remain visible in the current data:

1. **Learning economics.** Spring saves 1.329 seconds per build but spends
   88.668 seconds learning, so a real reduction can still be a bad customer
   transaction.
2. **Complete outputs.** Kafka and Groovy aggregate workflows require outputs
   from many otherwise unaffected projects in a clean workspace. Omitting
   those producers without materializing verified outputs would make the build
   faster but wrong.
3. **Aggregate task breadth.** The former Micronaut 73-entrypoint rejection is
   now handled through generic partitioning, but incomplete producer/output
   relationships must still retain native on unknown repositories.

The implementation addresses these shapes through task producers, lifecycle
selectors, variants and exact output relationships. It does not branch on
repository identity or borrow an old profile's expected percentage.

## Implementation-generic versus evidence-specific

Repository names and frozen mutations are valid in runners and immutable
evidence because the experiment must identify what was tested. Product code
reasons only about:

- typed projects, tasks, dependencies, variants and entrypoints;
- exact change ownership and global-change rules;
- required outputs and their producer/digest relationships;
- repository, revision, Wrapper, graph, workflow, output and executable
  bindings; and
- measured wall time, uncertainty, failures, fallback and payback.

No latest terminal decision depends on a repository-name rule. A new Gradle
repository can run `buildopt optimize <workflow>` and receives either a
measured candidate or an explicit native verdict.

## Next generalization steps

1. Reduce Spring's remaining learning/materialization cost through generic
   unchanged-content reuse, without moving output, fallback or payback gates.
2. Replay the four qualified profiles over compatible public descendants and
   measure cumulative value, selection cost, invalidation and fallback.
3. Add new holdout repository/workflow families only after the lifetime result,
   using the unchanged command and no repository-specific product rules.

## POC conclusion

BuildOpt's defensible idea is an evidence-gated structural optimizer, not a
faster reimplementation of Gradle's cache. The latest unchanged automatic path
now demonstrates general POC value on four unrelated repository/workflow
families and safe rejection on a fifth. The next proof must show that this
qualification pays back across later compatible changes; calibration speedup
alone is not lifetime customer value.
