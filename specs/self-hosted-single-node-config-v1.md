# Self-hosted single-node configuration v1

This contract closes `A2-001`: one strict declarative file now selects the
existing isolated private-beta server, Shared storage, summary export, and
authenticated cache boundaries. It does not install or upgrade the service.

## Activation and configuration

Copy [`self-hosted-single-node.example.json`](./self-hosted-single-node.example.json)
outside the state and export roots, replace the absolute paths, and make the
result mode `0600`. Start the server with only:

```bash
buildopt-server serve --self-hosted-config /etc/buildopt/self-hosted.json
```

The flag is exclusive: mixing it with individual `serve` flags fails before
configuration is consumed. Unknown JSON fields, a second JSON document,
symlinks, permissive modes, a file larger than 64 KiB, relative/root paths,
overlapping state/export/secret paths, non-summary export, non-loopback
listeners, and disabled beta-token authentication are rejected.

The file contains paths to authority, trust-root, credential, and optional
GitHub webhook-secret files; it never embeds secret values. Those existing
readers retain their private-file and cryptographic validation.

## Storage preflight

Before opening the HTTP listener, the server validates and opens the production
single-node Shared store. The nearest existing state ancestor must be a real
directory on an allowlisted proven-local filesystem. Preflight itself creates
nothing and rejects a symlink ancestor.

Effective deployment capacity is 50% of the volume, capped at 500 GiB. Both
that effective capacity and currently available space must be at least 20 GiB.
The existing Shared startup then owns private layout creation, the exclusive
writer lock, schema migration, integrity checks, and reconciliation.

Run the executable contract with:

```bash
./dev/check-self-hosted-single-node-config
```

The separate next blocks own reproducible service installation (`A2-002`),
compatible upgrade/restart migration (`A2-003`), and restore generation
rotation (`A2-004`). This slice does not claim the deferred soak, external
design-partner validation, HA, backups, or production RPO/RTO.
