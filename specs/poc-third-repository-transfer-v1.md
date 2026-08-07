# Third-repository clean-profile transfer

This POC transfers the already qualified clean BuildOpt profile, unchanged, to
Apache Kafka 4.3.1. Kafka is a substantial third public repository and adds a
new workload class: a 64-project Gradle 9.2.1 build combining Java, Scala,
generated protocol sources, shaded packaging, and test preparation.

The control is optimized native Gradle running the unqualified `testClasses`
selector with Build Cache, parallel execution, all 12 host CPUs, one shared
daemon, and offline dependencies. The candidate changes a production source in
the central `clients` module and uses the generic Build Impact graph to select
`:clients:testClasses`. Discovery must remain complete and conservative: the
control reaches 64 projects, the candidate reaches three, neither contains a
Gradle `Test` task, and any build-logic change restores the full selector.

The candidate also enables the previously qualified exact standard-`Jar`
adapter. Its only preregistered producer assertion is the unmodified
`:generator:jar`; a warm candidate creates the native-cache entry, every
measured candidate must restore it `FROM-CACHE`, and native Gradle must leave
the same task non-cacheable. No Hot State, Runtime Tuning, Copy adapter,
Managed L1, Shared Cache, Edge Cache, gateway, or telemetry may start.

The source archive, immutable revision, Gradle distribution and wrapper,
upstream workflow, build inputs, mutation, generated impact graph, outputs,
runner, ordering, four-pair budget, and value gate are frozen before timing.
Every arm starts from clean outputs and the same candidate-augmented native
cache seed. Required `clients` main/test classes and resources are compared
byte for byte. The unmeasured online preflight exists only to resolve
dependencies and prove that the full preparation succeeds; all accepted arms
run offline.

Qualification requires at least 500 ms and 2% mean saving, a positive
deterministic paired-bootstrap lower bound, four positive pairs, identical
non-empty outputs, successful full-graph fallback, and zero
product-attributable failures. No failed or unfavorable pair may be discarded,
and no threshold may move after measurement.

This experiment tests transfer of a POC mechanism, not production readiness.
It neither executes nor selects tests, adds Kafka-specific product logic,
changes upstream source, requires a soak/design partner, nor broadens any
previous workload-specific claim.
