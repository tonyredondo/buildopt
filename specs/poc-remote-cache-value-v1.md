# Controlled remote-cache value experiment v1

This protocol closes `POC-REMOTE-CACHE-VALUE-001` with one bounded POC
question: does a prewarmed BuildOpt Edge Cache reduce complete Gradle build
time relative to Gradle's native HTTP client reading the same committed Shared
objects across a modeled remote link?

Both arms use Gradle 9.6.1 `HttpBuildCache`, the same authenticated Shared
origin, the same eight cache entries, disabled native local cache, disabled
Configuration Cache, read-only measured invocations and identical outputs. The
control reads Shared directly through a loopback WAN model fixed at 80 ms per
response and 20 MiB/s. The candidate reads the same objects from a prewarmed
BuildOpt Edge on loopback and must make zero measured upstream requests.

One unmeasured seed and one unmeasured warm-up per arm are excluded. Four
alternating pairs retain every observation. Qualification requires at least
500 ms and 2% mean saving, a positive deterministic paired-bootstrap lower
bound, 4/4 positive pairs, eight `FROM-CACHE` tasks, 32 MiB of byte-identical
required outputs and zero product-attributable failures.

Passing qualifies only Edge locality under this controlled network profile. It
does not prove that Shared storage is faster than another Gradle-compatible
origin, and it is not a production, universal-network or cost claim. Failure
retains native remote cache. No latency, bandwidth, object-size or topology
search is allowed after the terminal result.

Run the frozen experiment and validate a result with:

```bash
./dev/check-poc-remote-cache-value
./dev/run-poc-remote-cache-value /absolute/path/to/new-result.json
./dev/check-poc-remote-cache-value /absolute/path/to/new-result.json
```
