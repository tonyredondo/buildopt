# GitHub automation

The repository-root [`action.yml`](../action.yml) installs BuildOpt on Linux
x64 runners. Consumers pin the Action source to a reviewed commit; the normal
path needs no inputs and resolves the latest native package plus its published
SHA-256. `version` pins native bits, while the paired `archive-url` and
`archive-sha256` inputs preserve the historical signed Release Bundle v1 path.
The Action verifies the complete archive and its internal manifest before
exposing the launcher, Build Impact, server, Edge, plugin, agent and init
script. [`release.yml`](./workflows/release.yml) creates the native packages on
Linux, macOS and Windows when a semantic-version tag is pushed.


[`ws-007-fixture.yml`](./workflows/ws-007-fixture.yml) is a manual,
read-only hosted conformance fixture. It pins every referenced Action by a full
commit SHA and receives no BuildOpt secret or write permission. It is not
normal CI and does not run on repository pushes or pull requests.

[`base-ci.yml`](./workflows/base-ci.yml) is the authoritative `F0-004`
push/pull-request workflow. Its read-only core lane provisions the locked Go,
JDK 21, protoc, ShellCheck, and actionlint tools, rejects generated-descriptor
drift, compiles Java 17 bytecode, and loads the agent on an exact Java 17
compatibility runtime. Its separately named Rust lane installs and verifies the
optional locked compiler. All external Actions are full-commit pinned in
[`base-ci.lock.json`](./base-ci.lock.json), and `./dev/check-base-ci --static`
rejects routing, permission, runner, version, or pin drift.

`CI-ORCH-001` still owns protected validation scheduling, isolation, budgets,
and recovery. Base CI does not claim that broader gate.

[`CODEOWNERS`](./CODEOWNERS) and [`OWNERS.md`](./OWNERS.md) define the Phase 0
path routing, accountable repository owner, cross-workstream review lenses, and
the boundaries that one repository principal cannot authorize alone. Validate
them with `./dev/check-ownership`.
