# BuildOpt POC: Current Evidence and Direction

## Executive Summary

- **BuildOpt is testing whether a generic decision layer can make substantial
  Gradle builds faster than optimized native Gradle.** Its current accelerator
  selects a smaller change-specific project graph before Gradle executes it,
  while preserving repository-declared outputs and the original full graph as
  the fallback.
- **The current evidence supports continuing the POC.** The same structural
  method qualified on OpenTelemetry, Kafka, Micronaut, Groovy, and the unseen
  Hibernate holdout, reducing wall time by **5.88% to 84.11%**. Spring improved
  by **17.94%** but correctly retains native Gradle because its frozen
  repeatability gate did not pass.
- **Cross-repository CI reproducibility passes 5/5, and the blind transfer is
  now complete.** The read-only Action recreated every terminal proposal on
  clean hosted runners with zero graph drift. The unchanged generic path then
  discovered a 29-to-1 Hibernate candidate without repository-specific logic.
  An attribution run exposed a material first/second-period effect rather than
  a product failure. A fresh preregistered crossover then stabilized the exact
  300/32-task shapes and qualified at **5.88%**, with eight of eight reciprocal
  blocks positive.
- **Onboarding now has one owner contract.** A reviewed
  `.buildopt/profile.json` carries the Gradle workflow, exact Git change source,
  confirmed outputs and fallback policy for both local and CI proposals. Every
  target revalidates those outputs and falls back to native Gradle with drift
  diagnostics before any timing.
- **Workflow breadth exposed the next generic constraint.** On substantial
  packaging, verification, distribution, and test-preparation workflows,
  Spring showed an exact-output **18.47% / 2.695 s** mean saving but missed the
  frozen repeatability gate at 7/8 positive pairs. Groovy and Kafka failed
  closed because owner artifacts contain wall-clock time, absolute workspace
  paths, or ZIP timestamps. The result is **0/4 qualified** and all four retain
  native Gradle; no output difference was normalized away.

## The Project in One Minute

Gradle already provides Build Cache, Configuration Cache, incremental tasks,
up-to-date checks, parallel execution, and remote-cache integration. Those
features optimize work inside the task graph requested by the build.

BuildOpt explores an additional layer: given the original Gradle workflow, an
exact Git change, and the outputs the repository requires, can it prove that a
smaller graph is sufficient and materially faster? If the answer is uncertain
or the measured value is weak, BuildOpt runs the optimized native full graph.

The current POC flow is:

```text
repository-owned Gradle task + required outputs
  -> buildopt profile outputs
  -> explicit owner confirmation in .buildopt/profile.json
  -> buildopt profile propose
  -> buildopt profile measure
  -> buildopt profile evaluate
  -> explicit review
  -> buildopt poc or optimized native Gradle
```

There are no repository-name rules and no automatic production activation.

## Mechanisms and Current Role

| Mechanism | What it does | Difference from native Gradle | Current POC decision |
| --- | --- | --- | --- |
| **Structural Build Impact** | Maps a change and required outputs to the smallest proven project/task entrypoint set. | Avoids configuring and executing unrelated parts of the requested graph; Gradle's incremental features normally act after the graph has been requested. | **Active accelerator.** This is the mechanism measured consistently across the six current public repositories. |
| **Profile measurement and evaluation** | Captures isolated paired timings, exact outputs, drift bindings, and fallback evidence before producing a profile. | Adds a cross-build evidence and activation policy rather than another Gradle execution optimization. | **Required safety layer.** Review remains explicit. |
| **Safe Cache / local L1** | Reuses verified outputs within repository, Wrapper, and platform boundaries. | Adds isolation and verification around native cache semantics. | **Not a speed differentiator.** It is at parity with a warm native Gradle cache and is not part of the current structural claim. |
| **Exact task optimization / Patch Autopilot** | Makes one exactly understood task shape reusable through a bounded adapter or reviewable patch. | Repairs or augments cacheability that the repository has not declared correctly. | **Promising but scoped research.** It must qualify independently for each generic task contract before joining the main path. |
| **Shared / Edge Cache** | Serves committed outputs from shared or nearer storage. | Adds controlled locality around Gradle's remote-cache protocol. | **Bounded supporting evidence.** Network-dependent results are kept separate from the structural matrix. |
| **Build History and launcher** | Records evidence, preserves process behavior, and applies bypass/fallback. | Provides orchestration and attribution, not avoided Gradle work. | **Supporting infrastructure.** Its overhead is included in installed-path measurements. |
| **Runtime Tuning, Hot State, standard Copy** | Previously tested worker/heap changes, plan reuse, and broader task adaptation. | Attempted to tune or reuse work beyond the retained structural path. | **Retired.** Terminal evidence was neutral, unstable, or regressive. |

