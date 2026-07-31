# Test cache isolation v1

This contract closes `A0-G08`. It proves that the managed Tier 1 policy makes
every Gradle `Test` task in the root build, actual `buildSrc` build, and an
included plugin build ineligible to consume or produce Build Cache entries
when no prior `TestCacheGrant` exists.

## Boundary

The rule matches the `Test` type hierarchy, not names or task paths. It is a
restriction on the already-selected managed cache and cannot make an
otherwise ineligible task cacheable. The explicit diagnostic is:

```text
BuildOpt Test task has no TestCacheGrant
```

The A0 implementation has no positive runtime grant path. Signed grant
resolution and activation remain a later Test Optimization integration; an
unknown, missing, expired, or incompatible grant therefore has the same
fail-closed result as absence.

## Executable proof

The Kotlin DSL fixture contains a root Java build, an actual `buildSrc`
plugin build, and a composite included plugin build. All three expose a real
cacheable `Test` task. A loopback `HttpBuildCache` observer stores exact bytes
and records every authenticated `GET` and `PUT`.

The unguarded control first populates the cache, removes only test outputs,
and restores each `Test` task as `FROM_CACHE`. This proves the cache and
fixture can both consume and produce entries. The no-grant path then applies
the packaged policy through the same init-script boundary, removes test
outputs, and executes the root and included tests twice plus `buildSrc`
directly. Every task remains `SUCCESS`, the diagnostic is present for each
scope, Configuration Cache is reused, and the observer records zero requests.

Run:

```bash
./dev/check-test-cache-isolation
```

This gate does not authorize a positive `TestCacheGrant`, choose tests, alter
their graph, or claim Test Optimization private-beta availability.
