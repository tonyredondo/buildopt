# Cross-commit profile lifetime

## Question

A structural profile is useful only while it can be reused often enough to
repay its one-time discovery and calibration cost. Projected break-even is not
enough: this POC follows one qualified profile through a real public commit
sequence and records every matching replay, non-applicable event, structural
invalidation and wall-time outcome.

The machine-readable contract is
[`poc-profile-lifetime-v1.json`](./poc-profile-lifetime-v1.json). The executable
proof is [`dev/run-poc-profile-lifetime`](../dev/run-poc-profile-lifetime), and
the retained evidence is validated by
[`dev/check-profile-lifetime`](../dev/check-profile-lifetime).

## Public history

The subject is Apache Kafka's `shadowJar` workflow. The profile is learned at
public upstream commit `46ad599a6ef15c6a5f6296cb574ad25a1b7836eb` and then
carried forward, in first-parent order, until public commit
`ab53829feb7280a1d453ebdaad032c4b64bb0f4d` changes Gradle build logic.

The sequence contains ten `DEPENDENCY_SOURCE` events owned by `:clients`, one
resource-only event that does not match that family, and the terminal
build-logic event. Commits between those observations remain part of the real
ancestry. There are no replacement refs, synthetic descendants or rewritten
public SHAs.

## Replay boundary

Intervening source, test, documentation and automation changes do not by
themselves invalidate a profile. They are already present in the current
checkout. BuildOpt still rejects Wrapper or build-logic drift, reassigns the
current event to the stored graph, recomputes its family and plan, and validates
the exact workflow, executable and required-output bindings before Gradle.

This distinction prevents an unrelated README or workflow edit from
permanently stranding useful evidence without weakening the fail-closed
boundary for structural inputs. An unknown current path, different family,
changed build script, Wrapper, executable, workflow or output contract retains
the complete native graph.

## Measurement

Qualification uses eight balanced pairs and the complete native Gradle local
cache as calibration input. The producer publishes the resulting portfolio,
evidence and safe Gradle cache objects through the authenticated local HTTPS
service.

Control and candidate are persistent, isolated workspaces that start from the
same qualified revision and cache opportunity. They then advance through the
same public commits. For every observed revision the harness deletes project
outputs, alternates arm order, records wall time and central cache hits, and
requires the exact same owner outputs. Their private Gradle and BuildOpt caches
survive between commits because that is the customer lifecycle being tested.

The result reports gross matching-replay saving, fallback deltas, calibration
cost, cumulative net saving and the first observed break-even point when one
exists. A negative net result is valid evidence; the checker must never turn a
short lifetime into projected success.

## POC boundary

This is one real repository family, not a universal Gradle claim. It does not
authorize production use, average repository percentages, require a design
partner or soak, or include Test Optimization.
