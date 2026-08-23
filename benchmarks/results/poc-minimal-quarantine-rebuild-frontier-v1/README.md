# Minimal quarantine rebuild frontier result

This experiment tests whether replacing the graph-proven project lifecycle
cover with the mathematically minimal direct-producer DAG frontier makes the
frozen Micronaut `assemble` replay economically useful. It uses the same public
revisions, producer portfolio, exact-output boundary, eight alternating pairs,
fallback and robust qualification policy as the preceding lineage experiment.

## Result

The direct frontier is correct but not valuable. All eight pairs finish with
one exact required-output digest, 101 transported outputs, 89 locally rebuilt
outputs, a successful full-graph fallback and zero product failures. The
frontier remains within the unchanged 64-task POC limit.

| Strategy | Entrypoints | Projects | Native mean | BuildOpt mean | Result |
|---|---:|---:|---:|---:|---:|
| Graph-proven project lifecycle cover | 58 | 52/70 | 13,318.25 ms | 13,253.25 ms | 65 ms / 0.49% faster; 5/8 |
| Minimal direct-producer DAG frontier | 63 | 52/70 | 12,656.125 ms | 13,365.5 ms | **709.375 ms / 5.60% slower; 3/8** |

The direct frontier contains 38 `javadocJar`, 22 `jar`, two `assemble` and one
`sourcesJar` task. It does not reduce the selected project graph and adds five
terminal entrypoints. Within the direct-frontier capture, Gradle accounts for
680.625 ms of the 709.375-ms mean regression; wrapper work adds only 28.75 ms.
The saved-time interval is -1,877.625..+500.25 ms and candidate p95 is 15,053
ms versus 13,589 ms for optimized native Gradle.

The two captures are independent and their absolute means are not treated as a
paired cross-run comparison. Each capture is interpreted only through its own
alternating control/candidate pairs. Neither strategy passes the unchanged
value gate.

## Decision

`RETAIN_GRAPH_PROVEN_PROJECT_FRONTIER`.

The experimental direct-frontier implementation is reverted after capture.
BuildOpt keeps the previously verified graph-proven lifecycle cover because the
new strategy reduces neither projects nor wall time. The negative evidence is
retained so this hypothesis is not repeated without materially different task
or critical-path evidence.

This is bounded POC evidence. It adds no repository-specific rule, production
authority, weaker output gate, soak or design-partner requirement, averaged or
added percentage, or Test Optimization behavior.

Revalidate the checked evidence with:

```bash
./dev/check-minimal-quarantine-rebuild-frontier
```
