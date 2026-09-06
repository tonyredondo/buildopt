# Complete native correction: native preparation and private-home refusal

Date: 2026-09-06. Evidence: `E-554`. Decision:
`INCOMPLETE_EXPERIMENT_INPUT`. One real native preparation completed with exit
zero, but its post-child harness check failed. CNC-004 is not qualified.

## Authority and frozen inputs

The owner explicitly approved a second attempt of at most two hours with the
corrected package, retaining the first attempt and all earlier costs, and a
temporary `performance` profile restored to `balanced` afterward. This is a
separate window, not an in-place reset of the
[first preflight refusal](../cnc004-preflight/README.md).

The execution revision is `b613fe46f9df43b98dafd237394a65e6c1e1c668`.
Its [Base CI](https://github.com/tonyredondo/buildopt/actions/runs/34031172987)
and [Native Platform CI](https://github.com/tonyredondo/buildopt/actions/runs/34031172991)
completed successfully before this attempt. The [package](./package.json)
SHA-256 is `609994ee1e30e6bac6cd4773781bc13b30c2535a0a8b9bbb23ef1f18728080c6`;
the retained executable SHA-256 is
`126c45dc8b5b7ac20e2b43712aafe59b7d0f57f9efe55072d1dd829b3f278f6b`.
Every package source digest was independently compared with that committed Git
tree before this repair, not with subsequently edited source.

The [state](./state.json) freezes 7,200 seconds, 60 starts and review at 1,800
seconds. Its SHA-256 is
`6b34c2a3b6a34a7c25eba7972ee540404a4fc2373cf6a289decfb80358039bc2`.
The clock started before execution setup. Three new registered detached
worktrees share the previously recovered bare repository; no new clone or
source copy was made. Original worktrees and first-window inputs remain intact.
The fresh source archive and all source/span bindings matched the frozen
GraphQL revision. Copies of the exact retained Corretto archives/install trees
passed full preflight, with mains, `intel_pstate` and `performance` EPP on CPUs
0-3 confirmed. The temporary profile hold was bound to the exact guardian and
remaining deadline, with no restart policy.

## What ran, and what did not

P01 invoked the unchanged Gradle 9.6.1 Wrapper with the frozen
`assemble testClasses` preparation profile. Both pinned Java runtimes were
verified and supplied through explicit installation paths and private homes;
source remained unchanged. The raw process
record reports exit zero and `118993481920` elapsed nanoseconds. Gradle printed
`BUILD SUCCESSFUL`; 21 actionable tasks executed. This is preparation cost,
not a value sample or a saving. `testClasses` compiled tests; it did not run
the owner test suite.

The [retained attempt](./attempt-result.json) is nevertheless `HARNESS_FAILURE`:
`post-child input verification: unexpected private user-home directory`.
The fresh private home acquired JVM/Groovy preferences under `.java/.userPrefs`
and Kotlin daemon metadata under `.kotlin/daemon`. The former checker admitted
only empty `.m2/repository` directories and rejected normal runtime-generated
state. The private Maven repository remained empty. This is a capture-policy
defect, not a failed native build or a candidate regression.

P02, D01/D02 and M01/M02 did not run. There is one real Gradle start, no fresh
strict diagnostic, no controlled materiality row, no public patch, no candidate
or value sample, and no speedup claim. Five JAR files exist after preparation,
but no paired output comparison or instrumented producer inventory was captured;
their existence is not correctness qualification.

The frozen runner's independent `check` reconstructed the failure before code
changed. The exact guardian was stopped at boot time `3473279.42`, 348.51
seconds after the second window began. Both guardian and profile hold were
absent afterward, and `balanced` was observed again. This is elapsed time
through shutdown, not the full repair/publication cost. No original clock,
attempt, source tree or runtime input was overwritten or removed.

## Retained raw evidence

The [raw archive](./p01-evidence.tar.gz) contains the immutable state/ownership
records and complete P01 reservation, request, lease, start/process/result,
input-before record, stdout, stderr and combined log. Its SHA-256 is
`7ad84f07cd277f0483f21f09263d2fee04694647387e82bae5468ed8d2777155`.
There is intentionally no successful input-after record: that check failed.
The archive preserves actual execution identities; it is not shared machine
configuration. Original source/home paths remain in the task-scoped recovery
record because independent replay requires those exact retained roots.

## Bounded repair and proof boundary

The corrected policy admits only ordinary files/directories below the two
runtime-owned roots, plus their required directory ancestors. It still rejects
Maven settings/artifacts, private Gradle settings, unrelated roots, prefix
lookalikes, files in place of roots, symlinks and non-regular members. Fresh
homes must still be newly created. Dependency seeding never copies this runtime
state to another home or arm; M02 keeps only its own evolving M01 state.

The generated-state fixture and the retained actual home both reproduced the
old refusal, then passed after repair. Negative cases cover the forbidden
inputs above. The six-slot fake native consumer now creates Java/Kotlin state
so the actual pre/post consuming path exercises it. Race tests passed (runner
19.909 seconds); explicit host ownership and six-slot integration passed
(21.026 seconds). These are harness proofs, not a second public Gradle run.

An opt-in read-only check accepts the retained real home and independently
reconstructs P01 as the same historical `HARNESS_FAILURE`, never upgraded to
success. With the existing absolute paths resolved as `cnc_retained_home` and
`cnc_retained_state`, repeat:

```bash
CNC_PRIVATE_RUNTIME_HOME="$cnc_retained_home" \
CNC_RETAINED_CAMPAIGN_ROOT="$cnc_retained_state" \
  ./dev/run --toolchain go -- go test -count=1 \
  -run '^TestRetainedPrivateRuntimeHome$' -v \
  github.com/tonyredondo/buildopt/dev/complete-native-correction-runner
```

The ordinary fixture suite skips this opt-in proof. Neither that skip nor the
repair substitutes for a newly frozen, explicitly authorized public attempt.
The native admission gate remains incomplete, and CNC-005..007 cannot advance.
