# Sticky wrapper longitudinal campaign v1

Status: accepted POC measurement contract for `SWL-015`.

## Purpose

This protocol measures the customer command that is actually committed to a
repository: `./buildoptw <the existing Gradle workflow>`. It compares that
candidate with the same workflow executed by the repository's native Gradle
Wrapper with build caching enabled. The five repository families and their
chronological commits are frozen by the existing cohort manifest before any
timing is accepted.

This is a proof-of-concept experiment. It can show value, parity or a
regression; it does not authorize production rollout, a design partner, an
eight-hour soak or Test Optimization.

## Fair comparison

For each revision the control and candidate use separate linked worktrees,
user homes, Gradle homes, native build-cache directories and BuildOpt state.
The
first pair runs control first and the next pair runs candidate first, then the
order alternates. Dependency and Wrapper preparation runs before the timed
pair and is recorded separately; it never copies task outputs, native task
caches or candidate decision state into either arm.

The candidate is not invoked through `buildopt optimize` or a repository
specific profile. The only BuildOpt entrypoint is the committed `./buildoptw`
file, with the exact workflow arguments from the frozen cohort. The wrapper
distribution is checksum verified and preloaded into the user cache solely to
make the run offline and repeatable.

Each successful arm records an external monotonic duration and a manifest of
the required output files. A pair is comparable only when both arms exit
successfully and the output manifests are byte-identical. Positive and
negative deltas, native failures, dependency exclusions and incomplete sample
counts remain visible. A bounded run is reported as `INCOMPLETE`, never as a
terminal decision.

## Running it

The default runner attempts all 20 primary commits per family and consumes a
frozen reserve only when a primary is excluded for a permitted reason. For a
quick diagnostic, limit each family without changing the frozen manifest:

```bash
SWL_MAX_PRIMARY_PER_FAMILY=1 \
  ./dev/run-sticky-wrapper-longitudinal /absolute/output/directory
```

The output directory contains `raw.json` and `report.json`. The independent
checker accepts both complete and bounded reports, but only a complete report
with at least 15 comparable pairs in every family may be passed to `SWL-016`.

## Acceptance and limits

The full campaign is ready for the terminal decision only when all five
families have at least 15 comparable pairs, every accepted pair has exact
required outputs, no product-attributable failure is hidden, and the raw
records bind the BuildOpt source revision, package digest, cohort digest and
contract digest. The campaign does not combine percentages across families and
does not claim that a cache hit alone is BuildOpt acceleration.
