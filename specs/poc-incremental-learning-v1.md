# Incremental ordinary-build learning

This POC replaces the synchronous control/candidate calibration transaction
inside `buildopt optimize` with exact-bound observations collected across
ordinary customer invocations. It answers one narrow question: can BuildOpt
learn whether a structural candidate beats optimized native Gradle without
running extra copies of the customer's workflow solely to measure it?

## Invocation contract

The first successful invocation remains a full-graph Gradle build. BuildOpt
adds one output-observation task to that same invocation, then performs only
configured-model inspection outside the customer workflow. It records a
baseline bound to the repository, revision, Wrapper, executable, Gradle argv,
change set, generated graph and required-output digest.

Later exact-bound invocations alternate eight adjacent pairs:

1. odd pairs run control then candidate;
2. even pairs run candidate then control;
3. each command executes the requested workflow exactly once;
4. no warm-up or measurement-only customer workflow is launched; and
5. any binding drift starts a new generation rather than borrowing evidence.

Candidate invocations use the same generic structural plan produced by Build
Impact. Their required outputs must match the baseline digest. A candidate
failure, missing output or byte drift triggers a full-graph recovery for that
customer command and permanently retains native Gradle for that generation.
The recovery is correctness work, not a learning observation.

A user cancellation is not a candidate failure. BuildOpt forwards the signal,
returns the child's cancelled status, starts no recovery build and retains
native Gradle for the generation. This prevents a cancelled command from
silently becoming a second customer workflow.

## Qualification and economics

After sixteen observations, BuildOpt renders the existing structural evidence
format and recomputes the unchanged gates:

- eight complete balanced pairs;
- exact required-output equality;
- successful full-graph controls and fallback;
- at least 500 ms and 2% mean saving;
- a positive 95% interval and eight positive pairs;
- no product-attributable failure; and
- payback within 30 matching builds.

Calibration cost now contains only incremental BuildOpt overhead observed
around those customer builds, including first-invocation discovery. It does
not charge the control or candidate wall time as an extra transaction because
those invocations delivered the customer's requested build.

## Executable fixture

[`dev/run-incremental-learning`](../dev/run-incremental-learning) creates a
clean three-project Git/Gradle fixture and invokes the exact installed binary
seventeen times: one discovery baseline plus sixteen alternating observations.
The evidence must show nine full-graph workflows, eight candidates, one output
observation and zero measurement-only workflows. The candidate omits one
independent project while preserving the exact required JAR.

The fixture is deliberately too small to promise a speedup. Qualification is
not required; correct native retention is valid evidence. The purpose of this
block is to prove the learning transaction and its economics, not to tune a
synthetic result.

## POC boundary

This work adds no repository-name rule, production authority, soak or
design-partner requirement. Test Optimization remains out of scope. Required
outputs that happen to remain in a dirty/warm workspace are not yet sufficient
proof of clean-workspace materialization; that separate limitation is owned by
the next verified-output-materialization block.
