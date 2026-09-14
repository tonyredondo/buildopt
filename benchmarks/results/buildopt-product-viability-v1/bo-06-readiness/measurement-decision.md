# Proposed measurement change before the long replay

14 September 2026 · **Proposal only. Not approved, implemented or allocated.**

I recommend continuing the Checkstyle investigation with a smaller measurement
setup. The next result should answer whether the correction saves time across
changing code, with the recording needed to check the results explicitly
included. Repeating the failed 100 ms precision campaign is not justified.

This proposal changes the instrument admission rules, not the candidate or
the required savings. Until it is approved and implemented, the existing
contract remains binding and BO-06 stays partial.

## Proposed experiment

Remove the diagnostic Java agent from the primary run. Keep the existing lean
task/output recording on both sides because the complete-output comparison
depends on it. Use the same runner's complete request boundaries, independent
state and exact source checks. Do not subtract estimated recording time from
any measured duration. Charge all required work outside the request once.

The immediate claim would be precise: **savings for this workflow with the
declared recording enabled**. This does not establish the same saving for a
plain, uninstrumented Gradle command. Recording may affect the two versions
differently; that limitation must remain visible even after a passing replay.
BO-09 must measure the eventual installed experience against the ordinary
workflow before a product-performance claim. If that result loses the benefit,
earlier research savings cannot justify an MVP performance claim.

Use external process samples for diagnosis, with missing observations and gaps
reported. They must not become the request clock or proof of continuous CPU
isolation. Exact process ownership, cleanup, native CPU affinity enforcement,
request boundaries and complete output comparison remain required. A missing
required observation still prevents acceptance of the affected result.

## The rule changes that need approval

| Existing rule | Proposed treatment |
|---|---|
| Actual-owner recording imbalance must be within 100 ms, including the registered precision confirmation | Preserve the failed qualification. Do not reuse it as a passing receipt. Admit a separately named research mode whose baseline and candidate both include the declared lean recording, after the control below. Keep the limitation on uninstrumented claims. |
| Endpoint-inclusive process snapshots must have no gap over 500 ms | Preserve the failed observation result. In the new mode these samples are diagnostic; exact request timing and process ownership remain mandatory and must have independent proof. No isolation claim. |
| Primary timing must exclude detailed diagnostic tracing | Remove the Java phase agent. Keep and disclose only the recording required by the existing output checker. |
| Both complete repetitions must pass the existing value criteria | Unchanged: 5%, one second, coverage, cost recovery, statistical support, tail limit and correctness all still apply. |

## Steps before any protected change runs

1. Introduce a separately named, versioned research mode. Qualify its actual
   invocation and comparison paths locally: exact inputs, report checks,
   failures, cancellation, source drift, missing results and closed processes.
   It must reject a confirmation manifest that lacks this mode's own readiness
   receipt. Merely filling the old manifest's `overhead` field is insufficient.
2. Freeze one identical-code control on development changes 17–20: two
   independent state directories, both with the same supported correction;
   change 17 is cold and 18–20 are measured. Alternate order. **Eight builds
   total**, with four live and four independent output comparisons. No retry
   or replacement. This is a check for a material false difference in this
   control, not a general precision estimate.
3. Reject that control if the absolute difference over all three measured
   requests is both at least one second per request and at least 5% of the
   faster arm's total. Any correctness or required-evidence failure also stops
   progression. Retain each pair, both cold requests, all host flags and the
   total including cold starts. A pass cannot prove that recording affects
   ordinary and corrected Gradle equally.
4. Run the complete development prefix once: anchor 0 and changes 1–20, native
   and corrected, with private state preserved within each arm. **42 builds**,
   with 21 live and 21 independent comparisons. Establish source applicability,
   fallback behavior, outputs and state transitions. This is a prerequisite
   check and remains exploratory. Across all twenty non-cold changes, it must
   also save at least 5% overall and one second per scheduled change after
   required outside-request costs. Keep the cold result and preparation costs
   visible. A negative result does not advance to BO-07. Any candidate mismatch
   or unexplained harness failure stops the attempt. No repair is inserted
   into the run.
5. Freeze the final implementation and actual measurement sources, complete
   history, cost rules and a feasible allocation for BO-07. Size time and disk
   from the observed prefix, including snapshots and comparison JVMs. Only
   then consider the separately allocated 404-start confirmation.

The proposed BO-06 allocation has a ceiling of **50 workflow starts and 50
standalone comparison JVMs**, including failures. There are no retries or
additional CPU profiles. Allow at most ten elapsed hours, 80 GiB of new state
and retain at least 40 GiB free disk; each build remains bounded at 900 seconds.
These are ceilings, not a promise that the checks will finish. Stop at a limit
with the partial result preserved. Local consuming-path tests need their own
small, recorded ceiling before execution and must not start owner builds.

If the control or prefix fails, keep that result and stop this allocation.
Do not replace it with another favorable window. If they pass, BO-07 is still
a separate allocation and the final decision still requires both complete
repetitions and later replication in other repositories.

Approval would authorize implementing and qualifying this research measurement
mode within BO-06. It would not mark the old gates passed, start protected
changes 21–100, or establish product viability.
