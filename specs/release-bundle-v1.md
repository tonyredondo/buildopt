# Release Bundle v1

**Owner:** `F0-038`<br>
**Decision:** `DEPLOY-001`<br>
**Platform:** `linux-amd64`

## Purpose

Release Bundle v1 is the minimum verifiable distribution contract required
before the first GitHub Action can install `buildopt`. It packages only the
components that exist at this Phase 0 gate:

- the `buildopt` launcher and local gateway;
- the owner-operated `buildopt-edge` cache process;
- the `buildopt-server` modular monolith;
- the `dev.buildopt` Gradle plugin JAR;
- the opt-in JVM agent JAR.

The Rust helper, patcher JAR, workflows, and release revocation remain absent
until their owning gates materialize them. The separate `DEPLOY-001` lifecycle
consumes this bundle without adding placeholder payloads to it.

## Bundle layout

For version `<version>`, the release directory contains exactly:

```text
buildopt-<version>-linux-amd64.tar.gz
buildopt-<version>-linux-amd64.spdx.json
buildopt-<version>-linux-amd64.provenance.json
buildopt-<version>-linux-amd64.release.json
buildopt-<version>-linux-amd64.sha256
buildopt-<version>-linux-amd64.sha256.sigstore.json
```

The TAR has one root and no links or additional files:

```text
buildopt-<version>-linux-amd64/
├── bin/
│   ├── buildopt
│   ├── buildopt-edge
│   └── buildopt-server
└── lib/
    ├── buildopt-gradle-plugin-<version>.jar
    └── buildopt-jvm-agent-<version>.jar
```

Directories and binaries have mode `0755`; JARs have mode `0644`. Both JVM
manifests carry the requested `Implementation-Version`. No credential,
private key, source file, optional component, or host path is included.

## Reproducible inputs and outputs

`dev/package-release` accepts a SemVer version and requires a clean Git
checkout. The source revision and `SOURCE_DATE_EPOCH` equivalent come from
the committed `HEAD`; callers cannot assert a different revision. Go builds
use the locked compiler with `CGO_ENABLED=0`, `GOOS=linux`, `GOARCH=amd64`,
`-mod=readonly`, `-buildvcs=false`, and `-trimpath`. Gradle uses the locked JDK,
the checksum-pinned Wrapper, offline dependency resolution, Java 17 bytecode,
and the requested release version.

The TAR is sorted, has root ownership, fixed modes, and commit-derived
timestamps; gzip omits its timestamp and original filename. Syft 1.50.0 emits
SPDX 2.3 from the exact payload tree. Its document namespace is derived from
the archive SHA-256 and its creation time is normalized to the source commit.

Two invocations from the same revision, version, platform, and toolchain must
produce byte-identical archive, SBOM, provenance, release manifest, and
checksum manifest. Cosign's ECDSA signature bytes may differ, but every valid
signature must bind the same checksum-manifest digest.

## Provenance

The provenance sidecar is an in-toto Statement v1 with the SLSA provenance v1
predicate type. It binds:

- the archive and SPDX document names and SHA-256 digests;
- version and `linux-amd64` platform;
- the source repository and exact 40-character Git commit;
- the commit-derived epoch;
- the locked Go, JDK, Gradle Wrapper, Syft, and Cosign artifacts;
- the revision-qualified `dev/package-release` builder identity.

This is deterministic build provenance, not a claim of a SLSA level or an
independent hosted build service.

## Signature and trust profile

The SHA-256 manifest covers the archive, SBOM, provenance, and release
manifest. Cosign 3.1.2 signs that manifest with a caller-supplied private key
and emits a Sigstore bundle v0.3. The private key must not be readable by group
or other users and is never copied into the output.

Phase 0 uses an explicit local signing configuration with no Fulcio,
transparency-log, or timestamp services. The Sigstore bundle must therefore
contain zero transparency-log entries and zero RFC 3161 timestamps. Verification
pins a separately distributed public key, compares its SHA-256 with the release
manifest, and invokes Cosign with the local-key/no-transparency-log profile.
The public key is deliberately not accepted from the release directory: a
bundle cannot nominate its own trust root.

This profile matches the RFC's isolated private-beta boundary. Public
transparency, OIDC identity, KMS/HSM custody, DSSE, independent attestations,
rotation, and revocation remain hardening work and must not be inferred from
this signature.

## Verification

`dev/verify-release` fails closed before any packaged executable runs. It
requires:

1. exactly the six version-derived regular files and no symlinks or nesting;
2. a structurally valid Release Bundle v1 manifest;
3. the externally pinned public-key fingerprint;
4. a valid Cosign signature over the canonical checksum manifest;
5. exact SHA-256 values for every covered file;
6. an SPDX 2.3 document naming all four payload artifacts;
7. provenance that binds subjects, source, builder, and locked tools;
8. the exact safe TAR layout, modes, and JAR implementation versions.

Missing, extra, renamed, traversing, unsigned, mismatched, or tampered content
is rejected with status `1`. Invalid CLI usage returns `64`.

## Commands

Provision and test the locked tools:

```bash
./dev/bootstrap --toolchain cosign
./dev/bootstrap --toolchain syft
./dev/check-supply-chain-toolchains
./dev/test-supply-chain-toolchains
```

Create a Cosign key outside the repository, then package from a clean checkout:

```bash
COSIGN_PASSWORD='<secret>' \
  ./dev/run --toolchain cosign -- \
  cosign generate-key-pair --output-key-prefix /secure/path/buildopt-release

COSIGN_PASSWORD='<secret>' \
  ./dev/package-release \
  --version 0.1.0-dev.1 \
  --output dist/buildopt-0.1.0-dev.1 \
  --signing-key /secure/path/buildopt-release.key \
  --verification-key /secure/path/buildopt-release.pub
```

The example path and password are placeholders, not repository defaults.
Production signing material must be supplied by the release environment.

Verify without access to the private key:

```bash
./dev/verify-release \
  --bundle dist/buildopt-0.1.0-dev.1 \
  --key /trusted/path/buildopt-release.pub
```

The executable conformance suite is `dev/check-release-package`.

## GitHub Actions consumption

The repository-root setup Action consumes the TAR through three independent
workflow inputs: release version, HTTPS archive URL, and lowercase SHA-256. The
consumer pins the Action itself by full commit SHA. The Action verifies the
archive checksum and exact safe TAR layout before extraction, then publishes
the launcher/server/plugin/agent paths without executing them.

Release authentication remains outside the download step: the checksum entered
in a customer workflow must come from a separately trusted verification of this
six-file bundle and its pinned public key. An archive and checksum retrieved
from the same unauthenticated source are not sufficient provenance.

`WS-007` proves this setup boundary with a synthetic layout-compatible archive.
`DEPLOY-001` proves signed install, upgrade, rollback, and uninstall locally
through `dev/check-deployment-lifecycle`. Public release publication and online
revocation remain later operational work.
