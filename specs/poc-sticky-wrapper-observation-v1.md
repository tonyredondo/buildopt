# Sticky-wrapper ordinary-build observation v1

This is the `SWL-009` contract for the Sticky Wrapper Learning POC. It
records what an ordinary requested Gradle build actually did so later trial
and action blocks can use evidence without asking the developer to run a
second, synthetic command. It is an observation plane only: it cannot select
an action, authorize a cache object, or change Gradle arguments.

## Record and storage

Each completed wrapper invocation appends one canonical JSON line to a private
user-cache file. The path is derived from a project scope and can be overridden
for a checker with `BUILDOPT_STICKY_OBSERVATION_OUTPUT`; committed files never
contain the checkout path or a credential. A sibling lock prevents concurrent
invocations from interleaving records. The default `light` mode avoids a
pre-build Git command, starts the expensive BuildOpt executable digest
concurrently with Gradle, and creates the recorder only after the child exits,
so a normal wrapper call pays only the bounded decision cost before Gradle
starts. If a short build finishes before that digest is ready, the record keeps
the explicit unavailable digest instead of delaying completion.
`full` mode is reserved for diagnostic runs that explicitly need a best-effort
source-revision lookup. Writes are best effort and never change the requested
build's exit code.

The record binds:

- a repository scope, source revision evidence, Gradle version and Wrapper
  digest;
- the BuildOpt executable and Gradle argument digests (the executable digest is
  best-effort in `light` mode and exact in `full` mode when available);
- the outcome (`SUCCESS`, `BUILD_FAILURE`, `INFRA_FAILURE` or `CANCELLED`);
- canonical UTC start/end timestamps and the original exit code; and
- whether Configuration Cache was requested and whether its state directory
  was observed after the build.

## Timing contract

The total is the wall time observed by the launcher. The timing object uses
mutually exclusive phase buckets:

| Phase | Evidence | Meaning |
| --- | --- | --- |
| decision | exact | local wrapper/connection decision work until execution preparation |
| network | exact or unavailable | connection setup when it was attempted |
| cache | approximated | cache/connection hand-off before the child starts |
| gradle | exact | child process start through completion |
| observation | exact or unavailable | post-build session observation/ingest |
| wrapper/bootstrap | unavailable unless measured outside BuildOpt | shell wrapper and distribution bootstrap are outside the launcher clock |
| unattributed | exact residual | measured time not safely assigned to another phase |

Unavailable phases have no duration. Every available duration plus the
residual must equal `totalNs`; missing knowledge is never converted to zero.
This makes ordinary records suitable for later cost accounting without
claiming causal savings.

## Configuration Cache and safety

The observer does not parse Gradle internals or infer task-level causality.
`PRESENT` only means that the standard `.gradle/configuration-cache` state was
observed after a requested build. Reuse is established by comparing multiple
records and the corresponding Gradle output, not by this flag alone.

If recording fails, the wrapper reports a diagnostic and retains the native
Gradle result. `BUILDOPT_BYPASS=1` and `BUILDOPT_STICKY_OBSERVATION=0` disable
the observer before it creates a directory or opens a connection. The Gradle
child never receives the output path or mode variables. The separate
sticky-wrapper no-op contract measures this startup trade-off and proves that
the no-op path does not start a gateway, plugin handshake, managed L1 or
central-cache probe when none is configured.

## Validation

The JSON Schema is
[`sticky-wrapper-observation.v1.schema.json`](../contracts/jsonschema/sticky-wrapper-observation.v1.schema.json).
Run the focused contract and real-wrapper check with:

```bash
./dev/check-sticky-wrapper-observation
```

The checked-in result is
[`sticky-wrapper-observation-v1.json`](../benchmarks/results/sticky-wrapper-observation-v1.json).
It is a local POC observation sample, not a qualification or speedup claim.
