# One-command POC onboarding contract

## Purpose

This contract fixes the customer-facing boundary before automatic discovery or
calibration is implemented. The stable entrypoint is:

```bash
buildopt optimize build
```

The command accepts ordinary Gradle arguments. BuildOpt owns all generated
state under `.buildopt/optimize/v1`; a supported repository must not require a
person to create a BuildOpt manifest, graph, changes file, output contract,
evidence document or qualified profile.

## Current executable behavior

The first implementation is deliberately fail-closed. It executes optimized
native Gradle, enables only Gradle's native Build Cache unless the caller
disables it, and returns:

```text
NATIVE_RETAINED / AUTO_DISCOVERY_PENDING
```

It does not call the existing internal discovery commands, fabricate a
candidate, calibrate, select a profile or claim a saving. This makes the final
CLI usable while the ordered tracker blocks replace the reason with real
discovery, learning and qualification decisions.

`BUILDOPT_BYPASS=1` keeps the Wrapper shortcut but skips optimize state and
reporting completely. Once Gradle starts, its exit or signal status remains the
process result. A final reporting problem is visible but cannot turn a failed
Gradle build into success or replace its exit status.

## State and resume

The command writes two private, atomic documents:

- `.buildopt/optimize/v1/state.json` is the resumable state machine;
- `.buildopt/optimize/v1/result.json` is the latest customer result.

No raw Gradle arguments, console logs, credentials or absolute repository path
are persisted. This contract-only checkpoint binds opaque SHA-256 values for
the BuildOpt executable, canonical local repository scope, Wrapper properties,
complete Gradle argument vector and calibration budget. The local scope is
intentionally not portable; the later discovery block must replace
`CONTRACT_ONLY` with a provider/repository identity before CI or central state
may reuse it.

`--resume auto` accepts only an exact binding match. Invocation, Wrapper,
executable, repository or budget drift creates a new generation and runs
native. A malformed checkpoint is hashed for diagnosis, rejected and replaced
without candidate reuse. `--resume never` always starts a new generation.

## Outcomes and authority

Every completed command has exactly one outcome:

- `QUALIFIED_AND_USED`: a matching profile passed all correctness, net wall
  time, tail, failure, fallback and payback gates;
- `LEARNING`: bounded comparable evidence is incomplete and native remains
  authoritative; or
- `NATIVE_RETAINED`: the workflow is unsupported, unsafe, drifted, ambiguous
  or lacks value, with an exact reason.

`ACTIVE` authorizes selection only inside the explicitly invoked POC command.
Every state and result keeps `productionAuthorized=false`; there is no
autonomous production promotion.

## Budget and output

The default future calibration envelope is 30 minutes, at most eight balanced
pairs and a maximum accepted break-even of 30 matching builds. Owners can
reduce or extend those bounded values through the CLI, up to the limits in the
machine contract. The current native skeleton records but does not spend that
budget.

Human mode preserves Gradle stdout and prints a four-line decision summary to
stderr. `--json` reserves stdout for one `buildopt.poc/optimize-result/v1`
document and forwards Gradle console output to stderr. The same JSON is always
written to the result file.

The exact machine contract is
[`poc-magic-onboarding-contract-v1.json`](./poc-magic-onboarding-contract-v1.json).
