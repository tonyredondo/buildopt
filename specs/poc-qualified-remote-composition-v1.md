# Qualified remote composition protocol

This protocol preregisters one end-to-end POC comparison on Apache Kafka 4.3.1.
It composes only mechanisms that already cleared an independent Kafka gate:
repository-authorized Build Impact, the exact standard `Jar` adapter, and
prewarmed Edge locality. It does not add their previous percentages.

The control is optimized native Gradle running `assemble` with daemon,
parallel execution, 12 workers and the native HTTP build-cache client pointed
directly at Shared. The candidate is the installed BuildOpt binary selecting
`:clients:jar`, enabling only the exact standard `Jar` adapter, and pointing the
same Gradle HTTP client at a prewarmed Edge backed by the same Shared origin.

Both arms use the same Kafka revision, source mutation, dependency state,
committed cache objects, JDK, Gradle version, required client JAR and modeled
network profile. Local and Configuration caches are disabled so the measured
remote path is observable. External dependency network is blocked after
preparation while loopback Shared and Edge remain reachable.

One unmeasured warm-up per arm precedes four alternating pairs. The composition
qualifies only if it saves at least 500 ms and 2% on average, all four pairs are
positive, the paired lower bound is positive, the client JAR is exact, the
candidate selects the preregistered plan and restores its `Jar` from cache, and
no product-attributable failure occurs.

Two unmeasured safety checks remain mandatory: a global change restores the
native full graph and an unavailable Edge preserves a successful build and the
required output. This is POC evidence for one Kafka profile, not production or
universal-network authorization.

Before Shared commit or timing, the seed must also prove that the required
client JAR is produced by the exact standard `Jar` task assumed by the
composition. A skipped `:clients:jar`, custom producer, or non-reproducing
historical digest terminates the protocol as an invalid component precondition;
it cannot be repaired by accepting new output bytes or dropping the adapter.

```bash
./dev/check-poc-qualified-remote-composition-v1
./dev/run-poc-qualified-remote-composition-v1 \
  /absolute/path/to/result.json \
  /absolute/path/to/buildopt \
  /absolute/path/to/kafka-source.tar.gz
./dev/check-poc-qualified-remote-composition-v1-result \
  /absolute/path/to/result.json
```
