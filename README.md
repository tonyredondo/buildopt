# BuildOpt

BuildOpt makes Gradle builds faster without changing their expected outputs.
It runs the existing Gradle command, observes what happened, and activates
only optimizations that have enough evidence. If an optimization is unavailable
or cannot be proven safe, the original build remains authoritative.

> **New here?** You do not need to read the implementation tracker, contracts,
> or architecture documents first. On Linux, follow **Get your first result**
> below. It uses a synthetic repository, needs no external service, and does
> not modify one of your projects.

## How it fits into a build

```text
your Gradle command
        |
        v
BuildOpt launcher -----> optional verification and cache gateway
        |                            |
        v                            v
      Gradle              optional Shared or Edge Cache
```

The launcher preserves the command, output, and exit status. BuildOpt records
evidence around that execution and uses conservative fallbacks: a rejected
cache entry becomes a normal cache miss, an unqualified optimization is not
applied, and `BUILDOPT_BYPASS=1` removes the optimization path immediately.

> **Installed two-machine proof:** the committed `./buildoptw` command now
> bootstraps one SHA-verified archive on isolated producer and consumer
> machines, publishes two Gradle cache tasks over HTTPS, and restores both on a
> clean read-only consumer after owner commit and service restart. An offline
> run falls back to native Gradle and reproduces the exact output. This is
> functional POC evidence (`wallTimeClaim=false`), not a speedup claim. See
> the [checked result](./benchmarks/results/sticky-wrapper-two-machine-v1.json)
> and [contract](./specs/poc-sticky-wrapper-two-machine-v1.md).

> **Reviewed-profile research:** this is an owner-operated proof of concept. In a fresh
> preregistered balanced rerun, the same generic structural Build Impact method
> qualified independently on Spring, OpenTelemetry, Kafka, Micronaut, and
> Groovy at **14.97% to 87.35% lower wall time** than optimized native Gradle.
> All 80 raw pairs improved, required outputs matched, tails improved, and full
> fallbacks passed. The unchanged generic path then transferred to Ktor's
> public JVM JAR workflow: **86.21% lower wall time**, 16/16 positive pairs,
> exact output, stable task shapes and both full-graph fallbacks. Three new
> Ktor change classes then qualified independently: dependency source at
> **85.80%**, a JVM resource at **86.51%**, and a two-module mixed-source
> change at **77.98%** lower wall time; all 48 pairs improved, while two root
> configuration probes correctly retained native Gradle. A separate
> semantic-output rerun also qualified Groovy
> `jar` (**73.10%**), Kafka Checkstyle (**29.73%**), and Kafka `shadowJar`
> (**66.55%**) across 48/48 positive raw pairs. A fresh change-breadth matrix
> then qualified six distinct source edits across those three workflows at
> **28.00% to 79.54% lower wall time**; all 96 raw pairs improved, while all
> eight build-logic/global-change probes correctly retained native Gradle.
> One-time profile calibration now has explicit and improved economics:
> digest-bound replay, fused discovery and adaptive stabilization reduce cold
> discovery by **8.01%–21.08%**. Reviewed Groovy JAR and Kafka `shadowJar`
> cells now repay after **9–13 qualifying builds**, while Kafka Checkstyle
> needs **24–26**; exact repeat evaluation repays after **4–12**.
> These results are bound to exact changes, workflows, and reviewed output contracts;
> profiles remain review-required and are not production-authorized. See the
> [current one-pager](./docs/findings/buildopt-poc-handoff.md)
> and [generalization audit](./docs/findings/buildopt-generalization-audit.md).

> **Current automatic status:** immutable public `v0.6.1` has completed the
> zero-manual-file terminal POC gate. From fresh Ktor state,
> `buildopt optimize jvmJar --max-workers=12` reduces 133 to ten projects and
> measures **79.82% lower wall time** with 8/8 positive pairs and 26-build
> payback. From fresh Apache Beam state, `buildopt optimize classes
> --max-workers=12` reduces 316 to six projects and measures **61.65% lower
> wall time** with 8/8 positive pairs and 28-build payback. Both preserve exact
> required outputs, lower p95 and pass full-graph fallback. A Ktor root
> build-logic change completes native Gradle and safely creates no calibration
> or timing claim. See the
> [published terminal result](./benchmarks/results/poc-magic-end-to-end-value-v2/README.md);
> the [v1 matrix](./benchmarks/results/poc-magic-end-to-end-value-v1/README.md)
> remains historical diagnostic evidence rather than being rewritten.

> **Current terminal decision:** one exact executable ran frozen Spring,
> OpenTelemetry, Kafka, Micronaut and Groovy ordinary-build windows. Four
> short-lived hypotheses stopped after one requested build, avoiding 64
> additional learning builds. Kafka qualified at **21.43% faster with 8/8
> positive pairs**; one of six eligible descendants selected the profile and
> saved **135.127 seconds / 75.67%**, leaving Kafka **82.527 seconds net
> positive** after qualification, publication and measured fallback wrapper
> work. Across the complete matrix only **1/5 repositories** is net positive
> and **1/6 eligible descendants (16.67%)** selects, below the frozen 3/5 and
> 50% breadth gates. All 27 output observations are exact and product failures
> remain zero. Applying every frozen criterion without moving thresholds yields
> **`STOP_GENERIC_POC`**: 1/5 net-positive families, 1/6 selected eligible
> descendants and 4.098 seconds of observed pre-Gradle retention overhead all
> miss their gates. This stops the current generic structural-profile
> hypothesis, not the bounded mechanisms and evidence that did work. See the
> [terminal decision](./benchmarks/results/poc-functional-coverage-decision-v1/README.md)
> and [lifetime breadth V3 evidence](./benchmarks/results/poc-lifetime-breadth-v3/README.md).

