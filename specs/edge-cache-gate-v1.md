# Edge Cache gate v1

This contract closes `C2-G01` and the owner-operated `MVP-C2` proof only
when the current-tree C2-001..005 contracts and the final real two-node
executable proof pass without changing the source tree.

The composition requires strict loopback/private configuration and immutable
Shared-only authority; complete verified Shared-committed publication and
current signed-revocation checks on every stable read; hard committed-plus-
pending byte accounting, TTL, and probation/protected byte-SLRU; exact-attempt
pending isolation with durable asynchronous replication and restart recovery;
and two independent loopback Edge roots whose collision is decided only by
Shared.

Run:

```bash
./dev/check-edge-cache-gate
```

The checker validates every constituent machine-readable contract directly,
then runs the final nested C2-005 checker, which composes configuration,
committed reads, capacity, pending replication, and the two-node online/offline
proof under the race detector. It also requires the complete operation to
preserve the source tree.

This closes bounded implementation and owner-controlled validation for optional
MVP-C2. It does not run the deferred eight-hour soak, substitute for external
validation, install or manage an operating-system service, hot reload authority
documents, or claim production readiness.
