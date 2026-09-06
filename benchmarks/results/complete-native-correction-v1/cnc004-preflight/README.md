# Complete native correction: first-window preflight refusal

Date: 2026-09-06. Decision: `INCOMPLETE_EXPERIMENT_INPUT` before any Gradle
start. CNC-004 is blocked, not a successful native/materiality result.

The owner approved reacquiring the missing exact GraphQL Git input once and
using registered shared detached worktrees. The committed harness was
`e9fc62a406aef373bad3d5cdef2a107e6a607aaf`, published normally to `main`.
The [execution package](./package.json) is distinct from CNC-003's earlier
uncommitted snapshot; its digest is
`30ce6416d93b473cee5518c01f9095a036abbbc866b67b1225a311b260ebe1ba`.
The retained [campaign state](./state.json) starts its immutable two-hour clock
before source acquisition, runtime downloads, extraction and validation.

## What actually happened

One new bare source repository was fetched at
`f2d8c9126f898c084b176631b7346bc6fbec296a`. Native-a, native-b and candidate
are registered detached worktrees sharing that Git directory. Its fresh Git
archive SHA-256 matches
`83e80bbaa2b3e3308dd35e0b7120399f02149507674d958b0e2e4cc81718d051`.
All three worktrees remain clean and no source patch was made.

Both exact Corretto archives were downloaded and matched their frozen SHA-256
bindings before extraction. The old runner's actual `preflight` returned
`unexpected installed runtime member`. An independent archive-name versus
installed-tree comparison found exactly one unlisted member in each runtime:
the directory `man`. Both archives list its descendants but omit its directory
header, so normal extraction creates it. This is a harness verifier defect,
not corrupt JDK bytes, a Gradle failure, or a product-value result.

Before changing code, the frozen runner's independent `check` returned `[]`.
There are zero attempts/reservations and zero Gradle starts. The exact transient
guardian was stopped and its absence verified. At that stop, boot time was
3468244.62 seconds, 418.33 seconds after the original campaign start. This is
time through shutdown, not the full research/repair cost. Later repair,
validation and publication time is additional effort; the original deadline
is retained and never reset. The source/JDK input trees remain available in
the task-scoped recovery record; no worktree or runtime was deleted.

## Correction and limits of proof

The runner now derives only the necessary directory ancestors of verified
archive members. It still verifies every declared file's bytes/executable bits,
link target and containment, and rejects extra files, unrelated directories,
duplicate members and ancestor symlinks. The new implicit-directory regression
failed on the original implementation and passed after this correction.

An explicit read-only integration test passed against **both real downloaded
archives and complete installed trees**. It creates no campaign and starts no
JVM or Gradle process. To repeat with the retained local state root:

```bash
CNC_PINNED_RUNTIME_ROOT="$cnc_retained_state" ./dev/run --toolchain go -- \
  go test -count=1 -run '^TestPinnedRuntimeArchives$' -v \
  github.com/tonyredondo/buildopt/dev/complete-native-correction-runner
```

The normal suite skips that opt-in test; a skip is not real-archive proof.
The explicit run passed in 9.366 seconds. Uncached runner/selector/canonical-JSON
race tests passed (runner 19.424 seconds); static contract and Go vet passed.
These are verifier/fixture results, not Gradle compatibility or materiality.
The [retained repeat log](./runtime-parity.log.gz) also passes (9.963 seconds).
Its compressed SHA-256 is `b1141ec58161a3a1346448b336063ff183464969f4346f4129ba92da15ec202f`;
the decompressed SHA-256 is `0f49cefc4d2247f8409022c3f0453105ac674b773d42f9019e19c5909cf2772c`.
All package source-file digests were independently compared with the original
committed Git tree, not the later repaired files. State arithmetic still gives
7,200 seconds, a 1,800-second review and 60 starts maximum; the original attempt
directory was independently confirmed empty.

The first [Base CI run](https://github.com/tonyredondo/buildopt/actions/runs/34028916140)
failed at the newly integrated generic fixture command (line 283), whose stdout
had been discarded. The individual failing case is therefore unavailable from
that log; it is not attributed to Corretto or called a resolved flaky test.
The local suite also passes with `CI=true` and `GITHUB_ACTIONS=true`. The two
CNC integration commands now preserve their output without skipping or
weakening assertions; hosted follow-up must establish the corrected revision's
actual result. The first Native Platform CI passed independently.

The [second Base CI run](https://github.com/tonyredondo/buildopt/actions/runs/34030348339)
now exposes `TestActualCLIInitializationAndReadOnlyCheck` refusing its freshly
initialized state with `state limit or identity drift`. This is a separate
harness bug: subtracting fractional boot timestamps can produce
`7199.999999999998` rather than exactly `7200`. A deterministic initialization,
JSON round-trip and read-only validation at boot time `12345.67` reproduces the
original failure. Boot time `248.01` covers the same issue for the review limit.
The corrected reader compares against the exact sums written by initialization,
without an epsilon or changed duration. Regressions also reject the next
representable float in either direction for both limits and refuse expiry.
Native Platform CI passed on that second revision; Base CI must be followed
again after this independently reproduced correction. Neither repair mutates
the first campaign state or starts a replacement campaign.

The first package/state/binary identities remain frozen. The repair does not
overwrite that campaign or authorize replacement execution. A separately
frozen corrected package and an explicit execution continuation decision are
needed before P01. No reserve start, candidate, diagnostic, materiality row,
timing sample or speedup exists. CNC-005..007 cannot advance from this refusal.

The [result record](./result.json) summarizes observed preparation and refusal,
not a successful aggregate gate. The original executable is retained in the
task-local package directory with SHA-256
`0d2347052b96f9079e1253493a8d620af45f9dd27d8772d9d15b456d9a36a636`.
