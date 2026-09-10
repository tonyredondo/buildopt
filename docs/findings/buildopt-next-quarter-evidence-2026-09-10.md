# Build Optimization: repositories, experiments and remaining questions

10 September 2026 · Evidence for the [research review](https://github.com/tonyredondo/buildopt/blob/main/docs/findings/buildopt-next-quarter-investment-review-2026-09-10.md).

Several experiments made selected builds faster while preserving their required results. What remains unresolved is whether the build corrections can keep helping through later code changes. This appendix records the repositories tested, the results and the limits of each finding.

Gradle is the tool that runs these builds. A task is one step, such as compiling files or checking coding rules. A paired comparison runs the reference version and the proposed change under matched conditions. “Saving” means less elapsed time for the specified command. A projected recovery of setup time estimates how many applicable builds would offset the preparation; it does not mean we observed those builds. Results from different commands cannot be added together.

The [research register](https://github.com/tonyredondo/buildopt/blob/main/docs/research-status.md#complete-plan-inventory) and [execution tracker](https://github.com/tonyredondo/buildopt/blob/main/docs/plans/buildopt-product-viability-v1-tracker.md) retain the individual plans, unsuccessful attempts and stopping decisions.

## Repository inventory

The records identify **48 upstream projects**: twelve reached performance comparisons, twenty-eight additional projects had build checks or diagnostic attempts, and eight were inspected through source or workflow review. Failed starts are included, with their reason. A repository appearing here is not a claim of a successful optimization.

Repository names below follow the identities recorded in the experiments. Each link points to retained evidence of the stated work. Different experiments on the same repository count as one project.

### Performance comparisons: twelve projects

| Project and repository | Work measured | Evidence |
| --- | --- | --- |
| Apache Beam — apache/beam | Build Impact on selected compilation work. | [Performance data](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/poc-magic-end-to-end-value-v2/summary.json) |
| Apache Groovy — apache/groovy | Build Impact, later changes and Edge Cache. | [Performance data](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/recurrent-root-paired-value-v1/summary.json) |
| Apache Kafka — apache/kafka | Build Impact, later changes and Edge Cache. | [Performance data](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/history-admitted-paired-value-v1/summary.json) |
| Elasticsearch — elastic/elasticsearch | Build corrections, Checkstyle comparisons and development-history measurements. | [Performance data](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-screen-completion/summary.json) |
| Hibernate ORM — hibernate/hibernate-orm | A task correction that reused cached results but made the measured build slower. | [Performance data](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/reviewed-native-patch-customer-economics-v1/README.md) |
| Ktor — ktorio/ktor | Build Impact on the selected library build. | [Performance data](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/poc-magic-end-to-end-value-v2/summary.json) |
| Micronaut Core — micronaut-projects/micronaut-core | Python build-task correction and plan-reuse studies. | [Performance data](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/reviewed-native-patch-portfolio-v1/result.json) |
| Mockito — mockito/mockito | A selected Build Impact gain and a separate cache experiment below its threshold. | [Performance data](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/poc-real-world-performance-v1.json) |
| OpenTelemetry Java Instrumentation — open-telemetry/opentelemetry-java-instrumentation | Build Impact with a packaging correction, plan reuse and cache-location comparisons. | [Performance data](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/poc-otel-optimization-v2.json) |
| SpotBugs — spotbugs/spotbugs | Selected Build Impact comparisons; no qualifying saving. | [Performance data](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/poc-real-world-performance-v1.json) |
| Spotless — diffplug/spotless | Selected Build Impact and broader build comparisons; no qualifying saving. | [Performance data](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/poc-real-world-performance-v1.json) |
| Spring Framework — spring-projects/spring-framework | Build Impact, build corrections, runtime settings and cache-location comparisons. | [Performance data](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/reviewed-native-patch-portfolio-v1/result.json) |

### Additional build checks and diagnostic attempts: twenty-eight projects

These runs helped decide whether a correction could be attempted. They did not establish a paired speed improvement for the listed projects.

| Project and repository | What happened | Evidence |
| --- | --- | --- |
| CycloneDX Gradle Plugin — CycloneDX/cyclonedx-gradle-plugin | The diagnostic could not complete without Java 8. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/critical-path-first-reviewed-native-patch-v1/diagnostic-summary.json) |
| Dependency Analysis Gradle Plugin — autonomousapps/dependency-analysis-gradle-plugin | The initial build check passed; no candidate speed comparison followed. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/configuration-input-native-corrections-v1/cinc-e002-cohort-freeze.json) |
| detekt — detekt/detekt | The initial build check passed; no candidate speed comparison followed. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/configuration-input-native-corrections-v1/cinc-e002-cohort-freeze.json) |
| GraalVM Native Build Tools — graalvm/native-build-tools | The initial build check passed; no candidate speed comparison followed. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/configuration-input-native-corrections-v1/cinc-e002-cohort-freeze.json) |
| Gradle IntelliJ Plugin — JetBrains/gradle-intellij-plugin | Build observations were captured successfully; this was a diagnostic result. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/wrapper-coordinated-native-corrections-v1/wcncp-e008-capture.json) |
| Gradle Maven Publish Plugin — vanniktech/gradle-maven-publish-plugin | The diagnostic exceeded its time limit. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/critical-path-first-reviewed-native-patch-v1/diagnostic-summary.json) |
| Gradle Node Plugin — node-gradle/gradle-node-plugin | The initial build check passed; no candidate speed comparison followed. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/configuration-input-native-corrections-v1/cinc-e002-cohort-freeze.json) |
| Gradle Profiler — gradle/gradle-profiler | The initial build check passed; no candidate speed comparison followed. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/configuration-input-native-corrections-v1/cinc-e002-cohort-freeze.json) |
| Gradle Test Retry Plugin — gradle/test-retry-gradle-plugin | Build observations were captured successfully; this was a diagnostic result. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/wrapper-coordinated-native-corrections-v1/wcncp-e008-capture.json) |
| Gradle Versions Plugin — ben-manes/gradle-versions-plugin | Build observations were captured successfully; this was a diagnostic result. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/wrapper-coordinated-native-corrections-v1/wcncp-e008-capture.json) |
| GraphQL Java — graphql-java/graphql-java | Build observations were captured. A later check found too little avoidable preparation and inconsistent output files. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/wrapper-coordinated-native-corrections-v1/wcncp-e008-capture.json) |
| Jib — GoogleContainerTools/jib | The project launcher and Java version were incompatible. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/critical-path-first-reviewed-native-patch-v1/diagnostic-summary.json) |
| JMH Gradle Plugin — melix/jmh-gradle-plugin | The diagnostic lacked the Java versions required by the tests. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/critical-path-first-reviewed-native-patch-v1/diagnostic-summary.json) |
| JUnit Pioneer — junit-pioneer/junit-pioneer | Build observations were captured successfully; this was a diagnostic result. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/wrapper-coordinated-native-corrections-v1/wcncp-e008-capture.json) |
| Kotlin Symbol Processing (KSP) — google/ksp | The initial build check passed; no candidate speed comparison followed. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/configuration-input-native-corrections-v1/cinc-e002-cohort-freeze.json) |
| Licensee — cashapp/licensee | The initial build check passed; no candidate speed comparison followed. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/configuration-input-native-corrections-v1/cinc-e002-cohort-freeze.json) |
| LSSS — marec-open-source/lsss | The diagnostic tied a build observation to its source, but the study found too few eligible projects. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/source-bound-configuration-input-corrections-v1/sbic-e002-strict-diagnostics/result.json) |
| MockK — mockk/mockk | Build observations were captured successfully; this was a diagnostic result. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/wrapper-coordinated-native-corrections-v1/wcncp-e008-capture.json) |
| OkHttp — square/okhttp | The initial build check ran, but tests failed. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/configuration-input-native-corrections-v1/cinc-e002-cohort-freeze.json) |
| OpenAPI Generator — OpenAPITools/openapi-generator | The initial build check passed; no candidate speed comparison followed. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/configuration-input-native-corrections-v1/cinc-e002-cohort-freeze.json) |
| Palantir Gradle Baseline — palantir/gradle-baseline | Project tests failed before the diagnostic could complete. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/critical-path-first-reviewed-native-patch-v1/diagnostic-summary.json) |
| Palantir Gradle Consistent Versions — palantir/gradle-consistent-versions | Build observations were captured successfully; this was a diagnostic result. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/wrapper-coordinated-native-corrections-v1/wcncp-e008-capture.json) |
| Paparazzi — cashapp/paparazzi | The initial build check ran, but exceeded its time limit. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/configuration-input-native-corrections-v1/cinc-e002-cohort-freeze.json) |
| Protobuf Gradle Plugin — google/protobuf-gradle-plugin | Build observations were captured successfully; this was a diagnostic result. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/wrapper-coordinated-native-corrections-v1/wcncp-e008-capture.json) |
| QuickCarpet — QuickCarpet/QuickCarpet | The diagnostic ran; its Fabric report was outside the supported format. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/source-bound-configuration-input-corrections-v1/sbic-e002-strict-diagnostics/result.json) |
| Shadow — GradleUp/shadow | The documentTest workflow ran. The possible correction affected only an 80-ms task. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/prospective-reviewed-native-patch-controlled-trial-v1/README.md) |
| Suwayomi Server — Suwayomi/Suwayomi-Server | The diagnostic ran, but the required report could not be selected. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/source-bound-configuration-input-corrections-v1/sbic-e002-strict-diagnostics/result.json) |
| Testcontainers Java — testcontainers/testcontainers-java | The initial build check passed; no candidate speed comparison followed. | [Build observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/configuration-input-native-corrections-v1/cinc-e002-cohort-freeze.json) |

