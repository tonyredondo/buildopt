# Next Checkstyle measurement

Proposal fixed on 15 September 2026. No builds have started under this proposal.

We need to find out whether the correction saves time across all 20 development
changes. Selected comparisons have passed, but the complete sequence remains
unmeasured with this recording method.

| Step | Work | Required result |
| --- | --- | --- |
| 1. Activate the allocation | Recheck the frozen source, executable, policy, readers, command and history. Create a separate allocation and set its deadline once at launch. Update only the proposal's run location, deadline and remaining limits. | The measurement identity stays unchanged. All preparation time and storage count toward the allocation. No synthetic admission receipt enters the run. |
| 2. Run the control | Use original changes 17–20, with the same correction on both sides. The first pair is cold; measure the remaining three pairs. Run four live and four independent output comparisons and close both process sessions. | Stop if the aggregate difference is at least three seconds **and** 5% of the faster side. All eight builds and all output checks must complete. |
| 3. Admit development | Attach the genuine control result, its independent checks, costs and process closure to development readiness. Run the full manifest validator before creating any development build state. | The real development manifest is accepted with this exact policy and implementation. The control's entire consumption is deducted from the shared allocation. |
| 4. Replay development once | Start from fresh, separate state for each side. Run anchor 0 and every change through 20, comparing native behavior with the fixed correction. Keep all 42 scheduled builds, including those where the correction does nothing. | All outputs match. Across the 20 non-cold changes, save at least 5% overall and one second per scheduled change after required work outside the request. Record preparation separately; a prefix pass does not prove that preparation has been repaid. |
| 5. Close the result | Independently check all 21 pairs, reconcile counts and costs, and verify that the experiment's processes have stopped. Publish the complete result. | A passing sequence permits the remaining BO-06 freeze. A negative or incomplete sequence stays negative or incomplete. Do not replace commits, retry favorable cases or start BO-07 automatically. |

## Limits

- 50 owner builds: eight for control and 42 for development.
- At most 50 comparator JVMs: one live and one independent comparison per pair.
- Ten hours from allocation start, including preparation and verification;
  at most 15 minutes per build request.
- At most 80 GiB of new state, with at least 40 GiB free and at most 256 MiB
  of diagnostic samples.
- No retries, additional CPU profiles or protected validation builds.

The preceding control and closeout took about 74 minutes and retained 13.2 GiB.
The ten-hour limit allows for the longer sequence and its output checks. The
storage limit is a cap, not a guarantee: check remaining space as state grows
and stop if it is reached. These are proposed execution limits; the completed
qualification allocation cannot supply unused starts to this experiment.

The [machine-readable proposal](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-comparator-qualification/inputs/next-measurement-proposal.json)
and [frozen input record](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-comparator-qualification/analysis/measurement-freeze.json)
fix the current candidate, command, output policy and measurement identity.
The stored manifests are proposals; their deadlines must be activated for the
new allocation before execution. Readiness may gain genuine trial evidence;
changing any measured input requires a new freeze and control.

A successful development result would still leave the 80 protected changes,
two full confirmation repetitions and preparation recovery to be tested.
Confirmation must include an evidenced preparation cost for each repetition.
The recording method also remains in both sides of this experiment. A saving
against ordinary Gradle and any benefit from automatic adaptation are later
questions in the governing plan.
