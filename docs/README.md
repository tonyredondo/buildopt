# BuildOpt documentation

This portal organizes the repository by the task a reader is trying to
complete. You do not need to read the master RFC or the implementation tracker
before running the product.

For research, start with the [current status register](./research-status.md).
It identifies the only current program, discarded directions, retained
foundations and every historical plan. Old handoffs and pending boxes do not
reopen a closed experiment.

## Choose a path

| You want to... | Start here | Continue with |
|---|---|---|
| Install and get a first result | [Product onboarding](./getting-started/product-onboarding.md) | [Product workflows](./guides/product-workflows.md) |
| Develop or review a change | [Developer onboarding](./getting-started/developer-onboarding.md) | [Repository map](./architecture/repository-map.md), [validation](./reference/validation.md) |
| Understand the system | [Architecture overview](./architecture/overview.md) | [Glossary](./glossary.md), [master RFC](../gradle-build-optimization-platform.md) |
| Add BuildOpt to CI | [CI integration](./guides/ci-integration.md) | [Configuration reference](./reference/configuration.md) |
| Operate self-hosted or Edge | [Operations guide](./guides/operations.md) | [Runbooks](../runbooks/README.md) |
| Diagnose a problem | [Troubleshooting](./troubleshooting.md) | [CLI reference](./reference/cli.md) |
| Decide next-quarter investment | [Build Optimization investment review](./findings/buildopt-next-quarter-investment-review-2026-09-10.md) | [Supporting evidence and experiment map](./findings/buildopt-next-quarter-evidence-2026-09-10.md) |
| Check what to pursue or discard | [Research status register](./research-status.md) | [Current tracker](./plans/buildopt-product-viability-v1-tracker.md), [evidence ledger](./plans/buildopt-product-viability-v1-evidence.md) |
| Review historical POC mechanisms and measured value | [Historical POC one-pager](./findings/buildopt-poc-handoff.md) | [Detailed performance findings](./findings/build-optimization-performance.md), [benchmark evidence](../benchmarks/README.md) |
| Plan the next product viability study | [Product Viability v1](./plans/buildopt-product-viability-v1.md) | [Detailed tracker](./plans/buildopt-product-viability-v1-tracker.md), [historical replay contract](./plans/buildopt-product-viability-v1-replay.md), [evidence and investment decisions](./plans/buildopt-product-viability-v1-evidence.md) |
| Review the closed complete native-correction study | [Complete Native Correction POC](./plans/complete-native-correction-poc.md) | [Execution tracker](./plans/complete-native-correction-poc-tracker.md), [terminal evidence](../benchmarks/results/complete-native-correction-v1/cnc004-native-admission/README.md); six native captures complete, admission rejected |
| Review the previous cross-machine handoff | [Historical handoff, 2026-09-07](./findings/buildopt-cross-machine-handoff-2026-09-07.md) | Retained baseline and evidence; resume new work from the current viability tracker |
| Review the closed source-bound correction experiment | [Source-Bound Configuration-Input Corrections POC](./plans/source-bound-configuration-input-corrections-poc.md) | [Contract](../specs/poc-source-bound-configuration-input-corrections-v1.md), [evidence index](../benchmarks/results/source-bound-configuration-input-corrections-v1/README.md), [generalization audit](./findings/buildopt-generalization-audit.md) |
| Review the stopped adaptive-fragment hypothesis | [Adaptive Fragment Generalization POC Tracker](./plans/adaptive-fragment-generalization-tracker.md) | [Terminal decision](../specs/poc-adaptive-fragment-terminal-decision-v1.md), [historical generalization audit](./findings/buildopt-generalization-audit.md) |
| Review the implemented onboarding foundation | [One-command POC onboarding roadmap](./plans/one-command-onboarding-roadmap.md) | [Product onboarding](./getting-started/product-onboarding.md), [generalization audit](./findings/buildopt-generalization-audit.md) |
| Inspect retained, deferred shared-state infrastructure | [Centralized cache and state POC roadmap](./plans/centralized-cache-and-state-roadmap.md) | [Storage contract](../specs/poc-central-storage-contract-v1.md), [architecture overview](./architecture/overview.md) |
| Inspect exact behavior | [Specifications index](../specs/README.md) | [Contracts index](../contracts/README.md), [ADRs](../adr/README.md) |

## Documentation map

### Getting started

- [Product onboarding](./getting-started/product-onboarding.md): package installation, first Gradle build, CI setup, component ownership, and rollout order.
- [Quickstart](./getting-started/quickstart.md): maintainer host checks, reproducible
  bootstrap, synthetic POC lab, first result, bypass, and cleanup.
