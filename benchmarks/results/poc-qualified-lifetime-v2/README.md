# Qualified Profile Lifetime V2

This bundle is the terminal result for `POC-QUALIFIED-LIFETIME-V2-001`. It
tests whether the five profiles that looked valuable during isolated
calibration continue to create customer value on later public first-parent
commits. It uses one exact installed BuildOpt executable across all subjects;
the three source revisions in the summary are evidence-harness corrections,
not different product binaries.

## Result

| Repository / workflow | Graph | Current calibration | Qualification | Portability | Later builds | Lifetime result |
| --- | ---: | ---: | ---: | --- | ---: | --- |
| Spring Framework `testClasses` | 27 -> 10 | **18.98% faster**, 8/8 | Qualified | Rejected: 2 AspectJ class files differ | 0 | **-7.592 s net**; non-portable. |
| OpenTelemetry Spring family | 1,008 -> 34 | **11.88% faster**, 8/8 | Qualified | Portable: 269 exact outputs | 1 | 0 selections, 1 native fallback; **-13.255 s net**. |
| Apache Kafka `testClasses` | 66 -> 3 | **18.02% faster**, 8/8 | Qualified | Portable: 4,440 exact outputs | 6 | 0 selections, 6 native fallbacks; **-39.961 s net**. |
| Micronaut Core `assemble` | 70 -> 21 | **13.67% faster**, 8/8 | Qualified | Rejected: 1 JAR differs | 0 | **-15.457 s net**; non-portable. |
| Apache Groovy `classes` | 37 -> 2 | 6.82% faster, 6/8; p95 regressed | Native retained | Not evaluated | 0 | **-1.835 s net**; current value not proven. |

The aggregate decision is **4/5 qualified, 2/4 portable, 0/7 selected
replays and 0/5 paid back**. All seven observed descendant builds retained
optimized native Gradle, produced exact required outputs and reported zero
product-attributable failures. Repository percentages are not averaged and
mechanism percentages are not added.

## What changed after calibration

- Groovy's prior large isolated result did not reproduce under the current
  preregistered qualification: only 6/8 pairs improved and candidate p95 was
  worse.
- Spring's two AspectJ-generated class files differ across independent native
  roots. BuildOpt rejects them rather than normalizing or moving unsafe bytes.
- Micronaut's `micronaut-jackson-databind-5.0.0-SNAPSHOT.jar` differs across
  independent native roots and is rejected for the same reason.
- OpenTelemetry remains portable, but its only descendant changes the Wrapper;
  the profile is inapplicable and native Gradle is retained.
- Kafka remains portable and keeps the same Wrapper across six descendants,
  but none has enough exact ordinary evidence for the qualified profile to be
  selected. The six native comparisons total **-33.163 s** before the
  **6.798-s** qualification/publication cost.

This is the key POC finding: a fast calibration is necessary but not sufficient.
Customer value requires a portable output boundary and enough structurally
compatible later builds to reuse the evidence before it becomes stale.

## Next experiment

The next block must improve cross-commit eligibility generically, not rerun
the same measurements until they win. It should:

1. distinguish exact structural compatibility from per-revision ordinary
   evidence that is still pending;
2. avoid synchronization/materialization work when a profile cannot yet be
   selected;
3. derive the smallest Gradle-owned portable output boundary without dropping,
   rewriting or semantically normalizing required customer outputs; and
4. rerun these frozen public windows only after that generic change, preserving
   the same correctness, fallback and economic gates.

## Recompute

```bash
./dev/check-qualified-lifetime-v2 \
  benchmarks/results/poc-qualified-lifetime-v2/summary.json
```

Each subject directory retains its compact result, the complete 17-invocation
qualification capture and all eight raw calibration pairs. Large repositories,
payload packs, Gradle caches and logs are intentionally excluded from Git.

The protocol is defined in
[`poc-qualified-lifetime-v2.md`](../../../specs/poc-qualified-lifetime-v2.md).
This is bounded 12-CPU POC evidence, not a soak, production gate or universal
Gradle claim. Test Optimization remains out of scope.
