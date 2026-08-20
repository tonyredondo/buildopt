# Incremental ordinary-build learning result

This bounded Linux AMD64 POC run validates the incremental learning transaction
implemented by `POC-INCREMENTAL-LEARNING-001`. The exact BuildOpt executable
from revision `269742419ec79b9fff65118886ffc1d1f5afdab8` ran the same useful
Gradle `assemble` workflow seventeen times: one discovery baseline followed by
eight alternating control/candidate pairs.

## Result

| Metric | Observed value |
| --- | ---: |
| Ordinary customer invocations | 17 |
| Measurement-only workflow runs | **0** |
| Full-graph workflows | 9 |
| Structural candidate workflows | 8 |
| Graph | 3 projects -> 2 projects |
| Control mean | 5,549.5 ms |
| Candidate mean | 5,499.375 ms |
| Mean saving | 50.125 ms / 0.90% |
| Positive pairs | 4/8 |
| 95% interval | -275.75..+405.125 ms |
| Incremental BuildOpt cost | 19,247 ms |
| Projected break-even | 384 matching builds |
| Decision | `NATIVE_RETAINED` |

Every observation produced the same required JAR digest and the full-graph
fallback remained available. The unchanged value, uncertainty, tail and
30-build payback gates correctly rejected this small fixture. This is protocol
and economics evidence, not a synthetic performance claim.

Unlike the previous synchronous breadth experiment, none of the control or
candidate customer workflows is charged as a separate measurement-only build.
The cost field contains only incremental BuildOpt work around useful
invocations, including first-invocation discovery. Results from the public
five-repository breadth matrix cannot be arithmetically compared with this
fixture because the workloads differ.

## Reproduce and validate

```bash
./dev/run-incremental-learning /absolute/path/to/buildopt /tmp/incremental-learning.json
./dev/check-incremental-learning /tmp/incremental-learning.json
```

The machine-readable evidence is [`summary.json`](./summary.json). The run used
a 12-CPU development host and is classified as bounded local POC evidence. It
adds no repository-specific product rule, production authority, soak or design
partner requirement. Test Optimization remains out of scope.
