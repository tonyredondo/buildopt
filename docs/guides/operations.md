# Operations

BuildOpt's persistent services are deliberately explicit. Installation does
not silently start a server or Edge node because both require owner-supplied
private configuration, credentials, storage roots, and retention choices.

## Choose a deployment path

| Path | Use when | Guide |
|---|---|---|
| Local invocation | Evaluating the launcher or running without persistent Shared state | [Quickstart](../getting-started/quickstart.md) |
| Owner POC lab | Proving the complete synthetic composition | [Quickstart](../getting-started/quickstart.md#run-the-complete-synthetic-lab) |
| Self-hosted single node | Operating one private Shared/cache/history server | [Self-hosted runbook](../../runbooks/self-hosted-single-node.md) |
| Edge Cache | Placing an optional bounded cache near runners | [Edge runbook](../../runbooks/edge-cache.md) |

All current server listeners are canonical IPv4 loopback. Remote exposure needs
a separately operated authenticated/TLS boundary and does not turn the POC into
a multi-tenant service.

## Private inputs and state

Use absolute, canonical, disjoint paths. Configuration, credentials, trust
roots, signed authority, and secret-bearing environment files must be regular
files owned by the current service identity with private permissions. Storage
roots must be dedicated, marked by BuildOpt, and on a platform-supported local
filesystem.

Never put secret bytes in:

- command arguments or service definitions;
- JSON configuration files that expect a credential *path*;
- source control, logs, exported sessions, or issue reports;
- Gradle's environment or init-script properties.

BuildOpt reads secrets through private file paths or launcher-only environment
inputs and passes minimum-scope invocation credentials to the local gateway.

## Server preflight and readiness

Start from [`specs/self-hosted-single-node.example.json`](../../specs/self-hosted-single-node.example.json)
and replace every path with an absolute private path appropriate for the host.
Validate configuration implicitly before listener creation:

```bash
BUILDOPT_SERVER_INGEST_TOKEN='<opaque-token>' \
  buildopt-server serve \
    --self-hosted-config /absolute/private/buildopt-server.json
```

The process is live before it is ready. During reconciliation, `/livez` may be
`200` while `/readyz` and product routes remain `503`:

```bash
curl --fail --silent --show-error --head http://127.0.0.1:8042/livez
curl --silent --show-error --head http://127.0.0.1:8042/readyz
curl --fail --silent --show-error http://127.0.0.1:8042/ops/v1/alerts | jq .
```

Admit a build only after readiness is `200` and no unexplained alert is firing.
Do not bypass reconciliation by editing SQLite, authority, leases, markers, or
generation state manually.

## Linux self-hosted service

The signed deployment manager installs immutable releases and renders a
reviewable systemd unit without changing the supervisor:

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

Review the generated unit before linking or starting it through the host's
privileged change process. Upgrade, rollback-safe descriptor composition,
manual restore, and preserve-by-default removal are detailed in the
[self-hosted runbook](../../runbooks/self-hosted-single-node.md).

## Edge Cache

Start from [`specs/edge-cache.example.json`](../../specs/edge-cache.example.json).
Validate before serving:

```bash
buildopt-edge validate --config /absolute/private/edge.json
buildopt-edge serve --config /absolute/private/edge.json
buildopt-edge status --config /absolute/private/edge.json
```

`status` succeeds only for a fresh `READY` observation. Edge opens no HTTP
admin route. Mixed or rolled-back authority disables cache routing until one
complete current signed set verifies. Shared always retains commit and
collision authority.

## macOS

Build a native archive on macOS with the host architecture:

```bash
./packaging/macos/package.sh \
  --version 0.1.0-local \
  --output /tmp/buildopt-package
```

Extract the archive and run its checksum-verifying installer:

```bash
./install.sh --prefix "$HOME/.local"
```

Server and Edge launchd user agents are opt-in and need absolute config paths:

```bash
./install-services.sh \
  --prefix "$HOME/.local" \
  --server-config /absolute/private/buildopt-server.json \
  --load
```

Use the matching `uninstall-services.sh` before `uninstall.sh`. The receipt
limits removal to files installed by the package; unrelated files beneath the
prefix are preserved.

## Windows

Build a native ZIP from PowerShell:

```powershell
./packaging/windows/package.ps1 `
  -Version 0.1.0-local `
  -Output "$env:TEMP\buildopt-package"
```

After extraction, install verified binaries for the current user:

```powershell
./install.ps1 -UpdatePath
```

Generate reviewable SCM definitions without installing them:

```powershell
./install-services.ps1 `
  -ServerConfig 'C:\private\buildopt-server.json' `
  -DefinitionOutput "$env:TEMP\buildopt-services.json"
```

Run the same command from an appropriately authorized administrator session
with `-Install` only after reviewing the definitions. Use
`uninstall-services.ps1` before `uninstall.ps1`. Windows packages include
`buildopt-service.exe`, which reports service lifecycle to SCM and delegates
to the selected server or Edge component.

## Shutdown, rollback, and removal

1. Stop admitting new optimized builds or activate bypass.
2. Wait for active invocations to finish.
3. Confirm readiness closes and the writer lease is released.
4. Stop the process through its supervisor.
5. Roll back by selecting a previously verified immutable release while
   preserving persistent data.
6. Start, wait for reconciliation/readiness, and run a baseline plus one
   normal-path canary.

Uninstall preserves data by default. Purging deployment data is a separate,
explicit retention decision and must target the exact marked root. Never use a
toolchain cleanup command as a deployment-data cleanup mechanism.

## Incident entry points

- [Private beta operations](../../runbooks/private-beta-operations.md): health,
  ten alert classes, revocation, circuit breaker, shutdown, rollback.
- [Base recovery](../../runbooks/base-recovery.md): bypass, CI kill switch,
  version rollback, uninstall, partial patch delivery.
- [Troubleshooting](../troubleshooting.md): symptom-first diagnosis.

Validate runbook contracts with:

```bash
./dev/check-base-runbooks
./dev/check-private-beta-operations
./dev/check-self-hosted-service-install
./dev/check-edge-operability
```