## Current Wall-Time Evidence

The comparison baseline is optimized native Gradle using the same repository
revision, Gradle workflow, runner resources, required outputs, and applicable
native cache/parallel settings. Each row contains eight isolated alternating
pairs. BuildOpt time includes proposal consumption, validation, launcher, and
Gradle execution.

| Repository | Full -> selected projects | Native mean | BuildOpt mean | Mean saving | Positive pairs | Decision |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| Spring Framework | 27 -> 10 | 13.940 s | 11.438 s | **2.501 s / 17.94%** | 7/8 | Retain native: one -260 ms pair fails the frozen 8/8 repeatability gate. |
| OpenTelemetry Java Instrumentation | 1,024 -> 34 | 83.934 s | 71.825 s | **12.110 s / 14.43%** | 8/8 | Qualify. |
| Apache Kafka | 64 -> 3 | 82.498 s | 13.113 s | **69.385 s / 84.11%** | 8/8 | Qualify. |
| Micronaut Core | 75 -> 22 | 27.407 s | 15.968 s | **11.439 s / 41.74%** | 8/8 | Qualify. |
| Apache Groovy | 37 -> 2 | 75.064 s | 19.629 s | **55.434 s / 73.85%** | 8/8 | Qualify. |
| Hibernate ORM holdout | 29 -> 1 | 216.724 s | 203.991 s | **12.733 s / 5.88%** | 8/8 reciprocal blocks | Review structural profile; outputs and both fallbacks pass. |

Every accepted observation preserved the declared required outputs byte for
byte, completed a scheduling-equivalent native full-graph fallback, and
recorded zero product-attributable failures. OpenTelemetry uses its separately
preregistered v4 correction because the earlier fallback changed scheduling;
none of the rejected timing was reused.

Repository percentages are independent. They are not averaged across
repositories and are not added to cache, task-adapter, or Edge results.

### Substantial non-compilation workflows

| Workflow | Full -> selected projects | Observed result | Decision |
| --- | ---: | --- | --- |
| Apache Groovy root `jar` | 37 -> 2 | Exact-byte validation stopped: one generated release-info payload embeds `BuildTime`. | Retain native. |
| Apache Kafka root `checkstyleMain` | 64 -> 2 | Exact-byte validation stopped: reports embed absolute isolated-workspace paths. | Retain native. |
| Apache Kafka root `shadowJar` | 64 -> 2 | Exact-byte validation stopped: upstream preserves ZIP timestamps/order; 4,378 payloads were identical across two native rebuilds. | Retain native. |
| Spring Framework root `testClasses` | 27 -> 10 | 14.588 s -> 11.893 s; **2.695 s / 18.47%**; positive interval; 7/8 pairs. | Retain native under the frozen 8/8 gate. |

These rows answer a different question from the compilation-oriented matrix:
can the same generic owner-input path add value to other Gradle workflow
families? The answer is “promising but not qualified yet.” The graph reduction
transfers, but exact output semantics and repeatability must be solved without
repository-name rules or post-result threshold changes.

### Hibernate qualifies after the recoverable variance is corrected

