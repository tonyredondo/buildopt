# Workflow input ownership

This evidence closes the generic ownership blocker found by the cross-commit
breadth replication. BuildOpt now ignores an unowned changed path only after the
requested Gradle workflow provides complete task-input evidence that no task
consumes it.

## Frozen OpenTelemetry result

The public JMX change contains four paths:

| Changed path | Gradle evidence | Decision |
| --- | --- | --- |
| `CHANGELOG.md` | No project owner and zero consuming tasks | Ignore for this workflow |
| `instrumentation/jmx-metrics/library/jetty.md` | Owned by the JMX library; zero direct consumers | Keep |
| JMX `jetty.yaml` | Consumed by `:instrumentation:jmx-metrics:library:processResources` | Keep |
| JMX `JettyTest.java` | Consumed by `:instrumentation:jmx-metrics:library:compileTestJava` | Keep |

With only `CHANGELOG.md` excluded, automatic discovery completes:

- 1,027 projects in the full graph;
- 8 selected projects;
- 1,019 omitted projects;
- one changed project, `:instrumentation:jmx-metrics:library`;
- candidate entrypoint
  `:instrumentation:jmx-metrics:library:testClasses`; and
- native workflow exit code 0 with zero product failures.

Calibration was deliberately skipped because the remaining budget was
insufficient. The setup observation took 34 seconds, but that duration includes
discovery work and is not a benchmark sample. This block therefore proves
structural ownership only; it does not claim a wall-time improvement or broaden
the cross-commit value claim.

## Why the capture runs late

OpenTelemetry exposed provider-backed task inputs whose files are not finalized
until producer tasks have completed. Querying all inputs before execution failed
closed. BuildOpt now captures the exact attributed workflow inputs after those
tasks run, skips legitimate `NO-SOURCE` tasks, and fails closed on every other
query error.

The one-time setup observation disables Configuration Cache because it inspects
live Gradle task objects. Measured control/candidate builds and ordinary native
fallback do not. Standalone Build Impact discovery attaches no workflow-input
action and remains Configuration Cache compatible.

## Safety matrix

Executable fixtures prove that BuildOpt retains native Gradle for consumed
unowned paths, deleted paths, missing or incomplete input evidence, all-unowned
changes, global changes and unsupported workflows. A provider-backed input
fixture proves the late capture rather than replacing it with a task-name rule.

There are no repository-name branches or filename-extension allowlists.

## Validate

```bash
./dev/check-workflow-input-ownership
```

Base CI also executes the full automatic discovery fixture and Go test suite.
The long public repository replay is not repeated on every push.

## Evidence boundary

- frozen public base `0ae4cc70d2c2a2bd02f71a33f90b80db7e65ef99`;
- frozen public target `dc7e94b40f1b86a19ec3449d895e80ff3c5754a4`;
- implementation `eeb02b6fc1665018fafed634b5d7781957954cd2`;
- 12-CPU Linux development-host structural evidence;
- local ancestry metadata reconstructed for the shallow target while preserving
  its public SHA and tree;
- no production authority, soak, design-partner dependency or performance claim;
  and
- Test Optimization remains out of scope.

The next POC block is native-volatility quarantine.
