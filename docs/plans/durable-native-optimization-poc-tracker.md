# Durable Native Optimization POC Tracker

**Status:** closed<br>
**Current block:** none<br>
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
| `DNO-003` | Generic digest-bound patch compiler and exact transaction | DNO-002 pass | Idempotent apply, exact revert, ambiguous/drifted sources rejected | `DONE` |
| `DNO-004` | Fixture and public-candidate correctness matrix | DNO-003 | Exact required outputs and zero product failures | `DONE_STOP` |
| `DNO-005` | Installed paired value | DNO-004 | Frozen value threshold in at least 3/5 families | `NOT_AUTHORIZED` |
| `DNO-006` | Twenty-commit durable value per admitted family | DNO-005 | Positive signed portfolio, at least 3/5 positive families and finite payback | `NOT_AUTHORIZED` |
| `DNO-007` | One-command proposal UX and terminal scorecard | DNO-006 or first failed prerequisite | Truthful terminal decision; no unmeasured value presented as zero | `DONE` |

## Detector catalog

| Detector | Required evidence | First compiler support |
| --- | --- | --- |
| Complete custom-task contract | DefaultTask implementation, TaskAction, declared input/output, no cacheability marker or prohibition | Additive `@CacheableTask` marker, owner review required |
| Recurrent clean elision | Recurrent exact request plus cleanless output equivalence | Not yet available |
| Declared graph scope | Exact source edge and complete output-preserving closure | Not yet available |
| Internal compile classpath | Unpublished module, ABI absence, consumer compile and metadata proof | Not yet available |

An unavailable compiler cannot contribute to breadth. Repository and task names
are evidence labels only and may not influence detection.

## Correctness outcome

The frozen marker-only compiler is not safe for every candidate admitted by
the first detector. Spring's two candidates and OpenTelemetry's candidate
preserve exact required outputs and restore from Gradle's build cache.
Micronaut's candidate fails Gradle validation because its `sourceDirectory` is
an `@InputDirectory` without a path-normalization annotation. That is a product
failure introduced by the patch, exhausting the zero-failure budget.

The failure is actionable but cannot be repaired inside DNO v1: the frozen
compiler may add only `@CacheableTask`, while choosing `RELATIVE`, `NAME_ONLY`,
`NONE`, `CLASSPATH` or `COMPILE_CLASSPATH` requires task semantics the current
detector does not prove. Groovy and the fixture matrix were therefore stopped,
not recorded as zero. `DNO-005` and `DNO-006` are not authorized; `DNO-007`
must now issue the terminal scorecard and may recommend a separately
preregistered normalization-aware successor.

## Terminal decision

`STOP_DURABLE_NATIVE_OPTIMIZATION_POC` is the only decision supported by v1.
The route passes contract immutability, source breadth and reversible patch
transactions, then fails the zero-product-failure correctness gate. Paired
value, longitudinal value and the installed proposal UX remain typed
`NOT_MEASURED_NOT_AUTHORIZED` or `NOT_IMPLEMENTED_NOT_AUTHORIZED`; they are not
reported as zero and no speedup claim exists.

The result does not prove durable native optimization has no value. It proves
that the first generic detector—declared input/output annotations plus a
marker-only patch—is insufficiently semantic. A possible successor must be a
new preregistered experiment that distinguishes already-normalized file inputs
from missing-normalization tasks, treats any new normalization as an explicit
owner-reviewed patch class and repeats fresh breadth/correctness before timing.
This recommendation is not automatic successor authority.

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