The immutable v3 diagnostic did not replace the 7.80% v2 value result. It added
a second excluded base-revision warm-up and reran all eight pairs from zero.
The first pair changed from **−1.118 s to +11.883 s**, proving that the original
negative was recoverable. The complete v3 run reached only **4/8 positive
pairs**, **2.50% mean savings**, and an interval of **−6.604..+20.190 s** while
both arms continued accelerating. Native Gradle therefore remains active.

The subsequent v4 replay completed all eight fresh pairs after an exact-target
warm-up. Native averaged **221.898 s** and BuildOpt **213.418 s**, a positive
**8.480 s/3.82%** signal, but only **5/8** pairs improved. All four
control-first pairs improved, compared with one candidate-first pair. This
diagnosed a period effect but did not qualify the candidate.

Version 5 froze a reciprocal AB/BA protocol before collecting new data. Two
independent batches produced eight order-adjusted blocks; exact target shapes
were stable at 300 native tasks and 32 BuildOpt tasks in all 20 observations
per arm. The aggregate averaged **216.724 s** for native Gradle and **203.991
s** for BuildOpt, saving **12.733 s/5.88%** with a bootstrap interval of
**+6.808..+19.859 s** and **8/8 positive blocks**. Required JARs were
byte-identical and both full-graph fallbacks passed. The result authorizes
review of a structural profile, not automatic activation.

The run also improved the generic collector. Duplicate identical Gradle task
console records initially invalidated task-shape evidence; BuildOpt now
normalizes identical repeats while rejecting conflicting outcomes. The fix is
covered generically and no Hibernate task rule was added.

## Clean-CI Reproducibility

