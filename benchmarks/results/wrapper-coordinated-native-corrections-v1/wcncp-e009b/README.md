# WCNCP-009B controlled materiality

The fixed 120-second post-prefetch quiescence interval produced a 1.028603
maximum/minimum preflight ratio under the unchanged 1.15 gate. Six fresh
controlled diagnostic builds then completed successfully on the exact frozen
revisions. No public source was modified and no candidate or paired value build
ran.

GraphQL Java is an `ACTIONABLE_MATERIAL_CORRECTION`: its two configuration
critical-path rows contribute 5,949 ms (40.70%) and 6,356 ms (38.87%). Apache
Groovy is a `NON_MATERIAL_BLOCKER`: the exact `ReleaseInfoGenerator` task class
contributes zero milliseconds to both longest hard-dependency paths. Test Retry
is `UNSUPPORTED_PROBLEM_CLASS`: its configuration contribution is material at
8,843 ms (82.50%) and 8,005 ms (81.87%), but the frozen source closure records
that external License plugin blockers remain after any repository correction.

The first GraphQL build completed before the original analyzer rejected
Gradle 9.6.1's current root-operation owner class. The generic correction keys
only on the unique complete root `Run build` operation. Its explicit recovery
reused the raw trace without repeating the Gradle build, records a null external
elapsed value, and is independently reconstructed. The materiality values come
from the operation trace, not that unavailable outer elapsed value.

Run:

```bash
./dev/check-wcncp-controlled-materiality-v2 \
  "$PWD/benchmarks/results/wrapper-coordinated-native-corrections-v1/wcncp-e009b"
```
