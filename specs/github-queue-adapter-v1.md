# GitHub queue adapter v1

`buildopt-server serve --state-dir ABSOLUTE_PATH --github-webhook-secret
ABSOLUTE_PRIVATE_PATH` enables `POST /internal/v1/ci/github/workflow-jobs`.
GitHub must send the `workflow_job` webhook and the installation/token used to
configure it needs Actions read permission.

The adapter verifies `X-Hub-Signature-256` over the unmodified, bounded request
body before decoding it. `X-GitHub-Delivery` is a durable idempotency key; reuse
with different authenticated bytes returns `409`. The lifecycle accepts only
matching `queued`, `in_progress`, and `completed` action/status pairs and keeps
repository, run, attempt, job, revision, and eligibility immutable.

For a runner-assigned job:

```text
ciQueueMs = workflow_job.started_at - workflow_job.created_at
```

The result is `EXACT` and retains runner ID/name, runner-group ID/name, and the
sorted unique provider labels. A job cancelled before `started_at` is
`UNAVAILABLE` with `CI_RUNNER_NOT_STARTED`; local Gradle timing is never used as
a substitute. Missing or conflicting provider state fails closed, so phase B
must remain in fixed cohorts and cannot promote a capacity-consuming decision.

The normalized lifecycle and delivery ledger live in the managed
`control.sqlite` schema v6 and therefore share the established private file,
integrity, deletion, and single-writer boundaries.