- [Developer onboarding](./getting-started/developer-onboarding.md): local
  setup, language stacks, change workflow, generated artifacts, and review
  expectations.

### Architecture

- [Architecture overview](./architecture/overview.md): execution sequence,
  control/data planes, persistence, security boundaries, and deployment
  profiles.
- [Repository map](./architecture/repository-map.md): binaries, packages,
  contracts, tests, and the folder in which each architectural concern lives.

### User and operator guides

- [Product workflows](./guides/product-workflows.md): launcher, build history,
  Task Intelligence, Patch Autopilot, Build Impact, and
  Edge Cache.
- [CI integration](./guides/ci-integration.md): immutable installation and
  execution on GitHub Actions and GitLab CI.
- [Operations](./guides/operations.md): deployment choices, service lifecycle,
  health, recovery, upgrades, and removal.

### Findings and recommendations

These are retained historical findings. Current investment decisions live in
the [research status register](./research-status.md) and viability plan.

- [Installed Elasticsearch experiment, 2026-09-08](./findings/buildopt-elasticsearch-installed-experiment-2026-09-08.md):
  verified C12/M24 and 52 owner methods; terminal persistent-delivery failure
  on this host at the unchanged 100-ms deadline; no installed saving claim.
- [Elasticsearch correctness output contract, 2026-09-07](./findings/buildopt-elasticsearch-correctness-output-contract-2026-09-07.md):
  successful annotated build, rejected compiler/report metadata differences,
  exhaustive diagnosis and the approved, separately qualified comparison rules.
- [Cross-machine handoff, 2026-09-07](./findings/buildopt-cross-machine-handoff-2026-09-07.md):
  complete transfer briefing, closed CNC evidence, platform/state boundaries
  and preparation of the proposed installed Elasticsearch trial.
- [BuildOpt POC one-pager](./findings/buildopt-poc-handoff.md): concise project
  idea, mechanism portfolio, historical wall-time evidence, the latest
  five-family cause analysis, and the latest closed customer-general POC route.
- [Build Optimization performance findings](./findings/build-optimization-performance.md):
  measured contribution by component, current activation decisions, evidence
  boundaries, and the recommended experimental roadmap.
- [BuildOpt generalization audit](./findings/buildopt-generalization-audit.md):
  the historical target wins, terminal chronological lifetime failure, retained
  repository-independent mechanisms and the boundary the next hypothesis must
  cross.

### Plans

#### Current research

- [Product Viability v1](./plans/buildopt-product-viability-v1.md)
  and [detailed tracker](./plans/buildopt-product-viability-v1-tracker.md):
  proposed native incremental corrections and adaptive management, retired research
  routes, a [100-transition replay contract](./plans/buildopt-product-viability-v1-replay.md),
  a required native/fixed/adaptive comparison, and separate technical and
  paid-customer gates backed by an
  [evidence ledger](./plans/buildopt-product-viability-v1-evidence.md).
  [BV-001 prerequisite evidence](../benchmarks/results/buildopt-product-viability-v1/bv001/inputs.md)
  is verified. The [20-transition native audit](../benchmarks/results/buildopt-product-viability-v1/bv002/opportunity.md)
  rejects the ForbiddenPatterns seed at G1; the
  [technical decision](../benchmarks/results/buildopt-product-viability-v1/viability-decision.md)
  preserves the original negative result and its unrun downstream phases.
  The later [Checkstyle admission](../benchmarks/results/buildopt-product-viability-v1/checkstyle-admission/README.md)
  permits a bounded content-aware prototype, with no measured candidate saving.
  The [prototype correctness proof](../benchmarks/results/buildopt-product-viability-v1/checkstyle-prototype/README.md)
  now verifies BV-003/BV-004 for the frozen owner. BV-005 replay qualification
  is next; candidate lifecycle saving and adaptive/customer gates remain unproved.
  All six other repository histories are verified; their value remains unmeasured.

#### Closed research and historical substudies