### Source or workflow inspection: eight projects

These projects supplied examples of build code or were considered for later experiments. The records below do not establish comparative build performance.

| Project and repository | What the inspection established | Evidence |
| --- | --- | --- |
| BlueMap | The proposed change could alter Git index metadata, so it was rejected during source inspection. | [Source review](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/source-bound-configuration-input-corrections-v1/sbic-e001-source-detector/result.json) |
| Flyway — flyway/flyway | Source reviewed; later workflow selection identified a Maven build, outside that Gradle experiment. | [Source review](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/prospective-reviewed-native-patch-controlled-trial-v1/source-classification.json) |
| Gradle — gradle/gradle | Source or workflow reviewed; no accepted speed comparison followed. | [Source review](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/economics-gated-reviewed-native-patch-v1/source-classification.json) |
| JUnit 5 / JUnit Framework — junit-team/junit5 | Source or workflow reviewed; no accepted speed comparison followed. | [Source review](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/economics-gated-reviewed-native-patch-v1/source-classification.json) |
| Kotlin — JetBrains/kotlin | Source or workflow reviewed; no accepted speed comparison followed. | [Source review](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/economics-gated-reviewed-native-patch-v1/source-classification.json) |
| Liquibase — liquibase/liquibase | Source reviewed; later workflow selection identified a Maven build, outside that Gradle experiment. | [Source review](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/prospective-reviewed-native-patch-controlled-trial-v1/source-classification.json) |
| OpenRewrite — openrewrite/rewrite | Source or workflow reviewed; no accepted speed comparison followed. | [Source review](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/economics-gated-reviewed-native-patch-v1/source-classification.json) |
| Quarkus — quarkusio/quarkus | Source or workflow reviewed; no accepted speed comparison followed. | [Source review](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/economics-gated-reviewed-native-patch-v1/source-classification.json) |