Hosted run
[`31467370391`](https://github.com/tonyredondo/buildopt/actions/runs/31467370391)
executed the review-only Action from immutable revision `18caa8f` on five
independent Ubuntu runners. Every repository reproduced its exact owner input,
source change, proposal, project counts, manifest, graph, generated binding,
fallback input, and checksums. The result was **5/5 `MATCH`, 0 drift**, and zero
active profiles.

Spring's successful replay does not reverse its value decision: it confirms
the 27-to-10 proposal is reproducible, while the prior 7/8 wall-time result
still requires native Gradle. The other four value decisions remain bound to
their existing paired timings. No timing was executed during this replay.

## What the Tests Demonstrate

- **The idea transfers.** Five different public Gradle families qualified
  without repository-specific product logic, including the unseen Hibernate
  family selected only after the method was frozen.
- **Avoided work compounds.** Omitting projects also removes their
  configuration, scheduling, cache lookup, compilation, and packaging work;
  this explains the larger Kafka, Micronaut, and Groovy gains.
- **Correctness is necessary but not sufficient.** Spring preserved outputs
  and improved on average, yet retained native Gradle because the complete
  result missed the repeatability gate.
- **The gain survives product overhead.** Timings include BuildOpt validation,
  launcher, profile loading, and Gradle execution rather than an internal
  microbenchmark.
- **Owner output mistakes now fail before measurement.** The generic preflight
  reran the original Hibernate `build/libs` declaration, rejected its empty
  result after one owner-workflow execution and exposed three
  `:hibernate-core` JAR candidates under `target/libs`. It started zero
  warm-ups/timings and wrote no candidate profile.
- **Local and CI semantics are now identical.** The same checked owner file
  derives the exact base-to-HEAD Git change, is SHA-bound into the proposal and
  reexecutes the confirmed output workflow. A synthetic `target`-to-`dist`
  drift returned `NATIVE_FULL_GRAPH / REQUIRED_OUTPUTS_EMPTY`, exposed the new
  Gradle-owned JAR candidate and wrote no candidate graph.
- **The owner-input path is not compilation-specific.** One unchanged schema
  produced exact changed-project candidates for `Jar`, typed verification,
  distribution ZIP and `testClasses` workflows. All four rebuilt their
  declared outputs byte for byte without executing a Gradle `Test`; an
  arbitrary executable task retained native with
  `ORIGINAL_WORKFLOW_UNSUPPORTED` before timing.
- **A raw pair can measure period as well as product.** Hibernate v4 preserved
  exact outputs while the first/second execution position dominated early
  results. Substantial measurements need target stability and reciprocal blocks
  before a strict per-observation gate is meaningful.
- **Byte identity can reject semantically equal owner outputs.** That is the
  correct default: BuildOpt retained native for time-bearing JAR metadata,
  workspace-bearing reports, and timestamped fat JARs. Any future equivalence
  must be explicit in the owner contract and independently validated; BuildOpt
  must never normalize a mismatch silently.

## Current Conclusion

BuildOpt now has a defensible POC value proposition: it can improve substantial
Gradle build wall time beyond native caching and incremental execution by
proving that less of the repository needs to run for a specific change and
declared output set.

The evidence does **not** support calling BuildOpt a universal optimizer yet.
The right product shape today is a review-required structural optimization
assistant that produces evidence, qualifies only repeatable wins, and otherwise
keeps native Gradle authoritative.

## Recommended Next Steps

1. **Generalize owner-approved output equivalence.** Keep byte identity as the
   default, but prototype explicit contracts for relocatable reports and
   canonical archive contents. Re-run the three blocked public workflows and
   prove that the contract compares semantics without hiding payload drift.
2. **Replicate Spring with an order-aware frozen protocol.** Preserve the
   current 7/8 result, investigate its negative pair, and collect fresh data;
   do not relax the existing gate.
3. **Measure only reviewed candidates.** A CI proposal remains an observation,
   not value evidence. Run isolated paired measurement only after its graph,
   outputs and fallback are accepted.
4. **Replay reviewed profiles through the installed path.** Confirm that the
   public package preserves the measured result for additional qualified
   repositories instead of treating harness evidence as deployment evidence.
5. **Keep wall time authoritative.** Users
   should provide a Gradle command, change source, and output contract—not
   hand-authored graphs. Promote only installed paths that preserve outputs,
   pass repeatability, prove fallback, and materially beat native Gradle.

## Scope and Evidence Sources

This is POC evidence, not production readiness, autonomous activation,
long-duration validation, customer operations, or a universal savings claim.
Test Optimization remains outside Build Optimization.

- [Terminal five-repository structural matrix](../../benchmarks/results/poc-generic-profile-matrix-v3/README.md)
- [OpenTelemetry fallback-equivalence correction](../../benchmarks/results/poc-generic-profile-matrix-v4/README.md)
- [Hosted review-only CI artifact run](https://github.com/tonyredondo/buildopt/actions/runs/31464264563)
- [Five-repository clean-CI replay](../../benchmarks/results/poc-generic-profile-ci-replay-v1/README.md)
- [Unseen Hibernate ORM holdout](../../benchmarks/results/poc-generic-holdout-v2/README.md)
- [Hibernate warm-up diagnosis](../../benchmarks/results/poc-generic-holdout-v3/README.md)
- [Hibernate target-workload attribution](../../benchmarks/results/poc-generic-holdout-v4/README.md)
- [Hibernate reciprocal crossover result](../../benchmarks/results/poc-generic-holdout-v5/README.md)
- [Hibernate output-contract preflight](../../benchmarks/results/poc-generic-output-contract-v1/README.md)
- [Generic owner-input contract](../../specs/poc-generic-owner-input-v1.md)
- [Hosted generic workflow-breadth result](../../benchmarks/results/poc-generic-workflow-breadth-v1/README.md)
- [Public workflow-family value and root causes](../../benchmarks/results/poc-generic-workflow-value-v1/README.md)
- [Generalization audit](./buildopt-generalization-audit.md)
- [Detailed performance findings and historical research](./build-optimization-performance.md)
- [Implementation tracker](../../implementation-tracker.md)
