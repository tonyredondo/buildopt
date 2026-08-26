# Sticky Wrapper verified bootstrap v1

Status: accepted POC implementation contract (`SWL-003`).

This block turns the two generated wrapper scripts into a verified,
user-scoped distribution bootstrap. It deliberately stops before Gradle
passthrough, cache/state connection or optimization decisions.

## Selection and cache identity

The POSIX script selects Linux AMD64, macOS Intel or macOS Apple Silicon from
`uname`. The Batch/PowerShell script accepts Windows AMD64. No fallback chooses
a nearby architecture. An unsupported tuple exits `69` before network access.

The installation key is:

```text
distribution version / platform / archive SHA-256
```

It lives under the operating system's user cache or an explicit
`BUILDOPT_WRAPPER_CACHE_HOME`. That override is an environment value for tests
and controlled runners; it is never committed. A valid warm entry is fully
reverified and causes zero network requests.

## Download and integrity

The initial URL is immutable HTTPS. GitHub's
[release-asset API](https://docs.github.com/en/rest/releases/assets?apiVersion=2022-11-28#get-a-release-asset)
documents downloads as either a `200` response or a `302` redirect, so the wrapper follows at most
five redirects and requires HTTPS for every hop. It applies the committed
5-second connect and 30-second request budgets and process proxy settings. The
pinned outer SHA-256 is checked before any extraction.

TAR and ZIP readers then require one exact platform/version root, bounded
portable relative names and no duplicate, traversal, absolute, link or reparse
entries. The extracted package must contain a regular `bin/buildopt` (or
`buildopt.exe`) and a strict internal `SHA256SUMS`; every listed `bin/` and
`lib/` file is verified before publication. No unverified executable starts.

## Publication and reuse

Only one writer can own the cache-key lock. It downloads and extracts in a
temporary sibling directory, writes a four-line identity marker, renames the
verified directory into place and verifies it again. A concurrent reader waits
for the complete directory and then verifies it. Failure removes the staging
directory and lock. An existing invalid entry fails closed rather than being
silently replaced from the network.

For this interim block, the only successful wrapper operation after bootstrap
is:

```text
./buildoptw --buildopt version [--json]
```

Other invocations exit `70` with an explicit `SWL-004` boundary. Gradle
passthrough and `BUILDOPT_BYPASS=1` equivalence are implemented next; this
prevents `SWL-003` from claiming untested process behavior.

## Evidence

```bash
./dev/check-sticky-wrapper-bootstrap
```

The deterministic Linux fixture exercises online and offline reuse, Linux and
both macOS package selections, two concurrent bootstraps, cached tamper,
archive checksum, internal manifest, traversal, link, redirect, interrupted
download and unsupported-platform failures. Go tests keep both templates under
their encoding and 32-KiB contracts; race tests and `go vet` cover the
generator integration.

Real public smokes download BuildOpt `v0.6.1`, validate its external and
internal digests and then reuse it after replacing the committed URL with an
unreachable HTTPS host. The same flow passes for the POSIX TAR and the Windows
PowerShell ZIP body. Native macOS and Windows execution is part of the pinned
`Native Platform CI` gate.

## POC boundary

`VERIFIED_BOOTSTRAP_ACCEPTED` is a functional onboarding result, not a build
time result. It grants no production authority and adds no soak, design-partner
or Test Optimization requirement.
