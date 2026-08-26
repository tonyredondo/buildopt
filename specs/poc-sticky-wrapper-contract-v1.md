# Sticky wrapper contract v1

## Purpose

This contract closes `SWL-001`. It fixes the repository-owned bytes and command
boundary before the generator or bootstrap is implemented. The canonical
machine contract is
[`poc-sticky-wrapper-contract-v1.json`](./poc-sticky-wrapper-contract-v1.json),
and executable parser fixtures live under
[`fixtures/sticky-wrapper-contract`](../fixtures/sticky-wrapper-contract/README.md).

This is a format and behavior contract, not an acceleration result. It does
not implement `buildopt wrapper`, download a distribution or authorize an
optimization.

## Generated files

The generator will publish exactly four files in one atomic operation:

| Path | Encoding/newlines | Mode | Limit |
| --- | --- | ---: | ---: |
| `buildoptw` | UTF-8 without BOM, LF, one final newline | `0755` | 32 KiB |
| `buildoptw.bat` | UTF-8 without BOM, LF, one final newline | `0644` | 32 KiB |
| `.buildopt/wrapper.properties` | US-ASCII, LF, one final newline | `0644` | 16 KiB |
| `.buildopt/config.toml` | UTF-8 without BOM, LF, one final newline | `0644` | 16 KiB |

NUL, CR, BOM, missing/folded final newline and oversized files reject. LF is
deliberate for both scripts so Git checkout settings cannot change the pinned
surface. POSIX execution comes from the committed executable bit; Windows uses
the `.bat` association.

## Wrapper properties

`wrapper.properties` is not Java Properties. It is a strict portable
`key=value` format with no whitespace, blank lines, comments or escapes. Keys
occur once in this exact order:

```text
schemaVersion
distributionVersion
distributionUrl.linux-amd64
distributionSha256.linux-amd64
distributionUrl.macos-amd64
distributionSha256.macos-amd64
distributionUrl.macos-arm64
distributionSha256.macos-arm64
distributionUrl.windows-amd64
distributionSha256.windows-amd64
network.connectTimeoutMs
network.readTimeoutMs
network.redirectPolicy
network.proxyMode
```

The schema is `buildopt.wrapper/v1`. The version is an exact three-component
numeric version without a moving alias. Every platform has an immutable HTTPS
URL with no userinfo, query or fragment and a lowercase 64-character SHA-256.
The version must appear in every URL path. Connection and read timeouts are
fixed at 5 and 30 seconds. GitHub release downloads may follow at most five
redirects, every hop must remain HTTPS, and the pinned SHA-256 remains the
content authority. GitHub's
[release-asset API](https://docs.github.com/en/rest/releases/assets?apiVersion=2022-11-28#get-a-release-asset)
documents assets as either a `200` stream or a `302` redirect. Proxy discovery
is limited to the process environment. A
proxy URL or credential is never committed.

Unknown, duplicate, security-sensitive or reordered keys reject identically
under the POSIX and Windows parsing contracts.

## Project configuration

`config.toml` uses a deliberately small flat TOML subset: one `key = value` per
line, no comments, blank lines, tables, arrays, inline tables or escapes. Keys
occur once in this order:

```text
schema_version
mode
server_url
project_scope
credential_env
trial_budget_percent
```

`mode` is `auto`, `observe` or `off`. Server URL, project scope and credential
environment name are either all empty or all present. A central URL is HTTPS,
except exact numeric loopback HTTP used by local fixtures. It has no userinfo,
query or fragment. Project scope is a portable two-segment lowercase identity.
The credential name begins with `BUILDOPT_`; its value never enters the file,
arguments, logs or Gradle environment. The trial budget is an integer from 0
through 5 percent.

Absolute POSIX paths, drive paths, UNC paths, `file:` URLs, home-variable
expansions and credential-bearing keys reject. This proves that the generated
surface is independent of checkout and user-machine paths.

## Argument routing

The default is Gradle passthrough. A Gradle task named `status`, `explain` or
`version` remains a Gradle argument:

```text
./buildoptw status
```

Management requires the reserved first argument:

```text
./buildoptw --buildopt status [--json]
./buildoptw --buildopt explain [--json]
./buildoptw --buildopt version [--json]
```

`--gradle` as the first argument is removed and forces every remaining value
to Gradle, including a literal `--buildopt`. An unknown management command is
usage error 64. Arguments after the routing prefix are preserved exactly; no
shell joins, reparses or expands them.

`BUILDOPT_BYPASS=1` is checked before either committed configuration file or a
download. It discovers and invokes the repository's Gradle Wrapper directly,
preserving the child result. Any other value does not enable bypass.

## Maintainer CLI and updates

The future generator exposes only:

```text
buildopt wrapper init [--server URL --project-scope SCOPE] [--mode auto|observe|off]
buildopt wrapper check
buildopt wrapper update --version VERSION [--allow-downgrade]
```

`init` refuses when any target exists. `check` is read-only. `update` first
requires all four current files to be canonical and changes only distribution
version, URLs and checksums. The scripts and owner configuration remain byte
identical. Repeating the same version is idempotent. A lower semantic version
rejects unless `--allow-downgrade` is explicit. Publication of all four files
is atomic; partial new state is never accepted.

## Failure behavior

During a normal Gradle invocation, wrapper-owned bootstrap, network, format or
integrity failure emits a redacted warning and runs the original Gradle Wrapper
without BuildOpt. It never runs an unverified binary. A management command has
no Gradle result to preserve and uses the fixed wrapper-owned codes in the
machine contract. Once Gradle starts, its ordinary or signal exit result wins.

## POC boundary

The wrapper generator belongs to `SWL-002`; verified platform bootstrap belongs
to `SWL-003`; process equivalence and bypass execution belong to `SWL-004`.
This block grants no production, automatic-merge or Test Optimization
authority and makes no wall-time claim.
