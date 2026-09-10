# Build Optimization

Research review · 10 September 2026

Build Optimization aims to reduce the time it takes to build software without changing what the build produces or skipping required checks. We want to identify unnecessary work and reuse valid results from earlier builds, so the same work does not have to be repeated after every code change.

The longer-term goal is for the system to maintain those improvements as a project evolves. It should detect when an optimization stops helping and look for a replacement, while accounting for the time that search takes. This research needs to establish whether the resulting savings can hold up across different open-source repositories and their development histories.

These experiments use Gradle, the tool that runs the projects' builds. They cover compilation and checks needed before tests. Selecting fewer tests is outside this study. The [appendix](https://github.com/tonyredondo/buildopt/blob/main/docs/findings/buildopt-next-quarter-evidence-2026-09-10.md) contains the evidence for the six proofs of concept (PoCs), SDTEST-3934–3939, and their follow-up experiments.

## Repositories used

We ran performance comparisons, build checks or diagnostics on 40 upstream projects, including attempts that failed. On eight more, we inspected source code or build workflows. The table separates those stages:

| Work performed | Repositories |
| --- | --- |
| Performance comparisons, including negative and inconclusive results | Apache Beam, Apache Groovy, Apache Kafka, Elasticsearch, Hibernate ORM, Ktor, Micronaut Core, Mockito, OpenTelemetry Java Instrumentation, SpotBugs, Spotless and Spring Framework. |
| Additional build checks or diagnostic attempts | CycloneDX Gradle Plugin, Dependency Analysis Gradle Plugin, GraalVM Native Build Tools, Gradle IntelliJ Plugin, Gradle Maven Publish Plugin, Gradle Node Plugin, Gradle Profiler, Gradle Test Retry Plugin, Gradle Versions Plugin, GraphQL Java, JMH Gradle Plugin, JUnit Pioneer, Jib, Kotlin Symbol Processing (KSP), LSSS, Licensee, MockK, OkHttp, OpenAPI Generator, Palantir Gradle Baseline, Palantir Gradle Consistent Versions, Paparazzi, Protobuf Gradle Plugin, QuickCarpet, Shadow, Suwayomi Server, Testcontainers Java and detekt. |
| Source or workflow inspection, without an accepted speed comparison | BlueMap, Flyway, Gradle, JUnit 5 / JUnit Framework, Kotlin, Liquibase, OpenRewrite and Quarkus. |

