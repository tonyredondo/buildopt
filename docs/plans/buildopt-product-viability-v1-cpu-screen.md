# BuildOpt CPU profile screening budget

Date: 2026-09-09. Program: `BUILDOPT-VIABILITY-V1`.
**Current execution: partial, closed as inconclusive.** The [walltime attempt](../../benchmarks/results/buildopt-product-viability-v1/bv006-walltime-decision/README.md)
starts two warmups at 8 CPU; an ownership-reader ESRCH aborts the candidate.
Zero measured comparable pairs. The 4/2-CPU profiles and independent history
never start. The failed frozen allocation is closed, without automatic resume
or replacement. Value remains unresolved. Follow the [bounded next prerequisite](../../benchmarks/results/buildopt-product-viability-v1/bv006-walltime-decision/next-step.md).
Historical zero-start/blocked states below are superseded by this current record;
no previous failed gate is promoted.
Planning amendment: verified. The 2026-09-09 [walltime decision](./buildopt-walltime-decision-v1.md) admits exploratory execution with diagnostic quality reported separately; formal S1/G0 remains blocked.
The owner requested substantially fewer than 80 builds per CPU profile.
Resume from the [current tracker](./buildopt-product-viability-v1-tracker.md).

## Fixed ceiling

Compare the native baseline with the frozen Checkstyle correction using three
CPU profiles. Each pair contains one baseline build and one corrected build.

| Available native CPU | Warmup builds per version | Measured pairs | Total Gradle starts |
|---|---:|---:|---:|
| 8 | 2 | 4 | 12 |
| 4 | 2 | 4 | 12 |
| 2 | 2 | 4 | 12 |
| Total | 12 warmup builds across both versions and all profiles | 12 pairs / 24 measured builds | 36 |

The ceiling includes warmup and failed starts. There are no replacement rows,
extra warmups, automatic sample-size increases or retries outside this ceiling.
If two warmups do not produce stable conditions, report that limitation or stop
as inconclusive. Do not copy the previous 80-start precision pilot into each
profile. Four pairs can guide investigation; they cannot establish a small
effect reliably or qualify product viability.

## Prerequisites and comparisons

The [observer isolation plan](../../benchmarks/results/buildopt-product-viability-v1/bv006-disk-observer/next-step.md)
remains a separate, single prerequisite experiment: at most 40 Gradle starts,
including its eight fixture starts. It is not repeated per CPU profile. Its
ceiling plus this screen is 76 starts. Isolation used its 40 starts, then
[failed observation quality](../../benchmarks/results/buildopt-product-viability-v1/bv006-cpu-isolation/README.md).
The screen has zero reservations and actual starts; no active allocation.
Existing failed gates and closed allocations remain closed.

Before screening, verify the observer's isolation, ownership, disk limits,
cancellation, source compatibility and output comparison. A failed prerequisite
leaves the screen blocked. Full statistical timing qualification is not repeated
for all three profiles: this screen reports exploratory diagnostics and cannot
pass G0 or unlock formal chronological value/confirmation work.

Freeze the exact workflow, engineering-only revisions/input sequence, candidate
hashes, affinity masks, worker limits, order and allocation before any launch.
The measured workload must exercise the correction; repeating only UP-TO-DATE
builds cannot assess its avoided work. Use identical inputs in all profiles and
both versions, independent native state per profile/version, and retained state
within each sequence. Keep the 80 held-out validation transitions untouched.

Use fresh baseline and corrected observations at eight CPU: the previous
old/new observer control had no Checkstyle candidate in either version.
Alternate baseline/corrected order within each profile and register the profile
schedule before observing timings. Fix worker limits per profile equally for
both versions; if they vary between profiles, interpret results as resource
profiles rather than attributing the effect solely to CPU affinity.

## Execution tracker

| Step | State | Work | Required outcome |
|---|---|---|---|
| S1: qualify prerequisites | blocked | Recover the isolation result and exact source/output/lifecycle proof | Usable observation and compatible candidate, or explicit blocked status |
| S2: freeze the short screen | partial | Register the identical engineering workload, all CPU/worker settings, order, versions and 36-start ledger; set wall/disk/free-space bounds and enumerate any additional prerequisite costs before execution | Reviewable inputs and fixed counts; no hidden native preparation or per-profile calibration campaign |
| S3: execute profiles | blocked | Run at most 12 starts per profile; retain every warmup, failure, raw timing and output result | At most 36 actual starts with complete accounting; no automatic extension |
| S4: report and decide | blocked | Show every paired delta, absolute and percentage wall saving, CPU cost, activation, range, correctness and observer cost for every profile | Exploratory signal, no observed benefit, insufficient opportunity, failure or inconclusive result |

Report paired mean and median differences with the individual observations.
Do not turn four pairs into a precision claim, discard inconvenient profiles,
or use the largest observed saving as confirmed value. Charge candidate overhead
to the product and report the isolated research CPU separately.

