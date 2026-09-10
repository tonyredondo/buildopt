# Build Impact on Ktor and Apache Beam

Build Impact tries to reduce the parts of a project that Gradle needs to prepare
for a requested build. In this experiment, it made a selected Ktor library build
79.82% faster and Apache Beam compilation 61.65% faster. Both produced the
required files. These are results for those two commands; the experiment did
not establish how often the same improvements would help on later code changes.

## What ran

The published [BuildOpt v0.6.1 package](https://github.com/tonyredondo/buildopt/releases/tag/v0.6.1)
was installed in a new directory and used with fresh copies of both repositories.
BuildOpt started without saved state or manually prepared configuration files.

Ktor ran `jvmJar --max-workers=12`, which builds its JVM library packages. Beam
ran `classes --max-workers=12`, which compiles production classes. Each command
was compared with ordinary Gradle eight times, alternating which version ran
first. Both versions started with the same dependencies and saved Gradle build
results. Warmup runs were excluded from timing.

## Results

| Repository | Average with Gradle | Average with BuildOpt | Time saved | Projects in the build plan, before → after |
| --- | ---: | ---: | ---: | ---: |
| Ktor | 38.810 s | 7.830 s | 30.979 s / 79.82% | 133 → 10 |
| Apache Beam | 65.081 s | 24.958 s | 40.123 s / 61.65% | 316 → 6 |

BuildOpt was faster in all eight comparisons for each repository. Every pair
produced matching required files, and the recorded task outcomes were stable.
There were no failures attributed to BuildOpt. The two percentages describe
different commands and must not be averaged.

The statistical checks also supported a saving. The 95% interval was
24.679–38.922 seconds for Ktor and 33.867–51.470 seconds for Beam. The p95, a
measure of the slower builds, fell from 61.575 to 11.957 seconds for Ktor and
from 102.621 to 25.946 seconds for Beam.

At these measured savings, recovering the recorded setup time would take
26 applicable Ktor builds or 28 applicable Beam builds. Those are estimates:
this experiment did not follow either project through that many later changes.

## Checks and limits

A separate Ktor check changed the root `settings.gradle.kts` file, which can
affect the whole build. BuildOpt declined to use the reduced plan, and the full
Gradle build succeeded. The recorded reason was
`GLOBAL_CHANGE_REQUIRES_FULL_GRAPH`. This checked the decision to leave Gradle
in control when a saved plan could not safely apply; it was not a timing result.

Two earlier v0.6.0 attempts remain in the records. One could not use the network
inside its sandbox. The other exposed a defect in finding output files when
Gradle's Configuration Cache was enabled; v0.6.1 fixed it. Neither attempt
contributed to the reported timings.

The [experiment data](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/poc-magic-end-to-end-value-v2/summary.json)
records the comparisons, output checks and setup calculation. To verify those
records from the repository root:

```bash
./dev/check-magic-end-to-end-value-v2
```

This establishes selected Build Impact savings and a working installation.
It does not establish a general saving across everyday development or measure
test selection.
