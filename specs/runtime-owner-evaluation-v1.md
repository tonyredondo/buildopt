# Runtime owner evaluation v1

Status: executable evidence for `B-G01` and `B-G03`.

The workflow uses the exact public-runner catalog class with four CPUs and
approximately 16 GiB. It creates one eight-project Groovy DSL fixture with
2,048 independent Java sources and a byte-reproducible aggregate ZIP. All arms
use Gradle 9.6.1, locked Temurin 21, offline dependencies, parallel execution,
disabled Build Cache, disabled Configuration Cache, isolated Gradle/project
state, removed outputs, and a single-use daemon.

The A/A slice measures four alternating pairs where labels A and B both use
`STABLE_CONTROL`. The production cohort ledger additionally persists 200
pre-outcome 50/50 assignments, validates sample ratio against recorded
propensities, and sends a complete reward one hour later through the production
bandit engine. Exact replay must not update learning twice.

The autotuning slice measures four alternating `STABLE_CONTROL` W2/H4G versus
`W4_H6G` pairs. Both arguments are materialized by the finite production
resource catalog. Acceptance requires positive mean and lower 95% exhaustive
paired-bootstrap savings, non-regressive p95 and p99, non-positive additional
runner time, an exact zero incremental queue effect because both arms share
one already-started workflow job, no cgroup OOM increment, and identical ZIPs.

Run on the golden runner:

```bash
./dev/run-runtime-owner-evaluation /tmp/buildopt-runtime-evidence
```

The checked-in result is validated without rerunning compilation by
`./dev/check-runtime-owner-evaluation`. This is owner-operated POC evidence;
it neither runs the deferred soak nor authorizes production promotion.
