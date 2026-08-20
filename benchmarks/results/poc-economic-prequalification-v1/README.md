# Economic prequalification result

This retained POC result follows the same public Ktor `jvmJar` history used by
the profile-lifetime experiment and tests whether BuildOpt can decline a new,
uneconomic learning attempt before structural discovery or eight-pair
calibration.

The run used installed BuildOpt revision
`917e456fcdc5bf97f887093de3e4b4d587f0dede`, Ktor revision
`24b8773fcee753314dfb753b3f994a7ef36823ef`, Gradle 9.2.1, Java 21 and the
12-CPU Linux POC host. Control and candidate arms received the same immutable
Gradle dependency and native build-cache seeds. Every required JAR matched
byte for byte.

## Results

| Event | Optimized native | BuildOpt | Direct result |
| --- | ---: | ---: | ---: |
| Jetty qualification mean | 77.419 s | 32.489 s | **44.930 s / 58.03% saved**, 8/8 positive pairs |
| Matching Jetty replay | 197.028 s | 96.284 s | **100.744 s / 51.13% saved** |
| Unrelated CORS change | 184.647 s | 198.543 s | **13.896 s / 7.53% overhead**, native retained |
| Global build-logic change | 186.553 s | 186.531 s | **22 ms parity**, native retained |

The CORS precheck examined at most 64 first-parent commits, found two analogous
direct-owner changes against a theoretical minimum of eight matching builds,
and returned `REJECT` in 192.442 ms. It performed no structural discovery and
no calibration. Direct owner boundaries were used instead of transitive task
inputs, so the CORS project remained unambiguous even though consumers also
depend on it.

The preceding lifetime experiment paid for discovery after rejecting the same
Jetty profile and observed a 220.761-second CORS penalty. The new run observes
13.896 seconds on the same public change: the measured fallback penalty is
206.865 seconds smaller, or 93.71% lower. This is a cross-run before/after
comparison, not an additional paired speedup claim; the precheck itself took
only 192.442 ms and the remaining full-build difference is treated as runner
variation inside the 10% native-retention guardrail.

## Economic conclusion

Qualification still cost 1,386.764 seconds and projects a 31-build break-even.
One matching replay saved 100.744 seconds; the two native-retained cases lost
13.874 seconds in aggregate. The three-build window therefore records 86.870
seconds of gross value and **-1,299.894 seconds net after learning**. The
profile is not paid back in the observed window.

Economic prequalification solves one bounded problem: it avoids making an
inapplicable build dramatically worse by learning a low-recurrence owner. It
does not reduce the original Jetty qualification cost or prove that future
commits will supply 31 matching replays.

## Validation

```bash
./dev/check-economic-prequalification \
  benchmarks/results/poc-economic-prequalification-v1/summary.json
```

The checker binds the public history, eight-pair qualification, exact outputs,
matching selection, CORS rejection, absence of discovery/calibration, global
fallback and cumulative arithmetic. This is POC evidence, not production
authority, a soak, a design-partner result or Test Optimization work.
