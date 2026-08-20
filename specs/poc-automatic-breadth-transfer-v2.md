# Incremental automatic breadth transfer

This POC repeats the same five frozen public-repository workflows used by the
first automatic breadth transfer, but replaces the synchronous sixteen-build
calibration transaction with evidence learned across ordinary invocations. It
tests whether BuildOpt can deliver structural build reduction through the
customer command while preserving the exact outputs required by that command.

## Customer path

Each repository starts from the public revision and change frozen in
[`poc-generic-profile-matrix-v3.json`](./poc-generic-profile-matrix-v3.json).
One exact installed BuildOpt binary receives seventeen ordinary invocations:
one discovery baseline followed by eight balanced control/candidate pairs.
There is no separate benchmark-only Gradle workflow.

Before every invocation the harness uses Git's ignore rules to delete ignored
workspace outputs inside the verified temporary checkout, while explicitly
preserving `.buildopt`. Tracked source paths are never inferred from directory
names. The private Gradle home and BuildOpt state remain available. This models
a clean CI workspace while giving native Gradle and BuildOpt equal access to
their caches. A candidate may omit work only when BuildOpt can restore the
required unaffected outputs from its verified materialization store.

The declared workflows and Gradle options remain unchanged from V1. The
Micronaut aggregate workflow may additionally use the generic partitioner to
split a large exact-revision candidate set into bounded exact task groups. No
repository name, module name or repository-specific threshold is allowed.

## Evidence and gates

The retained evidence includes all seventeen optimize results per repository,
the final state without materialization blob payloads, hashes for every result
and the terminal decision. The checker recomputes the balanced pairs, output
equality, incremental overhead and break-even result instead of trusting the
summary.

The unchanged value gates require eight complete pairs, identical required
outputs, successful full-graph fallback, at least 500 ms and 2% mean saving, a
positive 95% interval, eight positive pairs and payback within thirty matching
builds. Failure to prove any gate keeps optimized native Gradle authoritative;
native retention is a valid POC result, not a hidden success.

## POC boundary

This experiment proves or disproves transfer on five public repositories. It
does not authorize production activation, average repository percentages or
add the effects of different mechanisms. It requires neither soak testing nor
a design partner. Test Optimization remains outside BuildOpt.
