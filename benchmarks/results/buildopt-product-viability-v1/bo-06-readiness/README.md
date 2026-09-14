# Checkstyle correctness and readiness for the longer experiment

14 September 2026 · BO-06 remains **partial**.

The current Checkstyle correction passes the additional checks for native cache
restoration, changed configuration and disabling/reactivating the correction.
The longer experiment is still blocked by measurement qualification. The short
comparison's savings remain valid within their exploratory scope.

## What is now verified

Eight small Gradle builds ran in a separate fixture. Each compared the complete
XML reports from ordinary Checkstyle and the corrected version on the same two
files. All builds succeeded and every report comparison matched. A fresh compile
also reproduced all eight runtime classes used by the fixture exactly.

| Situation | Observed behavior |
|---|---|
| First build | Both versions checked both files and produced the same report. |
| Restore outputs from Gradle's cache | Gradle restored the reports and removed the correction's local history. No file checking ran. |
| Edit after that restore | Both files were checked again. The correction did not reuse history that Gradle had removed. |
| Run again with unchanged files | Ordinary Checkstyle checked both files; the correction reused both results and kept the full report. |
| Change configuration bytes | The correction fell back to full checking and saved no successful history. This case changed a comment; earlier tests cover changed rule behavior and diagnostics. |
| Restore the supported configuration | Full checking established fresh history. |
| Disable the correction and edit a file | Both files were checked normally. Disabling retained the old, inactive history; it was not an uninstall. |
| Reactivate it | The changed file was checked and the unchanged file reused. Reports still matched. |

These are correctness checks, not performance measurements. They used Gradle
9.7.1 and Java 21 with Configuration Cache disabled. The eight builds took
about three minutes in total. They consumed no Elasticsearch history.

The [coverage record](./analysis/correctness-coverage.json) distinguishes these
new checks from the retained evidence: twenty original correctness cases, six
supported finalization cases, exact patch removal, and the complete output
comparisons from BO-05. The original file-selection and diagnostic paths are
unchanged. The later changes to saving history have their own failure,
cancellation and recovery checks. This does not establish support for other
Gradle versions or a finished installation/uninstallation experience.

## What is fixed for validation

The Elasticsearch history still contains the same anchor and 100 changes.
Its commit metadata and changed-path digests match the earlier frozen history.
The six replication histories also have 101 revisions each, with the original
endpoints and order: Apache Groovy, Apache Kafka, Spring Framework,
OpenTelemetry Java, Micronaut Core and Hibernate ORM.

Only Git metadata was read to verify these histories. No protected source was
inspected to choose a correction and no validation timing was read or produced.
The replication rule remains the first two qualifying repositories in that
order, with all six outcomes retained. This verifies history availability;
it does not establish an opportunity or buildability in those repositories.

The [preserved scientific contract](./inputs/scientific-contract.json) keeps
the candidate, the 20/80 split, two independent complete repetitions, all
scheduled outcomes, cost accounting and every savings criterion. Each
repetition still needs at least 5% net saving and one second per scheduled
validation build, at least 76 comparable results out of 80, positive statistical
support and the existing limit on slower builds. Preparation must be recovered
within the sequence. No criterion has been weakened.

This is a partial freeze. A launchable confirmation protocol still needs
qualified measurement, a complete development-prefix check for this version,
and a feasible allocation for 404 builds plus comparison work. The retained
manifest is deliberately named `confirmation-not-admitted.json`. Its copied
short-screen allocation is expired and insufficient for confirmation. The
existing executable rejects it because the required overhead qualification is
missing; validation creates no run directory and starts no build.

## Why BO-06 cannot close yet

The old qualification tried to establish that recording the experiment added
almost the same delay to both versions: no more than a 100 ms difference. Its
first result missed that limit. A subsequent fixed-size pilot estimated that
25,656 pairs per version would be needed under its registered calculation,
above its maximum of 512. That attempt correctly stopped.

Later short comparisons deliberately used an exploratory measurement mode.
They did not pass the failed qualification. They also included a Java agent
that recorded detailed phases inside Checkstyle. Those measurements cannot
simply be renamed as primary confirmation data. The stricter process-sampling
coverage requirement is also unqualified. None of this demonstrates a new
defect in Checkstyle or disproves the observed savings.

The [readiness record](./analysis/measurement-readiness.json) retains the exact
failed results and the missing prerequisites. G0 is the unresolved measurement
and execution prerequisite. G3 is the result of the future longer experiment;
we do not require G3 to pass before running it.

The next decision is whether to adopt the [proposed measurement change](./measurement-decision.md).
It preserves the existing product-value criteria and narrows the initial claim
to builds with the declared recording enabled. It explicitly changes the
measurement admission rules, so it is a proposal for approval, not an active
replacement protocol. BO-07 remains deferred.

## Evidence and accounting

This block added eight fixture Gradle starts, one Java compiler command and no
standalone comparison JVMs or Elasticsearch builds. Every fixture service
closed. The candidate, earlier results and subject worktrees were preserved.
The new local state was under 4 MiB at the initial check, within the 1 GiB limit;
the final [verification](./analysis/verification.json) records the retained size.

The [history verification](./analysis/history-verification.json),
[eight boundary checks](./analysis/boundary-checks.json),
[source-to-runtime proof](./receipts/compiled-source-proof.json) and
[confirmation refusal](./receipts/confirmation-refusal.json) are included here.
Raw fixture logs, reports and the frozen scripts are retained alongside them.
This export references earlier local inputs; it is not a standalone package
for reconstructing every historical experiment.
