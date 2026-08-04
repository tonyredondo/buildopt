# Configuration reference

BuildOpt configuration is grouped by trust boundary. Supply a complete group
or omit it; partial configuration normally disables that optional capability
or prevents a persistent service from starting. Values marked secret must not
enter source control, command arguments, Gradle, logs, exported sessions, or
diagnostics.

This guide summarizes operator-facing inputs. The closest component README and
executable specification remain authoritative for exact validation.

## Launcher controls

| Variable | Secret | Meaning | Failure behavior |
|---|---:|---|---|
| `BUILDOPT_BYPASS=1` | No | Run the original argv without plugin, gateway, server, or policy | Only exact `1` activates it |
| `BUILDOPT_SAFE_CACHE=1` | No | Explicitly evaluate the repository-scoped Safe Cache; default is Gradle native cache | Invalid values fail before Gradle; unqualified use is not automatic |
| `BUILDOPT_GATEWAY_STATE_ROOT` | No | Absolute private persistent gateway state root | Incomplete managed pair falls back |
| `BUILDOPT_RUNNER_SLOT` | No | Stable 1–128 character runner-slot identity | Busy/invalid slot uses baseline |
| `BUILDOPT_GATEWAY_IDLE_TIMEOUT` | No | Managed gateway idle lifetime, `100ms`–`24h` | Invalid value disables managed path |

Launcher-owned `BUILDOPT_PLUGIN_*` and `BUILDOPT_GATEWAY_*` rendezvous values
are internal outputs. Do not prepopulate them; the launcher removes inherited
values and creates fresh invocation context.

## Session ingest and export

| Variable | Secret | Meaning |
|---|---:|---|
| `BUILDOPT_SERVER_URL` | No | Canonical loopback server origin used by the launcher |
| `BUILDOPT_SERVER_INGEST_TOKEN` | Yes | Independent 32–512 byte write-only ingest credential |
| `BUILDOPT_BUILD_SESSION_CONTEXT` | No secret content | Strict pre-outcome tokenized repository/revision/task JSON |
| `BUILDOPT_HISTORY_API_TOKEN` | Yes | Independent read token for history API/dashboard |

Server URL and ingest token must be present together. They are removed before
Gradle starts. Missing, invalid, rejected, or unavailable delivery is recorded
without replacing the Gradle exit status.

