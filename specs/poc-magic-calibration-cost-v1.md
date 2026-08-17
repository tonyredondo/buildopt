# Generic calibration-cost reduction

## Purpose

This POC contract reduces the first-decision cost of automatic structural
calibration without reducing the number of measured pairs, weakening output
equivalence, hiding task-shape drift or skipping native fallback.

## Immutable reuse

Calibration may reuse existing inputs only under content bindings:

- the authoritative Gradle dependency cache is read-only and hashed before and
  after use; locks, garbage-collection files, links and non-regular files are
  excluded or rejected;
- the native Gradle build cache is copied once into an immutable seed and
  hard-linked into private measurement arms;
- control and candidate still have separate Gradle homes, project checkouts,
  mutable caches and outputs.

This removes repeated multi-gigabyte copying while preventing one arm from
writing through another arm's cache state.

## Stabilization and measurement

The current policy is `PAIRED_BOUND_CACHE_MEASURED_SHAPE_V1`:

1. the control receives one unmeasured `help` invocation to initialize its
   daemon and repository configuration;
2. the candidate receives no unmeasured target invocation;
3. pair one is measured and establishes complete control/candidate task
   fingerprints;
4. all seven later pairs must reproduce those fingerprints exactly;
5. pair one remains in the statistics;
6. all eight pairs remain alternating and must preserve required outputs;
7. the full native fallback must succeed.

Legacy evidence remains readable, but only runs using the same policy and
source bindings are valid before/after cost comparisons.

## POC acceptance

The Apache Beam calibration experiment passes this block when it:

- reduces comparable first-decision cost by at least 94.1 seconds;
- retains eight pairs, exact outputs, stable task shapes and successful native
  fallback;
- has a positive paired interval, lower candidate p95 and 8/8 positive pairs;
- repays within 30 matching builds.

This block does not complete the public end-to-end gate. A fresh published
package must still repeat economically qualified Ktor and Beam decisions from
clean install-to-decision state. No result authorizes production activation,
cross-revision inference or Test Optimization.

The recomputable evidence is stored under
[`benchmarks/results/poc-magic-calibration-cost-v1`](../benchmarks/results/poc-magic-calibration-cost-v1/README.md).
