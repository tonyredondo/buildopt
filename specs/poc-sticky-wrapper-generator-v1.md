# Sticky Wrapper generator v1

Status: accepted POC implementation contract (`SWL-002`).

This contract implements the maintainer-facing generator defined by the
[repository wrapper contract](./poc-sticky-wrapper-contract-v1.md). It creates
and validates committed wrapper files but deliberately does not download or
execute a BuildOpt distribution. Verified bootstrap remains `SWL-003`.

## Commands

```text
buildopt wrapper init [--server URL --project-scope SCOPE] [--mode auto|observe|off]
buildopt wrapper check
buildopt wrapper update --version VERSION [--allow-downgrade]
```

`init` resolves the latest stable public GitHub release into an exact version,
four immutable asset URLs and four GitHub-provided SHA-256 digests. It refuses
before network access when any target already exists. Server and project scope
must be supplied together; the committed configuration records only the
credential variable name `BUILDOPT_TOKEN`, never its value.

`check` is offline and read-only. It requires all four targets to be regular,
within their size limits, LF-only, without BOM, with the expected POSIX modes
where the platform exposes them, strictly parseable and byte-for-byte
canonical. It never repairs drift.

`update` first runs the same canonical check. A same-version request performs
no metadata lookup and no write. An upgrade resolves one exact stable release
tag and changes only the version, URLs and checksums. Scripts and owner
configuration remain byte-identical. A downgrade requires
`--allow-downgrade`.

## Transaction boundary

Writers acquire an exclusive repository transaction directory, stage and
flush all four files in their target directories, preserve every prior file
and then publish the complete set. Any failure removes newly published files,
restores every prior byte and removes staging, backup and lock state. A stale
lock after process death fails closed and may be removed only after the owner
confirms that no generator is running.

This is transactional final-state atomicity across four filesystem paths, not
a claim that an arbitrary filesystem offers one multi-file rename primitive.
No completed command leaves a mixed old/new state.

## Release metadata boundary

The generator reads JSON metadata from the public GitHub Releases API using
5-second connect and 30-second request limits, process proxy settings and no
redirects. `GITHUB_TOKEN` is optional rate-limit authorization and never enters
generated files or diagnostics. Archive bytes are neither downloaded nor
executed by this block.

## Evidence

The executable gate is:

```bash
./dev/check-sticky-wrapper-generator
```

It runs 15 focused test functions under the race detector, `go vet` and
cross-compiles the generator tests for Windows AMD64 and macOS ARM64. The
matrix covers deterministic repeated generation, read-only checks, every
existing target, script/properties/config drift, owner-config preservation,
same-version idempotence, downgrade authority, pre-resolution rejection,
concurrent writers and an injected mid-publication rollback. A separate real
smoke resolves the public `v0.6.1` metadata and proves `init`, `check`,
same-version `update` and second-init refusal without downloading an archive.

## POC boundary

The outcome is `STICKY_WRAPPER_GENERATOR_ACCEPTED`. It makes no build-time or
production claim. Bootstrap, Gradle passthrough, centralized cache/state and
learning decisions remain later blocks. Test Optimization remains separate.
