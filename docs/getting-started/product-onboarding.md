# Product onboarding

The default BuildOpt experience has one installation step and one command:

```text
install BuildOpt -> open a Gradle repository -> buildopt gradle build
```

The package contains the launcher, Gradle plugin and init script, Build Impact,
the self-hosted server, Edge Cache, and the JVM agent. Users do not need to
clone this repository, build Go or Java sources, copy JARs, or set internal
plugin paths.

## Install

Published packages currently cover Linux AMD64, macOS Intel and Apple Silicon,
and Windows AMD64. Native runtime source remains portable beyond that matrix,
but the installer fails clearly instead of downloading the wrong architecture.

### Linux and macOS

Download the installer, inspect it if required by your organization, and run
it as your user. It defaults to `~/.local` and never uses `sudo`:

```bash
curl --fail --silent --show-error --location \
  --output buildopt-install.sh \
  https://raw.githubusercontent.com/tonyredondo/buildopt/main/install.sh
bash buildopt-install.sh
export PATH="$HOME/.local/bin:$PATH"
buildopt doctor
```

Use `--version 0.2.0` to pin a release or `--prefix /absolute/path` to choose
another user-owned installation. The installer detects the operating system
and architecture, downloads the matching archive and checksum, verifies the
complete archive, verifies its internal files again, and records only the
files owned by BuildOpt for safe removal.

### Windows

In PowerShell:

```powershell
Invoke-WebRequest `
  https://raw.githubusercontent.com/tonyredondo/buildopt/main/install.ps1 `
  -OutFile buildopt-install.ps1
./buildopt-install.ps1 -UpdatePath
buildopt doctor
```

Open a new terminal after `-UpdatePath`. `-Version 0.2.0` pins a release and
`-Prefix C:\absolute\path` changes the installation root.

## Run the first build

Change to a Gradle repository that contains its Wrapper and run:

```bash
cd /path/to/your-gradle-repository
buildopt gradle clean build
buildopt gradle clean build
```

On Windows the same command discovers `gradlew.bat`; on Linux and macOS it
discovers `gradlew`. BuildOpt supplies the packaged init script and plugin,
enables a private local build cache, preserves Gradle output and returns the
Gradle exit code. The first clean build populates qualified entries. The
second removes project outputs again and should show qualified tasks such as
`compileJava FROM-CACHE`.

The default cache lives below the operating system's user cache directory.
BuildOpt hashes the canonical repository path and Wrapper/platform
compatibility into separate opaque scopes, retains entries through Gradle's
native seven-day cleanup, and never requires a network service. Any ordinary
Gradle argument can follow `buildopt gradle`:

```bash
buildopt gradle --no-daemon clean assemble
```

Disable only the automatic build cache while retaining the packaged launcher
and plugin:

```bash
buildopt gradle --no-build-cache build
```

Immediate bypass keeps the same entrypoint while removing BuildOpt's plugin
and optimization path:

```bash
BUILDOPT_BYPASS=1 buildopt gradle build
```

In PowerShell use `$env:BUILDOPT_BYPASS = '1'` for that invocation.

## Add CI

### GitHub Actions

Pin the Action source to a reviewed 40-character commit SHA. No release inputs
are required for the normal path:

```yaml
- name: Install BuildOpt
  uses: tonyredondo/buildopt@<40-character-commit-sha>

- name: Build
  run: buildopt gradle --no-daemon build
```

Set `version` when the workflow must pin a particular binary release. The
Action resolves the published archive checksum, verifies the archive and adds
the installed commands to `PATH`. Existing consumers that provide the legacy
`archive-url` and `archive-sha256` pair remain supported.

### GitLab CI

The component is self-contained; the consuming repository does not copy
BuildOpt scripts:

```yaml
include:
  - project: tonyredondo/buildopt
    ref: <40-character-commit-sha>
    file: /.gitlab/buildopt-component.yml
    inputs:
      gradle-tasks: build
