# Installed structural profile adoption replay

## Question

Can the repository-independent v4 profile reproduce the value already measured
through direct installed Build Impact when a user invokes only
`buildopt poc --changes-file ...`?

This is an adoption test, not a new Micronaut-specific optimization. The
product implementation contains no repository identity, module name, task name
or source path from this experiment. Micronaut Core is the fixed substantial
public workload used to verify the generic flow.

## Frozen comparison

- source: `micronaut-projects/micronaut-core` revision
  `8de8f38aceb6239f7df05c92c4eb7a26113e882b`;
- optimized-native control: root `assemble` with daemon, offline dependency
  state, native build cache, parallel execution and 12 workers;
- candidate: the same installed package executing
  `buildopt poc --changes-file .buildopt-changes` with a deterministic v4
  profile produced by `buildopt profile qualify`;
- only Build Impact enabled; all other BuildOpt mechanisms and Test
  Optimization disabled;
- one unmeasured warm-up per arm and eight alternating measured pairs;
- the identical cache seed, checkout, Gradle user home, daemon, mutation and
  output reset are applied symmetrically;
- all non-empty required JARs must be byte-identical for every pair.

The profile is materialized before timing from the independently checked
qualification evidence derived from the prior direct structural measurement.
Its materialization time is excluded; profile validation, planning and launcher
overhead are included in every candidate observation.

## Value and safety gates

Qualification requires at least 500 ms and 2% mean savings, a positive
deterministic 95% paired lower bound, eight positive pairs, zero
product-attributable failures and identical stable outputs. A global
`gradle.properties` change must execute the full graph. A harmless byte-level
change to the measured graph must also execute the full graph with
`PROFILE_PRECONDITION_FAILED`.

Any failed arm, output mismatch, moved binding or failed threshold stops or
retains native Gradle. Pairs are never discarded and thresholds cannot move
after timing.

## Run and validate

After committing the preregistration on a clean checkout:

```bash
./dev/check-poc-structural-profile-adoption-v1 --contract
./dev/run-poc-structural-profile-adoption-v1 \
  /absolute/path/to/poc-structural-profile-adoption-v1.json
./dev/check-poc-structural-profile-adoption-v1 \
  /absolute/path/to/poc-structural-profile-adoption-v1.json
```

The online dependency preflight is excluded from timing. The measured phase is
offline. No soak, design partner, production authorization, public release,
automatic activation or Test Optimization work belongs to this POC block.