The [complete plan inventory](./research-status.md#complete-plan-inventory)
also classifies the smaller substudies not listed below. Their original
outcomes remain evidence; none is a current work queue.

- [Installed Elasticsearch Native Correction v1](./plans/installed-elasticsearch-native-correction-v1.md)
  and [preparation tracker](./plans/installed-elasticsearch-native-correction-v1-tracker.md):
  frozen base and five descendants, verified local recipe/installed integration,
  three-arm protocol qualification and independently verified native admission,
  C12/M24 and composite owner tests. The final installed delivery prerequisite
  fails on persistent storage; V/L/H/O remain unrun. Historical native and
  correctness rejections are retained unchanged.
- [Complete Native Correction POC](./plans/complete-native-correction-poc.md)
  and [execution tracker](./plans/complete-native-correction-poc-tracker.md):
  directed GraphQL Java blocker closure, followed by separately gated installed
  value, chronological persistence, and unseen replication. CNC-001 maps the
  five blockers. CNC-002's [static contract](../specs/poc-complete-native-correction-v1.md)
  and checker freeze the accepted local scope and 60-start/two-hour ceiling,
  with review at 30 minutes. CNC-003's [capture runbook](./reference/complete-native-correction-capture.md)
  and generic/host fixtures qualify the native harness, including detached
  ownership and private input/output reconstruction. CNC-004's explicitly
  approved fourth window completed [six native captures and rejected admission](../benchmarks/results/complete-native-correction-v1/cnc004-native-admission/README.md).
  Both reports bind five blockers; warmed configuration is 362 ms and two
  native JAR manifests change. All four campaigns/costs remain retained.
  No recipe, candidate or value pair followed the failed gate.
- [Source-Bound Configuration-Input Corrections POC](./plans/source-bound-configuration-input-corrections-poc.md):
  the closed source-enriched three-family mechanism study; SBIC-002 reconstructs
  3/3 conclusive but only 1/3 diagnostic-bound families and stops before
  materiality, candidates, or timing.
- [Configuration-Input Native Corrections POC](./plans/configuration-input-native-corrections-poc.md):
  the closed fresh search for repository-owned Configuration Cache input
  corrections; CINC-003 stops at 7/10 conclusive and 0/3 eligible families
  before candidates or timing.
- [Wrapper-Coordinated Native Corrections POC](./plans/wrapper-coordinated-native-corrections-poc.md):
  the completed wrapper-as-onboarding experiment; functional coordination
  passes, but prospective breadth stops at 10/10 conclusive and 1/3 actionable
  material families before candidates or paired value.

- [Critical-Path Build-Logic Correction v1](./plans/critical-path-build-logic-correction-v1.md):
  five exact source trees and retained native traces, terminal 5/5 conclusive
  and 0/5 proposal-family stop before a public patch, Gradle build, or timing.
- [Critical-Path-First Reviewed Native Patch v1](./plans/critical-path-first-reviewed-native-patch-v1.md):
  ten frozen owner-workflow diagnostics, terminal 4/10 conclusive and 0/10
  proposal-family stop before source inspection, candidates, or timing.
- [Prospective Reviewed Native Patch Controlled Trial v1](./plans/prospective-reviewed-native-patch-controlled-trial-v1.md):
  frozen ten-family replication, source/workflow economics and the terminal
  80-ms off-critical-path stop before any candidate patch or timing.
- [Economic Opportunity First POC Tracker](./plans/economic-opportunity-first-poc-tracker.md):
  the closed route whose source ledger stopped at 1/5 recurrence families;
  later value blocks were not authorized and the terminal recommends a separate
  equal-opportunity cache-locality experiment.
- [Normalization-Aware Cacheability POC Tracker](./plans/normalization-aware-cacheability-poc-tracker.md):
  the closed seven-block route that separates already-normalized marker-only
  actions from reviewed relative-path normalization, then repeats fresh breadth,
  correctness and value gates before making any speedup claim.
- [Durable Native Optimization POC Tracker](./plans/durable-native-optimization-poc-tracker.md):
  the closed seven-block route from generic source opportunities through a
  marker-only correctness stop; paired value, chronological value and proposal
  UX were not authorized.
- [Verified Request Hit POC Tracker](./plans/verified-request-hit-poc-tracker.md):
  the closed seven-block route from structural opportunity through a complete
  safety contract, shadow replay, Gradle-free execution, installed value,
  chronological combined value and a terminal decision.
- [Observed Recurrent Request Portfolio POC Tracker](./plans/observed-request-portfolio-poc-tracker.md):
  a closed route, terminal cause baseline, exact evidence-precision
  work and ordered proof over commands actually observed through the wrapper.
- [Request-aligned Recurrent Learning POC Tracker](./plans/request-aligned-learning-poc-tracker.md):
  the closed predecessor, implemented exact ordinary-request identity/current
  producer-output discovery, relevance classifier and 110-transition fresh
  public capture. Independent reconstruction confirms only 2/5 complete/action
  families; its terminal scorecard authorizes no timing, speedup or successor.
- [Change-aware Producer Closure POC Tracker](./plans/change-aware-producer-closure-poc-tracker.md):
  the closed successor hypothesis, completed 25-transition producer capture,
  independently failed 1/5-versus-3/5 breadth gate, unauthorized timing blocks
  and terminal stop decision.
- [Fresh Generic Optimization POC Tracker](./plans/fresh-generic-optimization-poc-tracker.md):
  the closed zero-history predecessor, complete producer gate, fresh public
  capture and terminal 1/5 action-breadth decision.
- [Sticky Wrapper Learning POC Tracker](./plans/sticky-wrapper-learning-poc-tracker.md):
  the superseded diagnostic route and retained wrapper/lifecycle history.
- [Adaptive Fragment Generalization POC Tracker](./plans/adaptive-fragment-generalization-tracker.md):
  the completed post-`STOP_GENERIC_POC` hypothesis, terminal
  `STOP_ADAPTIVE_FRAGMENT_POC` scorecard, ordered AF-001..AF-015 work, evidence
  outcomes and mandatory documentation updates.

#### Retained foundations and deferred infrastructure

- [One-command POC onboarding roadmap](./plans/one-command-onboarding-roadmap.md):
  the `buildopt optimize build` north star, automatic state machine, ordered
  implementation blocks, end-to-end value gates and explicit POC boundaries.
- [One-command POC onboarding contract](../specs/poc-magic-onboarding-contract-v1.md):
  the executable CLI, private state/result, exact resume, bounded budget,
  exit behavior and non-production authority used by that roadmap.
- [One-input CI onboarding contract](../specs/poc-magic-ci-onboarding-v1.md):
  GitHub/GitLab command input, provider-bound portable exact state, review
  artifacts and service-free native fallback.
- [Centralized Gradle cache and BuildOpt state POC roadmap](./plans/centralized-cache-and-state-roadmap.md):
  deferred expansion of an optional HTTPS service for native Gradle cache
  objects and separately governed BuildOpt profiles, evidence and checkpoints
  across build machines.
- [Optional central storage contract](../specs/poc-central-storage-contract-v1.md):
  executable namespaces, immutable publication, exact-generation CAS,
  retention and native fallback before any remote state service exists.
- [Restart-safe typed central state](../specs/poc-central-state-storage-v1.md):
  local CAS/SQLite persistence, exact replay, corruption rejection and
  independent state retention before HTTPS or client synchronization.
- [Central HTTPS and scoped access](../specs/poc-central-https-auth-v1.md):
  TLS 1.3 listener, owner-issued capability tokens, live revocation and exact
  cache/state namespace enforcement before client forwarding is enabled.
- [Central state synchronization](../specs/poc-central-state-sync-v1.md):
  one-time repository connection, exact generated-state publication,
  optimistic concurrency, interrupted retry and verified offline snapshots.
- [Automatic central profile reuse](../specs/poc-central-optimize-integration-v1.md):
  pre/post optimize synchronization, source-commit revalidation and native
  fallback before Gradle on structural or service drift.

### Reference

- [CLI reference](./reference/cli.md): installed binaries, commands, options,
  exit codes, and audiences.
- [Configuration reference](./reference/configuration.md): environment groups,
  files, secret boundaries, defaults, and failure behavior.
- [Validation reference](./reference/validation.md): targeted checks grouped by
  subsystem and the complete validation lanes.
- [Troubleshooting](./troubleshooting.md): symptom-first recovery.
- [Glossary](./glossary.md): terms used throughout the code and contracts.

## Normative and explanatory documents

BuildOpt deliberately keeps these roles separate:

| Document type | Role | May define executable authority? |
|---|---|---:|
| RFC | Product intent, invariants, accepted/deferred decisions | Only at the decision level |
| Contract | Normative wire, schema, and state representation | Yes |
| Specification | Cross-component executable behavior and gate | Yes |
| ADR | A durable architectural choice and its consequences | Yes, within its scope |
| Tracker | Status, dependencies, and evidence | No new behavior |
| Guide or README | Explanation and operating procedure | No |

When a guide and a contract differ, the contract wins. Update the guide in the
same change that corrects or intentionally revises the implementation.

## Maintaining this documentation

Repository text is written in English. Keep examples copyable, identify the
platform and working directory, state expected output and cleanup, and link to
the executable check that proves a claim. Avoid duplicating long normative
field definitions; link to their schema or specification instead.

Run the documentation gate after changing Markdown, package documentation,
scripts referenced by guides, or repository structure:

```bash
./dev/check-documentation
```

The gate verifies required entry points, local Markdown links, referenced
repository commands, package documentation, English-language Markdown and JSON,
and navigation back to this portal.