> **Adaptive successor result:** the now-closed
> [Adaptive Fragment Generalization Tracker](./docs/plans/adaptive-fragment-generalization-tracker.md)
> replaces all-or-nothing structural profiles with independently compatible
> producer, subgraph, task, patch and cache-locality fragments. It requires
> sub-second native retention, chronological no-lookahead learning and positive
> cumulative value in at least three of five public repository families before
> restoring any generic claim. Direct AF-011 timing shows 68.56% Groovy
> and 79.32% Kotlin savings when Build Impact and a reviewed task patch run
> together, with exact outputs. The composition remains unauthorized because
> Kotlin Build Impact reaches only 6/8 positive isolated pairs versus the fixed
> 7/8 gate. AF-013 preserves 14 historical chronological comparisons: Spring is
> **+59.550 s**, Kafka **+88.219 s**, OpenTelemetry **-168.751 s**, Groovy
> **-37.684 s**, and Micronaut remains `INCONCLUSIVE` after a byte-
> reproducibility rejection. Only **2/5** rows are positive versus the frozen
> 3/5 target. Those measurements audit earlier behavior but do not represent the
> current adaptive implementation. The current installed campaign now closes
> 100 exact-output pairs across the same five repositories with zero product
> failures. All 100 candidates retain native Gradle, no fragment activates,
> only 25 pairs improve and cumulative signed value is **-368.623 seconds**.
> AF-014D attributes **179.029 seconds** to the recorded BuildOpt path and
> **-189.593 seconds** to residual Gradle/runner variation. Discovery/learning
> is the largest recorded slice at **98.385 seconds**; native retention is
> **0.531 seconds p50 / 8.656 seconds p95**. Because no profile or fragment
> activated, attributable mechanism saving is **zero** and the current outcome
> is `CURRENT_VALUE_NOT_ATTRIBUTABLE`. AF-015 now closes the successor as
> `STOP_ADAPTIVE_FRAGMENT_POC`: 9/15 frozen criteria pass, but activation is
> **0/71** eligible builds, breadth and positive confidence are both **0/5**,
> no family repays and native-retention tails exceed the fixed limits. With
> zero activated saving, bounded specialization is not evidence-backed either.
> The result stops this hypothesis; it does not turn historical isolated wins
> into a current customer claim.

> **Closed fresh-evidence experiment:** BuildOpt tested a repository-committed
> wrapper as the sticky customer integration. A maintainer can now generate
> `buildoptw`, `buildoptw.bat`, checksum-pinned wrapper properties and portable
> non-secret configuration deterministically with `buildopt wrapper init`.
> The generated scripts now select, download, checksum, safely extract and
> atomically cache the pinned native package, with verified zero-network warm
> reuse. Developers and CI can now run `./buildoptw <gradle args...>` without a
> global BuildOpt installation or a
> hand-authored profile. The implemented wrapper now composes verified
> bootstrap, passthrough, optional Gradle-compatible HTTPS cache, bounded
> observation, budgeted paired trials, conservative value qualification,
> signed active execution, suspension and native fallback. Cache objects never authorize actions, server
> failure retains native Gradle and credentials remain private. Continuation
> requires positive cumulative value in at least three of five public families
> after every wrapper, learning, trial, cache and fallback cost. Follow the
> [Fresh Generic Optimization POC Tracker](./docs/plans/fresh-generic-optimization-poc-tracker.md).
> The deterministic lifecycle proof and all ten native-fallback negatives are
> checked by `./dev/check-sticky-wrapper-learning-lifecycle`. The first public
> opportunity gate found **0/5** complete actions, but a route
> audit proved that its historical inputs lacked the generic public producers
> needed to distinguish missing input from no opportunity. That result is now
> diagnostic-only. The closed experiment started from zero BuildOpt evidence,
> now has complete generic producers with typed conclusive and incomplete
> outcomes. The five-family cohort is now frozen and captured from empty
> state: all five inputs are complete, one Spring task-contract action is
> testable and no declared-graph action is safe. Independent recomputation
> confirms **1/5** action breadth against the fixed **3/5** threshold, so
> installed timing and the chronological campaign are not authorized. The
> terminal scorecard now records
> `STOP_FRESH_GENERIC_POC_FOR_CURRENT_DETECTOR_SET`. Value, confidence,
> payback and overhead were not measured because timing never opened. This
> rejects these two detectors as a broad route; it does not claim that Gradle
> itself cannot be improved by a different generic mechanism.

> **Current generic hypothesis:** BuildOpt now maps real adjacent-commit
> changes through finalized Gradle inputs and direct/transitive output
> producers to the exact work and output closure required by that change. The
> fresh producer completed 25/25 conclusive transitions across Spring,
> OpenTelemetry, Kafka, Micronaut and Groovy. Spring exposes one exact testable
> action; the other 24 transitions safely retain the ordinary requested graph.
> No wall time was measured and no action was activated. The next block must
> independently recompute whether safe actions reach the unchanged 3/5-family
> breadth gate before any timing is allowed. Follow the
> [change-aware tracker](./docs/plans/change-aware-producer-closure-poc-tracker.md).

