# BuildOpt Generalization Audit

## Audit question

Can the retained BuildOpt POC run unchanged on an arbitrary Gradle repository,
find a safe optimization and beat optimized native Gradle without
repository-name rules?

The current answer is: **the generic path runs and fails closed, but it does
not yet deliver broad automatic economic value**. The latest transfer observes
five different public repositories with zero manual BuildOpt files and zero
product failures. It discovers four complete candidates, yet none repays the
synchronous learning cost within 30 matching builds.

## What is generalized today

| Layer | Generic behavior | Current boundary |
| --- | --- | --- |
| Installation / launcher | Native packages locate the Wrapper and preserve argv, exit and signal behavior. | Linux, macOS and Windows lifecycle is tested; comparable wall-time breadth evidence is Linux. |
| Change and workflow discovery | Derives provider/local revisions, exact changed paths and requested Gradle entrypoints. | Global/build-logic ambiguity retains native. |
| Output discovery | Reads Gradle-owned outputs and rejects missing, external, symlinked or ambiguous declarations. | A root aggregate workflow can legitimately declare a very broad output surface. |
| Structural proposal | Uses typed project/task relationships and changed-project ownership; no repository-name branch is allowed. | Unknown relationships, excessive candidate task sets and no reduction retain native. |
| Measurement / decision | Alternating native/candidate observations verify outputs, execution shape, interval, fallback and payback. | The current synchronous eight-pair method can cost more than the optimization can repay. |
| Portfolio / central state | Reuses exact compatible evidence across checkouts or machines. | Reuse cannot infer lifetime or value from another profile/repository. |
| Gradle-compatible cache | Supports local and optional HTTP/HTTPS reuse with safe miss/outage behavior. | Cache is supporting infrastructure; native-cache parity is not a speed advantage. |

Runtime Tuning, Hot State and standard Copy remain retired. The standard `Jar`
adapter and Patch Autopilot retain only their exact qualified scopes.

## Latest unchanged breadth evidence

| Repository | Automatic graph | Direct timed effect | Economics / decision |
| --- | ---: | ---: | --- |
| Spring Framework | 27 -> 10 | 26.83% faster, 7/8, positive interval | 339.603-s learning; 103-build payback; native retained. |
| OpenTelemetry Java Instrumentation | 1,024 -> 34 | 20.25% faster, 8/8, positive interval | 1,555.444-s learning; 101-build payback; native retained. |
| Apache Kafka | 64 -> 36 | 13.42% faster, 3/8, interval crosses zero | 374.762-s learning; 190-build payback; native retained. |
| Micronaut Core | 73 candidate entrypoints | Not timed | Candidate surface too large; native retained before calibration. |
| Apache Groovy | 37 -> 30 | 2.81% faster, 7/8, positive interval | 1,423.987-s learning; 710-build payback; native retained. |

All timed candidates preserve exact required outputs and pass full-graph
fallback. This is stronger generalization evidence than a manually reviewed
profile matrix because the target repositories contain no BuildOpt files and
the same binary makes all five decisions. It is also less flattering: older
Kafka/Groovy/Micronaut profiles narrowed the output contract manually and must
not be presented as current zero-configuration results.

## Why graph reduction alone is insufficient

Three distinct constraints appear in the current data:

1. **Learning economics.** OpenTelemetry omits 990 projects and saves 15.407
   seconds per build, but sixteen timed arms plus stabilization cost 1,555.444
   seconds. A real reduction can still be a bad customer transaction.
2. **Complete outputs.** Kafka and Groovy aggregate workflows require outputs
   from many otherwise unaffected projects in a clean workspace. Omitting
   those producers without materializing verified outputs would make the build
   faster but wrong.
3. **Aggregate task breadth.** Micronaut `assemble` yields 73 candidate
   entrypoints. Raising the limit would spend more time without proving a
   useful partition.

The next implementation must address these generically. It may reason about
task producers, variants, ABI relationships, output digests and cache
materialization; it may not branch on repository identity or borrow an old
profile's expected percentage.

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

1. Persist calibration as exact-bound incremental observations across ordinary
   invocations, rather than duplicating sixteen builds synchronously.
2. Materialize unaffected required outputs from verified Gradle-compatible
   cache/state before selecting a smaller graph in a clean workspace.
3. Partition aggregate workflows into bounded producer groups using generic
   task/variant/ABI relationships; retain native when completeness is unknown.
4. Rerun the same frozen five-repository contract without moving output,
   fallback or payback gates.

## POC conclusion

BuildOpt's defensible idea remains an evidence-gated structural optimizer, not
a faster reimplementation of Gradle's cache. The mechanism can produce large
wall-time wins, including current public Ktor/Beam results, and the generic
automatic path is safe. The latest breadth evidence shows that safety is not
enough: the POC still needs cheaper learning and verified output
materialization before the one-command experience can claim general customer
value.
