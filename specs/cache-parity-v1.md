# Safe-cache parity

This contract answers whether BuildOpt's default safe local cache adds value
when compared with both no cache and an already warm Gradle native local cache.
The candidate is the public first-run command:

```bash
buildopt gradle --no-daemon --no-configuration-cache clean :app:distZip
```

When no server or local authority is configured, the launcher selects a
cache-only fast path. It keeps the repository-scoped managed L1 and Tier 1
task policy, but does not start the session handshake, local gateway, or
per-project telemetry listener that those absent components would consume.
Any configured or malformed control-plane input restores the fully
instrumented path so observability and fail-closed validation are not bypassed.

## Measurement design

The immutable public Kotlin and Groovy pilots are each measured against two
controls: Gradle with its cache disabled and Gradle with its native local cache
enabled. Every arm uses an isolated archived workspace, Gradle home, and user
cache. One run warms each arm, then four measured pairs alternate execution
order. Project outputs and local Gradle state are removed before every sample;
the Gradle daemon and Configuration Cache are disabled to isolate build-cache
behavior.

Valid evidence requires:

- the expected `compileJava` cache hit in native and BuildOpt arms;
- no cache hit in the cache-off arm;
- no authenticated telemetry handshake in the cache-only candidate;
- byte-identical distribution output in every pair;
- positive mean savings and at least three positive pairs out of four against
  cache-off in both pilot DSLs;
- no more than 2% mean regression against the already warm native cache.

The cache-off threshold tolerates one noisy workstation sample without
accepting a negative mean. Native-cache timings are treated as a parity
guardrail because both arms use Gradle's cache engine; BuildOpt's additional
value in this arm is repository isolation and the Tier 1 safety policy, not a
claim that identical cached bytes can inherently restore faster. The 2% bound
matches the repository's existing long-session no-hit overhead limit and is
stricter than the previous onboarding contract, which recorded native-cache
overhead but did not gate it.

Run and validate a fresh report with:

```bash
./dev/run-cache-parity-benchmark \
  /tmp/cache-parity.json \
  "$(command -v buildopt)" \
  0.3.0-dev \
  /path/to/buildopt-pilot \
  /path/to/buildopt-pilot-groovy
./dev/check-cache-parity-performance /tmp/cache-parity.json
```

This is bounded POC evidence, not a universal claim for every repository or a
production-promotion decision. The raw result retains every signed difference,
runner fact, revision, and binary digest. It does not run the deferred soak.
