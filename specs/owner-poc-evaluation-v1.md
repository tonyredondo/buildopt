# Owner-operated POC evaluation v1

Status: implemented by `A1-006`.

This evaluation uses two public synthetic repositories owned by the project:
[`tonyredondo/buildopt-pilot`](https://github.com/tonyredondo/buildopt-pilot)
and [`tonyredondo/buildopt-pilot-groovy`](https://github.com/tonyredondo/buildopt-pilot-groovy).
They intentionally exercise the same eight-project Java 17 workload through
Kotlin and Groovy Gradle DSLs at immutable revisions.

For each repository, four pairs alternate which arm runs first. Assignments
are persisted before outcomes. Control uses the authenticated BuildOpt Tier 1
path with Gradle build cache disabled; candidate uses the same launcher and
plugin with a pre-warmed isolated managed L1. Every measured run starts with
project outputs and project-cache state removed, uses a single-use daemon, and
produces `app/build/distributions/app-1.0.0.zip`.

The gate requires positive mean savings, a positive lower 95% paired-bootstrap
bound, a non-positive customer-visible p95 delta, zero build-failure delta,
zero product-attributable failures, and byte-identical required deliverables
in both repositories. Cache-hit rate alone cannot pass.

Run locally with both sibling checkouts:

```bash
./dev/run-owner-poc-evaluation \
  /tmp/buildopt-owner-poc-evidence \
  ../buildopt-pilot \
  ../buildopt-pilot-groovy
```

The `Owner POC Evaluation` workflow repeats the exact measurement on a public
`ubuntu-24.04` runner and uploads the immutable result. The checked-in evidence
is validated by `./dev/check-owner-poc-evaluation`.

This closes `A1-006` and `A1-G06` for the owner-operated proof of concept. It
does not claim external design-partner evidence, execute the deferred
eight-hour soak, or authorize production promotion.
