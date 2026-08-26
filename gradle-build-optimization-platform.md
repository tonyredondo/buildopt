# Gradle Build Optimization Platform

## Product specification and technical design

**Status:** master POC architecture RFC — generic structural value qualifies across public Gradle families; one-command onboarding is the next product-direction objective<br>
**Last technical review:** August 16, 2026<br>
**Working name:** Gradle Build Optimization<br>
**Scope:** autonomous optimization of Gradle builds in CI and local environments<br>
**Relationship with Test Optimization:** complementary product with explicit ownership: Build Optimization optimizes build work; Test Optimization retains test selection, execution, and policy

---

## Table of contents

1. [Executive summary](#1-executive-summary)
2. [Problem statement](#2-problem-statement)
3. [Objectives](#3-objectives)
4. [Scope and non-goals](#4-scope-and-non-goals)
5. [Design principles](#5-design-principles)
6. [Conceptual acceleration model](#6-conceptual-acceleration-model)
7. [High-level architecture](#7-high-level-architecture)
8. [Evidence model](#8-evidence-model)
9. [Functional pillars](#9-functional-pillars)
10. [Optimization catalog](#10-optimization-catalog)
11. [Learning without duplicate manual builds](#11-learning-without-duplicate-manual-builds)
12. [Shadow mode, canaries, and rollout](#12-shadow-mode-canaries-and-rollout)
13. [Contractual task qualification](#13-contractual-task-qualification)
14. [Resource autotuning](#14-resource-autotuning)
15. [Build Impact Analysis](#15-build-impact-analysis)
16. [Configuration Cache optimization](#16-configuration-cache-optimization)
17. [Automatic patch validation](#17-automatic-patch-validation)
18. [CI and local behavior](#18-ci-and-local-behavior)
19. [Product modes](#19-product-modes)
20. [Security, privacy, and trust](#20-security-privacy-and-trust)
21. [Performance and cost budget](#21-performance-and-cost-budget)
22. [Success metrics](#22-success-metrics)
23. [User experience](#23-user-experience)
24. [Implementation roadmap](#24-implementation-roadmap)
25. [Recommended MVP](#25-recommended-mvp)
26. [Example repository evolution](#26-example-repository-evolution)
27. [Risks and mitigations](#27-risks-and-mitigations)
28. [Product decisions](#28-product-decisions)
29. [Implementation readiness](#29-implementation-readiness)
30. [Official technical references](#30-official-technical-references)
31. [Conclusion](#31-conclusion)

---

## 1. Executive summary

Gradle Build Optimization will be a platform that observes build execution, avoids redundant work through several cache layers, applies optimizations directly, and learns from natural CI executions to activate additional improvements progressively and safely.

It will not be merely an observability tool or recommendation dashboard. Observability will be the control system that lets the product make decisions, prove the benefit of every action, and automatically revert regressions.

The primary success criterion will be net customer-visible time saved and observed causally across the complete build session; cache hits, avoided tasks, and active actions explain the outcome rather than replace that metric.

The operating loop will be:

```text
Observe → model → select an action → validate → activate → measure → learn
```

The product promise will be:

> We intercept the build, apply the fastest validated policy within the supported matrix, and revalidate every action whenever a contract, compatibility, or trust-domain boundary changes.

The solution combines four pillars:

1. **Build observability:** task graph, critical path, resource utilization, cache-miss causes, project dependencies, and historical evolution.
2. **Cache acceleration:** remote build cache, local build cache, dependency cache, Configuration Cache, and specialized caches that are part of the build.
3. **Autonomous optimization engine:** automatic configuration, autotuning, contractual task qualification, Build Impact Analysis, and validated build-logic and CI changes.
4. **Validation and guardrails:** shadow mode, learning from natural builds, canaries, artifact comparison, fallback, and rollback.

Developers will not have to run a build twice manually. Natural builds provide observations; any additional repetitions required for validation run automatically in CI within a budget and isolated environments. Local environments consume already validated policies but create their own Configuration Cache entries.

The current objective is a **proof of concept**, not a private-beta or production launch. The combined BuildOpt path has demonstrated customer-visible build-time reduction against a well-configured native Gradle baseline across the qualified synthetic Kotlin/Groovy workload matrix. Safe Cache, Runtime Tuning, Build Impact, reviewed task contracts, and Patch Autopilot are measured separately so that value is attributable; the complete path receives its own comparison because overlapping percentages are never added.

The successor POC onboarding north star is one repository-committed wrapper
and one repeated command:

```text
generate and commit BuildOpt Wrapper -> ./buildoptw build
```

The generated `buildoptw`, `buildoptw.bat`, pinned wrapper properties and
portable non-secret configuration remain in the customer repository. They
bootstrap one checksum-verified BuildOpt distribution and invoke the existing
Gradle Wrapper. The command may run native Gradle, observe, shadow, schedule a
bounded isolated trial, execute an exact qualified runtime profile or report a
reviewed durable Gradle patch. Gradle-cache objects and typed BuildOpt decision
state share an optional owner-operated HTTPS service but never share authority,
metadata or protocol. Missing, expired, corrupt or incompatible state retains
optimized native Gradle. Credentials are never committed or passed to Gradle.

This is a new value experiment, not a reinterpretation of the stopped generic
profile or adaptive-fragment results. The wrapper is onboarding and control
infrastructure; it succeeds as an accelerator only if its complete
chronological portfolio, including observation, trial, cache, fallback and
wrapper costs, beats the same optimized native Gradle cache opportunity across
the frozen breadth gate. The ordered work and immutable scorecard are in the
[Sticky Wrapper Learning POC Tracker](./docs/plans/sticky-wrapper-learning-poc-tracker.md).
The earlier [one-command onboarding roadmap](./docs/plans/one-command-onboarding-roadmap.md)
and [central cache/state roadmap](./docs/plans/centralized-cache-and-state-roadmap.md)
remain implemented foundations, not current value authority.

`SWL-001` freezes the repository boundary before implementation. Both wrapper
scripts use UTF-8/LF; only the POSIX script has the Git executable bit.
`wrapper.properties` is an ordered ASCII `key=value` grammar with one exact
version, immutable HTTPS URL and lowercase SHA-256 per supported platform,
5-second connect/30-second read timeouts, at most five HTTPS-only download
redirects and environment-only proxy discovery. The checksum remains the
archive authority. `config.toml` is a strict flat subset with
mode, optional HTTPS server identity, private credential-environment name and
a 0..5% trial budget; it contains no credential or machine path. All ordinary
arguments remain Gradle arguments. Only a leading `--buildopt` routes to
management, while leading `--gradle` escapes that prefix.
`BUILDOPT_BYPASS=1` is checked before configuration or download and invokes the
repository Gradle Wrapper directly. The exact contract is
[`poc-sticky-wrapper-contract-v1`](./specs/poc-sticky-wrapper-contract-v1.md);
the implemented generator is specified by
[`poc-sticky-wrapper-generator-v1`](./specs/poc-sticky-wrapper-generator-v1.md),
and verified bootstrap by
[`poc-sticky-wrapper-bootstrap-v1`](./specs/poc-sticky-wrapper-bootstrap-v1.md).

`SWL-002` implements `buildopt wrapper init`, offline/read-only `check` and
distribution-only `update`. The generator resolves a stable public GitHub
release into four immutable asset URLs and GitHub-provided SHA-256 digests but
does not download an archive. Identical inputs produce identical bytes; init
refuses any target before metadata access; update requires canonical current
state, preserves scripts/configuration, performs no same-version write and
requires explicit downgrade authority. A repository transaction stages and
flushes all files, preserves prior bytes and restores the complete old state
after any publication failure. Bootstrap, Gradle passthrough and performance
authority remain false in the generator block.

`SWL-003` embeds thin POSIX and Windows bootstrap scripts. They select one of
the four pinned native packages, follow at most five HTTPS-only redirects,
verify the archive SHA-256 before extraction, reject unsafe archive entries,
verify the internal manifest and publish one user-cache directory atomically.
Warm use re-verifies the complete entry without network access and concurrent
first use performs one download. The public GitHub smoke corrected the initial
zero-redirect assumption because release assets may return a `302`; the pinned
checksum remains the final content authority. Only `--buildopt version` is
enabled here. Ordinary Gradle passthrough remains `SWL-004`, so this block adds
no build-time claim.

The result authorizes a `CONTINUE` decision for further POC exploration only. It does not prove universal savings or production readiness. The initial realistic change-class matrix in [`poc-breadth-validation-v1`](./specs/poc-breadth-validation-v1.md) qualified 2/8 cells. Attribution and calibrated paired experiments reproduced the bounded Groovy and leaf Kotlin value cells, while shared-source and build-logic Kotlin remained order-sensitive. The terminal decision therefore retained the qualified synthetic claim and prohibited more unchanged replication or product tuning against noisy evidence.

The bounded signal was then tested on fixed public repositories. Compatibility was established before timing: source revisions, wrapper/settings files, Gradle distributions, representative tasks, excluded integrations, required outputs, and the installed BuildOpt entry point were pinned and checked. Spotless, Mockito, and SpotBugs entered a preregistered paired comparison against optimized native Gradle. Only Mockito retained no-change parity and cleared the unchanged leaf-source accelerator threshold; Spotless and SpotBugs did not. The terminal result therefore retains the qualified synthetic claim and does not authorize a general public-repository claim, new tuning, or threshold movement. An eight-hour soak, design partners, high availability, enterprise identity, shared multi-tenancy, and production promotion samples are not required. A feature does not justify activation merely because it is safe or technically interesting: when it cannot demonstrate net value for a workload class, BuildOpt keeps it disabled for that class.

A subsequent diagnostic profiled the exact upstream workflows once on the same strict runner without making a performance comparison. Spotless took 165.173 seconds, of which startup plus configuration was only 1.55%; Mockito took 629.165 seconds with 0.47% in those phases; SpotBugs took 271.920 seconds and its main test task alone occupied 242.120 seconds. These facts reject invocation fusion and Configuration Cache repair as primary levers for the measured workflows. They do not place test-source compilation or the other artifacts required by tests outside Build Optimization: Mockito's 593.290-second `build` invocation spent 242.690 seconds in `:mockito-core:compileTestJava`, a Tier 1 `JavaCompile` task over 402 test sources. The raw diagnostic remains unchanged, while the corrected decision authorizes two separately preregistered experiments: change-aware Build Impact on Spotless's exact workflow, including `testClasses`, and Safe Cache value for Mockito's `:mockito-core:testClasses` against an optimized native Gradle cache. Parity alone is not value for the latter, and only a qualifying mechanism may advance to the exact workflow with every requested test preserved. SpotBugs receives no follow-up because its build-owned `compileTestJava` occupied only 1.119 seconds while the already-scoped `Test` task occupied 242.120 seconds. No diagnostic figure is a savings claim; both experiments retain the unchanged value gate and byte-identical required outputs.

---

## 2. Problem statement

Gradle builds lose time through a combination of causes:

- Ephemeral runners start without previous outputs.
- Remote cache is missing or misconfigured.
- Custom tasks fail to declare inputs and outputs correctly.
- Generators are nondeterministic.
- Configuration Cache is disabled or blocked by incompatible build logic.
- Builds run `clean` unnecessarily.
- Multiple Gradle invocations repeat initialization and configuration.
- Parallelism, heap, and workers are configured statically.
- Dependencies resolve prematurely or inefficiently.
- `api` dependencies that could be `implementation` cause transitive recompilation.
- Annotation processors disable incremental compilation.
- Eager APIs are used instead of lazy configuration.
- Pipelines request every project even when a change affects only part of the build.
- There is no safe way to test and activate optimizations automatically.

Gradle already includes powerful primitives, but using them correctly requires specialized knowledge, discipline across every plugin, and continuous maintenance. The opportunity is not to replace Gradle, but to turn those primitives into a closed autonomous system.

---

## 3. Objectives

### 3.1 Primary objective

Primarily reduce build time while keeping infrastructure consumption within the customer's explicit budget, without changing observable behavior, required artifacts, or pipeline correctness guarantees.

The product north star is **net build time saved and observed causally**, together with its reduction percentage. The neutral measurement envelope measures from the moment it hands the session to the assigned arm—candidate Launcher or native control Gradle command—until the result and deliverables are available. It includes all Launcher, plugin, gateway, policy, and cache-materialization overhead. Regressions subtract from savings; they are not truncated to zero.

Hit rate, action count, and avoided tasks are diagnostic drivers, not success by themselves. An optimization counts as primary success only when it reduces customer-visible time in a valid comparison while preserving correctness, `customerVisibleFeedbackMs`, queue behavior, and p95/p99. Economic value is reported separately and may limit rollout according to the customer budget; saving cost without reducing time does not increase the north star. Modeled savings are labeled `ESTIMATED` and never mixed with `OBSERVED_CAUSAL`.

### 3.2 Functional objectives

- Automatically avoid work already completed.
- Execute only the build subgraph authorized by the deliverables manifest and declared graph.
- Find and activate the most efficient resource configuration for each runner and build type.
- Make tasks cache-reusable only when they independently satisfy four properties: a complete input/output contract, repeatable outputs, relocatability, and absence or contractual mediation of external state.
- Correct build logic that prevents incrementality, cacheability, parallelism, or Configuration Cache.
- Learn from normal builds without requiring developers to repeat them manually.
- Apply optimizations with progressive confidence and automatic rollback.
- Show which actions were taken, how much they saved, and the evidence used to activate them.
- Allow all observability, evidence, and the action ledger to be exported through open formats and a versioned schema.

### 3.3 Non-functional objectives

- Security against cache poisoning and untrusted builds.
- Safe degraded operation when the remote service is unavailable.
- Versioned compatibility across Gradle, JDK, and plugin versions.
- Low instrumentation overhead.
- Reproducible, auditable decisions.
- Explicit control over telemetry transmission and repository-data handling.

---

## 4. Scope and non-goals

### 4.1 In scope

- Gradle initialization, configuration, and execution phases.
- JVM compilation and, through adapters, other Gradle-supported toolchains.
- Code generation.
- Resource processing.
- Dependency resolution.
- Packaging JAR, ZIP, TAR, WAR, and other artifacts.
- Custom tasks and tasks supplied by plugins.
- Multi-project builds and monorepos.
- Gradle-related CI configuration.
- Remote build cache and related caches.
- CPU, memory, worker, and compiler-process optimization.
- Automatic validated changes to build logic.

### 4.2 Initially out of scope

- Test Impact Analysis.
- Predictive test selection.
- Test-specific sharding or distribution.
- Flaky-test management.
- Autonomous modification of releases, deployments, signing, migrations, or tasks with external effects.
- Replacement of Gradle's internal scheduler.
- Application-architecture changes not directly related to the build.

Test Optimization remains responsible for deciding which tests run. Build Optimization reduces the cost of producing the classes, resources, and artifacts those tests need.

### 4.3 Contract with Test Optimization

**Accepted decision — INT-001:** a Gradle invocation may contain build work and tests, but each product retains one owner per action:

- Build Optimization owns cache transport and backend, compilation, generation, resource, and packaging tasks, and Gradle configuration optimizations.
- Test Optimization owns selection, ordering, sharding, retries, and policy for Gradle `Test` tasks, as well as test reports.
- Build Optimization does not autonomously qualify `Test` tasks, modify their graph, or use Build Impact Analysis to omit them.
- Enabling Build Cache in a mixed invocation does not implicitly authorize caching `Test` tasks. Before configuration, Test Optimization must issue a signed, versioned grant identifying permitted types/adapters, namespace, and policy. Without that grant, Build Optimization configures `doNotCacheIf` for `Test` and test types registered by the integration contract; the grant digest, or digest of its absence, is a configuration input. The rule covers root, `buildSrc`, composite/included, and plugin builds; if the adapter for a combination cannot apply it before configuring any of them, it disables Build Cache for the entire invocation.
- The Shared Cache Backend may physically store test outputs, but Test Optimization decides eligibility, reads, writes, and invalidation. Because Gradle configures one remote cache per invocation, a dedicated test invocation may use a test namespace; a mixed invocation uses a shared namespace only when both policies approve it, otherwise `Test` task caching remains disabled. Build Optimization does not force that decision.
- A canary for a build patch that may alter classes, resources, or artifacts consumed by tests requests `FULL_RELEVANT_VALIDATION` from Test Optimization. If that mode is unavailable or fails, the patch is not promoted.
- The ledger attributes every saving and regression to the product that owns the action. Time avoided for the same task is not counted twice.
- Integration uses a versioned event-and-decision contract, not mutual access to internal state.

Minimum contract:

```text
BuildValidationRequest {
  contractVersion, requestId, actionId, repository, revision,
  candidateArtifactRefs[], controlArtifactRefs[],
  changedBuildInputs[], requestedMode = FULL_RELEVANT_VALIDATION,
  deadline
}

TestValidationResult {
  contractVersion, actionId,
  status = PASSED | FAILED | INCONCLUSIVE,
  artifactSetDigest, policyDigest,
  evidenceRef, completedAt, expiresAt, signature
}

TestCacheGrant {
  contractVersion, grantId, grantEpoch,
  repository, revisionOrPolicyRange,
  allowedTaskTypesOrAdapters[], namespace,
  read, write, issuedAt, expiresAt,
  policyDigest, signature
}
```

Test Optimization resolves internally which tests constitute `FULL_RELEVANT_VALIDATION`; Build Optimization cannot replace that result with its own heuristic. `TestCacheGrant` is fail-closed and must exist before Gradle configures tasks. A timeout, incompatible version, missing grant, or `INCONCLUSIVE` blocks the corresponding action, but the normal build preserves the baseline and does not fail because of that optimization.

#### 4.3.1 Deployable contract with Test Optimization

**Accepted decision — TESTOPT-API-001:** the private beta implements `INT-001` as a REST/JSON API over HTTPS described by OpenAPI 3.1. There is no direct database access or inbound callback into Build Optimization.

- Every request carries `contractVersion`, `requestId`, repository identity, revision/source state, and deadline; validation requests also carry `actionId`. `requestId`—and `actionId` for validation—form the idempotency key: repeating an identical request returns the same operation; reuse with another payload is rejected.
- In beta, the client authenticates with an opaque integration-scoped token, and Test Optimization signs `TestCacheGrant` and `TestValidationResult` with an Ed25519 key whose trust root is configured explicitly in Build Optimization. TLS protects transport; the signature binds the result to repository, action, revision, policy, artifact digests, and expiration.
- `POST /v1/test-cache-grants:resolve` returns a signed grant or denial before Gradle configuration. The grant includes `grantId` and `grantEpoch`; before authorizing a commit, Build Optimization queries `GET /v1/test-cache-grants/{grantId}/status` and requires a current signed status with an equal or greater epoch. A revoked, expired, unrefreshable, or digest-mismatched grant aborts pending work even when valid at the start.
- `POST /v1/build-validations` returns the final result or `202` with an `operationId`; the beta polls `GET /v1/build-validations/{operationId}` with backoff and jitter until the deadline. It does not implement webhooks. A timeout, non-recoverable error, incompatible version, or incomplete result produces `INCONCLUSIVE`.
- Retries apply only to idempotent operations and respect the original deadline. Errors use stable codes, state whether they are retryable, and never turn missing validation into `PASSED`.
- Artifact references are content-addressed, include size and SHA-256, bind to `actionId`/repository, and resolve through a customer-owned channel or authorized ephemeral URL. Test Optimization verifies the digest before consumption and rejects arbitrary caller paths.
- Phase 0 delivers producer/consumer fixtures for missing/expired/revoked grants, delayed results, corrupt artifacts, retries, and N/N-1 incompatibility. Both products run those fixtures in CI before declaring `FULL_RELEVANT_VALIDATION` available.

### 4.4 POC boundary

The POC validates technical feasibility and measurable value on project-owned synthetic repositories. It is not a private-beta qualification and must not reuse a private-beta gate name for weaker evidence without the `POC` suffix.

The POC is successful only when it demonstrates end to end:

- a native Gradle control with every applicable first-party optimization enabled;
- a BuildOpt candidate using the same Gradle, JDK, workload, warm state, outputs, and resource boundary;
- paired net build-time savings from the complete candidate, not a sum of mechanism estimates;
- zero required-output divergence and zero additional product-attributable build failures;
- attributable results for Safe Cache, Runtime Tuning, Build Impact, reviewed task contracts, and Patch Autopilot;
- `NO_VALUE_NO_ACTION`: an optimization stays observing or disabled when its expected net value does not clear the POC threshold.

Four alternating pairs are acceptable for bounded mechanism exploration. They are labeled `PRELIMINARY` and do not close a causal product gate. The combined value gate requires at least eight alternating pairs in each of two representative workload classes, Kotlin and Groovy coverage, a positive 95% lower bound, and point-estimate savings of at least `max(500 ms, 2%)`.

The reviewed source-contract route may validate Task Intelligence while Agent discovery and hermetic producer enforcement remain `UNAVAILABLE`. Safe fallback is evidence of correctness, not evidence that an unavailable capability exists. Safe Cache may qualify as a safety enabler at native-cache parity, but parity is not advertised as acceleration.

The POC explicitly excludes the eight-hour soak, external design-partner operation, production promotion samples, contractual SLOs, HA/RPO/RTO, enterprise identity, shared multi-tenancy, and production support. Passing the POC creates a decision point about whether productization is worth funding; it does not silently start productization.

---

## 5. Design principles

### 5.1 Act, do not merely recommend

The preferred outcome is an applied, validated optimization, not a recommendation the customer must interpret and implement manually.

### 5.2 Observability is part of the control plane

Telemetry is used to:

- Select the next action.
- Estimate risk and benefit.
- Validate that the action worked.
- Detect regressions.
- Explain and audit decisions.

### 5.3 Not all optimizations are equal

Actions are classified by risk:

- **Direct:** activated when an objective precondition holds.
- **Proof-gated:** require accumulated evidence or a canary.
- **Patch-based:** modify build logic or CI and require transactional validation.
- **Prohibited:** not automated because they have external effects, unknown semantics, or disproportionate risk.

### 5.4 CI validates; local consumes

CI provides repeatable environments, frequent builds, and a central source of evidence. Higher-risk experiments run in CI. Local environments receive validated policies and perform only low-risk, machine-specific autotuning.

### 5.5 Fail safely

If a policy has expired, the service is unavailable, or an action loses validity **before work executes**, the build uses standard Gradle behavior. A discrepancy discovered in a normal build does not block access or turn a correct build into a failure: the task finishes, its contract becomes `SUSPENDED`, and the optimization is disabled. Strict blocking is used only inside isolated validation whose failure does not control the final pipeline result. After task actions begin, transparent replay of tasks with side effects is not promised; the fallback limits in 12.4 apply.

### 5.6 Do not falsify inputs

An input is never ignored solely to increase cache hit rate. It is normalized or excluded only under a reviewed contract or hermetic guarantee; observed correlation is insufficient.

### 5.7 Less work before faster work

The order of preference is:

1. Do not add an unnecessary task to the graph.
2. Restore its result from cache.
3. Run it incrementally.
4. Run it with better resources and parallelism.
5. Optimize its implementation.

### 5.8 Observation is not proof

Repeating a task and obtaining identical bytes provides evidence of determinism but never proves the absence of a hidden or conditional input. Observation discovers candidates, estimates benefit, and detects regressions; by itself it does not authorize caching or work omission.

An autonomous action that affects correctness needs an executable contract: an official guarantee, reviewed adapter, source patch, or hermetic profile applied continuously to every producer authorized through that path. An isolated sandbox run validates only the path exercised and is not permanent authority.

Hermeticity and determinism are separate gates. A sandbox can prevent a process from reading undeclared state, but does not by itself eliminate internal timestamps, randomness, races, unstable ordering, or non-repeatable behavior. A task is cache-safe only when it passes both gates and its contract is relocatable; repetitions detect violations but do not prove their universal absence.

### 5.9 Compose independently valid fragments

The POC does not treat a complete repository profile as the generic unit of
reuse. The adaptive unit is an independently versioned fragment representing a
producer subgraph, exact output-materialization boundary, task contract,
reviewed patch or bounded cache-locality policy.

Each fragment has a stable repository-scoped family identity and an evidence-
bound revision identity. Checkout paths and Git revisions are not identity
inputs. The revision declares the exact semantic bindings it consumes, such as
Wrapper, workflow, task implementation, producer lineage, output contract,
change family, patch base, platform or cache context. Compatibility compares
only those declared bindings: missing, ambiguous or changed relevant state
suspends that revision, while unrelated drift leaves other fragments eligible.
Stable structural identity never makes stale output bytes or evidence current.

Correctness authority remains local and explicit: Gradle's model or native
contract, a reviewed adapter or patch, or a verified producer boundary.
Cross-repository evidence may prioritize a structurally similar hypothesis but
cannot authorize correctness, qualification or activation in another
repository. Repository scope is therefore isolation data, never a repository-
name product branch.

Fragments progress through `OBSERVED`, `SHADOW`, `QUALIFIED` and `ACTIVE`, with
`SUSPENDED` and `EXPIRED` terminal detours. A suspended fragment must return
through shadow evaluation and qualification; observation count alone never
reactivates it. Dependencies, conflicts, economics and composition are checked
before activation, and native Gradle remains authoritative whenever any
required state is unavailable or ambiguous. The executable contract is
[`Adaptive fragment contract v1`](./specs/poc-adaptive-fragment-contract-v1.md).

The adaptive-fragment generalization POC is now terminally stopped. The
current installed-package evaluation retained exact outputs across 100 paired
builds, but activated no fragment in 71 structurally eligible builds, produced
zero positive repository families, accumulated -368.623 seconds of signed
value and missed the frozen native-retention tail limits. The checked
`STOP_ADAPTIVE_FRAGMENT_POC` decision therefore does not authorize this model
for onboarding or production. The fragment contracts, correctness boundaries
and negative evidence remain architectural research artifacts; any successor
hypothesis must be materially different and separately preregistered rather
than reopening these thresholds. See the
[`AF-015` terminal contract](./specs/poc-adaptive-fragment-terminal-decision-v1.md).

---

## 6. Conceptual acceleration model

Gradle provides different mechanisms that must not be confused:

| Mechanism | What it preserves | When it avoids work | Main limitation |
|---|---|---|---|
| Up-to-date checks | Inputs, outputs, and workspace state | When a task already produced the correct result locally | Lost when the workspace is deleted |
| Incremental execution | Relationship between changed inputs and partial work | When a task supports processing only changes | Depends on the task implementation |
| Local build cache | Task outputs by content key | When previously processed inputs repeat on the same machine | Does not help other runners |
| Remote build cache | Shared task outputs | When any trusted build already produced the same result | Requires correctly modeled tasks and low latency |
| Dependency cache | Dependency artifacts and metadata | Avoids repeated resolution and downloads | Does not avoid compilation or code generation |
| Configuration Cache | Configured task-graph model | Avoids rerunning the configuration phase | Requires compatible build logic and plugins |
| Build Impact Analysis | Minimum required subgraph | Avoids requesting unaffected projects or variants | Requires a conservative impact model |

The product coordinates these mechanisms instead of treating them as independent options.

---

## 7. High-level architecture

```text
┌──────────────────────────────────────────────────────────┐
│                     Customer Repository                  │
│ build.gradle(.kts), settings, plugins, CI configuration  │
└───────────────────────────┬──────────────────────────────┘
                            │
                 ┌──────────▼──────────┐
                 │     CI Launcher     │
                 │ command rewriting   │
                 │ dependency cache    │
                 │ canary assignment   │
                 └──────────┬──────────┘
                            │
              ┌─────────────▼──────────────┐
              │ Gradle Optimization Plugin │
              │ init/settings plugin       │
              │ project plugin/adapters    │
              │ instrumentation/actions    │
              └───────┬───────────┬────────┘
                      │           │
             ┌────────▼───┐   ┌──▼────────────────┐
             │ Build Cache │   │ Evidence Pipeline │
             │ local/remote│   │ fingerprints      │
             └─────────────┘   │ timings/hashes    │
                               └──┬────────────────┘
                                  │
                       ┌──────────▼───────────┐
                       │ Optimization Service │
                       │ policy engine        │
                       │ qualification       │
                       │ autotuning           │
                       │ rollback             │
                       └──────┬────────┬──────┘
                              │        │
                    ┌─────────▼──┐  ┌──▼────────────┐
                    │ Policy API │  │ UI / Audit Log │
                    └──────┬─────┘  └───────────────┘
                           │
                    ┌────────▼─────────┐
                    │ Local Launcher / │
                    │ Gradle Plugin    │
                    │ validated policy │
                    └──────────────────┘
```

The diagram above summarizes the control plane. The operational path and the boundaries that constrain correctness and measurement are:

```text
Customer job / local invocation
  │ repository + original command + deliverables manifest
  ▼
Neutral measurement envelope
  │ timestamps, eligibility and outcome for every experiment arm
  ▼
CI/local Launcher
  ├─ receives authenticated policy from Optimization Service
  ├─ receives Test Optimization grant
  ├─ Gradle + Optimization Plugin
  │    ├─ L1 DirectoryBuildCache, security-generation segmented
  │    └─ L2 Local Verifying Cache Gateway
  │         └─ Edge optional / Shared Backend
  └─ Evidence/events
       └─ Experiment Store/Analyzer
            ├─ Optimization Service / Policy API
            └─ Export Gateway / UI
```

To avoid ambiguity, this document uses these names consistently:

| Name | Process | Responsibility |
|---|---|---|
| Launcher | Go binary that runs before Gradle | Policy, credentials, local proxy, workspaces, baseline, and lifecycle |
| Gradle Optimization Plugin | Java plugin inside Gradle | Public Gradle configuration, listeners, adapters, and inputs |
| JVM Instrumentation Agent | Optional Java `-javaagent` inside the daemon | Deep tracing; not a sandbox |
| Linux Hermetic Helper | Experimental Rust helper supervised by Go | OS observation and isolated process-tree enforcement |
| Neutral measurement envelope | External wrapper shared by every arm | Timestamps, eligibility, and outcome without differential treatment |
| Experiment Store/Analyzer | Service outside the critical path | Assignment, versioned causal results, and promotion gates |

Unqualified “agent” is not used in public schemas or contracts.

### 7.1 Deployment profiles

The architecture retains the same logical contracts while explicitly separating the beta learning profile from the hardened profile:

| Profile | Trust boundary | Identity and provenance | Persistence | Operational promise |
|---|---|---|---|---|
| `PRIVATE_BETA_ISOLATED` | One tenant per deployment, separate repositories/namespaces | TLS, opaque read/read-write tokens, and records authenticated with a deployment key | Single node; filesystem blobs and SQLite WAL metadata | Acceptance targets and fallback; no HA, SLO, RPO/RTO, or claimed resistance to a compromised backend |
| `MANAGED_HARDENED` | Multi-tenant with the data plane treated as untrusted | Workload identity, ephemeral tokens, KMS/HSM, and interoperable attestations verified by the gateway | Object store + HA metadata store | SLO, negative isolation, recovery, RPO/RTO, and customer-facing operation |
| `SELF_HOSTED_HARDENED` | Customer trust domain | OIDC/PKI adapter and customer-managed keys | Supported stores and tested backup/restore | Guarantees declared by the installed profile |

The private beta does not implement a weakened version of multi-tenancy: it removes that boundary through per-deployment isolation. Policy, provenance, attempt, and revocation schemas are versioned from the beginning so local tokens and authentication can later be replaced with workload identity and interoperable signatures without changing cache-key semantics.

### 7.2 CI Launcher

Component that runs before Gradle and can:

- Detect whether the workspace is new or persistent.
- Remove `clean` only when it is proven not to contribute to correctness.
- Merge Gradle invocations only when they pass the semantic allowlist and the control in section 10.2.
- Select only alternate entrypoints authorized by the Build Impact Analysis manifest.
- Set a private `GRADLE_USER_HOME` for each trust variant, reusable only when lifecycle and locking are compatible.
- Acquire the exclusive workspace/runner-slot lease, start or reconnect its stable gateway, register invocation context, and wait for readiness before Gradle.
- Restore dependency cache and wrapper distributions.
- Choose stable or canary policy.
- Download and verify an authenticated immutable policy for the invocation before starting Gradle; beta uses a local deployment key, while the hardened profile uses separate identity/signing.
- Inject the bounded configuration-policy digest and contract version as Gradle-trackable inputs; retain the complete digest for audit.
- Apply cgroup-derived limits.
- Pin the JDK, `org.gradle.jvmargs`, daemon compatibility key, and `--max-workers` before Gradle starts; a plugin inside the daemon cannot change that JVM's heap.
- Create isolated workspaces, credentials, and namespaces for candidate, control, and baseline.
- Fall back to the baseline when a candidate fails and the isolation/replay rules in 12.4 hold.
- Preserve the original command immutably and distinguish failures before and after task actions begin: before work executes it may retry once without the product; afterward it does not blindly repeat tasks with side effects and uses only an already isolated, authorized baseline.

A Gradle plugin cannot merge invocations that have not started, so this responsibility belongs to the CI integration.

### 7.3 Gradle Optimization Plugin

It is preferably injected through an init script and Settings Plugin to minimize initial repository changes. It has adapters for relevant Gradle and plugin versions.

Responsibilities:

- Configure build caches.
- Apply read and write policies.
- Register build listeners and services.
- Model tasks and dependencies.
- Apply compiler, task, and archive configuration; daemon heap and `--max-workers` belong to the Launcher.
- Instrument custom tasks.
- Calculate fingerprints.
- Generate evidence and action records.
- Validate and record entrypoints selected by the Launcher; do not prune the task graph through unsupported internal APIs.

Public Gradle APIs take priority. Any unavoidable internal API use is isolated behind versioned adapters and compatibility tests. Every metric declares its capability as `EXACT`, `APPROXIMATED`, or `UNAVAILABLE` for the observed Gradle/plugin combination; the product never invents a critical path or cache-miss cause when an adapter cannot obtain it reliably.

#### 7.3.1 Optional JVM Instrumentation Agent

The Gradle plugin observes the model exposed by Gradle; it cannot always see the actual behavior of build logic or custom tasks that access Java APIs directly. For those cases, the product offers a deep-diagnostic JVM agent implemented in Java 17 and loaded when the Gradle Daemon starts through [`-javaagent`](https://docs.oracle.com/en/java/javase/17/docs/api/java.instrument/java/lang/instrument/package-summary.html). Dynamic loading through the Attach API is not used.

The Launcher enables it only for selected builds or cohorts. Because `-javaagent` loads when the JVM is created, it is not attached or removed per task: task filtering limits event capture, not installed instrumentation. Instrumented daemons use a separate compatibility key and managed `GRADLE_USER_HOME` for each agent version/mode; they are never reused as uninstrumented daemons. The Gradle plugin provides phase and task context, while the agent emits bounded events for:

- Reads and writes through Java IO/NIO APIs.
- Environment-variable and system-property queries without recording secret values.
- Process creation and redacted command metadata.
- Network attempts through supported Java APIs.
- Use of clock, locale, timezone, and randomness.

This evidence discovers undeclared inputs, suspends discrepant contracts, and generates adapter or source-patch drafts. It never authorizes caching by itself: it does not necessarily cover native code, unintercepted syscalls, or child-process internals. The agent has a transformation allowlist, bounded buffers, an overhead budget, and tested Configuration Cache compatibility. Capturable hooks degrade fail-open for the build result; a fatal daemon error cannot promise that guarantee and follows the limited fallback in 12.4. An overflow, lost event, or budget overrun emits `traceComplete=false`, aborts pending publication, and never counts as positive evidence; `TaskQualificationState` remains `OBSERVING` or `SUSPENDED`. The agent + Configuration Cache matrix uses real Gradle Wrapper processes; TestKit remains for plugins/adapters because [Gradle documents limitations](https://docs.gradle.org/current/userguide/configuration_cache_status.html) when combining TestKit, Configuration Cache, and Java agents.

To prevent an agent beta from causing customer failures, rollout begins in an isolated shadow/diagnostic invocation whose uninstrumented baseline retains the exit code and deliverables. Only an agent/Gradle/JDK combination that passes fixtures, fault injection, and soak can enable `TRACE_OBSERVE` on the normal path, and it remains opt-in per pilot with a kill switch. Local builds and pipelines with side effects do not load the agent by default. A crash attributable to the agent disables it for that compatibility class; the next build does not retry it.

The instrumentation system has two explicit modes:

- `TRACE_OBSERVE`: the JVM Agent and available recorders allow access, record evidence, and emit `CACHE_SUSPENDED` on discrepancy; this is the only mode permitted on a customer's normal path. It must not be confused with the `Observe` product mode.
- `ENFORCE_ISOLATED`: the Linux Hermetic Helper denies undeclared access only in an isolated candidate/producer workspace; on failure, the candidate is discarded, publishes nothing, and preserves the baseline result.

The JVM agent is a tracer, not a security sandbox. Proving hermeticity requires closed enforcement at the operating-system boundary and published coverage for the entire relevant process tree.

#### 7.3.2 Experimental Linux Hermetic Helper

The private beta includes a small Rust helper for Linux x86-64, started and supervised by the Go Launcher. It does not replace the JVM Agent; it closes the boundary the agent cannot observe: child processes, native code, syscalls, filesystem, and network.

The helper builds a versioned `HERMETIC_PRODUCER_PROFILE` using user/mount/PID/network namespaces, read-only mounts for declared inputs, exclusive writable mounts for outputs and temporary files, cgroups, and—when confirmed by the capability probe—Landlock/seccomp. It captures the full process tree inside the boundary; any syscall, mount, descendant, or network channel it cannot mediate produces `traceComplete=false` and never positive evidence.

`traceComplete` derives from an exported coverage vector—filesystem, process tree, network, environment, clock, and randomness—not merely from the helper being installed. The beta can close filesystem/process/network/environment with supported capabilities and block `getrandom`/entropy devices; it does not assume seccomp observes clock access resolved through vDSO or can virtualize every native time source. A task that may consume an unmediated dimension needs an additional adapter/contract or remains `INCONCLUSIVE`; identical repetitions do not fill that coverage gap.

The helper can wrap a complete candidate Gradle process and all descendants to obtain build-level evidence, but that broad boundary **does not by itself prove the hermeticity of an in-process task**: the daemon must read settings, build logic, toolchains, caches, and other invocation inputs, and the kernel does not know which task caused each read. Plugin context may attribute events when exact correlation exists, but incomplete attribution cannot retroactively narrow the OS policy or promote part of the process.

Therefore, `HERMETIC_PRODUCER_PROFILE` can be a task's primary contractual source only when its effective work runs in a dedicated sandboxable process with a task-specific manifest, minimal mounts, and an unambiguous `taskExecutionId → process tree → outputs` relationship. The Launcher creates that producer with explicit toolchains, dependency inputs, outputs, and temporary files. An arbitrary custom task whose logic runs inside the Gradle daemon remains `INCONCLUSIVE` under a whole-invocation sandbox and needs a reviewed adapter or source patch; such a correction may externalize its work into a dedicated producer.

Its operation is bounded:

- `TRACE_OBSERVE` on the normal build continues allowing access and never fails the baseline.
- `ENFORCE_ISOLATED` runs only on a separate candidate, control, or trusted producer whose result remains pending.
- A crash, kernel without required capabilities, lost event, or denial invalidates the candidate and preserves the baseline as the visible result.
- A task qualified only through `HERMETIC_PRODUCER_PROFILE` must run its work in a dedicated producer, with **all** authorized writers under the same profile digest; it cannot validate once and later return to permissive observation.
- Tasks qualified through an official contract, reviewed adapter, or source patch do not need the helper when their contractual path covers relevant behavior.

Rust reduces memory-safety risk at this native boundary and keeps the binary small and auditable, but does not itself guarantee hermeticity. The guarantee comes from fail-closed kernel policy, verified coverage, and refusing promotion when coverage is incomplete.

### 7.4 Optimization Service

Central control plane that:

- Correlates executions from the same repository.
- Manages each optimization's state machine.
- Manages contractual task qualification and configuration validation.
- Runs contextual autotuning.
- Assigns canary cohorts deterministically.
- Invalidates policies when their fingerprint changes.
- Publishes authenticated policies for CI and local environments; the cryptographic mechanism depends on the deployment profile.
- Calculates estimated or causally measured benefit, labels it, and detects regressions.

### 7.5 Build Cache

**Accepted decision — CACHE-001:** the platform provides its own cache backend compatible with Gradle's HTTP Build Cache protocol. It is available as both a managed service and a self-hosted deployment. Third-party backend compatibility remains an interoperability option but does not replace our offering.

Gradle supports one local and one remote cache. The strategy respects that model: `DirectoryBuildCache` is L1 and the HTTP endpoint is L2. By default, that endpoint is the Local Verifying Cache Gateway, backed by the Shared Cache Backend or, when deployed, an Edge Cache Node.

```text
Gradle process
  │
  ├─ L1: Native DirectoryBuildCache
  │       managed by our Gradle plugin
  │
  └─ L2: HttpBuildCache
          └─ Local Verifying Cache Gateway
                 ├─ Shared Cache Backend (default)
                 └─ Edge Cache Node (optional) ──► Shared Cache Backend
```

#### 7.5.1 Managed native local cache

We do not reimplement the local cache format already provided by Gradle. The Gradle Optimization Plugin configures and operates `DirectoryBuildCache`:

- Managed directory and, where possible, dedicated volume or quota.
- Native Gradle [retention and cleanup](https://docs.gradle.org/current/userguide/directory_layout.html#dir:gradle_user_home), configured through an init script and adapted to the supported version.
- Persistent volumes and restoration between jobs when the volume has one compatible writer.
- Separation by repository and compatibility.
- Segmentation of any L1 that may receive remote hits by tenant, repository, trust domain, `cacheCompatibilityClass`, and `l1SecurityGeneration`.
- Metrics, export, and diagnostics.
- Fallback when remote connectivity is unavailable.

This is the no-additional-infrastructure mode for developers and persistent runners. The same writable local Gradle directory is never mounted concurrently on multiple runners. The product does not interpret the internal format or implement its own LRU over those files. Under disk pressure it reduces retention, rotates a managed directory, or disables it for a later invocation. Rotation occurs during Launcher startup, after gaining exclusivity and before starting Gradle; it never mutates a directory a daemon or build may be using. Maintenance failure does not fail the build.

Gradle checks L1 before L2 and may copy a remote hit into it; the gateway does not reverify that object on later builds. `l1SecurityGeneration` is an authenticated monotonic generation derived from cumulative revocation state. Revoking an identity, key, provenance record, or namespace atomically rotates **all** L1 content potentially populated from that remote scope before the next build. A future generation is rejected, and an old generation is accepted only against authenticated cumulative state proving it remains current. In private beta, that authentication uses the local deployment key; the hardened profile uses a signature verifiable against its trust root. If policy requires strict expiration, L1's maximum lifetime does not exceed that horizon; if state cannot refresh, L1 rotates or is disabled. A broader L1 is allowed only when the product can prove it never received L2 hits.

Managed lifecycle and deletion cover directories or volumes under Launcher control. Copies a customer exports or mounts outside that control are external destinations and outside the deletion guarantee.

In A0/A1, every invocation authorized to write L2 disables `DirectoryBuildCache` and uses only pending L2 during that attempt. A pending PUT therefore leaves no reusable local copy that bypasses abort, while stable writer configuration can remain compatible with Configuration Cache. A later evolution may use a private L1 per `attemptId`, but must discard it on abort and promote only the complete directory after the verdict and with exclusivity; it never mixes opaque files into an active L1.

#### 7.5.2 Local Verifying Cache Gateway

The default managed mode inserts a loopback Go proxy between Gradle and remote storage. Gradle continues using the public `HttpBuildCache` protocol; the gateway adds guarantees that protocol does not carry:

- Exposes `GET/PUT {key}` through a stable loopback rendezvous per managed workspace/runner slot and trust domain, with exclusive ownership and `gatewayConnectionGeneration`. Concurrent builds use different slots/endpoints and never change an active connection's context. The Launcher registers the invocation and authenticated policy over a privileged local channel before Gradle, and the gateway routes no request without current context. The credential visible to Gradle authenticates only that local hop, lives no longer than the Configuration Cache generation, and is never the remote credential. The gateway renews upstream tokens outside the Gradle model. Rotating the endpoint or local credential deliberately changes `gatewayConnectionGeneration` and causes one safe reconfiguration; restarting the process while retaining the slot does not destroy a hit.
- Retrieves the payload and sidecar provenance record from the backend into an encrypted or exclusively permissioned temporary spool with byte reservation, per-object limit, and separate quota. It always verifies checksum, namespace, revocation, and policy compatibility over the complete payload before returning `200`; the hardened profile also verifies an independent signature and trust root. Any failure deletes the spool and becomes a safe miss; bytes are never delivered before verification completes.
- On PUT, calculates the checksum while streaming into a pending attempt. In private beta, the provenance record is authenticated with the local deployment key and Shared is part of the TCB. In the hardened profile, an ephemeral workload-bound key or authority separate from the data plane signs the attestation; Shared/Edge cannot mint `stable` provenance by themselves.
- Routes GET and PUT independently: it may read `stable` and write `pending/{attemptId}` even though Gradle knows only one URL.
- Keeps trust material in memory during the invocation and receives it inside the authenticated policy.

The spool supports backpressure, cancellation, idempotent crash cleanup, and conservative reservation when `Content-Length` is absent or untrusted. `Expect: 100-continue` and a valid `Content-Length` enable early rejection but are not a security boundary. Quota exhaustion, full disk, late checksum failure, or cancellation produces a miss/fallback, never a partial hit or cache-caused build failure. A chunk/Merkle protocol may reduce buffering after beta; first-version targets do not assume it.

In the hardened profile, compromising Shared/Edge is insufficient to deliver arbitrary bytes to Gradle. In private beta, Shared and the local key are explicitly part of the trusted computing base: the gateway detects accidental corruption and incompatible state but does not claim resistance to a malicious backend. Direct Gradle → Shared compatibility remains for third parties and deployments that choose it; direct mode uses a separate trust domain/namespace and never mixes objects with verified hardened mode.

Publication flow: the gateway opens an `attemptId`, serves committed reads, and retains each pending candidate as `(tenant, namespaceGeneration, key, attemptId, checksum)`. First-writer-wins CAS occurs at **authorized commit**, not upload start; aborting or expiring a lease releases the candidate. From MVP-A0, a trusted writer of allowlisted tasks finalizes an attempt only after a successful build, current policy/grants, and profile-valid provenance/attestation; failure aborts the whole attempt and affects cache warming, not the already correct build result. In MVP-C1, plugin and gateway correlate task outcome ↔ cache key over a versioned local channel and add the tracing/enforcement verdict. If a version lacks exact correlation, the product aborts the complete attempt and never guesses which PUT belongs to a task.

Authorization materializes as a canonical immutable authenticated `CommitDecision` binding `attemptId`, repository/trust domain, the exact list of `(namespaceGeneration, key, checksum, size)`, applicable `policyDigest`, `configurationPolicyDigest`, and `cacheContractDigest`, grant digest and expiration, `revocationEpoch`, validation verdict, and the decision's own expiration. The backend rejects an incomplete, expired, revoked, payload-reused, or incompletely covering decision. It does not query `control.sqlite` inside the visibility transaction or implement a distributed transaction across stores: it persists the verified `CommitDecision` and `COMMITTED` records in the **same `cache.sqlite` transaction**. The `control.sqlite` ledger later references that digest idempotently; failure of that later write does not change object validity, and the reconciler repairs the audit index from the durable decision.

#### 7.5.3 First-party Shared Cache Backend

This is the shared backend and central storage product. It is offered in two forms:

- **Managed:** one isolated deployment per pilot during private beta, and multi-tenant SaaS only after GA-D.
- **Self-hosted/on-premises:** deployed inside customer infrastructure.

The backend stores opaque payloads produced by Gradle; it does not interpret or reconstruct their format. The physical immutable committed identity is `(tenant, namespaceGeneration, key)`. Provenance is a many-to-one set of beta provenance records or hardened attestations associated with the object; it creates no separate identity and does not require two readers to share a revision.

##### 7.5.3.1 HTTP conformance contract

The Gradle → gateway boundary and direct interoperability mode implement the official [`HttpBuildCache` contract](https://docs.gradle.org/current/dsl/org.gradle.caching.http.HttpBuildCache.html) exactly:

- `GET {baseUrl}/{key}` returns `200` with the payload for a hit and `404` for a miss.
- `PUT {baseUrl}/{key}` accepts any `2xx`; `413 Payload Too Large` rejects entries above quota without corrupting a previous object.
- PUT redirects use only `307` or `308`; `301`, `302`, or `303` would cause the redirect to be followed with GET.
- Managed mode emits no credentialed cross-origin redirects; self-hosted permits them only to hosts explicitly listed in trust policy.
- When the client enables `Expect: 100-continue`, the backend may return `413` before receiving the body and responds correctly to the handshake.
- Any other response is an error, and Gradle disables remote cache for the rest of that build. The Launcher and observability reflect this without failing the build.
- TLS is mandatory between the gateway and remote services except for an explicit self-hosted exception. The Gradle → gateway hop may use HTTP only over the validated stable loopback rendezvous, without redirects and with `allowInsecureProtocol` limited to that URL; alternatively it uses local TLS. The Gradle client uses Basic Auth with a minimum-scope local credential bound to `gatewayConnectionGeneration`; it is not a remote token, rotates in coordination with Configuration Cache, and is never logged.
- A conformance suite exercises hit, miss, PUT, concurrent collision, `413`, redirects, timeout, retry, and corrupt payload against every supported Gradle version.

The `pending/committed/tombstoned` state, provenance records/attestations, and `start/commit/abort attempt` belong to a versioned internal protocol between gateway, control plane, and backend; they are not attributed to `HttpBuildCache`. A Gradle PUT may receive `2xx` after becoming durable in `pending`, but no general GET sees it before authorized commit. A timeout, crash, or incomplete verdict aborts the attempt through lease/TTL.

##### 7.5.3.2 Storage and security semantics

The private beta uses a deliberately small single-node implementation:

- The Shared Go process keeps content-addressed blobs on a dedicated local filesystem and authoritative metadata in SQLite WAL on local disk; SQLite and the blob directory never reside on a network filesystem.
- Cache metadata (`cache.sqlite`) and control/experiment state (`control.sqlite`) have separate files, migrations, and lifecycles. `BUILD_SESSION`/`EXPERIMENT_RESULT` are also exported as append-only JSONL; losing cache state does not silently erase evidence, and losing control state does not make an old local policy valid.
- A PUT first writes to spool, calculates SHA-256, runs `fsync` according to policy, and moves the immutable blob to its digest path. Only then does a `cache.sqlite` transaction—with a unique index on `(tenant, namespaceGeneration, key)`—persist the verified `CommitDecision`, perform CAS, and atomically make every `COMMITTED` record visible.
- A crash before the transaction leaves at most an orphan blob for the reconciler to delete; a record without a blob, with inconsistent size/checksum, or with a partial file becomes a miss and quarantine, never a hit.
- `lastAccess` updates in batches so SQLite does not become the bottleneck. One instance owns the writer lock; this profile claims no HA, failover, or RPO/RTO, and losing the node rebuilds cache from natural builds.
- If `control.sqlite`, the local key, or monotonic epochs cannot be recovered, the service starts without active optimizations, rotates policy/namespace/L1 generations, and requires relearning; it never reconstructs authorization from blobs or historical telemetry.
- Storage sits behind blob and metadata interfaces with conformance tests. The hardened profile replaces these implementations with object storage and an HA transactional database without changing the attempt, CAS, or visibility protocol.

The following semantic contract is common from MVP-A0. Customer-facing multi-tenant operation is implemented only during hardening:

- Namespaces and isolation across tenants, repositories, trust domains, platforms, and policy classes.
- Independent read and write credentials scoped to specific namespaces. In beta they are manually provisioned and rotated opaque tokens; in hardened mode they are short-lived and derived from workload identity.
- Atomic first-writer-wins commit through compare-and-set after the verdict: a reader sees the complete committed object or a miss, never partial or pending bytes.
- Immutable committed objects by `(tenant, namespaceGeneration, key)`: a later identical commit reuses the blob and idempotently adds a valid provenance record/attestation; different content is rejected, makes that identity unreadable, quarantines both evidence sets, and raises a collision or poisoning alert. A poisoned or tombstoned object is never overwritten: correct rebuilding uses a new authenticated `namespaceGeneration`.
- Payload checksum verified on write and read. The digest protects transport and storage integrity; beta authenticates its provenance record with the local key, while the hardened profile requires an attestation signed by an authorized writer identity outside the data plane.
- Concurrency control and deduplication without exposing an incomplete upload. The isolated beta does not deduplicate across deployments; the hardened profile does not physically deduplicate across tenants or trust domains either.
- Authoritative metadata for visibility, provenance/attestations, revocation epochs, tombstones, and leases; a blob without committed metadata is treated as nonexistent.
- Bounded renewable leases for uploads/downloads/replications; a dead process cannot block eviction indefinitely.
- Retention, size limits, quotas, and size-aware SLRU according to section 7.5.3.4.
- TLS in transit for every remote hop starting in beta. Every managed beta deployment uses infrastructure-provided volume encryption; it does not yet implement per-tenant envelope keys or KMS/HSM. Self-hosted deployments must explicitly declare an unencrypted volume before storing real artifacts.
- Transfer, latency, error, and circuit-breaker metrics.
- QPS, concurrent-stream, ingress/egress, and in-flight byte limits starting in beta; principal/tenant hierarchy, cross-tenant fair queuing, and noisy-neighbor guarantees belong to the hardened profile.
- Cache warming only from objects with trusted provenance.
- Administration and export API with the same isolation.
- Fail-open: if L2 degrades or fails, the circuit opens and Gradle runs normally using L1.

`HttpBuildCache` has no native write-only mode separate from reads. The gateway separates internal GET and PUT paths; without the gateway, an experiment that must prevent reuse uses an initially empty namespace or `push=false`. It never assumes `push=true` disables reads.

##### 7.5.3.3 SLI, beta targets, and later SLO

Pre-release tests cannot prove a monthly SLO. The profiles separate two contracts:

- **Private-beta acceptance targets:** load, fault-injection, and soak tests with concurrency, warm/cold state, and size distribution pinned in the benchmark specification. Observed results are published, not a contractual availability promise.
- **Later hardened SLO:** monthly measurement over eligible managed traffic. The availability SLI is `authenticated syntactically valid requests with a conforming response / authenticated syntactically valid requests`. A correct `404` is protocol success, as is a `413` consistent with the contracted limit; timeouts, `5xx`, internal authentication failure for a valid credential, corruption, and invalid responses are failures. Correct rejection of an invalid credential and client cancellations are measured separately.

| Target and published benchmark environment | Beta acceptance target; hardened SLO target where stated |
|---|---|
| Monthly GET/PUT availability | No beta SLO; hardened target ≥99.9% after observing operational traffic |
| p95 GET miss measured at gateway and backend | ≤50 ms under published benchmark load |
| p95 `backendFirstByteMs` | ≤100 ms with published benchmark object and state; upstream diagnostic, not a customer promise |
| p95 gateway `verifiedHitReadyMs` | ≤150 ms up to 1 MiB; ≤400 ms up to 10 MiB; ≤2.5 s up to 100 MiB; includes upstream download, spool, and verification before `200` |
| p95 downstream materialization | ≤`150 ms + payloadBytes / 200 MiB/s` on the benchmark runner; from verified payload to complete restore |
| p95 durable pending PUT confirmation after receiving the body | ≤150 ms for the benchmark object class; commit measured separately from the verdict |
| Known corruption served | 0; read blocked and object quarantined |
| Tenant/repository isolation | 100% in negative authorization tests |

The 99.9%-equivalent error budget—approximately 43 minutes per month—is modeled and observed in beta but not marketed as an SLO. When the hardened profile activates, consuming it stops warming and nonessential work, opens circuit breakers, and prioritizes reads. Targets are measured from client and server. Beta rejects objects above 100 MiB; a class up to 1 GiB requires its own opt-in, benchmark, and SLO after beta. Protocol availability does not measure disappearance of an expected object: durability/unexpected loss, hit-rate anomaly, revocation freshness, and spool exhaustion are separate SLIs. Verified-hit and restore targets include their corresponding transfers and are always published by size bucket.

The beta self-hosted distribution shares the single-node profile and explicitly provides no high availability. Health/readiness, reconciliation, and fail-closed recovery are mandatory; contractual backup/restore, HA stores, and RPO/RTO close during hardening. After restart, the service remains closed to reads/writes until visibility, revocation, and tombstone metadata load.

##### 7.5.3.4 Admission, retention, and eviction

**Accepted decision — CACHE-004:** the Shared Cache Backend and Edge use hard quotas, TTL, and byte-weighted Segmented LRU, not pure LRU or LFU.

Private-beta defaults: maximum object 100 MiB; 100 GiB logical per repository and 500 GiB per deployment; 30-day `stable` TTL; 85%/75% high/low watermarks; pending and quarantine share at most 10% of quota and cannot directly evict `stable`. In self-hosted mode, the installer limits total capacity to the smaller of 500 GiB and 50% of usable volume, and requires at least 20 GiB. A pilot may reduce any value; increasing above 100 MiB per object requires another benchmark class.

Each tenant has limits by repository, trust domain, and namespace, plus maximum object size. With valid `Content-Length`, admission control checks object limit, reserved quota, and capacity to complete the upload before the body. Without trustworthy length, it reserves an initial segment, accounts the stream against in-flight bytes, and aborts with `413` as soon as it reaches the maximum; it never trusts the client to stay within capacity. Oversized objects never enter probation. Quarantine/pending have separate quota and logical disk so malicious inputs cannot evict stable. Every pool has configurable high and low watermarks: reaching the first triggers eviction until the second; if space cannot be freed safely, the service stops admitting new PUTs before disk exhaustion and preserves possible reads.

SLRU has two segments:

- `probation`: every new entry starts here; a one-time load does not immediately displace the stable working set.
- `protected`: an authorized correctly served remote hit promotes the entry; entries no longer used become evictable by recency again.

Both segments are sized by bytes. If a promotion exceeds the `protected` target, its least-recent entry returns to `probation`; eviction frees as many objects as needed to recover the low watermark, not a fixed number of keys.

Tainted, revoked, or invalid objects leave the readable index immediately and move to an evidence store with its own quota; they do not wait for disk pressure. Eviction of still-valid objects is byte-based and follows this order:

1. Expired entries.
2. Expired or resolved quarantine within its pool.
3. Least-recent entries in `probation`.
4. Least-recent entries in `protected`.

An upload, download, or replication with a valid lease is never removed. `lastAccess` and frequency update in batches to limit write amplification; only authenticated hits count and rate limits prevent a client from keeping objects artificially hot. Deletion is logical first and asynchronous physical deletion second.

Pressure resolves first within the over-quota namespace/tenant and then through weighted fair share while respecting reserved minima; a noisy neighbor cannot consume another tenant's reservation. Segment-membership and size metadata can be rebuilt, but expiration, commit state, revocation, and tombstones are durable and authoritative. After a crash, the node rebuilds a conservative index before accepting traffic and never treats a partial object as visible.

The first version implements neither pure LFU nor a value formula based on task times without exact correlation. TinyLFU may later be added as an admission filter if metrics demonstrate scan pollution; once exact task/key association exists, the score may include avoided time and transfer cost without replacing quotas or SLRU.

#### 7.5.4 Edge Cache Node

Optional service deployed on a workstation, CI host, cluster, or local network. The Local Verifying Cache Gateway uses it as a nearby upstream, and the Edge Node proxies the Shared Cache Backend; Gradle still sees only the loopback gateway.

It provides:

- Persistent cache shared across containers and workspaces on the same host.
- Low latency for runners on the same network.
- Temporary offline operation.
- Compression only when measurement proves benefit without changing the opaque payload, deduplication limited to the same tenant/trust domain, quotas, TTL, and the same size-aware SLRU as the shared backend.
- Asynchronous replication to the shared backend.
- Trusted-writer/read-only policy enforcement.

The Shared Backend is the only commit authority. Every PUT received by Edge remains `PENDING` and is not a general hit until central acceptance; two Edge nodes never resolve a collision themselves. Offline reads serve only already `COMMITTED` objects whose provenance/attestation and revocation can be validated against current state. An offline write may be reread only within the same isolated attempt and is never presented as `stable`; it aborts if not replicated before its TTL.

It is not mandatory for a developer with one `GRADLE_USER_HOME`, where native local cache is usually sufficient.

### 7.6 Patch Engine

Component of `MVP-C4 / Beta Autopilot`. It generates ephemeral or persistent changes to:

- `settings.gradle(.kts)`.
- `build.gradle(.kts)`.
- Convention plugins.
- `gradle.properties`.
- CI configuration.

A patch can be promoted only after passing its validation policy.

The transformation materializes inside the isolated CI workspace: the control plane sends a recipe/adapter version and receives digests, status, and redacted evidence—not source or diff by default. The patch bundle remains a customer-owned artifact until the C4 workflow uses it to open the PR. Uploading the diff to the service would require a different opt-in data profile and is unnecessary for beta.

The only private-beta delivery mechanism for a persistent source change is a draft pull request bound to `sourceRevision`, `sourceStateDigest`, and patch digest; it never automatically rebases over conflicting customer changes. Because Git requires a ref to open a PR, a protected job inside the repository—not the backend—may use its short-lived `GITHUB_TOKEN` only to create a `buildopt/<actionId>` head branch from the validated SHA and open the draft PR. The patch bundle and that head are supporting customer-owned artifacts, not alternate integration paths. The workflow does not modify existing/default branches or automatically merge. Runtime injection is limited to reversible configuration through public Gradle APIs. A source/build-logic change that alters declarations, actions, or semantics is not disguised as an ephemeral transformation: it requires a reviewable patch.

#### 7.6.1 Canonical patch bundle

**Accepted decision — PATCH-BUNDLE-001:** C4 uses a declarative `PatchBundle v1` without executable scripts or hooks. Its `manifest.json` is canonicalized with JCS, bound to `repositoryId`, `actionId`, `baseRevision`, `baseTree`, `sourceStateDigest`, recipe/version, and expiration, and authenticated with the beta deployment key. To avoid circular definitions or ambiguous concatenation, `bundleDigest` is the SHA-256 of `JCS({"manifest": <manifest without digest/signature>, "blobs": [<blobRef, blobSha256, size sorted>]})`; the signature covers that digest and version/key-ID fields.

Operations are ordered, and each declares a relative path, `ADD|MODIFY` type, expected mode, preimage digest—absent for `ADD`—postimage digest, and replacement blob. Beta permits no deletes, executable-bit changes, binary patches, fuzzy matching, or commands. Before applying, the customer-side Java patcher:

1. verifies signature, expiration, repository, `actionId + bundleDigest`, base revision/tree, and source state;
2. rejects absolute, empty, NUL-containing, `..`, `.git`, symlink-escaping, submodule, and outside-worktree paths;
3. opens each path without following symlinks and requires exact mode and preimage;
4. writes into a staging tree, verifies every postimage digest, and only then creates the commit on the ephemeral head;
5. recalculates `sourceStateDigest` and records the result without executing code contained in the bundle.

`actionId + bundleDigest` is the idempotency key. If the exact head already exists and points to the expected commit, a retry reuses that state; if it contains different content, the workflow stops without force-pushing. If the branch was created but opening the PR failed, the workflow searches for an exact draft PR and retries only its creation. It does not delete branches or resolve divergence automatically.

The first C4 slice implements only two recipes with positive and negative fixtures:

- `ARCHIVE_REPRODUCIBILITY_KOTLIN_DSL_V1`: configures reproducible order and non-preserved timestamps for supported archives.
- `CUSTOM_TASK_CONTRACT_JAVA_V1`: adds input/output declarations to a Java custom task only when they come from a reviewed C1 adapter and the preimage matches exactly.

Groovy DSL, general Configuration Cache repair, eager→lazy conversion, annotation processors, and CI YAML editing remain on the C4 roadmap but are not developed in parallel before these two recipes are proven.

### 7.7 Observability Export Gateway

Component responsible for exposing telemetry outside our UI and preventing data lock-in. It offers:

- Local per-build export.
- API to query historical builds and actions.
- Event streaming for real-time integrations.
- Delivery to customer object storage, data lakes, SIEMs, or observability platforms.
- Centralized schema, compatibility, and redaction.
- Bounded retries and buffering that do not block the build.

JSON and JSONL are part of the initial contract. OpenTelemetry, Prometheus, and other destinations are implemented as adapters over the same canonical model.

In private beta, the export gateway is limited to file/stdout and JSON/JSONL CI artifacts with bounded buffering; the historical API may be exposed only inside the isolated deployment. Remote streaming, OTLP, Prometheus, Parquet/object storage, SIEM, and webhooks belong to GA-D or expansion and do not block functional learning.

### 7.8 Initial compatibility matrix

**Accepted decision — COMPAT-001:** MVP-A0/A1 has a narrow, testable matrix pinned against Gradle 9.6.1 and its [official compatibility matrix](https://docs.gradle.org/current/userguide/compatibility.html) as of July 21, 2026:

| Tier | Gradle Wrapper | JVM running Gradle | Platform | Scope |
|---|---|---|---|---|
| Tier 1 | 8.14.x | JDK 17 or 21 | Linux x86_64 | Core Java/JVM plugins, Groovy/Kotlin DSL, cache, and observability |
| Tier 1 | 9.6.x | JDK 17, 21, or 25 | Linux x86_64 | Core Java/JVM plugins, Groovy/Kotlin DSL, cache, and observability |
| Observe-only | Other detected combinations | According to official Gradle compatibility | Linux, macOS, or Windows | No autonomous actions or exact-metric promise |

The **implementation golden lane** is narrower than the Tier 1 gate: Gradle 9.6.1, JDK 21, Linux x86_64, Kotlin DSL, and a 4 vCPU/16 GiB development runner class, executed directly on the workstation and inside a digest-pinned container image. The class matches the initial development workstation and serves conformance and integration; its measurements do not become a performance baseline extrapolated to customer runners. GitHub Actions reuses the contract when base CI is implemented but does not block this local gate. The first walking skeleton and plugin/agent spikes begin there. The hermetic helper also uses a Linux test runner with pinned kernel/capabilities because an ordinary container does not prove that namespaces/Landlock/seccomp are available. Gradle 8.14.x, JDK 17/25, Groovy DSL, and the rest of the Tier 1 matrix are A0/C1 exit gates, not parallel initial development fronts.

Android, Kotlin Multiplatform, native builds, and third-party-plugin-specific optimizations remain outside initial Tier 1. In `Verified`, shared cache is default-deny: it is enabled only for an exact set of verified task types, implementations, plugins, and versions, and the instance must retain its expected actions/contract. The plugin applies `doNotCacheIf` to every disallowed task. Artifact transforms do not pass through `TaskOutputs`; Shared is enabled only when the adapter can prove there are no unknown transforms or enumerate an allowlisted set. If the capability is `UNAVAILABLE` or another provider/version appears, managed Build Cache is disabled for the whole invocation. A disposable namespace is allowed only in isolated validation whose result is not customer-visible. The product does not pretend a nonexistent per-transform switch exists.

A `CUSTOMER_ASSUMED_PASSTHROUGH` mode may retain the customer's existing cache configuration, but it uses a separate trust domain/namespace and is outside product promises for verification, correctness, or attributable impact. Every expansion requires TestKit, one fixture repository per ecosystem, custom-task and transform tests, Configuration Cache, HTTP protocol, and a published capability matrix. An unknown combination degrades to `Observe`, private L1, or Gradle without our plugin; it never automatically receives a policy produced for another version.

### 7.9 Implementation stack

**Accepted decision — STACK-001:** Java and Go are the primary stacks; the private beta adds a bounded experimental Rust helper for Linux hermetic enforcement without making Rust a requirement for installing core capabilities.

| Component | Initial language | Operational reason |
|---|---|---|
| Gradle init/settings plugin, adapters, and TestKit fixtures | Java 17 | Native Gradle API, binary compatibility, and execution on JDK 17/21/25 |
| JVM Instrumentation Agent | Java 17 | `java.lang.instrument`, shared daemon context, and one JVM artifact |
| CI/local Launcher and Local Verifying Cache Gateway | Go | Static binary, loopback proxy, provenance/attestation verification, and cross-platform operation |
| Shared Cache Backend and Edge Cache Node | Go | Networking, concurrency, self-hosted deployment, and simple operation |
| Optimization Service, Policy API, and Export Gateway | Go | Unified control plane, services, and operational tooling |
| Patch Engine orchestration and patch bundles | Go | State machine, evidence binding, diff packaging, and PR-workflow coordination |
| Gradle DSL/source-patch adapters | Java 17 | Source-aware transformations tied to concrete Gradle APIs, plugins, and fixtures |
| Experimental Linux Hermetic Helper | Rust | Small native boundary for namespaces, mounts, Landlock/seccomp, and process-tree tracking |

The Go Launcher orchestrates lifecycle, policy, cgroups, baseline, and communication; the Rust helper applies the profile's OS boundary when enabled. Repositories that do not qualify tasks through hermeticity need not install it. Rust is not the source of hermeticity: the guarantee comes from fail-closed kernel policy, verified coverage, and retaining every output in pending state until the verdict.

### 7.10 Private-beta artifact topology

**Accepted decision — DEPLOY-001:** beta is packaged into a small number of artifacts with clear ownership:

| Artifact | Content and lifecycle |
|---|---|
| `buildopt` | Go binary distributed to the runner; contains Launcher, neutral measurement envelope, and Local Verifying Cache Gateway |
| `buildopt-server` | Per-pilot Go modular monolith; hosts Shared Cache, Policy API, experiment/evidence state, and export using local filesystem + SQLite |
| `buildopt-gradle-plugin.jar` | Java 17 init/settings/project plugin and adapters |
| `buildopt-jvm-agent.jar` | Separately versioned opt-in Java 17 JVM Instrumentation Agent |
| `buildopt-hermetic-helper` | Optional C1 Linux x86-64 Rust binary; not installed for A1/B |
| `buildopt-patcher.jar` | Customer-side Java patcher that validates and materializes C4 bundles without executing their content |
| GitHub Action + protected workflows | SHA/checksum-pinned installation, normal build, isolated validation, and PR materialization |

`buildopt-server` retains module and schema boundaries, but the private beta does not split cache, policy, experiment, and export into microservices. Physical separation is justified later only by operational metrics or hardening requirements.

Beta UX is CLI/CI-first: job summary, bounded annotations, and `BUILD_SESSION`/`EXPERIMENT_RESULT` JSON/JSONL artifacts. The view described in section 23 serves as an information model and future direction; a customer-facing web UI does not block A1+B+C1+C4. An internal diagnostic console may exist but is not part of the pilot contract.

---

## 8. Evidence model

### 8.1 Work units and build fingerprints

The identity of the requested work is separated from the variant that executes it. Without this separation, candidate and control would never be comparable because the policy itself would be part of the fingerprint.

```text
WorkUnitsFingerprint =
    repository identity
  + source revision
  + source state digest (tracked, untracked and dirty content)
  + requested work manifest
  + required deliverables/checks manifest
  + build logic and settings digests
  + Gradle and JDK/toolchain versions
  + operating system, architecture and runner/cgroup class
  + baseline-relevant environment
  + Test Optimization policy/grant digest

BuildFingerprint =
    WorkUnitsFingerprint
  + treatment policy/action digest
  + launcher/plugin/JVM agent/hermetic helper output-semantics versions
  + cache namespace and trust domain
  + workspace, daemon and cache state
```

`WorkUnitsFingerprint` deliberately excludes the evaluated action, cache warmth, daemon warmth, and any candidate/control-specific policy. Those elements are covariates or treatment metadata. A paired control requires the same fingerprint; a randomized cohort requires balanced/stratified fingerprint distributions and the same eligibility rule. Candidate and control also retain the same Test Optimization grant/policy; if they differ, the result measures a combined effect and is not attributed exclusively to Build Optimization. For local builds or revisions without a clean commit, `sourceStateDigest` incorporates normalized content and paths: a Git revision alone is insufficient.

The configuration the customer would use without the product is captured as a versioned `baselineDefinition`, and its digest is included in every `PRODUCT_TOTAL` assignment. If the customer voluntarily changes that baseline, a new `measurementEpoch` begins and effects across eras are not aggregated as though they shared a control. A neutral envelope outside the treatment measures both arms using the same time boundary.

The digests have separate responsibilities:

- `policyDigest`: hash of the complete policy for signing, auditing, and provenance; changing UI, rollout, or telemetry does not by itself invalidate outputs.
- `configurationPolicyDigest`: canonical hash of only the decisions that alter configuration/task selection; it is registered as a Configuration Cache input.
- `cacheContractDigest`: hash of a task's semantic contract—inputs, outputs, normalization, and tools; it enters the native key only if that information is not already represented or can change the bytes.
- `cacheCompatibilityClass`: namespace boundary for incompatible platform/format/trust domains.
- `revocationEpoch`: monotonic counter per trust domain that the gateway checks before serving a hit.
- `l1SecurityGeneration`: L1 directory generation that rotates state potentially materialized from L2.
- `gatewayConnectionGeneration`: version of the local rendezvous/credential; it changes only when the serialized connection must be invalidated.

The full Launcher, plugin, JVM agent, or hermetic helper version is metadata. Only a bounded `outputSemanticsVersion` participates as a task input when the component can modify the result; `HERMETIC_PRODUCER_PROFILE` also retains the helper/kernel capability digest. Internal fingerprints are not used as cross-tenant identifiers; on export, they are tokenized with the tenant's HMAC/key version to prevent correlation or attacks against known repositories.

### 8.2 Task fingerprint

Represents a potentially equivalent execution of a task:

```text
TaskFingerprint =
    task implementation classpath
  + task type
  + plugin versions
  + normalized task configuration
  + declared inputs
  + inputs declared by an approved adapter
  + adapter/contract digest
  + relevant environment
  + toolchain identity
```

Observed inputs that have not yet been registered with Gradle are not part of a safe key: they are only candidates for correcting the contract. No `observed cache keys`, aliases, or second output cache will be created. Before enabling caching, the plugin must register the approved manifest as real task inputs, including path sensitivity and modeled environment properties. Gradle will then compute its native key from normalized paths, content hashes, implementation, and declared properties.

The task path must not be assumed to be the semantic identity, but it will not be removed from the identity without a relocatability and equivalence guarantee. Reuse across projects is allowed only when the approved contract, implementation, inputs, outputs, and platform are equivalent.

### 8.3 Evidence record

```yaml
task: ":frontend:bundle"
sourceRevision: "8c74f2a"
sourceStateDigest: "sha256:..."
taskImplementationHash: "sha256:..."
inputFingerprint: "sha256:..."
outputDigest: "sha256:..."
workspaceClass: "ephemeral-linux-x86_64"
launcherVersion: "1.3.0"
pluginVersion: "1.3.0"
jvmAgentVersion: null
policyDigest: "sha256:..."
cacheNamespace: "tenant-7/repo-42/stable/linux-x86_64"
qualificationSource: "REVIEWED_ADAPTER"
qualificationState: "QUARANTINE_VALIDATED"
cacheContractDigest: "sha256:..."
observedReads:
  - "frontend/src/**"
  - "package.json"
  - "package-lock.json"
observedWrites:
  - "build/frontend/**"
environmentInputs:
  - name: "NODE_VERSION"
    classification: "PUBLIC_VERSION"
    valueDigest: "hmac-sha256:..."
networkAccess: false
wallClockAccess: false
randomnessObserved: false
traceComplete: true
traceCoverage: "JVM_IO_PLUS_LINUX_PROCESS_TREE_V1"
durationMs: 48120
outcome: "SUCCESS"
```

### 8.4 Optimization policy

```yaml
schemaVersion: "1.0"
policyId: "repo-42/build-logic-b93/linux-jdk25"
policyVersion: 17
policyDigest: "sha256:..."
configurationPolicyDigest: "sha256:..."
revocationEpoch: 42
l1SecurityGeneration: 42
gatewayConnectionGeneration: 7
signatureKeyId: "policy-signing-2026-q3"
issuedAt: "2026-07-21T09:55:00Z"
launcherVersionRange: ">=1.3.0 <2.0.0"
pluginVersionRange: ">=1.3.0 <2.0.0"
mode: "VERIFIED"
allowedActions:
  - "REMOTE_CACHE_ALLOWLISTED"
  - "CONFIGURATION_CACHE"
remoteCache:
  read: true
  write: "trusted-ci-only"
  namespace: "tenant-7/repo-42/stable/linux-x86_64"
  namespaceGeneration: 12
configurationCache:
  enabled: true
  contractVersion: "cc-policy-v3"
testOptimizationGrant:
  digest: "sha256:..."
  expiresAt: "2026-07-22T00:00:00Z"
resources:
  maxWorkers: 6
  gradleHeapMb: 6144
budgets:
  maxSynchronousOverheadMs: 500
  maxSynchronousOverheadRatio: 0.02
  maxValidationRunnerMsPerDay: 3600000
exportProfile: "tasks"
qualifiedTasks:
  - implementationHash: "sha256:..."
    adapter: "frontend-bundle-v2"
    cacheContractDigest: "sha256:..."
    qualificationState: "QUARANTINE_VALIDATED"
affectedBuild:
  enabledInCi: true
  enabledLocally: false
expiresAt: "2026-09-01T00:00:00Z"
```

### 8.5 Action record

Every intervention will be auditable:

```yaml
action: "ENABLE_TASK_CACHE"
target: ":frontend:bundle"
stateType: "ACTION_ROLLOUT"
state: "ACTIVE_IN_CI"
policyVersion: 17
policyDigest: "sha256:..."
promotionRuleId: "cache-low-risk-v3"
launcherVersion: "1.3.0"
pluginVersion: "1.3.0"
cacheNamespace: "tenant-7/repo-42/stable/linux-x86_64"
actor: "build-optimization"
metricDefinitionVersion: "build-impact-v1"
evidenceCount: 14
qualificationSource: "REVIEWED_ADAPTER"
measurementKind: "PAIRED_CONTROL"
effectScope: "ACTION_INCREMENTAL"
measurementRunId: "experiment-01K4Y6ZZ"
experimentResultRef: "experiment-01K4Y6ZZ/result/4"
pairCount: 18
intervalMethod: "PAIRED_BOOTSTRAP"
observedNetBuildTimeSavedInterval95Ms: [42100, 48900]
estimatedNetBuildTimeSavedMs: 47000
observedNetBuildTimeSavedMs: 45600
customerVisibleBuildP95DeltaMs: -43800
incrementalActionOverheadMs: 320
additionalValidationComputeMs: 512000
validation: "PASSED"
rollbackAvailable: true
```

`observedNetBuildTimeSavedMs` is populated only when a valid causal measurement exists, such as a paired control or comparable cohorts, and it already includes the incremental overhead of that action. For `effectScope=ACTION_INCREMENTAL`, the overhead field is named `incrementalActionOverheadMs`; `productSynchronousOverheadMs` is reserved for a `PRODUCT_TOTAL` result. A difference against a historical baseline is exported as `estimatedNetBuildTimeSavedMs`, not as observed savings. `additionalValidationComputeMs` is not subtracted from customer-visible latency, but it is subtracted from net economic value.

Contractual evidence and statistical uncertainty are recorded separately. An interval around the benefit can prioritize canaries, but it never replaces a contractual source or independently authorizes a corrective action; counters always publish population, sample, and method.

---

## 9. Functional pillars

### 9.1 Observability

Observability will collect:

- Distinct customer-visible end-to-end duration and Gradle process duration.
- Initialization, configuration, and execution time.
- Launcher, policy fetch/verification, gateway startup/verification, finalization, and any product-owned synchronous overhead.
- Exact critical path when capability permits, or an approximate labeled value otherwise.
- Task graph and project graph.
- Outcome per task: executed, `UP-TO-DATE`, `FROM-CACHE`, `NO-SOURCE`, `SKIPPED`, or failed.
- Local and remote cache hit rate.
- Time-weighted savings, not only task counts.
- Cache-miss reasons.
- Dependency resolution and remote calls.
- CPU, memory, GC, disk, and network.
- Requested versus effective parallelism.
- Time waiting for worker leases or shared resources.
- Volatile inputs and nondeterministic outputs.
- Applied actions and active policies.
- Context required for comparison: pipeline/runner class, requested tasks, workspace/daemon/cache state, change size/class, and work-units fingerprint.

Instrumentation will publish a capability matrix per build. Task outcomes and completion will use public APIs when available; critical path, detailed miss causes, and wait time will be marked `EXACT` only if the adapter for that version proves it. Otherwise, they will be exported as `APPROXIMATED` with a method or `UNAVAILABLE` with a reason.

The primary experience will not be a list of problems, but a ledger of actions and results:

```text
Build: 8m 42s → 4m 16s

Net causal savings: 4m 26s
Observed incremental attribution, following action-ledger order:
  Remote cache                    2m 04s
  Configuration Cache               38s
  Worker autotuning                 51s
  Removal of clean                  19s
  Caching generateOpenApi           34s

Not enabled:
  :signArtifact  task with external effects
  :legacyCodegen nondeterministic output
```

#### 9.1.1 Build measurement contract

The canonical grain is a **build session**: a request to the neutral measurement envelope with a stable manifest of requested work and deliverables. The product arm can pass through the Launcher, and each arm can contain one or more Gradle invocations—for example, before and after merging them; in the simple case, it coincides with one invocation. Primary KPIs are calculated at session level so that savings cannot be claimed merely by removing an invocation from the measurement.

The primary metric uses a monotonic clock and a boundary that includes the complete product:

| Canonical metric | Start | End | Use |
|---|---|---|---|
| `customerVisibleBuildMs` | Neutral envelope hands the session to the assigned arm | Exit code and required deliverables become available | Latency north star |
| `customerVisibleFeedbackMs` | Job becomes eligible | Exit code and required deliverables become available | End-to-end guardrail, including queue time caused or experienced |
| `gradleProcessMs` per invocation | Gradle process starts | Gradle process ends | Per-invocation diagnostics, not a primary KPI |
| `buildCriticalPathMs` | First critical-path node | Last required node | Locate serial work |
| `timeToFirstBuildFailureMs` | Neutral envelope hands over the session | First actionable build failure | Early feedback; not mixed with successful builds |
| `ciQueueMs` | Job becomes eligible | Runner starts | Per-build context; its causal change from control/canary consumption is attributed to the product per runner pool |
| `runnerOccupiedMs` | Runner is assigned to any product work | Runner is released | Capacity and cost, including asynchronous control/canary work |
| `testOwnedTaskExecutionMs` | Start of each task action owned by Test Optimization | End of that task action | Sum of work for ownership; may overlap and is not part of wall-clock decomposition |

`customerVisibleBuildMs` is decomposed without overlap into policy/launcher, Gradle startup+init, configuration, task execution/cache restore, and finalization. If processes or tasks run in parallel, the temporal union within the session is used: summing their wall times may exceed elapsed time and never replaces the north star. Subsequent asynchronous telemetry does not count toward latency, but its CPU, bytes, `runnerOccupiedMs`, and cost do count toward economic overhead. An explicit compliance gate is measured separately.

The authoritative clock lives in the neutral measurement envelope, outside the candidate launcher, and wraps product, native control, and policy-off in the same way. Assignment, eligibility, and `baselineDefinitionDigest` are persisted before the outcome. If arms share a cache, daemon, saturable runner pool, or another resource capable of causing interference, the experiment isolates them or is declared `INCONCLUSIVE`; a model must not silently correct it.

`job eligible` and `runner assigned` come from the authenticated CI adapter; they are not inferred from the Gradle process. If the first provider does not expose both timestamps with stable semantics, `customerVisibleFeedbackMs`/`ciQueueMs` remain `UNAVAILABLE` for that pipeline. The beta may validate `customerVisibleBuildMs`, but it cannot claim end-to-end queue impact and must respect the compute budget; GA-D requires an adapter that closes that guardrail.

The export retains `gradleInvocations[].processMs` and `gradleProcessUnionMs`; the latter is the union of intervals within the session, not the sum when parallel invocations exist.

Every build publishes:

- `metricDefinitionVersion`, unit, clock source, and state `COMPLETE | PARTIAL | UNAVAILABLE`.
- Non-negative timestamps and durations; the component sum includes an `unattributedMs` field and must reconcile with the total within a published tolerance.
- Outcome and class `SUCCESS | BUILD_FAILURE | INFRA_FAILURE | CANCELLED`; only comparable classes are aggregated together.
- The `workUnitsFingerprint` defined in 8.1, deliverables-manifest digest, and component invocations; treatment dimensions separately retain workspace/daemon/cache warmth, policy, change class, and trust domain.
- `effectScope`, cohort/candidate/control/action IDs, and control-definition digest; an explicit reason when a build is excluded from a causal measurement.
- Capability and method `EXACT | APPROXIMATED | UNAVAILABLE` for each metric family.

High-cardinality paths or labels are tokenized before export. Aggregations keep CI/local, success/failure, pipeline class, and runner class separate; no global average mixing incompatible workloads is published.

#### 9.1.2 Exportable observability

All information visible in the product must be extractable without depending on the proprietary UI. This includes:

- Summary and result of every build.
- Phases and critical path.
- Tasks, outcomes, fingerprints, and timings.
- Cache hits, misses, latency, and transferred bytes.
- Resources used.
- Actions applied, rejected, suspended, or rolled back.
- Evidence and contract used to qualify and validate an optimization.
- Shadow, canary, and rollout state.
- Estimated and observed savings.
- Policy version and digest, Launcher/plugin/JVM agent/hermetic helper versions, actor, namespace, and provenance accompanying every decision.

The canonical model will have a stable `schemaVersion`. Compatible changes will add optional fields; incompatible changes will create a new major version. IDs, timestamps, units, enums, and treatment of absent values will be formally defined.

##### Formats and destinations

The private beta will support:

- **JSON:** canonical `BUILD_SESSION`, `EXPERIMENT_RESULT`, and `ACTION_RECORD` documents.
- **JSONL:** one event per line for streaming or incremental processing.
- **Local file:** automatic export to a configurable workspace or Launcher path.

Later adapters may offer:

- Paginated HTTP API by repository, build, task, action, or time range.
- OpenTelemetry Protocol for traces, metrics, and logs.
- Prometheus for aggregated metrics.
- Object storage such as S3-compatible, GCS, or Azure Blob.
- Customer data warehouses and observability platforms.
- Signed webhooks for relevant states or actions, during hardening and with their own replay/SSRF/rotation contract.

The records have distinct lifecycles and are not mixed:

- `BUILD_SESSION` is immutable once the build completes. It contains facts from that session, pre-outcome experiment assignment, observed costs/resources, and model estimates; it never contains an aggregated causal effect that does not yet exist.
- `EXPERIMENT_RESULT` is calculated off the critical path over an explicit window and population. It is versioned, append-only, and carries `PRELIMINARY | FINAL | INVALIDATED`, `asOf`, samples, exclusions, method, and intervals.
- `ACTION_RECORD` references the evidence and `EXPERIMENT_RESULT` that authorized each transition. It does not rewrite the build record.

This separation prevents publishing cohort sizes or observed effects in `BUILD_FINISHED` that can only be known later.

##### `BUILD_SESSION` example

```json
{
  "schemaVersion": "2.0",
  "recordType": "BUILD_SESSION",
  "complete": true,
  "build": {
    "id": "build-01K4Y7B9",
    "repository": "payments-platform",
    "revision": "8c74f2a",
    "startedAt": "2026-07-21T10:14:02Z",
    "completedAt": "2026-07-21T10:18:18.340Z",
    "outcome": "SUCCESS",
    "policyVersion": 17,
    "policyDigest": "sha256:1a2b...",
    "configurationPolicyDigest": "sha256:9f8e...",
    "revocationEpoch": 42,
    "launcherVersion": "1.3.0",
    "pluginVersion": "1.3.0",
    "cacheNamespace": "tenant-7/repo-42/stable/linux-x86_64"
  },
  "gradleInvocations": [
    {
      "id": "gradle-invocation-1",
      "requestedTasks": ["assemble"],
      "processMs": 255925
    }
  ],
  "measurementMetadata": {
    "metricDefinitionVersion": "build-impact-v1",
    "status": "COMPLETE",
    "clockSource": "MONOTONIC",
    "envelopeVersion": "ci-provider-1-v1",
    "reconciliationToleranceMs": 5
  },
  "experimentAssignment": {
    "experimentId": "experiment-01K4Y6ZZ",
    "measurementEpoch": 3,
    "effectScope": "PRODUCT_TOTAL",
    "baselineDefinitionDigest": "sha256:6c1f...",
    "assignmentUnit": "CI_JOB",
    "arm": "CANDIDATE",
    "assignmentProbability": 0.5,
    "assignedAt": "2026-07-21T10:13:58Z",
    "eligibility": "ELIGIBLE",
    "exclusionReason": null
  },
  "performance": {
    "customerVisibleBuildMs": 256340,
    "customerVisibleFeedbackMs": 271340,
    "ciQueueMs": 15000,
    "runnerOccupiedMs": 256500,
    "gradleProcessUnionMs": 255925,
    "launcherAndPolicyMs": 210,
    "gatewayStartupMs": 85,
    "gradleStartupAndInitializationMs": 6495,
    "configurationMs": 18420,
    "executionOrRestoreMs": 230600,
    "finalizationMs": 410,
    "unattributedMs": 120,
    "buildCriticalPathMs": 228110,
    "timeToFirstBuildFailureMs": null,
    "testOwnedTaskExecutionMs": 78000
  },
  "model": {
    "version": "saving-model-12",
    "estimatedNetBuildTimeSavedMs": 267000,
    "estimatedNetBuildTimeSavedInterval95Ms": [219000, 306000]
  },
  "workload": {
    "environment": "CI",
    "pipelineClass": "pull-request",
    "runnerClass": "linux-amd64-4c-16g-v1",
    "workspaceState": "EPHEMERAL",
    "daemonState": "COLD",
    "cacheState": "WARM_REMOTE",
    "changeClass": "SMALL_IMPLEMENTATION",
    "sourceStateDigest": "hmac-sha256:91ac...",
    "testOptimizationPolicyDigest": "sha256:55de...",
    "requiredDeliverablesManifestDigest": "sha256:cf10...",
    "workUnitsFingerprint": "hmac-sha256:7a9d...",
    "tokenKeyVersion": "tenant-7-token-2026-q3"
  },
  "resources": {
    "cpuMs": 812000,
    "peakRssBytes": 6384218112,
    "gcPauseMs": 3290,
    "diskReadBytes": 921330176,
    "diskWriteBytes": 413200384,
    "networkBytes": 27122240
  },
  "observedCostInputs": {
    "grain": "BUILD_SESSION",
    "currency": "EUR",
    "pricingRegion": "eu-west",
    "priceModelVersion": "ci-price-2026-07",
    "productRuntimeCost": 0.01,
    "incrementalStorageCost": 0.004,
    "incrementalNetworkCost": 0.006
  },
  "cache": {
    "localHits": 12,
    "remoteHits": 31,
    "misses": 7,
    "timeWeightedHitRate": 0.72,
    "usefulHitRate": 0.68,
    "downloadedBytes": 18423312,
    "lookupMs": 820,
    "restoreMs": 6130,
    "storeMs": 940
  },
  "capabilities": {
    "taskOutcomes": "EXACT",
    "criticalPath": "APPROXIMATED",
    "cacheMissReasons": "UNAVAILABLE",
    "resourceUsage": "APPROXIMATED",
    "productOverhead": "APPROXIMATED",
    "costInputs": "APPROXIMATED"
  },
  "tasks": [
    {
      "path": ":openapi:generateClient",
      "outcome": "FROM_CACHE",
      "durationMs": 1420,
      "modeledAvoidedExecutionMs": 45600,
      "modeledAvoidanceVersion": "task-duration-v4"
    }
  ],
  "actions": [
    {
      "type": "ENABLE_TASK_CACHE",
      "actor": "build-optimization",
      "target": ":openapi:generateClient",
      "state": "ACTIVE_IN_CI",
      "validation": "PASSED",
      "rollbackAvailable": true
    }
  ]
}
```

##### `EXPERIMENT_RESULT` example

```json
{
  "schemaVersion": "2.0",
  "recordType": "EXPERIMENT_RESULT",
  "experimentId": "experiment-01K4Y6ZZ",
  "resultVersion": 4,
  "status": "FINAL",
  "asOf": "2026-07-28T00:00:00Z",
  "window": {
    "startedAt": "2026-07-21T00:00:00Z",
    "endedAt": "2026-07-28T00:00:00Z"
  },
  "effectScope": "PRODUCT_TOTAL",
  "measurementEpoch": 3,
  "baselineDefinitionDigest": "sha256:6c1f...",
  "analysisUnit": "BUILD_SESSION",
  "assignmentUnit": "CI_JOB",
  "analysisMethod": "STRATIFIED_CLUSTER_BOOTSTRAP",
  "effectStatistic": "STRATIFIED_MEAN_DELTA",
  "estimand": "SUCCESSFUL_BUILD_SESSION_LATENCY",
  "intervalType": "CONFIDENCE_95",
  "assignedCandidateSampleSize": 214,
  "assignedControlSampleSize": 206,
  "successfulLatencySample": {
    "candidate": 210,
    "control": 203
  },
  "outcomes": {
    "candidate": {"SUCCESS": 210, "BUILD_FAILURE": 2, "INFRA_FAILURE": 1, "CANCELLED": 1},
    "control": {"SUCCESS": 203, "BUILD_FAILURE": 1, "INFRA_FAILURE": 1, "CANCELLED": 1}
  },
  "excludedSampleSize": 9,
  "exclusions": {
    "PREDECLARED_INFRA_FAILURE": 7,
    "MISSING_WORK_UNITS_FINGERPRINT": 2
  },
  "measurementCoverageRatio": 0.979,
  "observedNetBuildTimeSavedMs": 254000,
  "observedBuildTimeReductionRatio": 0.4977,
  "observedBuildTimeReductionInterval95": [0.472, 0.521],
  "observedNetBuildTimeSavedInterval95Ms": [241000, 287000],
  "buildFailureRateDelta": 0.0045,
  "customerVisibleBuildP95DeltaMs": -313000,
  "customerVisibleBuildP95DeltaInterval95Ms": [-348000, -279000],
  "customerVisibleFeedbackP95DeltaMs": -302000,
  "ciQueueP95DeltaMs": 11000,
  "productSynchronousOverheadP95Ms": 142,
  "additionalRunnerOccupiedMsPerEligibleSession": 512000,
  "economics": {
    "grain": "PER_ELIGIBLE_SESSION",
    "currency": "EUR",
    "priceModelVersion": "ci-price-2026-07",
    "buildComputeCostAvoided": 0.84,
    "productRuntimeCost": 0.01,
    "validationAndControlComputeCost": 0.08,
    "incrementalStorageCost": 0.004,
    "incrementalNetworkCost": 0.006,
    "netInfrastructureValue": 0.74
  }
}
```

The latency estimand does not mix outcomes: it uses `SUCCESS` sessions, while the record retains all assigned outcomes by intention-to-treat and publishes failure/cancellation deltas separately. A conditional latency improvement cannot be promoted if the treatment raises failures above the guardrail.

##### Export profiles

The following profiles will control cost and sensitivity:

- `summary`: result, aggregate timings, cache, and savings.
- `tasks`: adds per-task detail and critical path.
- `evidence`: adds fingerprints, contractual qualification, and action ledger.
- `diagnostic`: adds deep, time-limited troubleshooting information.

By default, the profile will never include secrets or source contents. Sensitive paths, arguments, and variables are redacted or tokenized using keyed HMAC: the beta uses a deployment key and retains `tokenKeyVersion`; hardened adds per-tenant rotation and an overlap window. A plain hash vulnerable to dictionary attacks on low-entropy values is not used. Prometheus receives only aggregates with bounded-cardinality labels; never task paths, cache keys, or fingerprints as labels.

##### Operational behavior

- Export does not block or fail the build by default.
- Redaction is applied before the event is persisted in the bounded local buffer.
- JSONL uses **at-least-once** delivery. Every event includes a globally unique `eventId`, `buildId`, monotonically increasing per-build `sequence`, `occurredAt`, `emittedAt`, `schemaVersion`, and `idempotencyKey`.
- Consumers deduplicate by `eventId`. Per-build order within a stream is preserved while there is no retry; late events or events from different streams may arrive out of order and must be reordered using `buildId` and `sequence`.
- A destination failure causes bounded exponential retry with jitter, followed by an encrypted local spool or dead letter according to policy; no unbounded loop is maintained. The spool has a byte quota, TTL, minimum permissions, and a beta deployment key or hardened secret manager. When full, it first discards the oldest `diagnostic` events, emits a loss counter, and retains the final summary when possible; it never fills the build volume.
- The final `BUILD_SESSION` is published atomically after `BUILD_FINISHED` and indicates `complete: true`; a partial recovery indicates `complete: false` and the missing sequence ranges. A later `EXPERIMENT_RESULT` never mutates that document: it is linked by `experimentId`.
- An optional strict mode is a separate, explicitly authorized compliance gate with a maximum timeout and its own result. It can mark a delivery violation after Gradle closes, but it does not reinterpret a successful Gradle build or wait indefinitely.
- Every event has correlation IDs linking CI job, build, task, action, and cache object without using a secret as an identifier.
- Producers may add optional fields in a minor version; they do not remove or reinterpret fields. A major version uses a separate endpoint/topic or explicit negotiation and retains at least one published compatibility window.
- The customer can export both permitted raw data and already-calculated aggregates.
- In beta, export access is limited to the deployment/token and authorized profile. In hardened, the API and Export Gateway add per-tenant/repository/profile scopes, RBAC, rate limits, and an audit log; permission to query evidence does not imply permission to export it.
- Customer-controlled downstream destinations are outside our managed deletion boundary. The product revokes its own access, deletes its copies, and emits a deletion event/tombstone; the customer governs retention and deletion of its copies.
- HTTP destinations and webhooks introduced after the private beta require HTTPS, resolution and egress allowlists, SSRF protection, per-delivery signatures, rotation/replay protection, and a prohibition on redirects to an unapproved origin.

### 9.2 Cache acceleration

The platform will coordinate:

- Local build cache.
- Remote task-output cache.
- Dependency cache.
- Configuration Cache.
- Toolchain and Gradle distribution caches.
- Specific caches such as BuildKit, npm, or compiler caches through explicit adapters.
- Cache warming from trusted branches.
- Segmentation by platform compatibility.
- No shared-cache credentials for external PRs or untrusted builds; they use a disabled cache or an empty, isolated, disposable backend.
- Quarantine namespaces and the evidence store during qualification; no candidate writes to `stable`.

Complete `build/` directories will not be cached manually as a substitute for Gradle's build cache. Gradle must decide reuse per task and content key.

The shared dependency cache will be mounted read-only through Gradle's supported mechanism, and each runner will retain its own private writable cache. A writable `GRADLE_USER_HOME` will not be shared concurrently. Configuration Cache entries will not be transported through the Shared Cache Backend either.

### 9.3 Autonomous optimization engine

The engine will select actions compatible with the build and the customer's policy. It will not enable every option blindly because some are mutually exclusive, environment-dependent, or unsafe in poorly modeled builds.

### 9.4 Validation and guardrails

Every action will have:

- Preconditions.
- Risk level.
- Required evidence.
- Validation method.
- Rollout cohort.
- Minimum benefit threshold.
- Invalidation conditions.
- Fallback and rollback strategy.

---

## 10. Optimization catalog

### 10.1 Direct actions

These actions can be applied when an objective precondition has been demonstrated.

#### Remote build cache for allowlisted, officially cacheable tasks

- Configure `HttpBuildCache`.
- Read from authorized builds.
- Write only from trusted CI.
- Apply an exact type/implementation/plugin/version allowlist and verify that the instance adds no actions or behavior outside the contract. The `@CacheableTask` marker, a customer `cacheIf`, or a “built-in type” is not sufficient by itself for Shared admission.
- Apply `doNotCacheIf` to every task outside the allowlist. If an unknown artifact transform cannot be excluded through a public API, disable the managed Build Cache for the invocation; a disposable namespace exists only in an isolated candidate and does not control the customer's result. Do not promise nonexistent per-transform control.
- Also respect Gradle's effective decision: do not force tasks with overlapping outputs, unreviewed actions, or disabled caching.
- Proactively disable caching for `Test` and equivalent types in mixed invocations unless a current Test Optimization grant exists.
- Place a local cache in front of the remote cache.
- Temporarily omit remote-cache configuration in later invocations when the repository/runner circuit breaker is open because latency exceeds avoided work. Within a build, respect Gradle's native fail-open behavior for errors; do not promise a public per-task switch that Gradle does not provide.

#### Persistent dependency cache

- Restore a compatible cache into the runner's private `GRADLE_USER_HOME`.
- To share across runners, mount [`GRADLE_RO_DEP_CACHE`](https://docs.gradle.org/current/userguide/dependency_caching.html) read-only and maintain a private writable cache per build. Because this capability is incubating in Gradle, the adapter enables it only for tested combinations and falls back to a private cache when unavailable.
- Reuse Gradle distributions and toolchains.
- Verify checksums/provenance of distributions and toolchains before reuse; these caches do not inherit trust from the task-output cache.
- Avoid invalidations caused by excessively specific CI keys.
- Segment metadata by compatible Gradle format/version and keep artifacts consistent with repository dependency verification/locking.
- Do not share or restore lock files, daemons, or a concurrently writable `GRADLE_USER_HOME`.

#### Safe removal of `clean`

- If the CI launcher created and verified an empty workspace, prior physical cleanup is redundant.
- A `./gradlew clean` is omitted only if it matches the allowlisted core task, deletes declared outputs exclusively, and has no actions, dependencies, finalizers, or side effects added by build logic/plugins.
- In a persistent workspace, retaining outputs enables `UP-TO-DATE` and incremental behavior, but `clean` is not removed until the workspace lifecycle is also proven to prevent stale outputs and the pipeline is shown not to use it as a barrier.
- `clean` will not be removed when it forms part of an explicit release contract or reproducibility validation.
- For a customized task or unavailable model, the original command is preserved.

#### Safe cache-write policy

- External PRs or fork builds receive no shared-cache credentials, not even read-only credentials; they use a disabled cache or an empty, isolated, disposable backend.
- Internal branches: read and write according to trust.
- `main`: primary producer with an authenticated identity in beta or attested identity in hardened.
- Candidate/canary: `push=false` toward stable and writes exclusively to quarantine.
- Release: no experiments; it may read only stable entries whose provenance/attestation, policy digest, platform, and origin satisfy its explicit policy.

### 10.2 Proof-gated actions

#### Merging Gradle invocations

When semantically compatible, transform:

```bash
./gradlew compileJava
./gradlew assemble
```

into:

```bash
./gradlew assemble
```

This avoids repeated initialization, configuration, and resolution, but it is applied only when **all** of the following conditions hold and an isolated control validates the pipeline class:

- Same repository, commit, working directory, Gradle Wrapper, JVM/JDK, JVM arguments, and compatible `GRADLE_USER_HOME`.
- Same Gradle properties, system properties, relevant environment, credentials, init scripts, and cache policy/namespace.
- The second invocation transitively contains all work requested by the first, proven by an entrypoint contract or a versioned Gradle model; matching names are insufficient.
- No intermediate consumer reads artifacts, logs, or effects from the first invocation before the second begins.
- The merged tasks have no external effects, and merging preserves order, failure semantics, retry, `continue`, exclusions, and finalizers.
- There is no explicit CI barrier between invocations, such as a credential, workspace, or container change, or artifact publication.

If any condition cannot be proven or the control diverges, the original invocations are preserved.

#### Configuration Cache

- Detect incompatibilities.
- Enable initially in noncritical CI builds.
- Promote after correct natural hits.
- Disable and invalidate the policy after an attributable failure.
- Do not retain warning mode permanently as a substitute for compatibility.
- Distribute only the enablement decision; each workspace creates its own local entries.

#### Worker, heap, and compilation autotuning

The engine will select:

- `org.gradle.workers.max`.
- Gradle heap.
- Heap and number of compilation processes.
- Use of forked compilation for sufficiently large source sets.
- Concurrency limits for memory-, disk-, or network-intensive tasks.
- Configuration by runner class and build type.

#### Parallelism

- Enable parallel execution only after observing dependencies and shared resources.
- Detect tasks writing to overlapping outputs.
- Limit concurrency for tasks using databases, ports, or global directories.
- Promote configuration through a canary.

#### Contractual qualification of custom tasks

- Observe inputs and outputs to propose a contract without assuming observation is exhaustive.
- Require an official contract, reviewed adapter, or source patch; alternatively, maintain a fail-closed hermetic profile on every producer writing those outputs.
- Register the approved manifest as real Gradle inputs and outputs before producing a reusable entry.
- Reject enablement if network, time, randomness, child processes, or external state are neither declared nor blocked, or if a separate repeatability contract is absent.
- Use cross-workspace comparison as a regression test, not as proof that no hidden inputs exist.

#### Project-level Build Impact Analysis

- Calculate affected projects and artifacts.
- Ask Gradle only for the required entrypoints.
- Authorize omissions only from a customer-owned deliverables manifest and Gradle's declared graph.
- Use conservative rules for global or unknown changes.
- Validate first in shadow mode.

#### Reproducible outputs

For compatible archives:

```kotlin
tasks.withType<AbstractArchiveTask>().configureEach {
    isPreserveFileTimestamps = false
    isReproducibleFileOrder = true
}
```

Signed artifacts, or artifacts whose contract requires specific timestamps or ordering, will be excluded.

#### Repository optimization

- Prioritize repositories that actually contain the requested groups.
- Apply content filters when a known ownership relationship exists.
- Eliminate lookups in repositories that never serve artifacts.
- Use a proxy close to the runner.
- Avoid resolution during the configuration phase.

Changing the origin of a dependency may have security and reproducibility implications, so this optimization requires explicit validation and policy.

### 10.3 Automatic build-logic changes

#### `api` to `implementation`

Analyze whether a dependency appears in the public ABI:

- Supertypes.
- Public signatures.
- Generic types.
- Public fields and constants.
- Relevant annotations.
- Metadata consumed by other compilers.

If it is not exposed and every consumer still compiles, change:

```kotlin
api("com.example:internal-helper:1.0")
```

to:

```kotlin
implementation("com.example:internal-helper:1.0")
```

This reduces transitive recompilation, but it [also changes consumers' compile classpath and published metadata](https://docs.gradle.org/current/userguide/java_library_plugin.html). Therefore:

- It can be applied autonomously only to applications or internal modules marked as unpublished.
- A published library requires explicit approval, API ownership, POM/Gradle Module Metadata validation, compilation of representative consumers, and a compatibility/semver policy.
- If publication status cannot be proven, the action is limited to a recommendation or a PR for review.

#### Eager to lazy build logic

Possible transformations:

```kotlin
tasks.create("generate")
tasks.getByName("compileJava")
```

to:

```kotlin
tasks.register("generate")
tasks.named("compileJava")
```

Also:

- Replace `all` with `configureEach` when correct.
- Avoid materializing providers during configuration.
- Defer dependency resolution until execution.
- Apply plugins only where needed.
- Move shared conventions to compiled plugins.

#### Annotation processors

- Move processors out of `compileClasspath`.
- Separate annotations required by code from the executable processor.
- Use `annotationProcessor`, `kapt`, or an equivalent mechanism correctly.
- Detect non-incremental processors.
- Declare internal processors as `isolating` or `aggregating` only when their implementation satisfies and validates the corresponding contract.

#### Task inputs and outputs

- Add omitted inputs/outputs through the runtime API only additively and before the key is calculated.
- Do not attempt to overwrite existing file inputs/outputs through the runtime API: [Gradle does not support it](https://docs.gradle.org/current/userguide/build_cache.html#sec:enable_caching_of_non_cacheable_tasks). An incorrect annotation, output, or path sensitivity requires a corrected plugin version, a source patch, or a reviewed override of the getter in a subclass.
- Remove or transform inputs only through a reviewed contract and implementation change; historical correlation or an isolated sandbox alone does not authorize it.
- Separate overlapping outputs.
- Convert monolithic tasks into incremental tasks when a safe input-to-output mapping exists.

#### Dependencies and resolution

- Avoid dynamic versions when dependency locking exists and policy permits.
- Remove unused dependencies from the compile classpath after static analysis and validation.
- Centralize repositories.
- Avoid resolution logic that performs network or intensive work during configuration.

### 10.4 Actions that are not initially automatable

- Signing.
- Publishing with external effects.
- Deployments.
- Migrations.
- Creation or deletion of remote resources.
- Tasks whose output deliberately depends on time or randomness.
- Tasks that consume state from an unmodeled external service.
- Dependency-version changes without an explicit policy.
- Forced caching enablement without complete inputs and outputs.

### 10.5 Initial enablement matrix

| Optimization | Initial enablement | Primary evidence | Fallback |
|---|---|---|---|
| Remote cache for allowlisted native tasks | Immediate in trusted CI, except tests without a grant or unknown transforms | Exact plugin/type/implementation allowlist, unmodified instance, ownership grant when applicable, and authenticated cache | Managed Build Cache off for the entire invocation if a transform cannot be isolated |
| Dependency cache | Immediate | Gradle, repository, and metadata compatibility | Resolve and download normally |
| Remove `clean` | Direct only in a new workspace with an allowlisted task/cleanup and no side effects | Runner lifecycle and exact `clean` contract | Restore original command |
| Merge invocations | CI canary after complete whitelist | Transitive subsumption, environment/semantic equivalence, and isolated control | Run original invocations |
| Configuration Cache | CI canary | Compatibility and correct natural hits | `--no-configuration-cache` |
| Parallelism | CI canary | Absence of conflicts and critical-path improvement | Last stable configuration |
| Workers and heap | Bounded autotuning | Contextual model of natural builds | Conservative profile per runner |
| Cache custom task | Isolated canary after an executable contract | Official contract, reviewed adapter/source patch, or continuous producer enforcement; separate determinism and real registered inputs | Run without cache |
| Build Impact Analysis | Shadow and internal CI canary | Customer-owned manifest and declared graph; history only estimates | Request the complete graph |
| Reproducible archives | CI canary | Semantic equivalence and binary stability | Original archive configuration |
| `api` to `implementation` | Patch canary; autonomous only in an unpublished module | ABI, published metadata, and consumer compilation | Revert the patch |
| Eager to lazy build logic | Patch canary | Equivalent configuration, execution, and artifacts | Revert the patch |
| Incremental annotation processor | High-risk patch canary | Processor contract and incremental scenarios | Original configuration |
| Repository optimization | Canary per environment | Same resolved graph and authorized provenance | Original repository order and set |

---

## 11. Learning without duplicate manual builds

Developers will not be asked to run the same build twice. The system will accumulate evidence from natural executions.

The Launcher will automatically assign a budgeted fraction of comparable CI builds to candidate or control. High-risk actions may trigger an additional isolated paired control in CI, never a manual repetition on the workstation. Local builds are not duplicated: they apply already-validated policies, contribute permitted natural observations, and label their impact as estimated until a CI counterfactual or enough comparable local cohorts are available.

### 11.1 Fundamental constraint

It is impossible to prove that an arbitrary task has no hidden inputs through a finite number of executions. Natural repetitions and additional CI executions provide regression and benefit evidence, but they are not sufficient authority for enabling caching.

Autonomous enablement requires a contractual source and an independent repeatability gate:

1. The task is already officially cacheable by Gradle or by a plugin version in our verified allowlist, and the instance adds no actions or behavior not covered by its effective inputs.
2. An explicit reviewed adapter declares inputs, outputs, path sensitivity, environment, and constraints for a specific implementation hash.
3. A repository patch corrects the task declarations and passes the transactional pipeline, with approval when applicable.
4. As an alternative route, a supported `HERMETIC_PRODUCER_PROFILE` maintains a fail-closed policy over filesystem, network, processes, and external sources on **every** execution authorized to produce that entry. This route requires the task's work to execute in a dedicated producer with task-specific manifest and mounts, and an unambiguous relationship among task, process tree, and outputs. Every descendant remains within the published boundary, the profile digest enters the contract, and an unmediated surface invalidates the route. A sandbox around the entire Gradle invocation, or an isolated execution that later returns to `TRACE_OBSERVE`, produces evidence only for routes 2 or 3; it does not qualify an in-process task.

Each route must also demonstrate that the same input set produces byte-repeatable outputs, or outputs normalized before storage, and that the result is relocatable. Repeatability authority is an official guarantee or a versioned, reviewed deterministic implementation/normalizer; repeated builds are a regression test, not the authority. The gateway treats payloads as opaque and never normalizes an output after Gradle has snapshotted it: normalization occurs in the task/plugin/source patch before the snapshot. Stable instrumentation does not add task actions to allowlisted instances. Incomplete tracing can produce only a recommendation or an adapter draft. If the customer does not permit additional compute or sandboxing, a high-risk action learns more slowly or remains disabled; the proof standard is never lowered.

### 11.2 State machines

Task contract and action rollout are separate objects. A task may remain qualified while an action is rolled back for performance, or lose its contract and suspend all dependent actions.

```text
TaskQualificationState

UNKNOWN → OBSERVING → CONTRACT_QUALIFIED → QUARANTINE_VALIDATED
   └── unsupported/unsafe → REJECTED

Any state ──contract drift/incomplete evidence──► SUSPENDED
SUSPENDED ──new contract + complete revalidation──► OBSERVING

ActionRolloutState

PROPOSED → SHADOW → CI_CANARY → ACTIVE_IN_CI → ACTIVE_LOCALLY

Any state ──regression/policy violation──► SUSPENDED
CI_CANARY/ACTIVE_* ──rollback──► ROLLED_BACK
```

`INCONCLUSIVE` is a gate result and never a positive transition. Promoting an action that depends on a task requires `TaskQualificationState=QUARANTINE_VALIDATED`; suspending that contract atomically suspends dependent actions. A discrepancy is a safety transition, not an error in the normal build: the observed task action may finish with its original behavior, but it authorizes no new cache lookups and does not promote outputs from that execution.

### 11.3 Invalidation key

A qualification expires when any of these relevant elements changes:

- Gradle version.
- Plugin version or classpath.
- Task implementation.
- Applicable build logic.
- Toolchain.
- Input/output schema.
- Adapter or hermetic-profile digest and version.
- Output/enforcement contract version of the plugin, adapter, JVM agent, or hermetic helper; a purely observational version does not invalidate the task.
- Platform, if the output is not portable.
- Security policy, trust domain, or namespace.

Invalidation is not required for every source change: inputs are part of the reuse key. Qualification is invalidated when the contract used to interpret those inputs changes.

### 11.4 Discrepancies and cache invalidation

When `TRACE_OBSERVE` detects a read, write, process, network access, or nondeterminism source not included in the contract:

1. It allows the task to continue in the normal build and records redacted evidence.
2. It transitions qualification to `SUSPENDED` for the minimum safe selector—instance, implementation, or adapter—increments `revocationEpoch`, and publishes a new authenticated policy.
3. It disables cache reads and writes for the task before the next invocation through [`TaskOutputs.doNotCacheIf`](https://docs.gradle.org/current/javadoc/org/gradle/api/tasks/TaskOutputs.html) or the equivalent public mechanism for the supported version. The new `configurationPolicyDigest` invalidates any Configuration Cache entry retaining the prior decision.
4. It creates no synthetic key or alias from the incomplete key and does not retain the payload as a reusable cache.
5. It tombstones remote keys when an exact association exists. Otherwise, it revokes the `cacheContractDigest` or rotates its namespace; waiting only for TTL is insufficient.

Purging is not done by task path: Gradle may reuse the same key across semantically equivalent tasks, and the path alone does not identify the object. When the contract is corrected, the plugin registers the new `TaskInputs` before the first lookup, and Gradle computes a different native key with the corresponding content hashes and path sensitivity. The `cacheContractDigest` also differentiates semantic manifest versions.

To close the window between PUT and verdict, every instrumented execution capable of producing custom-task outputs uses a fresh disposable L1 and writes to L2 exclusively at `pending/{attemptId}` through the gateway. Only `traceComplete=true`, no discrepancies, and an exact task-to-key correlation allow `commit`; if any is missing, all writes for the attempt are aborted and L1 is discarded. `doNotCacheIf` is a defense for the next invocation, not a mechanism for withdrawing the current PUT. A simpler mode uses `push=false` and later rebuilds with a trusted writer. Pending is not a second key and is not promoted through aliases.

The gateway checks the revocation epoch/denylist on every remote GET. CI requires a fresh online policy; local persists the highest authenticated epoch seen to prevent rollback and uses last-known-good only up to the maximum TTL for its risk class. When it expires without a refresh, it disables any read that may have been populated from Shared and rotates its `l1SecurityGeneration`; protection is not limited to custom tasks. `doNotCacheIf` also prevents the contractual selector from restoring a suspended task. A broader native-only L1 requires evidence that it was never populated remotely.

A task with outcome `FROM-CACHE` does not execute its code and therefore generates no new access events. The tracker cannot discover a hidden dependency on that hit. Consequently, an unqualified or `SUSPENDED` task never reads from stable, and active contracts are periodically audited through an isolated cohort with cache reads disabled and a disposable L1. Audits detect drift, but they do not replace the contractual authority that permits the first hit.

---

## 12. Shadow mode, canaries, and rollout

### 12.1 Evidence store and isolation namespaces

For an unqualified custom task:

1. The task runs normally, without caching added by us.
2. An observational fingerprint is calculated.
3. If policy permits, the evidence store retains the observed manifest, access classifications, and an output digest; it does not retain a restorable payload or alternative cache key.
4. When the fingerprint reappears, the task executes again and the digests are compared. Complete artifacts are retained only within an isolated contractual validation when a semantic comparison requires them.
5. Matches create a contract candidate; they do not enable caching by themselves.

Example:

```text
Build 101: fingerprint F42 → output A
Build 117: fingerprint F42 → output A
Build 126: fingerprint F42 → output A, different workspace
```

Only after a contractual source from section 11.1 is also approved is quarantine validation considered.

Quarantine after an approved contract may contain cacheable outputs, but always under the native key Gradle calculates after all inputs are registered. This validation quarantine must not be confused with discovery: discovery retains evidence; validation retains, when necessary, isolated artifacts that no normal build can read.

Every variant uses explicit isolation:

| Variant | Workspace | Read | Write |
|---|---|---|---|
| Stable | Normal trusted workspace | `stable/{platform}/{cacheCompatibilityClass}` | Trusted writer only |
| Candidate/canary | New workspace or container | Stable read-only only if the action is not validating cache; quarantine namespace if it is | `quarantine/{actionId}/{attempt}`; never stable |
| Action control | New independent workspace or container | `control/action/{policyDigest}` seeded only with stable objects of compatible provenance/attestation, or cache disabled | `push=false` |
| Product baseline | New independent workspace or container | `control/product/{baselineDefinitionDigest}` with the versioned pre-product configuration; cache state fixed by the experiment | `push=false` |
| External fork/PR | Untrusted runner | No shared credentials | Empty disposable backend or cache disabled |

Candidate and control do not share workspace, outputs, writable `GRADLE_USER_HOME`, Configuration Cache, write credentials, or namespace. For `ACTION_INCREMENTAL`, control uses the same stable policy except for the evaluated action. For `PRODUCT_TOTAL`, baseline uses the pre-product configuration captured during onboarding; it is never retrospectively redefined as “the latest stable policy.” A real customer change opens another `measurementEpoch`.

In a paired control, both use the same `WorkUnitsFingerprint`, including the Test Optimization grant/policy, and differ only in treatment and prescribed isolated state. If that equivalence cannot be maintained, or cache/daemon/capacity interference exists between arms, the result is classified as a contextual estimate or `INCONCLUSIVE`, not as causal validation.

`cacheCompatibilityClass` changes only when a boundary capable of making objects incompatible changes—for example, platform, format, trust domain, or normalization rules—not for every worker or UI adjustment. Provenance/attestation retains the exact policy digest and producer source revision as origin; the revision need not match the reader's to allow cross-commit reuse. The reading policy decides which contracts, identities, and digests are compatible.

For a high-correction action, candidate and control first run at least one cold validation with Build Cache disabled or empty namespaces. Cache-enabled measurement occurs in a separate execution; a hit cannot replace proof that both paths produce the deliverables.

### 12.2 Canary reads

A small internal CI cohort consumes output from the quarantine namespace after the real contract is registered with Gradle. Assignment will be deterministic to aid reproduction and auditing.

The canary will verify:

- Build result.
- Downstream artifacts.
- No missing outputs.
- No unauthorized differences.
- Net savings after transfer cost.
- Producer provenance/attestation according to profile, policy/cache-contract digests, component versions, and expected namespace.

Passing the canary does not blindly copy bytes from quarantine to stable. Promotion creates an authenticated policy and a trusted writer rebuilds the entry. In hardened, an attestation authority separate from the data plane may instead sign the promotion after verifying the flow; Shared/Edge do not sign their own evidence. Every stable entry retains the provenance chain corresponding to its profile.

### 12.3 Progressive rollout

The private beta uses explicit profiles instead of a universal percentage:

| Class | Candidate stages | Stable control | Constraint |
|---|---|---|---|
| Direct/reversible | 5% → 25% → 50% → 95% | 5% | May become customer-visible after the smoke gate; immediate rollback |
| Proof-gated | shadow → 1% → 5% → 25% → 50% → 95% | 5% + periodic cache-off audit | Does not enter a visible canary until its contract and quarantine are complete |
| Patch/high correction | paired candidate + control within budget → PR | Periodic isolated control applying the inverse patch when valid | No partial rollout exists after merge unless the patch has a contractual switch; candidate does not determine exit code/deliverables |

The selector may stop at any stage until it reaches sample or budget, and it never skips a correctness gate to consume more traffic. The stable 5% remains randomized within comparable cohorts to measure drift; releases and pipelines with external effects are excluded. During hardening, these defaults may become configurable per tenant, but reducing control requires demonstrating equivalent power through periodic revalidation.

### 12.4 Automatic fallback

For a candidate variant:

```text
Run isolated candidate
   ├─ success → record evidence; run control if the promotion gate requires it
   └─ failure → run baseline automatically
                 ├─ baseline passes → attribute to candidate and suspend
                 └─ baseline fails → failure probably not attributable
```

The second execution occurs inside CI without developer intervention. For high-correction actions—custom-task caching, graph omission, or patches—the gate includes at least one isolated paired control even when candidate passes. For low-risk autotuning, a randomized control cohort may replace repetition of the same build.

Fail-open has an explicit operational boundary. If Launcher/policy/plugin fails before any task action begins, the Launcher may retry the immutable original command **once** without the product. After task actions begin, the same workspace is not blindly repeated: a task may have published or mutated external state. At that point, only a baseline already running in isolation, or whose manifest guarantees no effects, may be returned; otherwise, the real failure is preserved and the kill switch is enabled for subsequent builds. Cache, telemetry, and policy fetch remain fail-open, but the product does not promise that an arbitrary Launcher/plugin bug cannot fail a build.

During qualification and the first canary for custom-task caching, Build Impact Analysis, or patches, control/baseline is always the authoritative source for exit code and deliverables. Candidate is withheld until artifact comparison and, when applicable, `FULL_RELEVANT_VALIDATION` complete; a zero exit code is insufficient. Visible candidate-first is allowed only for an already-qualified, low-risk, reversible runtime action. If an optimizer-introduced action fails, the Launcher runs or preserves baseline; a validation-sandbox failure is recorded as evidence and never independently turns a correct baseline into a pipeline failure.

### 12.5 Control builds

Complete `main`, merge-queue, or nightly builds act as a safety net for actions such as Build Impact Analysis, but they do not replace the first control required to promote an omission rule. Every PR need not be duplicated: after promotion, budgeted sampling and control cohorts are used.

### 12.6 Beta learning budget

A natural assignment to baseline instead of candidate does not duplicate compute and does not count as additional compute, although its savings opportunity is reported. Shadow, paired candidate/control, and patch validation do consume budget:

- Default per repository: at most 5% of eligible natural runner-minutes in a rolling seven-day window.
- Burst: at most 10% in any 24-hour window, without borrowing from future weeks.
- Only one concurrent additional validation per repository unless the pilot explicitly opts in.
- When the budget is exhausted, the action remains `SHADOW`, `CI_CANARY`, or `INCONCLUSIVE`; the gate is never reduced and repetition is never moved to the developer's computer.
- A pilot may reduce the budget to zero. Exceeding these maxima requires specific authorization and is recorded in the action ledger.

---

## 13. Contractual task qualification

### 13.1 Discovery

Classify each task as:

- Cacheable by contract.
- Cacheable through a reviewed adapter.
- Eligible only under a continuous `HERMETIC_PRODUCER_PROFILE`.
- Incremental.
- Up-to-date only.
- Justifiably non-cacheable.
- Unmodeled.
- Potentially nondeterministic.
- With external effects.

### 13.2 Instrumentation

For candidate tasks, observe the following when technically possible:

- File reads and writes.
- Child processes.
- Environment variables accessed.
- System properties.
- Network access.
- Time, locale, and timezone.
- Sources of randomness.
- Paths outside the workspace.
- Relevant permissions and metadata.

Instrumentation may combine Gradle APIs, the optional JVM Instrumentation Agent, process wrappers, and the experimental Rust Linux Hermetic Helper. It must have an overhead budget and support automatic disablement. Every backend publishes coverage and a `traceComplete` signal; if it does not intercept child processes, native code, network, or filesystem with closed coverage, its results are not used as proof of hermeticity. A crash, overflow, or dropped events make the verdict `INCONCLUSIVE` even when the build continues. The implementation language does not confer this property: the guarantee comes from continuous enforcement and verified coverage.

In normal builds, all backends operate in `TRACE_OBSERVE` mode: an undeclared access is allowed, recorded, and suspends the optimization. `ENFORCE_ISOLATED` is enabled only in a separate candidate/producer with its own workspace, outputs, `GRADLE_USER_HOME`, credentials, and namespace. The system does not attempt to switch from deny to allow halfway through a task action or rerun an arbitrary task within the same Gradle process after partial effects.

### 13.3 Contract proposal and approval

Observation may propose a manifest:

```text
Task: :frontend:bundle

Inputs:
  frontend/src/**
  package.json
  package-lock.json
  NODE_VERSION

Outputs:
  build/frontend/**

Properties:
  repeatabilityContract: "adapter-guaranteed-v2"
  relocatable: true
  network-independent: true
  producerMode: "HERMETIC_ENFORCED"
```

The manifest also includes path sensitivity, normalization, relevant permissions, toolchains, allowed processes, and adapter version. It becomes a contract only through one of these routes:

- It maps exactly to an officially cacheable task/plugin and an allowlisted version.
- A versioned adapter passes code review, negative fixtures, and mutation tests for every input.
- A reviewed hermetic profile denies every unlisted access in candidate and every later trusted writer. An undeclared attempt invalidates that producer, aborts pending, and preserves baseline as the visible result.
- The customer accepts a patch that adds the declarations to its build logic and passes validation.

Equivalent occurrences and different workspaces remain useful tests, but they do not confirm that the observed set is exhaustive.

### 13.4 Promotion

When an approved contract exists and evidence passes the gate:

1. Persist the descriptor by implementation hash, plugin version, platform, and contract digest.
2. Before the first cache lookup, register with Gradle all inputs, outputs, properties, environment providers, and path sensitivities that can be added through public APIs. For files and directories, Gradle calculates content hashes and normalized paths; its algorithm is not replaced with a proprietary key. `cacheContractDigest` and `outputSemanticsVersion` participate only when they can change the output; `policyDigest` and observational versions remain in configuration/attestation.
3. During configuration, verify that there is no conflict with build declarations and that every path resolves within the permitted trust domain. Because the runtime API does not overwrite existing file inputs/outputs, any annotation, output, or path-sensitivity conflict blocks the adapter and requires a reviewed plugin/source patch/subclass.
4. Enable caching through Gradle's supported API only in the quarantine namespace.
5. Run the paired control and canary; then promote through reconstruction or attestation, not by copying an opaque entry without provenance.
6. On hermetic routes, run every producer under the same profile digest; on other routes, maintain bounded observation. Every instrumented write remains pending until a complete verdict.
7. Continuously monitor digests, new accesses, isolated denials, mismatches, and regressions.

If the manifest cannot be registered as real Gradle inputs before the first cache lookup, caching is not enabled for the task.

### 13.5 Automatic repair of a discrepant contract

Example for a hidden file:

```text
TRACE_OBSERVE detects generator.yaml
  → contract-v3 becomes SUSPENDED
  → caching is disabled
  → candidate adapter registers generator.yaml as TaskInputs.file
  → contract-v4 and its digest participate in configuration
  → Gradle calculates the native key with path + content hash
  → control and canary validate v4 in quarantine
  → an authenticated policy enables caching again
```

The optimizer persists the resource's logical identity, classification, and proposed path sensitivity; it does not need to retain an alternative cache payload. Environment variables are registered as properties or opaque versions; secrets use a managed version or keyed digest, never the value or a plain low-entropy hash. Network, clock, and randomness become inputs only if a stable contractual representation exists; otherwise, the task remains non-cacheable.

### 13.6 Rejection causes

- Unmodeled network access.
- Writes outside declared outputs.
- Different output for the same fingerprint.
- Dependence on time or randomness.
- Use of secrets as an unmanaged input.
- External effects.
- Incomplete sandbox or tracer coverage presented as proof.
- Observed manifest that cannot be registered as a Gradle contract.
- Cache transfer cost exceeds the cost of running the task.

---

## 14. Resource autotuning

### 14.1 Problem

A static configuration is not optimal for every repository, runner, or change type. More workers may reduce duration or worsen it through GC, memory, disk, or contention.

### 14.2 Contextual model

Context will include:

- Runner CPU and memory.
- Cgroup limits.
- JDK type and version.
- Number of projects and tasks.
- Volume of modified source.
- Cache hit rate.
- Source-set sizes.
- Proportion of CPU-, memory-, disk-, or network-bound tasks.
- Critical-path shape.

### 14.3 Tunable parameters

- Maximum workers.
- Gradle heap.
- Stable JVM arguments.
- Compiler forking.
- Compiler heap.
- Heavy-task concurrency.
- Number of simultaneous downloads or external processes when the adapter permits.

The Launcher materializes workers, JDK, heap, and JVM arguments before selecting/starting the daemon. The plugin configures only compiler heaps/forking and task limits that public APIs allow it to change during configuration.

### 14.4 Learning

The beta begins with A/A tests and fixed-probability randomized cohorts until timestamps, reward, sample ratio, and absence of contamination are validated. Only then does it enable a contextual bandit with safe bounds. Every assignment records arm, prior context, probability/propensity before outcome, policy version, and late outcome; it retains at least the control from section 12.3 and chooses only among profiles previously declared safe. A/A and sample-ratio checks are evaluated against expected assignment, not necessarily 50/50, and adaptive analysis uses approved propensity-aware/off-policy estimators.

The bandit is part of the private beta because it turns natural observations into a configuration choice, but its authority is deliberately limited: it ranks reversible resources—for example, compatible worker/heap settings—and decides how much to explore within the compute budget. It never declares a task cacheable, approves a patch, removes an entrypoint, or replaces correctness gates. If propensities are missing, an A/A test fails, or reward arrives incomplete, it returns to fixed assignment and the experiment becomes `INCONCLUSIVE`.

Exploratory assignment is randomized within equivalent cohorts and records cache warmth, cold/warm daemon/JIT state, filesystem cache, runner exclusivity, and noisy-neighbor signals. A sample with a noncomparable revision change, throttling, or contamination is excluded only under predeclared criteria; a treatment-caused failure remains by intention-to-treat. Changes require declared repetitions/intervals; raw timings from different builds are not compared, and an improvement is not attributed to a single observation.

Example:

```text
Runner: 4 CPU / 16 GB
Large build, low cache hit rate

2 workers / 3 GB → 6m 10s
3 workers / 4 GB → 4m 48s
4 workers / 6 GB → 5m 21s

Learned policy:
  large build → 3 workers / 4 GB
  small build → 2 workers / 3 GB
```

#### 14.4.1 Beta bandit contract

**Accepted decision — BANDIT-001:** the beta will use contextual epsilon-greedy, not a continuous optimizer. Each `runnerClass + buildClass + compatibilityClass` receives a finite, versioned catalog of `RESOURCE_PROFILE` values; arbitrary values are not synthesized. A profile declares JDK, workers, Gradle/compiler heap, process limits, and cgroup limits, and must pass startup, memory, and rollback fixtures before becoming eligible.

The initial catalog for the 4-vCPU/16-GiB development golden runner contains exactly four arms: `STABLE_CONTROL`, `W2_H3G`, `W3_H4G`, and `W4_H6G`. `STABLE_CONTROL` is the stable resource profile before that experiment—it may match the customer's starting configuration, but it does not redefine the `PRODUCT_TOTAL` baseline. Only `--max-workers` and daemon heap change; JDK, compiler forking/heaps, and all other flags remain fixed to isolate the effect. An arm whose cgroup headroom does not satisfy the declared minimum is removed before assignment and retains zero propensity. Another runner class requires another catalog and A/A test; these values are not scaled through an implicit formula.

The pre-outcome feature vector is fixed by schema: runner CPU/memory and limits, repository/task-graph/change classes, toolchain, historical/predicted hit-rate class, observable warmth before the build, daemon/JIT state, and available contention signals. It does not contain actual hits from that execution, final duration, result, or any other future information. Assignment records the vector, catalog, arm, propensity, and seed before Gradle starts.

To keep it auditable, the beta discretizes the vector into versioned buckets and does not generalize across buckets. Within each bucket, the predictor is the trimmed mean reward per arm with shrinkage toward `STABLE_CONTROL`; the greedy arm is the eligible one with the best prediction, and a tie retains control. No neural network or opaque online model is introduced. Changing buckets, estimator, or window creates another `banditPolicyVersion` and forces A/A.

The sequence is:

1. A/A and fixed cohorts with the conservative profile verify instrumentation and sample ratio.
2. Every new arm receives a bounded fixed cohort; it does not enter the bandit until the minimum valid outcomes defined by `bandit-policy.v1` exist.
3. Epsilon-greedy begins with at most 10% exploration, may decrease to 2%, and also retains the 5% stable control. Additional compute remains bounded by `BUDGET-001`.
4. A successful outcome primarily optimizes `customerVisibleBuildMs`; `runnerOccupiedMs`, cost, and queue/feedback are incorporated as versioned penalties and published separately. Failure, OOM, sustained swapping, incorrect artifact, or guardrail breach cannot be offset by speed: they suspend the arm for that compatibility class and trigger rollback.
5. Outcomes may arrive up to 24 hours late. A missing, duplicate, late, or propensity-less outcome does not update the model and remains `INCONCLUSIVE`.
6. A change to `measurementEpoch`, catalog, feature/reward definition, runner class, Gradle/JDK, material build logic, or detection of drift beyond the versioned threshold resets the affected model and returns to fixed cohorts. Samples are not mixed across eras.

The policy exports epsilon, propensities, reward components, samples per arm, exclusion reasons, resets, and rollback. The algorithm may change after beta, but only through another contract version and A/A; a model change never expands its authority beyond the prevalidated profiles.

### 14.5 Guardrails

- Do not exceed available memory.
- Maintain conservative limits under swapping or OOM.
- Penalize configurations that fragment daemons by heap/JVM arguments and avoid incompatible variants whose startup erases the benefit.
- Suspend exploration for releases or urgent builds.
- Roll back when p95 degrades persistently.
- Bound exploration percentage, additional compute, and number of variants by policy; a signed kill switch immediately returns to the conservative profile.
- Local exploration is opt-in and has its own frequency, battery/temperature, and regression budgets; by default, local builds only consume the validated CI profile and adjust hard limits to the machine.

---

## 15. Build Impact Analysis

This capability affects the build graph, not test selection.

### 15.1 Objective

Among alternative entrypoints declared by the customer, select the minimum set that preserves the pipeline's deliverables and checks. For example, replace a broad entrypoint:

```bash
./gradlew assemble
```

with the minimum required artifact set:

```bash
./gradlew :service-a:assemble :library-c:assemble
```

The optimizer does not synthesize `:project:task` merely because the observed graph appears equivalent: it may lose finalizers, aggregators, or side effects from the root lifecycle. The manifest must enumerate valid alternatives, or a versioned adapter must prove their equivalence. If the original pipeline requests `build`, `Test` tasks and their checks remain under Test Optimization's decision; Build Optimization reduces only artifact entrypoints assigned to it by contract `INT-001`.

### 15.2 Model sources

- A versioned, customer-owned manifest enumerating mandatory deliverables, checks, and entrypoints for each pipeline type.
- Project dependencies.
- Variants and configurations.
- Task dependencies.
- `api` and `implementation` dependencies.
- Shared code generation.
- Build logic and convention plugins.
- Global resources.
- Final artifacts required by CI.
- Conservative static rules for source sets, generated code, resources, and artifacts.
- Historical evidence of tasks activated by change types, used only to prioritize shadow evaluation, estimate savings, and warm the cache.

History never authorizes an omission: absence of prior execution does not prove absence of a dependency. The customer manifest and Gradle's declared graph are authoritative; if they conflict, the conservative union executes.

Minimal customer-owned contract example:

```yaml
schemaVersion: "1.0"
pipelineClass: "pull-request"
buildEntrypoints:
  - ":distribution:assemble"
allowedAlternativeEntrypoints:
  - [":service-a:assemble", ":library-c:assemble"]
requiredArtifacts:
  - "distribution/build/libs/*.jar"
requiredChecks:
  - id: "jvm-tests"
    owner: "test-optimization"
globalChangePaths:
  - "settings.gradle.kts"
  - "build-logic/**"
  - "gradle/**"
unknownChangePolicy: "FULL_GRAPH"
```

The manifest is versioned with the pipeline or held in a signed control plane with repository binding. Changing it is an auditable policy change and cannot be inferred silently by the optimizer.

### 15.3 Conservative rules

```text
Known localized change
  → minimum subgraph

Change to settings, build logic, wrapper, or version catalog
  → run the complete original entrypoints defined by the manifest

Unknown relationship
  → include; never exclude because evidence is absent

Test task or test report
  → delegate to Test Optimization; Build Optimization does not omit it
```

“Full graph” means the original entrypoints for that pipeline class, not every task in the repository. Changes to convention plugins, included builds, catalogs, toolchains, resolution rules, unknown plugins, or global files without ownership also force that fallback. A rule may exclude only work that contributes to no mandatory manifest deliverable/check.

### 15.4 Validation

During shadow mode, the system calculates what would have been omitted, but Gradle runs the complete build. The prediction is compared with the graph, deliverables, and manifest checks. A historical match is insufficient for promotion.

Then:

- First paired control in isolated workspaces and namespaces.
- Canary only on internal PRs.
- Periodic full build on `main` or nightly.
- Rollback of the specific rule on a discrepancy.
- Explicit false-negative-rate metric.

The correctness objective is zero known false negatives. The numerator, number of controls, and upper confidence bound are reported; zero observations are not presented as proof of zero risk. Any missing deliverable, omitted check, or divergence suspends the rule and every rule depending on the same unknown relationship.

**Accepted decision — BIA-002:** the promotion gate is not satisfied merely by observing zero failures. Per `pipelineClass`, the initial baseline requires at least 30 days of shadow, 3,000 total eligible decisions, 100 full controls in every mandatory change class, validation coverage ≥99%, and a one-sided 95% upper confidence bound for the false-negative rate ≤0.1% in aggregate and ≤3% in every mandatory stratum. A single false negative rejects the rule until it is corrected. Phase 0 may tighten these values based on risk, but not silently reduce them.

Changing the manifest, a global rule, an included build, or the impact adapter resets the sample for affected strata. Insufficient samples, lower coverage, or a control unable to materialize every deliverable produce `INCONCLUSIVE`; rollout does not advance. Periodic full builds continue after promotion and feed a sequential suspension gate, not an excuse to weaken the initial gate.

### 15.5 Limitation

If the final artifact transitively depends on every project, Build Impact Analysis cannot remove those dependencies. In that case, the remote build cache remains the primary mechanism for avoiding their execution.

---

## 16. Configuration Cache optimization

### 16.1 Operational and security boundaries

Configuration Cache is not an extension of the remote Build Cache. Gradle stores its entries under `.gradle/configuration-cache` in the project and, [as of this specification, does not support sharing them](https://docs.gradle.org/current/userguide/configuration_cache_status.html) among developers or CI machines.

- A persistent runner can reuse entries if it preserves the project workspace and corresponding encryption key.
- A truly ephemeral runner creates a new entry in every workspace and obtains no hits; enabling the option may verify compatibility, but does not count as acceleration.
- The MVP neither transports Configuration Cache entries between machines nor stores them in our Shared Cache Backend.
- Local downloads the enablement decision, not the entry; every developer generates and encrypts their own cache.
- [Entries may contain secrets used during configuration](https://docs.gradle.org/current/userguide/configuration_cache_requirements.html). By default, Gradle uses a machine-specific key in `GRADLE_USER_HOME`; if an authorized strategy preserves the same project cache across runs, it must securely preserve the same key or provide `GRADLE_ENCRYPTION_KEY` from a secret manager. The entry and key are never exported together.
- An entry is reused only in place within the same persistent project cache/workspace and host, with the same trust domain, repository, `configurationPolicyDigest`, Gradle/configuration-contract version, and encryption strategy. It is not packaged or “restored” through a generic CI cache; cross-machine distribution remains disabled while Gradle does not officially support it.
- `HttpBuildCache` URL and authentication are part of configuration. The gateway uses the stable rendezvous from 7.4.2 and keeps remote credentials outside Gradle. If Gradle serializes the local credential, it is protected by Configuration Cache's machine-specific encryption, grants no remote access, and rotates with its generation. `gatewayConnectionGeneration` enters `configurationPolicyDigest`: a compatible restart preserves the hit; rotating the local connection invalidates once and never reuses a stale endpoint/token.

### 16.2 Automatic adoption

1. Detect Gradle and plugin versions.
2. Run compatibility analysis, including the init/settings plugin itself: its listeners, Build Services, and Providers must produce zero unsuppressed problems.
3. Verify that the environment can preserve the workspace and encryption key; otherwise, classify the action as compatibility-only with no expected savings.
4. Enable it in a noncritical internal CI cohort.
5. Allow a natural build to create the entry.
6. Validate a natural hit in the same persistent environment.
7. Gradually promote the enablement policy.
8. Distribute that policy to local, where an independent entry is created.

The Launcher obtains the authenticated policy **before** starting Gradle; verifies its local or hardened signature according to profile, expiration, repository binding, revocation epoch, and compatibility; and derives a minimal immutable document containing only the fields Gradle needs to configure that invocation. It injects `-Dbuildopt.configuration-policy.digest=<sha256>` and `-Dbuildopt.configuration-contract.version=<version>`; during configuration, the plugin directly uses `providers.systemProperty(...)`, or passes them as declared parameters of a `ValueSource`, in accordance with [Configuration Cache requirements](https://docs.gradle.org/current/userguide/configuration_cache_requirements.html). It does not read system properties covertly inside `obtain()`.

The complete `policyDigest` remains in the ledger and provenance/attestation, but telemetry, UI, or rollout changes do not destroy a Configuration Cache hit. `configurationPolicyDigest` includes, among other fields, cache enablement, Test Optimization grants, qualified tasks, entrypoints, `l1SecurityGeneration`, `gatewayConnectionGeneration`, and parameters that alter the model. Two functional sequences are tested for every Tier 1 combination:

1. Store with digest A → same-invocation hit → digest B → miss/reconfiguration → next hit with B.
2. Gateway A → store → terminate A → gateway B at the same rendezvous with a new upstream credential → Configuration Cache hit and exclusive use of B. Rotating `gatewayConnectionGeneration` must cause one intentional miss, never connect to A, and never leak upstream credentials or plaintext in entries/logs/exports.

### 16.3 Automatic repairs

- Replace eager access with providers.
- Avoid capturing nonserializable objects in task actions.
- Move work from configuration to execution.
- Avoid mutable cross-project access.
- Update adapters for known plugins.
- Mark isolated incompatibilities only as an explicit temporary measure.

A repair that changes build logic follows the patch pipeline and Test Optimization gate. It is not promoted merely because Configuration Cache warnings disappear.

### 16.4 Isolated Projects

It may be evaluated in a later phase for large multi-project builds. In Gradle 9.6.1 it remains [experimental and has not yet reached incubating status](https://docs.gradle.org/current/userguide/isolated_projects.html); it will require a separate policy, specific diagnostics, and a more conservative rollout.

---

## 17. Automatic patch validation

### 17.1 Transactional pipeline

```text
Detect opportunity
  → generate ephemeral patch
  → build candidate in quarantine workspace and namespace
  → build control with stable policy in an isolated environment
  → validate invariants, deliverables, and artifacts
  → request FULL_RELEVANT_VALIDATION from Test Optimization
  → measure benefit
  → canary
  → rebuild/attest with trusted writer
  → promote as runtime policy or open a persistent pull request
```

### 17.2 Validations

**Accepted decision — ARTIFACT-001:** the generic private-beta validator recognizes only exact byte equality or exact equality of an ordered tree manifest—normalized path, type, relevant permissions, symlink target, size, and SHA-256. A difference is never ignored heuristically. Semantic equivalence requires a versioned adapter that enumerates which metadata may vary and proves that consumers are unaffected; the first adapters will cover reproducible JAR/ZIP files and POM/Gradle Module Metadata. An unknown format or an adapter without complete coverage produces `INCONCLUSIVE`.

- Full compilation from an empty workspace.
- Incremental build after an implementation change.
- Build after an ABI change.
- Execution from a different path.
- Resolved dependency graph.
- Byte-for-byte artifacts when they must be identical.
- Semantic comparison only when the artifact contract defines equivalence and the normalizer is versioned and reviewed. A cacheable task must produce repeatable bytes for the same key; if normalization occurs, it happens before the output is stored.
- No missing outputs.
- Configuration Cache compatibility.
- Stability across several natural executions.
- No candidate writes to the stable namespace.
- Policy digest, configuration/cache-contract versions, and provenance present in every action/cache record.

Existing tests are part of the gate when the patch may affect their inputs, but their selection and execution remain owned by Test Optimization. Build Optimization requests `FULL_RELEVANT_VALIDATION` and consumes the result; if the integration contract does not respond, the patch is not promoted.

For `api` to `implementation`, the gate adds comparison of Gradle Module Metadata/POM, compile classpath, runtime classpath, and compilation/execution fixtures for representative consumers, including known reflective uses. In published modules, technical validation does not replace explicit approval from the API owner.

### 17.3 Persistent promotion

In the private beta, a validated patch may:

- Remain as a transformation injected by the Gradle Optimization Plugin only if it is reversible runtime configuration through a public API and does not rewrite source or task actions.
- Generate a pull request.

A pull request is the beta's only persistent delivery mechanism for a change: there is no auto-merge or writing to existing/default branches, even if the repository has authorized runtime optimizations. The customer-side job may create only the ephemeral `buildopt/<actionId>` head required for the PR; it is not reused for another action. Every patch is bound to revision/source state; if the repository diverges or the patch conflicts, it returns to `PROPOSED` and is never automatically rebased over or allowed to overwrite customer changes. Direct writes to managed long-lived branches will be reconsidered after beta as a separate capability and authorization.

---

## 18. CI and local behavior

### 18.1 CI

CI is the primary learning environment:

- Produces trusted cache entries.
- Runs shadow mode.
- Hosts canaries.
- Validates patches.
- Provides periodic full builds.
- Validates and signs policies for local.
- Obtains the Test Optimization grant before configuration or disables test-task caching.
- Runs all instrumentation capable of producing custom-task outputs with a disposable L1 and pending writes, or with `push=false`.
- Runs trusted writers whose sole authority is `HERMETIC_PRODUCER_PROFILE` under continuous enforcement.
- Suspends discrepant optimizations without turning a correct baseline into a customer-visible failure.

Only a protected, authenticated internal job may produce in `stable`, and even then only for tasks/transforms covered by the allowlist or a later C1 contract. In the private beta, authentication uses the repository/deployment read-write token and local provenance record; the hardened profile requires workload identity and attestation. A fork or external-PR job receives no control-plane or shared-cache secrets; making a credential read-only does not prevent malicious build logic from exfiltrating it.

No initial experiments will run in:

- Release.
- Publish.
- Signing.
- Deploy.
- Migrations.
- Builds that mutate external resources.

These pipelines may use a stable policy and cache entries with accepted provenance if their owner authorizes it, but they receive no experimental cohorts, exploratory autotuning, or candidate patches.

#### 18.1.1 Initial CI integration

The primary contract is the CLI, not a shell rewritten by the backend:

```bash
buildopt run -- ./gradlew build
```

Everything after `--` is preserved as child-process argv. The Launcher creates a process group, forwards `SIGINT`/`SIGTERM`, respects the CI grace period, returns the original exit code, and writes observability outside customer artifact paths. A failure before task actions may use the bounded fallback; after that, section 12.4 applies.

GitHub Actions will be the first tested fixture: an action pinned by commit SHA installs a binary pinned by version+checksum, and the workflow explicitly preserves the prior command under `buildopt run --`. The private beta receives `BUILDOPT_READ_TOKEN` or `BUILDOPT_WRITE_TOKEN` from repository secrets and never gives them to fork jobs. The Launcher consumes the token to register the gateway/upstream and removes `BUILDOPT_*` from the environment of the Gradle process and its children. OIDC, audience/claims, reusable-workflow identity, and ephemeral-token exchange are implemented during hardening and do not block the first pilot. The core CLI remains portable to any CI that respects argv, signals, and exit codes; another CI is not declared supported until it has its own fixtures.

C4 adds a separate opt-in workflow to materialize PRs, defined and executed from the protected default branch; it does not use `pull_request_target` to execute an untrusted checkout. It consumes a signed patch bundle bound to the SHA, rechecks digest/source state without executing bundle code, and only then receives `permissions: contents: write` and `pull-requests: write` to create `buildopt/<actionId>` and a draft PR. The backend never receives or stores that `GITHUB_TOKEN`; fork jobs do not execute this workflow, the token exists only during materialization, and branch protection prevents modification of the default branch. If the customer does not grant those permissions, the result remains a downloadable patch bundle, not a partial PR attempt.

The workflow does not assume automatic GitHub Actions recursion: a `push` made with `GITHUB_TOKEN` does not create another run and, under the [current `GITHUB_TOKEN` contract](https://docs.github.com/en/actions/concepts/security/github_token), runs derived from the `pull_request` event when this PR is opened/synchronized remain `approval-required`. Candidate/control and artifact validation must already have completed before creating it; those PR checks are an additional defense and remain pending until a maintainer approves them. A repository that wants to automate them may explicitly configure a secure workflow through `workflow_dispatch`/`repository_dispatch` and grant its minimum permission, but that is not the beta default. Using a GitHub App to eliminate this approval is deferred to GA-D; no long-lived PAT is introduced.

#### 18.1.2 Candidate/control/baseline topology

**Accepted decision — CI-ORCH-001:** `buildopt run` remains the normal, authoritative pipeline path. It executes exactly one assigned arm among stable policy, an already-qualified reversible runtime candidate, or control; returns the original command's exit code; and publishes its deliverables. Natural cohorts compare equivalent jobs without duplicating every build.

High-correction actions never try to turn that process into two builds inside the customer's workspace. `buildopt run` registers an idempotent `VALIDATION_REQUEST`, and `buildopt-server` places it in a budgeted queue. A customer-owned workflow, defined in the protected default branch and run only on trusted revisions, automatically consumes those requests. In GitHub Actions, the beta activates it every 15 minutes through `schedule`, supports `workflow_dispatch` for recovery, and uses one `concurrency` group per repository; each run takes at most one validation lease. It requires neither a GitHub App nor a backend token to trigger workflows.

Within that separate validation job, fresh worktrees and `GRADLE_USER_HOME` directories are created, and candidate/control run in independent containers or sandboxes, sequentially in randomized order by default so that they do not compete for the same host. Independent jobs may be used when the CI adapter proves runner-pool equivalence. This validation does not determine the state of the originating pipeline or publish its deliverables as customer-visible artifacts.

The durable lifecycle of every attempt is:

```text
CREATED
  → POLICY_BOUND
  → GRADLE_STARTED
  → TASK_ACTION_STARTED
  → VALIDATED
  → COMMITTED | ABORTED
```

`TASK_ACTION_STARTED` is persisted when the plugin confirms the first task action, not when the JVM launches; that boundary governs safe replay in 12.4. If the event cannot be persisted, the attempt has an unknown boundary: the build continues, but it is neither automatically rerun nor allowed to commit pending. Every transition uses `attemptId` and compare-and-set. Repeating an already-applied command returns the existing state; skipping a transition, changing policy/source state, or receiving two owners aborts the attempt.

Authority is explicit:

- In the normal job, the single arm executed by `buildopt run` controls exit code and deliverables.
- In a high-correction validation, baseline/control is the correctness reference; candidate remains pending and is not delivered to the original pipeline.
- An infrastructure failure, cancellation, or timeout in the validation workflow produces `ABORTED`/`INCONCLUSIVE`, releases leases, and keeps the optimization disabled; it does not retrospectively change the customer's result.
- Runner-minute and concurrency budget is reserved before scheduling. Cancellation, expiration, or completion returns the unused reservation; a reconciler aborts dead owners and never silently duplicates an attempt.
- Comparison artifacts are identified by digest, retained according to the data lifecycle, and materialized only in the workspace that validates them. No arm shares writable outputs, Configuration Cache, daemon, L1, or write credentials.

Product queue metrics come from exact CI-provider timestamps for natural jobs, not from the internal sequential order of a validation. The GitHub adapter must calculate creation→start, start→finish, runner labels/pool, and cancellation as `EXACT` before enabling B in a pilot; if those fields are unavailable, B remains in fixed cohorts without a queue claim and does not promote a configuration.

### 18.2 Local

Before Gradle starts, the local Launcher and plugin:

- Download and verify authenticated, validated policies; the plugin does not use the network during configuration to obtain a new policy.
- Normally use remote cache read-only through the verifying gateway.
- Apply the validated Configuration Cache decision and generate local entries; they never download an entry created in CI.
- Perform autotuning bounded to local resources.
- May contribute evidence if privacy policy permits.
- Persist the highest authenticated revocation epoch and `l1SecurityGeneration` and, while offline, use the last valid policy only up to the TTL for its risk class; afterward, they disable remote reads and rotate any L1 potentially populated from them, including native tasks.
- Return to standard Gradle when a policy expires or is inapplicable.
- Allow and record an observed discrepancy, disable the optimization for the next invocation, and do not run experimental hermetic enforcement.

Example:

```text
Policy: repository R / build logic B93

remote cache                 ACTIVE
configuration cache          ACTIVE
workers                      ADAPTIVE, max 6
generateOpenApi caching      CONTRACT_QUALIFIED
affected-project pruning     CI ONLY
api→implementation patch     CI CANARY
```

Local will not be used as a laboratory for risky patches.

---

## 19. Product modes

The following modes describe a possible product. They are not gates for the current POC. The POC exposes explicit measurement and candidate commands, retains `BUILDOPT_BYPASS=1`, and never performs autonomous production promotion. A future productization decision must materialize the mode matrix, persistence, transitions, CLI, and authorization tests before these names become customer-facing promises.

### Observe

- Instruments.
- Models the build.
- Calculates candidates.
- Does not modify execution except for minimal telemetry.

Useful for onboarding, evaluation, or customers with strict constraints.

### Verified

- Enables safe caches.
- Applies reversible runtime configurations.
- Runs shadow mode and canaries.
- Enables proof-gated actions after contract and validation.
- Does not persist patches in the repository.

This should be the recommended default mode.

### Autopilot

- Includes all of Verified.
- Generates and validates patches.
- In the private beta, publishes them only as pull requests; any later direct promotion requires separate authorization.
- Automatically rolls back runtime policies and, for an already-merged patch, generates the revert PR or exact instruction without silently writing to the repository.

Modes do not change what is considered correct; they change only which actions are authorized.

---

## 20. Security, privacy, and trust

### 20.1 Trust boundaries and threat model

| Actor or boundary | Trust | Primary risk | Mandatory control |
|---|---|---|---|
| Authenticated developer/internal CI | Partial | Stolen credential or compromised build logic | Short-lived token, minimum scope, read-only by default |
| Protected `main` writer | TCB, high but revocable | Can sign semantically incorrect bytes if compromised | Attested runner, identity-bound write token, stable namespace, audit, revocation, and rebuild |
| External fork/PR | Untrusted | Secret exfiltration, poisoning, reading proprietary artifacts | No shared credentials; empty disposable backend or cache off |
| Candidate/canary | Experimental | Contaminating stable or falsifying validation | Quarantine workspace and namespace, `push=false` to stable |
| Local Verifying Cache Gateway | Per-invocation TCB | Stolen key/trust root or skipped verification | Loopback, ephemeral token, binary provenance, fail-closed signature/checksum/revocation |
| Edge Cache Node | Partially trusted infrastructure | Replay, offline operation, compromised host | mTLS/control-plane authentication, attested objects, offline quarantine, revocation |
| Shared Cache Backend | Untrusted data plane in gateway mode | Cross-tenant leak, corruption, server compromise | Isolation, encryption, negative authorization, and independent verification before delivery to Gradle |
| Optimization Service/Policy API | Privileged control plane | Malicious policy or replay of an old policy | Signature, expiration, monotonic version, repository binding, and key rotation |
| Export destinations | Outside the build | Exfiltration of paths, arguments, or fingerprints | Redaction before buffering, field allowlist, and consent |

The threat model covers at least cache poisoning, confused deputy, credential theft, replay, policy downgrade, key collision, compromised backend/Edge, compromised runner, cross-tenant leakage, and denial of service from large objects or hot keys. An attestation proves provenance and integrity, not semantic correctness: a compromised protected writer is part of the TCB. The baseline detects, contains, and recovers from that compromise; it does not claim to prevent it. An optional high-assurance profile may require independent reproduction in another trust domain before `stable`. The threat model is reviewed before every expansion of a trust boundary.

#### 20.1.1 Private-beta trust profile

The table above describes the hardened target. The beta reduces actors and boundaries through these nonnegotiable rules:

- One deployment serves a single tenant/pilot. Repositories, stable/quarantine/control, and credentials remain separate, but no isolation from another tenant inside the same process is claimed because that scenario does not exist.
- Every remote hop uses TLS. Distinct opaque read and read-write tokens are manually provisioned, scoped by repository/namespace, stored hashed by the backend, expire within 30 days, and can be rotated by immediate invalidation. There are no users, SSO, general RBAC, or OIDC.
- Untrusted forks and PRs receive no tokens. Stable writes come only from the protected workflow configured for the pilot. The C4 workflow may receive a job-scoped `GITHUB_TOKEN` solely to create `buildopt/*` and the PR; that token never reaches the backend or Gradle process.
- Policies, revocation state, and provenance records use the final schema but are authenticated with a local Ed25519 key generated at installation and protected by host permissions. Onboarding pins the deployment ID and public-key fingerprint outside the policy; a policy cannot replace its own trust root. That key is not separated from the deployment TCB, so the beta does not promise resistance to total backend/control-plane compromise.
- Complete checksum, pending/commit, namespace separation, revocation, exact artifact validation, and fallback are beta guarantees because they protect correctness from corruption, incompleteness, and invalid candidates.

OIDC/workload identity, ephemeral audience-bound tokens, KMS/HSM, DSSE/in-toto, multi-tenant RBAC, and external trust roots are added during hardening. Migration requires dual-read of provenance for a bounded window and rebuilding stable; a beta record is never retroactively labeled as a hardened attestation.

### 20.2 Integrity, provenance, and cache poisoning

- Only protected internal CI may write to `stable`; read permission never implies write permission.
- A digest verifies that bytes did not change, but does not prove who produced them or under which policy. In hardened, every stable object needs an attestation signed outside the data plane that includes tenant, repository, namespace generation, key, payload checksum, producer identity, source revision, source-state digest, policy digest, cache-contract/output-semantics versions, platform, revocation epoch, and timestamp/expiry. The beta retains these fields in a locally authenticated provenance record without claiming separation from the data plane. The trusted writer requires a clean protected checkout or explicitly records additional state; `source revision` is provenance, not an equality requirement for the reader.
- The backend rejects overwrite with different bytes for the same identity and quarantines both pieces of evidence.
- The local gateway verifies namespace, checksum, provenance/attestation by profile, key status, revocation epoch, and compatibility **before** serving the object to Gradle; a failure is a safe miss, not a partial hit. In beta and direct mode, Shared is part of the TCB and does not provide hardened protection from a compromised backend.
- Every revocation includes `effectiveAt` and a freshness SLA for CI, local, and Edge. `objectsServedAfterRevocation` counts a violation only after `effectiveAt + SLA`, but the UI also shows the propagation window; a policy without fresh state stops serving the affected class.
- In hardened, signing keys are stored in KMS/HSM or as ephemeral workload-bound keys outside Shared/Edge. Trust roots arrive in signed policy and are rotated and revoked; the system retains key ID and validity windows and denies attestations issued outside them. The beta uses the local key and is explicitly outside this guarantee.
- Launcher, plugin, JVM agent, Rust helper, gateway, and backend are distributed with verifiable signatures, checksums, SBOMs, and provenance. The Launcher verifies artifacts before executing them; a release revocation opens the kill switch and blocks that version.
- Repository/namespace isolation and negative tests are beta requirements. Multi-tenant isolation within a shared service is a hardening gate and is not simulated with a single beta instance.
- No physical cross-tenant deduplication exists in any profile. The beta is already isolated by deployment; hardened preserves independent deletion/encryption and avoids existence side channels.

### 20.3 Credentials, secrets, and replay

- Never include secret values in logs, UI, policy, exported fingerprints, or action records.
- In beta, cache tokens are opaque, scoped by repository/namespace and operation, stored hashed, expire within 30 days, and are rotated manually; they are not reused among candidate, control, and stable. In hardened, they become ephemeral, audience-bound, and derived from workload identity.
- The Launcher obtains secrets immediately before the build, passes them through a supported channel, and removes them from the inherited environment of processes that do not need them.
- If a secret legitimately affects output, use an opaque version managed by the secret manager or a keyed digest; a plain hash of a low-entropy secret may expose it.
- Policies include a nonce/monotonic version, `issuedAt`, `expiresAt`, repository binding, revocation epoch, and Launcher/plugin compatibility. CI requires online freshness; local persists the highest signed epoch seen. A validly signed but old policy may also be a replay and is rejected.
- Configuration Cache is treated as sensitive material because it may serialize secrets; section 16.1 controls apply.

### 20.4 Local data and export

- Local telemetry is opt-in according to policy.
- Analysis, evidence store, and hashes may be kept on-premises.
- Keyed redaction/tokenization of paths, arguments, and sensitive variables occurs before persistence or transmission.
- The `evidence` and `diagnostic` profiles require permissions distinct from `summary`; no exporter can expand the authorized profile.

Phase 0 publishes a contractual data-lifecycle matrix. The private beta minimizes retention and does not implement automatic legal hold; the hardened profile may expand only justified classes:

| Class | Private-beta default | Hardened target | Quota and deletion |
|---|---|---|---|
| `stable` cache blob | 30-day TTL | Subject to plan, initially at most 30 days unless contractually configured | Immediate logical and asynchronous physical deletion |
| Managed L1 / persistent volume | Native retention bounded by policy | Also bounded by attestation/freshness | Rotation by `l1SecurityGeneration`; controlled volume purge |
| `pending` | Lease and maximum 24-hour TTL | Same or lower | Separate pool; automatic abort |
| Metadata/provenance/tombstones | Blob lifetime + 7 days | Blob lifetime + approved recovery/audit | Immediate revocation; do not delete tombstone before its objects |
| Quarantine/security evidence | 7 days | 7 days, extendable for an approved incident/legal hold | Separate pool; restricted access |
| Optimization evidence | 30 days | Configurable by tenant/residency | No source payloads; coordinated deletion |
| `summary` / `diagnostic` telemetry | 30 days / 7 days opt-in | Up to 90 days / 7 days by policy | Separate quotas; diagnostic disabled by default |
| Local spool/DLQ | 24 hours and byte limit | Same or lower | Explicit drop policy; best-effort secure wipe |
| Security audit | 90 days, minimized and pseudonymized | Up to 365 days; documented legal exceptions | Append-only within the window |

Beta deletion revokes access immediately and covers blobs, metadata, managed L1/persistent volumes, evidence, and the deployment spool; as a single-node system, it claims no replica/backup SLA. There is no silent legal hold: extraordinary retention requires explicit pilot consent with recorded scope, reason, and expiry. The hardened profile adds replicas, backups, and deletion/crypto-shred SLAs. Immutable logs retain no source, raw paths, or secrets: they use keyed identifiers. Customer-controlled downstream destinations and copies are outside managed deletion; they receive a tombstone/event and retain their own obligation.

### 20.5 Tasks with external effects and releases

A task that publishes, deploys, signs, or mutates a service is not treated as an ordinary cacheable transformation. Releases and privileged pipelines do not participate in experiments. They may consume only stable objects with permitted provenance and will execute every external-effect task normally.

### 20.6 Compromise response

On compromise of a runner, token, signing key, Edge, or backend:

1. Revoke the identity and stop writes from the affected trust domain.
2. Increment the revocation epoch and open the read circuit breaker for potentially affected namespaces.
3. Locate affected objects and L1 instances by producer identity, key ID, time window, policy digest, and `l1SecurityGeneration`.
4. Quarantine them without deleting audit evidence.
5. Rotate `namespaceGeneration`, rebuild from trusted writers, and rotate credentials/keys; never overwrite an immutable committed identity.
6. Emit an exportable security event and document the blast radius.

Compromise exercises measure detection, revocation freshness, potentially exposed objects/L1 instances, time to containment, and recovery. They must include an authorized writer attempting to publish incorrect bytes, not only a hostile Shared/Edge.

---

## 21. Performance and cost budget

The optimizer must not consume more time than it saves.

### 21.1 Target overhead

- Lightweight instrumentation in stable builds.
- Deep instrumentation only for candidate tasks and during bounded windows.
- Batching and asynchronous evidence delivery after the build.
- Incremental hashing and reuse of Gradle information when possible.

### 21.2 Cache-versus-execute decision

Not every output is worth transferring. A cache-opportunity score is separated from a session-latency prediction:

```text
expected_task_work_saving =
    P_hit * (T_execute - T_restore_hit)
    - (1 - P_hit) * T_lookup_miss
    - P_write * T_store_on_critical_path

expected_session_time_saving =
    critical_path_model(task/action effects, concurrency and overlap)
    - product_session_synchronous_overhead

expected_net_infrastructure_value =
    build_compute_cost_avoided
    - product_runtime_cost
    - validation_and_control_compute_cost_amortized
    - incremental_storage_cost
    - incremental_network_cost
```

`expected_task_work_saving` is used for admission and opportunity ranking; it is not summed to claim build savings because task wall times may overlap and exceed elapsed time. `expected_session_time_saving` uses the critical-path/temporal-union model and decides only exploration. Final promotion uses the observed causal effect on `customerVisibleBuildMs`.

`T_restore_hit` includes lookup, local verification, download, decompression, and materialization; only the portion of store that blocks deliverables enters `T_store_on_critical_path`. Shadow, control, and asynchronous validation are not subtracted from latency: their compute and `runnerOccupiedMs` are amortized in economic value over the window in which the policy remains active. The policy also requires an uncertainty interval and a minimum net time benefit; very fast copy or packaging tasks may be cheaper to execute than restore.

Starting with MVP-A0, this formula governs L2 enablement by build/runner class. The backend first applies the hard guarantees from section 7.5.3.4: quotas, TTL, maximum size, watermarks, and size-aware SLRU. Admission and retention use only signals the backend knows directly—size, approximate frequency, recency, and transfer cost—and incorporate task time only when an adapter provides an exact association between cache key and task record. Remote per-task selection will be implemented only if a public API or proven versioned adapter exists; L1 will not also be disabled to simulate it. Pure LFU is rejected, and TinyLFU will be evaluated only after beta if telemetry demonstrates scan pollution.

### 21.3 Validation compute

- Prioritize natural evidence.
- Use a small deterministic canary.
- Make candidate-first visible only for already-qualified, low-risk runtime actions; for high-correction actions, baseline/control remains authoritative until validation completes.
- Use a second execution on failure or ambiguity, or as the first gate for a high-correction action.
- Make the budget configurable per repository.

Candidate-first is an availability and cost strategy; it does not independently create a causal measurement when candidate passes.

### 21.4 Estimation, measurement, and attribution

Three distinct comparisons are maintained:

| Question | Candidate | Control | Result |
|---|---|---|---|
| How much does the complete product accelerate builds? | Product and stable policy active | Preserved, versioned Gradle configuration the customer would use without the product | Product north star, including all overhead |
| How much does one action contribute? | Stable policy + candidate action | Same stable policy without that action | Incremental effect of the action |
| How much does instrumentation/orchestration cost? | Policy-off/passthrough mode with components loaded | Native/pre-product Gradle | Product-owned overhead by component and version |

The dashboard does not add an action's incremental effect to the total product effect. The first control is never retrospectively redefined as “latest stable policy,” because doing so would remove already-obtained improvements from the baseline.

The pre-product baseline is recorded during onboarding with `baselineDefinitionDigest`. A customer-decided change starts another `measurementEpoch`; the prior effect remains historical but is not combined with the new one as if the control were identical. An action's incremental result subtracts `incrementalActionOverheadMs`; base product overhead cancels between those arms and appears in full only in `PRODUCT_TOTAL`.

For a paired comparable revision:

```text
observed_net_build_time_saved_ms =
    T_control_customer_visible - T_candidate_customer_visible

observed_build_time_reduction_ratio =
    observed_net_build_time_saved_ms / T_control_customer_visible
```

`T_candidate_customer_visible` includes all synchronous product overhead. A negative `observed_net_build_time_saved_ms` is a regression and is retained as such. In randomized cohorts, the effect is estimated between equivalent distributions; the p95 delta is calculated between cohort quantiles, not by averaging per-build p95 deltas.

The sign convention is contractual: `*Saved*` and `*Reduction*` fields use positive values for improvement; `*Delta*` fields are candidate minus control and use negative values for improvement. UI and exporters cannot implicitly reverse it.

In a paired aggregate, absolute savings per session is the mean of deltas, and percentage is the sum of deltas divided by the sum of control times; individual percentages are not averaged. Cohorts of different sizes use the declared stratified estimator. Every result exports `effectStatistic`, and p50/p95/p99 remain separate statistics. The portfolio total multiplies the effect by sessions actually eligible and treated, not by all theoretical traffic.

- `estimatedNetBuildTimeSavedMs` uses a historical/contextual model and always exposes an interval, model version, and comparison features. It ranks candidates and decides exploration; it is not presented as demonstrated impact.
- `observedNetBuildTimeSavedMs` requires an isolated paired control, a randomized experiment, or an approved quasi-experimental design with published covariates and assumptions. Simply comparing with the prior build does not satisfy the contract.
- A paired control preserves the same `WorkUnitsFingerprint`—including Test Optimization policy/grant—pipeline, runner/cgroup, and prescribed treatment of workspace/daemon/cache state. In randomized cohorts, revisions are not artificially paired: the same eligible population, recorded propensities, and balance/stratification of fingerprints/covariates are required. Any unadjusted difference invalidates the comparison or degrades it to an estimate; if Test Optimization changes, the effect is labeled joint.
- Variant assignment and probability are recorded before the outcome is known and analyzed by intention-to-treat. A candidate that fails, is slow, or is canceled because of the product cannot disappear from the sample or be reclassified as a generic infrastructure failure; exclusion criteria and treatment of infrastructure failures are fixed before the experiment.
- Repeated observations of the same revision, PR, or runner are grouped as correlated units when estimating intervals. The catalog fixes stratified bootstrap, frequentist or Bayesian model, and stopping rule; a conventional test is not watched continuously until it becomes positive.
- The experiment platform runs A/A tests and checks sample-ratio mismatch against expected propensities, covariate balance, namespace/cohort contamination, capacity/cache interference, and assignment changes. Adaptive experiments use propensity-aware inference and retain the assignment policy. An integrity failure invalidates the effect even if the delta appears favorable.
- Success, build failure, infrastructure failure, and canceled outcomes are never mixed. For build failures, `timeToFirstBuildFailureMs` is optimized; a faster failure does not count as a saved successful build.
- Every aggregate publishes time window, population, candidate/control sample, exclusions and reasons, p50/p75/p90/p95/p99, dispersion, and effect interval. Results are calculated by stratum first; a portfolio view weights by natural traffic without mixing incompatible raw observations.
- A stable control cohort or periodic sampling is reserved to detect drift. Its proportion adapts to risk, minimum detectable effect, variability, and compute budget, not to a universal percentage.
- `customerVisibleFeedbackMs` and `ciQueueMs` are analyzed per runner pool as portfolio causal guardrails. If controls/canaries saturate capacity and worsen the queue beyond budget, the cost is attributed to the product even if `customerVisibleBuildMs` improves once a runner is assigned.
- When several actions interact, attribution uses factorial experiments or a recorded enablement sequence. If the effect cannot be separated, the combined benefit is exported and potentially overlapping savings are not added.
- Test Optimization and Build Optimization use common `actor` and `actionId` values for deduplication. `testOwnedTaskExecutionMs` reconciles work ownership but is not added to session wall-clock; an avoided task belongs to only one primary action, and test savings are not claimed as Build Optimization impact.

Economic value is calculated separately from latency:

```text
net_infrastructure_value =
    build_compute_cost_avoided
    - product_runtime_cost
    - validation_and_control_compute_cost
    - incremental_storage_cost
    - incremental_network_cost
```

Additional shadow/control compute may not affect the developer, but it does reduce net value. Both latency and cost must retain currency units, region/provider, tariff period, and price-model version.

The private beta does not block a time improvement because of an incomplete monetary model. Its `costForLatencyBudget` is operational: additional compute within the weekly 5%/daily 10% from section 12.6, synchronous overhead within the A0 gate, and no queue saturation outside the guardrail. When the pilot configures prices, `netInfrastructureValue` is exported as evidence but does not replace the north star. A mandatory monetary profile and per-tenant exceptions belong to hardening.

### 21.5 Benchmark and pilot operational gate

**Accepted decision — OPS-001:** Phase 0 versions `benchmarks/beta-v1.yaml` and a reproducible harness with seed, runner image by digest, component versions, and exportable raw results. The minimum profile runs:

- 1, 8, and 32 concurrent clients;
- a deterministic mix of 70% 64-KiB objects, 20% 1-MiB objects, 8% 10-MiB objects, and 2% 100-MiB objects;
- cold, warm-at-70%-hits, 60-minute sustained-load, and eight-hour soak phases;
- small/medium/large golden-lane fixture builds with known outputs and critical paths;
- a fault matrix with gateway/server restart, mid-PUT/GET cancellation, truncated/corrupt blob, network latency and loss, SQLite busy, expired lease, disk at high watermark and out of space, revoked policy/grant, and process death between pending and commit.

Every beta release publishes hardware/cgroup, actual observed distribution, p50/p95/p99, throughput, errors, bytes, recovery, and deviations from the specification. A result is comparable only when it retains the same benchmark digest.

`buildopt-server` exposes separate liveness and readiness. Readiness remains false until blobs/metadata are reconciled, revocations loaded, and the local key verified; liveness does not promise that serving cache is safe. Minimum alerts cover disk/quota, corruption, stuck attempts/leases, revocation lag, policy freshness, circuit breaker, SQLite contention, export backlog, and acceptance error/latency budgets.

Online revocation must reach the Launcher/gateway within 60 seconds and always before a new invocation begins; an invocation keeps its policy immutable until completion, and pending is not committed if the epoch changed. A local bypass independent of the control plane—action input or `BUILDOPT_BYPASS=1`, consumed and removed by the Launcher—executes the original argv without plugin, gateway, or policy fetch. The runbook includes kill switch, bypass, version rollback, removal of the step/init script, explicit preservation or purge of managed directories, and recovery from a partial C4 branch/PR.

`OPS-001` closes by profile. The `OPS-001/A1` profile requires benchmark/fault/soak, readiness, revocation, bypass, and runbooks to open the pilot. The `OPS-001/B` extension also requires the GitHub adapter with `ciQueueMs=EXACT`; A1 may initially operate with that metric `UNAVAILABLE` and no queue claim, but B cannot promote decisions consuming additional capacity until it closes.

---

## 22. Success metrics

### 22.1 North star and primary KPIs

The product will have three primary time KPIs. The first measures the central effect; the others prevent optimizing it at the expense of queue time or the long tail.

| KPI | Definition | Decision it enables |
|---|---|---|
| `observedNetBuildTimeSavedMs` and `observedBuildTimeReductionRatio` | Causal difference in `customerVisibleBuildMs` between control and candidate, net of synchronous overhead and including regressions | Promote, retain, suspend, or roll back an optimization |
| `customerVisibleBuildP95DeltaMs` | Difference between comparable candidate and control p95, accompanied by p50 and p99 | Prevent a central improvement from degrading slow or unstable builds |
| `customerVisibleFeedbackP95DeltaMs` | Causal difference from job eligibility to deliverables, with `ciQueueP95DeltaMs` per runner pool | Prevent validation/control from moving the regression into the queue |

`netInfrastructureValue` is a companion economic KPI: it controls total cost and learning budget but does not replace time reduction. A customer may explicitly authorize higher cost for lower latency within a budget; in that case, the UI does not present it as infrastructure savings.

The organizational north star aggregates only `effectScope=PRODUCT_TOTAL` measurements. `ACTION_INCREMENTAL` explains decisions and is not added on top of the total. Results always include absolute value, percentage, interval, sample, window, stratum, and measurement coverage. A portfolio sum of hours saved may use only `OBSERVED_CAUSAL`; estimates are shown in a separate series.

### 22.2 Latency and avoided work

To explain the north star, the following will be measured:

- `customerVisibleBuildMs` p50/p75/p90/p95/p99, trimmed mean, dispersion, and regression rate by repository, pipeline, and comparable workload.
- `customerVisibleFeedbackMs` and `ciQueueMs` p50/p95/p99 by runner pool, including saturation attributable to shadow/control/canary.
- Startup/initialization, configuration, execution-or-restore, finalization, and `unattributedMs`, both in milliseconds and as a percentage of the total.
- `buildCriticalPathMs`, critical-path length/node count, and contribution of its leading tasks.
- `timeToFirstBuildFailureMs` by failure class and `timeToFirstActionableOutputMs` when a contractual intermediate deliverable exists.
- Time in dependency resolution, downloads, locks, worker leases, internal queues, GC, and waiting for shared resources.
- Incremental-build duration by `changeClass`: no change, implementation, API, build logic, dependencies, toolchain, or global change.
- Latency variability and outlier frequency, separating cold/warm daemon, clean/reused workspace, and cold/warm cache.
- Tasks requested, executed, `FROM_CACHE`, `UP-TO-DATE`, `NO-SOURCE`, `SKIPPED`, and failed; execution-avoidance rate by count and modeled time.
- Graph nodes/projects requested and avoided, merged invocations, removed `clean` executions, and avoided startup/configuration.
- Build sessions and deliverable sets completed per runner-hour, separated by workload; this is a throughput metric and does not replace per-session latency.
- Configuration Cache hit/miss, miss reason, avoided configuration time, and entry-load cost.
- Incremental compilation/processors: sources recompiled versus affected, full recompilations, and known cause.
- In MVP-C3, entrypoints/projects omitted by Build Impact Analysis and cost actually avoided; never tests selected or avoided.

### 22.3 Cache effectiveness

Hit rate by count is insufficient. The following will be published for L1, L2 Shared, and Edge:

- Lookups, hits, misses, writes, skipped writes, and errors, with eligible tasks as denominator.
- **Time-weighted hit rate:** sum of counterfactual execution time for hits divided by the total eligible opportunity, labeling whether the counterfactual is observed or modeled.
- **Useful-hit rate:** percentage of hits for which lookup + verification + transfer + decompression + materialization was faster than execution; negative-benefit hits are retained.
- p50/p95/p99 latency for lookup-hit, lookup-miss, `backendFirstByteMs`, `verifiedHitReadyMs`, restore, and store by size bucket; net time saved or lost by tier.
- Bytes read/written, compression ratio, throughput, egress, object size, and disk materialization.
- Miss reasons: no-entry, key changed, policy/namespace mismatch, revocation, invalid provenance/attestation, expired/evicted object, backend unavailable, and circuit breaker.
- Eligibility, declared cacheability, excluded tasks, and cause; custom tasks qualified through `OFFICIAL`, `REVIEWED_ADAPTER`, `HERMETIC_PRODUCER_PROFILE`, or `SOURCE_PATCH`, plus repeatability and relocatability gates.
- Admissions/rejections, evictions by bytes and cause, age at hit/eviction, reuse count, one-hit ratio, probation/protected hit rate, and pressure by tenant/namespace.
- Warm-up time to a stable hit rate, orphaned/pending/tombstoned objects, effective deduplication within the trust domain, and operations recovered after restart.

### 22.4 Resources and cost

- CPU core-seconds, CPU utilization, wall/CPU ratio, and effective worker utilization.
- Peak/average RSS, heap used/committed, allocation rate, GC count/pause, and OOM/kill rate.
- Disk read/write bytes, IOPS, maximum temporary space, and time blocked on I/O.
- Network ingress/egress bytes, requests, throughput, and time blocked on network, by service/tier.
- Runner billed-seconds and avoided compute cost; incremental storage byte-hours, request cost, and network egress.
- CPU, memory, disk, network, and product-owned cost of the Launcher, plugin, JVM agent, Rust helper, gateway, backend, export, shadow, control, and canary.
- `validationAndControlComputeMs` and the ratio of additional learning compute to avoided build compute.
- `runnerOccupiedMs` and `additionalRunnerOccupiedMs` by arm/action, together with changes in saturation and queue by runner pool.
- Net cost by build, repository, pipeline, and month, with currency, region, tariff, and price-model version.

### 22.5 Autonomous engine and learning

- Candidates discovered, qualified, rejected, in shadow/canary, active, suspended, and rolled back by action type.
- Coverage: percentage of eligible builds receiving each policy and percentage of baseline time on which at least one optimization acts.
- Time from discovery → sufficient evidence → canary → enablement, and duration until drift/revalidation.
- Observed incremental benefit per action and combined benefit when interactions exist; never the sum of overlapping estimates.
- Promotion, rollback, and suspension rates; reasons; time to detect a regression; and time to return to a safe policy.
- Inconclusive experiments, power/minimum detectable effect achieved, sample consumed, and compute spent per useful result.
- Evidence/policy freshness, revalidation frequency, latency/cache/resource-profile drift, and actions invalidated by contract changes.
- Conflicts among actions, selector decisions, and the difference between expected benefit before canary and observed benefit after it.

### 22.6 Correctness and security guardrails

- Artifact/deliverable divergence rate and incomplete manifests, by comparison type.
- Incorrect cache-hit, corruption, and poisoning incidents; invalid provenance/attestations/checksums, revocations, and quarantined objects.
- Candidate/control outcome mismatch, repeatability/relocatability failures, and detected undeclared accesses.
- Change in build-failure rate by failure class and change in `timeToFirstBuildFailureMs`; infrastructure failures are counted separately.
- Canary failure, fallback, and rollback rates; builds recovered through fallback and builds affected before containment.
- False-negative rate, upper confidence bound, shadow days, sample/controls by change class, coverage, and Build Impact Analysis manifest resets; loss of deliverables or entrypoints.
- Tenant/repository/trust-domain isolation violations, unauthorized attempts, and cache objects served after revocation.
- Secret/raw-sensitive-field exposure, redaction/tokenization failures, and access to unauthorized export profiles; contractual target zero.
- Missing/invalid `TestCacheGrant`, a `Test` task cached without a grant, and any duplicate attribution with Test Optimization. The contractual target for these violations is zero.

### 22.7 Overhead, reliability, and data quality

- p50/p95/p99 synchronous overhead and percentage of `customerVisibleBuildMs` for Launcher, policy, plugin, gateway, cache verification, and finalization, with a passthrough baseline.
- Lightweight/deep tracing overhead, events per second, drops, saturated buffers, and builds that automatically degrade instrumentation.
- Availability, error rate, saturation, throughput, and p50/p95/p99 latency of Shared, Edge, policy, and export gateway; builds opening the circuit breaker and time spent open.
- QPS/streams/in-flight bytes rejected by principal/tenant, fairness lag, hot-key coalescing, temporary spool reserved/used, and cleanup failures.
- Cache/backend timeouts, retries, fallback to L1/Gradle, and net contribution of each dependency to the critical path.
- Telemetry/export lag, retry, duplicate, dead letter, spool utilization, and drop rate; complete deduplicable JSONL sequences and partial final documents.
- `COMPLETE | PARTIAL | UNAVAILABLE` and `EXACT | APPROXIMATED | UNAVAILABLE` coverage by metric, Gradle/JDK version, and pipeline.
- Reconciliation error between components and total, `unattributedMs`, clock anomalies, late events, and schema-validation failures.
- Builds eligible, included, and excluded from analysis; exclusion rate, reason, and bias by stratum. Absence of telemetry is never interpreted as zero.
- Assignment coverage, sample-ratio mismatch, candidate/control balance, cross-arm contamination, crossover, and experiment-platform A/A-test results.

### 22.8 Experience and adoption

- Onboarding time to first observed build, first useful hit, and first confirmed causal saving.
- Builds/repositories requiring manual changes, change type, and maintenance time.
- Percentage of CI and local builds receiving a current policy; disablements/overrides and reason.
- Repositories with causal improvement, no change, or regression, showing the distribution rather than only an average.
- Cumulative customer-visible waiting hours and compute-hours saved, separating CI/local, observed/estimated, and build/test ownership.
- Fallback frequency, interruptions attributed to the product, and tickets/incidents per thousand builds.
- Use and success of JSON/JSONL/API export, time required to explain an action, and percentage of actions with accessible complete evidence.

### 22.9 Governance, targets, and promotion

The metric catalog is versioned. Every definition specifies owner, purpose, formula, unit, grain, population/denominator, source, time boundary, valid dimensions, null policy, quality, retention, and caveats. A semantic change increments `metricDefinitionVersion`; it does not rewrite historical series.

The current POC uses the bounded decision rule in section 4.4 and [`poc-value-validation-v1`](./specs/poc-value-validation-v1.md). Mechanism exploration answers whether an idea merits more work; it does not authorize production promotion. A safety enabler may pass with no more than 2% native-baseline regression, but only an accelerator clearing `max(500 ms, 2%)` with a positive lower bound counts as build-time value. The combined product path must pass separately against optimized native Gradle.

The following longer-window policy is retained only as a possible private-beta/productization gate. It does not block or relabel POC evidence:

| Gate | Direct/reversible | Proof-gated or patch |
|---|---|---|
| Minimum window | 7 days and ≥100 comparable observations per arm | 14 days and ≥200 per arm, in addition to correctness control |
| Minimum benefit | One-sided 95% lower bound ≥ `max(500 ms, 2% of control)` | Same time gate; correctness is never offset by speed |
| p95 regression | Upper bound ≤ `max(500 ms, 3% of control p95)` | Same, with no `customerVisibleFeedbackMs`/queue regression outside budget |
| p99 | Always exported; gated when ≥1,000 observations per arm exist, with budget `max(1 s, 5%)` | Same; insufficient sample is labeled, not presented as proof about the tail |
| Correctness | Zero confirmed divergences and zero confirmed product-attributable failures | Isolated candidate/control, artifact validation, and `FULL_RELEVANT_VALIDATION` when applicable |
| Post-promotion control | 5% within comparable cohorts | 5% plus periodic cache-off audit |

The maximum learning window with additional compute is 28 days. If power, coverage, or comparability is insufficient, the action remains `INCONCLUSIVE`, additional spending stops, and natural observation may continue; elapsed time does not promote it. These defaults are recalibrated through pilots with a new `measurementPolicyVersion`, without rewriting prior results.

Opening a pull request is not equivalent to automatically promoting an action. C4 may open a PR with complete correctness gates, at least one paired control, and clearly labeled `PRELIMINARY` benefit; it need not consume 200 pairs before requesting review. The patch is presented as a confirmed causal improvement and contributes to the north star only when it reaches the statistical gate with comparable observations.

The operational rule is:

1. Before canary, the conservative bound on **estimated** time benefit, already net of restore/lookup and synchronous overhead, must exceed the configured absolute/relative thresholds, or the action enters explicitly budgeted exploration.
2. After enough sample is collected, the conservative bound on **observed causal** savings must exceed both components of the effective threshold `max(minimumNetTimeBenefitMs, minimumNetTimeReductionRatio × control)`; improving hit rate, CPU, or task count is insufficient.
3. No primary KPI may offset artifact divergence, an isolation/Test-ownership violation, or a correctness-guardrail breach.
4. Queue regression beyond budget or excessive overhead suspends or rolls back the action even if p50 improves. Negative economic value does the same only when it exceeds the cost-for-latency budget; policy must explicitly authorize any exception.
5. If the comparison becomes invalid or telemetry is partial, the state is `INCONCLUSIVE`; promotion never fills missing data with an estimate.

---

## 23. User experience

In the private beta, this experience is delivered through CI summaries/annotations, CLI output, and JSON/JSONL artifacts. The “view” examples define the information model and may be rendered in a later web UI, but they do not imply that the UI is an MVP gate.

### 23.1 Onboarding

The active POC generates four portable repository files once and commits them:
`buildoptw`, `buildoptw.bat`, `.buildopt/wrapper.properties` and
`.buildopt/config.toml`. The repeated local and CI command is
`./buildoptw <gradle args...>`. The wrapper verifies a pinned BuildOpt
distribution, discovers the repository's Gradle Wrapper and obtains credentials
only from private runtime state.

The first invocations use native Gradle and may contribute bounded observation
evidence. Trusted CI may run isolated trials within the fixed learning budget.
Only exact qualified decisions become active; drift, missing authority, expired
state, service failure and `BUILDOPT_BYPASS=1` retain native Gradle. A cache hit
does not authorize an action, and no mode owns Test Optimization behavior.

The prior install-plus-`buildopt optimize` flow remains a lower-level maintainer
surface and implementation foundation. The wrapper experiment must prove that
the committed surface removes global installation and manual profile steps
without hiding the complete cost of learning or fallback.

### 23.2 Main view

```text
Repository: payments-platform
Mode: VERIFIED
Window: last 14 days

Candidate p50 / control p50: 4m 16s / 8m 42s
Observed net p50 delta:       -4m 26s (-50.9%)
Candidate p95 / control p95: 6m 58s / 12m 11s
Observed p95 delta:           -5m 13s
Effect interval 95%:         4m 01s .. 4m 47s saved/build
Measurement:                 randomized control cohort, n=420
Measurement coverage:        97.9% complete, 2.1% excluded
Observed saving in window:   187 customer-visible build-hours
Product overhead p95:        184 ms (0.07%)
Estimated monthly CI saving: 1,240 compute-hours (model, not observed)

Active actions: 12
Canary actions: 2
Learning: 7
Suspended: 1
```

### 23.3 Action detail

```text
Action: Cache :openapi:generateClient
State: ACTIVE_IN_CI

Evidence:
  qualification: REVIEWED_ADAPTER frontend-bundle-v2
  cache contract digest: sha256:...
  repeatability contract: adapter-guaranteed-v2
  declared inputs registered in Gradle: yes
  18 equivalent executions
  4 distinct runners
  3 distinct workspace paths
  0 output mismatches
  hermetic canary denied undeclared access
  trace complete: yes
  revocation epoch: 42

Impact:
  median execution: 47.2 s
  median restore: 1.4 s
  incremental action overhead: 0.32 s/build
  observed net saving: 45.6 s/build, paired control
  effect interval 95%: 42.1 s .. 48.9 s saved/build
  p95 delta: -43.8 s
  sample: 18 candidate / 18 control, 0 excluded
  net infrastructure value: positive after validation compute

Invalidates on:
  task implementation
  generator version
  JDK major version
```

### 23.4 Explainability

Every action must answer:

- What changed?
- Why was it safe?
- What evidence was used?
- How much time did it actually save?
- When will it be revalidated?
- How is it rolled back?

---

## 24. Implementation roadmap

The active roadmap is POC-first. The functional mechanisms and combined public path have completed the bounded value gate against optimized native Gradle. The completed decision sequence is:

```text
native Gradle baseline
  → mechanism attribution
  → combined product benchmark
  → no-value disablement
  → continue/stop decision
```

The historical Phase 0 and MVP sections below retain the architecture and possible productization gates. Their names do not describe current POC qualification. The positive synthetic result supported one explicitly chosen public-repository replication phase, but did not automatically activate soak, external pilots, operational scale, product modes, or GA hardening.

That public-repository replication is complete:

```text
pin and audit released public repositories
  → prove native and installed-BuildOpt compatibility with identical outputs
  → preregister representative no-change and source-change comparisons
  → run paired optimized-native versus BuildOpt measurements
  → broaden the claim only for repositories and change classes that reproduce value
```

The matrix qualified Mockito but not Spotless or SpotBugs, so the decision is
`RETAIN_BOUNDED_SYNTHETIC_CLAIM`. This is a terminal POC result for the current
hypothesis: no unchanged rerun, product tuning, or broader claim follows from
it. Another experiment requires a separately preregistered value hypothesis,
not a retrospective change to workloads or thresholds.

The exact-workflow diagnostic supplied separate hypotheses without moving the
gate. Startup and configuration were below 1.6% for all three repositories.
Spotless advances to a leaf-project Build Impact comparison against its
optimized native Gradle configuration, preserving the real `testClasses`
request as well as production outputs. Mockito advances to an isolated
`:mockito-core:testClasses` comparison because its test compilation is
build-owned and occupied 242.690 seconds; BuildOpt must beat the optimized
native cache rather than treating parity as success. A qualifying isolated
result must still reproduce net value in the exact workflow with every test
unchanged. SpotBugs receives no build-preparation action because its visible
`compileTestJava` cost is only 1.119 seconds and the dominant `Test` execution
remains owned by Test Optimization. Every uncertainty falls back to the
complete upstream workflow.

Public repositories are evidence inputs, not design partners. Their build code
runs only from exact revisions in disposable homes without host credentials,
build scans, publishing tasks, or repository-specific CI opt-ins.

### Phase 0: executable contracts and fixtures

**Deliverables:**

- Normative `CONTRACTS-001` package from section 29: JSON Schema, OpenAPI, local Protobuf, canonicalization, errors, idempotency, N/N-1, Go/Java clients, and golden vectors.
- Versioned schemas and independent lifecycles for `BUILD_SESSION`, `EXPERIMENT_RESULT`, task, evidence, policy, provenance/attestation, `CommitDecision`, patch bundle, and action ledger.
- `METRICS-001` catalog, neutral measurement envelope, `WorkUnitsFingerprint`, versioned baseline, outcome classes, pre-outcome propensities, A/A tests, and beta promotion/p95/p99 rules.
- `HttpBuildCache` contract and gateway/backend protocol for attempts, pending, commit/abort, revocation, commit CAS, atomic `CommitDecision`, and `l1SecurityGeneration`.
- Local rendezvous compatible with Configuration Cache, spool verified before `200`, limits of 100 MiB/100 GiB per repository/500 GiB per deployment, TTL/watermarks/SLRU, and conservative recovery.
- Beta implementations behind interfaces: content-addressed filesystem + separate WAL-mode `cache.sqlite`/`control.sqlite`, read/read-write tokens, and local Ed25519 key; conformance tests fix their semantics without turning them into public APIs.
- Default-deny allowlist for tasks/plugins/versions and fail-closed behavior for unknown artifact transforms.
- Deployable `TESTOPT-API-001` contract, `TestCacheGrant`, and deliverables/checks manifest.
- `CI-ORCH-001` topology, durable lifecycle, scheduler/budget, and cancellation/crash fixtures.
- `BANDIT-001` and `PATCH-BUNDLE-001` contracts, even though their execution is enabled in B/C4.
- Tier 1 fixture repositories; TestKit for plugin/adapters; real Gradle Wrapper for agent/Configuration Cache; Linux harness for namespaces, child processes, denial, and `traceComplete`.
- `GRADLE-CORR-001`, agent, helper, and patcher spikes described in 29.3.
- JSON/JSONL, redaction profiles, beta data lifecycle, and bounded loss/buffering.
- GitHub Actions as the first fixture through `buildopt run --`, preserving argv, signals, artifacts, and exit code; OIDC is explicitly outside the gate.
- Pinned Gradle 9.6.1/JDK 21/Linux x86-64 golden lane and the end-to-end walking skeleton from 29.4.
- `DEPLOY-001` topology, `OPS-001` benchmark/fault matrix, minimum supply chain, checksums, release signing, SBOM/provenance, bypass, and kill switch.
- Java 17 and Go as the primary stacks; Rust limited to C1's experimental Linux Hermetic Helper.

**Exit gate:** section 28 decisions closed for the private beta; applicable foundational row in 29.3 closed; validated schemas/catalog/golden vectors; green walking skeleton; executable conformance/fixtures first in the golden lane and then in Tier 1 according to the module gate. No later choice of OIDC, KMS, multi-tenancy, HA, RPO/RTO, UI, or exporter blocks A0.

### MVP-A0: foundation and internal pilot

**Depends on:** Phase 0. Not customer-facing.

**Includes:** the already-tested walking skeleton, Launcher, gateway, Tier 1 plugin, managed L1, internal single-node Shared, default-deny allowlist, dependency cache, locally authenticated policy, neutral measurement envelope, and JSON/JSONL `BUILD_SESSION`/`EXPERIMENT_RESULT` export. Only exact allowlisted tasks/transforms may use Shared; unknown disables our Build Cache for the invocation.

**Exit gates:**

1. Green conformance per Tier 1: hit/miss/PUT, `413`, redirects, timeout, corruption, modified built-in, custom task, and unknown transform.
2. L2 hit → L1 → revocation → miss/rotation in the next build; a writer with an aborted attempt leaves no local or remote hit.
3. Gateway restart/rotation works with Configuration Cache without serializing upstream credentials; concurrent slots do not cross policy or namespace.
4. Spool verifies the complete payload before `200` and covers full disk, concurrent reservation, cancellation, late checksum, and crash cleanup.
5. Concurrent attempts prove commit CAS and `CommitDecision + COMMITTED` atomicity; a later `control.sqlite` failure reconciles by digest. An orphaned blob, record without blob, and expired lease become a miss, never a hit.
6. In no-hit builds, p95 overhead is ≤500 ms **and** ≤2% for sessions ≥5 s; below 5 s, it is ≤100 ms or L2 is omitted.
7. JSON Schema distinguishes complete/partial records, and `BUILD_SESSION` does not incorporate future aggregate effects.
8. Without `TestCacheGrant`, no root/composite `Test` task consumes or produces entries.
9. An internal pilot shows net causal savings including overhead and regressions; hit rate alone does not pass the gate.

### MVP-A1: Autonomous Cache — isolated external private beta

**Depends on:** stable A0 and the `OPS-001/A1` operational profile for the pilot deployment.

**Includes:** one `PRIVATE_BETA_ISOLATED` deployment per pilot, opaque read/read-write tokens, protected workflow, quotas/SLRU, beta data lifecycle, runbook, kill-switch support, and GitHub Actions as the first tested CI. It does not include shared multi-tenancy, OIDC, KMS/HSM, DSSE, HA, or a contractual SLO.

**Exit gates:**

1. Negative tests prove scopes by repository/namespace, stable/quarantine/control separation, and no tokens in forks; tokens are stored hashed and revocation takes effect before the next build.
2. Fault/soak meets published acceptance targets without presenting them as an SLO; a flood, large object, or full disk opens the circuit breaker and preserves the Gradle build.
3. A one-hit scan remains in `probation`; the 85% high watermark evicts down to 75%, and pending/quarantine do not directly evict stable.
4. Restart remains fail-closed until blobs, commits/revocations/tombstones are reconciled; corruption or lost cache metadata become misses. If `control.sqlite`, the local key, or monotonic state cannot be recovered, the system disables all actions, rotates policy/`namespaceGeneration`/`l1SecurityGeneration`, and serves no prior object.
5. Export/redaction and deletion cover managed copies in the beta deployment; `diagnostic` remains opt-in.
6. At least one external pilot exceeds the conservative causal-benefit bound and keeps p95 within budget; feedback/queue are validated when the adapter marks them `EXACT`, or remain explicitly `UNAVAILABLE` with no claim and additional compute within the limit.

### MVP-A2: self-hosted single-node private beta

**Depends on:** stable A1 protocol; may be developed in parallel with B.

**Includes:** the same isolated profile packaged for pilot infrastructure, health/readiness, declarative configuration, compatible migration, and runbook. It promises no HA, RPO/RTO, or enterprise identity.

**Exit gates:** reproducible installation; filesystem/SQLite never reside on unsupported storage; restart/upgrade does not serve partial objects; a manual restore rotates `namespaceGeneration` when it cannot prove revocation continuity.

### MVP-B: Runtime Optimizer and safe adaptive learning

**Depends on:** stable A1, valid A/A measurement, `CI-ORCH-001`, and `OPS-001/B` with an `EXACT` GitHub queue adapter; it does not depend on multi-tenancy or OIDC.

**Includes:**

- Separate candidate/control/stable workspaces/namespaces and a shadow, canary, fallback, and rollback framework.
- Local Configuration Cache when the environment is compatible; policy is distributed, never entries.
- Worker/heap autotuning, contractual removal of `clean`, allowlisted invocation merging, and policy prefetch.
- Initial randomized fixed assignment; then a contextual bandit limited to already-qualified reversible arms, with propensities and permanent control.
- Weekly 5%/daily 10% compute budget and kill switch.

**Exit gates:**

1. A/A, sample-ratio, late reward, and propensity logging pass before enabling the bandit; any failure returns to fixed assignment.
2. Candidate/control share no writable state, and intentional contamination never reaches stable.
3. Autotuning reduces `customerVisibleBuildMs` under beta policy, preserves p95/p99/queue, and satisfies cgroup, OOM, and rollback constraints.
4. Merging and `clean` removal pass fixtures for failures, finalizers, side effects, and CI barriers.
5. The bandit never authorizes cacheability, patches, graph omission, or releases; it chooses only among arms declared safe.
6. Every active action retains 5% control or equivalent revalidation and is suspended for drift, regression, or incomplete telemetry.

### MVP-C1: Task Intelligence, JVM Agent, and Linux hermeticity

**Depends on:** B and a demonstrated control pipeline; it does not depend on identity hardening.

**Includes:**

- Opt-in Java 17 JVM Instrumentation Agent for capturable I/O/NIO, environment/system properties, processes, network, clock, and randomness.
- Experimental Rust Linux Hermetic Helper supervised by Go, with capability probe and `ENFORCE_ISOLATED` over the process tree.
- Custom-task qualification through official contract, reviewed adapter/source patch, or continuous `HERMETIC_PRODUCER_PROFILE`, plus separate repeatability and relocatability gates.
- Real manifest registration as Gradle inputs/outputs, native keys, and an evidence store without restorable payloads during discovery.
- Task-to-key correlated pending/commit, automatic suspension on discrepancy, and repair through a new contract version.

**Exit gates:**

1. Mutation tests for every input change the native key or invalidate the task; a key is not reused after registered content changes.
2. No task is enabled solely from historical outputs, and no `Test` task enters Build Optimization qualification.
3. An undeclared access in `TRACE_OBSERVE` lets baseline finish, suspends the contract, and aborts pending; in `ENFORCE_ISOLATED`, it invalidates only candidate.
4. Child processes, native executable, network, external paths, unmediated clock/randomness, and a kernel without capabilities test every `traceCoverage` dimension; any incomplete required dimension makes `traceComplete=false` and the result `INCONCLUSIVE`.
5. A task qualified only through hermeticity runs its work in a dedicated task-specific producer; every trusted producer uses the same profile digest and never writes stable outside enforcement. A whole-invocation sandbox does not pass this gate.
6. `GRADLE-CORR-001` demonstrates exact task ↔ key ↔ PUT association in the supported combination; any `UNATTRIBUTED` event aborts the attempt and disables selective publication.
7. Artifact validation uses exact bytes/tree except through a versioned adapter; Test Optimization completes `FULL_RELEVANT_VALIDATION` when required.
8. JVM Agent or helper crash/fault injection preserves baseline, disables that compatibility class, and does not contaminate the daemon/L1 of later builds.
9. At least one real pilot custom task moves UNKNOWN → OBSERVING → CONTRACT_QUALIFIED → QUARANTINE_VALIDATED → ACTIVE without falsifying inputs and produces causal savings.

### MVP-C2: optional Edge Cache Node

**Depends on:** stable A1 protocol. Does not block the private-beta functional target.

**Includes:** nearby proxy, byte-based SLRU, offline reads of committed objects, and pending replication. Shared remains the sole commit authority; Edge does not promote offline and stops serving without current revocation state.

### MVP-C3: optional conservative Build Impact Analysis

**Depends on:** B and `INT-001`. Does not block the private-beta functional target.

**Includes:** shadow and selection only among customer-owned entrypoints; never test selection. Retains `BIA-002`: ≥30 days, ≥3,000 decisions, controls/coverage, zero known false negatives, and one-sided UCB within the limit; insufficient evidence is `INCONCLUSIVE`.

### MVP-C4: PR-only Patch Autopilot

**Depends on:** B and `PATCH-BUNDLE-001`; patches derived from custom tasks also depend on C1, and any patch requiring tests depends on `TESTOPT-API-001`.

**Includes:** general patch-bundle infrastructure and the two initial recipes `ARCHIVE_REPRODUCIBILITY_KOTLIN_DSL_V1` and `CUSTOM_TASK_CONTRACT_JAVA_V1`. Additional input/output declarations, Groovy DSL, eager→lazy build logic, Configuration Cache, annotation processors, and other versioned transformations are added after those recipes pass their gates. Every patch is validated in an isolated workspace and may persist only as a pull request bound to `sourceRevision`, `sourceStateDigest`, and patch digest.

**Exit gates:**

1. A patch is never automatically rebased, never modifies an existing/default branch, and never auto-merges; only the customer-side workflow may create the ephemeral `buildopt/<actionId>` head and draft PR. Source-state divergence returns it to `PROPOSED`.
2. Candidate/control pass clean and incremental compilation, relocatability, exact artifact validation or explicit adapter, Configuration Cache, and `FULL_RELEVANT_VALIDATION` when applicable.
3. The PR contains the change, evidence, risk, validation, expected impact, and rollback; it includes no captured source, secrets, or unauthorized diagnostics.
4. The draft PR may open with `PRELIMINARY` impact after correctness gates and at least one paired control; it is not counted as confirmed causal savings until it reaches `MEASURE-001`. The product neither marks the PR ready nor assumes its checks started: the maintainer retains workflow approval and the review decision.
5. After merge, the product does not pretend Git provides percentage rollout. Natural builds are monitored and, within budget, isolated controls on the same revision apply the inverse patch only when that operation is exact; if no comparable control exists, the post-merge effect is labeled contextual. A regression generates a revert PR or precise instruction, never a silent push.
6. At least one accepted pilot patch causally reduces build time without divergences, and the customer can audit the complete evidence → patch → validation → effect chain.
7. The patcher passes `PatchBundle v1` golden/negative vectors, applies `actionId + bundleDigest` idempotently, and recovers branch-without-PR without fuzz, force-push, or content execution.

### GA-D: production hardening

Begins after functional value is demonstrated; it does not block the private beta:

- Shared multi-tenancy and negative cross-tenant tests.
- OIDC/workload identity, ephemeral tokens, RBAC/SSO, KMS/HSM, and DSSE/in-toto attestations verifiable outside the data plane.
- HA object storage + metadata store, backups, recovery order, RPO/RTO, migration from beta provenance, and operational SLO/error budget.
- Per-tenant encryption/keys, legal hold, residency, replica/backup deletion, and contractual audit retention.
- Hardened API/export, OTLP/Prometheus/object-storage adapters, rate limits, and secure webhooks.
- Supply chain and operational support for a general customer-facing offering.

### Later expansion

- Direct writes to existing/managed long-lived branches or auto-merge under separate authorization.
- macOS/Windows hermeticity and expanded Android/Kotlin Multiplatform matrices.
- Specialized caches only after measuring demand and incremental savings over Gradle cache.
- Self-hosted HA clusters, geographic replication, measured TinyLFU, and advanced organizational policies.

---

## 25. Recommended MVP

The first external milestone within the private beta is **MVP-A1**, deployed in isolation; it is neither a production-ready SaaS nor the beta exit gate. It installs the Launcher, gateway, and plugin; operates L1/L2; applies the allowlist; exports observability; and demonstrates causal savings with simple credentials. It allows learning to begin with a design partner while B, C1, and C4 are added.

The **complete functional private-beta MVP** is `A1 + B + C1 + C4`:

- A1 avoids work through caching and measures end-to-end impact.
- B automatically learns and applies runtime configurations through valid experimentation and a safe bandit.
- C1 discovers, repairs/qualifies, and caches custom tasks using the JVM Agent and Linux enforcement when necessary.
- C4 turns evidence into validated pull requests instead of returning a passive recommendation list to the customer.

The private beta is not declared functionally complete and does not reach its exit gate until all four blocks have passed their respective gates; deploying A1 first sequences the pilot and does not exclude B, C1, or C4 from beta scope.

A2, Edge, and Build Impact Analysis are optional tracks; GA-D adds authentication, multi-tenancy, HA, and operation at scale. This sequence prioritizes proving real build-time reduction and closing the autonomous loop before investing in production-ready infrastructure.

Starting with A0, the complete build session is measured against a versioned pre-product baseline, and control is reserved for drift. The product communicates success only when it lowers `customerVisibleBuildMs` including overhead, maintains `customerVisibleFeedbackMs`, queue, and guardrails, and separates estimation, causal effect, and cost. Hit rate explains the mechanism but does not replace that proof.

No capability is announced before its gate: custom-task caching requires C1, Patch Autopilot requires C4, and the bandit does not leave B's safe arms. Likewise, the private beta is not presented as multi-tenant, highly available, or resistant to a compromised backend.

---

## 26. Example repository evolution

```text
MVP-A0/A1 / onboarding and isolated private beta
  The Launcher verifies a policy authenticated with the deployment's local key.
  main writes only allowlisted outputs with beta provenance through the gateway; its L1 is disabled while producing pending.
  Pilot runners verify checksum/provenance and restore from L2; forks receive no tokens.
  Test tasks do not use cache without a Test Optimization grant.
  The dependency cache is mounted read-only and every runner maintains its writable layer.
  BUILD_SESSION closes when each build ends; EXPERIMENT_RESULT later publishes the aggregate causal effect without rewriting it.

MVP-B / after its gates
  Candidate and control run in isolated workspaces and namespaces.
  On a persistent runner, Configuration Cache creates a natural hit.
  Local receives the policy and creates its own entry.
  A/A and fixed cohorts validate reward; then the bandit adjusts workers from 8 to 6 and proves causal savings.

MVP-C1 / custom-task contract
  The JVM Agent proposes a manifest for generateOpenApi.
  A new access lets baseline finish, aborts pending, and suspends caching.
  A reviewed adapter declares and registers all inputs/outputs before lookup.
  Gradle calculates a new native key with normalized paths and content hashes.
  The repeatability gate is independent of input coverage.
  If hermeticity is the sole authority, the task runs in a dedicated producer and the Rust helper applies the same fail-closed profile to every trusted writer.
  Canary uses quarantine; a trusted writer rebuilds the stable entry.

MVP-C4 / Patch Autopilot
  Evidence of an incomplete declaration generates a patch bound to sourceRevision/sourceStateDigest.
  Candidate and control pass artifact validation and FULL_RELEVANT_VALIDATION.
  The customer-side workflow creates only the buildopt/* head and opens an explainable PR; it touches no existing branches and does not auto-merge.
  After merge, natural telemetry and comparable isolated controls measure the effect and monitor drift.

MVP-C3
  Build Impact Analysis evaluates the deliverables manifest in shadow.

GA-D
  The deployment migrates tokens/local key to workload identity and hardened attestations.
  Multi-tenancy, HA stores, and SLOs are enabled only after their own gates.
```

This sequence is illustrative. Promotion depends on contracts and gates, not on a fixed number of executions or an isolated confidence score.

---

## 27. Risks and mitigations

| Risk | Mitigation |
|---|---|
| Incorrect cache hit | Official contract/reviewed adapter/source patch or continuous enforcement, separate repeatability gate, Gradle inputs, and control |
| Undeclared input discovered in production | Allow baseline, abort pending/disposable L1, revoke contract, and repair before reactivation |
| Sandbox failure or candidate breaks CI | `ENFORCE_ISOLATED`, independent workspace, and baseline exit code/deliverables as the visible result |
| Incomplete key reused through an alias | Create no observed keys or aliases; discovery stores evidence and Gradle calculates a native key after inputs are registered |
| Nondeterministic outputs | Reject caching or correct the generator; repeated comparison only detects and does not prove absence |
| Cache poisoning or compromised backend | Beta: backend/local key are TCB, with no fork credentials, checksum, immutable objects, and revocation; hardened adds attestation verified outside the data plane |
| Compromised authorized writer | Declare it TCB, detection/revocation/blast-radius/rebuild; optional high-assurance independent reproduction |
| L1 serves an object after revocation | `l1SecurityGeneration` for every L1 that can be populated from L2, rotation before build, and TTL bounded by freshness |
| Global Build Cache enables unknown custom tasks/transforms | Exact allowlist, `doNotCacheIf` for tasks, and managed cache off for the invocation if a transform cannot be excluded |
| Ephemeral gateway invalidates or leaves stale Configuration Cache | Stable rendezvous, upstream credential outside Gradle, and tracked `gatewayConnectionGeneration` |
| Gateway delivers bytes before verification | Bounded spool, reservation/backpressure, complete checksum before `200`, and acceptance target by size; SLO only in hardened |
| Pending blocks a correct key | Candidate per attempt and CAS only on commit; abort/lease releases, poisoned rotates namespace generation |
| Control authorizes commit but cache/control state diverge | Immutable `CommitDecision`; decision and visibility persist together in `cache.sqlite`, while `control.sqlite` reconciles by digest |
| Candidate contaminates stable/control | Separate workspaces, credentials, and namespaces; `push=false` toward stable |
| Expensive or incomplete instrumentation | Sampling and budget; `traceComplete=false` preserves the build but aborts the optimization |
| PUT attributed to the wrong task | `GRADLE-CORR-001` spike, versioned local channel, and fail-closed `UNATTRIBUTED`; never timing/thread-name correlation |
| Whole-Gradle sandbox presented as task-level hermeticity | Only a dedicated producer with task-specific manifest/mounts may use `HERMETIC_PRODUCER_PROFILE` as a contractual source |
| Comparing different builds | Contextual model and equivalent cohorts |
| Control/canary improves runtime but worsens queue | `customerVisibleFeedbackMs`, `runnerOccupiedMs`, and causal queue guardrail by runner pool |
| Parallelism regression | Canary, resource limits, and rollback |
| Patch changes semantics | Isolated candidate, paired control, `FULL_RELEVANT_VALIDATION`, and artifact validation |
| Patch bundle escapes worktree or is reapplied to different source | Signed manifest, exact paths/preimages, rejection of symlink/submodule/traversal, and idempotent application without fuzz |
| Stale policy | Fingerprint, expiration, revocation epoch, and server-side suspension |
| Replay of an old signed policy | Persistent anti-rollback, freshness, repository binding, and configuration-policy digest |
| Cache latency exceeds execution | Admission/eviction per object, policy by build class, and circuit breaker between invocations |
| Full disk or scan pollution | Quotas and maximum size, TTL, high/low watermarks, and size-aware SLRU; stop admitting before capacity is exhausted |
| Excessive service dependency | Last-known-good policy and fallback to standard Gradle |
| Launcher/plugin bug after side effects | Retry without product only before task actions; isolated baseline or preserve failure and enable kill switch |
| Use of Gradle internal APIs | Versioned adapters and compatibility matrix |
| Data exposure | Prior redaction, keyed identifiers, opt-in, and on-premises option |
| Final export attributes an effect not yet calculated | Immutable `BUILD_SESSION` and append-only/versioned `EXPERIMENT_RESULT` with separate lifecycle |
| Configuration Cache leaks secrets or becomes stale | Do not share entries, separate encryption/keys, and tracked Provider read of `configurationPolicyDigest` |
| Build Impact Analysis omits a deliverable | Customer-owned manifest, history does not authorize, complete original entrypoints on unknown, and periodic control |
| Self-hosted upgrade breaks data | Beta starts fail-closed and turns uncertainty into miss/rotation; hardened adds backup/restore and contractual rollback |
| Test task restored without authorization | Prior `TestCacheGrant`; without a grant, fail-closed `doNotCacheIf` and negative gate |
| Security metadata lost | Start without reads, actions off, and policy/namespace/L1 rotation in beta; hardened adds HA store and backup/recovery order |
| Edge diverges or operates offline | Shared is sole commit authority; offline serves only committed objects with current freshness |
| Noisy tenant exhausts CPU/network/connections | Beta removes cross-tenant sharing and applies deployment limits; hardened adds hierarchies, fair queuing, and a circuit breaker per tenant |

---

## 28. Product decisions

### 28.1 Closed decisions

| ID | Status | Decision |
|---|---|---|
| POC-001 | Accepted | The active program is an owner-operated POC on controlled repositories; soak, external design partners, production promotion samples, HA, enterprise identity, and multi-tenancy do not block it, and private-beta milestone names are not reused for weaker POC evidence |
| VALUE-001 | Accepted | The POC continues only when the complete BuildOpt path beats a well-configured native Gradle control with identical required outputs and no additional failures; mechanisms are measured separately, parity is not acceleration, percentages are not added, and no-value actions remain disabled |
| CONTRACTS-001 | Accepted | Phase 0 materializes normative schemas/IDL, canonicalization, errors, idempotency, N/N-1 compatibility, generated clients, and test vectors; RFC examples are not implementation contracts |
| BETA-001 | Accepted | The private beta reduces the operational boundary through one isolated deployment per tenant but validates the A1+B+C1+C4 functional loop; production hardening neither replaces nor blocks those capabilities |
| CACHE-001 | Accepted | Offer a first-party Gradle-compatible Shared Cache Backend; isolated managed and self-hosted single-node private beta, with multi-tenant SaaS only after hardening |
| CACHE-002 | Accepted | Manage `DirectoryBuildCache` and its native cleanup; every L1 that can be populated from L2 is segmented/rotated by `l1SecurityGeneration`; do not reimplement its format or add a proprietary local LRU |
| CACHE-003 | Accepted | Add an optional Edge Cache Node in MVP-C2 for CI hosts, clusters, and local networks |
| CACHE-004 | Accepted | Use quotas, TTL, high/low watermarks, and size-aware SLRU in Shared/Edge; no pure LFU, with TinyLFU only as a measured evolution |
| CACHE-005 | Accepted | Use a Local Verifying Cache Gateway with a Configuration Cache-compatible rendezvous and complete spool before `200`; provenance/attestations, revocation, and pending/commit travel through the internal protocol |
| CACHE-006 | Accepted | Shared is the sole commit authority; pending is identified by attempt, first-writer CAS occurs at commit, replacing poisoned data requires another namespace generation, and recovery starts fail-closed |
| CACHE-007 | Accepted | Shared is default-deny: exact task/plugin/version allowlist; an unknown artifact transform disables the managed cache for the entire invocation |
| CACHE-008 | Accepted | A canonical `CommitDecision` binds attempt, objects, checksums, policies, grants, epochs, and verdict; the decision and `COMMITTED` records persist atomically in `cache.sqlite`, without a distributed transaction with `control.sqlite` |
| LIMITS-001 | Accepted | Private beta: objects ≤100 MiB, 100 GiB per repository and 500 GiB per deployment, stable TTL 30 days, 85/75% watermarks, and pending/quarantine ≤10%; larger sizes require another benchmark class |
| OBS-001 | Accepted | Make all observability exportable through an open versioned schema, beginning with JSON and JSONL |
| OBS-002 | Accepted | Separate immutable `BUILD_SESSION` records from versioned `EXPERIMENT_RESULT` and `ACTION_RECORD`; a final build contains no future aggregate effects |
| EXPORT-001 | Accepted | The private beta exports JSON/JSONL and CI artifacts; OTLP, Prometheus, Parquet/object-storage sinks, and webhooks do not block beta |
| METRICS-001 | Accepted | Primary success is causally reducing `customerVisibleBuildMs`, net of overhead, with `customerVisibleFeedbackMs`, queue, p95/p99, and correctness as guardrails; hit rate, actions, and avoided tasks are drivers |
| MEASURE-001 | Accepted | Beta policy: 7 days/100 per arm for reversible actions and 14 days/200 for proof-gated actions; one-sided 95% LCB ≥ max(500 ms, 2%), p95 guardrail max(500 ms, 3%), p99 gate from 1,000 per arm, 5% stable control, and at most 28 days of additional compute |
| INT-001 | Accepted | Build Optimization owns build/cache and Test Optimization owns tests; without prior `TestCacheGrant`, test tasks do not use cache; patches require `FULL_RELEVANT_VALIDATION` |
| TESTOPT-API-001 | Accepted | REST/JSON OpenAPI integration, signed grants/results, idempotent `actionId`, bounded polling, and content-addressed artifact refs; absence or incompatibility produces `INCONCLUSIVE` |
| COMPAT-001 | Accepted | Initial Tier 1: Gradle 8.14.x with JDK 17/21 and Gradle 9.6.x with JDK 17/21/25 on Linux x86_64 |
| GOLDEN-LANE-001 | Accepted | First vertical slice: Gradle 9.6.1 + JDK 21 + Linux x86_64 + Kotlin DSL + 4-vCPU/16-GiB development runner on host and pinned container; GitHub Actions, expanded Tier 1, and benchmark runners are later gates |
| TASK-001 | Accepted | Observation is never sufficient; require an official contract, adapter/source patch, or continuous producer enforcement, plus separate repeatability and relocatability gates |
| TASK-002 | Accepted | On discrepancy, abort pending, increment revocation, and repair with Gradle inputs; there are no observed keys, discovery payloads, or aliases |
| GRADLE-CORR-001 | Accepted | C1 does not publish selectively until a spike proves `taskExecutionId → cacheKey → PUT` correlation; any ambiguity is `UNATTRIBUTED` and aborts the attempt |
| DIAG-001 | Accepted | Add an opt-in Java 17 JVM Instrumentation Agent in MVP-C1: capturable hooks degrade and optimization is fail-closed; a fatal failure follows SAFETY-002; it is neither a sandbox nor proof of hermeticity |
| HERMETIC-001 | Accepted | Add an experimental Rust Linux Hermetic Helper supervised by Go in C1; `TRACE_OBSERVE` never blocks baseline and `ENFORCE_ISOLATED` covers candidates/producers, with incomplete coverage = `INCONCLUSIVE` |
| HERMETIC-SCOPE-001 | Accepted | The helper qualifies a task through hermeticity only when its work runs in a dedicated producer with task-specific manifest/mounts and unambiguous correlation; sandboxing all of Gradle provides only build-level evidence |
| ARTIFACT-001 | Accepted | Generic equality is exact bytes/tree only; every semantic equivalence requires a versioned adapter, beginning with reproducible JAR/ZIP and POM/Gradle Module Metadata |
| SAFETY-001 | Accepted | `TRACE_OBSERVE` does not block baseline; enforcement occurs in isolation, and high-correction actions do not publish candidate before control/validation |
| STACK-001 | Accepted | Java 17 for plugin/agent, Go for Launcher/data/control plane, and Rust only for C1's experimental Linux hermetic helper; Rust is neither a core requirement nor a source of hermeticity by itself |
| ROLLOUT-001 | Accepted | Candidate, control, and stable isolate state; instrumented writes use pending and only a complete verdict allows commit |
| ROLLOUT-002 | Accepted | Beta: direct 5→25→50→95% and proof-gated 1→5→25→50→95% after shadow/contract, both with 5% control; patches use paired candidate/control and PR, without pretending there is partial post-merge rollout |
| BUDGET-001 | Accepted | Additional learning compute ≤5% of natural runner-minutes over 7 days, burst ≤10%/24 h, and one concurrent validation per repository; exhausting it leaves the action inconclusive and never relaxes gates |
| LEARN-001 | Accepted | The contextual bandit is part of B, activates only after valid A/A and fixed assignment, records propensities, and selects only already-qualified reversible runtime arms |
| BANDIT-001 | Accepted | Beta uses contextual epsilon-greedy over a finite catalog of resource profiles, 10→2% exploration, permanent control, bounded late outcomes, hard stop on failure/OOM, and reset on drift/measurement epoch |
| STATE-001 | Accepted | `TaskQualificationState` and `ActionRolloutState` are separate machines; `INCONCLUSIVE` does not promote, and suspending a contract suspends its dependent actions |
| CC-001 | Accepted | Distribute the decision, not entries; bounded `configurationPolicyDigest` is a configuration input, and every trust domain creates its local cache |
| BIA-001 | Accepted | Only a customer-owned manifest authorizes selecting among declared entrypoints; history does not authorize, and unknown returns to the complete original entrypoint |
| BIA-002 | Accepted | Promotion requires the defined window, sample, coverage, and maximum UCB; insufficient evidence is `INCONCLUSIVE`, and one false negative suspends |
| AUTH-001 | Accepted | Isolated private beta: TLS, scoped opaque read/read-write tokens stored hashed with a 30-day maximum, forks without tokens, and provenance/policy using a local Ed25519 key; OIDC/KMS/DSSE/RBAC are deferred |
| SEC-001 | Accepted | Forks receive no credentials; beta treats Shared/local key as TCB and separates repositories/namespaces; verification outside the data plane and multi-tenant isolation are hardening gates |
| STORAGE-001 | Accepted | Single-node private beta: content-addressed blobs + separate WAL-mode `cache.sqlite` and `control.sqlite`, transactional CAS, and fail-closed recovery that rotates generations; object store, HA database, and RPO/RTO are deferred |
| PRIVACY-001 | Accepted | Beta: cache 30 days, evidence 30, summary 30, diagnostic 7 opt-in, and audit 90; no source/secrets/raw paths or silent legal hold |
| PATCH-001 | Accepted | Patch Engine enters in C4: a customer-side workflow may create only the `buildopt/<actionId>` head and draft PR bound to revision/source state; backend has no Git token and does not touch existing branches, auto-rebase, or auto-merge |
| PATCH-BUNDLE-001 | Accepted | Signed declarative JCS bundle, exact path/pre/post digests, no scripts/fuzz/symlink escape, `actionId + bundleDigest` idempotency; C4 begins with two fixture-backed recipes |
| ECOSYSTEM-001 | Accepted | The private beta adds no specialized caches: first demonstrate the impact of Gradle cache and task intelligence; choose other ecosystems later using telemetry |
| CI-001 | Accepted | Portable `buildopt run -- <argv>` CLI and GitHub Actions as the first fixture; cache tokens remain in secrets and C4 uses `GITHUB_TOKEN` only in the customer-side workflow for head/draft PR, without assuming recursive CI and deferring OIDC/GitHub App |
| CI-ORCH-001 | Accepted | The normal job executes one authoritative arm; high-correction validations are queued and run automatically in a separate protected workflow with isolated workspaces/state and a durable lifecycle |
| DEPLOY-001 | Accepted | Beta distributes `buildopt`, modular-monolith `buildopt-server`, plugin/agent JAR, optional Rust helper, patcher JAR, and workflows; initial customer-facing UX is CI summary + JSON/JSONL, not a web UI |
| OPS-001 | Accepted | Reproducible beta benchmark, health/readiness, revocation ≤60 s, alerts, independent bypass, rollback/uninstall, and exact GitHub queue adapter before B |
| SAFETY-002 | Accepted | Transparent retry without product only before task actions; afterward, side effects are not blindly repeated and baseline is used only if already isolated/authorized |
| MVP-001 | Accepted | A0 is the internal pilot, A1 the first isolated external pilot, and the complete private-beta functional target is A1+B+C1+C4; A2/C2/C3 are optional and GA-D concentrates production-ready hardening |

### 28.2 Deferred productization decisions outside the POC

These choices are deliberately unresolved because they do not help answer the POC value question. A positive `VALUE-001` result may justify reopening them; a negative result stops or narrows the experiment before this work is funded.

| Later gate | Deferred decision |
|---|---|
| GA-D Identity | OIDC provider, issuer/audience/claims, final DSSE/in-toto format, KMS/HSM, SSO, and RBAC |
| GA-D Storage | Object store, HA metadata database, replication, backup, RPO/RTO, and migration from SQLite/beta provenance |
| GA-D Privacy | Commercial retention, legal hold, residency, backup deletion, and regulatory exceptions |
| GA-D Export | Ordering among OTLP, Prometheus, Parquet/object storage, SIEM, and webhooks |
| Platform expansion | macOS/Windows hermeticity and supported kernel runtimes beyond Linux x86-64 |
| Ecosystem expansion | First specialized cache—Kotlin/Android/npm/BuildKit—according to demand and observed savings |
| Autopilot write | Whether any mode may write directly to existing/managed long-lived branches or auto-merge, under separate authorization and threat model |

---

## 29. Implementation readiness

### 29.1 Readiness verdict

The original Phase 0 package and walking skeleton are materialized. The combined path cleared `POC-VALUE-G01` on the qualified synthetic workload matrix. `POC-BREADTH-001` initially completed with 2/8 realistic change/DSL cells qualifying. Attribution and calibrated paired experiments raised bounded coverage while the terminal Kotlin decision retained unstable shared/build-logic cells outside the claim. Public-repository replication then qualified only Mockito out of Spotless, Mockito, and SpotBugs, so the bounded synthetic claim remains unchanged. Exact-workflow profiling rejected configuration work and exposed both Spotless's cross-project graph and Mockito's 242.690-second test compilation. The corrected boundary therefore authorizes preregistered Spotless Build Impact and Mockito test-build experiments while retaining no build-preparation hypothesis for SpotBugs. The current implementation truth lives in the tracker and executable checks.

The generic structural-profile POC and its adaptive-fragment successor are now
both stopped by their frozen terminal gates. The active successor is the
repository-committed sticky-wrapper learning POC. It reuses the implemented
launcher, packages, Gradle HTTP cache, typed central state and fail-open
controls, but grants no authority to the stopped profiles. Its first seven
implementation blocks now define the wrapper contract, generator, verified
bootstrap, passthrough, portable connection, native cache integration and a
typed decision/evidence store; the next block is the local no-op selector.
Its terminal gate requires exact outputs, zero
product failures, negligible native-retention cost, positive cumulative value,
positive confidence and payback in at least three of five public families.

The distinction is deliberate:

- Accepted decisions record architecture and safety; they do not prove implementation or value.
- Schemas, APIs, state machines, vectors, fixtures, and spikes are materialized and remain regression inputs.
- Agent discovery and hermetic producer enforcement are explicitly `UNAVAILABLE`; reviewed-source paths remain testable.
- All synthetic POC value and terminal measurement gates are closed. `POC-BREADTH-G01` remains preliminary and the claim stays bounded. `POC-REALWORLD-001..002` proved pinned compatibility and retained the bounded claim; `POC-PUBLIC-BUILD-TASKS-001` now freezes the Spotless exact-workflow Build Impact and Mockito test-build Safe Cache experiments. Productization remains a separate decision and is not implied by either the bounded `CONTINUE` verdict or public-source replication.

The sticky-wrapper successor now includes `SWL-007`, a POC-only typed decision
store. It keeps observations, action transitions, trials, signed decisions and
the economic ledger as canonical immutable control-plane records with one
generation-CAS head, while Gradle cache objects remain opaque data-plane
entries. Local files and the existing central `EVIDENCE` state adapter share
the same validation rules, idempotent replay and revocation/expiry checks; no
record grants production authority and the next block must still preserve the
native no-op path.

The normative source is divided as follows: this RFC retains intent, invariants, and gates; `contracts/`, `specs/`, `benchmarks/`, and ADRs retain executable details. If a contract contradicts a safety invariant in this RFC, the contract is corrected; if the invariant needs to change, the corresponding decision is reviewed first.

### 29.2 Phase 0 normative package

**CONTRACTS-001** is materialized with this minimum structure:

```text
contracts/
  jsonschema/
    build-session.v1.schema.json
    experiment-result.v1.schema.json
    action-record.v1.schema.json
    evidence-record.v1.schema.json
    optimization-policy.v1.schema.json
    attempt-state.v1.schema.json
    ci-validation-request.v1.schema.json
    commit-decision.v1.schema.json
    resource-profile.v1.schema.json
    test-cache-grant.v1.schema.json
    test-validation-result.v1.schema.json
    patch-bundle.v1.schema.json
    sticky-wrapper-decision-store.v1.schema.json
  openapi/
    buildopt-control.v1.yaml
    buildopt-cache-control.v1.yaml
    test-optimization.v1.yaml
  proto/
    local-events/v1/task_events.proto
  test-vectors/
    canonical-json/
    signatures/
    state-machines/
    compatibility/
specs/
  poc-value-validation-v1.{md,json}
  poc-breadth-validation-v1.{md,json}
  ci-orchestration-v1.md
  gradle-correlation-v1.md
  benchmark-beta-v1.md
  test-optimization-integration-v1.md
  patch-bundle-v1.md
  capability-matrix-v1.md
benchmarks/
  beta-v1.yaml
adr/
  0001-golden-lane.md
  0002-single-node-commit-atomicity.md
  0003-local-task-event-channel.md
```

Common normative rules:

- Exportable JSON and signed commands use JSON Schema 2020-12. Control HTTP APIs use OpenAPI 3.1/JSON. Concurrent local plugin/agent → Launcher/Gateway events use length-delimited Protobuf v3 over a Unix domain socket in the golden lane; another transport requires the same conformance suite.
- The cache payload remains opaque and uses the public `HttpBuildCache`. Attempts, revocation, provenance, and commit belong to the internal API; no private headers that Gradle must interpret are added.
- Signed JSON is UTF-8 encoded and canonicalized using JCS; duplicate keys, unrepresentable numbers, and nonconforming payloads are rejected. Artifact SHA-256 is calculated over exact bytes and represented as `sha256:<lowercase-hex>`. Dates use UTC RFC 3339. Path normalization belongs to the per-platform contract and never implicitly alters source bytes.
- Every mutation has an idempotency key and state precondition. `attemptId`, `actionId`, `policyDigest`, `bundleDigest`, and `CommitDecision` are not interchangeable. Reusing an ID with different content is a conflict, not a retry.
- Every endpoint defines deadline, cancellation, stable error codes, `retryable`, maximum backoff, and the effect of an unknown response. Only idempotent operations are retried; timeout on a positive gate produces `INCONCLUSIVE` or `ABORTED`.
- Clients and servers support the two latest minor versions N and N-1 within the same major. Signed commands reject unknown fields; exportable records may preserve/ignore additional fields according to schema. An incompatible major fails closed for optimization and preserves the Gradle baseline.
- Phase 0 generates Go/Java clients, validates that they have no uncommitted changes, and runs the same golden vectors in both languages. Every state machine includes valid, invalid, crash/retry, and recovery transitions.

Contracts will not prematurely fix table names as a public API. `cache.sqlite` and `control.sqlite` are beta implementations; their observable semantics are tested through conformance.

### 29.3 Open gates by module

| ID | Evidence required to close it | Blocks |
|---|---|---|
| `CONTRACTS-001` | Schemas/IDL above, Go/Java clients, N/N-1, and green golden vectors | Integration of A0 and all later modules |
| `CI-ORCH-001` | GitHub fixture with authoritative normal job, validation queue, isolation, cancellation/crash recovery, budget, and durable lifecycle | B and every candidate/control gate in C1/C4 |
| `GRADLE-CORR-001` | Real spike that unambiguously maps `taskExecutionId → native cacheKey → PUT` or proves the all-attempt fallback | Selective publication of custom tasks in C1 |
| `HERMETIC-SCOPE-001` | Dedicated task-specific producer, coverage vector, child/native process, and negative fixtures; complete Gradle invocation remains evidence only | C1 hermetic-qualification route |
| `BANDIT-001` | Arm/feature/reward schema, simulator/replay, A/A, delayed outcome, drift reset, and rollback | B bandit; does not block fixed cohorts |
| `PATCH-BUNDLE-001` | Parser/applier with golden vectors, traversal/symlink/submodule negatives, idempotency, and branch-without-PR recovery | C4 materialization |
| `TESTOPT-API-001` | Mock producer/consumer, trust/version/retry/grant/artifact fixtures, and conformance between both products | Test cache grant and patches requiring `FULL_RELEVANT_VALIDATION`; does not block A1 with uncached tests |
| `DEPLOY-001` | Installable/versioned artifacts, upgrade/uninstall, and end-to-end modular-monolith fixture | First external A1 pilot |
| `OPS-001` | A1 profile: reproducible harness, fault/soak, readiness/alerts, revocation, bypass, and runbooks; B profile: exact queue adapter | `OPS-001/A1` blocks external A1; `OPS-001/B` blocks B |
| `GOLDEN-LANE-001` | Toolchain/runner-image lock and green Gradle 9.6.1/JDK 21/Linux x86-64/4-vCPU/16-GiB fixture | First walking skeleton |

`GRADLE-CORR-001` is tested with Gradle 9.6.1 first and then a pinned 8.14.x version. The matrix includes parallel tasks, two tasks with equivalent outputs, cache hit/miss, in-process and process-isolated Worker API, child processes, cancellation, failure, and Configuration Cache. Every PUT must associate with one execution and outcome. If the association is missing or admits more than one task, the event is `UNATTRIBUTED`; the capability is marked `UNAVAILABLE` for that combination and the entire attempt is aborted. It is not inferred from temporal order, thread name, or task path.

Three bounded spikes also run before committing to the complete C1/C4 implementation:

- `SPIKE-AGENT-001`: Java agent with I/O/NIO, environment, processes, and network on a real daemon, transformation conflicts, buffer overflow, crash, and Configuration Cache matrix.
- `SPIKE-HERMETIC-001`: Rust helper with capability probe, mounts, process tree, native child, network, unmediated clock/randomness, and task-specific producer.
- `SPIKE-PATCHER-001`: apply two C4 recipes over exact preimages, repeat idempotently, reject a divergent worktree, and recover a branch created without a PR.

A spike may close a capability as `UNAVAILABLE`; that is not a Phase 0 failure if the product degrades according to this RFC. It does prevent announcing or enabling the corresponding capability.

### 29.4 First walking skeleton

The first vertical slice does not attempt to implement all optimization. It must demonstrate, in this order:

```text
GitHub Action
  → buildopt run -- ./gradlew build
  → Gradle Optimization Plugin
  → Local Verifying Cache Gateway loopback
  → buildopt-server
  → BUILD_SESSION v1 JSON
```

It runs exclusively in the golden lane with optimizations off. Done means:

1. argv, signals, exit code, and deliverables are identical to the direct Gradle command;
2. `BUILD_SESSION` validates against schema, contains neutral-envelope timestamps, and declares unavailable fields without inventing them;
3. plugin and gateway use the authenticated local channel and survive a restart before task actions;
4. local bypass executes the original command without contacting the control plane;
5. a failure/cancel fixture preserves classification and leaves no active attempt or lease;
6. overhead is measured from the first execution even though it is not yet a promotion gate.

Then implement A0 → A1 → B → C1 → C4. A2, C2, and C3 may wait. Within B, fixed cohorts precede the bandit; within C1, observation and adapters precede the hermetic route; within C4, the bundle/patcher and two initial recipes precede the broad catalog. This sequence reduces uncertain integration without removing any module from the beta target.

### 29.5 Start and completion checklist

Before opening parallel work:

- the golden lane and versions are pinned;
- owners and repositories for `contracts/`, Go, Java plugin/agent, Rust helper, and workflows are defined;
- CI runs codegen/conformance and does not allow generated-client drift;
- every module references contract IDs and does not copy structs from RFC examples;
- benchmarks and fixtures have reproducible seeds/digests;
- kill switch, bypass, and `UNAVAILABLE` classification exist before the first active optimization.

The project can begin now with Phase 0, schemas, mocks, conformance tests, and bounded spikes. It must not be presented as an “all-module MVP implementation ready to parallelize” until the applicable foundational row for every module closes.

Explicitly outside this readiness gate are OIDC/SSO, KMS/HSM, shared multi-tenancy, HA/RPO/RTO, object storage, web UI, Edge, Build Impact Analysis, macOS/Windows, and ecosystem expansion. These are later decisions and must not delay the walking skeleton or A0/A1.

---

## 30. Official technical references

- [Current Gradle release metadata](https://services.gradle.org/versions/current)
- [Gradle Build Cache](https://docs.gradle.org/current/userguide/build_cache.html)
- [Build Cache concepts](https://docs.gradle.org/current/userguide/build_cache_concepts.html)
- [Build Cache performance](https://docs.gradle.org/current/userguide/build_cache_performance.html)
- [Diagnosing Build Cache misses](https://docs.gradle.org/current/userguide/build_cache_debugging.html)
- [HTTP Build Cache protocol semantics](https://docs.gradle.org/current/dsl/org.gradle.caching.http.HttpBuildCache.html)
- [Build Cache use cases and trusted writers](https://docs.gradle.org/current/userguide/build_cache_use_cases.html)
- [Gradle User Home cache cleanup](https://docs.gradle.org/current/userguide/directory_layout.html#dir:gradle_user_home)
- [TaskInputs runtime API](https://docs.gradle.org/current/javadoc/org/gradle/api/tasks/TaskInputs.html)
- [TaskOutputs cache controls](https://docs.gradle.org/current/javadoc/org/gradle/api/tasks/TaskOutputs.html)
- [Configuration Cache](https://docs.gradle.org/current/userguide/configuration_cache.html)
- [Enabling and configuring Configuration Cache](https://docs.gradle.org/current/userguide/configuration_cache_enabling.html)
- [Configuration Cache status and sharing limitation](https://docs.gradle.org/current/userguide/configuration_cache_status.html)
- [Configuration Cache requirements, secrets and encryption](https://docs.gradle.org/current/userguide/configuration_cache_requirements.html)
- [Gradle build performance](https://docs.gradle.org/current/userguide/performance.html)
- [Gradle build environment and JVM properties](https://docs.gradle.org/current/userguide/build_environment.html)
- [Gradle command-line performance options](https://docs.gradle.org/current/userguide/command_line_interface.html#sec:command_line_performance)
- [Dependency caching](https://docs.gradle.org/current/userguide/dependency_caching.html)
- [Gradle compatibility matrix](https://docs.gradle.org/current/userguide/compatibility.html)
- [Java Library Plugin: `api` and `implementation`](https://docs.gradle.org/current/userguide/java_library_plugin.html)
- [Securing Gradle builds](https://docs.gradle.org/current/userguide/security.html)
- [Gradle public API policy](https://docs.gradle.org/current/userguide/public_apis.html)
- [Build event listener public API](https://docs.gradle.org/current/javadoc/org/gradle/build/event/BuildEventsListenerRegistry.html)
- [Java instrumentation agents](https://docs.oracle.com/en/java/javase/17/docs/api/java.instrument/java/lang/instrument/package-summary.html)
- [SQLite Write-Ahead Logging](https://sqlite.org/wal.html)
- [JSON Schema Draft 2020-12](https://json-schema.org/draft/2020-12/json-schema-core)
- [OpenAPI 3.1 specification](https://spec.openapis.org/oas/v3.1.0)
- [Protocol Buffers proto3](https://protobuf.dev/programming-guides/proto3/)
- [RFC 8785 — JSON Canonicalization Scheme](https://www.rfc-editor.org/rfc/rfc8785)
- [Linux Landlock userspace API](https://docs.kernel.org/userspace-api/landlock.html)
- [Linux seccomp filter](https://docs.kernel.org/userspace-api/seccomp_filter.html)
- [GitHub Actions `GITHUB_TOKEN` behavior](https://docs.github.com/en/actions/concepts/security/github_token)
- [GitHub Actions OpenID Connect reference](https://docs.github.com/en/actions/reference/security/oidc)
- [in-toto Attestation specification](https://github.com/in-toto/attestation/blob/main/spec/README.md)
- [in-toto DSSE envelope](https://github.com/in-toto/attestation/blob/main/spec/v1/envelope.md)
- [Incremental annotation processing](https://docs.gradle.org/current/userguide/java_plugin.html)
- [Isolated Projects](https://docs.gradle.org/current/userguide/isolated_projects.html)
- [Reproducible archive configuration](https://docs.gradle.org/current/javadoc/org/gradle/api/tasks/bundling/AbstractArchiveTask.html)

---

## 31. Conclusion

Gradle Build Optimization must be an autonomous optimization platform, not a collection of flags or a passive dashboard.

Observability will make the build understandable and explainable. The different caches will avoid repeated work. The optimization engine will reduce the graph, tune resources, and correct build logic. The validation system will enable those improvements with progressive confidence by using natural builds and avoiding manual repetitions by the developer.

The primary differentiation will be closing the complete loop:

```text
see the problem
  → identify a possible action
  → establish its contract and validate it
  → apply it automatically
  → measure or estimate the benefit without conflating them
  → keep it valid as the build evolves
```

The product will be judged by net customer-visible time saved with causal evidence, not by hit rate or enabled actions. p95/p99, correctness, overhead, and cost will bound that improvement; the versioned catalog and open export will allow it to be verified outside our UI.

The result will be a build that gets faster as it is used while retaining auditable evidence for every decision made.