> **Ordinary-build evidence:** the sticky wrapper now records private,
> append-only phase timing and provenance for requested Gradle builds. The
> first real sample shows 19.876 s for a cold Gradle 9.6.1 invocation and
> 3.732 s when Configuration Cache is reused; this is instrumentation evidence,
> not a speedup claim. The first four-pair candidate/native trial is now
> complete: BuildOpt averages 7.534 s versus 6.979 s for optimized native
> Gradle (0/4 positive pairs), while all 4/4 output trees match exactly. The
> result is a safe-measurement proof and a current overhead warning, not a
> speedup claim; it authorizes no action.

> **Wrapper overhead hardening:** the common sticky-wrapper path now resolves
> to native Gradle without starting BuildOpt services when no configured
> consumer needs them. Ordinary observation defaults to a lazy `light` mode;
> it skips the pre-build Git lookup, computes the executable digest concurrently
> when possible and writes its private record only after the child exits. A
> 20-sample local microbenchmark measured **+9 ms p95** for
> native no-op and **+38 ms p95** for light observation over direct execution,
> with **0.093 ms p95** pre-child decision time. These are bounded wrapper-cost
> results, not a Gradle speedup claim; the full numbers and reproducible check
> are in the [overhead contract](./specs/poc-sticky-wrapper-noop-overhead-v1.md).

> **Historical longitudinal diagnostic:** a bounded `SWL-015 v1` run compared the
> committed `./buildoptw` command with optimized native Gradle on one frozen
> current revision in each of Spring Framework, OpenTelemetry Java
> Instrumentation, Apache Kafka, Micronaut Core and Apache Groovy. The five
> pairs produced exact required outputs with zero product failures; **2/5**
> were positive and **3/5** negative, for **-22.149 s** signed value. A route
> audit found that v1 injected `--build-cache` only into control and exercised
> candidate no-op/light observation with no trial or action. It is now immutable
> `DIAGNOSTIC_ONLY` compatibility evidence and cannot feed the terminal
> decision. Cache-symmetric arms and lifecycle composition were subsequently
> proven, but the generic opportunity screen produced **0/5** complete actions:
> structural graph observations lacked exact candidate-plan/critical-path
> bindings and the public task-proposal input was unavailable. It therefore
> diagnoses the missing evidence path rather than the product opportunity. See the
> [sample evidence](./benchmarks/results/poc-sticky-wrapper-longitudinal-sample-v1/README.md)
> [opportunity result](./benchmarks/results/sticky-wrapper-opportunity-gate-v1.json)
> and the [superseded v2 contract](./specs/poc-sticky-wrapper-longitudinal-v2.json).
> Neither is an input to the [fresh contract](./specs/poc-fresh-generic-optimization-v1.md).

> **Ordinary-build learning economics:** the POC now learns only from builds
> the user already requested. It predicts structural recurrence before paired
> calibration and requires projected payback within five compatible matches.
> Deterministic contract evidence stops a short-lived hypothesis after one
> requested build (avoiding 16 remaining observations) and a six-match payback
> after three (avoiding 14). A positive probe only permits continued learning;
> the unchanged eight-pair robust gate still decides qualification. This is a
> learning-cost control, not a new wall-time result. See the
> [ordinary-learning evidence](./benchmarks/results/poc-ordinary-learning-economics-v1/README.md).

> **POC onboarding north star:** install BuildOpt, open a Gradle repository and
> run `buildopt optimize build`. The command now has a stable state/result,
> resume, budget, exit and POC-authority contract. It now derives the exact Git
> change, Gradle-owned outputs and structural graph automatically for supported
> build-owned workflows and calibrates the candidate with eight balanced
> native/candidate pairs, exact outputs, full fallback and bounded break-even.
> A qualified candidate is stored as a private, digest-bound profile in a
> bounded structural portfolio. Dependency-source, resource-only, leaf-source
> and mixed-source families are derived from Gradle ownership and dependency
> facts rather than repository names; exact profiles resume without
> remeasurement and tampering is rebuilt only from still-valid evidence.
> The first positive result reports `LEARNING / QUALIFIED_PROFILE_STORED`.
> Repeating the exact command validates eleven revision, toolchain, workflow,
> graph, output, evidence and profile bindings before Gradle, then reports
> `QUALIFIED_AND_USED / QUALIFIED_PROFILE_SELECTED` and executes the smaller
> graph. The decision overhead is measured; any drift runs optimized native
> Gradle instead. Every completed command now writes a readable value report
> plus recomputable JSON covering graph reduction, measured mean/tail value,
> calibration cost, break-even, exact replays and fallback. One bounded Ktor
> public-history experiment now measures cross-commit profile lifetime. Its
> first run found that a matching replay saved 112.198 seconds, but an
> unrelated-owner fallback cost 220.761 seconds. The follow-up adds generic
> economic prequalification: on the same CORS change, two analogous commits
> were insufficient to justify the eight-build theoretical payback floor, so
> BuildOpt rejected discovery/calibration in 192 ms. The observed fallback
> penalty fell to 13.896 seconds while a matching replay still saved 100.744
> seconds. The 1,386.764-second qualification still projects 31 matching
> builds and did not repay in this three-build window. Useful lifetime remains
> profile-specific; it is never inferred from steady-state speedup alone. The
> five-repository lifetime baseline reinforces that boundary: four current
> calibrations qualify, but its original portability and eligibility rules
> produce zero selected replays. The recovery experiment then keeps Kafka's
> exact six-descendant window fixed and turns one verified local replay from
> -5.404 seconds to **+104.975 seconds / 71.14%**, with exact outputs and
> **+66.772 seconds cumulative net** after learning/publication. This proves
> cross-commit value on one public workflow. The unchanged breadth replication
> does not broaden that claim: Spring root `classes` fails calibration,
> OpenTelemetry JMX fails closed on mixed unowned-path ownership, and Spring
> JMS calibrates **11.43%** faster but is rejected because 14 native class
> outputs differ across roots. The next POC work targets those generic blockers
> through workflow-input ownership and native-volatility quarantine, while
> keeping exact bytes for every output BuildOpt transports.
> This authority exists only inside the explicit POC command
> and never grants production promotion. The ordered work and success
> scorecard live in the
> [one-command onboarding roadmap](./docs/plans/one-command-onboarding-roadmap.md).

