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

The subject is Ktor's repository-wide `jvmJar` workflow. The profile is learned
at public upstream commit `24b8773fcee753314dfb753b3f994a7ef36823ef`, a
source-only change owned by the Jetty Jakarta client project. Its direct child,
`eb60b722d1ef1fbfeba8708a7d594fd4083a2aec`, changes that same owner and must
replay the profile. Two further first-parent commits remain deliberately
unobserved before public commit `c237e88696debef1ba860a947cfaad3a0b7cb49b`
changes the unrelated CORS server project and must retain the complete native
graph without deleting the stored profile.

The direct child of that native-retained build,
`835d7f9ff09caf794e9f058d97ad3690d52f960a`, changes publication inputs under
`gradle/**` and must invalidate the profile. The six-commit sequence contains
no replacement refs, synthetic descendants or rewritten public SHAs.

## Replay boundary

Intervening source, test, documentation and automation changes do not by
themselves invalidate a profile. They are already present in the current
checkout. BuildOpt still rejects Wrapper or build-logic drift, reassigns the
current event to the stored graph, recomputes its family and plan, and validates
the exact workflow, executable and required-output bindings before Gradle.

This distinction prevents an unrelated source, README or workflow edit from
permanently stranding useful evidence without weakening the fail-closed
boundary for structural inputs. An unknown current path, different owner or
family, changed build script, Wrapper, executable, workflow or output contract
retains the complete native graph. A later matching event may still select the
unchanged profile after such a safe native-retained build.

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
Gradle daemons do not survive individual invocations: Ktor configures a 10 GiB
heap, so retaining one daemon while a preflight starts another would exceed the
16 GiB host. Both arms therefore use the same `--no-daemon` boundary while all
disk caches and profile state remain persistent.

The result reports gross matching-replay saving, fallback deltas, calibration
cost, cumulative net saving and the first observed break-even point when one
exists. A negative net result is valid evidence; the checker must never turn a
short lifetime into projected success.

## POC boundary

This is one real repository family, not a universal Gradle claim. It does not
authorize production use, average repository percentages, require a design
partner or soak, or include Test Optimization.
