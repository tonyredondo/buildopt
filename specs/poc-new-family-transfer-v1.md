# Ktor new-family transfer experiment

## Question

Can the unchanged generic owner-input and structural-profile path find useful,
output-preserving Build Impact in a substantial Kotlin Multiplatform Gradle
family that was not used to develop the current profiles?

## Frozen subject

Ktor is frozen at public revision
`bc7de799f4eb997a63250f2f70492d85f5c92f50` and requires JDK 21. The POC uses
the public unqualified `jvmJar` task selector exposed by Ktor's Gradle model.
That selector covers the repository's JVM JAR tasks without claiming to
represent the full multiplatform release build.

Before this contract was committed, a non-measured `./gradlew help` run proved
only that the fixed checkout, Gradle 9.5.1 Wrapper and locked Temurin 21 could
configure. `help --task :ktor-http:jvmJar` confirmed one Gradle-owned JVM JAR
task. No owner build, BuildOpt proposal, candidate graph or target-workflow
timing was run or observed.

The initial preregistration selected root `assemble` with `-Ptarget.*=false`
options. A pre-measurement compatibility run proved that Ktor's custom target
loader reads `gradle.properties` files directly and therefore ignores those
CLI target properties; native tasks were scheduled. The attempt was stopped
before proposal completion, warm-up or timing. A second non-measured
`help --task jvmJar` run confirmed the public JVM-only selector across Ktor
subprojects. This amendment changes only the workflow description: the public
revision, change, required output, thresholds and generic mechanism remain
frozen, and no data from the invalid attempt is accepted.

The repository-owned inputs are frozen as follows:

- original workflow: unqualified `jvmJar` across matching Ktor subprojects;
- exact change: append one fixed comment to the internal
  `ktor-http/common/src/io/ktor/http/AsciiBitSet.kt` source;
- required output: `ktor-http/build/libs/ktor-http-jvm-*.jar`;
- semantic boundary: exact bytes;
- optimized-native control: the same workflow, daemon, build cache, parallel
  execution, disabled Configuration Cache and 12 workers;
- candidate mechanism: generic structural Build Impact only.

No candidate entrypoint, project count, expected saving or favorable result is
preregistered. Missing output ownership, an uncertain graph or unsupported
semantics retain the native full graph before timing.

## Independent value decision

Two fresh captures each contain eight alternating control/candidate pairs.
Adjacent opposite-order pairs form eight balanced blocks across both captures.
Qualification requires all of the following:

- mean saving of at least 500 ms and 2%;
- positive median and deterministic 95% bootstrap lower bound over blocks;
- at least six of eight positive blocks;
- candidate p95 no slower than native p95;
- exact required-output bytes and stable measured task shapes;
- successful full-graph fallback in both captures; and
- zero product-attributable failures, with no failed or timed-out observation
  discarded.

An unsupported, weak, noisy or non-repeatable result retains optimized native
Gradle. Repository-name product rules, post-result tuning and threshold
movement are forbidden. A generic correctness defect may be fixed only if the
failed attempt is preserved and the frozen subject is rerun from zero.

## POC boundary

This experiment can validate transfer to one additional public family. It does
not establish universal savings or authorize automatic activation,
production, Test Optimization, soak testing or design-partner work.

Validate the preregistration before any owner workflow, proposal or timing:

```bash
./dev/check-new-family-transfer --spec-only
```

The capture and offline evidence commands are added only after this
preregistration is committed, so their implementation cannot encode an
observed Ktor result.

The implementation commit distinguishes the per-capture envelope schema from
the balanced aggregate schema. This naming correction was made before any
owner build, proposal or timing and changes no frozen subject, method or gate.
