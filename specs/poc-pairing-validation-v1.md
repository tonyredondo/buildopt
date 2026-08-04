# POC paired breadth validation v1

This contract determines whether the realistic breadth classifications remain
stable after removing long control/candidate time separation. It changes the
measurement schedule, not the product, fixture, tasks, outputs, sample count,
runner, or decision thresholds.

Each strict batch keeps two digest-pinned containers alive concurrently: one
for the native Gradle control and one for the BuildOpt candidate. Their
workspaces, Gradle homes, daemons, installed prefixes, and writable state remain
private. Each workload cell is warmed once in each arm. The eight measured
pairs then run as consecutive `docker exec` operations across the two
containers, alternating which arm runs first. The second batch reverses the
starting order. The next arm must begin no more than five seconds after the
first arm completes; timestamps and the observed gap are checked for every
pair.

Both reports must be bound to the same commit and artifacts, preserve all
correctness guardrails, and use the unchanged thresholds. A stable outcome
requires the same classification for every Kotlin/Groovy change cell across
the opposite pair sequences. Reproduced failures are valid targets for the next
POC experiment. A mismatch remains valid negative evidence: it records
`MEASUREMENT_UNSTABLE`, retains the narrow claim, and authorizes no product
change from these measurements.

Generate a strict batch with:

```bash
./dev/run-poc-pairing-container /absolute/report.json batch-id CONTROL_FIRST
```

Generate the cross-batch decision and validate checked evidence with:

```bash
./dev/assemble-poc-pairing-decision \
  /absolute/decision.json \
  /absolute/control-first.json \
  /absolute/candidate-first.json
./dev/check-poc-pairing \
  /absolute/control-first.json \
  /absolute/candidate-first.json \
  /absolute/decision.json
```

This remains an owner-controlled POC experiment. It does not claim universal
savings, production readiness, an eight-hour soak, external validation, or any
Test Optimization behavior.
