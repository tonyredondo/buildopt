# CLI reference

Native packages contain four user/operator binaries on Linux and macOS.
Windows additionally contains the SCM host `buildopt-service.exe`.

## `buildopt`

### Run a command

```text
buildopt run -- <command> [args...]
```

The delimiter is mandatory. The command is started directly without a shell.
The child inherits the working directory, standard streams, and non-reserved
environment. Once started, its ordinary exit status wins.


### Run Gradle with packaged integration

```text
buildopt gradle [gradle args...]
```

Run this command from a Gradle repository root. BuildOpt discovers `gradlew`
or `gradlew.bat`, supplies the installed init script and plugin, and then uses
the same launcher, cache and session path as `run --`. The package owns the
internal JAR paths; users should not configure them.

`BUILDOPT_BYPASS=1 buildopt gradle ...` invokes the Wrapper without the
BuildOpt init script or plugin.

### Run the qualified POC profile

```text
buildopt poc --changes-file PATH [--config PATH] [--timings-file PATH] [--edge-url LOOPBACK_ORIGIN]
```

The default config is the repository-owned
`buildopt-qualified-profile.json`. The command validates the exact clean
profile and emits a `buildopt.poc/qualified-profile-plan/v1` JSON object before
Gradle starts. The plan exposes selection/fallback reason, entrypoints,
affected scope, omitted-project count, expected outputs, preserved Test-owned
checks, enabled adapters and every disabled mechanism.

The v1 profile permits only native Gradle cache, Build Impact and the exact
standard-`Jar` adapter. The v2 profile permits only Build Impact plus an
explicit read-only IPv4 loopback Edge endpoint and evaluates repository-owned
file-SHA preconditions first. Unknown/global changes, failed preconditions,
absent/invalid Edge and bypass use native full-graph execution without Edge.
HTTP failures retain Gradle-native local execution. Both plans are reviewable
POC decisions, not autonomous or production authorization.

### Discover a qualified POC profile

```text
buildopt profile discover \
  --manifest PATH \
  --graph PATH \
  --generated-manifest PATH \
  --matrix-summary PATH \
  --cell-evidence PATH \
  --profile-contract PATH
```

This read-only command emits a deterministic review document on standard
output. It embeds a profile only when the matrix cell, graph, generated state,
trace/input digests, mechanism set, preconditions, outputs, and safety
fallbacks all remain qualified. Unqualified or uncertain inputs emit
`NATIVE_FULL_GRAPH` with `profile: null`. The command does not write the
repository, run Gradle, activate the profile, or grant production authority.
See [deterministic POC profile discovery](../../specs/poc-profile-discovery-v1.md).

### Run an explicit Build Impact POC candidate

```text
buildopt impact \
  --repository-id OWNER/REPO \
  --changes-file PATH \
  [--pipeline-class CLASS] \
  [--manifest PATH] \
  [--graph PATH] \
  [--generated-manifest PATH] \
  [--gradle-option VALUE ...]
```

`PATH` is a bounded repository-relative file with one unique changed path per
line. The command validates all checked-in Build Impact state and then runs
either the exact repository-owned alternative or the manifest's original full
entrypoints. `--gradle-option` is repeatable and accepts only execution options
that cannot add/exclude tasks or change the project root; task entrypoints come
only from the repository manifest.

This command is an explicit POC experiment, not production authorization.
Malformed or drifted state returns code `78`; unknown and global changes safely
execute the full graph. Existing Test-owned checks remain outside this
build-owned selection and stay in their current workflow.
`BUILDOPT_BYPASS=1` also restores the original full entrypoints.

### Inspect platform capabilities

```text
buildopt doctor
```

Prints one `buildopt.doctor/v1` JSON document containing OS, architecture,
persistent-gateway, managed-L1, bootstrap-cache, server, Edge, process
isolation, resource isolation, storage policy, and background-service
capabilities.

### Exit codes

| Code | Meaning |
|---:|---|
| `0` | Help/report or child success |
| `64` | Invalid CLI usage |
| `78` | Gradle Wrapper or packaged integration is unavailable |
| `126` | Resolved command could not execute |
| `127` | Command not found |
| `128 + signal` | Child terminated by an unhandled signal on Unix |
| other | Original child exit status |

See [`cmd/buildopt/README.md`](../../cmd/buildopt/README.md) for the complete
launcher, signal, gateway, L1, Shared, ingest, and export contract.

## `buildopt-server`

### Serve

```text
buildopt-server serve [options]
```

Common options:

