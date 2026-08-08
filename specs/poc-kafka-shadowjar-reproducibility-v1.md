# Kafka shadow JAR reproducibility protocol

This protocol isolates the safety failure that stopped the Kafka Build Impact
and Edge composition. Apache Kafka 4.3.1 configures every Gradle
`AbstractArchiveTask` with non-reproducible file order and preserved file
timestamps. Its required client artifact is produced by custom
`:clients:shadowJar`, so a successful rebuild after an unavailable remote cache
can be logically equivalent while differing byte for byte from the cached JAR.

The experiment performs two independent clean baseline builds from the pinned
source and mutation. Their archive bytes must differ while their entry payload
fingerprints remain equal. It then changes only the two explicit archive
properties in temporary source copies and performs two more independent clean
builds. Those normalized JARs must have the same SHA-256 and the same logical
payload as the baseline builds.

A fifth normalized build points Gradle at a loopback cache that always returns
HTTP 503. Gradle must disable the remote cache, complete `:clients:shadowJar`
locally, and reproduce the normalized digest. This proves the exact safety
precondition needed before a new composition experiment; it does not measure
performance, modify Kafka upstream, or authorize production rollout.

```bash
./dev/check-poc-kafka-shadowjar-reproducibility-v1
./dev/run-poc-kafka-shadowjar-reproducibility-v1 \
  /absolute/path/to/result.json \
  /absolute/path/to/kafka-source.tar.gz
./dev/check-poc-kafka-shadowjar-reproducibility-v1-result \
  /absolute/path/to/result.json
```
