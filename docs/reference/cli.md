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

### Run the one-command optimization POC

```text
buildopt optimize [options] [--] <gradle args...>
```

The stable north-star invocation is `buildopt optimize build`. The command runs
optimized native Gradle first, then derives repository identity, immutable
base/target, exact changes, non-empty Gradle-owned outputs and a typed
structural graph. A complete candidate returns
`LEARNING / STRUCTURAL_CANDIDATE_DISCOVERED`; unsupported or ambiguous state
retains native Gradle. Generated private state, discovery documents and the
latest result live under `.buildopt/optimize/v1`, with exact resume only for
matching executable, repository scope, Wrapper, Gradle arguments, derived
discovery context and budget bindings.

Defaults are a 30-minute future calibration budget, eight balanced pairs and a
maximum accepted break-even of 30 matching builds. `--json` reserves stdout
for one result object and sends Gradle console output to stderr.
`BUILDOPT_BYPASS=1` skips optimize state and reporting. No state grants
production authority or enters Test Optimization scope. Discovery performs no
calibration, selection or performance claim yet.

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

### Evaluate a structural POC profile

```text
buildopt profile evaluate \
  --manifest PATH \
  --graph PATH \
  --generated-manifest PATH \
  [--evidence PATH --profile-output PATH]
```

This is the recommended generic decision surface. Without evidence it reports
whether a complete repository-owned graph contains a candidate worth measuring.
With exact installed-path evidence it writes a digest-bound v4 profile only
when all timing, output and fallback gates qualify. Otherwise it reports
`NATIVE_FULL_GRAPH` and writes nothing. The command never infers required
outputs, activates a profile, or grants production authority.

### Review Gradle output ownership

```text
buildopt profile outputs \
  --repository-id OWNER/REPO \
  --pipeline-class CLASS \
  --entrypoint TASK \
  [--entrypoint TASK ...] \
  [--required-output GLOB ...]
```

This preflight executes the exact owner workflow once and inspects the output
declarations of its Gradle task graph. It emits non-empty, repository-contained
candidate paths with their most-specific project owners and producer tasks.
Confirmed declarations return `VALIDATED_REQUIRED_OUTPUTS`; absent declarations
return review candidates; missing, empty, symlinked or ambiguously owned output
contracts retain `NATIVE_FULL_GRAPH`. It never warms, times, proposes or
activates an optimization.

### Confirm a repository owner input

```text
buildopt profile input \
  --output-contract PATH \
  --confirm \
  [--gradle-command PATH] \
  [--gradle-option VALUE ...] \
  [--output .buildopt/profile.json]

buildopt profile input --check .buildopt/profile.json
```

Only a validated output contract can become an owner input, and creation
requires explicit confirmation. The file records the workflow,
`GIT_DIFF_BASE_TO_HEAD` change source, confirmed outputs, global fallback
paths, Gradle options, timeout, observed revision, and source-contract digest.
It is review input for local and CI proposals, not an active profile.

### Propose a structural POC measurement

```text
buildopt profile propose \
  --owner-input .buildopt/profile.json \
  --base-revision REVISION
```

This is the first command for a repository that has no BuildOpt manifest. It
runs the output-contract preflight before structural discovery and always
writes `buildopt-output-contract.json`. Only a validated non-empty declaration
can continue into the two configured-model discovery passes that map the exact
Git change to Gradle projects, propose each terminal task selector on those
projects and validate the smaller graph. Repeated `--entrypoint` values preserve
a real multi-entrypoint workflow without replacing it with an artificial root
task. It writes reviewable manifest, graph, generated binding, fallback and
proposal documents only for a supported complete candidate. Global,
ambiguous, custom, Test-bearing, unknown or invalid-output workflows retain
`NATIVE_FULL_GRAPH`. It never writes an active profile or predicts a speedup.
The explicit legacy flags remain available for compatibility; the owner input
is the recommended shared local/CI path and derives the exact Git diff when no
changes file is supplied. See the [owner-input contract](../../specs/poc-generic-owner-input-v1.md)
and [generic structural profile onboarding](../../specs/poc-generic-profile-onboarding-v1.md).

### Measure a structural POC candidate

```text
buildopt profile measure \
  --manifest PATH \
  --graph PATH \
  --generated-manifest PATH \
  --changes-file PATH \
  --fallback-changes-file PATH \
  --base-revision REVISION \
  --buildopt-revision REVISION \
  --evidence-output PATH \
  [--gradle-option VALUE ...] \
  [--target-stability-confirmations 1|2|3] \
  [--adaptive-candidate-stability] \
  [--calibration-only] \
  [--timeout DURATION]
```

This command supplies the evidence consumed by `profile evaluate`. It requires
a clean tracked target revision and an exact base-to-target changes file. The
optimized-native and installed-BuildOpt arms use independent local clones,
Gradle homes and native-cache seeds; eight pairs alternate execution order.
Required outputs must remain byte-identical in every observation and under the
full-graph fallback. Non-positive evidence remains `INCONCLUSIVE`; invalid
source state, build failure or output mismatch writes no evidence. See
[generic isolated structural measurement](../../specs/poc-generic-measurement-v1.md).
With `--calibration-only`, the command records only the candidate cache seed,
base-daemon stabilization and bounded target-workload stabilization. It makes
no timing or qualification claim and is intended for POC break-even studies
that are bound to a separate terminal performance result.

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
entrypoints. `--gradle-option` is repeatable and accepts bounded execution
options plus explicit `-Pname=value` project properties recorded by the owner.
It cannot add/exclude tasks, change the project root or load an init script;
task entrypoints come only from the repository manifest.

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
| `task-intelligence-evaluation` | Evaluate a strict Task Intelligence input JSON into result JSON |
| `beta-benchmark` | Run and validate bounded smoke/fault/sustained/soak evidence |
| `metrics-catalog-validator` | Validate the versioned metric catalog |

Use the matching `dev/check-*` gate and specification rather than relying on
unstated flags. The soak subcommand exists as a deferred qualification tool;
the quickstart and owner POC lab do not run it.

Runtime Tuning, exact-bound Hot State, and standard Copy have no installed or
source-tree evaluation CLI. Their immutable results are historical evidence.
