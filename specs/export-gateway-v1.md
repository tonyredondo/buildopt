# Export gateway v1

This specification materializes `A0-008`. It extends the existing atomic
`BUILD_SESSION v1` JSON producer with a bounded private JSONL stream,
deterministic at-least-once delivery, immutable partial recovery, and an exact
stdout export path. It does not put aggregate experiment effects into a build
record; the validated `EXPERIMENT_RESULT` producer remains part of `A0-009`.

## Durable local boundary

The server owns one current-user mode-`0700` directory. Complete and recovered
documents are immutable mode-`0600` regular files published with a same-directory
temporary file, `fsync`, and first-writer hard link. A replay with the same
bytes is retained; different bytes for the same identity are a conflict.

`buildopt-events.jsonl` is a current-user mode-`0600`, no-symlink regular file
bounded to 64 MiB. Each encoded event is at most 1 MiB. The producer appends and
syncs one complete line at a time, so it cannot consume unbounded build volume.
Only an unterminated final line may be truncated during startup recovery; a
malformed newline-terminated event fails closed without altering the stream.

## BUILD_SESSION event sequence

Every completed handoff emits two deterministic `SUMMARY` events:

1. `BUILD_SESSION_OBSERVED` embeds the complete immutable candidate;
2. `BUILD_SESSION_PUBLISHED` binds its final filename and exact SHA-256 bytes.

The envelope contains exactly `eventId`, `buildId`, `sequence`, `occurredAt`,
`emittedAt`, `schemaVersion`, `idempotencyKey`, `profile`, and `payload`.
Identity hashes a domain separator, build ID, sequence, and payload.
`idempotencyKey` is `<buildId>/<sequence>`. Replays append byte-identical
events, consumers deduplicate by `eventId`, and changed reuse of an event ID or
per-build sequence is a conflict.

The observed event is durable before the complete document is published, and
the publication event is durable afterward. If startup finds the immutable
complete document but not sequence 2, it verifies the document against
sequence 1 and appends the deterministic publication event.

## Partial recovery

If startup finds a valid observed event but neither sequence 2 nor its complete
document, it publishes one deterministic-name partial `BUILD_SESSION`.
The recovered document retains only the already-observed facts, changes
`complete` and measurement status to `false`/`PARTIAL`, records
`source: EVENT_REPLAY`, and declares the exact missing range `[2, 2]`. It never
fills missing measurements or includes a later aggregate causal effect.

Existing complete and partial files are strictly decoded, bounded, and checked
against the event that owns them. Recovery never overwrites them. A later
successful identical delivery may publish the complete document and final
event while preserving the earlier immutable partial record.

## File and stdout export

The server's ingest path remains fail-open: export failures are diagnostic and
do not reinterpret the Gradle exit. Operators and CI can copy the validated
stream without mutation:

```bash
buildopt-server export --export-dir /private/buildopt-exports --format jsonl
```

The command writes only JSONL bytes to stdout. Remote delivery, retry/jitter,
encrypted spool/DLQ, destination authorization, retention/deletion execution,
and an `EXPERIMENT_RESULT` producer are separate gates.

## Executable evidence

Run:

```bash
./dev/check-export-gateway
```

The checker validates the exact machine contract, race-enabled unit and server
tests, vet, a static server build, the existing schema/data-lifecycle corpus,
and real Gradle success/failure export. The real binary path proves complete
JSON, a four-event JSONL stream, byte-exact stdout export, schema-valid partial
recovery, private permissions, atomic publication, and credential isolation.
