# GitHub automation

The repository-root [`action.yml`](../action.yml) installs BuildOpt on Linux
x64 runners. Consumers pin the Action source to a reviewed commit; the
one-command POC path needs only `command: optimize build` and resolves the
latest native package plus its published SHA-256. It derives provider
repository/base/head facts, restores exact state through an immutable
`actions/cache` revision and publishes checksummed machine and customer value
reports. Restored
state remains untrusted until every launcher binding passes. `version` pins
native bits, while the paired `archive-url` and
`archive-sha256` inputs preserve the historical signed Release Bundle v1 path.
The Action verifies the complete archive and its internal manifest before
exposing the launcher, Build Impact, server, Edge, plugin, agent and init
script. [`release.yml`](./workflows/release.yml) creates the native packages on
Linux, macOS and Windows when a semantic-version tag is pushed.

Omitting `command` preserves the install-only compatibility surface. The
one-command path requires no service or credential and never uploads the
private portfolio as a review artifact. See the
[CI onboarding contract](../specs/poc-magic-ci-onboarding-v1.md).

The same Action exposes an explicit `profile-proposal` mode for Linux CI. It
builds the CLI from the Action's immutable source commit, reads the consumer's
checked-in `.buildopt/profile.json`, derives the exact base-to-target change
and uploads a review bundle. Before graph discovery it executes the declared
workflow once and includes `buildopt-output-contract.json`; empty or ambiguous
required outputs retain native and cannot reach measurement. The mode never
measures or activates a profile; unsupported or incomplete discovery is a
successful `NATIVE_FULL_GRAPH` decision. Operational failures upload
diagnostics when possible and then fail the job. See the
[owner-input contract](../specs/poc-generic-owner-input-v1.md) and
[CI proposal contract](../specs/poc-generic-profile-ci-v1.md).

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
[replay contract](../specs/poc-generic-profile-ci-replay-v1.md). Hosted run
[`31467370391`](https://github.com/tonyredondo/buildopt/actions/runs/31467370391)
passed all five repository jobs and the aggregate summary with zero drift.

[`generic-workflow-breadth-fixture.yml`](./workflows/generic-workflow-breadth-fixture.yml)
is the manual read-only capability matrix for the shared owner input. It checks
packaging, typed verification, distribution, and build-owned test preparation,
then proves native fallback for an executable workflow whose structural
semantics are unsupported. Its artifact contains no timing or active profile.
Hosted run
[`31598631537`](https://github.com/tonyredondo/buildopt/actions/runs/31598631537)
passed all four supported cells and the native fallback on immutable source
`0c1b64f`.


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
