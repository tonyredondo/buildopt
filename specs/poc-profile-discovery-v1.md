# Deterministic POC profile discovery

## Purpose

This contract turns already-qualified, checked POC evidence into a reviewable
repository profile. It does not search for faster workloads, activate an
optimization, grant production authority, or create a new performance claim.
The only current positive cell is the retained Build Impact plus read-only Edge
mechanism set; every other cell remains on optimized native Gradle.

Discovery is based on contracts and digests, never a repository-name allowlist.
The implementation proves this by rebinding the complete positive fixture to a
different repository identity and obtaining the same mechanism decision.

## Command

```text
buildopt profile discover \
  --manifest PATH \
  --graph PATH \
  --generated-manifest PATH \
  --matrix-summary PATH \
  --cell-evidence PATH \
  --profile-contract PATH
```

The command reads only bounded regular files below the current repository. It
emits one deterministic `buildopt.poc/profile-discovery/v1` JSON document on
standard output and does not write or activate `buildopt-qualified-profile.json`.

## Positive decision

`GENERATED_QUALIFIED_PROFILE` requires all of the following:

1. the terminal matrix binds the exact cell evidence SHA-256 and marks it
   qualified under `SPECIALIZE_QUALIFIED_PROFILES`;
2. at least four pairs are all positive, the lower paired bound is positive,
   mean savings exceed 500 ms and 2%, and no product failure occurred;
3. the retained mechanisms are exactly Build Impact, read-only Edge locality,
   and the qualified source normalization input;
4. exact output, global/native fallback, and HTTP-failure local fallback all
   passed, while Test Optimization stayed disabled;
5. the manifest, graph, generated state, trace evidence, source input, required
   output, repository revision, Gradle version, and JDK are digest-bound;
6. the graph is complete, has one unambiguous reviewed alternative, contains no
   unknown relationship, and the selected entrypoint contains no Test task;
7. the reviewed profile contract matches the evidence mechanism set and exact
   file-SHA precondition.

The emitted document explains every enabled and disabled mechanism, original
and selected entrypoints, required outputs, omitted project count, source
bindings, and review/authority boundary. Its embedded v2 profile must equal the
manually reviewed Kafka profile byte-for-byte after canonical JSON rendering.

## Native fallback decisions

Any uncertainty emits `NATIVE_FULL_GRAPH` with `profile: null`. This includes:

- an unqualified matrix cell, including the fixed Spring and OpenTelemetry
  negative fixtures;
- evidence SHA drift, incomplete trace or safety evidence, a moved value gate,
  or a mechanism-set mismatch;
- an invalid, incomplete, unbound, or drifted manifest/graph/generated state;
- unknown relationships, multiple alternatives, or selected Test tasks;
- source-precondition or reviewed-contract drift.

Native fallback is a successful discovery decision, not an execution error.
Malformed arguments or unreadable/unsafe input paths return configuration or
usage failure instead. Neither result changes the repository or runs Gradle.

## POC boundary and validation

The checked Kafka output is
[`poc-profile-discovery-v1.json`](../benchmarks/results/poc-profile-discovery-v1.json).
It reproduces only the fixed 81.85% Kafka profile from prior evidence; it does
not broaden that percentage to other repositories. Spring stays native despite
14.33% mean savings because one of eight pairs was negative. OpenTelemetry stays
native because it produced zero accepted observations.

Validate deterministic generation, exact profile equivalence, negative cells,
identity independence, invalid usage, and fail-closed drift cases with:

```bash
./dev/check-poc-profile-discovery
```

Soak testing, design partners, production authorization, autonomous activation,
and Test Optimization are outside this proof-of-concept block.
