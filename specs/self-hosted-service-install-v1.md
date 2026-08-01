# Self-hosted service installation v1

This contract closes `A2-002` by composing the existing signed immutable
deployment lifecycle with the strict `A2-001` single-node configuration. It
does not introduce another release format or mutate the host supervisor.

`dev/manage-self-hosted install` requires absent, canonical, disjoint
deployment and persistent-data roots. It verifies Release Bundle v1 through
an externally supplied pinned key before installing, then creates private
Shared, export, and self-hosted configuration directories. Authority, trust
root, cache credential, and ingest-token values remain in external mode-`0600`
files; the generated configuration and unit contain paths only.

The installed private systemd unit selects the exact verified immutable
`buildopt-server`, the strict configuration, the external environment file,
the persistent working directory, and conservative restart/hardening options.
The manager never calls `systemctl`: enabling and starting a service is an
explicit operator action described in the runbook.

`status` revalidates the signed deployment, exact manifest, paths, modes,
configuration, secret references, and byte-exact generated unit before
reporting `INSTALLED`. The executable checker installs the same signed bundle
twice from clean roots and requires identical configuration, service unit, and
manifest bytes.

Run:

```bash
./dev/check-self-hosted-service-install
```

Compatible upgrade/restart belongs to `A2-003`; restore and revocation
continuity belong to `A2-004`.
