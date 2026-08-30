# Durable Native Optimization POC Tracker

**Status:** active<br>
**Current block:** `DNO-003`<br>
**Terminal outcomes:** `CONTINUE_DURABLE_NATIVE_OPTIMIZATION_POC` or
`STOP_DURABLE_NATIVE_OPTIMIZATION_POC`

## Objective

Test whether BuildOpt can find, compile and validate generic persistent Gradle
corrections that reduce customer-visible wall time after one owner-reviewed
commit. Native Gradle remains the runtime after acceptance. The route must beat
an optimized native control; enabling a standard Gradle feature is not credited
as differentiation by itself.

## Blocks

| Block | Deliverable | Entry gate | Exit gate | State |
| --- | --- | --- | --- | --- |
| `DNO-001` | Frozen catalog, thresholds, authority and documentation ledger | User-selected materially different hypothesis | Contract validator rejects drift and every old route remains terminal | `DONE` |
| `DNO-002` | Five-family source opportunity report | DNO-001 | 5/5 conclusive and at least 3/5 source-bound action families; no timing | `DONE` |
| `DNO-003` | Generic digest-bound patch compiler and exact transaction | DNO-002 pass | Idempotent apply, exact revert, ambiguous/drifted sources rejected | `IN_PROGRESS` |
| `DNO-004` | Fixture and public-candidate correctness matrix | DNO-003 | Exact required outputs and zero product failures | `WAITING` |
| `DNO-005` | Installed paired value | DNO-004 | Frozen value threshold in at least 3/5 families | `WAITING` |
| `DNO-006` | Twenty-commit durable value per admitted family | DNO-005 | Positive signed portfolio, at least 3/5 positive families and finite payback | `WAITING` |
| `DNO-007` | One-command proposal UX and terminal scorecard | DNO-006 or first failed prerequisite | Truthful terminal decision; no unmeasured value presented as zero | `WAITING` |

## Detector catalog

| Detector | Required evidence | First compiler support |
| --- | --- | --- |
| Complete custom-task contract | DefaultTask implementation, TaskAction, declared input/output, no cacheability marker or prohibition | Additive `@CacheableTask` marker, owner review required |
| Recurrent clean elision | Recurrent exact request plus cleanless output equivalence | Not yet available |
| Declared graph scope | Exact source edge and complete output-preserving closure | Not yet available |
| Internal compile classpath | Unpublished module, ABI absence, consumer compile and metadata proof | Not yet available |

An unavailable compiler cannot contribute to breadth. Repository and task names
are evidence labels only and may not influence detection.

## Measurement and stop policy

Timing uses a controlled local runner, balanced alternating arms and signed
wall-time differences. Hosted CI owns correctness only. Percentages from
different mechanisms or repositories are never added or averaged. A breadth or
correctness failure stops before timing. A value failure preserves every pair
and stops the route without threshold movement or repository specialization.

## Documentation ledger

Every completed block updates this tracker, the machine contract, specification
index, benchmark index, validation reference, implementation tracker,
generalization audit, performance findings and POC one-pager. Runtime or CLI
changes additionally update architecture, onboarding, configuration and
troubleshooting.
