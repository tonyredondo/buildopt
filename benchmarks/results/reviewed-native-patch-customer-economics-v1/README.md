# A Hibernate build correction that did not save time

This Patch Autopilot experiment tried to make Hibernate's settings-documentation
task reuse its previous output safely. The correction produced the right files,
but the measured task was 1.26% slower. The experiment stopped at that result.

## How the correction was selected

Five repositories outside the earlier Micronaut, Spring and OpenTelemetry
study were inspected at fixed source versions. Hibernate was the only one
with candidates that could proceed to further checks: a task that builds an
index and `SettingsDocGenerationTask`, which generates settings documentation.
The other four had no safe change under the selection rules.

The documentation command, `:documentation:generateSettingsDoc`, was selected
from that source inspection. The experiment
used fresh measurements; it did not reuse an earlier positive result.

## Correct output, no measured advantage

All six builds used to check correctness succeeded. The corrected task restored
identical output from the cache, including when run in a different directory.
Changes to file contents or relative paths made the task run again, as required.
Undoing the patch restored the original source exactly.

The eight timing comparisons alternated which version ran first and retained
every result:

| Measurement | Result |
| --- | ---: |
| Average with the original task | 15,143.75 ms |
| Average with the correction | 15,334.625 ms |
| Extra time with the correction | 190.875 ms / 1.26% |
| Comparisons in which the correction was faster | 4 of 8 |
| 95% interval for the saving | −829.625 to +380.875 ms |

The interval allows both a loss and a small saving. These measurements do not
support adopting this correction to improve build time, even though the output
checks passed and there were no failures attributed to BuildOpt.

The recorded stopping decision is
`STOP_THIRD_FAMILY_CUSTOMER_ECONOMICS_VALUE_GATE`. The later patch-delivery and
review step did not run. It must not be counted as another failed experiment
or as a completed delivery.