```

The component installs the matching public release into `.buildopt/runtime`,
runs `buildopt gradle`, and retains the normalized job event and redacted
exports. Add `version: 0.2.0` to pin the native package.

## What improvement should you expect?

The current scorecard measures mechanisms separately so one feature cannot
hide another's cost:

| Mechanism | Measured result | What the comparison proves |
|---|---:|---|
| Safe cache, Kotlin | 15.9% faster than cache-off; 0.02% faster than native cache | Reuse helps and the safety layer is within the 2% native-parity guardrail |
| Safe cache, Groovy | 13.7% faster than cache-off; 0.47% slower than native cache | Reuse helps and the safety layer is within the 2% native-parity guardrail |
| Runtime Tuning | 0.7% faster | The bounded resource profile saved 66 ms with a positive lower 95% bound, identical artifacts and no OOM regression |
| Build Impact | 27.6% faster | Omitting one declared unaffected project saved 2.254 s with required outputs unchanged |

These are controlled POC workloads, not universal predictions, and their
percentages must not be added. A repository that needs the full graph receives
no Build Impact saving; an already warm native cache is a parity baseline, not
the cache-off comparison. Run `./dev/check-build-optimization-performance` to
validate and print the scorecard without rerunning Gradle. The
[benchmark index](../../benchmarks/README.md#build-optimization-scorecard)
links every raw observation and measurement contract.

## Component ownership and configuration

Start with only the launcher. Add another component when a concrete need
justifies operating it.

| Component | Installed with | Configured in | Orchestrated by | First proof |
|---|---|---|---|---|
| Launcher, safe local cache and Gradle plugin | Native package or CI integration | Environment; defaults work locally | `buildopt gradle` | Run the same build twice and inspect normal Gradle output |
| Runtime tuning | No separate installed CLI; owner evaluation in this source repository | Bounded policy inputs and rollout contracts | `runtime-evaluation` and owner checks | Produce evaluation evidence before any product integration |
| Task Intelligence | No separate installed CLI; owner evaluation in this source repository | Qualification evidence and trace contracts | `task-intelligence-evaluation` and owner checks | Prove qualification without publishing an optimization |
| Build history | `buildopt-server` in the package | Server config and export directory on the server host | Service manager plus server API/dashboard | Start loopback server and inspect a redacted session |
| Build Impact | `buildopt-impact` in the package | Manifest and generated graph committed in the target repository | CI calls `generate` during adoption and `check` afterward | Generate, review, commit, then run `check` |
| Patch Autopilot | Java patcher and owner workflow in this source repository; not yet a native-package CLI | Recipe registry, signing/trust material and repository policy | Owner-controlled candidate/validation workflow | Produce a draft bundle; applying remains explicit and reversible |
| Shared Cache | `buildopt-server` | Private server config, authority and scoped credentials on operator hosts | Server service plus launcher gateway | Validate config, start loopback service, run an authorized repository |
| Edge Cache | `buildopt-edge` | Private Edge config and authority on the Edge host | launchd, Windows SCM, or foreground process | `buildopt-edge validate`, then `serve` and `status` |
| JVM agent | Native package | Enabled only by the relevant policy | Launcher/plugin path | Use the owning runtime workflow; no separate installation |

The [product workflows](../guides/product-workflows.md) contain the detailed
commands for Build Impact, history, runtime policy, Patch Autopilot and Edge.
The [configuration reference](../reference/configuration.md) separates safe
defaults from operator-owned credentials and private configuration.

## Recommended adoption order

1. Run `buildopt doctor` and one local `buildopt gradle` build.
2. Put the same command in one CI job and keep `BUILDOPT_BYPASS=1` as rollback.
3. Collect build history before changing policy so improvements have a
   baseline.
4. Evaluate runtime tuning, Task Intelligence and Patch Autopilot through their owner workflows; they are not first-run package switches.
5. Add Build Impact independently and compare it against the history baseline.
6. Operate Shared or Edge only when local/CI reuse justifies a persistent
   service and its credential lifecycle.

This ordering keeps the first result local and reversible. Persistent storage,
network access, credentials and repository mutation are never prerequisites
for the first build.

## Develop BuildOpt itself

Source bootstrap, the synthetic product lab and repository validation belong
to contributors, not first-time product users. Follow
[developer onboarding](./developer-onboarding.md) when changing BuildOpt, or
the [source evaluation path](./quickstart.md#source-based-product-lab) when
you need the complete project-owned synthetic proof.
