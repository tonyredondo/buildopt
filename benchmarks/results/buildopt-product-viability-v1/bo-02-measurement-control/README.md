# Identical-code timing control

BO-02 completed on 14 September 2026. Decision: **`NO_SPURIOUS_MATERIAL_SIGNAL_IN_THIS_CONTROL`**.

This control did not reproduce a material timing difference with identical code. The registered control is complete; measurement precision and optimization value remain unproven.

## What ran

Four Elasticsearch builds replayed the same change from original commit ordinal 6
to ordinal 7. Both sides used the existing ForbiddenPatterns correction and the
same supported V2 Checkstyle correction. Checkstyle checks source formatting and
coding rules; the correction lets it reuse checks on unchanged files while
preserving complete reports.

The command was `:server:precommit --continue`, including its dependencies, with
Gradle's build cache enabled and Configuration Cache disabled. The pinned runner,
JDKs, instrumentation, output contract and schedule were reused unchanged. Native
work used CPUs 0–7, observation CPU 8 and the controller CPU 9. Each side had its
own working directory and cache. The [manifest](./manifest.json) records the full
command, source revisions and input hashes.

Both sides compiled identical effective source. The original runner installs the
N correction during preparation and applies the I correction inside its measured
request; that application cost remains included. These are arm labels, not an
unoptimized-versus-optimized comparison.

## Timings and decision

| Original ordinal | Arm | Role | Native seconds | Whole-request seconds |
| --- | --- | --- | ---: | ---: |
| 6 | N | Cold | 220.700 | 220.782 |
| 6 | I | Cold | 207.863 | 208.000 |
| 7 | N | Measured | 83.947 | 84.035 |
| 7 | I | Measured | 81.182 | 81.294 |

The signed measured difference, N minus I, is **2.742s**.
Its absolute size is **3.37% of the faster
request**. The threshold fixed before execution was at least one second **and**
5% of the faster request. Cold builds remain in the report but do not enter that
decision. Whole-request time includes recorded work on the build machine outside
the native command where the original accounting requires it.

These timings describe the control. They provide no new estimate of optimization
savings and do not change the previous mixed finalization result.

## Same work and complete results

All four builds succeeded. Both output pairs passed the live comparison and the
independent reconstruction, using all four allowed comparator JVMs. The verifier
checked all tracked source files and the six effective patched files at both
revisions. Both arms produced matching complete outputs under the existing C5
contract. C5 is the retained check of declared files, reports, producers and
permitted normalization; it was not reduced for this control.

Actual Checkstyle work on the measured change was:

| Arm | Task | Input files | Checked again | Reused |
| --- | --- | ---: | ---: | ---: |
| I | :server:checkstyleTest | 2825 | 6 | 2819 |
| I | :server:checkstyleMain | 4959 | 5 | 4954 |
| N | :server:checkstyleTest | 2825 | 6 | 2819 |
| N | :server:checkstyleMain | 4959 | 5 | 4954 |

The third Checkstyle task, `:server:checkstyleInternalClusterTest`, was already
up to date on both measured builds. Gradle skipped it. Its retained success
state was also verified.

The [verification](./verification.json) records source identity and run limits.
The [task records](./task-execution.json) preserve execution and reuse counts.

## Order and machine activity

| Measured order | Arm | Seconds since its own cold build ended | Host flag |
| --- | --- | ---: | --- |
| 1 | I | 156.1 | FLAGGED |
| 2 | N | 1106.3 | FLAGGED |

The idle intervals include file preservation, checks and the other arm's work.
They are not pure daemon sleep. [Host observations](./host-pressure.json) retain
every request and sampling limitation. CPU affinity does not isolate this shared
workstation; a pressure flag does not identify the cause of a slow build. No
sample was removed and no timing-driven retry was run.

## Limits and next step

One measured pair cannot establish precision, long-term savings or product
viability. G0 and G3 remain unqualified. The protected 21–100 history and other
repositories were not built.

The allocation closed with four owner starts, four comparator JVMs and
12.09 GiB of local state, inside the registered
90-minute and 32 GiB limits. Both owned service scopes closed. No historical
allocation or result was overwritten. Changes remain local; no commit or push
was performed.

Proceed to BO-03: estimate how much of a complete build each retained candidate could realistically save, including code changes where it does nothing. BO-04 can assess the focused Build Impact cases. A later short Checkstyle comparison still needs its own readiness decision and fixed inputs; this control does not start the 17–20 screen or the protected validation history.

Follow the [governing tracker](../../../../docs/plans/buildopt-research-execution-plan-2026-09-14.md)
and the [registered control protocol](../../../../docs/plans/buildopt-checkstyle-measurement-control-v1.md).
The [evidence index](./evidence-index.json) identifies exact copies and the retained
local raw evidence. The export includes reports, inputs and controller changes;
full binary outputs and process samples remain at their bound local paths.
