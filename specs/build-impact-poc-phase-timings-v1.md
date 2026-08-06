# Build Impact POC phase timings

`buildopt impact --timings-file PATH` writes a private, canonical JSON report
after the Gradle child and launcher teardown complete. The option exists only
for owner-operated POC attribution and does not change the selected
entrypoints, production authority, fallback rules or required outputs.

The top-level phases are non-overlapping:

- `impactPreparationNs`: CLI parsing, repository/change resolution and the
  complete planner;
- `gradleSetupNs`: Wrapper and Gradle invocation preparation;
- `runtimeSetupNs`: launcher work between Gradle setup and child execution;
- `gradleExecutionNs`: starting and waiting for the Gradle child;
- `teardownNs`: launcher work after the child returns;
- `unattributedNs`: bounded transition time needed to reconcile the monotonic
  total exactly.

The nested `planner` timing subdivides manifest loading/validation, declared
graph loading/validation, generated-state loading/validation and impact
evaluation. Its phases are contained within `impactPreparationNs` and must not
be added again to the top-level total.

The report is diagnostic evidence, not a performance claim. Enabling it keeps
BuildOpt as the parent process so post-child teardown can be observed; normal
uninstrumented native-only execution may still replace the process on Unix.
The destination must be repository-relative, may not escape through a symlink,
and is written atomically with private permissions.

Validate a report with:

```bash
./dev/check-build-impact-phase-timings /absolute/path/to/timings.json
```
