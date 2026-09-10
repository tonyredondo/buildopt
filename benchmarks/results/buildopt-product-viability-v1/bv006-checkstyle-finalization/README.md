# Checkstyle supported finalization result

Completed 2026-09-10. Decision: `FINALIZATION_VALUE_NOT_ESTABLISHED`. Product viability: **NOT_ESTABLISHED**.

The supported finalizer does not meet the predeclared whole-request saving criterion. Its safety and integration are verified, but a performance benefit sufficient for adoption is not established.

Whole customer-request time changes from **103.770s to 90.250s** on average: **+13.520s saved (+13.03%)**. Both arms already contain the optimization: N is the current implementation, I is the supported shared-service implementation. These measurements are not a new comparison with native Elasticsearch.

| Replication | Current request s | Supported request s | Saving s | Saving % | Host flag N / I |
| --- | --- | --- | --- | --- | --- |
| 1 | 124.537 | 83.160 | +41.377 | +33.22% | FLAGGED / FLAGGED |
| 2 | 83.003 | 97.339 | -14.337 | -17.27% | FLAGGED / FLAGGED |

The criterion was fixed before the run: each measured pair must save at least one second, with pooled saving at least five percent and complete correctness/phase evidence. There are only two selected measured pairs, with balanced order. Keep every result; no slow sample was discarded and no timing-driven rerun was added. CPU affinity restricts the experiment to eight CPUs but does not isolate the workstation. Host PSI includes the experiment itself, and external activity is inferred rather than attributed to another agent or a named tool. [Full host observations](host-summary.json) include sampling and ownership qualifications.

## What changed and what passed

History publication now uses Gradle's public shared BuildService closure. Preparation remains the only extra task action. Publication requires successful completed native work, matching context/request identity and the exact closed XML report digest. No deprecated execution listener or trailing publication action remains. The implementation is qualified only for pinned Gradle 9.7.1 with the owner's Configuration Cache disabled.

The fresh trial completed **eight successful owner builds**, four cold/warmup builds and four measured builds. All four complete-output pairs passed both the live comparator and independent reconstruction. All three task-local histories and report hashes were verified; processed/reused counts and context/commit phases were captured for every executed Checkstyle task. Full customer time includes the finalizer and outside-envelope customer machine costs; native command time is retained separately in [requests.tsv](requests.tsv).

Six final strict-warning lifecycle fixtures pass: cold/warm reuse, native and late failures, changed/deleted reports, changed context, no-source, actual worker cancellation, recovery and up-to-date. The native formatter, portable patch/inverse, four retained core tests, 25 output-reader cases, and exact replay consumer proofs are retained in [correctness-v2.md](correctness-v2.md).

| Replication | Arm | Task | Processed | Reused | Adapter s | Adapter end to task completion s | Commit work s |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | I | :server:checkstyleTest | 6 | 2819 | 1.725 | 0.063 | 0.029 |
| 1 | I | :server:checkstyleMain | 5 | 4954 | 1.437 | 0.057 | 0.041 |
| 1 | N | :server:checkstyleTest | 6 | 2819 | 2.453 | 55.232 | 0.626 |
| 1 | N | :server:checkstyleMain | 5 | 4954 | 6.332 | 50.836 | 0.056 |
| 2 | I | :server:checkstyleTest | 6 | 2819 | 2.253 | 0.073 | 0.171 |
| 2 | I | :server:checkstyleMain | 5 | 4954 | 1.525 | 0.059 | 0.288 |
| 2 | N | :server:checkstyleTest | 6 | 2819 | 1.827 | 43.550 | 0.033 |
| 2 | N | :server:checkstyleMain | 5 | 4954 | 1.589 | 40.898 | 0.049 |

The task-completion interval and publication interval have different meanings. The service can publish after task completion; a shorter reported task duration alone is not customer saving. Phase deltas combine sampled clock alignment and millisecond task timestamps. [All phases](task-phases.tsv) retain both intervals.

## Why the timing result remains mixed

The actual checking work is equivalent: Main processes five files and reuses 4,954; Test processes six and reuses 2,819. InternalClusterTest is up-to-date. Successful reuse is present in both implementations, including the slower supported request. A Checkstyle history miss does not explain that regression.

The first measured request in each replication takes about 83s, although the implementation order reverses. Both second requests are slower and their private daemons have been idle longer. These observations are descriptive; they do not establish which activity caused the delay.

