# Producer-Bound Lifetime Generalization Evidence

This experiment applied the unchanged producer-atomic quarantine and
cross-commit lifetime protocol to Micronaut Core. It is negative but useful POC
evidence: the isolated structural candidate was fast, while exact output
portability changed across revisions and correctly prevented a replay claim.

## Result

- The public qualification reduced the graph from 70 projects to 2 and saved
  **4,092.375 ms / 10.33%** on average, with **7/8** positive pairs, a
  **+2,095.75..+5,833 ms** saved-time interval and a lower candidate p95
  (**39,300 ms** versus **41,231 ms**).
- Two target-revision native observations found one volatile
  `:micronaut-http-server-netty:jar` output. Producer-atomic quarantine kept 193
  other outputs transportable and rebuilt that producer locally.
- The first preregistered descendant did not select the profile. Its broad
  source change failed checkpoint and ownership bindings, so BuildOpt retained
  optimized native Gradle before execution.
- Even with both arms running native Gradle, one different JAR appeared:
  `micronaut-jackson-databind-5.0.0-SNAPSHOT.jar`. The original quarantined
  HTTP-server JAR was identical in both arms.
- Extracted JAR comparison isolated two generated bean-definition classes.
  Their only decoded bytecode difference reverses the two operands passed to
  `Set.of`. This is revision-dependent native build volatility, not a BuildOpt
  selective-execution or cache corruption.
- Exact hashes were not normalized or ignored. The experiment stopped before
  timing the two remaining descendants, records no selected replay, and ends
  **-11,676 ms net** after qualification and publication cost.

## Decision

The Spring cross-commit result does not yet generalize to Micronaut. Static
producer quarantine learned at one revision is insufficient when a different
producer becomes volatile later. The next bounded hypothesis is a
cross-revision volatility portfolio: learn producer volatility only from
authoritative native observations, then evaluate value on a later
preregistered descendant. The observation that discovers new volatility must
not itself become a performance claim.

This result does not weaken exact-output checks, introduce repository-name
rules, average repository percentages, authorize production use, require soak
or design-partner work, or include Test Optimization.

Validate the checked-in evidence with:

```bash
./dev/check-producer-bound-lifetime-generalization
```
