# Aggregate workflow partition result

The bounded local POC completed on 2026-08-20 using implementation commit
`43d934b4140df67217db4875f66c01d37b6f187e`.

A generic 66-project Groovy DSL `assemble` workflow contained one directly
changed `core` project and 65 transitively affected consumers. The previous
flat proposal required 66 candidate entrypoints and exceeded the unchanged
64-entrypoint safety limit. The partitioned proposal rebuilt only
`:core:assemble` and restored the other 65 JARs from exact state captured for
the same repository revision and workflow.

| Observation | Result |
|---|---:|
| Projects in aggregate workflow | 66 |
| Direct change owners | 1 |
| Previous flat candidate | 66 entrypoints, rejected |
| Partitioned candidate | 1 entrypoint |
| Exact outputs materialized | 65 JARs / 60,589 bytes |
| Complete baseline outputs | 66 JARs |
| Complete clean-candidate outputs | 66 JARs |
| Aggregate output comparison | Identical SHA-256 |
| Consumer producer tasks in candidate | 0 |

The implementation derives groups from Gradle producer, lifecycle selector,
variant and output relationships. It does not raise the task cap, branch on a
repository name or assume ABI compatibility across revisions. More than 64
directly changed owners, incomplete ownership or ambiguous output
relationships continue to use native Gradle.

This is correctness and structural-breadth evidence, not a timing result. The
next five-repository transfer must determine whether the smaller candidate
improves wall time and repays learning under the existing gates.

Validate the checked-in evidence with:

```bash
./dev/check-aggregate-workflow-partition
```
