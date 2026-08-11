# Five-repository review-only CI replay

Hosted run
[`31467370391`](https://github.com/tonyredondo/buildopt/actions/runs/31467370391)
executed the repository-root Action from immutable BuildOpt revision
`18caa8fdf1e71819c1334ac432fd1c6829bd085c`. Five independent Ubuntu runners
created clean public-repository checkouts, committed the frozen owner input,
applied the frozen source change, and produced a review artifact.

| Repository | Reproduced proposal | Graph | Historical value decision |
|---|---|---:|---|
| Spring Framework | `:spring-jms:testClasses` | 27 -> 10 | Retain native Gradle |
| OpenTelemetry | `:instrumentation:spring:spring-boot-autoconfigure:testClasses` | 1,024 -> 34 | Qualified |
| Apache Kafka | `:clients:testClasses` | 64 -> 3 | Qualified |
| Micronaut Core | `:micronaut-http-client-jdk:assemble` | 75 -> 22 | Qualified |
| Apache Groovy | `:groovy-json:classes` | 37 -> 2 | Qualified |

All five verdicts were `MATCH`. Owner input, exact source change, proposal,
project counts, manifest, declared graph, generated binding, fallback input,
reference plan and artifact checksums passed. No active profile was written.

This is reproducibility evidence, not new performance evidence. Spring's graph
matches but remains native because the earlier timing gate accepted only seven
of eight pairs. The other four historical value decisions remain bound to
their existing paired measurements. The replay performed no timing and created
no automatic activation, production, soak, design-partner, or Test
Optimization authority.

Validate the checked summary and fail-closed fixtures with:

```bash
./dev/check-generic-profile-ci-replay
```
