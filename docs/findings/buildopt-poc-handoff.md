# BuildOpt POC: One-Page Handoff

## The idea

BuildOpt tests whether a generic decision layer can make substantial Gradle
builds faster than an already optimized native Gradle baseline. Gradle remains
the execution engine and safe fallback. BuildOpt inspects the exact Git change
and requested workflow, derives a smaller sufficient graph, measures the
complete installed path, verifies required outputs and reuses only evidence
that remains structurally and economically valid.

This is an owner-operated proof of concept, not a production product. Soak,
design-partner evidence, production SLOs, autonomous promotion and Test
Optimization are outside the current scope.

## Intended customer experience

```text
install BuildOpt
cd <a Gradle repository>
buildopt optimize <the existing Gradle workflow>
```

The target repository should need no BuildOpt manifest, graph, profile, plugin
path or output contract. An ambiguous, global, unprofitable or drifted case
must run optimized native Gradle and make no performance claim.

## Components and current role

| Component | What it does | Current conclusion |
| --- | --- | --- |
| **Structural Build Impact** | Selects only task/project producers needed by the exact change and required outputs. | Primary accelerator; large reviewed and automatic wins exist, but automatic breadth is not yet economical. |
| **Automatic discovery** | Derives change ownership, Gradle outputs and a structural candidate without repository-name rules. | Works on 5/5 latest breadth subjects; four complete candidates and one safe aggregate-workflow rejection. |
| **Incremental learning and value gate** | Alternates one control or candidate across each useful invocation and checks outputs, interval, tail, fallback and payback. | The transaction now adds zero measurement-only workflows; a bounded fixture correctly retained native when 0.90% did not prove value, and user cancellation preserves the cancelled build without starting recovery. |
| **Verified output materialization** | Restores exact unaffected outputs that the requested workflow still requires when their producer is omitted from a clean candidate graph. | The bounded POC rebuilt one changed JAR, materialized two unaffected JARs and reproduced the full three-JAR digest; corrupt or stale state is rejected before candidate execution. This is correctness evidence, not a timing claim. |
| **Aggregate workflow partition** | Splits broad lifecycle workflows into changed-owner producer groups plus exact revision-bound outputs that can be materialized. | A generic 66-project `assemble` proposal fell from 66 entrypoints to one; 65 consumer JARs were materialized and all 66 outputs matched. Performance is not yet measured. |
| **Profile portfolio / central state** | Reuses qualified evidence under exact repository, Wrapper, workflow, graph, output and executable bindings. | Functional across checkouts/machines; value depends on profile lifetime and cannot rescue an uneconomic first decision. |
| **Gradle-compatible cache** | Reuses verified outputs locally or through optional HTTP/HTTPS central storage. | Supporting infrastructure. Safe Cache is near native-cache parity, not the principal speed claim. |
| **Task adapters / Patch Autopilot** | Makes one exact task shape safely reusable after review. | Strong scoped evidence exists; not generalized to arbitrary tasks. |
| **Launcher, history and reports** | Preserves process behavior and exposes wall time, uncertainty, payback and fallback. | Necessary infrastructure; its overhead is included in candidate time. |
| **Runtime Tuning, Hot State and standard Copy** | Earlier broad resource/state hypotheses. | Retired after neutral, unstable or regressive end-to-end evidence. |

## Latest unchanged automatic breadth result

One exact BuildOpt binary ran the zero-manual-file path on five frozen public
repositories with eight pairs, exact outputs, full fallback and a maximum
30-build payback.

| Repository / workflow | Graph | Native -> candidate | Direct effect | Learning / payback | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| Spring `testClasses` | 27 -> 10 | 12.340 -> 9.029 s | **26.83% faster**, 7/8 | 339.603 s / 103 builds | Native retained. |
| OpenTelemetry Spring family | 1,024 -> 34 | 76.087 -> 60.681 s | **20.25% faster**, 8/8 | 1,555.444 s / 101 builds | Native retained. |
| Kafka `testClasses` | 64 -> 36 | 14.766 -> 12.785 s | **13.42% faster**, 3/8 | 374.762 s / 190 builds | Native retained. |
| Micronaut `assemble` | no timed candidate | n/a | 73 candidate entrypoints | no calibration | Native retained. |
| Groovy `classes` | 37 -> 30 | 71.480 -> 69.472 s | **2.81% faster**, 7/8 | 1,423.987 s / 710 builds | Native retained. |

The important result is not “0/5 therefore the idea fails.” Automatic
discovery finds genuine reductions, and OpenTelemetry has a strictly positive
interval with 8/8 favorable pairs. The blocker is that the current command
duplicates too much Gradle work to learn synchronously. Kafka/Groovy also show
that a complete workflow output contract is much broader than older reviewed
profiles. Verified output materialization and generic aggregate partitioning
are now implemented and checked separately: the latter reduces a 66-project
aggregate proposal to one rebuild entrypoint while composing the other 65
exact outputs. The unchanged five-repository rerun must now determine whether
those mechanisms improve wall time and learning economics on public builds.

