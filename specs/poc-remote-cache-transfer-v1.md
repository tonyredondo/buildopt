# Real-repository remote-cache transfer experiment v1

This protocol closes `POC-REMOTE-CACHE-TRANSFER-001` by transferring the
already qualified Edge-locality mechanism, unchanged, from the synthetic
remote-cache fixture to Apache Kafka 4.3.1. It asks whether a prewarmed
BuildOpt Edge reduces complete `:clients:testClasses` time relative to Gradle's
native HTTP build-cache client reading the same committed Shared objects.

The network profile was derived before any Shared/Edge timing. Three
sequential HTTPS downloads of Kafka's fixed source archive produced the same
SHA-256 and byte count. Their median time-to-first-byte becomes the modeled
per-response latency and their median effective download rate becomes the
modeled bandwidth. The resulting profile is 337 ms and 6,994,831 bytes/s. No
post-result latency, bandwidth, object or topology search is allowed.

Both arms use the same Kafka revision, JDK 25, Gradle 9.2.1 Wrapper, native
`HttpBuildCache`, authenticated Shared origin, committed cache-entry bytes,
offline dependency state and required outputs. Native local cache and
Configuration Cache are disabled. The control traverses the frozen modeled
link; the candidate reads from a prewarmed loopback Edge and must make zero
measured upstream requests. Seeding and one warm-up per arm are excluded.

Four alternating pairs retain every observation. Qualification keeps the
existing gate: at least 500 ms and 2% mean saving, a positive deterministic
paired-bootstrap lower bound, 4/4 positive pairs, byte-identical non-empty
outputs, identical task outcomes and zero product-attributable failures.

Passing qualifies Edge locality only for this Kafka workload and independently
derived network profile. Failure retains Gradle's native remote-cache path.
Neither outcome is a production, universal-network, Shared-storage or cost
claim.

Run the frozen experiment and validate a result with:

```bash
./dev/check-poc-remote-cache-transfer-v1
./dev/run-poc-remote-cache-transfer-v1 \
  /absolute/path/to/new-result.json \
  /absolute/path/to/kafka-26b251a451ce941d3d7a55e6487bcb7f16b5ad48.tar.gz
./dev/check-poc-remote-cache-transfer-v1-result \
  /absolute/path/to/new-result.json
```
