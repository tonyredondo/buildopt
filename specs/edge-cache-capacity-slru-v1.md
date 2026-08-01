# Edge Cache capacity and byte-SLRU v1

This contract closes `C2-003`. The owner-operated Edge bounds committed local
storage with conservative reservations, a hard byte quota, durable TTL, and a
byte-accounted segmented LRU without acquiring commit or collision authority.

## Admission and reservation

Before reading an accepted Shared body, Edge reserves its exact declared
`Content-Length`. The serialized admission check requires committed logical
bytes plus all outstanding reservations plus the new object to remain at or
below configured capacity. A failed reservation reads no body and publishes
nothing. Every reservation is released exactly once on success or failure.

Publication rechecks the hard quota transactionally, accounting for replacement
of the same authority-bound key. New or replaced committed entries enter
`PROBATION`; a verified local hit promotes the exact current entry to
`PROTECTED`.

## Byte-SLRU and maintenance

The fixed production policy uses a 85% high watermark, 75% low watermark, and
80% protected-byte target. When logical committed usage reaches the high
watermark, Edge deletes metadata in deterministic order: oldest `PROBATION`
first, then oldest `PROTECTED`, until usage is at or below the low watermark.
The entry being published is excluded from its own pressure eviction; if the
target cannot be reached, the publication transaction rolls back.

When protected bytes exceed their target, the oldest protected entries are
demoted to probation until the target is met. TTL maintenance deletes expired
authority metadata before deleting unreferenced physical blobs. Content-address
deduplication preserves a blob while any committed entry still references it.
Capacity snapshots and maintenance reports expose byte and object counts without
changing authority.

Schema v1 metadata migrates transactionally to schema v2 by placing existing
entries in probation with their cached time as the last-access time. Future
schema versions remain rejected.

Run:

```bash
./dev/check-edge-cache-capacity-slru
```

The checker runs the complete Edge race suite, including deterministic reduced
capacity fixtures for pressure behavior, plus the real Shared committed
read-through regression and focused vet. This slice does not implement pending
writes or replication, an operator listener, the final C2 gate, soak, or
production hardening.