A promising profile may motivate a separately bounded, independent follow-up
on fresh inputs. Selection evidence is not its confirmation evidence. No
follow-up sample count is automatically allocated here. The original G0 and
lifecycle-value criteria remain in force; the 80 historical validation
transitions are distinct from repeated builds in this diagnostic screen.

## 2026-09-09 execution checkpoint

S1 fails on a 2,104.017374-ms gap in one measured isolation-control request,
against its registered 500-ms bound. All 40 isolation builds pass, but this is
not usable timing qualification. No exclusion, replacement or relaxed limit.

S2 has six consecutive engineering revisions selected: original 15/16 warm up,
17..20 are measured. Static patch preimages/prerequisites match all six. Draft
scripts use the existing full replay CLI and output comparator; they include
candidate inverse costs and exclude the aggregate Checkstyle lifecycle task
from work activation. They remain unexecuted and unqualified. No frozen profile
manifest, subject worktree, Gradle start or metadata helper exists yet.

The durable locator is `.tools/state/buildopt-product-viability-v1/task-state.json`,
key `engineeringPrefix.cpuProfileScreen`; drafts live under
`.tools/state/buildopt-product-viability-v1/bv006-cpu-screen`. Resume through the
[focused sampler diagnosis](../../benchmarks/results/buildopt-product-viability-v1/bv006-cpu-isolation/next-step.md).
The 36-start ceiling and all original G0/lifecycle criteria are preserved.

The [2026-09-09 observer-only reproduction](../../benchmarks/results/buildopt-product-viability-v1/bv006-observer-pause/README.md)
completed with zero Gradle/JVM starts. It validates phase-delay diagnostics but
does not reproduce or explain the original pause; S1 remains blocked. Resume
through the [actual-sampler observation plan](../../benchmarks/results/buildopt-product-viability-v1/bv006-observer-pause/next-step.md).
The screen still has no active allocation and zero starts.

The [live sampler diagnostic](../../benchmarks/results/buildopt-product-viability-v1/bv006-live-observer/README.md)
then verifies nine non-Gradle cases and two successful cold owner builds. It
localizes 323/341 ms synchronous writes but does not explain the old 2,104 ms gap.
The cold daemon endpoint gaps are 504/603 ms, so the original 500 ms criterion is
not passed. S1 remains blocked, with no screen allocation or starts. Continue
with [bounded writer separation and consuming proof](../../benchmarks/results/buildopt-product-viability-v1/bv006-live-observer/next-step.md);
the 36-start screen ceiling and original gates remain unchanged.

The [bounded persistence continuation](../../benchmarks/results/buildopt-product-viability-v1/bv006-buffered-observer/README.md)
verifies R2/R3 with zero Gradle/JVM starts: 15 final cases plus one retained
fixture-reader failure, including delayed writes, saturation, cancellation and
process death. A 2,104-ms controlled write leaves a 100.774-ms sampling gap and
40/40 persisted records. This does not qualify owner endpoint coverage or cost;
S1 remains blocked and the screen stays 0/36. Resume with [actual replay-consumer
integration](../../benchmarks/results/buildopt-product-viability-v1/bv006-buffered-observer/next-step.md),
first without Gradle. Original thresholds, per-profile budgets and held-out
history remain unchanged; no allocation is active.

The [real replay integration](../../benchmarks/results/buildopt-product-viability-v1/bv006-observer-integration/README.md)
now verifies R4.1–R4.3 with 29 final non-Gradle fixture workflows, legacy/C5
compatibility, failure handling and race checks. The actual CLI consumes complete
bounded observation and costs; its release checker rejects altered evidence.
Zero new Gradle/JVM starts. All 81 fixture reservations are charged; five early
launches remain unresolved, with no missing launch receipt in final proof.
S1/G0 and the CPU screen remain blocked at 0/36. Continue with [actual-owner cold
coverage and symmetric cost qualification](../../benchmarks/results/buildopt-product-viability-v1/bv006-observer-integration/next-step.md).
The existing direct-worker overhead fixture does not consume the new observer.
No native allocation is active, and the 12-per-profile / 36-total limits remain.

## Approved walltime decision amendment

The owner authorized all three steps on 2026-09-09. Follow
[W1/W2/W3](./buildopt-walltime-decision-v1.md): close the cold diagnostic within
two starts, execute this unchanged 36-start resource screen, and only follow a
material signal with a separate independent engineering window. Source/output,
process ownership and exact request-walltime requirements remain mandatory.
The original 500-ms diagnostic coverage and formal S1/G0 failures stay failed;
this exploratory admission does not relabel them. The frozen isolated v2 replay
is the W2 measurement consumer; the new v3 observer is used only for W1 here.
All three profile manifests were frozen before native timings. The optional
process snapshots do not supply the start/end clock. No 80-start pilot, automatic
screen expansion or held-out validation transition is allowed.
