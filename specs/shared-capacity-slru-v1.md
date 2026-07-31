# Shared capacity and byte SLRU v1

This contract materializes the capacity substrate required by `A1-003` and
`A1-G03` for the private-beta single-node Shared store. It adds hard logical
quotas, durable stable TTL, high/low watermarks, byte-weighted segmented LRU,
and conservative disk admission without making filesystem presence a cache
authority.

## Capacity and admission

The default deployment capacity is the smaller of 500 GiB and 50% of the
local volume. The installer must still require at least 20 GiB before a real
pilot. A repository is limited to 100 GiB, and the same hard boundary is
enforced for each isolated tenant/repository/trust-domain/namespace
generation. Pending admission reserves against a pool shared with quarantine
and capped at 10% of deployment capacity; neither pool can evict stable data.
Newly discovered corrupt evidence is never discarded merely to make a PUT
succeed, so it may cross that admission ceiling, blocks every later PUT, and
expires after seven days. Active-evidence compaction remains with `A1-004`.

A trustworthy nonnegative body length reserves its exact bytes before the
body is read. An absent length reserves the complete configured object limit.
One mutex accounts all in-flight reservations, so concurrent uploads cannot
oversubscribe a pool. Admission checks the maximum object, pending/quarantine
pool, namespace, repository, deployment, and current filesystem availability.
Failure returns `413` through the cache HTTP boundary without reading rejected
bytes. The stream remains independently bounded, so a false length cannot
escape the reservation or object limit.

## TTL, watermarks, and SLRU

Every commit creates a durable `PROBATION` entry with a 30-day expiry. A
complete authenticated and checksum-verified hit promotes it to `PROTECTED`.
Access timestamps coalesce to at most one write per minute after promotion.
Protected data targets 80% of the deployment low watermark; its least-recent
entry returns to probation when that target is exceeded.

At 85% logical usage, eviction frees bytes until usage is at or below 75%.
Expired entries go first, then least-recent probation bytes, then
least-recent protected bytes. The algorithm removes as many entries as their
actual byte weights require; it never evicts a fixed key count. New entries in
the transaction are excluded from satisfying their own admission pressure,
and a commit rolls back if existing safe candidates cannot recover the low
watermark.

Deletion removes metadata authority first. A later exclusive maintenance
step removes only blobs that have no pending or committed reference. Startup
runs reconciliation and this capacity pass before readiness. A five-minute
worker repeats maintenance, while the 30-second operational sampler feeds
logical high-watermark and admission-blocked state into the existing
`DISK_QUOTA` alert.

## Migration and evidence

Schema v4 adds `storage_entries` beside the unchanged committed-object
authority and a control-plane maintenance ledger. The transactional v3→v4
migration reconstructs repository/trust identity from the immutable decision
and attempt, assigns the original commit plus 30 days as expiry, and places
every existing entry in probation. Opening with an authorized reduced TTL
monotonically clamps both migrated and existing expiries; it never extends
previous retention.

Run:

```bash
./dev/check-shared-capacity-slru
```

The race-enabled checker exercises admission before body reads, overlapping
reservations, simulated disk exhaustion, promotion/demotion, byte eviction,
TTL deletion, schema migration, and operational signals. Phase-sequenced
`A1-003`/`A1-G03` remain open until the exact benchmark fault evidence consumes
this substrate; this also does not close the broader soak, beta restart gate,
managed-copy data lifecycle, or external pilot.
