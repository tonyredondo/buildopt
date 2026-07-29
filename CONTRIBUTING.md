# Contribution Conventions

## Repository language

All repository content must be written in English. This includes code, owned identifiers, comments, documentation, examples, fixtures, CLI and log messages, commit messages, pull requests, and release notes. Upstream-generated artifacts may be checked in only when required and must not be edited manually.

## Change boundaries

- Every change must cite the tracker, contract, or decision IDs it implements.
- Name every workstream crossed and follow [the ownership and review map](./.github/OWNERS.md); `CODEOWNERS` routes review but does not grant product authority.
- The RFC retains invariants and scope; executable details belong in contracts, specifications, benchmarks, and ADRs.
- Do not copy structs from RFC examples. Generate Go and Java clients from the normative IDLs.
- Build Optimization does not decide `Test` task selection, sharding, retries, or policy.
- An observation backend may degrade to `INCONCLUSIVE`; incomplete evidence never becomes authorization.

## Stack conventions

- Go: binaries live in `cmd/`, unexported implementation lives in `internal/`, and inter-process APIs are defined in `contracts/` first.
- JVM: the plugin, agent, and patcher are separate artifacts under `jvm/`; they must avoid internal Gradle APIs unless a capability is explicitly bounded.
- Rust: only the optional Linux helper lives under `rust/hermetic-helper/`; its presence does not change the core contract.
- Schemas and APIs: field names, public symbols, and error codes are written in English.

## Build and validation rules

- Do not use unpinned global toolchains as implicit dependencies.
- Add the smallest test that proves the changed contract and run the applicable conformance vectors.
- Do not update snapshots, golden vectors, or generated code without inspecting the semantic change.
- Record the command, result, and evidence before moving an item to `DONE`.
- Keep the Gradle baseline and bypass functional before activating any optimization.

## Toolchain changes

- Follow the lock scope and update procedure in [`dev/README.md`](./dev/README.md).
- Change a tool's version, immutable source URL, and SHA-256 together.
- Never add workstation paths, usernames, credentials, or package-manager-specific install locations to the repository lock.
- A pinned candidate is not adopted until its tracker decision and smoke evidence close.
- Validate every lock change with `./dev/check-toolchains-lock`.

## Generated changes

Generated files must identify their source schema or IDL and the reproducible
command. Follow [`GENERATED_CODE.md`](./GENERATED_CODE.md), update the normative
source first, run the manifest-owned generator, inspect source and generated
diffs together, and execute `./dev/check-generated-code`. CI rejects stale or
manually edited generated output.
