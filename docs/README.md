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
| Review measured value and next priorities | [Performance findings](./findings/build-optimization-performance.md) | [Benchmark evidence](../benchmarks/README.md) |
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

- [BuildOpt POC handoff](./findings/buildopt-poc-handoff.md): concise product
  idea, component map, Gradle differentiation, synthetic and public-repository
  results, current decisions, and next work.
- [Build Optimization performance findings](./findings/build-optimization-performance.md):
  measured contribution by component, current activation decisions, evidence
  boundaries, and the recommended experimental roadmap.

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