| Option | Purpose |
|---|---|
| `--self-hosted-config ABSOLUTE_PATH` | Load the strict single-node JSON configuration |
| `--listen 127.0.0.1:8042` | Canonical loopback listener |
| `--state-dir ABSOLUTE_PATH` | Shared/control state root |
| `--export-dir PATH` | Immutable build-session export root |
| `--export-profile summary|tasks|evidence|diagnostic` | Select redaction/detail profile |
| `--authorize-expanded-export` | Required for profiles broader than summary |
| `--diagnostic-until RFC3339` | Required bounded expiry for diagnostic export |
| `--cache-authority ABSOLUTE_PATH` | Signed cache authority document |
| `--cache-trust-root ABSOLUTE_PATH` | Authority trust root |
| `--cache-credential ABSOLUTE_PATH` | Local cache credential file |
| `--cache-token-auth` | Enable scoped beta-token authentication |
| `--github-webhook-secret ABSOLUTE_PATH` | Enable the private GitHub queue adapter |

`--self-hosted-config` owns listener, storage, export, cache, and optional
GitHub queue fields. Do not combine it with independent values for those same
fields.

### Export

```text
buildopt-server export --export-dir PATH [--format jsonl] [profile options]
```

Validates and copies the private event stream to stdout. Expanded profiles
require the same explicit authorization used during production.

### Scoped tokens

```text
buildopt-server token issue --state-dir ABSOLUTE_PATH [scope options]
buildopt-server token revoke --state-dir ABSOLUTE_PATH --token-id ID
```

`issue` exposes the opaque token once. Only a domain-separated digest and its
exact repository/namespace/plane/access/expiry scope persist.

### Authority inspection

```text
buildopt-server authority inspect \
  --authority ABSOLUTE_PATH \
  --trust-root ABSOLUTE_PATH \
  --credential ABSOLUTE_PATH
```

Verifies the complete private authority without starting a listener or
printing the credential.

### Data deletion

```text
buildopt-server data delete --data-root ABSOLUTE_PATH [identity and generation options]
```

Performs the coordinated isolated-profile deletion contract. It refuses active
leases, records logical revocation before physical removal, emits a tokenized
tombstone, and requires strictly advanced namespace/L1 generations.

The exact option set and behavior are in
[`cmd/buildopt-server/README.md`](../../cmd/buildopt-server/README.md).

## `buildopt-edge`

```text
buildopt-edge validate --config ABSOLUTE_PATH
buildopt-edge serve --config ABSOLUTE_PATH
buildopt-edge status --config ABSOLUTE_PATH
```

- `validate` checks config, private inputs, authority, and storage preflight
  without serving traffic.
- `serve` runs the loopback cache endpoint and background replication,
  maintenance, status, and authority-reload loops.
- `status` reads aggregate local status and succeeds only for a fresh `READY`
  process observation.

## `buildopt-impact`

```text
buildopt-impact <generate|check> \
  --repository ROOT \
  --manifest PATH \
  --repository-id OWNER/REPO \
  --pipeline-class CLASS \
  --graph PATH \
  --generated-manifest PATH \
  [--gradle-command PATH] \
  [--timeout 5m]
```

`generate` atomically writes repository-relative graph and generated-manifest
files. `check` rediscovers state and requires byte-exact checked-in files.
Outputs may not escape the repository. Discovery never replaces the
repository-owned policy manifest.

## `buildopt-service.exe`

```text
buildopt-service.exe \
  --service-name NAME \
  --component server|edge \
  --config ABSOLUTE_PATH
```

This Windows-only binary is an SCM lifecycle host. Use the package's
`install-services.ps1` and `uninstall-services.ps1`; do not construct service
commands by hand unless debugging a reviewed definition.

## Development and evaluation commands

These binaries are source-tree tools rather than general installed product
interfaces:

| Binary | Purpose |
|---|---|
| `neutral-envelope` | Record, report, validate, assign, and export paired measurement/pilot evidence |
| `runtime-evaluation` | Evaluate a strict runtime-optimizer input JSON into result JSON |
| `task-intelligence-evaluation` | Evaluate a strict Task Intelligence input JSON into result JSON |
| `beta-benchmark` | Run and validate bounded smoke/fault/sustained/soak evidence |
| `metrics-catalog-validator` | Validate the versioned metric catalog |

Use the matching `dev/check-*` gate and specification rather than relying on
unstated flags. The soak subcommand exists as a deferred qualification tool;
the quickstart and owner POC lab do not run it.
