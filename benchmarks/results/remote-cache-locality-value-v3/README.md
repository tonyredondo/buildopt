# Edge Cache across five open-source projects

This experiment tested whether reading saved build results from a nearby
BuildOpt cache offered a useful advantage over reading them directly from a
shared cache. Five projects were checked; three could proceed to timing.
None of those three met all the savings requirements.

## What was compared

Both versions used the same build command and required output. The difference
was where they read the cached files. The simulated remote connection added
30 ms per response and limited transfers to 100 MiB/s. These conditions were
fixed before measurement.

The cache itself passed checks for restarting offline, rejecting corrupted
data and refusing unsafe reuse. Five trial downloads of 1 MiB took
41.045–41.668 ms, consistent with the chosen network settings. These were
checks of the experiment setup, not build-speed results.

| Repository | Build being checked | Could timing proceed? |
| --- | --- | --- |
| Apache Groovy | `jar`, packaging the Groovy library | Yes. Required output matched. |
| Apache Kafka | `:clients:shadowJar`, packaging the client library | No. Ordinary Gradle builds produced inconsistent required output. |
| Micronaut Core | `:micronaut-core:jar`, packaging the core library | No. The workflow reused no cached task results, so there was no remote read to improve. |
| OpenTelemetry Java Instrumentation | Packaging its Spring Web 6.0 instrumentation library | Yes. Required output matched. |
| Spring Framework | `:spring-core:jar`, packaging Spring Core | Yes. Required output matched. |

The [repository and command list](https://github.com/tonyredondo/buildopt/blob/main/specs/poc-remote-cache-locality-value-v3.subjects.json)
records the exact source versions and full command names. Kafka and Micronaut
were not measured speed failures; they did not reach that comparison.

## Timing results

Each eligible project completed eight comparisons, alternating which version
ran first. All 24 pairs produced matching required output, with no failures
attributed to BuildOpt.

| Repository | Average reading directly | Average with Edge Cache | Change in elapsed time | Why it did not pass |
| --- | ---: | ---: | --- | --- |
| Apache Groovy | 68,184.375 ms | 66,945.875 ms | 1,238.5 ms faster, or 1.82% | Below the required 2% saving. |
| OpenTelemetry Java Instrumentation | 59,443.25 ms | 59,607.625 ms | 164.375 ms slower, or 0.28% | Slower on average and in its p95 result. |
| Spring Framework | 25,852.875 ms | 25,180.125 ms | 672.75 ms faster, or 2.60% | The statistical result still allowed a loss. |

Groovy was faster in seven of eight pairs, OpenTelemetry in three and Spring
in six. The lower ends of their 95% saving estimates were +340.5, −603.375 and
−605.125 ms respectively. Spring's positive average therefore did not establish
a reliable saving under the agreed check. The p95 measures the slower builds.

## Decision and records

The study required three projects to pass; none did. It stopped with
`STOP_REMOTE_CACHE_LOCALITY_VALUE_V3`. The later installed-use and preparation
recovery study was not run because this prerequisite failed.

The nearby cache served the files correctly, but the saved transfer time did
not produce a qualifying build-time advantage across these projects. The
earlier [Kafka experiment](https://github.com/tonyredondo/buildopt/blob/main/docs/findings/buildopt-kafka-edge-cache-experiment.md)
used a much slower simulated connection. Its selected gain does not establish
the same benefit under the conditions tested here.

The original records remain available: [setup checks](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/remote-cache-locality-value-v3/harness-proof.json),
[output checks](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/remote-cache-locality-value-v3/public-correctness.json),
[all timed comparisons](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/remote-cache-locality-value-v3/paired-value.json)
and [stopping decision](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/remote-cache-locality-value-v3/terminal-decision.json).
The [experiment rules](https://github.com/tonyredondo/buildopt/blob/main/specs/poc-remote-cache-locality-value-v3.md)
and [tracker](https://github.com/tonyredondo/buildopt/blob/main/docs/plans/remote-cache-locality-value-v3-poc-tracker.md)
retain the requirements and completed stages. No timing from the earlier
version of this study was used in these results.
