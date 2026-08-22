# Native volatility quarantine POC v1

## Purpose

This contract tests whether BuildOpt can retain exact output transport when a
small part of a Gradle build is not byte-reproducible across independent native
workspaces. It does not normalize volatile files or treat semantic similarity
as byte equality. Instead, it makes the Gradle producer task the atomic safety
boundary: a producer with any volatile output is rebuilt locally and none of
its outputs may be present in the transported pack.

The block is a POC capability and correctness result. It does not authorize a
production rollout or retroactively turn a historical calibration into a
selected replay or a performance result.

## Inputs

Two independent native builds provide observations with:

- the same repository, revision, workflow and environment binding SHA-256;
- the complete same canonical repository-relative, `/`-separated output-path universe;
- a SHA-256 for every regular output; and
- one or more observed Gradle producer tasks for every output.

The comparison is bounded to 250,000 files, matching the existing verified
materialization POC. Missing paths, duplicate paths, missing producers,
producer disagreement, binding drift and malformed hashes all retain native.

## Decision

For every output path:

1. Compare the two native SHA-256 values exactly.
2. For a difference, add every attributed producer task to the quarantine.
3. Exclude every output of every quarantined producer, including outputs whose
   bytes happened to match in the two observations.
4. Keep only outputs from non-quarantined producers in the transport plan.
5. Bind the ordered transported path, digest and producer inventory to a new
   SHA-256.

The decision is `TRANSPORT_READY` only after the complete comparison. Its
reason is `INDEPENDENT_NATIVE_OUTPUTS_EXACT` when no producer is quarantined or
`VOLATILE_PRODUCERS_QUARANTINED` otherwise. Unsafe or incomplete evidence
returns `NATIVE_RETAINED` with no transport entries.

## Candidate verification

A candidate must provide two disjoint observations:

- transported outputs, with the exact path, digest and producer attribution
  recorded by the first native build; and
- locally rebuilt outputs, covering the complete output set of every
  quarantined producer.

BuildOpt requires byte equality only for outputs it transports or reuses.
Locally rebuilt volatile outputs may differ from either native observation,
but they must exist and remain attributed to a quarantined producer. Missing,
extra or modified transported outputs fail before the candidate is accepted.

## Executable proof

The generic fixture has two producers and four outputs. One of two outputs from
the volatile producer changes between native observations. The mechanism:

- quarantines that producer and both of its outputs;
- transports the two outputs from the stable producer;
- accepts the candidate only when both quarantined outputs are rebuilt locally;
- rejects a modified transported digest; and
- retains native for path, binding or producer ambiguity.

Run:

```bash
./dev/check-native-volatility-quarantine
```

## Public finding retained

The frozen Spring JMS breadth evidence compared 8,385 class outputs across two
independent native roots and found 14 differing files: 13 under the main
AspectJ compiler output and one under the main Java compiler output. Existing
Gradle output-contract evidence maps those output patterns generically to
`:spring-aspects:compileAspectj` and `:spring-messaging:compileJava`.

This demonstrates that the historical rejection has the shape the generic
mechanism is designed to repair. The compact evidence does not retain the full
per-file producer inventory for that exact revision, so it does not authorize
a filtered Spring pack or a replay. The next experiment must recapture the
complete producer-bound observations at its preregistered public revisions,
apply the quarantine, rebuild both producers locally and then measure a
structurally compatible descendant.

## Boundaries

- POC only; `productionAuthorized=false`.
- Exact bytes remain the default transport rule.
- No repository name, filename or extension appears in product decisions.
- No percentage is added to earlier mechanism results.
- No performance or broadened customer-value claim is made by this block.
- Soak testing and design-partner evidence are not required.
- Test Optimization remains out of scope.
