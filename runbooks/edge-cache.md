# Owner-operated Edge Cache

This runbook operates the bounded Linux AMD64 `POC-O1` process. Shared remains
the only commit and collision authority. This procedure does not provide HA,
backup RPO/RTO, enterprise identity, external validation, or the deferred
eight-hour soak.

## Prepare private inputs

Start from [`specs/edge-cache.example.json`](../specs/edge-cache.example.json).
Place the configuration, Shared credential, trust root, and current signed
authority in disjoint current-user-owned mode-`0600` regular files. Use a real
mode-`0700` local state directory with at least the configured capacity. Do not
place secret bytes in arguments, the configuration, logs, or the unit file.

The authority must bind the decoded Shared credential and a current cache
grant. Edge persists its monotonic anti-rollback state beneath the configured
state root; do not delete it during an authority rotation.

## Run and inspect directly

From a verified release root:

```bash
/opt/buildopt/release/bin/buildopt-edge serve \
  --config /etc/buildopt/edge.json
```

In another terminal, inspect only aggregate local state:

```bash
/opt/buildopt/release/bin/buildopt-edge status \
  --config /etc/buildopt/edge.json
```

The status command exits zero only while the process reports `READY` with an
observation no older than five seconds. `NOT_READY`, `STOPPING`, `STOPPED`, a
stale document, or an unreadable document exits nonzero. The HTTP listener
still exposes only `GET|PUT /cache/{key}` and has no status/admin route.

## Rotate authority safely

Publish the new credential, trust root, and signed authority with private
sibling-temp-file plus atomic rename operations. Multi-file replacement may
briefly produce a mixed set; Edge deliberately disables the route and returns
`503` until one complete set verifies. A rollback generation stays disabled.
A current signed generation with monotonic policy, revocation, L1 security,
gateway connection, and namespace state restores `READY` automatically.

Check status after rotation and verify that the prior authority digest is no
longer accepted by Shared. Never recover by deleting anti-rollback state or by
copying cached metadata into authority files.

## Generate a systemd unit

With the signed release already verified and the state directory prepared:

```bash
./dev/render-edge-service \
  --release-root /opt/buildopt/release \
  --config /etc/buildopt/edge.json \
  --output /opt/buildopt/buildopt-edge.service

systemd-analyze verify /opt/buildopt/buildopt-edge.service
```

Review the generated private unit. Linking and starting it are explicit host
changes outside the generator:

```bash
sudo systemctl link /opt/buildopt/buildopt-edge.service
sudo systemctl enable --now buildopt-edge.service
/opt/buildopt/release/bin/buildopt-edge status \
  --config /etc/buildopt/edge.json
```

The unit restarts only on failure, sends `SIGTERM`, allows writes only beneath
the configured state directory, and reads secrets indirectly through the
private configuration paths. Stop and disable the service before moving or
purging state. Preserve state by default; deletion is a separate retention
decision.