## Get your first result

Install a published package; you do not need this source checkout or a Go
toolchain. On Linux or macOS:

```bash
curl --fail --silent --show-error --location \
  --output buildopt-install.sh \
  https://raw.githubusercontent.com/tonyredondo/buildopt/main/install.sh
bash buildopt-install.sh
export PATH="$HOME/.local/bin:$PATH"
buildopt doctor
```

Then open any Gradle repository that has its Wrapper:

```bash
cd /path/to/your-gradle-repository
buildopt gradle clean build
buildopt gradle clean build
```

BuildOpt discovers the Wrapper and packaged Gradle integration automatically.
The default path enables Gradle's native local Build Cache; the second clean
build can therefore restore compatible outputs without a plugin path, service,
credential, or `--build-cache` flag. BuildOpt's stricter Safe Cache remains an
explicit POC experiment because it has not demonstrated incremental build-time
value over that native baseline.

This two-command check validates installation and cache-compatible execution;
it does not by itself activate or prove the structural accelerator. The
separate `optimize` path below owns paired measurement, output equivalence,
payback qualification and exact native fallback before it may select anything.

The current one-command POC can also discover, calibrate and replay a verified
structural profile automatically:

```bash
buildopt optimize build
```

In GitHub Actions, the same path needs one BuildOpt-specific input:

```yaml
- uses: tonyredondo/buildopt@<40-character-commit-sha>
  with:
    command: optimize build
```

GitLab uses the same `command` input. Both integrations derive provider
revisions, restore only exact compatible state and upload `value-report.md`,
its recomputable JSON source and the exact machine result with checksums;
cache loss or drift runs optimized native Gradle and no BuildOpt service is
required. See the [CI integration guide](./docs/guides/ci-integration.md).

An optional cross-machine cache/state service is the current POC direction. Its
[storage contract](./specs/poc-central-storage-contract-v1.md) is executable,
and the [restart-safe local state store](./specs/poc-central-state-storage-v1.md)
now persists typed portfolios, evidence and checkpoints on the shared CAS with
independent SQLite visibility. The
[central HTTPS boundary](./specs/poc-central-https-auth-v1.md) now adds a real
TLS 1.3 listener plus independently scoped, live-revocable cache/state tokens.
The [central Gradle-cache proof](./specs/poc-central-gradle-cache-v1.md) now
routes native Gradle GET/PUT traffic through the credential-containing local
gateway and proves clean `FROM-CACHE` reuse plus outage fallback. Connection,
owner commit orchestration is still explicit for cache publication. The
[central state-sync proof](./specs/poc-central-state-sync-v1.md) adds one-time
`buildopt connect` plus exact online/offline synchronization for generated
portfolios, evidence and checkpoints. The
[central optimize integration](./specs/poc-central-optimize-integration-v1.md)
now makes that synchronization automatic around `buildopt optimize`: verified
remote profiles may cross ordinary source commits only after canonical
workflow, Wrapper, producer-lineage, output-contract and change-family
revalidation. Evidence ancestry and exact output revisions remain separate;
build-logic or producer drift requires native refresh or full native fallback.
The
[isolated two-machine proof](./specs/poc-central-two-machine-v1.md) now connects
that state path to the central Gradle cache automatically: a clean read-only
consumer selected the remote profile, restored one task `FROM-CACHE` after a
server restart and produced the exact producer JAR without receiving the
central credential. Service loss retained the verified profile and rebuilt the
same bytes with zero remote hits. The later equal-opportunity experiment
measured the complete central path at 82.45% lower wall time on Ktor and 56.41%
on Beam. The
[profile-lifetime result](./benchmarks/results/poc-profile-lifetime-v1/README.md)
then showed why those steady-state wins are not sufficient: observed matching
frequency and fallback cost must repay learning before BuildOpt should spend
on calibration.

