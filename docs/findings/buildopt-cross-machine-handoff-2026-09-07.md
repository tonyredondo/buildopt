# BuildOpt cross-machine handoff

> **Research disposition: HISTORICAL evidence (2026-09-08).** Original
> findings, limits and outcomes are preserved below. Old recommendations and
> successor instructions are scoped to that campaign. Use the
> [research status register](../research-status.md) and
> [current tracker](../plans/buildopt-product-viability-v1-tracker.md) for new work.

Snapshot date: 2026-09-07. Conversation language: Spanish. Repository language:
English. This file preserves the complete cross-machine briefing that was
truncated in chat. It is a dated handoff, not a new experiment contract.

## 1. Receiving task and immediate boundary

Continue BuildOpt work on another, more capable computer after recovering the
repository and its instructions. The destination OS, architecture, tools and
checkout are not yet verified.

On receipt:

1. Load this handoff and identify the actual destination checkout.
2. Read applicable instructions and inspect repository identity, branch, SHA,
   upstream and pending changes before editing.
3. Do not start public Gradle builds, patch a public subject, or reopen a rejected
   experiment merely because this handoff was received.
4. When the owner asks to continue, begin with the successor preparation in
   section 10. Do not repeatedly request permission for routine work already
   included in the agreed scope.

There is no active experiment process to resume. The latest experiment, CNC,
is closed with a negative admission decision. The recommended Elasticsearch
successor is distinct, not started, and has no dedicated plan/tracker yet.

This handoff does not launch another agent, create a workspace, transfer a
credential, or authorize concurrent writers to `main` on two computers.

## 2. Product idea and objective

BuildOpt is an owner-operated Build Optimization POC. The decision boundary is
measurable end-to-end wall-time improvement over already optimized native
Gradle, with the exact required outputs and behavior and zero additional
product-attributable failures.

Count observation, validation, delivery, execution and fallback costs. Benefit
must survive ordinary builds and changes, not just a favorable prepared replay.
The objective is product viability, not a growing collection of detectors,
patches or benchmarks without realized savings. General commercial viability
has not been established.

### Agreed wrapper-native pivot

The onboarding hypothesis keeps a wrapper over the customer's ordinary Gradle
command:

```text
Gradle command through buildoptw
    -> bounded typed observation
    -> opportunity detection
    -> proposed native correction
    -> isolated validation
    -> owner review and acceptance
    -> continued native Gradle through the wrapper
```

An accepted patch must also work without BuildOpt. Gradle remains authoritative
for task selection, execution, outputs, exit codes and signals. Successful
native execution must not depend on backend availability.

The existing cache infrastructure may store both Gradle cache objects and
logical backend records: observations, proposals, validation results, review
states and runner coordination. Sharing infrastructure does not merge their
namespaces or authority. Cache objects are not permission to apply a patch.
Validators require explicit actor permissions, single-active leases and
publication bound to the corresponding validation.

## 3. Published repository state and CI

