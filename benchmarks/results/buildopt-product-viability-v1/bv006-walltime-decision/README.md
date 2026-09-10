# Walltime decision: comparison incomplete after supervisor failure

Date: 2026-09-09. Outcome: `INCONCLUSIVE_HARNESS_OWNERSHIP_FAILURE`.
The requested three-step objective is **partial**. There is no measured saving,
CPU-profile conclusion, candidate value rejection or product-readiness claim.
The [W1/W2/W3 amendment](../../../../docs/plans/buildopt-walltime-decision-v1.md)
and all profile inputs were frozen before the new timings.

| Step | State | Outcome |
|---|---|---|
| W1: cold diagnostic | verified | Native build succeeds in 206.323 s. Daemon coverage gap 615.790 ms fails the unchanged 500-ms bound. No correction or retry |
| W2: CPU screen | partial | Two warmup starts at 8 CPU. Native succeeds; supervisor aborts candidate. Zero measured comparable pairs. Profiles 4/2 do not run |
| W3: independent history and investment | deferred | No valid material signal. No history, replication or adaptive admission. Checkstyle value remains unresolved |

## Observed failure

The candidate warmup ended with signal 9 and `OWNERSHIP_OR_LIMIT_FAILURE`:
`read /proc/1282075/cgroup: no such process`. The frozen production reader
propagates the cgroup-read error; enumeration skips `os.IsNotExist` but does not
handle this `ESRCH` case. This is a failed ownership observation, not evidence
of an actual cgroup escape or a candidate correctness failure. The actual-owner
supervision path needs new proof; earlier fixture/build successes remain facts.

The native warmup request took 202.698 s. The candidate
request was aborted after 334.323 s. These are **not a valid
performance pair**. The raw runner retains failed adoption cost; its incomplete
aggregate is not mean saving or regression over the unexecuted measured changes.

Output/state capture cost 561.140 s after native and
607.495 s after candidate, outside request clocks.
During capture, Btrfs `wait_log_commit` waits and high host I/O pressure were
observed. Their ultimate cause is unproved. The controller closed its own units
before the post-failure cleanup cap; no forced cleanup was needed.

## Evidence and accounting

- [Decision and step outcomes](./analysis/decision.json).
- [Cold reconstruction](./analysis/cold.json) and [CPU failure reconstruction](./analysis/cpu-screen-failure.json).
- [Frozen inputs](./cpu-screen/inputs/freeze.json) and [prelaunch proof](./cpu-screen/analysis/prelaunch-proof.json).
- [Scope and closure audit](./independent-audit.json), [evidence hashes](./evidence-manifest.json) and [external seal](../bv006-walltime-decision-seal-audit.json).

This allocation charged 14 Gradle reservations and observed three top-level
starts: one diagnostic and two warmups. Sixteen metadata-helper reservations
were charged; no metadata helper ran. Each raw runner result retains one unknown
reservation after incomplete observation/daemon evidence. All three direct
launches have PID receipts, but complete invocation accounting is not promoted
to verified and no extra launch is inferred. Older uncertainties remain retained.
Program totals: 660 charged reservations / 550 observed Gradle starts.

No BuildOpt production source change or recompilation, non-Gradle fixture
workflow or native retry was added. The compact bundle retains receipts, logs, source, inventories and bound
output indexes. Large private repositories, dependency stores and output bytes
remain at their original local paths. This is not a portable successful owner
comparison. No state was deleted; all 80 validation transitions remain untouched.

Raw state is under `/home/tonyredondo/repos/github/tonyredondo/buildopt/.tools/state/buildopt-product-viability-v1/`,
in `bv006-walltime-decision` and `bv006-cpu-screen`. The durable locator is
`task-state.json`, key `engineeringPrefix.walltimeDecision`. BuildOpt remains on
`main` at `b76ded08c952ebb386576fafce4ae2d8fdcc09f1`, with existing dirty work preserved.

Follow the [bounded next prerequisite](./next-step.md). This failed frozen
allocation is closed and must not be resumed as though its first pair passed.