## Latest incremental-learning result

The new transaction ran one baseline plus eight control/candidate pairs over
17 useful `assemble` invocations. It launched **zero workflows solely to
measure**, preserved the same required JAR digest in every observation and
kept full-graph fallback available. The three-project fixture measured
5,549.5-ms control versus 5,499.375-ms candidate: 50.125 ms/**0.90%** saved,
4/8 positive pairs, interval -275.75..+405.125 ms and 384-build projected
payback after 19,247 ms of incremental BuildOpt work. The unchanged gates
correctly retained native Gradle. This proves the learning mechanism and its
honest rejection, not a transferable speedup.

## Latest verified-output materialization result

A clean three-project `assemble` candidate rebuilt the changed `service-a`
JAR and materialized the required unaffected `library-c` and `service-b`
JARs from digest-bound local state. The baseline, optimized candidate and
corruption fallback all produced the same three-file output digest. After one
blob was deliberately corrupted, BuildOpt rejected the state before starting
the candidate and completed the full native graph with exit code zero. The
result proves safe clean-workspace composition; it does not claim a wall-time
improvement.

## Latest aggregate-workflow partition result

A generic 66-project Groovy `assemble` fixture made every project transitively
affected while only `:core` directly owned the change. The old flat proposal
needed 66 entrypoints and exceeded the unchanged safety limit of 64. The new
partition ran only `:core:assemble`, restored 65 exact consumer JARs and
produced the same complete 66-JAR SHA-256 as native Gradle. No consumer
producer task executed. This proves generic structural breadth and exact
composition, not a wall-time win or cross-revision ABI compatibility.

## Prior positive evidence and how to interpret it

The published zero-manual-file terminal POC previously qualified Ktor `jvmJar`
at **79.82% faster** with 26-build payback and Beam `classes` at **61.65%
faster** with 28-build payback, both 8/8 with exact outputs and fallback.
Optional central cache/state composition under equal cache opportunity measured
**82.45% faster on Ktor** and **56.41% on Beam**.

Reviewed profiles also showed larger savings on Spring, OpenTelemetry, Kafka,
Micronaut and Groovy. Those results prove structural potential, not automatic
customer value: reviewed profiles narrowed the required outputs manually. The
latest breadth result is the authoritative answer for the unchanged onboarding
path and shows where generalization is still incomplete.

## Current conclusion

Continue the POC, but focus narrowly. Structural Build Impact remains the only
broad accelerator with compelling evidence. Incremental learning removes the
16-extra-build customer transaction, verified materialization closes the
clean-workspace output gap and aggregate partitioning removes the synthetic
entrypoint-cap blocker. BuildOpt is not yet a generally valuable one-command
optimizer because these three improvements have not been measured together on
the unchanged public breadth set. Native retention when value is unproven is
correct behavior, not a hidden failure.

## Next work

1. **Repeat the same five-subject transfer.** Use the exact same revisions and
   commands with incremental learning, verified output materialization and
   aggregate partitioning enabled. Require lower learning cost,
   exact outputs, native fallback and repository-specific payback; do not
   average percentages or add mechanism effects.
2. **Decide from measured value.** Qualify only repository/workflow families
   that beat optimized native Gradle and repay within 30 matching builds;
   diagnose and retain native for the others.

## Evidence

- [Latest automatic breadth result](../../benchmarks/results/poc-automatic-breadth-transfer-v1/README.md)
- [Incremental ordinary-build learning result](../../benchmarks/results/poc-incremental-learning-v1/README.md)
- [Verified output materialization result](../../benchmarks/results/poc-verified-output-materialization-v1/README.md)
- [Aggregate workflow partition result](../../benchmarks/results/poc-aggregate-workflow-partition-v1/README.md)
- [Machine-readable breadth summary](../../benchmarks/results/poc-automatic-breadth-transfer-v1/summary.json)
- [Automatic breadth contract](../../specs/poc-automatic-breadth-transfer-v1.md)
- [Published terminal Ktor/Beam result](../../benchmarks/results/poc-magic-end-to-end-value-v2/README.md)
- [Central end-to-end result](../../benchmarks/results/poc-central-end-to-end-value-v1/README.md)
- [Detailed performance findings](./build-optimization-performance.md)
- [Generalization audit](./buildopt-generalization-audit.md)
- [One-command roadmap](../plans/one-command-onboarding-roadmap.md)
- [Implementation tracker](../../implementation-tracker.md)
