# Onboarding performance evidence

This specification measures the command shown to a first-time user, not an
internal launcher configuration:

```bash
buildopt gradle --no-daemon --no-configuration-cache clean :app:distZip
```

Its purpose is to answer two separate questions:

1. Does the default BuildOpt onboarding make repeated clean builds faster when
   reusable outputs were not previously available?
2. What overhead or benefit remains when the control already uses Gradle's
   native local build cache?

## Workloads and arms

The workload uses the immutable public Kotlin and Groovy synthetic pilot
repositories. Each repository is measured against two controls:

- `CACHE_OFF`: `BUILDOPT_BYPASS=1` plus `--no-build-cache`.
- `NATIVE_CACHE`: `BUILDOPT_BYPASS=1` plus `--build-cache`.
- Candidate: the public BuildOpt command above, with no hidden cache flag.

Every arm has its own archived workspace, Gradle user home, and user cache.
One unmeasured run warms that arm. Before each measured run, the harness removes
all project `build/` directories and project-local `.gradle` state. Four
pairs alternate control-first and candidate-first order.

## Required evidence

A result is valid only when:

- every measured command succeeds;
- candidate and native-cache samples restore `compileJava` from cache;
- cache-off samples do not restore `compileJava`;
- every candidate reports the authenticated Gradle plugin handshake;
- the final distribution SHA-256 is identical in both arms of every pair;
- all raw durations, order, repository revisions, runner facts, and the exact
  BuildOpt binary digest remain in the report;
- all four cache-off pairs save time in both pilot repositories.

The native-cache comparison has no positive-result requirement. A negative
value is retained because it exposes the cost of BuildOpt's stricter task
allowlist and launcher verification rather than turning that cost into a
misleading success.

## Interpretation boundary

This is descriptive owner-operated POC evidence. It proves that the published
onboarding command activates the safe local cache and measures its effect on
two controlled repositories. It does not prove savings for every repository,
does not qualify the host as the fixed golden runner, does not run the deferred
eight-hour soak, and does not authorize production promotion.

Run the local evidence validator with:

```bash
./dev/check-onboarding-performance
```

Create a fresh report with the command documented in
[the benchmark index](../benchmarks/README.md). The manual hosted workflow uses
the same runner and uploads its raw JSON result.
