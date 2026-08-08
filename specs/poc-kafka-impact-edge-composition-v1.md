# Kafka Build Impact and Edge composition protocol

This protocol preregisters one corrected end-to-end POC comparison on Apache
Kafka 4.3.1. It composes only mechanisms independently qualified on Kafka:
repository-authorized Build Impact and prewarmed Edge locality. The BuildOpt
standard-`Jar` adapter is explicitly absent because the previous seed proved
that custom `:clients:shadowJar`, not skipped `:clients:jar`, produces the
required client artifact.

The control is optimized native Gradle running the full root `assemble` graph
through the native HTTP build-cache client pointed directly at Shared. The
candidate is the installed BuildOpt binary selecting the three-project client
packaging scope and pointing the same Gradle client at a prewarmed Edge backed
by the same Shared origin. Both arms use the same fixed source mutation,
dependency state, committed cache objects, JDK, Gradle version, output scope,
12-worker setting and modeled network profile.

An unmeasured optimized-native seed populates one temporary remote cache. Its
objects are committed unchanged to Shared and warmed unchanged into Edge. The
seed binds the required shaded client-JAR digest for this run. Both measured
arms must restore `:clients:shadowJar` from the same committed object and
produce that exact digest; the candidate must not activate any BuildOpt task
adapter.

One unmeasured warm-up per arm precedes four alternating pairs. The composition
qualifies only if it saves at least 500 ms and 2% on average, all four pairs are
positive, the paired lower bound is positive, outputs remain exact, the
candidate selects the preregistered Build Impact plan, measured Edge reads make
zero Shared requests, and no product-attributable failure occurs.

Two unmeasured safety checks remain mandatory: a global change restores the
native full graph, and an unavailable Edge preserves a successful build and
the required output. Percentages from the prior Build Impact and Edge
experiments are not added. This is evidence for one POC profile, not production
or universal-network authorization.

```bash
./dev/check-poc-kafka-impact-edge-composition-v1
./dev/run-poc-kafka-impact-edge-composition-v1 \
  /absolute/path/to/result.json \
  /absolute/path/to/buildopt \
  /absolute/path/to/kafka-source.tar.gz
./dev/check-poc-kafka-impact-edge-composition-v1-result \
  /absolute/path/to/result.json
```