### Supporting projects, forks and dependencies

The controlled Kotlin and Groovy cache examples used **buildopt-pilot** and **buildopt-pilot-groovy**. The internal **buildopt-poc/incremental-learning-fixture** supplied predictable changes for learning and replay checks. These small test projects do not provide independent evidence from large open-source builds. [Cache examples](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/cache-parity-v1-local.json), [learning experiment](https://github.com/tonyredondo/buildopt/blob/main/docs/plans/sticky-wrapper-learning-poc-tracker.md).

The **Micronaut Core** and **Spring Framework** forks were used to review the patches already counted under their upstream projects. **BuildOpt** is the tool under investigation. **bndtools/bnd** appeared as a plugin dependency in the GraphQL Java study; it was not a separate performance subject. [Patch review](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/reviewed-native-patch-owner-acceptance-v1/README.md), [GraphQL Java dependencies](https://github.com/tonyredondo/buildopt/blob/main/specs/poc-complete-native-correction-v1.subjects.json).

These projects supplied build measurements, diagnostic failures and source examples. To establish broader savings, the next study needs the same approach to work on different project structures through their histories.

## Safe caching and runtime tuning

**Safe caching, SDTEST-3934: no established advantage over Gradle's cache when it is working.** Gradle can save a task's output and reuse it when the inputs match. BuildOpt's extra cache path worked correctly, but the useful comparison is with Gradle caching enabled.

The [two small cache examples](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/cache-parity-v1-local.json) saved 15.87% and 13.65% against caching disabled. Against Gradle's own local cache, the Kotlin example saved only 2.5 ms and the Groovy example lost 49.5 ms. Each comparison used four pairs with matching required results. This shows the benefit of enabling a cache, without establishing an additional saving from BuildOpt.

The separate [Mockito test-compilation experiment](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/poc-mockito-test-build-v1.json) saved an average 281.375 ms, below its 500-ms requirement. Five of eight pairs were faster, and the statistical range also allowed a loss. The 1,260 required class files matched. This compiled tests; it did not execute them. Research into an additional general-purpose cache remains closed. Mockito's different Build Impact result is described below.

**Runtime tuning, SDTEST-3935: the tested settings will not be applied.** In [Spring's worker experiment](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/poc-runtime-research-v1.json), reducing workers from twelve to six made the build 2.00% slower. Earlier CPU and memory changes also made builds slower. Keeping OpenTelemetry build state ready reduced one preparation step, but the full build became 7.68% slower. The smaller step did not translate into a faster build. [Runtime findings](https://github.com/tonyredondo/buildopt/blob/main/docs/findings/build-optimization-performance.md).

The CPU limits used in the later Checkstyle experiment test how that correction behaves with fewer resources. They do not reopen the search through runtime settings.

## Selected build savings

**Build Impact, SDTEST-3936**, chooses which parts of a project need to be built for a change. **Patch Autopilot, SDTEST-3937**, prepares a small correction to the build itself. Gradle then runs with that correction. Their results need to be assessed separately.

Two task names need explanation. Elasticsearch's **ForbiddenPatterns** checks files for text patterns the project disallows. **Checkstyle** checks Java source against coding and formatting rules and produces violation reports. Reusing a whole completed task through Gradle's cache is different from checking only changed files when the task has to run.

### Ktor: Build Impact on library compilation

The `jvmJar` experiment built the libraries needed for a selected change. BuildOpt reduced the selected projects from 133 to ten. Mean elapsed time fell from 38.810 to 7.830 seconds: **79.82% saved**, with eight faster comparisons and matching required results. A separate change to shared build logic correctly kept the full build.

The recorded setup time would be recovered after an estimated 26 applicable builds. We did not observe those 26 later builds. This remains a strong result for the selected command, with its frequency and lifetime through Ktor development unmeasured. The later unsuccessful five-project study of saved plans did not include Ktor. [Experiment and original results](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/poc-magic-end-to-end-value-v2/README.md).

### Apache Beam: Build Impact on project compilation

The `classes` experiment reduced the selected projects from 316 to six. Mean time fell from 65.081 to 24.958 seconds: **61.65% saved**, again with eight faster comparisons and matching required results.

The estimated setup recovery was 28 applicable builds. As with Ktor, the test did not show how often the plan could apply to later Beam changes. Beam was not part of the later unsuccessful five-project study either. It saved time on the selected build. Sustained benefit for that setup remains unmeasured. [Summary and original results](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/poc-magic-end-to-end-value-v2/summary.json).

### Micronaut Core: a correction to Python compilation

The task `compileVfsPythonBytecode` compiles Python files bundled with Micronaut. The patch corrected how file paths were treated and allowed Gradle to reuse the output safely. Mean task time fell from 10.909 to 3.988 seconds: **63.44%, or 6.921 seconds saved**. All eight pairs were faster. Required outputs matched, and undoing the patch was checked.

This is a positive task correction. We have not measured its contribution to a complete build through a fresh history. Later failures in selecting saved build plans do not establish that this patch stopped working. [Patch results](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/reviewed-native-patch-portfolio-v1/result.json).

### Spring Framework: a correction to architecture checks

The `checkArchitectureMain` task checks rules about code structure and dependencies in Spring Core. Allowing reuse of its previous result reduced mean task time from 2.789 to 1.803 seconds: **35.34%, or 0.986 seconds saved**. All eight pairs were faster, with matching results and a successful reversal check.

The correction passed its original task-level criterion. Its smaller absolute saving makes frequency important: it has not been shown to meet the current whole-build history requirements. That is an unmeasured question, not a failed history test for this patch. [Patch results](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/reviewed-native-patch-portfolio-v1/result.json).

### Elasticsearch: caching the ForbiddenPatterns result

The patch allowed Gradle to reuse the completed task. Mean time fell from 46.139 to 38.967 seconds: **15.54%, or 7.172 seconds saved**, with all eight pairs faster. Before each invocation, the experiment removed the marker showing that the task had completed. The candidate could restore its cached result. The saving applies to those conditions. [Corrected experiment](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/economics-gated-reviewed-native-patch-v2/README.md).

A later idea would also check only changed files when this task ran. That study already included the earlier caching patch on both sides. Gradle skipped ForbiddenPatterns on 17 of 20 builds; even removing the task entirely fell below the required average saving. We rejected this additional incremental change. That decision does not invalidate the earlier patch or prove its benefit through an ordinary history. [Opportunity calculation](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv002/opportunity.md).

### Elasticsearch: reusing Checkstyle checks

The selected experiment ran the server's checks before accepting a change (precommit) on two code changes. The new Checkstyle implementation reused checks for unchanged files while preserving the full reports of coding-rule violations. The patch was applied directly in the experiment; automatic discovery and delivery were not part of this speed result.

It saved **41.29%, 40.11% and 24.12%** with eight, four and two CPUs, using two measured pairs per setting. Across a different sequence of thirteen changes, the corrected version was **19.16% slower overall**. The correction has a selected benefit, but the daily-development result remains unresolved. The investigation and revised implementation are described in the history section below. [CPU and history results](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-screen-completion/README.md).

### Apache Kafka: Edge Cache under controlled remote access

**Edge Cache, SDTEST-3939**, keeps cached files close to the build machine. The Kafka comparison compiled the client code and tests with `:clients:testClasses`, restoring the same required files while changing the cache location. It saved **15.21%, or 1.351 seconds**, across four faster pairs. Two other controlled experiments saved 34.74% and 35.07%. These show that avoiding an expensive remote read can help. [Kafka](https://github.com/tonyredondo/buildopt/blob/main/docs/findings/buildopt-kafka-edge-cache-experiment.md), [controlled remote cache](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/poc-remote-cache-value-v1.json), [adaptive cache path](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/adaptive-cache-locality-v1.json).

The broader [second study](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/remote-cache-locality-value-v2/README.md) stopped before timing: Groovy had too little cached data and Kafka's required output differed between ordinary Gradle runs. The remaining three projects did not run after those failures.

The corrected [third study](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/remote-cache-locality-value-v3/README.md) completed 24 pairs on three projects. None passed all criteria: Groovy missed the 2% minimum, OpenTelemetry was slower, and Spring's statistical range included a loss. Edge Cache therefore remains closed as the default accelerator. Its earlier result supports the specific remote-access conditions tested.

## Additional selected savings

These experiments also found useful reductions. Each needs its own conclusion; a failure in a different optimization does not erase a measured win.

| PoC and repository | What ran and what it saved | What happened afterward |
| --- | --- | --- |
| Build Impact — Mockito | After a change confined to the JUnit Jupiter extension, the build compiled that extension rather than all five listed projects. The eight-pair mean fell from 4.085 to 3.341 seconds: 18.22% / 0.744 seconds. Six pairs were faster, and the original criterion passed. | Recurring benefit through a fresh history was not established. The later Mockito cache experiment was a different intervention. [Record](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/poc-real-world-performance-v1.json). |
| Build Impact combined with a packaging correction — OpenTelemetry Java Instrumentation | A selected Spring-instrumentation compilation workload reused build preparation and cached a standard library-packaging task. Four pairs saved 39.92% / 4.377 seconds with matching required outputs. | The combined result cannot assign all the saving to one component. Related packaging changes in Spring were slower, and the Kafka task did not fit the same correction. This supports the measured OpenTelemetry setup. [Record](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/poc-otel-optimization-v2.json), [follow-up results](https://github.com/tonyredondo/buildopt/blob/main/docs/findings/build-optimization-performance.md). |
| Build Impact — Apache Kafka | The `testClasses` command compiled production and test code after a selected dependency change. It saved 19.09% / 6.866 seconds in eight faster pairs. The required outputs matched. | In the later chronological study, the saved plan was used on none of three subsequent changes. Including recorded extra work, that study lost 58.055 seconds. The selected saving did not become a sustained result. [Selected comparison](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/history-admitted-paired-value-v1/README.md), [history](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/three-class-chronological-value-v1/README.md). |
| Build Impact — Apache Groovy | The `classes` command compiled project code after a change in a repeatedly modified part of Groovy. It saved 29.00% / 9.287 seconds in eight faster pairs, with matching results. | The later chronological study stopped with a 2.995-second loss after recorded extra work. It did not establish a positive history result. [Selected comparison](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/recurrent-root-paired-value-v1/README.md), [history](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/three-class-chronological-value-v1/README.md). |
| Build Impact — Spring Messaging | The `testClasses` command compiled production and test code after a selected Spring Messaging dependency change. It saved 19.44% / 7.803 seconds in eight faster pairs, with matching results. | Its part of the later three-project study was not run because earlier results prevented the study from passing. Its sustained saving was therefore unmeasured, not a measured regression. [Selected comparison](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/spring-messaging-paired-value-v1/README.md), [history](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/three-class-chronological-value-v1/README.md). |

The early Mockito, SpotBugs and Spotless study also explains why averages alone are insufficient. SpotBugs' apparent saving on a change to an independent project was uncertain; Spotless' saving was too small. Only Mockito qualified. These were controlled code changes, not complete histories. [Three-project comparison](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/poc-real-world-performance-v1.json).

## Why saved build plans remain closed

The negative evidence concerns how often the optimizer could safely apply a useful plan. Large savings on selected commands did not solve that problem.

| Investigation | Result | Decision |
| --- | --- | --- |
| Whole saved plans across five repositories | One of five saved time after recorded extra work. A plan was selected on one of six eligible later builds. | Too few recurring applications; closed. [Decision](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/poc-functional-coverage-decision-v1/README.md). |
| Adapting smaller parts of plans | Across Groovy, Kafka, Spring, OpenTelemetry and Micronaut, 100 comparisons applied no optimization. All required outputs matched, but total time was 368.623 seconds worse. Of that difference, 179.029 seconds was recorded BuildOpt work; 189.593 seconds was the remaining Gradle/machine difference, without a demonstrated cause. | Smaller plans did not solve the problem of finding a plan that could be used; closed. [Attribution](https://github.com/tonyredondo/buildopt/blob/main/docs/findings/buildopt-build-plan-history-experiment.md), [decision](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/adaptive-fragment-terminal-decision-v1.json). |
| Following selected Kafka, Groovy and Spring plans | Kafka and Groovy together lost 61.050 seconds in the completed part of the study. Spring's stage was not run. | The study stopped without sustained value. This is not a negative measurement for Spring. [Record](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/three-class-chronological-value-v1/README.md). |
| Looking for recurrence before building | Source review covered 320 history rows across five repositories. Only Kafka met the recurrence requirement, against three required. | No timing followed; closed for insufficient recurring opportunities. [Record](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/economic-opportunity-first-v1/README.md). |

These studies did not test the lifetime of the Micronaut, Spring architecture or Elasticsearch caching patches. Those corrections remain separate task-level results.

## Other searches and their stopping reasons

Some approaches failed a performance or correctness test. Others never reached one. The distinction matters when deciding what the research has ruled out.

| Approach | What prevented progress | Status |
| --- | --- | --- |
| Find build plans from changed files, dependencies and repeated commands | More candidates appeared, but too few projects supplied complete inputs and usable actions. Later source reconstruction covered 110 changes without enough breadth for timing. | Closed without a performance result for those candidates. [Search history](https://github.com/tonyredondo/buildopt/blob/main/docs/findings/buildopt-generalization-audit.md). |
| Learn from observed build requests | 128 observations covered 113 changes. Missing reports or incomplete inputs prevented safe use across the required projects. | Closed before timing. [Decision](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/observed-request-portfolio-terminal-decision-v1.json). |
| Reuse a previous result without starting Gradle | None of 69 public-history cases met every safety condition; 35 had mismatched outputs. | Closed before timed use without Gradle. [Replay](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/verified-request-hit-shadow-replay-v1.json). |
| General rules for allowing tasks to use cached results | The first search failed a directory-input correctness check. A path-handling follow-up passed some checks but did not finish all required checks. | Original correction rejected; follow-up incomplete. No general performance proof. [First study](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/durable-native-optimization-v1/README.md), [follow-up](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/normalization-aware-cacheability-v2/README.md). |
| More reviewed build patches | Two of four initial proposals passed: Micronaut Python compilation and Spring architecture checks. OpenTelemetry version generation and Spring shadow-source tasks failed their savings criteria. A later five-repository search found one actionable Hibernate patch, which was 1.26% slower despite correct results. | Broad discovery did not supply enough useful corrections. Keep the two measured wins; reject the failed candidates. [Initial proposals](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/reviewed-native-patch-portfolio-v1/result.json), [Hibernate](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/reviewed-native-patch-customer-economics-v1/README.md). |
| Prospective patch search in ten new repositories | Three possible changes looked safe in source review. Only Shadow reached the workflow check; the relevant task took 80 ms in a 471-second build and did not delay completion. | Closed before candidate timing; too little whole-build opportunity. [Record](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/prospective-reviewed-native-patch-controlled-trial-v1/README.md). |
| Start with work delaying the end of a build | Four of ten workflows supplied complete data. Their large tasks used standard Gradle/Kotlin code. A later search found six tasks whose code explicitly disabled caching, all in tasks too small to justify correction. | No qualifying patch; six incomplete workflows remain unanswered. [First study](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/critical-path-first-reviewed-native-patch-v1/README.md), [source follow-up](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/critical-path-build-logic-correction-v1/README.md). |
| Coordinate corrections through the BuildOpt launcher | Capturing and coordinating build observations worked, but only one of ten repositories supplied a sufficiently large usable change, against three required. | Implementation retained; discovery route closed before candidate timing. [Final decision](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/wrapper-coordinated-native-corrections-v1/wcncp-e013-final.json). |
| Correct build-preparation inputs | The first study completed seven of ten assessments and found no eligible repository. The source-based follow-up completed three diagnostics but tied an observation to the relevant source in only one, against two required. | Both studies stopped before candidate timing. [First study](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/configuration-input-native-corrections-v1/README.md), [follow-up](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/source-bound-configuration-input-corrections-v1/README.md). |
| Complete a preparation correction on GraphQL Java | Warmed build preparation took only 362 ms, below the 500-ms requirement. Required archive files also differed between ordinary Gradle runs. | Closed before applying a correction. [Admission result](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/complete-native-correction-v1/cnc004-native-admission/README.md). |
| Run an installed correction on Elasticsearch | Output and integration checks passed, but persistent delivery of build observations failed. | Installation path incomplete; installed performance and history savings were not measured. [Record](https://github.com/tonyredondo/buildopt/blob/main/docs/findings/buildopt-elasticsearch-installed-experiment-2026-09-08.md). |

These results justify retiring the specific searches already tried. They do not establish that every cause of unnecessary rebuilding has been examined. A new candidate needs an observed build-time problem, not just source code that resembles a previous patch.

## Elasticsearch history: the current unresolved experiment

The sequence contains 100 consecutive changes along the main line of Git history, spanning about 46 hours. Twenty changes are for development and measurement checks; eighty remain reserved for validation and have not run. Histories are also prepared for Groovy, Kafka, Spring, OpenTelemetry, Micronaut and Hibernate. Preparing those histories is not a completed validation on those projects.

Both Elasticsearch versions use Gradle's build cache and the earlier ForbiddenPatterns cacheability patch. **Configuration Cache** is a separate Gradle feature that reuses build preparation. It is disabled here because the selected workflow failed with it enabled.

| Stage | What we learned | Conclusion |
| --- | --- | --- |
| Additional incremental ForbiddenPatterns check | Gradle already skipped the task on seventeen of twenty builds. Removing all its recorded work would save at most 0.787 seconds per build, or 3.134%. | Below the one-second and 5% requirements; no implementation of this additional idea. [Calculation](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv002/opportunity.md). |
| Ordinary Checkstyle caching | Simply enabling the existing cache omitted parts of the reports and missed required violations when timestamps were preserved or rules changed. | That implementation was unsafe. [Initial checks](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/checkstyle-admission/README.md). |
| Checkstyle tracking file contents | The prototype kept complete reports through changes to files and rules, failures, recovery and removal of the patch: twenty cases and 67 observations. An optimistic estimate suggested a 5.945% whole-build saving, leaving little room for extra work. This was a model, not a measured speedup. | The output checks passed. A measured speedup was still needed. [Prototype](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/checkstyle-prototype/README.md). |
| Corrected CPU and history comparisons | Selected changes were faster at all three CPU settings. The separate thirteen-change sequence was 19.16% slower; Checkstyle ran on only one change. All 54 builds succeeded and all 27 output comparisons matched. The experiment also failed to keep every monitoring thread within its assigned CPU limit. | Selected savings exist. A reliable sustained saving was not established. [Results](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-screen-completion/README.md). |
| Slowdown investigation | Checkstyle was still reusing earlier checks. A fresh diagnostic run was faster, and subsequent requests returned to roughly seven seconds. A previous long delay was not reproduced. Some profiling data was missing. | Lost Checkstyle history did not explain the observed slowdown. The cause of every delay remains unresolved. [Investigation](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-regression/README.md). |
| Revised saving of Checkstyle state | The change passed failure and recovery checks. The timed comparison against the previous optimized implementation had one faster pair and one slower pair. | The agreed performance rule did not pass. This was not a comparison against ordinary Gradle. [Latest result](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-finalization/README.md). |

The latest timing result is easier to judge by looking at its two measured comparisons:

| Comparison | Previous optimized version | Revised optimized version | Result |
| --- | --- | --- | --- |
| First | 124.537 s | 83.160 s | 41.377 s faster |
| Second | 83.003 s | 97.339 s | 14.337 s slower |

The average improved by 13.03%, but both comparisons had to improve under the rule agreed beforehand. Only one did. The revised implementation reused the same amount of Checkstyle work in both comparisons, so loss of that checking history does not explain the difference. The machine was shared and busy at times; this is a plausible source of timing variation, not a proven explanation for every delay. Both results remain in the record. [Timing data](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-finalization/profile.json), [shared-machine assessment](https://github.com/tonyredondo/buildopt/blob/main/docs/findings/buildopt-checkstyle-shared-host-disposition-2026-09-10.md).

Earlier measurement attempts also had recording overhead and unstable timings. An initial precision estimate would have required far more comparisons than the declared run limit, so that confirmation did not run. Correcting the measurement tools was necessary to interpret the data, but it was not a build-time saving. The [tracker](https://github.com/tonyredondo/buildopt/blob/main/docs/plans/buildopt-product-viability-v1-tracker.md) retains these attempts and their status.

The next unresolved check is the proposed [four-build identical-code comparison](https://github.com/tonyredondo/buildopt/blob/main/docs/plans/buildopt-checkstyle-measurement-control-v1.md). It asks whether the measurement schedule can produce a large difference when neither side contains a different optimization. It has not run. A valid comparison against ordinary Gradle and the reserved history would still be required afterward. Experiments remain paused for this review.

<a id="what-the-retained-implementation-is-for"></a>

## What the existing implementation is for

| PoC or tool | What worked | Intended use and remaining limit |
| --- | --- | --- |
| Safe caching and Edge Cache | Cache connections, verified storage and fallback worked in the tested cases. | Support the experiments that need them. They have not earned a role as a general accelerator. [Connection](https://github.com/tonyredondo/buildopt/blob/main/internal/launcher/central_gradle_cache.go), [storage](https://github.com/tonyredondo/buildopt/blob/main/internal/edgecache/store.go). |
| Build Impact planner | Checks could reject an inapplicable plan and return to Gradle. | Retain the analysis and safety code. The tested adaptive plan-selection approach is closed because it did not produce recurring savings. [Planner](https://github.com/tonyredondo/buildopt/blob/main/internal/adaptivefragment/planner.go). |
| Patch Autopilot | Patches could target known source, be checked and be undone. Both patches passed the controlled delivery check and were accepted by the same reviewer. The draft-delivery test used a simulated pull-request service. | Support experiments with known corrections. Automatic discovery and unattended delivery across changing projects remain unproven. [Patcher](https://github.com/tonyredondo/buildopt/blob/main/jvm/patcher/README.md), [delivery](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/reviewed-native-patch-delivery-v1/README.md), [review](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/reviewed-native-patch-owner-acceptance-v1/README.md). |
| Build History, SDTEST-3938 | Build records, an API and a local dashboard were implemented. | Used to inspect experiments. No independent reduction in build time has been measured. [Records](https://github.com/tonyredondo/buildopt/blob/main/internal/buildhistory/history.go), [API](https://github.com/tonyredondo/buildopt/blob/main/internal/buildhistory/http.go), [dashboard](https://github.com/tonyredondo/buildopt/blob/main/internal/buildhistory/dashboard.go). |
| History replay tools | They advance through fixed commits, separate build states and compare required outputs. | Needed for the proposed validation. Measurement overhead and supported workflows still require checks for each setup. [Tools](https://github.com/tonyredondo/buildopt/blob/main/dev/history-replay/README.md). |
| Revised Checkstyle patch | It reused previous checks and saved their state at the end of the build through a supported Gradle mechanism. | Current research candidate. Tested with Gradle 9.7.1 and Configuration Cache disabled; sustained savings against ordinary Gradle are unproven. [Patch](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-finalization/candidate-v2.patch). |

The [two-machine wrapper trial](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/sticky-wrapper-two-machine-v1.json) also showed that the installed tool could reuse the required state on another machine and fall back to Gradle when offline. It tested that process, not its speed.

## Evidence required to continue

The research recommendation is to investigate corrections that avoid repeating work inside ordinary builds. Existing task savings justify testing that idea further. They do not justify declaring the PoC successful.

The proposed research OKRs use the technical parts of the [existing acceptance criteria](https://github.com/tonyredondo/buildopt/blob/main/docs/findings/buildopt-research-validation-criteria.md):

| Question | Required evidence |
| --- | --- |
| Does the correction remain correct? | Preserve all required files and reports. Exercise changes to files and rules, failures, recovery and removal. Unsupported changes must stop reuse safely. A refused optimization still contributes its elapsed time. |
| Does it help through real development? | Two complete 100-change runs from separate state, with twenty development changes and eighty reserved validation changes. Each run must save at least 5% overall and an average of at least one second per scheduled validation build after the required added work. The statistical result must support a positive saving while accounting for related commits. Setup time must be recovered and remain recovered within the sequence. Keep all scheduled outcomes and the existing limit on slow-build regressions. |
| Does the same approach transfer? | Test two additional independent repositories selected before their performance is known. At least two of the three total must pass the same history criteria. Keep the failures and inapplicable cases in the result. |
| Does adaptation add value? | Compare ordinary Gradle, a fixed correction and an adaptive version on history not used to develop its policy. It must detect loss of usefulness, validate replacements and stop unsafe reuse. It must save more than the fixed correction after its own checking and search work. A single slow build on a busy machine should not provoke repeated searches. |

Fixing a measurement problem allows an experiment to proceed; it does not establish an optimization. An incomplete or uncertain run remains unproven. If recorded builds offer another concrete cause of substantial repeated work, that can justify a new candidate. If the remaining approaches cannot produce positive results across histories and repositories, close the initiative and preserve the evidence.

## Source record

The research snapshot reviewed here was [published in the repository](https://github.com/tonyredondo/buildopt/tree/0e89140e98554cb94a84963badbcf1bb747d6e67/). The source files and numerical results were checked against the retained records. These wording changes add no new experimental results.

The six work-item names and IDs come from the supplied screenshot. No issue status was changed. This research covers Gradle build work, including compilation and checks needed before tests. It does not assess selecting fewer tests or the wider Test Optimization offering.
