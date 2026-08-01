# Edge Cache configuration and authority boundary v1

This contract closes `C2-001` for the owner-operated proof of concept. One
private declarative file selects a loopback single-node Edge, one authenticated
Shared origin, bounded local storage, and the authority rules that every later
Edge block must preserve.

## Fixed authority boundary

- Shared is the only commit and collision authority.
- Offline reads are limited to locally verified `COMMITTED` objects while the
  authenticated cumulative revocation snapshot is current.
- Offline writes remain pending and visible only to their originating attempt.
- Compression stays disabled until measurements justify a byte-preserving
  policy.

These values are declarations checked by the production parser, not feature
flags. An operator cannot relax them in configuration.

## Configuration and transport

The configuration is a regular non-symlinked mode-`0600` file no larger than
64 KiB. JSON decoding rejects unknown fields and trailing documents. It embeds
no credential or trust material: the Shared token, trust root, and current
authority snapshot are referenced through absolute disjoint paths outside the
managed Edge state directory.

The POC listener is exact IPv4 loopback. Shared uses an origin-only HTTPS URL;
plain HTTP is accepted only through an explicit exception for
`127.0.0.1`, which supports isolated same-host conformance fixtures without
creating a remote clear-text mode.

Storage is constrained to a proven-local filesystem declaration, 1–500 GiB,
objects no larger than 100 MiB, stable TTL no longer than 30 days, pending TTL
no longer than 24 hours, and fixed 85/75 watermarks with an 80% protected
segment target. `C2-003` implements and proves those runtime semantics.

Run:

```bash
./dev/check-edge-cache-config
```

This slice does not open a listener, serve an object, create stable authority,
replicate a pending write, close `C2-G01`, or claim the deferred soak and
productization validation.
