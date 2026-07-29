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

`ENV-010` provides locked ShellCheck and actionlint binaries.
`F0-004` still owns reproducible authoritative checks, while `CI-ORCH-001`
owns protected validation scheduling, isolation, budgets, and recovery. No
workflow in this directory claims either gate.
