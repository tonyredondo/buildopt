# Edge operability v1

This contract closes `POC-O1` by turning the completed `MVP-C2` package into
one owner-operated process without widening Edge authority. Shared remains the
only commit and collision authority.

`buildopt-edge serve --config ABSOLUTE_PATH` validates the strict private
configuration, opens the exclusive durable store, verifies current signed
authority and its Shared credential, advances local anti-rollback state, and
only then opens the configured IPv4 loopback listener. The same process runs
pending replication, byte-SLRU/TTL maintenance, authority reload, private
status publication, and graceful `SIGINT`/`SIGTERM` shutdown.

Authority, trust-root, or credential changes are detected every second. The
route is disabled before replacement verification; invalid, expired, torn, or
rollback input therefore returns byte-free `503` and never uses the old
generation. A fully verified monotonic generation re-enables the cache route.

`buildopt-edge status --config ABSOLUTE_PATH` reads the mode-`0600` aggregate
`edge-status.json` beneath managed state without taking the writer lease. It
returns success only for a `READY` observation no older than five seconds. The
document excludes credentials, paths, cache keys, repository identity, and
administrative HTTP routes.

The signed Linux AMD64 release includes `bin/buildopt-edge`.
`dev/render-edge-service` deterministically creates a private hardened systemd
unit scoped to the configured state directory; linking, enabling, starting,
and stopping that unit remain explicit host-operator actions.

Run:

```bash
./dev/check-edge-operability
```

The gate builds the real command, runs the loopback runtime under the race
detector through replication, invalid reload, higher-generation recovery and
graceful shutdown, verifies aggregate status redaction, renders and validates
the systemd unit twice, and preserves the source tree. It does not execute the
deferred soak, use external design partners, add HA/backups/enterprise
identity, expand platforms, or claim production readiness.
