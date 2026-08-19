# Ktor cross-commit profile lifetime result

This bounded POC follows one automatically qualified Ktor structural profile
through a real six-commit first-parent sequence. It answers a different
question from the earlier steady-state benchmarks: whether the one-time cost
of learning a fast profile is repaid before ordinary repository evolution
makes that profile inapplicable.

## Result

The profile learned at public commit `24b8773fcee753314dfb753b3f994a7ef36823ef`
reduced the `jvmJar` graph from 128 to 12 projects. Eight alternating
calibration pairs measured 80.529 seconds for optimized native Gradle and
33.530 seconds for BuildOpt, a 46.999-second/**58.36%** mean saving with all
eight pairs positive and a +42.914..+54.206-second 95% saving interval. The
complete discovery and calibration cost was 1,443.324 seconds, projecting
break-even after 31 matching builds.

The observed public history did not provide 31 matching builds:

| Observation | Public commit | Decision | Native | BuildOpt | Delta |
| --- | --- | --- | ---: | ---: | ---: |
| Matching Jetty replay | `eb60b722d1ef` | Central profile selected | 211.042 s | 98.844 s | **+112.198 s** |
| Unrelated CORS source change | `c237e88696de` | Native retained after structural drift | 191.141 s | 411.902 s | **-220.761 s** |
| Global build-logic change | `835d7f9ff09c` | Native retained before structural work | 189.178 s | 189.105 s | **+0.073 s** |

Every observation produced the same required Jetty JAR bytes. The matching
replay used the centrally synchronized profile and the same six committed
Gradle cache objects as its control. The CORS change correctly rejected the
Jetty profile, but the current POC then paid for a full build plus discovery of
the new owner. That safe fallback cost 220.761 seconds more than optimized
native Gradle. The later global change rejected early and remained at native
parity.

Across the three observed builds, matching reuse saved 112.198 seconds while
fallbacks contributed -220.688 seconds. The observed window therefore lost
108.490 seconds before learning cost and **1,551.814 seconds after learning
cost**. No observed break-even exists.

## Decision

The profile mechanism is technically valuable and safe, but this profile was
not economically useful over the observed public history. A projected
steady-state break-even is insufficient. The next generic POC work must:

1. prequalify likely profile lifetime and savings before starting eight-pair
   calibration;
2. avoid full discovery after an unrelated profile rejection unless the new
   owner is itself likely to repay that work; and
3. include rejection/discovery overhead in cumulative customer-visible value.

This does not establish a universal Ktor or Gradle conclusion. It is one
workflow, profile family, public commit sequence and 12-CPU host with a common
eight-worker cap. Percentages are not averaged with other repositories or
mechanisms. Production authorization, soak, design-partner validation and Test
Optimization remain outside the POC.

The machine-readable source of truth is [`summary.json`](./summary.json).
Validate every commit, classification, timing, output digest and economic
calculation with:

```bash
./dev/check-profile-lifetime \
  benchmarks/results/poc-profile-lifetime-v1/summary.json
```
