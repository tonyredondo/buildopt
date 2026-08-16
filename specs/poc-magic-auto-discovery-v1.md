# Automatic discovery behind the one-command POC

## Purpose

This contract implements the second step of the one-command onboarding
roadmap. After the owner runs an ordinary command such as:

```bash
buildopt optimize jar
```

BuildOpt first preserves the optimized native Gradle result. It then derives
the repository, workflow, immutable comparison revisions, exact Git changes,
non-empty Gradle-owned outputs and complete structural graph without requiring
an owner-authored BuildOpt manifest, graph, output contract or changes file.

## Discovery boundary

The workflow comes only from the Gradle argument vector. Options that move the
project, inject another build/init script, exclude tasks or select individual
tests are rejected before discovery. The target is the current immutable HEAD.
The comparison base comes, in order, from the GitHub event, GitLab merge/push
metadata or the local branch's configured upstream merge base. Any tracked or
untracked repository change outside generated `.buildopt` and root `.gradle`
state, a missing upstream or an invalid provider revision retains native
Gradle.

Repository identity comes from provider metadata, `origin`, or a private
opaque local ID. The opaque ID is suitable only for local state; no absolute
repository path or raw Gradle argument is written to a discovery document.

The output preflight executes the owner workflow and accepts only non-empty,
repository-contained outputs with unambiguous Gradle project/task ownership.
Typed Gradle discovery must then prove complete project and task relationships.
Build-logic and root/global changes, unknown ownership and incomplete graphs
retain the original workflow.

## Result

A safe candidate reports:

```text
LEARNING / STRUCTURAL_CANDIDATE_DISCOVERED / DISCOVERED
```

and writes seven current-user-private documents under the command's state
directory: the exact changes, output contract, proposed manifest, graph,
generated binding, fallback change and structural proposal. These files are
generated evidence, not customer input and not activation authority.

The real Gradle fixture covers packaging (`jar`), custom verification,
distribution (`distZip`) and build-owned test preparation (`testClasses`). It
also proves native fallback for an unsupported custom workflow, a global build
change and an ambiguous local base. A workflow that executes tests remains
native because Test Optimization is outside this POC.

## POC authority

Discovery performs no timing, calibration or selection and creates no build
speed claim. The native execution remains authoritative and every state/result
keeps `productionAuthorized=false`. The next ordered block may calibrate only
the discovered candidate against the same optimized-native workflow and must
still reject candidates that do not produce equivalent outputs or positive net
wall-time value.

The exact machine contract is
[`poc-magic-auto-discovery-v1.json`](./poc-magic-auto-discovery-v1.json).
