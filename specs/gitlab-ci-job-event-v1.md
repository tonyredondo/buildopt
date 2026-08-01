# GitLab CI job event v1

The adapter converts the bounded GitLab project, pipeline, job, revision, and
time variables into one provider-neutral JSON event. Success, failure,
cancellation, skipped, and manual states are explicit; unknown states fail
closed instead of being reported as success.

The event is written atomically with mode 0600 at one fixed artifact path.
Exact replay is idempotent and different bytes cannot overwrite an existing
event. A differing merge-request source project ID marks the event as a fork;
the record confirms that no credential was consumed and never serializes the
raw environment, URLs, tokens, runner paths, or source content.

Run `./dev/check-gitlab-ci-job-event`.