| Replication / order | Implementation | Customer s | Time since own cold command ended s | Sampled daemon major faults | Sampled daemon reads MiB |
| --- | --- | --- | --- | --- | --- |
| 1 / first | Supported | 83.160 | 334.95 | 0 | 0 |
| 1 / second | Current | 124.537 | 1394.44 | 8,696 | 176.88 |
| 2 / first | Current | 83.003 | 210.26 | 0 | 0 |
| 2 / second | Supported | 97.339 | 652.12 | 963,926 | 3,744.97 |

The slow current request coincides with an external-CPU estimate near 13.2 core equivalents; its 3.314s maximum sampling gap exceeds the registered 3s limit, so precise attribution is unqualified. The slow supported request has little inferred external CPU (about 0.32 cores) but roughly 3.66GiB of sampled daemon reads and many major page faults. This supports a paging/I/O explanation for part of the observed delay, with an order/state concern requiring a control. No swap counters were collected: the specific cache layer, storage path and external process remain unverified. Linux documents these counters in its [proc interface reference](https://www.kernel.org/doc/html/latest/filesystems/proc.html). See [order and idle-age evidence](order-and-idle-age.json).

All eight requests have prospective host-pressure flags; those flags include pressure from the experiment itself. None is removed. The next measurement must compare identical code with itself before assigning a material difference to an optimization.

## Retained failures and cost

The first controller selected the wrong replay consumer and failed before any native command. The V1 task-listener candidate then failed Elasticsearch's warning-as-error policy: current exit0, V1 exit1. Those **two owner starts** are retained outside the eight successful comparison starts. V1's small fixture had suppressed its own deprecated diagnostic listeners; V2 removes them and uses `--warning-mode=fail`. V1 artifacts remain visibly rejected; source and comparator acceptance were not weakened to obtain success.

The phase totals are **10 actual owner starts**, **17 small correctness starts** (12 success, three expected fault exits and two controlled cancellations), **two formatter starts**, **20 standalone JVM helpers**, and **nine standalone compiler commands**. All earlier setup/probe/reader failures remain accounted for. Controller reservations are 24 including abandoned runs; total phase Gradle reservations 43 and observed starts 29. Program totals are **791 charged / 651 observed Gradle starts**, retaining the same 29 older nested starts and predecessor accounting qualifications. Closed units and final disk accounting are in [the closeout audit](proof-v2/analysis/closeout-v2.json); the conservative allocated-byte bound is 42.90GiB, including a 2MiB reserve for final records. No publication or other-repository build occurred.

The first phase-analysis pass classified the plain `:server:checkstyle` aggregate as a Checkstyle worker. Its retained failure was corrected by using the three actual checker tasks from the frozen plugin; full graph/action equality still includes the aggregate. This was an analysis-only correction of the same successful run, with no native rerun or relaxed output contract.

## What this means for viability

The previous disjoint history contains 13 measured transitions but only original 7 executes Checkstyle. It would need 14.453s of net saving across that complete window to meet the existing five-percent/one-second-per-build targets, before extra adoption or maintenance costs. The old 20-transition dependency model places direct critical-path opportunity at 19 and 20, predicts zero direct saving at 7, and leaves only 4.746s above a five-percent target. That is a model, not a measured counterfactual; removing CPU competition can also change other task durations.

Do not substitute this current-versus-supported delta into older native measurements or promote G0/G3/product value. The next step is [a four-build identical-code measurement control](next-step.md). The proposed native 17–20 screen is deferred behind that control. Formal held-out 21–100, the other six repositories, and adaptive-controller implementation remain untouched.

Reviewable source: [supported candidate patch](candidate-v2.patch), [inverse](candidate-v2.inverse.patch), [delta from current](current-to-supported.patch), and [exact owner-runner extension](owner-runner-extension.patch). Machine-readable results: [summary](summary.json), [profile](profile.json), [source bindings](comparison-inputs.json), [evidence index](comparison-evidence.json).

Raw locator: `/home/tonyredondo/repos/github/tonyredondo/buildopt/.tools/state/buildopt-product-viability-v1/bv006-checkstyle-finalization`; current task key `engineeringPrefix.checkstyleFinalization` in the parent `task-state.json`. BuildOpt remains main at `b76ded08c952ebb386576fafce4ae2d8fdcc09f1`; no commit or push was made.
