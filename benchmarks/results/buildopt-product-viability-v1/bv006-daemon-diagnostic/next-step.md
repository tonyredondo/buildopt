# Next BV-006 step: compatible disk observation and a fixed control

Status: **pending**. This is a bounded next deliverable, not a completed
correction or native-run registration. The [diagnostic](./README.md) establishes
scanner CPU cost; native latency impact and earlier variability remain unresolved.

Hypothesis: removing repeated full-tree counting from the 100-ms process/liveness
loop reduces supervisor interference while preserving the runner's disk,
ownership, failure and lifecycle contracts. Do not disable budget enforcement
or credit harness improvements to C5 product value.

| Step | State | Work and proof | Required outcome |
|---|---|---|---|
| D1: recover and freeze | pending | Verify this seal, source pins, current task identity and closed units; read `diskGuardUntil`, `treeBytesUntil`, their callers and existing disk/lifecycle tests. Record the actual size/free-space enforcement and cancellation contract, including response bounds | An explicit contract and exact before-source target; no reliance on recollection |
| D2: smallest compatible correction | pending | Separate expensive counting from frequent ownership/liveness checks; preserve cheap live free-space checks and exact size limits. Prove any accounting or scheduling change against every existing failure/cancellation consumer | A reviewable implementation with equivalent limits, or a documented inability to make a compatible correction |
| D3: affected behavior proof | pending | Exercise size growth across the limit, low free space, traversal errors, cancellation during a long scan, child exit, driver death and owned-process closure. Test the consuming loop and the actual native output/lifecycle path | All affected unit/race/fixture proofs pass with exact source and command receipts; no silent budget fallback |
| D4: register one control | pending | Freeze the corrected and old supervisor versions before owner timings; use the same frozen baseline without C5 in both variants, workflow, JDKs, affinity and independent run state. Fix eight warmups plus eight measured requests per variant, alternating order, and all quality/statistical criteria | One reviewable old-versus-new design with 32 owner requests; no row exclusions, repeat-until-green or reuse of current N/I samples |
| D5: execute and reconstruct | pending | Run the single fixed control; retain all native exits, task/outcome coverage, disk/failure receipts, native wall, supervisor CPU and cost. No new JFR profiling is needed for this comparison | Compatible observer behavior plus measured CPU/latency effects and uncertainty, or an explicit negative/inconclusive result |
| D6: owner timing readiness | pending | Only after compatible observation is established, prospectively register fresh C5 N/I plain/capture qualification with the original 10-ms wrapper and 100-ms imbalance limits and interval/sizing rules | G0 instrumentation readiness passes or remains insufficient; this requires its own later allocation |
| D7: chronological product work | blocked | After full G0 prerequisites pass, run C5 engineering ordinals 0..20 and freeze untouched confirmation inputs | Resume lifecycle value research; no adaptive or commercial viability claim from a harness correction |

Proposed local ceiling for D1-D5: three hours, at most eight fixture Gradle
starts plus 32 owner starts (40 total), zero standalone JFR/helper JVM starts,
60 GiB new state and at least 40 GiB free. Reserve exact costs before starting;
failed or nested starts consume the allocation. Build/test helper commands must
be accounted explicitly if the selected proof requires any. Do not silently
expand this prospective ceiling. One fixed control follows green prerequisites;
a failure or exhausted bound closes the attempt with evidence.

There is no need to weaken a disk guarantee to make a timing experiment faster.
Moving exact counting outside the native interval is acceptable only if its
contract is preserved; otherwise a bounded accounting design needs its own
proof, or the proposed correction is rejected. No new host permissions,
services, kernel features, heap/worker changes or plugin changes belong here.

Preserve the initial sampler failure, all export/checker failures, the prior
20-request readiness failure and the variance pilot. Do not accumulate or trim
those rows into a new pass. G2 remains verified for its exact owner scope;
validation 21..100, H3 and cross-repository value tests remain untouched.
