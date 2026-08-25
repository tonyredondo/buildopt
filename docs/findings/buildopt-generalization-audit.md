# BuildOpt Generalization Audit

## Audit question

Can the retained BuildOpt POC run unchanged on an arbitrary Gradle repository,
find a safe optimization and beat optimized native Gradle without
repository-name rules?

The current answer is: **not with the complete structural-profile unit tested
so far**. Target qualification produced substantial savings on several public
repositories, but the later chronological lifetime evaluation found repeatable
net value in only one of five families. One of six structurally eligible
descendants selected a profile, so the frozen terminal decision is
[`STOP_GENERIC_POC`](../../benchmarks/results/poc-functional-coverage-decision-v1/README.md).

That decision stops the current whole-profile hypothesis, not the bounded
mechanisms that worked. The next preregistered direction is an
[adaptive fragment model](../plans/adaptive-fragment-generalization-tracker.md)
that independently retains, invalidates and composes producer, subgraph, task,
patch and cache-locality evidence.

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

## Historical target-qualification evidence

| Repository | Automatic graph | Direct timed effect | Economics / decision |
| --- | ---: | ---: | --- |
| Spring Framework | 27 -> 10 | 12.71% faster, 7/8, positive interval | 88.668-s learning; 67-build payback; native retained. |
| OpenTelemetry Java Instrumentation | 1,024 -> 34 | 14.97% faster, 8/8, positive interval | 201.913-s learning; 19-build payback; qualified. |
| Apache Kafka | 64 -> 3 | 54.92% faster, 8/8, positive interval | 70.808-s learning; 14-build payback; qualified. |
| Micronaut Core | 75 -> 22 | 66.24% faster, 8/8, positive interval | 114.284-s learning; 8-build payback; qualified. |
| Apache Groovy | 37 -> 2 | 75.97% faster, 8/8, positive interval | 73.857-s learning; 2-build payback; qualified. |

All 85 ordinary invocations preserve exact required outputs and pass full-graph
fallback. This remains useful mechanism evidence because the target
repositories contain no BuildOpt files and the same binary makes all five
decisions. It is not the current lifetime conclusion: target qualification does
not prove that a profile will recur often enough across later commits.

## Current chronological lifetime evidence

| Repository | Requested qualification builds | Later selection | Signed lifetime net |
|---|---:|---:|---:|
| Spring Framework | 1 | 0 / 0 | **-10.113 s** |
| OpenTelemetry Java Instrumentation | 1 | 0 / 0 | **-9.961 s** |
| Apache Kafka | 17 | **1 / 6 eligible** | **+82.527 s** |
| Micronaut Core | 1 | 0 / 0 | **-9.149 s** |
| Apache Groovy | 1 | 0 / 0 | **-2.760 s** |

The sequence uses 21 requested builds and zero measurement-only builds. It
verifies 27 exact-output observations with zero product failures. Kafka's one
selected descendant saves 135.127 seconds, but five other eligible descendants
retain native Gradle and incur measured wrapper cost. The complete frozen gate
therefore fails repository-family breadth, selection coverage and native-
retention overhead even though the signed cross-repository total is positive.
Repository percentages are not averaged and Kafka's gain does not override the
four negative family outcomes.

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

The detailed order, outcomes and documentation obligations live in the
[Adaptive Fragment Generalization POC Tracker](../plans/adaptive-fragment-generalization-tracker.md).
Its first proof must happen before another broad timing campaign:

1. replace a complete-profile decision with independently compatible fragments;
2. prove fragment applicability over the frozen histories without lookahead;
3. make no-value selection cheaper than 500 ms median and 1,000 ms p95;
4. learn value and decay from ordinary builds using a signed economic ledger;
5. activate and directly measure only compositions whose individual fragments
   retain correctness and positive value authority; and
6. rerun the five chronological repository families only if shadow coverage
   reaches at least 50% of eligible descendants.

## POC conclusion

BuildOpt's defensible idea remains an evidence-gated optimizer on top of native
Gradle, not a faster reimplementation of Gradle's cache. The current complete-
profile implementation demonstrates bounded target value and strong safety,
but not generic lifetime customer value. The new fragment hypothesis must
increase cross-commit selection coverage and cumulative net savings without
repository-specific rules, weaker output gates or slower native retention. If
it cannot, the generic POC should stop rather than reinterpret the evidence.
