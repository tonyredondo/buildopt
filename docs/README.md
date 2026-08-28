# BuildOpt documentation

This portal organizes the repository by the task a reader is trying to
complete. You do not need to read the master RFC or the implementation tracker
before running the product.

## Choose a path

| You want to... | Start here | Continue with |
|---|---|---|
| Install and get a first result | [Product onboarding](./getting-started/product-onboarding.md) | [Product workflows](./guides/product-workflows.md) |
| Develop or review a change | [Developer onboarding](./getting-started/developer-onboarding.md) | [Repository map](./architecture/repository-map.md), [validation](./reference/validation.md) |
| Understand the system | [Architecture overview](./architecture/overview.md) | [Glossary](./glossary.md), [master RFC](../gradle-build-optimization-platform.md) |
| Add BuildOpt to CI | [CI integration](./guides/ci-integration.md) | [Configuration reference](./reference/configuration.md) |
| Operate self-hosted or Edge | [Operations guide](./guides/operations.md) | [Runbooks](../runbooks/README.md) |
| Diagnose a problem | [Troubleshooting](./troubleshooting.md) | [CLI reference](./reference/cli.md) |
| Review the POC idea, mechanisms, current value, and next steps | [Current POC one-pager](./findings/buildopt-poc-handoff.md) | [Detailed performance findings](./findings/build-optimization-performance.md), [benchmark evidence](../benchmarks/README.md) |
| Follow the active generic experiment | [Request-aligned Recurrent Learning POC Tracker](./plans/request-aligned-learning-poc-tracker.md) | [Active contract](../specs/poc-request-aligned-learning-v1.md), [classifier evidence](../benchmarks/results/request-aligned-classifier-fixtures-v1.json), [closed change-aware route](./plans/change-aware-producer-closure-poc-tracker.md) |
| Review the stopped adaptive hypothesis | [Adaptive Fragment Generalization POC Tracker](./plans/adaptive-fragment-generalization-tracker.md) | [Terminal decision](../specs/poc-adaptive-fragment-terminal-decision-v1.md), [current generalization audit](./findings/buildopt-generalization-audit.md) |
| Review the implemented onboarding foundation | [One-command POC onboarding roadmap](./plans/one-command-onboarding-roadmap.md) | [Product onboarding](./getting-started/product-onboarding.md), [generalization audit](./findings/buildopt-generalization-audit.md) |
| Plan optional shared state across machines | [Centralized cache and state POC roadmap](./plans/centralized-cache-and-state-roadmap.md) | [Storage contract](../specs/poc-central-storage-contract-v1.md), [architecture overview](./architecture/overview.md) |
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

- [BuildOpt POC one-pager](./findings/buildopt-poc-handoff.md): concise project
  idea, mechanism portfolio, historical wall-time evidence, the latest
  five-family cause analysis, and the active customer-general POC route.
- [Build Optimization performance findings](./findings/build-optimization-performance.md):
  measured contribution by component, current activation decisions, evidence
  boundaries, and the recommended experimental roadmap.
- [BuildOpt generalization audit](./findings/buildopt-generalization-audit.md):
  the historical target wins, terminal chronological lifetime failure, retained
  repository-independent mechanisms and the boundary the next hypothesis must
  cross.

### Plans

- [Request-aligned Recurrent Learning POC Tracker](./plans/request-aligned-learning-poc-tracker.md):
  the active successor, implemented exact ordinary-request identity/current
  producer-output discovery and relevance classifier, next five-family fresh
  capture, frozen breadth gate and ordered evidence-before-timing route.
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
  an optional HTTPS service for native Gradle cache objects and separately
  governed BuildOpt profiles, evidence and checkpoints across build machines.
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
