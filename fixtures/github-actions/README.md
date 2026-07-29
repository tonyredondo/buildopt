# GitHub Actions fixture

This fixture closes `WS-007` without claiming the authoritative CI topology
owned by `F0-004` and `CI-ORCH-001`.

The repository-root [`action.yml`](../../action.yml) is a Linux x64 composite
setup Action. A consumer pins the Action to the full commit in
[`fixture-lock.json`](./fixture-lock.json), then supplies an exact release
version, HTTPS archive URL, and lowercase SHA-256. The Action:

1. downloads only over HTTPS, including redirects;
2. verifies the complete archive before extraction;
3. rejects missing, extra, linked, or wrongly permissioned entries;
4. installs atomically under the runner's temporary directory;
5. reuses only an installation with matching metadata and file checksums;
6. adds `buildopt` to `PATH` and exposes the server, plugin, agent, and pinned
   Gradle init-script paths as outputs.

The synthetic 502-byte archive is deliberately not a product release. It has
the exact Release Bundle v1 TAR layout and executable passthrough/server probes
needed to exercise the setup boundary cheaply on a hosted runner. Real release
contents, SBOM, provenance, signature, and tamper verification remain covered
by `dev/check-release-package`.

The manual-only
[`ws-007-fixture.yml`](../../.github/workflows/ws-007-fixture.yml) grants only
`contents: read`, pins both `actions/checkout` and BuildOpt by full commit SHA,
pins the fixture archive by version and checksum, and proves the installed
wrapper preserves empty, whitespace, wildcard-like, and variable-like argv
plus exit `37`. It receives no BuildOpt token or write-capable
`GITHUB_TOKEN`.

Run the complete local contract with:

```bash
./dev/check-github-action
```

The workflow is dispatched explicitly when hosted evidence is required. It is
not normal CI, does not run on pushes or pull requests, and does not close
`F0-004`, `CI-ORCH-001`, token/fork policy, release publication, or the
installation/upgrade/uninstall lifecycle in `DEPLOY-001`.
