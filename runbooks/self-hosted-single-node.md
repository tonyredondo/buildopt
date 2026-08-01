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

## Admission and normal operation

Follow [`private-beta-operations.md`](./private-beta-operations.md) for
`/livez`, `/readyz`, alerts, revocation, circuit fallback, bypass, shutdown,
and rollback. Admit builds only after readiness is `200`. Keep the service on
loopback behind any separately managed TLS boundary.

## Stop or remove

Stop and disable the linked unit before changing deployment state. Confirm the
Shared writer lease is released, then use the signed deployment manager for
rollback or preserve-by-default uninstall. Data purge remains an explicit,
separate retention decision. The later `A2-003` and `A2-004` procedures extend
this runbook with compatible upgrade and manual restore.
