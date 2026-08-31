# Normalization-Aware Cacheability POC Tracker

**Status:** closed — `STOP_NORMALIZATION_AWARE_CACHEABILITY_POC`<br>
**Current block:** none; all seven blocks are resolved<br>
**Terminal outcomes:** `CONTINUE_NORMALIZATION_AWARE_CACHEABILITY_POC` or
`STOP_NORMALIZATION_AWARE_CACHEABILITY_POC`

## Objective

Test whether BuildOpt can generate safe, owner-reviewed and durable Gradle task
cacheability corrections after explicitly accounting for file-input
normalization. Native Gradle remains the runtime after patch acceptance. The
route must beat optimized native Gradle; merely enabling standard cacheability
does not count as differentiation unless the complete customer-visible
economics qualify.

## Frozen decisions

- DNO v1 remains closed and supplies no v2 evidence row or timing sample.
- The same five exact source revisions are reused only to isolate the detector
  change. Every v2 analysis, patch, build and result is generated again.
- Marker-only actions require portable primary normalization on every file
  input before admission.
- Missing normalization is a separate semantic action. V2 may propose only
  `PathSensitivity.RELATIVE`, after the frozen relocation/mutation proof and
  explicit owner review.
- Repository names, task names and hand-authored customer profiles are never
  detector inputs.
- A failed gate stops dependent blocks. Thresholds are not moved after seeing
  results, and unmeasured value is not recorded as zero.

## Blocks

| Block | Deliverable | Entry gate | Exit gate | State |
| --- | --- | --- | --- | --- |
| `NAC-001` | Frozen normalization model, exact cohort, thresholds, authority and documentation ledger | User-authorized successor to closed DNO v1 | Contract checker rejects drift and DNO remains terminal | `DONE` |
| `NAC-002` | Fresh five-family source classification | NAC-001 | 5/5 conclusive and 4/5 action families after normalization classification; no DNO report consumed and no timing | `DONE` |
| `NAC-003` | Relocation/mutation fixture matrix and digest-bound compiler | NAC-002 pass | Gradle 8.14.3/9.6.1 × Kotlin/Groovy pass; marker-only and reviewed-relative transactions apply idempotently, revert byte-exact and reject ambiguity/drift | `DONE` |
| `NAC-004` | Public-candidate correctness and portability matrix | NAC-003 | Exact outputs, content/path invalidation, cross-root restore, fallback and zero product failures for every admitted candidate | `STOPPED` — 4/9 candidates fully proved; the bounded Groovy run left the complete matrix unmet |
| `NAC-005` | Installed paired value against optimized native Gradle | NAC-004 | Eight balanced pairs per family; frozen value threshold in at least 3/5 families | `NOT_AUTHORIZED` |
| `NAC-006` | Twenty-commit durable value per admitted family | NAC-005 | At least 3/5 positive families, positive signed portfolio and finite payback without BuildOpt on candidate build path | `NOT_AUTHORIZED` |
| `NAC-007` | One-command proposal/review/revert UX and terminal scorecard | NAC-006 or first failed prerequisite | Truthful terminal decision; no unmeasured value or automatic merge authority invented | `DONE` — terminal scorecard only; installed UX was not authorized |

## Action classes

| Action | Admission | Compilation authority |
| --- | --- | --- |
| `ADD_CACHEABLE_TASK_MARKER_V2` | Complete custom-task contract and every file input already has a portable primary strategy | Add fully qualified `@CacheableTask`; owner review required |
| `ADD_RELATIVE_PATH_NORMALIZATION_AND_CACHEABLE_MARKER_V1` | Otherwise-complete task, supported missing file normalization, two-root native equivalence, content/path invalidation, cross-root restore and owner semantic review | Add fully qualified `@PathSensitive(PathSensitivity.RELATIVE)` to the bound declaration and `@CacheableTask`; no other strategy may be inferred |

