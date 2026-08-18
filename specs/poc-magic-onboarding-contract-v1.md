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

The command is deliberately fail-closed. Its first qualifying invocation
executes optimized native Gradle, enables only Gradle's native Build Cache
unless the caller disables it, then performs bounded structural discovery and
calibration. A qualified candidate is stored in the structural portfolio and
returns:

```text
LEARNING / QUALIFIED_PROFILE_STORED
```

An exact later invocation validates the checkpoint, portfolio, evidence,
profile, Wrapper, executable, revision, workflow, options, graph and outputs
before Gradle starts. It selects the qualified smaller graph without an extra
flag and returns:

```text
QUALIFIED_AND_USED / QUALIFIED_PROFILE_SELECTED
```

When the repository has a private central connection, the same command also
performs automatic pre/post state synchronization. Exact local replay remains
first; a remote profile may cross source commits only after local ancestry,
build-logic, graph ownership, family, tool, output, precondition and evidence
revalidation. `--connection-dir` overrides the private
`.buildopt/central/v1` default. Service or binding drift retains native Gradle.

The command derives its exact Git change, workflow, Gradle-owned outputs and
typed graph without hand-authored BuildOpt files, calibrates through eight
balanced pairs and stores only candidates that clear correctness, value and
payback gates. Unsupported workflows, global changes, dirty/ambiguous
repository state and incomplete ownership or relationships retain native
Gradle with an exact reason. Selection authority exists only inside this
owner-invoked POC command and never becomes production authority.

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
the BuildOpt executable, repository scope, Wrapper properties,
complete Gradle argument vector, derived repository/base/target/change context
and calibration budget. A selected replay reports
`AUTOMATIC_REPLAY_COMPLETE`. A ready context reports `DISCOVERY_COMPLETE`; an
ambiguous one remains `CONTRACT_ONLY`.

`--resume auto` accepts only an exact binding match. Invocation, Wrapper,
executable, repository or budget drift creates a new generation and runs
native. A malformed checkpoint is hashed for diagnosis, rejected and replaced
without candidate reuse. Local runs bind the canonical checkout path; GitHub
and GitLab bind provider repository identity so an exact checkpoint can move
between clean runners while every other binding remains unchanged. `--resume
never` always starts a new generation.

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
machine contract. Discovery and calibration share the bounded wall-time
envelope; replay performs neither when exact evidence is accepted.

Human mode preserves Gradle stdout and prints a four-line decision summary to
stderr. `--json` reserves stdout for one `buildopt.poc/optimize-result/v1`
document and forwards Gradle console output to stderr. The same JSON is always
written to the result file. It distinguishes `OPTIMIZED_NATIVE` from
`SELECTIVE_PROFILE`, records whether the full graph started, and includes the
pre-Gradle selection duration in nanoseconds.

The exact machine contract is
[`poc-magic-onboarding-contract-v1.json`](./poc-magic-onboarding-contract-v1.json).
