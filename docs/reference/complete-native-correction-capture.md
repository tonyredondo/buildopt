# Complete Native Correction capture runbook

Status: **CNC-004 third window passed P01/P02 and refused D01 before spawn; two Gradle starts in this window.**
This document describes the implemented interface and its evidence boundary. It
is not authority to start the two-hour experiment. The [tracker](../plans/complete-native-correction-poc-tracker.md)
owns advancement; the [contract](../../specs/poc-complete-native-correction-v1.md)
owns the unchanged 60-start, 7,200-second and 1,800-second-review limits.

## What can be checked now

Run from the selected BuildOpt worktree. These commands use Go and fake children,
not Gradle, public worktrees, installed JDKs, network downloads or timing rows:

```bash
./dev/check-complete-native-correction contract
./dev/check-complete-native-correction fixtures
./dev/run --toolchain go -- go test -race -count=1 github.com/tonyredondo/buildopt/dev/complete-native-correction-runner ./internal/strictdiagnostic ./internal/contractcrypto
./dev/run --toolchain go -- go vet github.com/tonyredondo/buildopt/dev/complete-native-correction-runner github.com/tonyredondo/buildopt/dev/complete-native-correction-validator
```

The versioned owner is `dev/complete-native-correction-runner`. It reuses
`internal/strictdiagnostic.SelectRootReportV2`, not the clone-producing SDCR
shell entrypoint. Tests create only their own synthetic shared-Git repository
and registered detached worktree; `.git` is a file. Synthetic tar archives test
installed runtime byte/mode drift. They do not prove compatibility with a real
Corretto installation.

The separate local host gate requires a working user systemd manager and
delegated cgroup v2. It creates only exact test-owned transient units, fake
children and synthetic repositories; it stops/collects those units afterward:

```bash
./dev/check-complete-native-correction host-fixtures
```

It proves detached-child survival between successful requests, complete owned
subtree termination on row timeout/campaign expiry/client loss, preservation of
an unrelated process, and six P01-M02 requests through the native command
builder, source-state preparation, capture and independent checker. Ordinary
CI runs generic fixtures and explicitly skips this host-only gate; that skip
is not ownership proof. Neither suite executes the Gradle instrumentation or
proves real Corretto/Gradle/plugin compatibility. CNC-004 supplies that evidence.

The historical CNC-003 [package snapshot](../../benchmarks/results/complete-native-correction-v1/contract/capture-package.json)
binds launcher, state/capture/preparation code, tests, static checker, selector,
instrumentation, toolchain lock, contracts and the actual Go executable hash.
It identifies that earlier local implementation, not the executable rebuilt
after later commits. The first committed execution package and its zero-start
refusal are retained in the [CNC-004 preflight record](../../benchmarks/results/complete-native-correction-v1/cnc004-preflight/README.md).
Do not overwrite either historical identity. For a future package, first
resolve a new absolute file path in an existing parent as `cnc_new_package`:

```bash
./dev/run-complete-native-correction freeze --package "$cnc_new_package"
./dev/run-complete-native-correction package-check --package "$cnc_new_package"
```

`freeze --package ABSOLUTE_NEW_FILE` creates a new snapshot without overwrite.
The parent must already exist. Changes to source, compiler/build identity or
executable require a new package, never rewriting an active campaign binding.
Before any future public start, every package source must match committed
BuildOpt bytes and the final executable must be frozen again. The runbook does
not authorize committing or publishing those bytes.

## Prospective execution layout and interface

The runner currently targets Linux amd64 and the frozen host envelope. Paths
are operator-supplied absolute canonical paths; no personal path is shared
configuration. A new campaign root contains:

```text
campaign/
  state.json, lock, guard-request.json, ownership.json, guard.log
  worktrees/native-a, worktrees/native-b, worktrees/candidate
  runtimes/ and archives/
  homes/prefetch-a, homes/prefetch-b
  homes/diagnostic-a, homes/diagnostic-b, homes/materiality
  dependency-seed/inventory.json
  environment.json, review.json
  attempts/P01/... through attempts/M02/...
```

