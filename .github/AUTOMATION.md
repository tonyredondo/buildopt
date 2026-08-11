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

The same Action exposes an explicit `profile-proposal` mode for Linux CI. It
builds the CLI from the Action's immutable source commit, reads the consumer's
checked-in `.buildopt/profile-ci.json`, derives the exact base-to-target change
and uploads a review bundle. The mode never measures or activates a profile;
unsupported or incomplete discovery is a successful `NATIVE_FULL_GRAPH`
decision. Operational failures upload diagnostics when possible and then fail
the job. See the [CI proposal contract](../specs/poc-generic-profile-ci-v1.md).

[`profile-proposal-fixture.yml`](./workflows/profile-proposal-fixture.yml) is a
manual, read-only hosted conformance run for that mode. It creates an external
clean Gradle checkout below runner temp, uploads the generated proposal and
asserts that no active profile exists. Hosted run
[`31464264563`](https://github.com/tonyredondo/buildopt/actions/runs/31464264563)
passed from immutable source `f6a2c5e` with an 11-file checksummed artifact.

[`profile-proposal-replay.yml`](./workflows/profile-proposal-replay.yml) is the
manual five-repository adoption replay. It creates clean Spring,
OpenTelemetry, Kafka, Micronaut, and Groovy checkouts, commits each frozen
repository-owned input before its source change, and invokes the same root
Action. Every job uploads both the proposal and a compact `MATCH` or `DRIFT`
verdict; the summary requires all five reference graphs to match. It performs
no timing and never activates the proposals. See the
[replay contract](../specs/poc-generic-profile-ci-replay-v1.md).


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
