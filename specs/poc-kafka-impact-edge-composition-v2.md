# Kafka Build Impact and Edge composition rerun protocol

This protocol preregisters a fresh correction of the Apache Kafka 4.3.1
composition experiment. It answers one question: after making Kafka's custom
`shadowJar` byte-reproducible, does repository-authorized Build Impact through
prewarmed Edge still beat optimized native Gradle through the same Shared
origin?

The source correction is not selected after observing performance. Evidence
`E-253` already qualified exactly two temporary source settings:
`reproducibleFileOrder = true` and `preserveFileTimestamps = false`. The runner
checks the original build-file digest, performs those two replacements before
dependency preparation, checks the normalized digest, and derives seed,
control, and candidate from that single normalized tree. Upstream Kafka is not
modified.

The control remains full root `assemble` through the native HTTP build-cache
client pointed at Shared. The candidate remains the installed BuildOpt path,
selecting the three-project client packaging scope through a prewarmed Edge
backed by the same Shared origin. No task adapter, Safe Cache, Runtime Tuning,
Hot State, or Test Optimization is enabled.

Preparation, native seeding, cache admission, Edge warming, and one warm-up per
arm are unmeasured. Four new alternating pairs must save at least 500 ms and
2% on average, all four pairs must be positive, and the paired bootstrap lower
bound must be positive. Every measured build must restore the same normalized
`:clients:shadowJar` bytes. A global change must select the full graph. A
loopback HTTP 503 from Edge must disable the remote cache, finish locally, and
reproduce the seed bytes.

No observation from the failed v1 composition may be reused. The reported
percentage is the single end-to-end composition effect; component percentages
are not added. Qualification applies only to this POC workload and modeled
network profile.

```bash
./dev/check-poc-kafka-impact-edge-composition-v2
./dev/run-poc-kafka-impact-edge-composition-v2 \
  /absolute/path/to/result.json \
  /absolute/path/to/buildopt \
  /absolute/path/to/kafka-source.tar.gz
./dev/check-poc-kafka-impact-edge-composition-v2-result \
  /absolute/path/to/result.json
```
