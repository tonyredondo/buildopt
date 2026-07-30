# Gateway verified spool v1

This contract closes `A0-G04` for the A0 local verifying gateway. A successful
upstream cache `GET` is never streamed directly to Gradle. The gateway first
materializes the complete response in a private spool, checks its size and
SHA-256 binding, syncs it, unlinks its pathname, and only then emits downstream
`200` and the verified bytes.

## Limits and verification

One object is limited to 100 MiB. The process-wide verified-read spool has a
separate 200 MiB quota. A valid known `Content-Length` reserves its declared
bytes atomically; an absent length reserves the full object limit. Actual
bytes remain independently bounded, so the header is an admission hint rather
than an integrity boundary. Concurrent reservations cannot oversubscribe the
quota and every terminal path releases its reservation.

The upstream success response must bind the payload through a quoted canonical
`sha256:` ETag. If `X-BuildOpt-Blob-Digest` is present it must match. Before
downstream `200`, the gateway has received the complete body, matched the
declared length, enforced the object limit, matched SHA-256, fsynced the
mode-`0600` current-user file, rewound it, removed its name, and fsynced the
mode-`0700` spool directory. Missing or inconsistent metadata, quota pressure,
I/O failure, cancellation, truncation, or corruption becomes a byte-free
`404`, preserving Gradle's safe cache-miss fallback.

## Fault and recovery evidence

The race-enabled conformance suite proves:

- an injected `ENOSPC` while writing returns a byte-free miss and leaves no
  file or reservation;
- two overlapping reservations admit only the request that fits;
- cancellation after partial receipt removes the partial file;
- a checksum mismatch discovered only after the complete body produces no
  earlier status or bytes; and
- startup removes a private stale `.get-*` file before the real
  `buildopt __managed-gateway` process accepts registration, after which a
  verified hit succeeds.

The spool accepts no symlink, directory, foreign-owner, multi-link, public, or
unexpected-name entry. A verified file is unlinked before serving, minimizing
the crash window; startup cleanup is idempotent for the remaining pre-unlink
window.

Run:

```bash
./dev/check-gateway-spool
```

This gate does not prove the `cache.sqlite` commit transaction and recovery
matrix (`A0-G05`), overhead (`A0-G06`), the no-grant Test rule (`A0-G08`), or
the later flood/circuit-breaker and fail-closed beta restart gates
(`A1-G02`/`A1-G04`). `MANAGED_SHARED_CACHE` remains unavailable.
