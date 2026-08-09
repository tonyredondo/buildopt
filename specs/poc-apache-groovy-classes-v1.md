# Apache Groovy classes generalization result

## Decision

The generic BuildOpt structural workflow qualifies one exact Apache Groovy
`classes` scope as a POC candidate. Optimized native Gradle remains the default
for every other workflow, output scope and change.

This result tests whether the repository-independent workflow built from
`profile analyze`, `profile measure` and `profile evaluate` transfers to a
fresh substantial Gradle family without a repository-name rule. It does not
authorize production rollout, automatic activation or a general Apache Groovy
profile.

## Frozen subject

| Field | Value |
|---|---|
| Public repository | [`apache/groovy`](https://github.com/apache/groovy) |
| Public base revision | `c16898314460cdfc3e0f0d7d24822585bbd1997e` (`GROOVY_5_0_8`) |
| Local measurement revision | `7e71c394ac27ba8f741613251181cb9e5129927a` |
| Changed source | `subprojects/groovy-json/src/main/java/org/apache/groovy/json/DefaultFastStringService.java` |
| Change | One package-private method returning the constant `1` |
| Gradle | Wrapper 8.14.4 |
| Java | Project-locked Temurin 21 |
| Optimized native entrypoint | `classes` |
| BuildOpt entrypoint | `:groovy-json:classes` |
| Required output | `subprojects/groovy-json/build/classes/**` |
| Structural reduction | 37 projects to 2; 35/37 or 94.59% omitted |

The local revision contains only the declared source mutation on top of the
public tag. Its exact changed-path digest is bound into the measurement
evidence. No BuildOpt change is applied to Apache Groovy.

## Candidate selection

Three repository-owned output scopes were assessed in sequence:

1. `:groovy-binary:distBin` was rejected before accepting any timing because
   its ZIP did not match the output of the root distribution workflow byte for
   byte.
2. `:groovy-json:assemble` was stopped before measured pairs because the root
   `assemble` graph also pulled documentation and Asciidoctor work. That was not
   a focused compilation comparison and would not answer the intended question.
3. `:groovy-json:classes` matched the requested compilation output and passed
   exact-output, isolation, fallback and value gates.

No rejected timing is reused. The accepted workflow was measured from fresh
isolated state after its semantic output scope was fixed.

## Method

The control and candidate both used `--no-daemon`, `--build-cache`,
`--parallel`, `--no-configuration-cache`, `--console=plain`, `--no-scan` and
`--max-workers=4`. The generic runner created separate clones, Gradle homes and
native-cache seeds; alternated which arm ran first; included launcher and
planning overhead; and required a successful full-graph fallback for a
`gradle.properties` change.

Every one of the eight pairs produced the same 66 class files with SHA-256
`c2031f4fe54da7b49d30e53c0dc2f2efefb2b0dc931b42a61f50101a3b2ca70e`.

| Metric | Optimized native Gradle | Installed BuildOpt | Difference |
|---|---:|---:|---:|
| Mean wall clock | 92,350.625 ms | 46,119.875 ms | **46,230.75 ms / 50.06% faster** |
| Alternating pairs | 8 | 8 | **8/8 positive** |
| 95% paired interval | — | — | **+44,190.25..+47,846.875 ms** |
| Required output | 66 class files | Same 66 class files | Byte-identical in every pair |

## Interpretation

The 94.59% graph reduction does not translate linearly into wall time, but it
does create a material end-to-end cascade: less unrelated configuration and
compilation work reduced the complete measured path by 50.06%. This is direct
composition evidence; no mechanism percentages are added together.

The experiment strengthens the claim that reviewed structural Build Impact can
beat optimized native Gradle on more than one repository family. It does not
prove that every `classes`, `build`, `assemble`, distribution or test workflow
is safely reducible. Required outputs remain repository-owned, and any unknown,
global or drifted state must retain the native full graph.

The exact evidence bundle is in
[`benchmarks/results/poc-apache-groovy-classes-v1`](../benchmarks/results/poc-apache-groovy-classes-v1/).
Validate its hashes, calculations, generated profile and tamper fallback with:

```bash
./dev/check-poc-apache-groovy-classes-v1
```
