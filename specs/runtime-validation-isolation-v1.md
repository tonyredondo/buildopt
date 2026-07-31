# Runtime validation isolation v1

This POC contract closes `B-001`. The scheduler persists every idempotent
request before a worker may lease it, permits one live validation lease per
repository, and recovers expired leases without duplicating a completed
request.

Candidate and control receive distinct mode-`0700` workspaces, output roots,
writable `GRADLE_USER_HOME` directories, and mode-`0600` credential files.
Candidate writes only `quarantine/{actionId}/{attemptId}`. Control reads a
policy-fixed `control/action/{policyDigest}` namespace with `push=false`.
Neither arm can write `stable/{platform}/{cacheCompatibilityClass}`; the stable namespace
is represented only as the normal trusted path outside validation.

Control is authoritative for the validation result. Candidate artifacts cross
the boundary only through later digest comparison. Scheduling does not enable
an action, publish candidate bytes, or claim causal evidence.

Run `./dev/check-runtime-validation-isolation` for the executable contract.
