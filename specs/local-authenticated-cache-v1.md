# Locally authenticated cache authority v1

This specification materializes `A0-006`. It composes the Tier 1 Gradle
restriction, generation-segmented native L1, single-node Shared storage, and
pending publication behind one locally authenticated policy and cumulative
revocation boundary. It does not close the broader A0 HTTP fault matrix,
production commit-decision flow, or revoked-directory deletion gates.

## Authority and anti-rollback boundary

The policy producer signs one exact JCS
`buildopt-local-cache-authority/v1` document with Ed25519. The launcher and
`buildopt-server` accept the deployment public key only from an out-of-band,
current-user-owned mode-`0600` trust-root file. The authority, trust root, and
32-byte base64url cache credential are private bounded regular files; final
symlinks, group/world permissions, unknown JSON members, non-canonical bytes,
unknown keys, tampering, invalid component ranges, and expired documents are
rejected.

The signed document binds the repository identity, source state, cache
contract, exact attempt and credential digest, policy/configuration digests,
namespace generation, revocation epoch, L1 security generation, gateway
generation, permissions, and all expirations. The launcher also requires the
signed repository identity to equal its local deployment configuration before
starting Gradle.

Both launcher and Shared persist the highest authenticated state. Exact replay
is idempotent. Policy, revocation, namespace, L1, or gateway rollback is
rejected; reusing a generation with different content is rejected; advancing
the revocation epoch requires a strictly newer L1 generation. Losing or
corrupting this state never derives authority from cache blobs.

## Bound Shared route

`buildopt-server serve` exposes `/cache/{key}` only when `--state-dir` and all
three authority flags are present and valid. Schema v3 adds the monotonic
authority state, canonical authority documents, and attempt-to-authority
binding. The raw credential is never written to SQLite.

Every request must present both the exact Bearer credential and
`X-BuildOpt-Authority-Digest`. Before delegating to the A0-005 handler, Shared
checks that the authority is still the current unexpired state and that a
write-enabled attempt retains the same durable authority binding. A newer
revocation or policy state immediately invalidates an older handler. Reads
still expose only completely verified committed objects; writes still target
only the signed pending attempt.

## Local gateway and Gradle

Gradle sees only the stable loopback gateway and its local Basic credential.
The launcher sends the upstream credential and authority context to a managed
gateway over its same-UID Unix control connection; neither the child
environment nor the persistent gateway-state file contains that upstream
credential. Closing the registration removes the context, and the gateway
routes no cache request without one.

For a valid context, the gateway accepts only `GET` and `PUT` on
`/cache/{key}`, replaces the local credential with the Shared Bearer credential,
adds the authority digest, strips all other incoming authority, and rejects
redirects. Upstream read errors, authorization failures, timeouts, and
redirects become a safe `404` miss. Upstream write failures become `503`, so
Gradle disables the remote cache while the build continues. Only bounded
cache response headers cross back to Gradle.

The existing `dev.buildopt.managed-l1` settings plugin configures the public
`HttpBuildCache` API only when every launcher-owned marker is complete and
valid. Read-only authority enables remote reads with `push=false`. A trusted
write attempt enables remote push and disables native L1 so an aborted pending
upload cannot leave a reusable local hit. Authority, policy, configuration,
and gateway generations are Configuration Cache inputs. The Tier 1
default-deny task policy remains active.

## Failure behavior

An absent authority preserves the neutral launcher and the pre-existing
native-only A0-003 path. Any partial, invalid, incompatible, expired,
wrong-repository, or rolled-back authority produces a diagnostic and starts
the requested build without the authenticated Shared route. It never falls
back to parent-supplied generations after an authority configuration was
attempted.

## Executable evidence

Run:

```bash
./dev/check-local-authority
```

The checker validates the machine contract, then runs race-enabled Go tests
for canonical Ed25519 authority, private files, anti-rollback state, schema-v3
installation, current-authority server routing, local credential translation,
managed-process context release, and safe read/write failures. It also runs
the golden Gradle/JDK row through a real `HttpBuildCache` PUT, GET hit, Tier 1
policy, disabled writer L1, and Configuration Cache reuse.

`./dev/check-l1-l2-revocation` composes this authority with the real pending,
commit, revocation, and abort backend plus Gradle's native L1 lifecycle. Full
production commit fault/recovery remains `A0-G05`.
