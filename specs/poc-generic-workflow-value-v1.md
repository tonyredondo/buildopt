# Public Workflow-Family Value Protocol

## Decision

Measure the unchanged generic structural Build Impact mechanism on four
substantial public, build-owned Gradle workflow families. Each family is an
independent experiment against optimized native Gradle; a win in one family
cannot authorize another.

The machine-readable preregistration is
[`poc-generic-workflow-value-v1.json`](./poc-generic-workflow-value-v1.json).

## Frozen subjects

| Family | Public workflow | Exact change | Required output | Candidate |
| --- | --- | --- | --- | --- |
| JAR packaging | Apache Groovy `jar` | `groovy-json` production source | `groovy-json` JAR | `:groovy-json:jar` |
| Typed verification | Apache Kafka `checkstyleMain` | `clients` production source | client Checkstyle HTML and XML reports | `:clients:checkstyleMain` |
| Distribution | Apache Kafka `shadowJar` | `clients` production source | client fat JAR | `:clients:shadowJar` |
| Test preparation | Spring Framework `testClasses` | `spring-jms` production source | `spring-jms` classes | `:spring-jms:testClasses` |

The distribution row uses Kafka's public `shadowJar` workflow because it is
an `AbstractArchiveTask` that creates a combined runtime artifact and has two
independent producers in the frozen repository. It therefore tests a real
full-workflow-to-affected-producer reduction rather than renaming the same
single distribution task.

## Measurement contract

- Package and install the current BuildOpt revision before proposal or timing.
- Shallow-fetch the exact public revision and apply one deterministic source
  mutation.
- Run `profile propose -> profile measure -> profile evaluate` without any
  repository-name rule in the product.
- Compare eight isolated alternating pairs with independent Gradle homes and
  independently restored native-cache seeds.
- Give both arms the same daemon, Build Cache, parallelism, Configuration
  Cache, console, scan, and worker settings.
- Include installed launcher and profile-selection overhead in the candidate.
- Require byte-identical declared outputs in every pair and a successful
  optimized-native full-graph fallback.
- Retain native Gradle for missing semantics, output drift, incomplete graphs,
  product-attributable failures, weak timing, or any family that misses the
  unchanged qualification gate.

Qualification requires at least 500 ms and 2% mean saving, a positive paired
95% lower bound, eight of eight positive pairs, exact outputs, and zero
product-attributable failures. Failed or timed-out observations cannot be
discarded.

## Interpretation boundary

This is POC value evidence, not production authorization. Workflow-family
percentages are not averaged, and they are not added to caching, Edge, task
adapter, or historical mechanism percentages. Online dependency preparation
is excluded. Test execution and Test Optimization remain outside this block;
`testClasses` measures build-owned preparation only.
