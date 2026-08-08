# Installed Kafka profile value protocol

This protocol preregisters the first value measurement of the exact
repository-owned Apache Kafka profile through the packaged `buildopt poc`
command. It answers one question: does the user-facing v2 profile retain a
material advantage over optimized native Gradle plus Shared Cache when every
launcher, profile-validation and Edge-configuration cost is included?

The workload remains Apache Kafka 4.3.1 at the fixed revision and normalized
`shadowJar` source input already qualified by the composition experiment. The
control runs full root `assemble` through Shared. The candidate installs the
current Linux package, consumes the committed v2 profile, and runs:

```text
buildopt poc --changes-file .buildopt-changes \
  --edge-url http://127.0.0.1:<PORT>
```
The prior 82.35% composition result is a precondition, not a reusable timing
sample. Preparation, package installation, native seeding, Edge warming and
one warm-up per arm are unmeasured. Eight fresh alternating pairs must save at
least 500 ms and 2% on average, all eight pairs must be positive, and the
paired bootstrap lower bound must be positive. Every arm must restore the
exact normalized `:clients:shadowJar` bytes.

A global change must select native full `assemble` without Edge. A loopback
HTTP 503 must disable remote cache, complete the selected graph locally and
reproduce the same output. No repository-specific product rule, threshold
change, discarded pair, component-percentage addition, production claim, soak,
design-partner work or Test Optimization change is permitted.

Three unmeasured warm-up failures corrected the harness and profile before any
timing was accepted: experiment-only Basic authentication was removed from the
public loopback path, the selected Edge cache was moved after repository
settings evaluation, and `--offline` was removed because Gradle disables HTTP
build caches in offline mode. Dependency preparation and the blocked external
network boundary remain unchanged. Evidence `E-259` through `E-261` records
these zero-observation corrections; the pair order and value gate did not move.

```bash
./dev/check-poc-kafka-installed-profile-value-v1
./dev/run-poc-kafka-installed-profile-value-v1 \
  /absolute/path/to/result.json \
  /absolute/path/to/installed/buildopt \
  /absolute/path/to/kafka-source.tar.gz
./dev/check-poc-kafka-installed-profile-value-v1-result \
  /absolute/path/to/result.json
```