Recover the known subject repository/shared Git directory before materializing
anything. Missing or ambiguous identity stops setup; do not substitute a clone
or scan for another checkout. Worktree creation and runtime preparation remain
operator-owned. The owner explicitly authorized one exact-commit reacquisition
after all recorded GraphQL copies were found absent; it and three shared
detached worktrees now exist. Neither `preflight` nor `capture` creates subject worktrees or
downloads/installs runtimes. Exact revision, archive and runtime hashes come
from the [subject manifest](../../specs/poc-complete-native-correction-v1.subjects.json).

All modes are exposed through `./dev/run-complete-native-correction`:

| Mode | Required flags and behavior |
| --- | --- |
| `freeze` | `--package ABSOLUTE_NEW_FILE`; immutable local package snapshot, no campaign clock. |
| `package-check` | `--package FILE`; compare source/executable fingerprints, print package digest, no campaign writes. |
| `init` | `--package FILE --state NEW_ROOT --acknowledge-phase-a`; create immutable state and start the boot-bound execution clock. Not run for CNC-003. |
| `guard` | Same flags as init, with an existing state root; start one non-restartable transient guardian before native capture. Requires at least 15 seconds remaining. |
| `preflight` | Common inputs below; verify input hashes, registered detached worktree, runtime archive/install trees, disk and host. No Gradle spawn. |
| `seed` | `--package FILE --state ROOT --acknowledge-phase-a`; successful P01/P02 required; copy only dependency modules and Wrapper distributions from P02. |
| `stabilize` | Same flags as seed; require successful preparation, bind probe threads to CPUs 0-3, wait 120 seconds and retain seven samples. Not executed here. |
| `review` | `--package FILE --state ROOT --decision continue\|stop --note TEXT`; immutable elapsed-30-minute checkpoint, never a deadline reset. |
| `capture` | Common inputs, `--slot P01` (or the next frozen P/D/M slot) and `--acknowledge-phase-a`; verify committed package, active ownership, exact inputs, order and budgets, reserve once, capture one native child, retain terminal evidence. |
| `check` | `--package FILE --state ROOT`; independently reconstruct reservations, raw process/log/report/artifact bindings and terminal outcomes without campaign writes. |

Common prospective preflight inputs are `--package`, `--state`, `--source`,
`--common-git-dir`, `--gradle-home`, `--jdk25`, `--jdk25-archive`, `--jdk21`
and `--jdk21-archive`. The wrapper fixes `--repo` to its own actual worktree
and rejects overrides. `--acknowledge-phase-a` is an operator safeguard, not
permission inferred by an agent.

The implemented native argument assembly reads the exact
machine profile and appends private Gradle home and verified Java installation
paths, disables runtime auto-discovery/download, and adds `--offline` after
prefetch. It runs the source Wrapper through `taskset --cpu-list 0-3`. M rows
also use `dev/complete-native-correction.init.gradle` and operation capture. P01/P02
map to separate native worktrees/prefetch homes; D01/D02 use those worktrees
with fresh diagnostic homes; M01/M02 share the native-a materiality home.
The fake-child consuming-path matrix proves this mapping without claiming a
real public output. D/M01 archive only declared generated build/project-cache
roots under their own attempt. Nonempty trees require ignored, untracked content;
directory-only trees may lack an ignore match because they have no Git content.
Any file, even zero-byte, still requires ignore proof; links/special members
always refuse. Empty structure is preserved too. These modes never reset Git
or delete a worktree. Earlier strict reports remain at their original recorded
paths for independent replay. M02 intentionally retains M01 execution state.

Every child gets an explicit environment rather than the operator environment.
`HOME` and JVM `user.home` point at the assigned Gradle home's private
`user-home`; JVM `maven.repo.local` points at its empty `.m2/repository`.
The owned `JAVA_TOOL_OPTIONS` sets these properties and UTF-8 symmetrically;
ambient JVM/Gradle options are rejected, not appended. Private Maven settings,
artifacts and unexpected entries refuse continuation. JVM preferences under
`.java/.userPrefs` and Kotlin metadata under `.kotlin/daemon` are declared
runtime-generated state in an initially empty private home. Only regular files
and directories in those trees and their required ancestors are admitted;
links and special members refuse. They are never copied into the dependency
seed or another home/arm. M02 retains only its own M01 runtime state.
No `mavenLocal()` source
change is made. This implements the contract's no-owner-home-reuse rule.
Gradle's [Maven-local resolution API](https://docs.gradle.org/current/kotlin-dsl/gradle/org.gradle.api.artifacts.dsl/-repository-handler/maven-local.html)
documents the relevant property/settings precedence; real pinned-runtime
behavior remains a CNC-004 check.

