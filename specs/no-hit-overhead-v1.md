# No-hit overhead gate v1

This contract closes `A0-G06` with externally timed real Gradle 9.6.1/JDK 21
sessions on the pinned Linux AMD64 4-CPU/16-GiB runner.

## Long sessions

Four measured pairs alternate native Gradle and the complete BuildOpt
launcher/plugin/gateway/server path after one warm-up per arm. Every arm
starts without build outputs. Every wrapper arm also starts with an empty
managed L1, presents a signed read-only authority to the real local gateway,
and receives at least one authenticated `404` from the controlled upstream.
No cache hit or PUT is eligible. The fixture declares no external repository
or dependency; Gradle itself cannot use `--offline` because that flag disables
the remote build-cache client being measured.

The neutral monotonic envelope includes the first measured pair and ends only
after the required reproducible JAR exists. All native and wrapper sessions
must last at least 5 seconds. Nearest-rank p95 must be no more than 500 ms and
no more than 2% of the paired native duration.

## Short sessions

The short stratum takes the RFC's permitted omission branch. Pre-outcome
policy supplies no local Shared authority, so the settings plugin disables L2
before task execution. The real wrapper session must finish below 5 seconds
while the still-running authenticated miss backend observes zero additional
requests. This does not claim a measured 100 ms short-build overhead.

## Evidence boundary

The strict report binds the runner, metric catalog, neutral envelope,
launcher, server, plugin, fixture manifest, and authenticated miss helper by
SHA-256. It is an A0 engineering gate, not beta promotion or causal savings
evidence. The four-pair nearest-rank p95 is conservatively the maximum
observed value; larger beta tail samples remain later work.

Run a host smoke and revalidate the immutable strict report:

```bash
./dev/check-no-hit-overhead
```

Create or remeasure the qualified report:

```bash
./dev/run-golden-lane-container --require-runner-class
```
