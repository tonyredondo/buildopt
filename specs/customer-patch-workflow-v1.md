# Customer patch workflow v1

This POC contract closes C4-007 and the draft-delivery gates C4-G01, C4-G03,
and C4-G04. The customer installs the inert reference workflow from its
protected default branch and invokes it explicitly with the exact validation
run, artifact name, action ID, and full trusted revision. Forks cannot run it.
There is no push, schedule, pull-request, or pull_request_target trigger.

The workflow starts with no permissions. Its only job receives actions:read,
contents:write, and pull-requests:write; GITHUB_TOKEN is exposed only to the
final materialization step. Checkout uses the protected default branch with
persisted credentials disabled. The signed PatchBundle is downloaded by exact
workflow run and artifact name, then the installed BuildOpt materializer must
reverify its signature, source state, revision, action, validation, and patch
digest before creating only buildopt/<actionId> and a draft pull request.

The deterministic PR body contains Change, Evidence, Risk, Validation,
Expected impact, and Rollback sections. PRELIMINARY is an allowed label for
review after correctness gates, but is never reported as confirmed causal
savings. The materializer contract has no operation for rebase, force-push,
default-branch update, marking ready, or merge. Missing permissions or any
binding conflict retains the downloadable bundle and returns PROPOSED.

Run `./dev/check-customer-patch-workflow`.

The checker validates the machine-readable policy, exact workflow shape,
pinned actions, minimal permission/token boundary, safe trigger and branch
guards, bounded invocation flags, and forbidden operations. It composes the
production PatchBundle, candidate, and full relevant validation gates. The
fixture is validated as data and performs no remote mutation.
