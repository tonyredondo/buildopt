# Tier 1 cache conformance v1

Status: implemented by `A0-G01`.

This contract composes the native Gradle HTTP client, the authenticated local
gateway, the opaque single-node backend, and the default-deny task policy. It
closes the compatibility and protocol matrix without claiming the later
revocation, restart, spool-fault, commit-atomicity, or Test-grant gates.

## Supported matrix

The executable matrix contains ten Linux AMD64 rows:

- Gradle 8.14.3 with JDK 17 or 21;
- Gradle 9.6.1 with JDK 17, 21, or 25; and
- both Groovy and Kotlin DSL for every runtime pair.

JDK 25 is the checksum-pinned Temurin runtime from the repository toolchain
lock. Unknown runtime combinations still disable the managed cache; proving
these rows does not widen the adapter beyond the declared matrix.

## Composed HTTP behavior

The backend returns `200` plus verified opaque bytes for a hit, `404` for a
miss, any accepted `2xx` for an idempotent PUT, and `413` before reading a
known-oversized body. Exact retries do not duplicate content, concurrent
content-addressed publication converges on one object, and a corrupt committed
blob is quarantined and becomes a byte-free `404`. Unknown methods return
`405` with `Allow: GET, PUT`.

Managed mode never follows an upstream redirect. Every `301`, `302`, `303`,
`307`, and `308` becomes a safe `404` for GET or `503` for PUT, and the
credential is never sent to the redirect target. Upstream timeouts use the
same normalization. The gateway preserves `Expect: 100-continue`, waits one
second before sending a body, and passes an early `413` back to Gradle without
reading the rejected payload.

Every matrix row runs Gradle's public `HttpBuildCache` through both DSL
fixtures. A cold request misses and stores; the next clean build restores
`compileJava` from the remote cache while reusing Configuration Cache. A
changed entry rejected with `413` leaves the build successful and cannot
replace an earlier object. Restoring the old source consumes that earlier
object. Normalized read misses and write failures retain the baseline build
and publish no candidate.

## Default-deny boundary

Every row also reruns the restriction policy:

- the exact source-set `JavaCompile` contract may replay;
- a custom cacheable task executes again;
- adding an action to the built-in rejects it; and
- registering an unknown artifact transform disables the managed cache for
  the invocation and forces reinventory.

This closes `A0-G01`. It does not prove L2-to-L1 revocation, gateway rotation,
the complete spool fault matrix, atomic commit/reconciliation, or root and
composite `Test` coverage; those remain `A0-G02..05` and `A0-G08`.

Run:

```bash
./dev/check-tier-one-cache-conformance
```
