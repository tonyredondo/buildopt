# Data lifecycle, redaction, and export v1

This specification materializes `F0-037`, `PRIVACY-001`, and the Phase 0
fixture contract for `EXPORT-001`.

## Export profiles

Profiles are monotonic only after explicit authorization:

- `SUMMARY`: outcome, aggregate timings, cache summary, and savings;
- `TASKS`: summary plus keyed task identifiers and critical-path detail;
- `EVIDENCE`: tasks plus fingerprints, qualification, and action ledger;
- `DIAGNOSTIC`: time-limited opt-in troubleshooting fields.

An exporter may reduce but never expand its authorized profile. No profile
includes source content, secret values, or raw high-cardinality paths,
arguments, cache keys, or fingerprints as labels. Sensitive identifiers are
tokenized before buffering with HMAC-SHA-256 and a declared
`tokenKeyVersion`; a plain digest is forbidden.

## JSON and JSONL

Canonical JSON exports retain their owning schemas. JSONL delivery is
at-least-once. Every line has `eventId`, `buildId`, monotonically increasing
per-build `sequence`, `occurredAt`, `emittedAt`, `schemaVersion`, and
`idempotencyKey`. Consumers deduplicate byte-identical events by `eventId`;
changed reuse is a conflict. Gaps create an explicit partial range rather than
invented completeness.

Redaction occurs before persistence in a bounded encrypted spool. Retry is
bounded. A full spool drops the oldest diagnostic records first, records loss,
and retains the final summary where possible; it never consumes unbounded
build volume. Export failure does not fail Gradle unless a separately
authorized strict compliance gate says so after Gradle closes.

## Private-beta lifecycle

The machine-readable
[`data-lifecycle-v1.json`](./data-lifecycle-v1.json) fixes the beta defaults:
stable blobs 30 days, pending at most 24 hours, quarantine 7 days, optimization
evidence and summary 30 days, diagnostic 7 days and opt-in, local spool/DLQ 24
hours, and minimized security audit 90 days. Metadata/tombstones outlive their
blob by seven days and are not removed before it.

Deletion revokes managed access immediately, then asynchronously removes
physical blobs, metadata, managed L1/volumes, evidence, and spool content.
Managed L1 rotates its security generation. Customer-controlled downstream
copies are outside that physical guarantee; they receive a tombstone and keep
their own obligation. There is no silent legal hold or backup/replica SLA in
the single-node beta.

## Fixtures

[`fixtures/data-lifecycle`](../fixtures/data-lifecycle/README.md) contains one
synthetic raw input, four exact redacted profile outputs, an at-least-once
JSONL stream with a duplicate and gap, and a conflicting duplicate stream.

Run:

```bash
./dev/check-data-lifecycle
```

The checker recomputes every keyed token, rejects raw sensitive values from all
managed outputs, verifies profile monotonicity, validates JSONL
deduplication/partial recovery, and executes the deletion cases.
