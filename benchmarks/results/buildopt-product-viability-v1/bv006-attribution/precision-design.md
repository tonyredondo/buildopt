# Prospective owner precision design

Registered before implementation and new native starts, 2026-09-09.

The complete fixed attribution study retained all 20 successful requests.
Root collector CPU medians are 541.334 ms for N and 552.130 ms for I;
root callback wall sums are 1009.580 and 1087.414 ms. Wall callbacks can
overlap across threads and include native provider realization. These are
observed diagnostic occupancies, not additive native delays or product savings.
Graph/event coverage is complete in all four build contexts. Native paired
differences range from -22.697 to +6.052 s; the inherited arm imbalance is
2011.844440 ms. The earlier uninstrumented v5 result remains a failed
588.682498-ms qualification. No collector phase identifies a supported
arm-specific correction. JVM compilation and GC vary, without causal proof.

Keep production collector/worker/runtime/C5 bytes unchanged. Use one two-stage
measurement design, with an independent variance pilot and fresh confirmation.

| Step | Fixed design | Required outcome |
|---|---|---|
| Qualify test changes | Preserve default 20-request fixture; test sizing boundaries and interval rejection; rebuild frozen owner test | Exact ordering/counts, failures and original 10/100-ms gates remain enforced |
| Variance pilot | Four balanced warmup cycles, eight requests per arm; sixteen measured pairs per arm; exactly 80 Gradle starts | Full native/source/output evidence; no possible owner qualification |
| Select sample size | All 16 differences per arm, using within-arm variance only; formula below | Freeze one count of 64..512 pairs per arm, or stop as insufficient precision |
| Confirmation | Fresh independent N/I states; same eight warmups per arm; fixed selected count; at most 2064 starts | Original wrapper p95 <=10 ms and absolute median overhead difference <=100 ms, plus both confidence interval gates |
| Downstream | Only after all confirmation proof passes | Engineering ordinals 0..20 remain conditional; validation 21..100 untouched |

For paired differences in nanoseconds, select
`ceil8(max(64, (pi/2)*2*(var(N)+var(I))*(1.96+1.645)^2/100000000^2))`.
Sample variance uses n-1. More than 512 pairs per arm closes this attempt as
`INSUFFICIENT_PRECISION_NO_CONFIRMATION`; it does not start a smaller lucky trial.
The factor two is a declared pilot-variance allowance. The normal/median
approximation is a sizing heuristic, not a guarantee under heavy tails or
serial dependence. Eight warmups per arm are a fixed preparation choice
informed by the observed early-to-late phase decline, not proof that JIT or
host variability has vanished.

For confirmation, independently reconstruct the difference of the two lower
nearest-rank medians. Use 20,000 seeded circular moving-block resamples of the
paired N/I vectors for each block size 4 and 8. Both 95% percentile intervals
must lie entirely inside [-100,+100] ms. These additional checks cannot relax
the existing point or wrapper gates. All samples and both failed prior
attempts remain visible. There is one pilot and at most one confirmation;
no timing exclusion, optional success, retries, or confirmation resizing.

The local allocation is prospectively extended under the owner's standing
budget instruction: 2400 phase Gradle reservations and 16 hours total, ending
2026-09-09 20:36:54 UTC. Child native requests remain bounded at 900 seconds,
free disk at 40 GiB, and new phase artifacts at 120 GiB. This changes no paid
service, model setting, system configuration, or previously sealed budget.

Exact machine-readable registration and allocation are retained in the
host-local `bv006-attribution/inputs/precision-design-v1.json` and task state.
