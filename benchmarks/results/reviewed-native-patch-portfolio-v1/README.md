# Build corrections in Micronaut, Spring and OpenTelemetry

Patch Autopilot proposes small changes to a project's build files for someone
to review and undo if necessary. This experiment tested four such changes in
three repositories. Two met the savings requirements: a Micronaut task that
compiles Python bytecode and a Spring task that checks architecture rules.

## What the comparisons showed

Each proposed correction went through fresh checks of its output and eight
timed comparisons with the original task. Earlier experiments helped select
the four candidates; their timings were not reused here.

| Repository and task | Result | Did it meet the requirements? |
| --- | --- | --- |
| Micronaut Core, `PythonVfsBytecodeCompile`: compiles Python bytecode | Saved 6,921.125 ms, or 63.44%. Faster in all eight comparisons. | Yes. |
| Spring Framework, `ArchitectureCheck`: checks architecture rules | Saved 985.5 ms, or 35.34%. Faster in all eight comparisons. | Yes. |
| OpenTelemetry Java Instrumentation, a task generating instrumentation version information | Faster in six of eight comparisons, but its slower builds became slower. | No. |
| Spring Framework, `ShadowSource`: prepares source files for packaging | Saved 137.875 ms. | No. The minimum was 500 ms. |

The two accepted corrections produced identical files even when the project
was built in a different directory. Neither caused a build failure. Their
statistical intervals supported a positive saving, and their p95 times, which
describe the slower builds, improved.

## What this establishes

The Micronaut and Spring corrections saved time on the selected tasks. They
were prepared as reviewable patches, tied to the source versions that had been
checked, and could be reversed exactly. The experiment did not apply or merge
them automatically.

The setup calculation added the two accepted task savings: 7,906.625 ms. It
counted 2,340,000 ms of machine time spent preparing and checking all four
proposals, including the unsuccessful ones. That calculation estimated
296 repetitions to recover the preparation time, assuming both accepted
savings occurred on every repetition.

This was an accounting model across two repositories. No single build combined
both savings, and no sequence of 296 real code changes was observed. Human
review time was not measured or included. The task results therefore support
these individual corrections; they do not yet show recurring savings from
automatically finding and maintaining patches.

The records contain the [candidate selection](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/reviewed-native-patch-portfolio-v1/selection.json),
[measurements and checks](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/reviewed-native-patch-portfolio-v1/result.json)
and [two accepted proposals](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/reviewed-native-patch-portfolio-v1/accepted-proposals.json).
