# Tier 1 Gradle fixtures

`F0-040` materializes the Linux AMD64 `COMPAT-001` target as two independent
consumer repositories: one Kotlin DSL and one Groovy DSL. Both include the same
fixture plugin, load the packaged BuildOpt plugin, run a Java 17 cacheable
custom task, resolve a real `ArtifactTransform`, and reuse Configuration Cache
and the build cache.

[`matrix.v1.json`](./matrix.v1.json) is the target matrix, not an execution
claim. The checker reports only runtime pairs it actually executes. The
repository's capability matrix remains authoritative about evidence at the
current revision.

Run all locally available rows:

```bash
./dev/check-tier1-fixtures
```

The check runs each DSL twice through Gradle TestKit and twice through a
temporary real Wrapper repository for Gradle 8.14.3 and 9.6.1. It requires the
locked JDK 21 and uses a detected Java 17 installation when present. JDK 25 is
left unproven until its lock-owned runtime is provisioned; missing runtimes
never inherit another row's result.
