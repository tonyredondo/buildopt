# Complete native correction: two preparations and an empty-directory refusal

Date: 2026-09-06. Evidence: `E-555`. Decision:
`INCOMPLETE_EXPERIMENT_INPUT`. P01/P02 passed native execution and post-checks;
D01 was reserved but refused before spawn. CNC-004 remains incomplete.

## Authority and execution identity

The owner explicitly approved a third attempt with the corrected package and a
new maximum of two hours, retaining all earlier results and costs. The temporary
performance profile and restoration to balanced remained in scope. Neither the
[first preflight failure](../cnc004-preflight/README.md) nor the
[second private-home failure](../cnc004-private-home/README.md) was overwritten.

Execution used committed BuildOpt
`3daa2718f84a138dc2211725d5d0902d0d2d8163`, after its
[Base CI](https://github.com/tonyredondo/buildopt/actions/runs/34034249614) and
[Native Platform CI](https://github.com/tonyredondo/buildopt/actions/runs/34034249571)
completed successfully. The [package](./package.json) SHA-256 is
`aedf31345276db2e60fea32c895b187cded4f7f40aeb6b6912550d3770c0dc17`;
the executable digest is
`ba6ad59c865c7dfc8025a4a3c8632c4d4ea5c91a0711ecf48926fb7450b1bafc`.
Every package source matched that committed Git tree before execution.

The new [state](./state.json), digest
`62adcda317d8d57ff0756b20b2fc9ad11df1cbad4879a4b7f27b3d59be456cae`,
started the 7,200-second boot-bound window before execution preparation, with
60 starts maximum and review at 1,800 seconds. Three new registered detached
worktrees share the already recovered Git repository. The frozen GraphQL
revision, full archive and file/span bindings passed; both exact copied Corretto
archive/install trees and the mains/performance/EPP envelope passed preflight.
No new clone, source copy, dependency upgrade or owner-home reuse occurred.

## Observed result and cost

| Slot | Actual native start | Native process seconds | Independently reconstructed outcome |
| --- | --- | ---: | --- |
| P01 | Yes | 120.130429109 | `CHILD_SUCCESS`; input-before/after digests match. |
| P02 | Yes | 200.013986002 | `CHILD_SUCCESS`; separate fresh home/worktree, input-before/after digests match. |
| D01 | No | 0 | `PRE_START_FAILURE`: `generated state is not declared ignored`. |

P01/P02 each ran the unchanged `assemble testClasses` preparation profile and
executed 21 tasks. They compiled test classes, not the owner test suite. Their
320.144415111 combined process seconds are preparation cost, not a paired value
comparison, materiality result or saving. A dependency-only seed completed with
1,521 files and inventory digest
`866d34d817d7b12824a8a6d659dc243072df080b63ecd3f3764c84aecfeeb234`.
It contains only dependency/Wrapper content, not output, execution or daemon state.

One premature D01 client was rejected with `campaign already active` while the
seed writer held its lock. That operator sequencing refusal created no attempt
reservation or Gradle process. After the seed completed, the actual D01
reservation reached source-state preparation and produced the retained
[pre-start result](./attempt-result.json). It has no `started.json`, successful
input-before/after record or native process duration. Empty stdout/stderr are
correct here; they are not a lost strict report.

Kotlin had left `.kotlin/sessions` as a directory-only tree: no regular file,
symlink or other member remained. The frozen source's ignore rules cover
`.gradle` and `build`, not `.kotlin`. Git does not track empty directories, so
the tracked/untracked status was clean while the runner's unconditional
`git check-ignore` test refused this empty root. This is a source-preparation
harness defect, not a Gradle failure or unsupported Configuration Cache blocker.

Before that refusal, D01 had already archived the ignored `.gradle` tree under
its own `prior-state`; it remains preserved there. The other generated roots,
including the empty `.kotlin/sessions`, remain at their observed paths. No
reset, forced cleanup, source patch or manual restoration was performed.

The exact guardian and its bound profile hold were stopped. Both were absent
and `balanced` was confirmed at boot time `3478251.80`, 746.29 seconds after the
third window began. This is a shutdown observation, not an exact service-exit
timestamp or the full repair/publication charge. Subsequent repair, validation
and waiting remain additional elapsed cost. The attempt stopped before its
30-minute progress checkpoint; no deadline was reset.

There are two real starts in this window and three across the retained windows
(0 + 1 + 2). D02/M01/M02 did not run. There is no fresh strict report, controlled
materiality, public candidate, correctness comparison, value sample or speedup.

## Retained evidence and bounded repair

The [raw archive](./p01-p02-d01-evidence.tar.gz), digest
`626bacca9090c20e462bbddb79dc5f6e11131add7ebb0a287a4f9a0d48376d0b`,
contains original state/ownership/guardian-request records, the complete
dependency-seed inventory and P01/P02/D01 capture files. Archive comparison
against the original campaign passed. Large generated `D01/prior-state` cache
bytes, dependency/runtime binaries and source worktrees remain local and are
explicitly not included in this compact archive. Their exact recovery roots
remain in the task-scoped record; neither the archive nor the repair supplies
a missing native diagnostic.

The correction still restricts preparation to the declared generated roots,
rejects tracked content, requires ordinary filesystem members and preserves
state by moving it into the attempt archive. Only a tree containing directories
and no files may proceed without an ignore match. Any file, including a
zero-byte or hidden nested file, still requires the original ignore proof.
Links, FIFOs and files replacing roots refuse. Fresh P slots still reject
preexisting generated directories. No public ignore rule is changed.

`TestNativeEmptyUnignoredGeneratedState` reproduced the original refusal and
passed after repair. Eleven boundary cases and the existing source/report
preservation check pass. The six-slot native consumer now creates the actual
empty project-local shape. Runner race tests passed in 21.241 seconds; the
explicit host ownership/consuming gate passed in 21.515 seconds. These are
synthetic harness proofs, not additional public Gradle starts.

The opt-in read-only real-state check accepts the retained directory shape and
independently reconstructs the original three rows, keeping D01 unstarted and
failed. With the existing absolute campaign root resolved as
`cnc_retained_empty_state`, repeat:

```bash
CNC_EMPTY_GENERATED_CAMPAIGN_ROOT="$cnc_retained_empty_state" \
  ./dev/run --toolchain go -- go test -count=1 \
  -run '^TestRetainedEmptyGeneratedSourceState$' -v \
  github.com/tonyredondo/buildopt/dev/complete-native-correction-runner
```

Its ordinary-suite skip is not real-source proof. The repair does not rewrite
the frozen package or make D01 resumable. Another execution requires its own
package and explicit continuation/budget decision, with all prior cost retained.
CNC-005..007 remain behind the unchanged native admission gate.
