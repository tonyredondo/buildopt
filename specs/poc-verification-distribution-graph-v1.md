# Verification and distribution graph experiment

## Question

Can BuildOpt replace the old name-based fallback for Gradle verification and
archive tasks with generic, public Gradle task contracts, and does one newly
complete real Spring verification scope beat an optimized native Gradle build?

This is a POC experiment. It grants no production or automatic-selection
authority.

## Generic graph contract

The discovery adapter accepts a requested task only when it is a conventional
build lifecycle task or implements one of Gradle's public build-owned task
contracts:

- `AbstractCompile` for compile producers;
- `VerificationTask` for build-owned verification;
- `AbstractArchiveTask` for archives and distributions.

The configured `TaskExecutionGraph` supplies each requested entrypoint's
dependency closure. A `Test` task, an arbitrary root task, a task absent from
the configured graph, or scheduled work that cannot be attributed through the
dependency graph still makes discovery incomplete. BuildOpt then retains the
full graph.

At Spring Framework revision
`4b5c92703c63966c3471242c621c4a63b382638d`, this resolves both previously
unknown scopes without repository-specific task names:

| Family | Native selector | Candidate | Native projects | Candidate projects | Measured |
|---|---|---|---:|---:|---|
| Verification | `checkstyleMain` | `:spring-webmvc:checkstyleMain` | 23 | 12 | Yes |
| Source distribution | `sourcesJar` | `:spring-webmvc:sourcesJar` | 22 | 12 | No |

Both graphs must be complete and contain no `Test` tasks. Source distribution
is retained as capability evidence only: the prior leaf-packaging experiment
did not qualify, and this block authorizes timing exactly one new scope.

## Measurement

The measured change appends a fixed comment to Spring MVC's
`DispatcherServlet.java`. Four paired observations alternate native-first and
candidate-first order after one unmeasured warm-up per arm. Both arms use Java
25, Gradle 9.6.1, daemon mode, offline dependencies, the local Gradle build
cache, parallel execution and 12 workers. BuildOpt launcher time is included.

The control runs the repository-owned `checkstyleMain` selector. The candidate
runs installed BuildOpt and the exact `:spring-webmvc:checkstyleMain` task. The
`spring-webmvc` Checkstyle XML report must be non-empty and byte-identical for
every pair. Root-build `Test`, publishing, scans and external actions are
forbidden.

The scope qualifies only when all of the following preregistered conditions
hold:

- mean saving is at least 500 ms and 2%;
- all four pairs are positive and the paired lower bound is positive;
- every required output is identical;
- there are no product-attributable failures;
- a `gradle.properties` change still selects the full graph and executes an
  outside-scope Checkstyle task.

If the gate fails, verification remains full-graph. Thresholds and failed pairs
cannot be changed or discarded after measurement.

The machine-readable contract is
[`poc-verification-distribution-graph-v1.json`](./poc-verification-distribution-graph-v1.json).

## Result

Both generic graph contracts passed. The accepted verification comparison did
not pass the value gate:

| Arm | Mean |
|---|---:|
| Optimized native `checkstyleMain` | 33,916 ms |
| Installed BuildOpt `:spring-webmvc:checkstyleMain` | 33,812.25 ms |

The mean saving is 103.75 ms/0.31%. Pair savings are -5,158, +2,921, -1,136
and +3,788 ms, so only 2/4 pairs are positive and the conservative lower bound
is -5,158 ms. Every pair produced the same non-empty Checkstyle report, no
root-build `Test` ran, product failures are zero, and the global fallback
passed. The terminal decision is `RETAIN_VERIFICATION_FULL_GRAPH`.

The result separates capability from value: BuildOpt can now model these task
families generically, but graph completeness alone is not sufficient reason to
select the narrower verification path. Source distribution remains capability
evidence only and was not timed.