The experiments also used small Kotlin and Groovy test repositories, an incremental-learning test project, and Micronaut and Spring forks for patch review. Bnd was a plugin dependency in the GraphQL Java investigation. The [inventory](https://github.com/tonyredondo/buildopt/blob/main/docs/findings/buildopt-next-quarter-evidence-2026-09-10.md#repository-inventory) lists these separately to avoid counting forks, test projects or dependencies as independent validations.

The projects include large applications, libraries and build plugins. Where a diagnostic failed, we still have no performance answer. Further validation needs the same optimization to work across different project structures and their Git histories.

## Results of the six PoCs

A task is one build step, such as compiling files or checking coding rules.

Closed means no further research on that approach. The table also identifies code we can still use to run or check other experiments.

| PoC | Result | Status and intended use |
| --- | --- | --- |
| Safe caching, SDTEST-3934 | Reusing saved results worked and helped when compared with caching disabled. Against Gradle's effective cache, we found no clear additional saving. | Closed as an extra accelerator. Cache connections and safety checks support experiments with Gradle's caching enabled. |
| Runtime tuning, SDTEST-3935 | CPU and memory changes produced inconsistent or slower results. Reducing Spring from twelve workers to six made the build 2.00% slower. | Closed; do not apply the tested settings. Later CPU limits test how a correction behaves with fewer resources. |
| Build Impact, SDTEST-3936 | Building fewer parts of a project saved substantial time on selected changes. Later attempts to find and reuse those plans failed to deliver recurring gains. | The general approach to selecting and reusing plans is closed. Keep the build analysis code and the evidence for the selected results. |
| Patch Autopilot, SDTEST-3937 | Some patches sped up individual tasks. We could prepare, check and undo them, but broad searches found too few useful corrections. | Keep patch delivery for research. Continue investigating repeated work inside tasks, including Checkstyle, whose sustained saving is still unresolved. |
| Build History, SDTEST-3938 | Build records can be stored, looked up and viewed in a local dashboard. No separate reduction in build time was measured. | Used to inspect and compare experiments. No separate acceleration research is proposed. |
| Edge Cache, SDTEST-3939 | Keeping cached files near the machine helped under controlled slow-network conditions. None of three projects passed the later study's full savings criteria. | Closed as the default accelerator. The evidence supports the remote-access conditions tested, with no further work proposed here. |

Gradle already reuses completed work through its build cache. For compatible projects, Configuration Cache also reuses build preparation. Comparisons must keep these features enabled where they work. [Gradle build cache](https://docs.gradle.org/current/userguide/build_cache.html), [Configuration Cache](https://docs.gradle.org/current/userguide/configuration_cache.html).

## Measured savings

Two Elasticsearch tasks need a brief explanation. **ForbiddenPatterns** scans files for text patterns the project disallows. **Checkstyle** checks Java source against coding and formatting rules and reports violations. These tasks examine source code; they do not test the application's behavior.

A cacheability patch lets Gradle reuse a task's entire result. An incremental patch helps when the task must run again: it checks the changed files and reuses valid results for the rest.

| PoC or follow-up | Repository | What the experiment ran | Measured saving |
| --- | --- | --- | --- |
| Build Impact | Ktor | The jvmJar command, which builds libraries for the Java platform. BuildOpt selected only the projects needed by the chosen code change. | 79.82%, about 31.0 seconds; eight faster comparisons. |
| Build Impact | Apache Beam | The classes command, which compiles project code. BuildOpt reduced the work to the projects needed for the selected change. | 61.65%, about 40.1 seconds; eight faster comparisons. |
| Patch Autopilot: build correction | Micronaut Core | A task compiling Python files bundled with the project. The patch made its results safely reusable through Gradle's cache. | 63.44%, about 6.92 seconds; eight faster comparisons. |
| Patch Autopilot: build correction | Spring Framework | Spring Core's architecture check, which checks rules about code structure and dependencies. The patch allowed reuse of the task's previous result. | 35.34%, about 0.99 seconds; eight faster comparisons. |
| Patch Autopilot: build correction | Elasticsearch | The ForbiddenPatterns task, with its completion marker removed before each invocation. The candidate could restore a cached result. | 15.54%, about 7.17 seconds; eight faster comparisons. |
| Build correction (Patch Autopilot follow-up) | Elasticsearch | The server's checks before accepting a code change (precommit), using Checkstyle results for unchanged files on two selected changes. The experimental patch was applied directly. | 41.29%, 40.11% and 24.12% at eight, four and two CPUs; two measured comparisons per setting. |
| Edge Cache | Apache Kafka | Compilation of Kafka's client code and tests, restoring the same required files from a cache. Only the location used to read cached data changed. | 15.21%, about 1.35 seconds; four faster comparisons under controlled network conditions. |

Each result preserved the required build files or reports. The commands and conditions differ, so these percentages cannot be combined into one product-wide figure. [Ktor and Beam](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/poc-magic-end-to-end-value-v2/README.md), [Micronaut and Spring](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/reviewed-native-patch-portfolio-v1/README.md), [ForbiddenPatterns](https://github.com/tonyredondo/buildopt/blob/main/docs/plans/buildopt-product-viability-v1-evidence.md#e06-native-patches-can-help-within-a-selected-scope), [Checkstyle](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-screen-completion/README.md), [Kafka Edge Cache](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/poc-remote-cache-transfer-v1.json).

## Results by repository

### Ktor

Ktor saved 79.82% on the selected build. Offsetting the recorded preparation time would take an estimated 26 builds where the optimization applies, but we did not run that sequence. We therefore do not know how often the same setup would help during Ktor development. The later negative studies of plan reuse used other projects.

### Apache Beam

Beam saved 61.65%, with an estimated 28 applicable builds needed to offset preparation. The open questions are how many later changes would qualify and how long the plan would remain useful before it needed replacing. Neither was measured for this setup.

### Micronaut Core

The Python compilation task saved 6.92 seconds and produced the same files. Reusing its output from a different directory and undoing the patch also worked. This correction is worth retaining. Its contribution to the complete build still needs measuring through a sequence of normal changes.

### Spring Framework

The architecture check saved about one second. Its usefulness to the complete build depends on how often that saving occurs, which we have not measured through Spring's history. A later Hibernate patch restored the right cached files but ran 1.26% slower, another reason to measure each correction in the build where it would be used. [Hibernate result](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/reviewed-native-patch-customer-economics-v1/README.md).

### Elasticsearch: ForbiddenPatterns

The caching patch saved 15.54% when the experiment deliberately made the task run again. We kept that patch in the reference build for the later history study.

We then considered a further change to check only modified files. In the first 20 historical builds, Gradle skipped ForbiddenPatterns on 17. Even removing the task entirely would fall below the required one-second and 5% average saving, so we dropped this additional correction. The earlier caching patch still has its measured benefit under the original conditions. [History calculation](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv002/opportunity.md).

### Elasticsearch: Checkstyle

Rechecking fewer files made the selected builds faster at all three CPU settings. Across a different sequence of 13 changes, the implementation was 19.16% slower overall. Checkstyle ran on only one of those changes. The overall result includes the other twelve builds, where it had no chance to save time.

The slowdown was not explained by losing the previous Checkstyle checks: the implementation was still reusing them. We changed how it saved that information for the next build. The revised code passed the correctness checks, but one comparison was faster and the other slower, failing the performance rule agreed beforehand. Both versions already included the optimization. The shared machine was busy at times, leaving uncertainty about the timings; both results remain in the record. [Latest comparison](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-finalization/README.md).

We need to resolve that timing variation and compare the revised code with ordinary Gradle through real changes before claiming a sustained Checkstyle saving.

### Apache Kafka: Edge Cache

Reading cached data nearby saved 15.21% under the tested remote-access conditions. The broader study did not show a consistent advantage: Groovy's saving was too small, OpenTelemetry was slower, and Spring's result was uncertain. The broader Edge Cache route remains closed. [Broader results](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/remote-cache-locality-value-v3/paired-value.json).

The [appendix](https://github.com/tonyredondo/buildopt/blob/main/docs/findings/buildopt-next-quarter-evidence-2026-09-10.md#additional-selected-savings) examines further selected gains in Kafka, Groovy, Spring Messaging, OpenTelemetry and Mockito, including which reached a later history test.

## Replaying development history

The following studies led us to close the work on selecting and reusing build plans. They did not test how long the Micronaut or Spring patches above would keep helping.

| Study | Result | Conclusion |
| --- | --- | --- |
| Saved plans across five repositories | Only one repository saved time after recorded overhead. A plan was used on one of six later builds where it might have applied. | The approach did not find enough recurring opportunities. |
| Smaller, adaptive plans across five repositories | No optimization was applied in 100 comparisons. Total elapsed time was worse, with recorded BuildOpt work adding delay. | Smaller plans still did not get used. |
| Selected Kafka, Groovy and Spring follow-up | Kafka first passed its selected comparisons, then used the plan on none of the next three changes. Groovy lost time after the added work was counted. Spring's part was not run because the earlier results meant the study could no longer pass. | The completed work failed to save time. Spring's part remains unmeasured. |

Sources: [saved-plan decision](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/poc-functional-coverage-decision-v1/README.md), [100 comparisons](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/current-longitudinal-attribution-v1.json), [three-project follow-up](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/three-class-chronological-value-v1/README.md).

The current Elasticsearch study follows 100 consecutive Git changes over about 46 hours, keeping build state between changes. The first 20 are for developing the correction and checking measurements; the next 80 are reserved for validation and have not run. This short window lets us test ordinary changes before attempting a longer history. Tool upgrades and other less frequent changes would need a later extension.

Before continuing, the proposed four-build check would run identical code on both sides to see how much the timings vary without an implementation change. That check is still pending, along with the comparison against ordinary Gradle and the reserved history. Experiments remain paused for this review.

## Research still needed

A correction must save time across the complete history, including builds where it cannot help. All required files and diagnostics must be preserved, and the comparison must count BuildOpt's added work, preparation and any checks needed after changes.

The same approach then needs to pass on at least two of three independent repositories. We should choose projects with different build structures and test their recorded commands through Git history. That would show whether the improvement transfers beyond the project used to develop it.

Adaptation also has to earn its place. We need to compare ordinary Gradle, one fixed correction and a version that replaces corrections when they stop helping. Searching for a replacement takes time, so the adaptive version must save enough to cover that work and still beat the fixed correction. It must also stop unsafe reuse immediately. A single slow build on a busy machine should not trigger repeated searches.

## Recommendation

We should continue investigating corrections to individual build tasks. The Micronaut and Spring patches saved time without changing the required results. Checkstyle also saved time on selected builds, although its benefit across a sequence of changes remains unresolved.

The next test is whether these corrections keep saving time through real code changes in several repositories. Builds where the correction does nothing must count, along with the time BuildOpt adds to every build.

The attempts to add another cache, tune runtime settings and reuse saved build plans should stay closed. Their results do not justify another broad search through the same options.

## Proposed research OKRs

For next quarter, the objective is to demonstrate repeatable build-time savings across different open-source repositories and their histories. If the remaining approaches cannot do that, we should close the research.

| Step | Result to obtain | What the result decides |
| --- | --- | --- |
| Resolve the Checkstyle timing question | Complete the identical-code control, then a valid comparison of the revised correction against ordinary Gradle. Preserve every result and the correctness checks. | Decide whether the timings are reliable enough to take Checkstyle into history validation. |
| Test sustained saving on the first repository | Run the complete 100-change sequence twice from separate state. Keep 20 changes for development and 80 for validation. Each run must save at least 5% overall and an average of at least one second per scheduled validation build after required extra work, with a positive statistical result that accounts for related changes. Preparation time must be recovered and stay recovered within the sequence. | Decide whether the correction offers a sustained advantage. Keep all scheduled outcomes, output checks and the existing limit on slow-build regressions. |
| Test the same approach on different repositories | Evaluate two additional repositories chosen before their performance is known. At least two of the three total must pass the same criteria. Record how often the correction applies, when it stops helping and what must be rerun after changes. | Distinguish a repeatable approach from a correction useful in only one place. |
| Test adaptive behavior, after the fixed correction passes | Compare ordinary Gradle, the fixed correction and a version that detects loss of benefit and validates replacements. Use history not already consumed while developing that policy. | Establish whether adaptation adds net saving while preserving results. If it does not, no self-managing PoC has been demonstrated. |

The [existing technical criteria for history, correctness and adaptation](https://github.com/tonyredondo/buildopt/blob/main/docs/plans/buildopt-product-viability-v1.md#6-evidence-ladder-and-decisions) remain the basis for judging these results.

Drop a candidate that fails these checks. Another candidate is worth investigating only if recorded builds show enough repeated work to make a meaningful difference to total build time. If none can sustain that saving across histories and repositories, close the initiative.
