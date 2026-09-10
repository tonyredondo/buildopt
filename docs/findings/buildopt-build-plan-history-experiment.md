# Smaller build plans across five project histories

This experiment tried to reuse smaller parts of a build plan as a repository
changed. A build plan describes the work Gradle needs to perform for a request.
The hope was that smaller plans would remain applicable more often than the
whole plans tested earlier.

Across 100 comparisons in five repositories, BuildOpt applied no optimization.
It kept the ordinary Gradle build every time. The approach therefore produced
no demonstrated optimization saving and added work of its own.

## What ran

Each project contributed twenty comparisons through recorded code changes.
The commands covered different work: assembling Groovy and Micronaut,
compiling Kafka's tests and Spring's production code, and packaging an
OpenTelemetry library.

| Repository | Comparisons | Total extra elapsed time with BuildOpt | Time recorded inside BuildOpt |
| --- | ---: | ---: | ---: |
| Apache Groovy | 20 | 25.207 s | 11.048 s |
| Apache Kafka | 20 | 42.309 s | 10.343 s |
| Micronaut Core | 20 | 144.656 s | 79.788 s |
| OpenTelemetry Java Instrumentation | 20 | 26.044 s | 13.025 s |
| Spring Framework | 20 | 130.406 s | 64.825 s |
| Total | 100 | 368.623 s | 179.029 s |

These totals have been rounded to milliseconds. Required outputs matched in
all 100 comparisons, with no failures attributed to BuildOpt. The source
record also retains one excluded attempt; it is not an additional comparison
in this table.

## Why faster individual runs were not optimization wins

Twenty-five comparisons happened to be faster and seventy-five slower. But no
saved plan or smaller plan was used in any of them. The faster observations
cannot be credited to an optimization that never ran.

BuildOpt recorded 179.029 seconds spent on its own work, including looking for
opportunities, managing saved state and checking outputs. The remaining
189.593 seconds of the total difference came from Gradle and the measurement
environment. The records do not establish the cause of that remainder, so the
full 368.623-second loss cannot all be assigned to BuildOpt's code.

All five projects took longer in total. The original assessment
classified four as negative and OpenTelemetry as inconclusive. A negative
total alone does not remove that uncertainty.

## Decision

Making the plans smaller did not solve the central problem: the system still
could not safely apply them often enough to save time. This research stopped.
The result concerns plan selection and reuse; it did not test how long the
separate Micronaut, Spring or Elasticsearch task corrections would keep helping.

The [full timing and attribution record](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/current-longitudinal-attribution-v1.json)
includes each project's result, recorded overhead and reasons for declining
an optimization. The [stopping decision](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/adaptive-fragment-terminal-decision-v1.json)
retains the conclusion for this approach.
