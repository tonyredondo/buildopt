# Adaptive state portability POC protocol

This protocol closes `AF-012` by persisting the same typed adaptive portfolio
and economic ledger locally and through BuildOpt's existing repository-scoped
HTTPS state plane. It proves portability and safe fallback only. It does not
add activation authority, a customer command or a performance claim.

## Local-first generation

The local store contains immutable RFC 8785 JCS documents for one fragment,
its append-only observations, the current portfolio and its economic ledger.
One mode-`0600` canonical head is the only mutable pointer. Its external
SHA-256 binds every document, semantic compatibility and the exact generation
directory. A writer must present the current head digest and can advance only
to the next portfolio and ledger generation with exact supersession links.

An exact replay is idempotent. A stale writer, a concurrent writer, an invalid
generation, a symlink, an unsafe permission, a changed byte or an impossible
cross-document lifecycle fails closed. No recovery rewrites or partially
repairs the previous generation.

## HTTPS representation

The existing central state protocol remains the sole cross-machine envelope:

- `EVIDENCE` carries the fragment, observations and economic ledger;
- `PORTFOLIO` carries the portfolio and the exact local head;
- the portfolio manifest references the exact evidence manifest;
- both kinds share repository, compatibility and binding digests; and
- every downloaded document is decoded, linked, canonicalized and compared
  with a freshly prepared local head before persistence.

The central head advances through exact-generation CAS. A losing concurrent
writer observes `CONCURRENT_REMOTE_WON` and retains the verified winner. A
clean second machine may restore the linked generation. A machine with a
verified local generation may continue offline; a clean machine without a
valid local or downloaded snapshot retains native Gradle.

## Independent planes

Gradle cache objects and BuildOpt control state remain different protocols.
Adaptive documents use repository-scoped `/state/` routes, typed manifests,
qualification references and state retention. They never use `/cache/`, never
become Gradle cache keys and cannot acquire authority from blob presence.
Gradle cache eviction is an ordinary miss; adaptive state corruption removes
the candidate and selects native behavior.

## Executable gate

```bash
./dev/check-adaptive-state-portability
```

The gate proves exact local round trips, optimistic local and remote
concurrency, private files, corruption rejection, a TLS producer and clean
consumer, exact second-machine bytes, verified offline reuse, clean-machine
native fallback and zero cache-plane requests. Passing yields
`ADAPTIVE_STATE_PORTABLE`.

Production hardening, soak, design partners, public onboarding and Test
Optimization remain outside this POC block.
