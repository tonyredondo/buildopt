# Owner-controlled pilot deployment v1

This specification closes `A1-001` by binding one immutable synthetic Gradle
repository to one signed and installed `PRIVATE_BETA_ISOLATED` BuildOpt release.
The repository is private and controlled by the project owner, so it supplies
repeatable POC evidence without importing another project's code or requiring
an external design partner.

## Repository and release boundary

[`tonyredondo/buildopt-pilot`](https://github.com/tonyredondo/buildopt-pilot)
pins Gradle 9.6.1, Java 17, eight linearly dependent projects, 64 main classes,
eight test classes, one custom cacheable task, and three deterministic
deliverables. Its immutable pilot manifest binds source revision
`e79a7bcc5eb31e838d4d886245d514fcdef3fc73` to BuildOpt revision
`5f4105a78dac26dd077ad886e8aa15549423c1fc`.

BuildOpt version `0.1.0-pilot.1` is packaged with its SPDX SBOM and provenance,
signed by an external key, verified, and installed outside both repositories.
Signing material, ingest credentials, managed state, L1 entries, logs, and
session exports remain private machine state and are never committed or sent
to GitHub. The pilot workflow has read-only permissions and no BuildOpt
credential.

## Recorded exercise

The pilot repository's `./dev/check` first proves all tests, deterministic
outputs, an unchanged worktree, and local-cache replay. The installed BuildOpt
launcher then runs the declared workload twice through the authenticated
Gradle plugin and private native managed L1. Both runs must succeed, emit valid
`BUILD_SESSION v1` records, and produce the same distribution bytes. The
second run must restore all eight main `compileJava` tasks from L1. The custom
`generatePilotManifest` task must execute under Tier 1 default deny rather
than being published or restored.

The immutable non-secret result is
[`a1-001-owner-controlled-pilot.json`](../benchmarks/results/a1-001-owner-controlled-pilot.json).
Validate the repository-owned contract and result with:

```bash
./dev/check-owner-controlled-pilot-deployment
```

The hosted GitHub job was blocked before executing any step by the account's
billing/spending-limit state. That external pre-execution condition is
recorded separately from the locally passing workflow and product evidence;
it is not reported as a source-code failure or silently avoided by making the
private repository public.

This deployment does not activate signed Shared authority, run the deferred
eight-hour soak, establish external-user evidence, or claim causal savings.
Those boundaries leave `A1-006` and `A1-G06` open for the subsequent
owner-operated evaluation.
