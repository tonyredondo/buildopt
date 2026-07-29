# Phase 0 base recovery

This runbook implements `F0-039` and the local safety boundary accepted in
`OPS-001`. It covers the current Linux AMD64 launcher and checksum-pinned
GitHub Action. It does not claim the online revocation deadline, durable
attempt/lease cleanup, a production release channel, or the installation
lifecycle owned by `OPS-001/A1`, `WS-008`, and `DEPLOY-001`.

## Choose the smallest recovery action

| Condition | Action | Keeps BuildOpt installed? | Persistent data |
|---|---|---:|---|
| Product path is suspect and the build must run now | Local bypass | Yes | Unchanged |
| Every new CI invocation must stop using the product path | CI kill switch | Yes | Unchanged |
| A newly selected Action or release is bad | Roll back the complete immutable pin set | Yes, at the known-good version | Unchanged |
| The integration must be removed | Uninstall the workflow integration and release | No | Preserve by default; purge only explicitly |

Do not combine these actions unless the narrower action fails. Capture the
failing invocation, selected Action commit, release version, archive URL, and
archive SHA-256 before changing them.

## Local bypass

Set the launcher-owned variable to the exact value `1` on the affected
invocation:

```bash
BUILDOPT_BYPASS=1 buildopt run -- ./gradlew build
```

The launcher consumes and removes `BUILDOPT_BYPASS`, removes every other
launcher-only `BUILDOPT_*` credential or rendezvous value, and executes the
original argv directly. It does not create the plugin handshake or local
gateway and does not parse or contact configured session-ingest or policy
services. Standard streams, working directory, arguments, process-group signal
forwarding, and the child exit status remain unchanged.

Only the exact value `1` activates bypass. Remove the variable to return to the
normal path. A value such as `true` is not a bypass request. Compare the
bypassed build's requested tasks, exit status, and outputs with the pre-BuildOpt
command before declaring recovery.

## CI kill switch

The Phase 0 kill switch is a CI-owned variable mapped to the launcher's local
bypass. It remains independent of the BuildOpt control plane:

```yaml
env:
  BUILDOPT_BYPASS: ${{ vars.BUILDOPT_EMERGENCY_BYPASS }}

steps:
  - name: Run the existing Gradle command
    run: buildopt run -- ./gradlew build
```

Activate `BUILDOPT_EMERGENCY_BYPASS=1` at the narrowest repository or
environment scope that contains the incident. Confirm that the next invocation
runs the baseline command and that no BuildOpt gateway, plugin-handshake, or
server-ingest diagnostic appears. Deactivate by removing the variable after a
known-good normal-path canary passes.

This switch affects invocations that start after the CI variable is read. It
does not revoke an already-running invocation and does not satisfy the
control-plane propagation guarantee of at most 60 seconds; that remains an
`OPS-001/A1` gate.

## Immutable version rollback

1. Select a previously exercised tuple from deployment evidence:
   the full 40-character Action commit, exact release version, HTTPS archive
   URL, and lowercase SHA-256 from the separately authenticated publication.
2. Change all four values together in one reviewed workflow change. Never roll
   back to `main`, a branch, or a mutable tag, and never retain the failed
   release URL with a known-good checksum.
3. Run the baseline Gradle command once with `BUILDOPT_BYPASS=1` while the
   rollback is prepared.
4. Install the known-good tuple in a new job, verify the Action outputs and
   archive checksum, then run the same requested tasks first in bypass and next
   as a normal-path canary.
5. Keep the kill switch active if installation, outputs, requested tasks,
   artifacts, exit status, or diagnostics differ from the recorded known-good
   result.

The GitHub Action installs under `runner.temp`, so a fresh job cannot reuse the
failed job's release directory. The recorded exercise intentionally rejects a
candidate with a wrong digest and then installs and executes the pinned
known-good fixture.

## Uninstall and state choice

Restore the consumer workflow before deleting any files:

1. Replace `buildopt run -- ./gradlew ...` with the original `./gradlew ...`
   argv.
2. Remove `--init-script "$BUILDOPT_GRADLE_INIT_SCRIPT"` and every BuildOpt
   plugin, agent, server, or export variable from the Gradle step.
3. Remove the `Setup BuildOpt` Action step and its outputs.
4. Stop any separately started `buildopt-server` process.
5. Run the original Gradle command and compare its requested tasks, exit status,
   and outputs with the bypass result.

The setup Action's release lives below
`$RUNNER_TEMP/buildopt-action/buildopt-<version>-linux-amd64` and is normally
discarded with the job. If a self-hosted runner retains that job directory,
resolve the exact `install-root` Action output and delete only that path after
verifying it is below the expected runner-temp directory. Never recursively
delete an unset variable, the runner-temp root, a home directory, or a guessed
default.

Current persistent-state choices are explicit:

| Path or state | Preserve (default) | Purge |
|---|---|---|
| Action release root under runner temp | Not required after the job | Delete the exact verified `install-root` |
| `buildopt-server --export-dir` | Archive or retain the customer-selected directory | Stop the server, verify the exact directory, then delete only after explicit evidence-retention approval |
| Any repository `.buildopt/` directory | Retain; the current launcher does not create it | Treat it as customer state and require a separately reviewed exact path |
| In-memory `buildopt-server` ingest | Nothing survives server shutdown | No filesystem action |

Uninstall is complete only when the baseline command succeeds without the
Action step, init script, `buildopt` wrapper, or BuildOpt environment values.

## Partial patch branch or pull request

The current repository does not activate C4 patch generation, but recovery is
defined before that feature exists. If a future BuildOpt patch branch or pull
request is only partially created:

1. Enable the CI kill switch and do not merge the patch.
2. Preserve its head SHA, logs, patch bundle, validation result, and linked
   evidence.
3. Close or mark the pull request superseded using the normal repository
   process. Do not force-push, rebase, or write the default branch as recovery.
4. Delete the remote branch only after confirming that policy permits deletion
   and the evidence is retained; branch deletion is not required to restore
   builds.
5. Resume only from a fresh branch based on the current target SHA after the
   root cause and rollback evidence are reviewed.

## Recorded exercise

Run:

```bash
./dev/check-base-runbooks
```

The checker builds the real launcher, records active and cleared kill-switch
paths, proves a bypassed non-zero child plus signal cleanup, rejects a bad
release digest before restoring the pinned fixture, and exercises both
preserve-state and explicit-purge uninstall paths in a temporary directory. It
also verifies that the repository working tree is unchanged by the exercises.
