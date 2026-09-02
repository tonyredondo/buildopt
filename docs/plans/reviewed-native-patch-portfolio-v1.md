# Reviewed Native Patch Portfolio v1 Tracker

| Block | Outcome | State |
|---|---|---|
| `RNPP-001` | Freeze the product pivot and reconstruct the four-candidate selection | `DONE` |
| `RNPP-002` | Reproduce proposal-local correctness from fresh experiment state | `DONE` |
| `RNPP-003` | Measure each correctness-qualified proposal independently | `DONE` |
| `RNPP-004` | Integrate qualifying proposals with the existing review-only Patch Autopilot path | `DONE` |
| `RNPP-005` | Issue the exact portfolio viability decision | `DONE` |

This route does not reopen NAC. It changes the unit of evaluation: every patch
is independently accepted or rejected, and accepted patches leave native
Gradle as the only runtime. The initial selector uses the four NAC classes with
complete public correctness only to choose bounded proposals. All correctness
and timing rows after `RNPP-001` must be fresh.

The terminal result qualifies Micronaut `PythonVfsBytecodeCompile` and Spring
`ArchitectureCheck`; OpenTelemetry and Spring `ShadowSource` are rejected by
their own runtime gates. The accepted proposal descriptors use the existing
digest-bound, owner-reviewed, exact apply/revert Patch Autopilot transaction;
they do not add automatic application or merge authority. Two proposals across
two families save a signed 7,906.625 ms per compatible portfolio build. The
conservative 2,340,000-ms machine campaign charge repays in 296 such builds.
Human review remains explicitly unmeasured and outside that projection.
