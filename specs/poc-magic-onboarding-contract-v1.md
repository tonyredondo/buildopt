# One-command POC onboarding contract

## Purpose

This contract fixes the customer-facing boundary used by automatic discovery
and the later calibration blocks. The stable entrypoint is:

```bash
buildopt optimize build
```

The command accepts ordinary Gradle arguments. BuildOpt owns all generated
state under `.buildopt/optimize/v1`; a supported repository must not require a
person to create a BuildOpt manifest, graph, changes file, output contract,
evidence document or qualified profile.

## Current executable behavior

The command is deliberately fail-closed. It executes optimized native Gradle,
enables only Gradle's native Build Cache unless the caller disables it, then
attempts bounded structural discovery and calibration. A qualified candidate
is stored in the structural portfolio and returns:

```text
LEARNING / QUALIFIED_PROFILE_STORED
```

The command derives its exact Git change, workflow, Gradle-owned outputs and
typed graph without hand-authored BuildOpt files, calibrates through eight
balanced pairs and stores only candidates that clear correctness, value and
payback gates. Unsupported workflows, global changes, dirty/ambiguous
repository state and incomplete ownership or relationships retain native
Gradle with an exact reason. A stored profile is learning state; the command
does not select or activate it yet.

`BUILDOPT_BYPASS=1` keeps the Wrapper shortcut but skips optimize state and
reporting completely. Once Gradle starts, its exit or signal status remains the
process result. A final reporting problem is visible but cannot turn a failed
Gradle build into success or replace its exit status.

## State and resume

The command writes two private, atomic documents:

- `.buildopt/optimize/v1/state.json` is the resumable state machine;
- `.buildopt/optimize/v1/result.json` is the latest customer result.

No raw Gradle arguments, console logs, credentials or absolute repository path
are persisted. The checkpoint binds opaque SHA-256 values for
the BuildOpt executable, canonical local repository scope, Wrapper properties,
complete Gradle argument vector, derived repository/base/target/change context
and calibration budget. A ready context reports `DISCOVERY_COMPLETE`; an
ambiguous one remains `CONTRACT_ONLY`. The local scope is intentionally not
portable; the later CI block must use provider identity before state may move
between runners.

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
machine contract. Discovery spends only the bounded wall-time envelope; no
calibration pair is executed yet.

Human mode preserves Gradle stdout and prints a four-line decision summary to
stderr. `--json` reserves stdout for one `buildopt.poc/optimize-result/v1`
document and forwards Gradle console output to stderr. The same JSON is always
written to the result file.

The exact machine contract is
[`poc-magic-onboarding-contract-v1.json`](./poc-magic-onboarding-contract-v1.json).
