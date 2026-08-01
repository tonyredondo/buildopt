# Self-hosted single-node operations

This is the operator runbook for the owner-operated `MVP-A2` proof of concept.
It uses one Linux AMD64 host, one supported local filesystem, loopback serving,
and systemd. It promises no HA, backup RPO/RTO, enterprise identity, external
pilot validation, or eight-hour soak.

## Prepare private inputs

Choose absent, canonical, disjoint deployment and persistent-data roots. Keep
the release verification key outside both roots. Provide current-user-owned
mode-`0600` authority, trust-root, cache-credential, and service-environment
files. The environment file contains exactly one non-whitespace 32–512 byte
token assignment and a final newline:

```text
BUILDOPT_SERVER_INGEST_TOKEN=<opaque-token>
```

Do not place secret values in command arguments, the declarative JSON, the
systemd unit, logs, or the repository.

## Install and inspect

From a trusted BuildOpt source checkout, install a previously verified signed
bundle with the same externally pinned public key:

```bash
./dev/manage-self-hosted install \
  --bundle /srv/releases/buildopt-<version> \
  --key /srv/trust/buildopt-release.pub \
  --root /opt/buildopt \
  --data-root /var/lib/buildopt \
  --environment-file /etc/buildopt/server.env \
  --authority /etc/buildopt/authority.json \
  --trust-root /etc/buildopt/trust-root.json \
  --credential /etc/buildopt/cache-credential \
  --listen 127.0.0.1:8042

./dev/manage-self-hosted status --root /opt/buildopt
```

Installation produces `/opt/buildopt/buildopt-self-hosted.service` without
changing systemd. Review that private unit and its status output before
linking it. For a system service, use the owning host's privileged change
process to link, enable, and start the exact generated unit; for example:

```bash
sudo systemctl link /opt/buildopt/buildopt-self-hosted.service
sudo systemctl enable --now buildopt-self-hosted.service
```

Those supervisor mutations are intentionally outside the repository checker.
Do not copy the unit to a second path: its immutable release and configuration
paths are part of the installed manifest.

## Upgrade and restart safely

Obtain the next signed release bundle and verify it with the same externally
pinned key. Selection is online: the running process keeps its already-open
old immutable release while the manager serializes publication and atomically
rewrites the generated unit and private manifest for the next restart:

```bash
./dev/manage-self-hosted upgrade \
  --bundle /srv/releases/buildopt-<next-version> \
  --key /srv/trust/buildopt-release.pub \
  --root /opt/buildopt

./dev/manage-self-hosted status --root /opt/buildopt
```

If descriptor composition fails after version selection, the command restores
the prior signed selection, unit, and manifest. Treat any nonzero result as
fail-closed and do not restart until `status` succeeds. The manager never calls
the supervisor. After a successful status check, explicitly restart the linked
unit and wait for readiness before admitting builds:

```bash
sudo systemctl restart buildopt-self-hosted.service
curl --fail --noproxy '*' http://127.0.0.1:8042/readyz
```

The persistent data and declarative configuration do not change. Startup
reconciliation completes before readiness becomes `200`, and pending or
partial objects remain misses rather than becoming cache hits.

## Admission and normal operation

Follow [`private-beta-operations.md`](./private-beta-operations.md) for
`/livez`, `/readyz`, alerts, revocation, circuit fallback, bypass, shutdown,
and rollback. Admit builds only after readiness is `200`. Keep the service on
loopback behind any separately managed TLS boundary.

## Stop or remove

Stop and disable the linked unit before changing deployment state. Confirm the
Shared writer lease is released. For a composed rollback, run
`manage-self-hosted upgrade` with the prior signed bundle; do not invoke the
lower-level rollback because selection, unit, and manifest must move together.
Use the signed deployment manager only for preserve-by-default uninstall after
the service is stopped. Data purge remains an explicit,
separate retention decision. The later `A2-004` procedure extends this runbook
with manual restore and revocation-generation rotation.
