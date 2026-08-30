# Durable native optimization POC v1

Status: preregistered contract for `DNO-001..007`.

## Hypothesis

BuildOpt may create durable value more reliably by compiling verified source or
build-logic corrections into a repository than by selecting a learned runtime
profile before every build. After an owner accepts a patch, optimized native
Gradle is the only execution engine and BuildOpt contributes no synchronous
runtime cost to the measured candidate.

This is a materially different hypothesis from whole-profile, adaptive-fragment
and verified-request-hit execution. Their evidence may motivate the detector
catalog, but cannot authorize a candidate, supply timing or satisfy a gate.

## Frozen catalog

The first route examines exactly four generic opportunity classes:

1. a custom `DefaultTask` with a task action, at least one declared input and
   output, no `@CacheableTask`, and no explicit caching prohibition;
2. a recurrent exact workflow containing `clean`, provided a cleanless native
   candidate preserves the complete required-output contract;
3. a source-bound declared dependency edge whose removal preserves the complete
   required-output closure; and
4. an `api` dependency in a proven unpublished internal module that is absent
   from its public ABI and whose consumers and metadata remain unchanged.

The first compiler supports only the additive cacheable-task marker. The other
classes remain typed `UNAVAILABLE` until their full source and validation facts
exist; structural opportunity alone never counts as an action.

## Ordered gates

- `DNO-001` freezes this contract and the documentation authority.
- `DNO-002` analyzes one exact revision from each of the five public families.
  All five must be conclusive, and at least three must contain a source-bound
  candidate, before any candidate build or timing.
- `DNO-003` compiles a digest-bound patch, applies it outside the source
  checkout and proves exact revert. It does not merge or activate the patch.
- `DNO-004` validates Gradle 8.14.3/9.6.1 and Kotlin/Groovy fixtures plus every
  admitted public candidate. Required outputs must be exact and the product
  failure budget is zero.
- `DNO-005` runs eight balanced alternating pairs per admitted family against
  optimized native Gradle. A family needs at least six positive pairs, at least
  500 ms and 2% mean saving, a positive lower 95% bound and exact outputs.
- `DNO-006` applies the patch once and replays at least twenty subsequent
  commits per family. At least three families and the signed portfolio must be
  positive, payback must be finite, and BuildOpt must not be on the candidate
  build path.
- `DNO-007` exposes proposal, proof and exact revert through the installed
  one-command experience and issues the terminal decision.

A failed gate closes every dependent block as `NOT_AUTHORIZED`; it is not
reinterpreted as zero value. Hosted CI checks contracts and correctness but
owns no wall-time threshold.

## Non-goals

- repository-name, task-name or hand-authored per-customer profiles;
- automatic patch merge;
- production rollout, soak, design partners or SLOs;
- reopening Runtime Tuning, Hot State, Copy or whole-request execution; and
- Test Optimization.

## Verification

```bash
./dev/check-durable-native-optimization
```
