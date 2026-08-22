# Producer-bound quarantine lifetime evidence

This directory contains the bounded public-repository evidence for
`POC-PRODUCER-BOUND-QUARANTINE-REPLAY` after producer-atomic quarantine was
connected to automatic qualification and cross-commit replay.

The run reused one exact BuildOpt executable and a public Spring Framework
first-parent sequence. Qualification changed `spring-jms`, measured eight
alternating control/candidate pairs, compared native outputs across independent
roots, quarantined every output of a volatile producer, and published only the
stable pack. Two later public revisions then exercised selection and native
fallback.

## Result

- Qualification saved **1,361.625 ms / 4.62%** on average with **6/8** positive
  pairs, a **+375.75..+2,435.75 ms** saved-time interval and a non-regressive
  candidate p95 (**29,370 ms** versus **32,523 ms**).
- The explicit automatic POC policy
  `ROBUST_6_OF_8_ALTERNATING_PAIRS_INTERVAL_P95_V1` qualified the profile. It
  does not rewrite the historical 7/8 evidence policy.
- Independent-root portability retained **7,864 exact transported outputs** and
  quarantined two producers containing **352 outputs**.
- The first compatible descendant selected the central profile: optimized
  native Gradle took **168,393 ms**, BuildOpt took **83,737 ms**, and the
  attributable saving was **84,656 ms / 50.27%**.
- The second descendant changed `spring-core`. Its graph could omit only 3 of
  27 projects and had insufficient analogous recent changes, so BuildOpt
  rejected the profile before Gradle and retained native execution.
- Both descendants preserved the same **8,033 stable required outputs** byte
  for byte. All **352 quarantined output paths** were present in both arms and
  were rebuilt locally rather than transported.
- Across the two observed descendants, the window records **63,190 ms gross**
  and **59,550 ms net** after **3,640 ms** of qualification and publication
  cost. The native-retained arm variation remains visible and is not attributed
  as an optimization saving.
- Product-attributable failures: **0**.

This proves one selected non-Kafka cross-commit replay with exact stable bytes
and safe rejection of an incompatible descendant. It does not establish a
repository-wide Spring percentage, authorize production activation, average
repository percentages, add mechanism percentages, require soak or design
partner work, or include Test Optimization.

## Files

- `spring-framework-classes-jms/result.json` is the complete subject result;
- `spring-framework-classes-jms/qualification-capture.json` is the exact
  automatic qualification capture;
- `spring-framework-classes-jms/calibration-evidence.json` contains all eight
  paired calibration observations and the fallback proof.

Validate the frozen evidence with:

```bash
./dev/check-producer-bound-quarantine-lifetime
```
