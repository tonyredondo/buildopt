# Research CPU isolation: observation quality rejected

Date: 2026-09-09. Program: `BUILDOPT-VIABILITY-V1`.
Decision: **`FAILED_OBSERVATION_QUALITY`**. Native execution completed;
observation qualification failed. BV-006 remains partial and the CPU profile
screen is blocked with **zero starts**. No timing or product-value pass.

All **40 allocated Gradle starts succeeded**: eight fixture builds and 32
Elasticsearch control builds. The same prototype executable kept native CPUs
0-7/eight workers in both versions, with the observer sharing 0-7 in N and
isolated on CPU 8 in I. Each version retained one daemon. Source, native argv,
actual supervisor/daemon thread masks, measured task outcomes and owned process
closure pass. Production source and host configuration are unchanged.

The frozen checker accepts 31 request records and rejects request 28, N at
cycle 13, for a **2,104.017374 ms** gap against the registered **500 ms** maximum.
Its observed daemon span covers 98.348% of the request, which does not excuse
that gap. This was a measured row and remains included in the fixed allocation.
The checker and threshold are unchanged; no replacement, discarded row,
additional owner build, bootstrap timing verdict or readiness claim follows.
See the [complete rejection record](./analysis/owner-rejection.json) and
[limited inspection after rejection](./analysis/rejected-row-remaining-checks.json).

The missing interval lies between 11:29:07.339697 and 11:29:09.440688 UTC.
The adjacent scans took 3.042 and 2.732 ms. There is no recorded process-exit
snapshot error. The [gap diagnosis](./analysis/snapshot-gap-diagnosis.json)
locates the pause between measured scan bodies; existing stamps cannot separate
JSON encoding, file writing, runtime scheduling or host delay there. This is
an unresolved cause, not proof that affinity isolation failed or improved builds.

The [isolation contract](./analysis/contract.md) and
[fixture qualification](./analysis/qualification.json) remain useful. Actual
launch, restoration, failure, cancellation and driver-death behavior pass;
unsafe restoration kills/reaps the child and discards its locked OS thread.
The prototype preserves complete disk polling and independent liveness guards.
I1-I4 and the rejection diagnosis are verified; I5 observation qualification
is blocked. The control contains no Checkstyle candidate, so it cannot estimate
optimizer saving even if native execution is successful.

Costs: **40 reservations / 40 actual Gradle starts**, zero standalone helper
JVMs or new nested Gradle starts. Non-Gradle fixture reservations: 110; actual
children: 107 (104 replay requests plus three direct affinity children).
The initial compile and fixture working-directory setup failures started no
native work; their evidence and unused reservations are retained. There was
no owner retry. Program totals are **644 reservations / 545 actual Gradle
starts**, including the historical 29 nested starts. The older one possible
metadata JVM remains unresolved. [Closeout](./closeout.json),
[independent audit](./independent-audit.json) and
[external seal](../bv006-cpu-isolation-seal-audit.json) preserve the evidence.

The [short profile screen](../../../../docs/plans/buildopt-product-viability-v1-cpu-screen.md)
has six engineering revisions selected and task-local execution/analysis drafts,
but no allocation, profile worktree, Gradle start or helper JVM. Its 36-start
ceiling is unchanged. All 80 validation transitions remain untouched.
[Next: diagnose the unobserved interval without repeating this control](./next-step.md).