Repository: [tonyredondo/buildopt](https://github.com/tonyredondo/buildopt).
Fetch/push URL: `git@github.com:tonyredondo/buildopt.git`. Target branch: `main`.

The experiment-closure baseline verified before writing this file was:

```text
HEAD == origin/main == live remote main
20292c73ed89c0fb8f7f41f91ceeb4ea1ae81b9e
```

That checkout was clean: no staged, unstaged or untracked work remained. Code,
evidence, plans, trackers, indexes and findings were published. Relevant commits:

- `55bf411f2ec6ac7eaa6d3eb7158f2b16fb0c0876`:
  `feat(cnc): retain and verify rejected native admission`.
- `20292c73ed89c0fb8f7f41f91ceeb4ea1ae81b9e`:
  `docs(cnc): close terminal synthesis with hosted CI proof`.
- The earlier actual capture executable/package used execution commit
  `cd164da42266d7b9de5f923d770fb5d5d74f0c95`.

Do not confuse execution, evidence publication and documentation commits.

### Closure-baseline CI proof

Both workflows passed on the actual `20292c73` checkout, attempt 1:

- [Base CI, run 34090407039](https://github.com/tonyredondo/buildopt/actions/runs/34090407039).
- [Native Platform CI, run 34090407027](https://github.com/tonyredondo/buildopt/actions/runs/34090407027).

All five checks succeeded: Go and Java 17, Rust 1.93 optional helper, macOS 15
ARM64, Windows Server 2025 AMD64, and Test Optimization boundary. Each job's
checkout log was inspected. Base CI logs also confirm the CNC static contract,
six-row terminal reconstruction, five forged-summary refusals and the synthetic
owner POC lab. A refresh more than 60 seconds after all-green reconfirmed the
checks and exact clean local/remote identity.

The handoff preparation rechecked the same remote SHA and five green checks.
There was no open PR with `main` as its head, commit feedback or additional
status context outstanding for that baseline.

### This file's publication is a later commit

This Markdown file and its navigation links are published after `20292c73`.
The baseline above is historical evidence, not a requirement that destination
HEAD remain at the parent of this handoff. The sending agent's final reply
provides the handoff delivery commit and its actual CI outcome. Do not treat
the older run links above as proof of a later commit's CI.

After receiving the file, identify its delivery revision and inspect the actual
destination HEAD and remote. No rollback to the baseline is authorized.

```bash
git log -1 --format=%H -- docs/findings/buildopt-cross-machine-handoff-2026-09-07.md
```

## 4. Source-computer identity and safe relocation

The verified source checkout was `/tmp/buildopt`, with common Git directory
`/tmp/buildopt/.git`, branch `main`, remote `tonyredondo/buildopt`.
These are source-computer recovery coordinates, not a required destination path.

On the new computer:

- Identify the real checkout; the agent's starting directory may be unrelated.
- Read applicable `AGENTS.md` and shared procedures before their matching work.
- Verify root, common Git directory, branch, SHA, remote and pending changes.
- Preserve existing work. Never reset or clean destructively to match this note.
- If a checkout is missing, resolve its location and preparation authority first.
- Do not copy a worktree whose `.git` file points to the old computer.
- If remote `main` has advanced, inspect the changes; never force it backward.
- Avoid concurrent edits/publication from the source and destination computers.
- Do not transfer credentials, tokens, environment files or personal directories.
  Use the destination's authorized authentication mechanisms.

Prior chat memory, terminal IDs, tool stores and workspace IDs are not a
portable state store. Git and this document are the durable recovery sources.

## 5. Implemented behavior versus unproved value

The wrapper/backend pivot has working components, not just a diagram:

- Ordinary Gradle passthrough through `buildoptw`.
- Bounded typed observation, local outbox and later publication.
- Observation-based detection, isolated validation and owner review.
- Actor-refined backend authority and lease-bound coordination.
- PatchBundle delivery, rejection and exact reversibility checks.
- Native-result preservation during backend outages.

Functional integration does not establish installed savings, chronological
persistence or commercial economics.

Selected historical native corrections passed bounded correctness/value gates:

| Selected case | Historical measured saving |
| --- | ---: |
| Micronaut `PythonVfsBytecodeCompile` | 6,921.125 ms / 63.44% |
| Spring `ArchitectureCheck` | 985.5 ms / 35.34% |
| Elasticsearch `ForbiddenPatternsTask` | 7,171.75 ms / 15.54% |

Do not pool these as one customer's workflow or claim arbitrary-build savings.
Still unproved: installed net value, persistence through ordinary changes,
actual reuse/fallback frequency, full costs, independent-family generalization,
and willingness to pay. Proposal acceptance is not commercial adoption.

## 6. Experiment history and lessons

Read the linked reports for authoritative detail; this history is a guide.

### DNO and normalization-aware cacheability

DNO v1 tried marker-only `@CacheableTask` patches. Micronaut failed Gradle
validation because an `@InputDirectory` lacked primary normalization. DNO
stopped before value timing. Preserve that failure; do not weaken or rewrite it.

NAC v2 separated a marker on already-portably-normalized inputs from a reviewed
relative-path semantic correction with its own relocation/mutation proof.
Never infer `NAME_ONLY`, `NONE`, classpath or compile-classpath semantics.
`ABSOLUTE` is explicit but non-portable. `NormalizeLineEndings` and
`IgnoreEmptyDirectories` are supplementary, not primary. Repository and task
names are labels, not classifier rules.

### Reviewed native patches and economics

Micronaut and Spring provided selected positive cases, isolated delivery,
signed verification, rejection and exact revert. The owner accepted proposals
in controlled trials. Hibernate later passed correctness but failed value at
-190.875 ms mean signed saving; it did not add a third value family.

Elasticsearch v2 added a selected positive family. Its v1 value attempt was
rejected for asymmetric stabilization; v2 consumed no invalid v1 value rows.

### Broader opportunity searches

- Prospective reviewed-native replication added no economically valid proposal.
- Critical-path-first failed required coverage/proposal breadth; material
  observed work belonged to standard tasks, not admitted repository-owned work.
- Explicit build-logic opt-outs: 5/5 complete families, 0/5 material proposal
  families. No public candidate followed.
- WCNCP: 30/30 complete ordinary observations across ten families and functioning
  integration, but only 1/3 investigated families actionable/material. Required
  prospective opportunity breadth failed before candidates and paired value.
- CINC: the remaining diagnostic cohort could not fit its fixed budget. That
  is not a negative measurement of a patch that was never executed.
- SDCR: improved capture, root-report selection and retention. Harness
  reliability is not product savings.
- SBIC: three conclusive families, only one bound the expected strict diagnostic;
  candidate and value phases did not open.

Earlier dynamic reuse/partition mechanisms had selected wins but failed to
maintain value in chronological trials. This motivated reviewed durable native
corrections. Do not infer that every durable patch shares the same limitation.

Selection lessons: a cache blocker is not proof of savings; passing tests is not
value; favorable repetition is not persistence; a known case is not prevalence.

## 7. Latest experiment: Complete Native Correction v1

Experiment: `COMPLETE_NATIVE_CORRECTION_V1` (CNC). Terminal evidence: `E-556`.
Decision: `STOP_NATIVE_ADMISSION`.

It investigated whether a complete correction of a directed workflow's
Configuration Cache blockers could preserve behavior/outputs and deliver value
over optimized native Gradle.

### Frozen subject and execution identity

Repository: `https://github.com/graphql-java/graphql-java.git`.
Revision: `f2d8c9126f898c084b176631b7346bc6fbec296a`.
Git archive SHA-256:
`83e80bbaa2b3e3308dd35e0b7120399f02149507674d958b0e2e4cc81718d051`.
Directed local workflow: `assemble`.

The subject contract fixes the exact Corretto JDKs and inputs. This runner
targets Linux amd64 and a frozen host envelope; destination compatibility must
not be assumed. The execution commit was `cd164da4`, not the later documentation
commit. Retained package SHA-256:
`07618fd835064dfa97eeea4f96d8ee3deafb32fd00b31e196fdcc20af0d1f628`.
Executable SHA-256:
`2d6e119813e45f2fc0d306fe49e56c0efedcc2c6b62020333e7542aade321a32`.

Both complete strict reports contain five unique problems with zero overflow:

1. Git worktree detection at `build.gradle:144`.
2. Git branch read at `build.gradle:169`.
3. `Gradle.buildFinished` registration at `build.gradle:186`.
4. `Gradle.buildFinished` registration at `build.gradle:938`.
5. Execution-time `Task.project` access at `BundleTaskExtension.java:407`,
   associated with `:jar`.

Binding all five does not prove a proposed lifecycle/Bnd correction safe or
valuable. Required real recipe fixtures were not executed.

### Four retained windows

| Window | Evidence | Real public starts | Outcome |
| --- | --- | ---: | --- |
| First | E-553 | 0 | Preflight rejected an implicit Corretto directory; verifier repair proved. |
| Second | E-554 | 1 | Native P01 completed, but post-check rejected generated private-home metadata. |
| Third | E-555 | 2 | P01/P02 completed; D01 refused before spawn on empty generated directories. |
| Fourth | E-556 | 6 | P01/P02, D01/D02 and M01/M02 complete; native admission rejected. |

Total: nine real public starts. Original failures and costs remain retained.
A later repair does not upgrade a failed historical row. A reservation without
a spawn is not a real start, but its failure record must not disappear.

Fourth-window process durations were P01 122.459808782 s, P02 175.870604048 s,
D01 33.605023969 s, D02 35.286764557 s, M01 107.682036551 s and M02 6.771103050 s.
The D rows have expected native strict-diagnostic exit 1, not additional
product-attributable failures. P/M rows exit successfully.

### Budget and cost

The owner approved that fourth attempt with a 7,200-second elapsed ceiling,
review at 1,800 seconds, and at most 60 starts. Preparation, validation and
publication were included; commits and CI cycles did not reset the clock.
The capture used CPUs 0-3 and four workers, with bounded disk/start lifetimes.

Historical boot-clock record:

- Start: `3532637.98`.
- Deadline: `3539837.98`.
- Final verified closure observation: `3539315.04`.
- Elapsed through closure: 6,677.06 seconds, approximately 1 hour 51 minutes.
- Sum of six native process durations: 481.675340957 seconds.

The final closure elapsed observation was retained in the sending conversation;
the published evidence also records the earlier publication checkpoint and
explicitly retains later checking/publication as additional cost. Process sums
are not total operational cost. Earlier research was not continuously timed:
unknown is not zero. Four windows capped at two hours do not imply exactly
eight hours consumed. This later handoff does not start another experiment.

These boot-clock values belong to the original computer. Never use them to
initialize or resume execution on a new host. The original stop review and
all campaign identities remain historical evidence.

### Native materiality

| Capture | Root workflow ms | Configuration interval-union ms | Share |
| --- | ---: | ---: | ---: |
| M01, fresh state | 105,425 | 12,869 | 12.206782% |
| M02, reused state | 6,091 | 362 | 5.943195% |

The conservative warmed result is 362 ms: it passes the 2% screen but fails
the unchanged 500-ms absolute floor. M01 cannot replace M02 to obtain admission.
These instrumented materiality rows do not measure attainable savings.

After 120 seconds of quiescence, the seven-sample short stability probe passed
at 1.0816372922167297. That does not certify all workload stability.

### Native output difference

Three of five JARs are byte-identical; two change:

- `build/intermediates/shadow-jar/graphql-java-0.0.0-HEAD-SNAPSHOT-shadow.jar`.
- `build/libs/graphql-java-0.0.0-HEAD-SNAPSHOT.jar`.

Their manifests shrink from 2,357 to 65 bytes, losing Bnd/OSGi headers including
`Bundle-Version`, `Bundle-SymbolicName` and `Import-Package`. The plain JAR is
unchanged. M02 reports `:jar UP-TO-DATE`, while `:shadowJar` and downstream
processing/packaging tasks execute again.

The evidence suggests an execution-dependent manifest handoff; the exact cause
is not proved. This is native cold-to-warm drift, not a candidate regression:
there was no candidate. These rows are not fresh-root C01/C02 correctness proof.

### What was not executed

No public CNC patch, complete recipe, real recipe fixture, candidate, value
pair, installed trial, replication or speedup claim followed. The owner test
suite was not executed as candidate proof. Preparations compiling `testClasses`
are not execution of that suite. Additional product failures remain zero.

## 8. What remains to finish CNC

Nothing remains to close CNC with its terminal rejected-admission outcome.
The [CNC tracker](../plans/complete-native-correction-poc-tracker.md) records:

- CNC-000..003: verified within planning/contract/harness boundaries.
- CNC-004: verified as rejected admission.
- CNC-005..007: blocked, unexecuted.
- CNC-008: verified negative Phase A closure.
- CNC-009..013: blocked, unexecuted.
- CNC-014: verified synthesis, publication and remote/CI proof.

Blocked dependent phases are not unfinished chores to execute to make the board
green: their prerequisites failed. Do not automatically continue at CNC-005 or
CNC-009, change the floor after observing 362 ms, or reopen CNC because the new
computer is faster. The product viability question remains open.

## 9. Durable evidence and non-portable temporary state

The [terminal evidence directory](../../benchmarks/results/complete-native-correction-v1/cnc004-native-admission/README.md)
is committed under `benchmarks/results/complete-native-correction-v1/cnc004-native-admission/`.
It contains `README.md`, `result.json`, `package.json`, `capture-check.json`,
`M01-materiality.json`, `M02-materiality.json`, `review.json` and
`native-evidence.tar.gz`.

The archive is 24,766,160 bytes, SHA-256:
`652bde5148757e2b495986fbe813349ae9fa50ff4815e669e2acd8cca3e7ca78`.
Its hash was rechecked during handoff preparation. It retains original clock,
ownership/environment records, dependency-seed inventory, all six attempts,
process logs/results, both HTML reports, operation traces, task graphs,
inventories and all ten exact M01/M02 JAR byte streams.

It does not contain every runtime, dependency cache or source worktree from the
old computer. Source-host recovery paths, not destination requirements:

```text
/tmp/buildopt-cnc004.ZV9DoO/state
/tmp/buildopt-cnc004-retry.Y1hoCc/state
/tmp/buildopt-cnc004-third.6y5SZP/state
/tmp/buildopt-cnc004-fourth.lPl4op/state
```

The subject common Git directory was
`/tmp/buildopt-cnc004.ZV9DoO/state/repository.git`. Each campaign retains
`worktrees/native-a`, `worktrees/native-b` and `worktrees/candidate`. All twelve
were clean at the frozen SHA at closure. They need not be copied to inspect the
published terminal evidence. Do not move clock/ownership files to another
machine and attempt to execute their occupied slots.

The source computer's campaign guardian/profile hold were stopped; `balanced`
was restored. The last closure disk observation was 24 GiB free. No active
capture or persistent CI watcher is part of this handoff.

### Portable terminal replay

`dev/check-complete-native-correction-terminal` reconstructs the refusal without
Gradle or the original host paths. It verifies package sources against the
execution commit through Git history, not against the subsequently edited
runbook. A shallow checkout that omits that history is insufficient.

Do not run the old frozen live checker against today's edited runbook and
expect old package bytes. Use terminal replay for retained evidence. It does
not recreate systemd/cgroup ownership or real runtime compatibility proof.
Passing replay is not a fresh public build.

## 10. Recommended next step: installed Elasticsearch value

The last recommendation was to test installed value and chronological
persistence using the previously value-qualified Elasticsearch
`ForbiddenPatternsTask` correction.

Status at this handoff: proposed in conversation, not started; no fresh builds,
dedicated successor plan/tracker or execution window exists. A two-hour maximum
was proposed, not silently started or authorized by the closed CNC budget.
This is not automatic permission to run CNC Phase B under a different subject.

### Why this selected case

The [Elasticsearch v2 evidence](../../benchmarks/results/economics-gated-reviewed-native-patch-v2/README.md)
records 8/8 positive pairs, mean optimized-native time 46,139 ms, candidate
38,967.25 ms, saving 7,171.75 ms / 15.54%, a paired interval of
+5,524.625..+9,833.125 ms and non-regressive p95. The owner understood and
accepted the proposal for a controlled trial, reporting 20 active review seconds.
The study's combined projected payback is 233 compatible builds.

The 20 seconds refer to human review, not an owner-required minimum speedup.
Projected payback is not observed repayment or commercial ROI. Old rows justify
case selection; they cannot serve as fresh measurements on the new computer.

### Historical source and workflow identity

- Repository: `https://github.com/elastic/elasticsearch.git`.
- Revision: `16bd5bc5355ac7c6ad736f8a6f93281b24a05ab7`.
- Bound owner workflow: `:server:precommit`.
- Selected task: `:server:forbiddenPatterns`.
- Class: `ForbiddenPatternsTask`.
- Source: `build-tools-internal/src/main/java/org/elasticsearch/gradle/internal/precommit/ForbiddenPatternsTask.java`.
- Source SHA-256: `61fe2eaa06ff463c2a49cea656b147889060855acfa094288ebb8b31567e11b6`.
- [Retained patch](../../benchmarks/results/economics-gated-reviewed-native-patch-v1/elasticsearch-forbidden-patterns.patch).

The patch adds the import and `@CacheableTask` marker. Historical admission
depended on existing input/normalization facts; do not generalize by task name.
Recover exact historical arguments from the artifacts. Measuring only the
selected task is not automatically measuring the owner workflow: freeze the
complete command that will represent product value.

### Immediate preparation block: no public Gradle

1. Audit historical Elasticsearch artifacts and the real installed wrapper path.
2. Identify the recipe/delivery implementation that exists and integration gaps.
3. Identify the base and five consecutive descendant revisions; freeze the
   chronological selection rule before outcomes are known.
4. Check source applicability, inputs and tools statically.
5. Specify three comparable arms:
   - `N0`: optimized native Gradle without correction.
   - `N1`: the same native workflow with the frozen correction.
   - `W1`: the same corrected workflow through the actual wrapper.
6. Budget preparation, tests, starts, disk, elapsed time and hosted-CI follow-up.
7. Write a separate English plan/tracker with commands, artifacts, gates,
   statuses and stop conditions. No filename has been chosen yet; do not cite
   an uncreated successor document as existing.
8. Verify that the entire proof matrix fits before any public start.

### Proposed subsequent measurements

Separate native patch value (`N0` versus `N1`), wrapper cost (`W1` versus `N1`)
and net installed value (`N0` versus `W1`). Use equivalent initial state/cache
access and a frozen order. Never compare a cold control with a prepared
candidate or subtract unrelated historical averages.

Measure ordinary changes and repeats, actual restoration/rejection/fallback,
exact outputs, zero additional failures, total costs and signed savings including
losses. Keep the same frozen recipe where valid; when it no longer applies,
retain native and charge the overhead. Do not hand-repair every descendant or
skip unfavorable revisions.

A known selected case can establish only a directed mechanism, not broad
prevalence or commercial viability. Existing CNC installed/chronology design is
a reference for proof obligations, not a waiver of its failed dependency gate.

Do not prioritize an automatic GraphQL manifest repair now. It is a real
correctness observation but does not repair the configuration materiality
failure. Investigating it requires a separate scoped task, not recipe expansion.

## 11. Reading map and implementation owners

Read current terminal records before old recommendation paragraphs. The broad
one-pager contains substantial historical material; an old "next step" is not
current execution authority.

### Product state and evidence

- [POC one-pager](./buildopt-poc-handoff.md): product idea, mechanisms and history.
- [Implementation tracker](../../implementation-tracker.md): status, blockers
  and evidence ledger; current CNC terminal evidence is E-556.
- [Generalization audit](./buildopt-generalization-audit.md): conclusion limits.
- [Performance findings](./build-optimization-performance.md): value and costs.
- [Documentation portal](../README.md) and [specification index](../../specs/README.md).

### Wrapper/backend pivot

- [Wrapper-coordinated plan](../plans/wrapper-coordinated-native-corrections-poc.md).
- [Human contract](../../specs/poc-wrapper-coordinated-native-corrections-v1.md).
- [Machine contract](../../specs/poc-wrapper-coordinated-native-corrections-v1.json).
- [Subjects](../../specs/poc-wrapper-coordinated-native-corrections-v1.subjects.json).
- Evidence root: `benchmarks/results/wrapper-coordinated-native-corrections-v1/`.

### Closed CNC study

- [Plan](../plans/complete-native-correction-poc.md).
- [Tracker](../plans/complete-native-correction-poc-tracker.md).
- [Human contract](../../specs/poc-complete-native-correction-v1.md).
- [Machine contract](../../specs/poc-complete-native-correction-v1.json).
- [Subjects](../../specs/poc-complete-native-correction-v1.subjects.json).
- [Capture runbook](../reference/complete-native-correction-capture.md).
- [Evidence index](../../benchmarks/results/complete-native-correction-v1/README.md).
- [Source feasibility](../../benchmarks/results/complete-native-correction-v1/selection/feasibility.md).
- [Terminal evidence](../../benchmarks/results/complete-native-correction-v1/cnc004-native-admission/README.md).

### Elasticsearch sources for successor preparation

- [Economics-first v1 plan](../plans/economics-gated-reviewed-native-patch-v1.md).
- [Corrected v2 plan](../plans/economics-gated-reviewed-native-patch-v2.md).
- [V1 human contract](../../specs/poc-economics-gated-reviewed-native-patch-v1.md),
  [machine contract](../../specs/poc-economics-gated-reviewed-native-patch-v1.json),
  [subjects](../../specs/poc-economics-gated-reviewed-native-patch-v1.subjects.json).
- [V2 human contract](../../specs/poc-economics-gated-reviewed-native-patch-v2.md)
  and [machine contract](../../specs/poc-economics-gated-reviewed-native-patch-v2.json).
- [Workflow binding](../../benchmarks/results/economics-gated-reviewed-native-patch-v1/binding.json).
- [Correctness](../../benchmarks/results/economics-gated-reviewed-native-patch-v1/correctness.json).
- [Patch](../../benchmarks/results/economics-gated-reviewed-native-patch-v1/elasticsearch-forbidden-patterns.patch).
- [V2 summary](../../benchmarks/results/economics-gated-reviewed-native-patch-v2/README.md),
  [raw rows](../../benchmarks/results/economics-gated-reviewed-native-patch-v2/raw.json),
  [value](../../benchmarks/results/economics-gated-reviewed-native-patch-v2/value.json),
  [review](../../benchmarks/results/economics-gated-reviewed-native-patch-v2/review.json)
  and [decision](../../benchmarks/results/economics-gated-reviewed-native-patch-v2/terminal-decision.json).

### Code owners to inspect before adding another path

| Path | Responsibility |
| --- | --- |
| `cmd/buildoptw/main.go` | Ordinary wrapper entrypoint and native passthrough. |
| `internal/wcncpobserve/` | Observation, native result, outbox and publication. |
| `internal/wcncpdetect/` | Observation-based detection. |
| `internal/wcncpvalidate/` | Isolated validation. |
| `internal/wcncpreview/` | Review. |
| `internal/sharedcache/wcncp_*.go` | Backend state, authorization, leases and HTTP. |
| `cmd/buildopt-server/wcncp_actors.go` | Actor management. |
| `internal/stickywrapper/templates/buildoptw` and `buildoptw.bat` | Launcher templates, distinct from the binary. |
| `dev/complete-native-correction-runner/` and `dev/complete-native-correction-validator/` | Versioned CNC harness. |
| `internal/strictdiagnostic/` | Strict-report selection and validation. |
| `internal/wcncpmateriality/` and `cmd/wcncp-controlled-materiality/` | Interval-union materiality analysis. |

Inspect reusable components before inventing another framework. Code reuse is
not permission to reuse old result rows as fresh evidence.

## 12. Destination platform and toolchains

The lock is [dev/toolchains.lock.yaml](../../dev/toolchains.lock.yaml), not a
root-level file. Despite its extension, its current contents use JSON syntax.
The inspected lock declares Linux amd64. The CNC runner additionally depends
on a working user systemd manager, delegated cgroup v2 and CPU affinity.

Native Platform CI success on macOS/Windows does not establish that the CNC
experimental harness runs there. Check destination OS/architecture, exact
runtime/tool versions and runner restrictions first. Missing support is a
reported gap, not permission to silently change vendor, version or architecture.

Prefer repository tools for Go:

```bash
./dev/run --toolchain go -- go version
```

If unprovisioned, `dev/run` identifies the required bootstrap. Read
`dev/bootstrap` and the lock before downloading. Do not install an arbitrary
global version merely to bypass a failure. The old Corretto subject runtime
is not interchangeable with a Temurin toolchain of a similar version.

Do not carry the old host's temporary power-setting permission to a different
machine automatically. More CPU/disk does not permit mixing host timings,
moving thresholds, skipping stabilization, increasing workers/starts/budget
without a frozen protocol, or assuming process ownership works. All compared
arms need one new documented, symmetric environment.

## 13. Reception and validation commands

First identify the correct checkout. The following inspection does not mutate
source; the remote query requires the destination's network/authentication:

```bash
pwd
git rev-parse --show-toplevel --git-common-dir HEAD origin/main
git status --short --branch
git remote -v
git ls-remote origin refs/heads/main
```

The closure baseline is `20292c73ed89c0fb8f7f41f91ceeb4ea1ae81b9e`, and this
handoff is delivered by a later commit. Resolve missing history/upstream before
editing, preserve dirty work and inspect later remote changes without reset.

After receipt, with tools prepared and validation in scope:

```bash
./dev/check-complete-native-correction-terminal --self-test
./dev/check-complete-native-correction contract
./dev/check-complete-native-correction fixtures
./dev/check-layout
./dev/check-tracker-consistency
./dev/check-base-ci --static
./dev/check-documentation
git diff --check
```

Contract/replay/generic fixtures do not authorize public Gradle. Generic fixtures
use synthetic processes/repositories. The separate `host-fixtures` gate uses
systemd/cgroups and transient units; do not automatically run it on an unknown
platform. For code changes select tests, race, vet and ShellCheck for the actual
affected owners. Reuse valid unchanged proof rather than restarting a campaign.

Closure-baseline validation passed terminal replay/five negatives, the static
CNC contract, materiality/strict-selector race suites, ShellCheck, layout,
tracker consistency with 53 accepted RFC decisions, Base CI static, and hosted
five-check CI. Final baseline documentation checked 622 Markdown files,
3,066 JSON files, 3,127 local links, 2,143 commands and 51 Go packages, including
English documentation. Those are historical counts, not immutable expectations
after legitimate documentation changes such as this handoff.

## 14. Operating agreement and authority

The owner wants engineering-partner/autopilot behavior, tangible progress and
periodic concise Spanish updates, not repeated routine permission questions.

- Keep repository code, docs, comments and configuration in English.
- Read applicable instructions before action; use `apply_patch` for edits and
  `rg`/`rg --files` for search. Make the smallest sound change.
- Preserve other work; no destructive Git cleanup/reset.
- Never specialize detector decisions by repository/task names.
- Do not reuse historical reports as new evidence or move gates to get a pass.
- Retain failures, excluded starts, unknown costs and negative outcomes.
- Separate compilation, synthetic fixtures and real consuming-path proof.
- A phase is not verified while required proof remains missing.
- Do not delegate without authorization or create persistent services for convenience.
- Do not transfer credentials or bypass permission boundaries.
- Do not promise background follow-up without a real watcher/scheduler.

The user authorized subsequent intended commits and normal pushes in this same
repository. Do not ask again for each commit/push within agreed work. That does
not authorize history rewriting, merging, upstream public-subject publication,
automatic customer patch application, production/deployment, billing/permission
changes or new experiments inferred from a failed gate. Soak, design partners,
production hardening, automatic merge and Test Optimization remain outside the
POC decision. A CI check named Test Optimization boundary does not widen scope.

Read shared procedures from `CODING_AGENT_PROCEDURES_DIR`, otherwise the
platform's agent-rules location (`$HOME/.agent-rules` on Linux/macOS):

- `implementation.md`: implementation and quality.
- `git-workflow.md`: branch/index/commit/publication operations.
- `pr-follow-up.md`: CI and same-session follow-up after publication.
- `review.md`: when a review is requested.

Do not substitute a same-named repository/network file for a missing procedure.
Before publishing, inspect all outgoing history and effective destinations.
After normal push, confirm remote SHA, actual tested checkout, complete checks
and feedback, and the procedure's final confirming interval within budget.
Report pending proof honestly. Publication permission does not waive human gates.

## 15. CI, timing and budgets

The owner explicitly rejected treating ordinary hosted CI as a stable timing
environment. Hosted CI owns contracts/correctness/integration, not product
wall-time gates. Noise is not permission to relax exact outputs or allow extra
product failures. Performance uses a separately controlled symmetric campaign;
new-computer timings cannot be compared directly with old-host rows.

The owner rejected an eight-hour proposal in favor of two hours. Do not recover
the old eight-hour proposal from stale notes as current authority. Before the
successor starts, freeze the full start allocation, preparation/downloads,
fixtures/correctness, stabilization/value, review checkpoint, disk/process
limits, validation/CI treatment and exact stop point. Verify the complete proof
matrix fits. A correction, new commit or chat continuation does not reset spend.
New host identity does not erase earlier campaign costs.

## 16. Acceptance criteria for successor preparation

The preparation block, not the public experiment, is complete only when:

- [ ] Destination checkout, environment and applicable instructions are verified.
- [ ] Historical Elasticsearch evidence and its limits are recovered.
- [ ] Actual recipe, wrapper and delivery path availability is established.
- [ ] Integration gaps are distinguished from proof gaps.
- [ ] Base revision and chronological selection rule are frozen.
- [ ] N0/N1/W1 arms, exact commands, caches and outputs are specified.
- [ ] Complete proof allocation and budget are coherent.
- [ ] A separate English plan/tracker exists with explicit not-started execution.
- [ ] Proportional checks and actual diff review pass.
- [ ] Authorized commit/push and exact remote confirmation are complete.
- [ ] Applicable hosted CI is followed to its actual outcome.

Only then may execution open under the accepted new scope/budget. These
checkboxes describe proposed future preparation, not omissions in closed CNC.

## 17. Resume summary

BuildOpt seeks realized net savings through reviewed native corrections, with
the wrapper as ordinary entrypoint and a separate backend control plane.
Selected wins and implemented integration exist; installed persistence and
general product viability remain unproved. CNC is fully closed as rejected
admission with published evidence and green closure-baseline CI, not a successful
optimization. No old process needs resuming on the new machine.

Prepare a separate installed/chronological Elasticsearch trial next; do not run
GraphQL's blocked phases. Future evidence must be fresh, symmetric and bound
to its new host/campaign. Work autonomously inside the agreed scope, without
repeated routine approvals or silently expanding the experiment.
