# Qualified profile lifetime V2

`POC-QUALIFIED-LIFETIME-V2-001` asks whether the five profiles qualified by
the materialization-economics experiment keep producing net wall-time value on
later, ordinary public commits. Qualification speedup is not treated as proof
of useful lifetime.

The frozen machine-readable protocol is
[`poc-qualified-lifetime-v2.json`](./poc-qualified-lifetime-v2.json). The
experiment is a bounded proof-of-concept measurement, not a production
promotion gate.

## Question

For each previously qualified change family:

1. do two independent native builds produce the same required output inventory
   and exact bytes across checkout roots;
2. can an independently installed BuildOpt consumer obtain the profile and its
   materialized outputs from the central HTTPS/CAS path;
3. does the generic selector reuse it only on structurally compatible public
   first-parent descendants;
4. does every selected or native-retained build preserve the exact required
   output inventory and bytes; and
5. does cumulative observed saving repay qualification plus central
   publication cost before the profile becomes inapplicable?

The last question may legitimately answer **no**. Negative value and zero
compatible replays remain evidence; they must not be hidden by averaging
repositories or adding percentages from unrelated mechanisms.

## Subjects

The five subjects and their frozen qualification/observation revisions are
defined in the JSON contract. They represent Spring Framework,
OpenTelemetry Java Instrumentation, Apache Kafka, Micronaut Core and Apache
Groovy. The repository name selects only the test fixture and public history;
product code contains no repository-specific branch.

Each qualification uses the unchanged V2 thresholds, tasks and gates inside
the same fresh-process envelope as its later observations:

- 17 ordinary invocations before calibration;
- eight alternating native/candidate calibration pairs;
- at most 30 projected builds to break even;
- one single-use Gradle process per invocation;
- the same public workflow and exact output boundary used by V2; and
- no weakened confidence, fallback or correctness threshold.

## Central materialization transport

A qualified portfolio may include an output pack larger than the central state
object limit. BuildOpt therefore:

1. validates the manifest and complete pack locally;
2. splits the pack into ordered CAS objects of at most 8 MiB;
3. binds every object digest and size into the signed portfolio snapshot;
4. downloads and verifies every object independently on the consumer;
5. reconstructs the pack atomically and verifies its final SHA-256; and
6. rewrites only the local absolute pack path before normal materialization.

The consumer rejects missing, reordered, oversized, duplicated or corrupt
chunks. A source change in a project whose outputs were materialized also
invalidates replay. Unrelated source changes may continue to the existing
structural and economic selector.

## Native output portability

Qualification speedup alone does not prove that captured outputs are safe to
move between machines. Before any lifetime observation, the producer and an
independent checkout run the same native workflow in different absolute roots.
Their complete required output inventories, modes, sizes and SHA-256 values
must match exactly.

The independent checkout is not connected to BuildOpt or the central cache
until after this comparison passes. Its Gradle home begins empty apart from the
pre-seeded Wrapper distribution, and `--no-build-cache` disables local and
repository-configured remote Gradle caches. A matching digest therefore cannot
be caused by restoring the producer's outputs.

A mismatch rejects the profile as
`REJECTED_NON_REPRODUCIBLE_NATIVE_OUTPUTS` before lifetime observation. The
result records the two aggregate digests, the total difference count, a digest
of the complete difference manifest, zero native cache hits and at most 25
sample paths. It does not drop volatile files, rewrite bytecode or weaken
correctness to manufacture a portable profile. A rejected profile is a
completed POC finding, not a product failure.

## Comparison

For every observation revision, two isolated arms with persistent cache state
receive the same central Gradle-cache opportunity:

- **control** runs `buildopt gradle` with optimized native execution;
- **candidate** runs `buildopt optimize` with the remotely transported
  portfolio and materialization.

The arm order alternates. Project outputs are removed before each measured
invocation while toolchain, Wrapper, dependency, Gradle build-cache and BuildOpt
state remain persistent. Build logic is retained because deleting it would
measure a cold bootstrap rather than ordinary descendant-build lifetime.

Every build uses a fresh Gradle process. This matches the bounded CI-like POC
envelope and prevents one co-located experimental arm from retaining Gradle,
Worker and Kotlin daemons while the other arm is measured. Persistent daemon
state is therefore not part of this comparison; running the arms on one host
must not create memory pressure that a real single build would not experience.
The maximum worker count remains 12 and no tasks or correctness gates are
removed. Each observation records whether its Wrapper still matches the
qualified profile; Wrapper drift must retain native execution.

The candidate must either:

- select `CENTRAL_PORTFOLIO`, run `SELECTIVE_PROFILE` and restore exact
  required outputs; or
- retain `OPTIMIZED_NATIVE` with no partial candidate execution.

Every observation records wall time, cache hits, selection and sync cost,
materialization cost, exact output digest and the native-retention reason.

## Economics

Per subject:

```text
one-time cost = qualification calibration cost + central connect/publication cost
gross saving  = sum(control wall time - candidate wall time)
net saving    = gross saving - one-time cost
```

The first observed break-even build is the earliest chronological observation
whose cumulative saving covers the one-time cost. It is `null` when that never
happens. Conclusions use this precedence:

- zero compatible replays: `NO_COMPATIBLE_REPLAY_IN_OBSERVED_WINDOW`;
- at least one replay and nonnegative net saving:
  `PAID_BACK_IN_OBSERVED_WINDOW`; or
- at least one replay and negative net saving:
  `NOT_PAID_BACK_IN_OBSERVED_WINDOW`.

Timing noise between native arms therefore cannot turn an unused profile into
a paid-back profile.

Results are reported per repository. The aggregate only counts subjects,
observations, exact outputs, selections, fallbacks and conclusions; it does not
produce an average speedup across unrelated repositories.

## Acceptance and boundaries

The block completes when all five qualification attempts pass the frozen
performance contract and receive an explicit portability decision. Portable
profiles observe every listed public descendant in first-parent order, preserve
every required output exactly and retain native execution whenever the profile
is inapplicable. Non-portable profiles stop before lifetime observation and
retain the exact mismatch evidence. Positive net value and universal
portability are findings, not prerequisites for accepting the experiment.

This work does not authorize production, require a soak or design partner, or
change Test Optimization. It validates whether the general mechanism is worth
continuing as a POC.

Run and validate with:

```bash
./dev/run-qualified-lifetime-v2 \
  /absolute/path/to/poc-qualified-lifetime-v2
./dev/check-qualified-lifetime-v2 \
  /absolute/path/to/poc-qualified-lifetime-v2/summary.json
```