For a non-invasive evaluation, the GitHub Action's `profile-proposal` mode
turns a checked-in workflow/output declaration and exact pull-request diff into
a downloadable review artifact. It never measures or activates the candidate;
uncertain analysis keeps optimized native Gradle. See the [CI integration
guide](./docs/guides/ci-integration.md#review-a-structural-proposal).
The POC also keeps a manual
[five-repository clean-CI replay](./specs/poc-generic-profile-ci-replay-v1.md)
that detects proposal drift without rerunning or inflating performance claims.
The same confirmed owner-input path has now been exercised unchanged across
packaging, typed verification, distribution, and build-owned test-preparation
workflows. Each candidate preserved its declared output bytes and omitted the
unaffected project; an executable task without supported structural semantics
stayed on native Gradle before timing. This is a capability result, not a new
performance claim.

The six reviewed Groovy and Kafka profiles have also been replayed from the
public `v0.3.1` package through `buildopt poc` in clean external checkouts.
All six selected the reviewed graph, all six exact manifest-drift probes fell
back to native Gradle, and candidate/native-fallback semantic outputs matched
in every cell. This transfers the already-qualified value to the public CLI;
it does not create a new timing percentage or automatic activation claim. See
the [installed replay evidence](./benchmarks/results/poc-installed-profile-replay-v1/README.md).

The remaining date-dependent output gap is also closed for future reviewed
evidence. Groovy's exact release-properties exception now declares only
`BuildDate` and `BuildTime`; 2/2 real-JAR date probes match, while changes to
undeclared `ImplementationVersion` or class payloads still fail closed. With
the four natural Kafka cross-capture matches, all six retained cells are
cross-date comparable without rewriting any historical qualification. See the
[cross-date evidence](./benchmarks/results/poc-cross-date-output-equivalence-v1/README.md).

The [product onboarding guide](./docs/getting-started/product-onboarding.md)
contains Windows installation, CI snippets, component ownership and the
recommended rollout order. Contributors who want the complete synthetic lab
can use the [source quickstart](./docs/getting-started/quickstart.md).

<details>
<summary>Historical research and earlier bounded experiments</summary>

The cache-compatible command is the zero-configuration starting point. The
measured accelerator is the qualified POC profile: after committing a reviewed
Build Impact manifest, generated graph and `buildopt-qualified-profile.json`,
run `buildopt poc --changes-file .buildopt-changes`. It reports the selected or
full graph, exact adapters and expected outputs before Gradle starts. Only
Build Impact plus the exact standard-`Jar` adapter can activate; unknown/global
changes and `BUILDOPT_BYPASS=1` restore native full-graph execution. On the pinned Spring
Framework workload, direct discovery saved 28.72%; the installed command still
saved 15.76% after package, launcher, manifest and graph-validation overhead,
with 8/8 positive pairs and identical declared outputs. See the
[Build Impact workflow](./docs/guides/product-workflows.md#build-impact).
An additional installed Spring matrix qualified `spring-webmvc` at 13.50%
faster, while a shared `spring-core` to `spring-jms` scope averaged 10.89%
faster but failed the preregistered 4/4-positive stability rule. BuildOpt
therefore keeps a bounded output-scope claim rather than generalizing to every
change.

The latest fresh-family check used Micronaut Core. A generic ownership fix now
separates each project's direct source roots from the larger conservative
boundary retained for cyclic dependencies. Discovery reduced the fixed
75-project `assemble` reach to 22 projects without a repository-name rule. In
eight alternating installed-path pairs, optimized native Gradle averaged
24.067 s and BuildOpt averaged 6.506 s: **17.562 s/72.97% faster**, with 8/8
positive pairs, three byte-identical JARs and full-graph fallback for a global
change. This qualifies only the fixed Micronaut structural scope, not every
repository or change.

That exact evidence can now be converted into a generic, review-required v4
profile with `buildopt profile qualify`. A fresh installed replay then used
only `buildopt poc --changes-file`: optimized native Gradle averaged 23.643 s
and the profile averaged 6.582 s, saving **17.061 s/72.16%** with 8/8 positive
pairs. Validation, planning and launcher overhead are included. Global changes
and even whitespace-only graph drift restored the full graph. No product rule
matches Micronaut or any repository name; new repositories remain native until
their own structure and evidence qualify.

`buildopt poc` is available in public `v0.3.1`. The installed-profile replay
above validates that exact release; earlier `v0.2.0` remains historical
onboarding evidence rather than the current qualified-profile package.

The checked scorecard measures each optimization separately and then measures
the complete public path without adding unrelated percentages. The tested
Runtime Tuning, Hot State, and standard-Copy mechanisms did not add defensible
incremental value over optimized native Gradle, so their activation code has
been removed from the POC. The final combined
path saved 63.5–84.1% across four Kotlin/Groovy synthetic workload cells, with
identical required outputs and zero product-attributable failures. The POC
decision is therefore `CONTINUE`, qualified only for those controlled workload
classes. The repeated realistic breadth gate retained that narrow claim after
4/8 cells qualified. A substantial Spring `testClasses` workload then saved
28.72% across 8/8 pairs. After task attribution and a conservative
standard-`Jar` cache adapter, the clean installed OpenTelemetry Spring-family
path saved 50.40% or 5,361.25 ms across 4/4 pairs, with a positive paired
interval, 125 identical outputs, retired Hot State absent, and safe full-graph
fallback. This is qualified POC value for that workload, not a universal-savings
or production-readiness claim. A separate unchanged Spring test workflow
rejected direct Test-fixture JAR reuse after it regressed by 11.31%, so that
diagnostic switch is not part of the recommended path. A subsequent three-arm
ablation narrowed plugin registration but still found the complete adapter
612.25 ms/9.53% slower than native; BuildOpt therefore keeps native Gradle for
that unqualified workflow instead of treating cache hits as value. The next
bounded Runtime Tuning hypothesis also failed: capping the same Spring
`testClasses` workload from 12 to 6 workers made it 191.5 ms/2.00% slower, with
only 2/4 favorable pairs and an interval crossing zero. BuildOpt therefore
retains the native 12-worker control and performs no further parameter search
for that trace. The controlled remote-cache experiment then isolated Edge
locality: the same eight committed Shared objects and Gradle HTTP client took
6,911.25 ms directly over an 80-ms/20-MiB/s modeled WAN and 4,510 ms through a
prewarmed loopback Edge. That is 2,401.25 ms/**34.74% faster**, with 4/4
positive pairs, identical 32-MiB outputs and zero measured upstream Edge
requests. This qualifies Edge locality only under that frozen profile; it does
not claim that Shared alone outperforms another remote-cache origin. Finally,
the unchanged clean profile transferred to Apache Kafka 4.3.1: native root
`testClasses` averaged 4,609.5 ms and installed BuildOpt 2,070 ms, saving
2,539.5 ms/**55.09%** with 4/4 positive pairs, 4,062 byte-identical outputs and
full 64-project fallback. This broadens the POC evidence to a Java/Scala and
generated-source workload without adding Kafka-specific product logic; it is
still not a universal claim. See the [POC value
contract](./specs/poc-value-validation-v1.md) and
[raw scorecard](./benchmarks/README.md#build-optimization-scorecard).

The same repository-owned profile has also been replayed through an installed
package on those fixed OpenTelemetry and Kafka revisions. Both candidates
reproduced their historical outputs; OpenTelemetry restored its standard JAR
and Kafka restored `:generator:jar`. Global changes completed the native full
graph. This is usability and fallback evidence only, so the earlier
percentages remain unchanged.

Fresh packaging evidence now extends the Kafka claim beyond test preparation.
For the fixed `Metadata.java` change, native root `assemble` averaged 8,054 ms
and installed `buildopt poc` averaged 3,416.5 ms, saving 4,637.5 ms/**57.58%**.
All 4/4 pairs were positive, the smallest saving was 4,050 ms, the 10.2-MB
client JAR was byte-identical, and the global fallback passed. This qualifies
only the declared Kafka client-packaging scope. A subsequent composition seed
proved that the required shaded artifact is produced by `:clients:shadowJar`
while `:clients:jar` is skipped. It stopped before any timing, so the 57.58%
result is Build Impact evidence and is not attributed to the standard-`Jar`
adapter. The first Build Impact + Edge composition produced a strong diagnostic
signal but remained unqualified because forced Edge failure rebuilt custom
`shadowJar` with different ZIP metadata. After qualifying reproducible archive
settings independently, a fresh preregistered composition used those settings
equally in both arms. Native full `assemble` through Shared averaged
42,992.75 ms; installed Build Impact through prewarmed Edge averaged
7,587.25 ms, saving 35,405.5 ms/**82.35%** with 4/4 positive pairs and interval
+30,162..+42,487.75 ms. Every arm produced exact `3ffd994e...3349` bytes,
global changes restored the full graph, and HTTP 503 disabled Edge and rebuilt
the same output locally. This qualifies the combined POC only for the fixed
Kafka change and modeled network profile.

The same composition is now available through the repository-owned v2 POC
profile instead of experiment-only cache variables:

```bash
buildopt poc --changes-file .buildopt-changes \
  --edge-url http://127.0.0.1:<PORT>
```

The plan exposes the exact normalized `build.gradle` SHA-256 and read-only
Edge endpoint. Global/unknown changes, precondition drift, missing/invalid
Edge, and bypass select the native full graph without Edge. HTTP 503 executes
the selected graph locally and preserves the exact output. This is usability
evidence; it does not widen or recompute the 82.35% result.

The profile can now be reproduced by the read-only
`buildopt profile discover` command from the checked matrix evidence, Build
Impact manifest/graph/generated state, trace digests, and reviewed profile
contract. The generated Kafka profile is exactly equal to the reviewed v2
profile. Spring and OpenTelemetry emit `NATIVE_FULL_GRAPH`, as do evidence
drift, incomplete graphs, unknown relationships, selected Test tasks, and
precondition drift. Discovery never writes or activates a profile.

The subsequent trace-gated decision found no new generic optimization worth
implementing. Across the retained installed synthetic and Spring traces,
BuildOpt-specific setup peaks at 1.238233 ms, startup at 364.875 ms,
finalization at 97 ms, and teardown at 87 ms. Configuration reaches 682 ms in
Spring but is neither causally attributed to BuildOpt nor reproduced above the
500-ms threshold in a second workload family. The checked result is therefore
`NO_ACTIONABLE_HYPOTHESIS`; no new timing or mechanism activation follows.

The earlier three-family portfolio decision was
`SPECIALIZE_BOUNDED_KAFKA_PROFILE`: only Kafka qualified in that installed
matrix. That decision remains scoped to its exact compositions. The later,
materially different structural-only method independently qualifies
OpenTelemetry, Kafka, Micronaut, Groovy and Hibernate without repository-name
rules, while Spring and every unqualified change remain on optimized native
Gradle. This is bounded POC evidence, not a production, automatic or universal
value claim.

Before proposing that structural experiment, `buildopt profile propose` now
executes the owner workflow once and validates its required outputs against
Gradle-declared task outputs. A missing, empty, symlinked or ambiguously owned
declaration retains native Gradle and writes a review artifact with concrete
candidates; it cannot enter warm-up or timing. `buildopt profile outputs`
exposes the same preflight independently. The frozen Hibernate proof caught
its original `build/libs` assumption and reported the real `target/libs` JARs
without adding repository-specific product logic.

The reviewed semantics now live in one `.buildopt/profile.json` generated by
`buildopt profile input --confirm`. It records the workflow, exact
`GIT_DIFF_BASE_TO_HEAD` change source, confirmed Gradle-owned outputs, fallback
scope, and output-contract digest. Local `profile propose` and the read-only
GitHub Action consume the same file, bind its SHA-256, and rerun the output
preflight on every target. Output drift therefore produces a concrete
`NATIVE_FULL_GRAPH` diagnostic rather than a stale or silently activated
candidate.

One owner-input schema is also sufficient for Gradle-owned `Jar`,
`VerificationTask`, `AbstractArchiveTask`, and `testClasses` workflows. The
checked breadth fixture derives the exact `service-a` task for each family,
rebuilds the repository-declared output byte for byte, executes no `Test`
task, and keeps an arbitrary executable task on
`NATIVE_FULL_GRAPH / ORIGINAL_WORKFLOW_UNSUPPORTED`. See the
[workflow-breadth contract](./specs/poc-generic-workflow-breadth-v1.md).

A new generalization foundation now separates structural opportunity from
activation. `buildopt profile analyze` detects a complete smaller graph without
matching repository names and emits a measurement proposal, never a predicted
speedup. The checked whole-profile scorecard then evaluates every mechanism
supported by exact evidence for each target: Spring Build Impact was **30.86%**
faster, clean OpenTelemetry Impact + standard `Jar` was **50.40%** faster, and
Kafka Impact + read-only Edge was **82.35%** faster. These are direct independent
compositions, not added percentages. Only Kafka passed that composition's
later strict installed replication; the newer structural-only qualifications
remain separate evidence. See the [general build-value
contract](./specs/poc-general-build-value-v1.md).

`buildopt profile measure` now performs the missing generic experiment step.
It checks one exact clean Git change, creates independent native and BuildOpt
clones and Gradle homes, alternates eight pairs, compares repository-declared
outputs byte for byte and proves full-graph fallback. Its evidence feeds
`buildopt profile evaluate`; neither command activates a profile automatically.
The deterministic conformance fixture adds no performance percentage.

The preceding setup is now executable as well. `buildopt profile propose`
starts from one original Gradle selector, the exact base-to-HEAD change and
owner-declared output globs. It performs two read-only Gradle discovery passes,
generates the manifest/graph/fallback bundle and emits the measurement handoff
without repository-name rules or hand-authored JSON. Unsupported, ambiguous,
global or Test-bearing workflows retain native full graph; no profile is
written or activated.

That same installed command has also been replayed from fresh Apache Groovy and
Micronaut Core checkouts with no retained BuildOpt JSON. It reproduced the
qualified 37-to-2 and 75-to-22 project boundaries exactly. No timing was
repeated because the candidates did not change materially; the existing 50.06%
and 72.16% measurements remain the associated value evidence. See the
[public-repository replay](./benchmarks/results/poc-generic-profile-realworld-v1/README.md).

That generic path now has fresh value evidence on Apache Groovy 5.0.8. For one
`groovy-json` source change, discovery reduced the checked `classes` graph from
37 projects to two. Across eight isolated alternating pairs, optimized native
Gradle averaged 92.351 s and installed BuildOpt averaged 46.120 s: **46.231 s
or 50.06% faster**, with 8/8 positive pairs, 66 byte-identical class outputs
and successful full-graph fallback for a global change. Distribution was
rejected on output mismatch and aggregate `assemble` was rejected as the wrong
semantic scope before this candidate was measured. The qualified result remains
bound to the exact classes output; it is not a claim for every Groovy build.

The same structural-only protocol has now been rerun uniformly across five
substantial public repositories. The terminal v3 matrix qualifies Kafka at
**84.11%**, Micronaut at **41.74%**, and Groovy at **73.85%** faster than their
declared optimized-native workflows. Spring is **17.94%** faster with a
positive interval, but one -260-ms pair retains native under the frozen 8-of-8
gate. A separately preregistered OpenTelemetry-only v4 correction preserves
the measured scheduling in the untimed fallback and qualifies at **14.43%**
faster, 12.110 s saved, 8/8 positive pairs, exact 125-file outputs, and a
successful full-graph fallback. This is the strongest current evidence that
generic graph reduction can create large cascade value while still failing
closed when any end-to-end gate is not met. See the [terminal v3 matrix](./benchmarks/results/poc-generic-profile-matrix-v3/README.md)
and [OpenTelemetry v4 correction](./benchmarks/results/poc-generic-profile-matrix-v4/README.md).

Once that exact scope has independently beaten optimized native Gradle,
`buildopt profile qualify` can turn the digest-bound evidence into a reviewable
Build-Impact-only profile. The command is repository-independent: it validates
the measured plan, timings, outputs, fallback and manifest/graph/generated
hashes, then emits JSON for repository review. `buildopt poc` rechecks those
bindings before every run and retains the native full graph on drift or
uncertainty. See the [structural profile contract](./specs/poc-structural-profile-v1.md).

</details>

## Choose what to do next

| Goal | Start here |
|---|---|
| Install and run BuildOpt | [Product onboarding](./docs/getting-started/product-onboarding.md) |
| Run the source-based POC lab | [Quickstart](./docs/getting-started/quickstart.md) |
| Add it to GitHub Actions or GitLab CI | [CI integration](./docs/guides/ci-integration.md) |
| Understand the design | [Architecture overview](./docs/architecture/overview.md) |
| Review the POC idea, current value, and next priorities | [Current POC one-pager](./docs/findings/buildopt-poc-handoff.md) |
| Make a code change | [Developer onboarding](./docs/getting-started/developer-onboarding.md) |
| Find a command or setting | [CLI](./docs/reference/cli.md) and [configuration](./docs/reference/configuration.md) references |
| Diagnose a failure | [Troubleshooting](./docs/troubleshooting.md) |

## What BuildOpt changes

| Capability | What it does | Safe fallback |
|---|---|---|
| Launcher | Runs the original argv without a shell and preserves its exit code | Original command |
| Managed L1 and Shared Cache | Reuses only authenticated, verified, committed Gradle outputs | Cache miss and normal execution |
| Task Intelligence | Qualifies only tasks with sufficient exact evidence | No publication or optimization |
| Patch Autopilot | Creates exact, signed, reviewable draft changes and exact revert bundles | Repository remains unchanged |
| Build Impact | Chooses only repository-authorized Gradle entrypoint alternatives | Full original graph |
| Build history | Exposes redacted immutable sessions through a loopback API and embedded dashboard | No history endpoint |
| Edge Cache | Provides an optional nearby cache while Shared remains commit authority | Shared or ordinary miss |

Runtime Tuning, exact-bound Hot State, and the standard-Copy adapter are
retired experiments. Their protocols and results remain under `specs/` and
`benchmarks/results/` so the negative decisions stay auditable, but no CLI,
launcher, plugin, or workflow can activate them.

Set `BUILDOPT_BYPASS=1` to remove the optimization path immediately while
preserving the original command and process behavior:

```bash
BUILDOPT_BYPASS=1 buildopt gradle build
```

## Documentation

Start at the [documentation portal](./docs/README.md). Common routes are:

- [Quickstart](./docs/getting-started/quickstart.md) — obtain a first result.
- [Developer onboarding](./docs/getting-started/developer-onboarding.md) — set
  up a reproducible development environment and make a first change.
- [Architecture](./docs/architecture/overview.md) — components, data flow,
  trust boundaries, and failure behavior.
- [Repository map](./docs/architecture/repository-map.md) — the architecture's
  exact correspondence with folders and artifacts.
- [Product workflows](./docs/guides/product-workflows.md) — launcher, profile
  evaluation, Patch Autopilot, Build Impact, and Edge.
- [CI integration](./docs/guides/ci-integration.md) — GitHub Actions and GitLab
  CI.
- [CLI reference](./docs/reference/cli.md) and
  [configuration reference](./docs/reference/configuration.md).
- [Validation guide](./docs/reference/validation.md) — choose the smallest
  useful proof instead of running every check.
- [Troubleshooting](./docs/troubleshooting.md) and
  [glossary](./docs/glossary.md).

Operator recovery procedures live in [runbooks](./runbooks/README.md).
Executable cross-component behavior lives in [specifications](./specs/README.md).

## Supported environments

| Surface | Linux | macOS | Windows |
|---|---:|---:|---:|
| Native launcher and Build Impact | Yes | Yes | Yes |
| Persistent gateway and managed L1 | Yes | Yes | Yes |
| Server and Edge storage | Yes | Yes | Yes |
| Process-tree cancellation | Process group | Process group | Job Object |
| Background services | systemd | launchd user agent | Windows SCM |
| Source bootstrap and full development lane | Primary | Native CI/package lane | Native CI/package lane |

The Gradle compatibility target and evidence levels are defined in the
[capability matrix](./specs/capability-matrix-v1.md). Run `buildopt doctor` on
an installed binary to see exact platform capabilities as JSON.

## Build and contribute

Repository-owned toolchains avoid dependence on global Go, Java, or lint
versions:

```bash
./dev/doctor
./dev/bootstrap --toolchain temurin-jdk-21
./dev/bootstrap --toolchain go
./dev/run --toolchain go -- go test ./internal/launcher ./cmd/buildopt
./dev/run -- ./gradlew --no-daemon check
```

Read [CONTRIBUTING.md](./CONTRIBUTING.md) before changing contracts, generated
code, tests, or product boundaries. Documentation is validated with:

```bash
./dev/check-documentation
```

## Sources of truth

Use this precedence when documents answer different questions:

1. [Master RFC](./gradle-build-optimization-platform.md) — product intent,
   safety invariants, and accepted decisions.
2. `contracts/` — normative wire and document formats.
3. `specs/` and `adr/` — executable behavior and architectural decisions.
4. [Implementation tracker](./implementation-tracker.md) — current status and
   evidence.
5. `docs/`, component READMEs, and runbooks — explanatory and operational
   guidance.

An example in documentation never overrides an executable contract. If the
implementation, a guide, and a normative contract disagree, stop and reconcile
the contract and its consumers before activating the behavior.
