# Normalization-aware cacheability POC v2

Status: active at `NAC-001`; implementation and timing are not yet authorized.

## Hypothesis

BuildOpt may safely create durable native Gradle value by distinguishing two
different source corrections:

1. add `@CacheableTask` only when every file input already declares a portable
   normalization strategy; or
2. propose `PathSensitivity.RELATIVE` together with `@CacheableTask` only after
   a relocation and mutation proof demonstrates that the checkout root is not
   part of the task's output semantics.

Both actions remain owner-reviewed, source-digest-bound and exactly reversible.
After acceptance, native Gradle is the only execution engine and BuildOpt adds
no synchronous runtime cost to the measured candidate.

This is a new experiment. DNO v1 explains why a normalization-aware detector is
needed, but none of its opportunities, correctness rows or timings can satisfy
a v2 gate. The same five exact public revisions are deliberately reused only to
isolate the detector and compiler change; every v2 report and build must be
generated again from source.

## Normalization model

A file input is marker-only eligible only when its declaration already has one
of these portable primary strategies:

- `@PathSensitive(PathSensitivity.RELATIVE)`;
- `@PathSensitive(PathSensitivity.NAME_ONLY)`;
- `@PathSensitive(PathSensitivity.NONE)`;
- `@Classpath`; or
- `@CompileClasspath`.

`PathSensitivity.ABSOLUTE` is explicit but not portable and is rejected.
`@NormalizeLineEndings` and `@IgnoreEmptyDirectories` are supplementary and do
not replace a primary strategy. BuildOpt never infers `NAME_ONLY`, `NONE`,
`CLASSPATH` or `COMPILE_CLASSPATH`.

The only v2 normalization proposal is
`ADD_RELATIVE_PATH_NORMALIZATION_AND_CACHEABLE_MARKER_V1`. It is allowed only
for an otherwise complete custom task whose missing primary strategy belongs
to an `@InputFile`, `@InputFiles`, `@InputDirectory` or `@InputDirectories`
declaration. Before the compiler can emit it, a task-specific proof must show:

1. byte-exact native outputs from two different absolute checkout roots with
   the same relative input tree;
2. a cache miss and correct output after changing input content;
3. a cache miss and correct output after changing an input's relative path;
4. a cross-root cache restore with byte-exact required outputs; and
5. explicit owner review of the source declaration and proposed semantics.

These checks prove the bounded candidate exercised by the POC. They do not
authorize automatic merge or claim that output equality alone proves arbitrary
task semantics.

## Ordered gates

- `NAC-001` freezes this contract, exact cohort, documentation ledger and stop
  conditions.
- `NAC-002` performs a fresh source audit. All five families must be conclusive
  and at least three must expose an action after normalization classification.
  No DNO report may be consumed as an input row.
- `NAC-003` proves the Gradle 8.14.3/9.6.1 and Kotlin/Groovy normalization
  matrix, then compiles digest-bound marker-only or reviewed-relative patches.
  Apply must be idempotent and revert byte-exact.
- `NAC-004` runs every admitted public candidate. Required outputs, content and
  path invalidation, relocation, cache restoration and native fallback must be
  exact. The product failure budget is zero.
- `NAC-005` runs eight balanced alternating pairs per admitted family against
  optimized native Gradle. A family needs at least six positive pairs, at
  least 500 ms and 2% mean saving, a positive lower 95% bound and exact outputs.
- `NAC-006` applies the accepted patch once and replays at least twenty later
  first-parent commits per admitted family. At least three families and the
  signed portfolio must be positive, with finite payback and no BuildOpt runtime
  on the candidate build path.
- `NAC-007` exposes proposal, evidence, owner decision and exact revert through
  the installed one-command experience and issues the terminal decision.

A failed prerequisite closes dependent work as `NOT_AUTHORIZED`. Unmeasured
value is never presented as zero. Hosted CI validates contracts and correctness
but owns no wall-time threshold.

## Non-goals

- repository-name, task-name or customer-specific detector rules;
- silent normalization inference or automatic patch merge;
- production rollout, soak, design partners, SLOs or autonomous promotion;
- reopening Runtime Tuning, Hot State, Copy or whole-request execution; and
- Test Optimization.

## Verification

```bash
./dev/check-normalization-aware-cacheability
```
