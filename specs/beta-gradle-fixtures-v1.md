# Private-beta Gradle fixture-size matrix v1

This contract materializes the small, medium, and large golden-lane build
matrix required by `OPS-001/A1` and the fixture portion of `A1-G02`. It binds
the three rows in `benchmarks/beta-v1.yaml` to deterministic Kotlin DSL
multi-project repositories without treating their elapsed time as performance
qualification.

## Repository shapes

Each private temporary repository is a linear chain of Java projects. Every
project contains a fixed number of generated Java 17 sources; the first source
in a project depends on the final source in the preceding project. Therefore
the ordered `compileJava` tasks are the declared Gradle critical path rather
than labels inferred after execution.

| Fixture | Projects | Sources/project | Known output | Critical-path tasks |
|---|---:|---:|---:|---:|
| `TIER1_SMALL` | 2 | 4 | 8 | 2 |
| `TIER1_MEDIUM` | 8 | 8 | 64 | 8 |
| `TIER1_LARGE` | 24 | 16 | 384 | 24 |

The final class must return the exact known output, and the repository must
contain exactly the declared number of compiled class files. These checks make
truncated or incorrectly connected fixture generation fail even if Gradle
itself exits successfully.

## Cache preservation

The real BuildOpt settings plugin selects a private generation-segmented
native L1. Gradle 9.6.1 on JDK 21 executes every critical-path task as
`SUCCESS`, deletes all project outputs, and repeats the same invocation. The
second execution must restore every critical-path task `FROM_CACHE`, retain
the known output, reuse Configuration Cache, and leave actual entries in the
managed L1.

Run the bounded matrix:

```bash
./dev/check-beta-gradle-fixtures
```

The checker emits and strictly inspects a private mode-`0600` result before
removing it. This closes only `OPS-001/A1-GRADLE-FIXTURE-SIZE-MATRIX`. The
eight-hour qualification remains required before `A1-G02` or the complete
`OPS-001/A1` profile can close.
