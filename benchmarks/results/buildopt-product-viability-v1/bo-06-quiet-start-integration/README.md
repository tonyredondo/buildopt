# Waiting for a quiet start inside the replay runner

15 September 2026. BO-06 remains partial.

The replay runner now checks the machine before each build. It waits until
there has been a full 30-second window with low CPU, disk and memory pressure,
then checks that the observation is still fresh immediately before launching
the build. This addresses the high pressure observed before the slow build in
the [previous control](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-qualified-measurement/README.md).
It does not establish that the next control will produce stable timings.

The runner also records the CPU and I/O used by the outer process that prepares
and copies results. That process was missing from the earlier observations.
The worker supervisor's existing observations and disk-limit checks continue
unchanged. We have not established how much of its CPU use comes from checking
the directory tree, or changed that check's frequency.

## What happens before a build

The observation starts after the expensive preparation and worker startup.
It reads Linux pressure counters once a second and requires every observed
interval in the last 30 seconds to have no more than 10% pressure for each
resource. A delayed read, a gap over three seconds or a reset counter prevents
that window from qualifying. These counters describe time spent waiting for
resources; they do not identify which process caused the pressure.

The maximum wait is 180 seconds, shortened by the experiment's remaining time.
The final sample must be at most one second old when the worker accepts it.
The receipt belongs to one request, one manifest and one system boot. The
worker rejects missing, mismatched, expired or incomplete evidence before
starting the native process. The independent result checker verifies those
same conditions against the recorded native start.

A timeout, failed observation or cancellation stops the sequence. The runner
keeps the observations and waiting cost, closes its worker processes and does
not turn the refusal into an automatic retry. Every build that does start
still counts, including builds affected by later pressure. The quiet window
is a condition for starting an experiment, not a filter applied to its results.

Waiting is recorded as research preparation outside the measured build request.
It still consumes the experiment's elapsed-time allocation. Resource records
bracket preparation and capture phases, including the observation itself;
their CPU and I/O counters must not be added to elapsed time as if they were
separate work. They cover the outer process, not its children. Recording the
resource receipts also consumes time within the complete experiment.

## Verification and retained evidence

The [verification record](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-quiet-start-integration/verification.json)
lists the commands, results, process counts and retained failures. The
[source difference](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-quiet-start-integration/source.diff)
shows the change from the runner used for the previous control.

The checks cover a real pair of fixture builds through the runner, worker
socket and Linux user service manager. They also cover pressure-read failure,
cancellation, an elapsed allocation and an observation deliberately allowed
to expire before launch. The independent checker rejects missing observation,
cost and resource records. Unit tests cover pressure bursts, exact thresholds,
clock and counter changes, receipt identity, cost placement and admission.
Selected existing tests check command execution, output mismatches, fallback,
source drift, process cleanup and CPU assignment.

All 34 unit tests passed, as did the final runner tests and five selected
compatibility groups. The final pair waited 91.03 seconds before N and 30.01
seconds before I. Both launched within one second of the last sample. This
shows why the wait must count toward the allocation even though it sits outside
the build-time comparison.

Across development and verification, 23 native fixture processes started;
19 used the final source. All 30 owned worker sessions were closed. No
Elasticsearch build or comparator JVM started. The
[fixture records](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-quiet-start-integration/fixture-records.tar.gz)
retain the complete run files, including the failed attempts.

One earlier fixture pair completed and passed its initial output checks, but
its adversarial test restored a removed file with different permissions.
The final integrity check caught that difference. The test now restores the
original permissions. Earlier setup failures and that failed test remain in
the evidence; they are not reported as passing runs.

This block starts no Elasticsearch build or comparator JVM and reads no
protected source changes. The fixture timings describe the test machinery;
they are not evidence of a BuildOpt saving.

To check the frozen source without native workflow starts:

```bash
./dev/check-quiet-start-integration --unit
```

To repeat the bounded integration tests on Linux amd64 with a working user
systemd manager:

```bash
./dev/check-quiet-start-integration --integration
```

The integration command allows two native fixture starts and up to 630 seconds.
It can fail if the machine never provides the required quiet window. It does
not run Gradle or the Elasticsearch workflow. CI runs the unit checks against
the archived source; the live service-manager checks are retained local proof.

## What comes next

The [next control inputs](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-quiet-start-integration/next-control-admission.json)
bind this runner, the quiet-start policy and the existing qualified output
comparator. The control and development proposals share one measurement
identity. The new identity prevents an older control from qualifying them.
Development remains refused until a genuine passing control is attached.

The next block needs a fresh allocation and the complete eight-build control
on the already selected changes 17–20, using identical corrected code on both
sides. The stopping rule remains a difference of at least three seconds and
5% of the faster measured side. Keep all results; if it fails, stop and
investigate instead of repeating the favorable parts.

Only a passing control can admit the complete 42-build development sequence.
The existing proposal's ceilings remain 50 owner builds, 50 comparator JVMs,
ten hours and 80 GiB, with no retries. Quiet waiting and all preparation consume
that allocation. These are future execution limits; no part of that allocation
was activated here. BO-07 and protected changes 21–100 remain deferred.

Even a positive result would describe this recorded workflow. The comparison
against ordinary Gradle and the benefit of adaptation remain later questions
in the governing plan. This block completes a measurement prerequisite; it
does not establish product viability.
