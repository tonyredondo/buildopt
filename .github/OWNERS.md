# Ownership and review boundaries

This file implements `F0-003`. It assigns repository accountability without
inventing teams that do not exist. During Phase 0, `@tonyredondo` is the single
verified GitHub owner for every workstream. The workstream boundary still
determines which contract, tests, and review lenses a change needs.

[`CODEOWNERS`](./CODEOWNERS) is review routing, not authorization. It does not
replace tracker gates, product-owner approval, branch protection, security
review, or the evidence required to move an item to `DONE`.

## Workstream map

| Workstream | Primary paths | Accountable repository owner | Boundary that must be reviewed |
|---|---|---|---|
| Contracts | `contracts/`, `specs/`, `adr/` | `@tonyredondo` | Producer and every affected consumer; RFC invariant changes are decided in the RFC first |
| Documentation | `README.md`, `docs/`, component READMEs | `@tonyredondo` | Audience path, copyable commands, architecture-to-code correspondence, links, and normative-source accuracy |
| Go core | `cmd/`, `internal/` | `@tonyredondo` | CLI/server behavior, local protocol consumers, process lifecycle, and exit compatibility |
| Gradle | `jvm/gradle-plugin/`, Gradle fixtures | `@tonyredondo` | Public Gradle API use, supported-version capability, Configuration Cache, and baseline parity |
| JVM Agent | `jvm/jvm-agent/` | `@tonyredondo` | Opt-in boundary, coverage, overhead, JVM compatibility, and failure degradation |
| Hermetic helper | `rust/hermetic-helper/` | `@tonyredondo` | Linux-only scope, producer identity, enforcement coverage, and fail-closed fallback |
| CI/orchestration | `.github/`, `action.yml` | `@tonyredondo` | Permissions, fork/secret trust, immutable pins, budgets, cancellation, and recovery |
| Cache/storage | Future cache packages under `internal/` and their contracts | `@tonyredondo` | Namespace isolation, pending/commit atomicity, corruption, eviction, and recovery |
| Experiments | `contracts/metrics/`, `benchmarks/`, future experiment packages | `@tonyredondo` | Metric version, cohort isolation, propensity, causal claims, guardrails, and drift |
| Patch Engine | `jvm/patcher/`, patch specifications and vectors | `@tonyredondo` | Exact preimage, idempotency, validation, draft-only delivery, and recovery |
| Test Optimization | Test grant/result contracts and integration fixtures | `@tonyredondo` | BuildOpt may integrate an explicit grant but cannot approve test selection, execution, or policy |
| Operations | `dev/`, `runbooks/`, release and pilot evidence | `@tonyredondo` | Reproducibility, credentials, supply chain, bypass, rollback, uninstall, and retained evidence |

## Review rules

1. A change cites its tracker, contract, gate, or decision IDs and names every
   workstream crossed. The accountable owner remains one person even when
   several review lenses apply.
2. A normative contract change is reviewed with its producer and all affected
   consumers. An RFC safety or scope invariant changes in the RFC before code
   or a contract implements the new behavior.
3. Workflow and release changes review least privilege, untrusted forks,
   immutable references, secret exposure, cancellation, and rollback in
   addition to the affected language stack.
4. Cache, experiment, patch, hermetic, and agent changes cannot infer authority
   from passing tests alone. Their owning tracker gates and fail-safe behavior
   remain mandatory.
5. Test Optimization retains approval of test selection, sharding, execution,
   retries, and policy. The repository owner can review BuildOpt integration
   mechanics but cannot substitute for that external product boundary.
6. Generated output is reviewed together with its normative source and
   reproducible generation command. Drift enforcement remains owned by
   `F0-005`.
7. Independent approval is not claimed while only one repository principal is
   configured. Any gate that requires independence stays open until a distinct
   qualified reviewer or automated control is evidenced.

When additional maintainers or teams exist, update the owner column and the
matching `CODEOWNERS` principals in the same change. Do not merge workstreams
or broaden product authority merely to fit the available GitHub accounts.
