# Workflow input ownership POC v1

## Purpose

This contract tests whether BuildOpt can handle a mixed Git change containing
Gradle-owned source plus an unowned repository path without a repository-name or
filename-extension rule. An unowned path may be removed from Build Impact
ownership only when complete evidence from the requested Gradle workflow proves
that no attributed task consumes it.

The mechanism is proof-of-concept discovery. It does not authorize production
use, claim performance value, or change Test Optimization.

## Inputs

The observation is bound to:

- the exact normalized changed-path set derived from the selected Git revisions;
- the requested Gradle task graph;
- the repository root and target revision; and
- one ordinary workflow execution used for discovery.

The observed document uses
`buildopt.build-impact/workflow-input-relevance/v1`. Every changed path appears
exactly once with the Gradle task paths that consume it.

## Completeness and decision rules

BuildOpt may mark an unowned path workflow-irrelevant only when all of the
following hold:

1. the observed path set exactly equals the Git changed-path set;
2. the task graph can attribute every scheduled task to the requested workflow;
3. finalized task inputs can be queried after their producers have run;
4. the path exists as a regular file inside the repository and is not a symlink;
5. no attributed task input contains the path; and
6. at least one changed path remains owned by a Gradle project.

Owned paths remain relevant even when their direct consumer set is empty.
Consumed unowned paths remain relevant and therefore retain native execution
when no project owner can be derived. Missing, deleted, symlinked, incomplete,
ambiguous, all-unowned, or malformed evidence also retains native execution.

No path extension, repository identity, task name, or observed timing may change
these rules.

## Gradle lifecycle

Some Gradle inputs are provider-backed and are not final until their producer
tasks complete. The one-time discovery invocation therefore captures task inputs
after the requested tasks execute. BuildOpt disables Configuration Cache only
for this setup observation so the capture action may inspect live task objects.
Control, candidate, normal native, and selected-replay invocations keep their
normal Configuration Cache behavior; the observation duration is never a
performance sample.

The standalone Build Impact discovery path does not attach the optional capture
action and remains Configuration Cache compatible.

## Required executable proof

The fixture matrix must demonstrate:

- an owned source change plus an unconsumed root document is discoverable;
- a provider-backed generated input is observed after its producer completes;
- a consumed unowned path retains native execution;
- a deleted or missing unowned path retains native execution;
- unsupported workflows, global changes, ambiguity, incomplete evidence and
  all-unowned changes retain native execution; and
- Configuration Cache remains valid for standalone Build Impact discovery.

## Public-repository proof

The frozen OpenTelemetry Java Instrumentation JMX change is:

- base: `0ae4cc70d2c2a2bd02f71a33f90b80db7e65ef99`;
- target: `dc7e94b40f1b86a19ec3449d895e80ff3c5754a4`; and
- workflow: the four JMX `testClasses` entrypoints recorded in the evidence.

The source mirror supplied the target as a shallow commit. The harness rebuilt
only local ancestry metadata with the public target tree and the public base as
parent. The target SHA, tree, changed paths and BuildOpt decision remain bound to
the public commits; the reconstruction is evidence plumbing, not product logic.

The observation must show:

- `CHANGELOG.md` has no consuming task and may be ignored;
- the JMX YAML is consumed by `processResources`;
- the JMX Java test is consumed by `compileTestJava`;
- the module-owned `jetty.md` remains relevant even with no direct consumer;
- discovery completes with 1,027 total, 8 selected and 1,019 omitted projects;
  and
- calibration remains skipped, so this block makes no wall-time claim.

## Boundaries

- owner-operated POC evidence on the 12-CPU Linux development host;
- public frozen commits and a single structural observation;
- no repository-specific product rules or extension allowlists;
- no threshold changes, production authority, soak or design-partner dependency;
- no broadened cross-commit performance claim; and
- Test Optimization remains out of scope.

The next hypothesis is native-volatility quarantine: volatile producers must be
rebuilt locally while every output that BuildOpt transports or reuses remains
byte-exact.
