# Source-Bound Configuration-Input Corrections v1

`SOURCE_BOUND_CONFIGURATION_INPUT_CORRECTIONS_V1` (`SBIC`) is a directed
mechanism experiment. It asks whether BuildOpt can turn repository-owned,
configuration-time process reads into reviewed native Gradle corrections that
preserve exact outputs and measurably improve repeated owner workflows.

Unlike CINC, SBIC deliberately selects from source facts before the first
strict diagnostic. It can prove that the correction mechanism works; it cannot
estimate prevalence in arbitrary repositories. CINC and SDCR motivate the
hypothesis and runner design but supply no SBIC row, duration, candidate, or
value evidence.

The [machine contract](./poc-source-bound-configuration-input-corrections-v1.json),
[three-family manifest](./poc-source-bound-configuration-input-corrections-v1.subjects.json),
and [execution tracker](../docs/plans/source-bound-configuration-input-corrections-poc.md)
are authoritative together.

## Frozen source class

The v1 source detector consumes repository-owned Kotlin, Java, or Groovy build
logic and emits one typed decision for each bound operation:

- `PURE_CONFIGURATION_PROCESS_READ`: a configuration-time process only reads
  bounded external state, with explicit command, output consumption, working
  directory, and error behavior available for review;
- `SIDE_EFFECTING_PROCESS`: the command can mutate source, Git metadata, remote
  state, a service, or another persistent resource;
- `SECRET_OR_INTERACTIVE_PROCESS`: credentials, stdin, or an interactive
  process may be involved;
- `TASK_ACTION_ONLY`: execution is provably deferred to a task action;
- `ALREADY_TRACKED`: Gradle already owns the input through a supported provider
  or `ValueSource`;
- `AMBIGUOUS_BINDING`: phase, ownership, call site, or behavior is incomplete;
- `SOURCE_DRIFTED`: an exact source digest no longer matches; or
- `NO_ACTION`: no supported operation is present.

Names of repositories, tasks, files, variables, and functions are labels only.
An executable or command may be retained as evidence but can never make a row
eligible; explicit command semantics may only reject an otherwise eligible
row. In particular, `git` is not intrinsically safe: BlueMap's
`update-index --refresh` was rejected during prospective selection because it
can update index metadata.

The only correction recipes are `PROVIDER_FACTORY_EXEC_V1` and
`TYPED_VALUE_SOURCE_V1`. A proposal must preserve command arguments, working
directory, environment delta, charset, stdout/stderr consumption, exit and
fallback behavior, and every value consumer. SBIC never changes a wrapper,
plugin, dependency, required output, or owner workflow to make a proposal
compile.

## Frozen cohort and gates

The exact three families are Suwayomi Server, QuickCarpet, and LSSS. Each has a
checked wrapper, an owner-documented workflow, a supported locked JDK, exact
source and call-site digests, and a pure process-read binding. Source selection
precedes every SBIC Gradle start.

The blocks are ordered and fail closed:

1. `SBIC-000` freezes this contract, subjects, resource limits, and zero-state.
2. `SBIC-001` independently reconstructs detector decisions and source spans.
3. `SBIC-002` uses the proven SDCR runner for one cold strict diagnostic per
   family. All three starts must be conclusive and at least two families must
   bind the expected strict problem to frozen source.
4. `SBIC-003` measures at most two controlled materiality starts per eligible
   family. At least two families must save both 500 ms and 2% before proposals.
5. `SBIC-004` compiles at most one digest-bound proposal per family in an
   isolated copy. At least two must preserve exact outputs, equivalent errors
   and input invalidation, store then reuse Configuration Cache, revert exactly,
   and add zero product failures.
6. `SBIC-005` runs one excluded stabilization plus eight balanced native versus
   candidate pairs per qualified family. Every pair must be positive, the mean
   saving must be at least 500 ms and 2%, the paired 95% interval must be
   positive, candidate p95 must be non-regressive, and combined payback must be
   at most 300 builds.
7. `SBIC-006` reconstructs the terminal decision and produces review-only
   PatchBundles. No automatic apply, merge, or upstream pull request is allowed.

Cold incompatibility is an admissibility outcome, not a silently lost start.
Every started child publishes atomic evidence. Standard hosted CI validates
contracts, fixtures, source reconstruction against retained snapshots, and
candidate correctness only; local controlled runs own wall-time conclusions.

## Resource and product boundary

Starts are sequential, use at most four workers, run for at most 1,200 seconds,
and stop before another start below 8 GiB free. The campaign may add at most
20 GiB and requires 10 GiB free before a public start. Before controlled timing,
the host must pass the existing seven-sample 1.15 stability ratio after 120
seconds of quiescence.

The product boundary remains:

`Gradle through wrapper -> observation -> reviewed proposal -> validated native correction`

The accepted repository continues running native Gradle through the wrapper.
SBIC cannot authorize production, autonomous mutation, automatic merge, Test
Optimization, or a prevalence claim.