### Future invocation sequence, not executed in CNC-003

Resolve the actual roots before assigning `cnc_package`, `cnc_state`,
`cnc_common`, `cnc_jdk25`, `cnc_archive25`, `cnc_jdk21` and `cnc_archive21`.
The state path must be new; source/runtime paths must be canonical and contained
inside it. Publication/final package freeze is a prerequisite to capture, not
authority granted by these examples. Initialize **before** execution preparation
or downloads, then establish the guardian:

```bash
./dev/run-complete-native-correction init --package "$cnc_package" \
  --state "$cnc_state" --acknowledge-phase-a
./dev/run-complete-native-correction guard --package "$cnc_package" \
  --state "$cnc_state" --acknowledge-phase-a
```

After exact shared-worktree/runtime preparation, one complete first-slot command
is:

```bash
./dev/run-complete-native-correction capture --package "$cnc_package" \
  --state "$cnc_state" --slot P01 --acknowledge-phase-a \
  --source "$cnc_state/worktrees/native-a" --common-git-dir "$cnc_common" \
  --gradle-home "$cnc_state/homes/prefetch-a" \
  --jdk25 "$cnc_jdk25" --jdk25-archive "$cnc_archive25" \
  --jdk21 "$cnc_jdk21" --jdk21-archive "$cnc_archive21"
```

All profiles retain the exact machine argument arrays. Change only these slot
bindings in the command above, in order; a failure stops advancement:

| Slot | Source suffix | Home suffix | Additional prerequisite |
| --- | --- | --- | --- |
| P01 | `native-a` | `prefetch-a` | Empty source output/cache state and home. |
| P02 | `native-b` | `prefetch-b` | Successful P01; separate empty state. |
| D01 | `native-a` | `diagnostic-a` | Successful P02 and dependency-only seed. |
| D02 | `native-b` | `diagnostic-b` | D01 root report captured. |
| M01 | `native-a` | `materiality` | D02 root report captured and stability gate passed. |
| M02 | `native-a` | `materiality` | Successful M01; reuse its state. |

After P02 run `seed --package FILE --state ROOT --acknowledge-phase-a`.
Before M01 run `stabilize` with the same flags; its 120-second quiescence and
seven samples count toward the same deadline. At 30 minutes use `review` to
record actual progress and `continue` or `stop`; no subsequent start can bypass
the checkpoint. These modes never execute extra Gradle probes.

### Detached process ownership

