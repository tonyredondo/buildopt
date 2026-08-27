# Sticky-wrapper paired trial v1

This is the `SWL-010` experiment contract for the owner-operated sticky-wrapper
learning POC. It turns ordinary observations into a bounded comparison between
the candidate command and an authoritative native Gradle command. It is not a
production scheduler and it cannot authorize a durable action.

## Assignment

The assignment is created before either command runs. Pair numbers are
one-based: odd pairs run `CANDIDATE_FIRST`, even pairs run `NATIVE_FIRST`.
Order is therefore balanced without looking at a duration, outcome or output.
The scheduler allows one active trial per repository scope. A developer build
is never duplicated locally; trials are trusted-CI work only.

## Budget

The caller supplies eligible natural runner time. The default POC ceiling is
`50/1000` of that time (5%). A reservation is made before the pair starts and
the actual cost is the sum of both direct command durations. Completion releases
unused reservation; cancellation charges observed work and releases the rest.
An exhausted budget rejects the next assignment and retains native execution.

## Isolation and correctness

Each pair receives separate project copies and distinct Gradle homes, daemon
homes, cache roots and BuildOpt state roots. Commands are executed directly,
without a shell, with an explicit child environment. Exactly two invocations
are recorded per trial. Required output trees are hashed by sorted relative
path and bytes; any missing output, symlink or differing digest makes the trial
`INCONCLUSIVE` and cannot support a value claim.

The customer's requested build remains authoritative. The trial report is
descriptive evidence only; qualification still requires the decision-store and
later longitudinal gates. A candidate that is faster in one pair is not an
activation decision.

The first checked-in result is intentionally retained even though it is not a
win: candidate mean wall time is 7.534 s, native mean is 6.979 s, mean saving
is -0.555 s, and zero of four pairs are positive. All four output digests are
exact. This is a mechanism proof and an overhead diagnostic, not evidence that
the sticky wrapper currently improves this workload.

## Evidence and checker

The implementation lives in [`internal/stickytrial`](../internal/stickytrial)
and the real fixture command is [`sticky-trial-benchmark`](../cmd/sticky-trial-benchmark).
Run:

```bash
./dev/check-sticky-wrapper-trial
```

The checker generates a 256-class Java project, runs four alternating pairs
with the repository-local Gradle distribution and requires eight invocations,
four unique isolation digests, exact required outputs and budget usage at or
below the declared 5% limit. The output retains every raw pair and can be
recomputed by a later evaluator. This fixture is a mechanism/contract proof,
not a claim that the current candidate beats optimized native Gradle.
