# CLI reference

Native packages contain four user/operator binaries on Linux and macOS.
Windows additionally contains the SCM host `buildopt-service.exe`.

## `buildopt`

### Generate the repository wrapper

```text
buildopt wrapper init [--server URL --project-scope SCOPE] [--mode auto|observe|off]
buildopt wrapper check
buildopt wrapper update --version VERSION [--allow-downgrade]
```

This maintainer CLI is implemented by `SWL-002`. `init` resolves the latest
stable public BuildOpt release into immutable URLs and GitHub-provided SHA-256
digests, then generates `buildoptw`, `buildoptw.bat`,
`.buildopt/wrapper.properties` and `.buildopt/config.toml` without a credential
value or machine path. It refuses if any target already exists. `--server` and
`--project-scope` must be supplied together; their private token remains in
`BUILDOPT_TOKEN`.

`check` is offline and read-only: non-canonical bytes, modes, order, values,
links or missing files fail without repair. `update` first requires a clean
canonical state, changes only release version/URLs/checksums and leaves both
scripts and owner configuration byte-identical. Repeating a version performs
no write; a downgrade requires `--allow-downgrade`. All writes are staged and
rolled back as one four-file transaction on failure.

The generated scripts now bootstrap and verify the pinned native distribution
in a user cache. The implemented management commands are:

```text
./buildoptw --buildopt version [--json]
./buildoptw --buildopt status [--json]
./buildoptw --buildopt explain [--json]
```

`status` is a read-only summary of the current wrapper decision, ordinary-build
observations, cache facts, trial/economic availability, fallback reason and
latest validated bindings. `explain` prints the same report model with a
human-readable explanation of why native Gradle or a verified action is used.
`--json` emits the exact structured report used to render the human output.
Missing evidence is `UNAVAILABLE`, never zero. These commands do not create or
modify repository, cache, decision or observation files, and an unverified
decision always retains native Gradle. Credentials and checkout paths are
excluded from both forms. Validate the contract with
`./dev/check-sticky-wrapper-status`.

The repeated customer command remains:

```text
./buildoptw <gradle args...>
./buildoptw --buildopt status [--json]
./buildoptw --buildopt explain [--json]
```

Ordinary Gradle arguments now pass through the verified `buildopt run --`
launcher. This proves process equivalence only; it does not activate or claim
cache, observation, learning or optimization value.

Every invocation without the first-argument `--buildopt` prefix goes to the
existing repository Gradle Wrapper. Therefore `./buildoptw status` still runs a
Gradle task named `status`. A leading `--gradle` is removed and forces the
remaining arguments to Gradle, including a literal `--buildopt`.

`BUILDOPT_BYPASS=1` skips configuration, distribution bootstrap, decision,
state, cache gateway, plugin and observation setup and invokes Gradle directly.
The exact routing, exit codes and fail-open behavior are in the
[wrapper contract](../../specs/poc-sticky-wrapper-contract-v1.md).
Generator behavior and evidence are in the
[generator contract](../../specs/poc-sticky-wrapper-generator-v1.md).
Bootstrap behavior and evidence are in the
[bootstrap contract](../../specs/poc-sticky-wrapper-bootstrap-v1.md).
Process behavior, pre-bootstrap bypass and fail-open fallback are in the
[passthrough contract](../../specs/poc-sticky-wrapper-passthrough-v1.md).

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

The stable north-star invocation is `buildopt optimize build`. The first
qualifying invocation runs optimized native Gradle, derives repository/change/
output/graph facts, calibrates a complete candidate and stores it only after
exact output, fallback, wall-time and payback gates pass. A later exact
invocation validates the executable, repository scope, revision, Wrapper,
arguments, graph, outputs, evidence and profile before selecting the smaller
graph. Unsupported, ambiguous, drifted or non-value state retains native
Gradle. Generated private state and the latest result live under
`.buildopt/optimize/v1`.

If `buildopt connect` has created a private central connection, the command
also synchronizes optimization state automatically before and after execution.
Exact local replay remains first. A remote profile crosses source commits only
after local structural and evidence revalidation; `--connection-dir` overrides
the `.buildopt/central/v1` default. Service or binding drift retains native.

Defaults are a 30-minute future calibration budget, eight balanced pairs and a
maximum accepted break-even of 30 matching builds. `--json` reserves stdout
for one result object and sends Gradle console output to stderr.
`BUILDOPT_BYPASS=1` skips optimize state and reporting. In GitHub/GitLab, the
repository scope uses provider identity instead of the ephemeral checkout path
so an exact checkpoint can move between runners; every other binding still
passes before reuse. No state grants production authority or enters Test
Optimization scope.

Every non-bypass JSON result includes a diagnostic `timing` object. Its
top-level `preExecutionNs`, `gradleExecutionNs`, `finalizationNs`,
`unattributedNs` and `totalNs` fields are non-overlapping and reconcile the
launcher interval. Nested setup, matching, state, materialization, output
verification and discovery/learning durations explain work inside those
phases; they are diagnostic and must not be added to the top-level phases.

### Connect and synchronize optional central state

```text
buildopt connect https://HOST[:PORT] --token-file PATH [--ca-file PATH]
buildopt sync
```

`connect` verifies a TLS 1.3 HTTPS endpoint, repository-scoped token and
optional CA, stores the connection privately below `.buildopt/central/v1`, and
performs the first synchronization. `sync` reuses that exact connection.
Generated evidence, portfolio and checkpoint documents are transported as
canonical digest-bound bundles; no hand-authored BuildOpt files are required.

An unchanged second sync is a no-op. Interrupted publication resumes from
immutable content, concurrent writers retain the verified winning generation,
and an outage may use only a previously verified private snapshot. Missing,
incompatible or corrupt state retains native behavior. This POC command does
not yet make `buildopt optimize` select remote profiles and never exposes the
central credential to Gradle.

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

The AF-015 terminal scorecard is intentionally a source-tree validation command
(`./dev/check-adaptive-fragment-terminal-decision`), not an installed customer
CLI. The adaptive-fragment hypothesis is stopped and has no supported
activation command.

Runtime Tuning, exact-bound Hot State, and standard Copy have no installed or
source-tree evaluation CLI. Their immutable results are historical evidence.
