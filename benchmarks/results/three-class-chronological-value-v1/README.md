# Saved build plans on later changes in Kafka and Groovy

This Build Impact experiment followed selected improvements into later code
changes. It was planned for Kafka, Groovy and Spring, with a requirement that
at least two projects save time after preparation. Kafka and Groovy both failed
that requirement. Spring was not run because it could no longer change the
overall decision.

## Kafka: the initial saving did not carry forward

Kafka's selected build was faster in all eight initial comparisons. Its output
checks also passed when the build moved to a different directory. But none of
the next three commits could use the saved plan.

Those later builds took 33,378, 4,556 and 11,534 ms longer with BuildOpt. Including
8,587 ms spent checking and publishing the plan, Kafka ended 58,055 ms behind.
The initial win was real for its selected build; it did not repay the recorded
cost across the later changes tested here.

## Groovy: the first comparison was slower

After three ordinary build requests, the first comparison took 26,761 ms with
Gradle and 32,592 ms with BuildOpt: 5,831 ms longer. BuildOpt kept the ordinary
Gradle path, recording `ORDINARY_SAVING_NOT_POSITIVE`. Further timing and later
commits were not run.

The experiment's accounting recorded a net loss of 2,995 ms for Groovy. That
figure covers the charged activity in the study; it is distinct from the
5,831-ms difference in the selected comparison.

## Decision

The two projects together lost 61,050 ms after recorded costs. Required outputs
matched, and no failures were attributed to BuildOpt. Correctness passed, but
recurring savings did not.

Spring is recorded as `NOT_RUN_DEPENDENCY`, meaning a prerequisite had failed.
Even a positive Spring result would have left only one of three projects
passing, below the required two. The final decision was
`STOP_THREE_CLASS_CHRONOLOGICAL_VALUE`.

To verify the experiment rules and results from the repository root:

```bash
./dev/check-three-class-chronological-value-contract
./dev/check-three-class-chronological-value
```