`BUILDOPT_BUILD_SESSION_CONTEXT` is limited to 32 KiB and contains tokenized
identity and digests, never an HMAC key or source content. See
[`cmd/buildopt/README.md`](../../cmd/buildopt/README.md#build_session-export-context)
for the exact JSON example.

## Managed L1 and Shared cache

### Managed L1 scope

With no cache-specific inputs, `buildopt gradle` enables Gradle's native local
Build Cache and does not start the BuildOpt plugin, gateway, or managed L1.
`BUILDOPT_SAFE_CACHE=1` opts into the POC Safe Cache: state is then stored below
the operating system user cache directory as `buildopt/state`; the canonical
repository path, operating system/architecture, and complete Wrapper
properties are hashed into opaque isolation scopes. Raw paths do not enter the
cache layout.

`--no-build-cache` disables automatic cache activation for the invocation.
`BUILDOPT_BYPASS=1` additionally removes the BuildOpt plugin and launcher
optimization path. A Wrapper change creates a new compatibility scope, and a
different checkout path creates a different repository scope when Safe Cache is
explicitly active.

The variables below are the advanced operator-owned override. Supplying any
one prevents automatic derivation; supply the complete group.

| Variable | Meaning |
|---|---|
| `BUILDOPT_L1_STATE_ROOT` | Absolute private L1 state root |
| `BUILDOPT_L1_TENANT_ID` | Tenant scope |
| `BUILDOPT_L1_REPOSITORY_ID` | Repository scope |
| `BUILDOPT_L1_TRUST_DOMAIN` | Trust boundary |
| `BUILDOPT_L1_COMPATIBILITY_CLASS` | Gradle/JDK/OS/architecture compatibility |
| `BUILDOPT_L1_SECURITY_GENERATION` | Monotonic security generation for the unsigned pilot path |
| `BUILDOPT_L1_L2_WRITE_AUTHORIZED` | `0` for normal native L1; `1` only for an already authorized L2 writer |

When signed local authority is configured, its generation and write permission
replace parent-supplied generation/authorization. Raw scope values remain
launcher-only; Gradle receives an opaque directory and fixed retention policy.

### Signed Shared authority

| Variable | Secret | Meaning |
|---|---:|---|
| `BUILDOPT_LOCAL_AUTHORITY_PATH` | No, private | Absolute signed authority JSON path |
| `BUILDOPT_LOCAL_TRUST_ROOT_PATH` | No, private | Absolute pinned trust-root path |
| `BUILDOPT_LOCAL_CACHE_CREDENTIAL_PATH` | Yes | Absolute local cache credential path |
| `BUILDOPT_SHARED_CACHE_URL` | No | Canonical HTTPS origin or canonical loopback HTTP |
| `BUILDOPT_SHARED_CACHE_TOKEN_PATH` | Yes | Optional scoped remote beta-token path |

Files must be regular, current-user-owned, and mode `0600` where the platform
supports POSIX modes. The launcher verifies signature, canonical form,
repository/component binding, expiry, revocation, and monotonic generations
before exposing a local cache context.

## Managed Gradle bootstrap cache

`BUILDOPT_GRADLE_BOOTSTRAP_CONFIG_PATH` points to the strict launcher-owned
bootstrap configuration. It binds a read-only dependency snapshot and a
checksum-verified Wrapper distribution to the runner slot, compatibility
class, Wrapper JAR, distribution digest, and configuration policy. Invalid or
drifted input leaves Gradle on its normal bootstrap path.

## Resource profile context

The launcher can consume orchestrator-provided runner/build facts:

- `BUILDOPT_RUNNER_CLASS`
- `BUILDOPT_BUILD_CLASS`
- `BUILDOPT_RUNNER_CGROUP_CPU_COUNT`
- `BUILDOPT_RUNNER_CGROUP_MEMORY_BYTES`
- `BUILDOPT_RUNNER_AVAILABLE_MEMORY_BYTES`
- `BUILDOPT_RUNNER_JDK_VENDOR`
- `BUILDOPT_RUNNER_JDK_VERSION`
- `BUILDOPT_RUNNER_JDK_ARCHITECTURE`

These facts select only a catalogued bounded profile. For the active POC,
`STABLE_CONTROL` is the only authorized profile: `W4_H6G` and `W3_H4G` failed
the value threshold and are rejected before arguments are applied. Invalid,
missing, incompatible, or experimental context keeps the baseline. macOS
reports resource isolation rather than emulating Linux cgroups; Windows uses
Job Objects.

## Server configuration

The strict self-hosted JSON is the preferred persistent-service interface:

```json
{
  "schemaVersion": "buildopt.self-hosted/config/v1",
  "profile": "PRIVATE_BETA_ISOLATED_SINGLE_NODE",
  "server": {"listen": "127.0.0.1:8042"},
  "storage": {
    "stateDirectory": "/absolute/private/shared",
    "filesystemPolicy": "ALLOWLIST_PROVEN_LOCAL",
    "minimumDeploymentBytes": 21474836480,
    "maximumDeploymentBytes": 536870912000,
    "usableVolumePercent": 50
  },
  "export": {
    "directory": "/absolute/private/exports",
    "profile": "summary"
  },
  "cache": {
    "authorityPath": "/absolute/private/authority.json",
    "trustRootPath": "/absolute/private/trust-root.json",
    "credentialPath": "/absolute/private/cache-credential",
    "betaTokenAuthentication": true
  }
}
```

Use the maintained
[`self-hosted-single-node.example.json`](../../specs/self-hosted-single-node.example.json)
as the copy source. Adjust paths for the target OS; do not weaken the profile,
filesystem policy, loopback listener, or private-file boundary.

## Edge configuration

Edge config binds:

- a stable `edgeId` and loopback listener;
- Shared base URL and credential path;
- local state root, byte quota, maximum object, TTL, and SLRU watermarks;
- trust root and current authority snapshot paths;
- immutable `SHARED_ONLY` commit/collision authority and bounded offline
  behavior.

Start from [`edge-cache.example.json`](../../specs/edge-cache.example.json) or
the Windows path example
[`edge-cache.windows.example.json`](../../specs/edge-cache.windows.example.json),
then run:

```bash
buildopt-edge validate --config /absolute/private/edge.json
```

## Export profiles

| Profile | Authorization | Intended content |
|---|---|---|
| `summary` | Default | Redacted high-level build/session facts |
| `tasks` | `--authorize-expanded-export` | Redacted task-level detail |
| `evidence` | `--authorize-expanded-export` | Bounded evidence detail |
| `diagnostic` | Expanded authorization plus future UTC `--diagnostic-until` within seven days | Time-bounded diagnostic detail |

All profiles tokenize repository, trust-domain, and task identities before
durable output. A broader profile is not inferred from an existing file or an
operator's read access.

## CI installer variables

GitHub Action and GitLab component users normally provide no installer data.
`version` optionally pins native bits; the integration resolves the matching
public archive and SHA-256. The Action additionally keeps the paired
`archive-url` and `archive-sha256` inputs for legacy Release Bundle v1
consumers. `BUILDOPT_ACTION_*`, `BUILDOPT_VERSION` and
`BUILDOPT_ARCHIVE_*` are installer internals rather than product configuration.
Prefer the published interfaces instead of setting them manually.


See [CI integration](../guides/ci-integration.md).
