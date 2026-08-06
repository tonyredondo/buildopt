# Standard Jar producer cache POC

## Purpose

Gradle deliberately treats the built-in `Jar` task as not worth caching by
default. On the pinned OpenTelemetry workload, however,
`:testing-common:jar` repeatedly packages a large stable input set and accounts
for roughly 3.5 seconds of candidate execution. This POC makes only an
unmodified standard `Jar` producer eligible for Gradle's native local build
cache so the installed Build Impact path can avoid that repeated work.

## Activation and boundary

The adapter is opt-in through:

```text
buildopt impact ... --cache-standard-jar-producers
```

The launcher translates that flag to private child context, loads the packaged
init script and plugin, and enables Gradle's native build cache. It does not
start Managed L1, a gateway, a handshake or telemetry unless another explicit
integration requires them. `BUILDOPT_BYPASS=1` removes the adapter with the
rest of BuildOpt, and combining the flag with `--no-build-cache` is rejected.

Eligibility is intentionally narrow. A task must have all of these exact
runtime properties:

- concrete class `org.gradle.api.tasks.bundling.Jar_Decorated`;
- direct superclass `org.gradle.api.tasks.bundling.Jar`;
- exactly one action; and
- that action is Gradle's `StandardTaskAction`.

Custom `Jar` tasks with additional actions, `Copy`, `JavaExec`, arbitrary
tasks, artifact transforms and every Test-owned task remain unchanged. The
adapter only adds cache eligibility; Gradle continues to calculate the cache
key, fingerprint inputs and outputs, serialize bytes and reject invalid hits.

## Executable evidence

Run:

```bash
./dev/check-poc-standard-jar-cache
```

The TestKit fixture proves an exact standard-Jar replay while a custom `Jar`
and a `Copy` task execute again. The installed OpenTelemetry diagnostic also
proved that the explicit Build Impact candidate restores
`:testing-common:jar FROM-CACHE` without starting the managed runtime.

This is a POC optimization, not a production-wide cache policy. It does not
change the Tier 1 safe-cache allowlist, Test Optimization, production
selection, soak scope or deployment readiness.