The guardian is a transient **user** service, not an installed service or host
policy change. It records its exact unit/invocation/cgroup/boot/deadline binding.
Native children enter its delegated worker cgroup atomically at creation.
They may create new sessions and keep daemons alive between successful requests.
The [kernel cgroup v2 interface](https://docs.kernel.org/admin-guide/cgroup-v2.html)
provides inherited subtree membership and `cgroup.kill`; a PID-safe directory
descriptor scopes cancellation to this campaign's original group.

Each attempt's lease records the capture client's PID and start ticks, so PID
reuse is not evidence of a live owner. Lease expiry or client disappearance
terminates the owned subtree and retains an ownership-stop reason. An unfinished
reservation then refuses replay. `KillMode=control-group`, `Restart=no` and a
service runtime limit at least ten seconds before the immutable deadline provide
an independent backstop, including idle/human waits. Losing or replacing the
unit never resets spend or authorizes a new guardian for the same state.
At a scoped stop, stop only the exact unit recorded in `ownership.json` through
the user manager; inspect its inactive state. This requires no extra Gradle
`--stop` start and must never target unrelated units or processes.

The state checker is also available as:

```bash
./dev/check-complete-native-correction capture --package "$cnc_package" --state "$cnc_state"
```

Here `cnc_package` and `cnc_state` must identify an existing recorded campaign,
not newly invented paths. Check mode creates no campaign files, but invoking
the Go wrapper can populate a regenerable Go cache: it is not strict read-only
filesystem execution. Inspection currently needs the original selected report
under its recorded source root as well as the retained copy. Preserve both;
missing or changed original evidence is a refusal, not automatic repair.

## Attempt evidence and tested refusals

Each attempt exclusively creates its slot directory, publishes an
argv/environment request and hash-bound reservation before spawn, then retains
`started.json`, separate `stdout.log`/`stderr.log`, merged `child.log`, raw `process.json`,
selected report copy/hash, required artifacts and terminal `result.json`.
M rows require `operations-log.txt`, `task-graph.jsonl`, exact retained JAR bytes
and `output-inventory.json`. Inventory reconstruction checks every frozen
selector, path, size, SHA-256 and unique declared producer; missing, additional,
unowned, ambiguous or symlink outputs refuse qualification. Pre/post verified
source/runtime/path bindings are separately retained and checked against the
frozen manifest, not trusted as a self-consistent edited summary. An interrupted
or failed post-check may lack the after record but cannot become child success.

The checker reparses the raw log with the v2 selector, compares the original
selected report with its retained bytes, reconstructs terminal classification
and required artifact hashes, and rejects summary/request/log drift. It also
reconstructs native argv/environment against the contract, process markers and
signal consistency, and rejects duplicate-key/trailing/unknown JSON and symlink
evidence. Identical
references deduplicate; distinct references remain ambiguous. Missing reports
never become successful diagnostics. Unknown/occupied slots, boot drift,
expired budgets and unresolved reservations refuse resumption. A pre-start
failure retains its reservation but records no successful spawn. A terminal
publication failure retains raw evidence and prevents silent replay.

Generic tests cover process-group cancellation; the separate host gate above
proves detached subtree ownership through the actual cgroup control path.
The 100-MiB combined-log cap and sampled
resource guard are harness limits, not proof of a kernel-enforced disk quota.
Disk is checked before/after starts and during children at one-second polling
intervals. R01/R02 automatic replacement is not implemented; a failed row
stops instead of being retried. No reserve or later F/C/V slot can start here.

## Next boundary and limits of proof

CNC-003 qualifies this native capture harness, not the experiment. The first
CNC-004 window recovered exact source/worktree identity and both real Corretto
archives, but preflight exposed an implicit-directory verifier defect before
Gradle. Corretto lists files under `man` without a `man/` header. The corrected
verifier accepts only required directory ancestors, still rejecting extra
files/directories and symlinks. Generic negatives and an explicit read-only
check against both real installed trees passed; neither JVM nor Gradle runtime
compatibility follows from archive parity alone.

The owner subsequently approved a second corrected-package/two-hour attempt,
retaining the first failure and cost, with temporary performance and balanced
restoration. That attempt's [P01 record](../../benchmarks/results/complete-native-correction-v1/cnc004-private-home/README.md)
shows successful native Gradle followed by a private-home post-check failure.
The generated-runtime-state correction passes the actual retained home and
six-slot consuming fixtures. The original failed row remains unchanged, both
guardians are stopped, and balanced restoration was verified. No P02/D/M,
candidate or value row existed at that checkpoint. The owner then approved a
third two-hour window. Its [two preparations passed](../../benchmarks/results/complete-native-correction-v1/cnc004-empty-source-state/README.md),
but D01 refused before spawn because Kotlin left an empty, unignored
`.kotlin/sessions` tree. The bounded directory-only repair passes synthetic and
retained real-state checks without upgrading that failed reservation. The third
guardian/profile hold stopped and balanced restoration was verified. Two starts
in this window, three across all windows, no fresh strict report or M/value row.
A further package/attempt needs an explicit continuation/budget decision; none
of the three campaigns is overwritten or silently reset.

The separate recipe/real fixture package is frozen in CNC-005 before F01 after
native admission. Its absence is not replaced by hashing a nonexistent recipe.
Retain both identities. F/C/V execution modes and automatic R replacements are
not implemented by this native-only block; no such row is advertised as usable.
Static/fake-child research time is not customer machine-payback evidence.
