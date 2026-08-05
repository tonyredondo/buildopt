# Public-repository workflow diagnostics

This contract preregisters `POC-REALWORLD-DIAGNOSTICS-001` before profiling
the pinned Spotless, Mockito, and SpotBugs revisions. Its purpose is to explain
where their real developer or CI workflows spend time and to turn those facts
into generic BuildOpt hypotheses. It is not a candidate-versus-control
benchmark and cannot claim savings.

The workflow commands come from files committed by each upstream repository.
Build scans, publishing, credentials, and external CI services are removed,
but the Gradle tasks remain unchanged. Spotless exercises its two-command
sanity workflow, Mockito exercises formatting, build, and coverage as three
separate invocations, and SpotBugs exercises its documented detector-regression
task without selecting individual tests.

Spotless also resolves changed files against `origin/main`. Its disposable
checkout therefore fetches that named reference while keeping `HEAD` and every
profiled source byte fixed to the preregistered revision. The first shallow
preflight discovered this requirement and stopped before any profile existed;
the complete matrix restarts from zero after this explicit input was added.

## Measurement boundary

An unmeasured online preflight resolves public dependencies for the exact
workflow. Generated `build/` directories and the checkout-local `.gradle`
directory are then removed. The measured workflow runs once, offline, in the
digest-pinned 4-CPU/16-GiB container with the repository's Wrapper and native
Gradle profiler. Build cache is disabled and tasks are rerun so the report
reveals actual task work rather than a cache-hit benchmark.

Every Gradle invocation produces its own profile. The evidence records wall
time, configuration phases, task-execution phase, task outcomes, and the most
expensive tasks. Task durations may overlap under parallel execution and are
therefore never added or presented as potential wall-clock savings.

## Decision boundary

The diagnostic may preregister a follow-up experiment only when the hypothesis:

- belongs to a named BuildOpt feature and is generic rather than a repository
  name check;
- compares against the best compatible native Gradle configuration;
- preserves required outputs and every test requested by the workflow;
- declares its failure signal and safe fallback;
- does not move a threshold after observing a result.

Test execution remains owned by Test Optimization. BuildOpt may reduce
compilation, generation, packaging, configuration, transfer, or other
build-owned work, but this block cannot omit or select tests. A repository with
no defensible generic opportunity is recorded as such instead of receiving a
special-case optimization.

Validate the frozen contract with:

```bash
./dev/check-poc-real-world-diagnostics --spec-only
```
