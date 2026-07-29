# GitHub automation

The repository-root [`action.yml`](../action.yml) is the `WS-007` Linux x64
setup Action. Consumers pin commit
`3fe068790878420a2a9e1d84b6ae5fc83f5752c3` and independently pin a Release
Bundle v1 archive by version, HTTPS URL, and SHA-256. The Action verifies the
complete archive and exact safe layout before exposing `buildopt`, the server,
plugin, agent, and Gradle init script.

[`ws-007-fixture.yml`](./workflows/ws-007-fixture.yml) is a manual,
read-only hosted conformance fixture. It pins every referenced Action by a full
commit SHA and receives no BuildOpt secret or write permission. It is not
normal CI and does not run on repository pushes or pull requests.

[`base-ci.yml`](./workflows/base-ci.yml) is the authoritative `F0-004`
push/pull-request workflow. Its read-only core lane provisions the locked Go,
JDK 21, ShellCheck, and actionlint tools, compiles Java 17 bytecode, and loads
the agent on an exact Java 17 compatibility runtime. Its separately named Rust
lane installs and verifies the optional locked compiler. All external Actions
are full-commit pinned in [`base-ci.lock.json`](./base-ci.lock.json), and
`./dev/check-base-ci --static` rejects routing, permission, runner, version, or
pin drift.

`CI-ORCH-001` still owns protected validation scheduling, isolation, budgets,
and recovery. Base CI does not claim that broader gate.

[`CODEOWNERS`](./CODEOWNERS) and [`OWNERS.md`](./OWNERS.md) define the Phase 0
path routing, accountable repository owner, cross-workstream review lenses, and
the boundaries that one repository principal cannot authorize alone. Validate
them with `./dev/check-ownership`.