`PathSensitivity.ABSOLUTE` is a typed non-portable result. Supplementary
normalizers do not satisfy admission without a portable primary strategy.
Classpath semantics must already be declared; v2 never invents them.

## Exact execution path

1. `NAC-002` implements a new v2 scanner or extends the scanner behind an
   explicitly versioned mode. It reads only the five frozen source trees and
   emits source paths, declaration spans, source SHA-256, file-input kinds,
   primary/supplementary normalization and one typed decision per candidate.
2. An independent checker reruns the scanner and counts families from rows,
   not from an aggregate summary. Any v1 report dependency fails the block.
3. `NAC-003` first proves the synthetic normalization matrix. It then compiles
   exact patches only for admitted rows; missing-normalization rows need an
   attached proof record and owner-review token in the local POC evidence.
4. `NAC-004` applies each patch only in disposable external checkouts. It runs
   control, content mutation, relative-path mutation, relocated checkout,
   cache restore and fallback, compares the complete required-output contract,
   and reverts before leaving the checkout.
5. Only a zero-failure `NAC-004` opens local paired timing. Hosted CI never owns
   a wall-time decision.
6. Only qualifying installed value opens the chronological campaign and only
   qualifying chronological value opens installed proposal UX.

## Measurement policy

The control is optimized native Gradle on the same request, exact revision,
toolchain and prepared dependency state. Timing uses a controlled local runner
and balanced alternating arms. Every negative pair is retained with sign.
Percentages from different tasks, repositories or mechanisms are never added.
Patch creation, validation and review costs enter payback.

## Stop conditions

- fewer than five conclusive families or fewer than three action families;
- any inferred normalization outside the one reviewed-relative action;
- any source drift, ambiguous declaration or non-exact revert;
- any product failure, required-output mismatch or invalidation/relocation
  failure;
- paired value below the frozen threshold in three families; or
- longitudinal portfolio value not positive or payback not finite.

The first failed prerequisite closes all dependent blocks as
`NOT_AUTHORIZED`; `NAC-007` records the terminal scorecard.

## Terminal result

`NAC-003` passes its four-row Gradle matrix and all eight marker-only patch
transactions apply idempotently and revert byte-exact. The reviewed-relative
Micronaut candidate also passes native fallback, content and relative-path
invalidation, cross-root restoration and exact output comparison. Fresh
OpenTelemetry and Spring executions restore all exercised task instances from
the shared cache in a second absolute root.

`NAC-004` nevertheless does not pass: only 4/9 admitted candidate classes
received the complete proof. Apache Groovy's `performanceTests` input producer
requires four versions with 350 compile iterations each; the bounded public
run was interrupted during the first version and is not classified as a
product failure. The zero observed product failures therefore remains true,
but it cannot substitute for the frozen every-candidate gate. `NAC-005` and
`NAC-006` are `NOT_AUTHORIZED`, no wall-time sample or speedup exists, and
`NAC-007` records `STOP_NORMALIZATION_AWARE_CACHEABILITY_POC` without
authorizing installed proposal UX, automatic merge, or a successor.

## Documentation ledger

Every block updates this tracker, the machine contract, specification index,
benchmark index, validation reference, implementation tracker, generalization
audit, performance findings and POC one-pager. Source/runtime/CLI changes also
update architecture, onboarding, configuration and troubleshooting.

| Block | Required documentation result |
| --- | --- |
| `NAC-001` | Active-route links, frozen contract, cohort, checker and explicit no-value status |
| `NAC-002` | Candidate counts by normalization class and typed no-action/unavailable reasons |
| `NAC-003` | Proof matrix, patch shapes, authority boundary and transaction evidence |
| `NAC-004` | Per-candidate correctness, relocation and failure accounting |
| `NAC-005` | All raw pairs, signed family results and threshold decision |
| `NAC-006` | Commit sequence, cumulative economics, payback and exclusions |
| `NAC-007` | Installed UX, final scorecard, retained/rejected claims and recommendation |
