# Aggregate output closure result

BuildOpt now closes a custom aggregate workflow from exact Gradle task,
producer and output evidence when the existing conventional partition cannot
identify direct outputs.

The immutable matrix uses implementation revision
`8c83ffd21b989ef77be67234a3ca3448aae1747a` and covers Gradle 8.14.3 and
9.6.1 with Groovy and Kotlin DSL. All 4/4 cases:

- traverse the three-task requested graph;
- identify two required custom outputs and one changed producer;
- rebuild only `:changed:emitPayload`;
- materialize the stable output without executing its producer;
- reproduce the baseline two-file SHA-256 exactly; and
- report zero product failures.

The fixture deliberately uses a lifecycle task with no outputs and arbitrary
extension-neutral files under `build/custom-output`. The implementation has no
repository, plugin, task-name, output-directory or extension rule. Unit tests
also prove fail-closed behavior for missing task dependencies, ambiguous output
ownership and unreachable producers.

This is bounded correctness evidence, not a wall-time claim. The state remains
revision-bound; structural profile rebinding is the next POC block.

Validate the frozen contract and evidence with:

```bash
./dev/check-aggregate-output-closure
```
