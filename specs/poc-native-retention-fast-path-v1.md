# Native-retention fast path V1

This bounded POC asks whether BuildOpt can retain optimized native Gradle close
to native cost when a previously qualified cross-commit profile is not safe or
economic for the current change. It does not broaden profile selection and it
does not turn a native-retained build into a performance claim.

The machine contract is
[`poc-native-retention-fast-path-v1.json`](./poc-native-retention-fast-path-v1.json).
It freezes the same three public repositories, target qualifications, nine
descendant commits, customer tasks, alternating arm order and exact output
gates used by cross-commit breadth V2. The one selected Ktor descendant remains
part of the source run but is outside this eight-fallback analysis.

## Attribution before implementation

The V2 wall-time deltas do not identify product overhead by themselves. Each
native-retained revision has only one control/candidate pair and ordinary
Gradle duration varied by tens of seconds. The existing structured data shows:

- three economic rejections already skipped output observation and spent only
  about 2--3.5 seconds in selection plus central synchronization, despite wall
  deltas ranging from a 12-second apparent gain to a 105-second loss;
- two global changes attached a full output/impact observation unnecessarily;
- one nested `build.gradle.kts` change was rejected by central preflight but
  local discovery did not recognize nested build logic as global, so it also
  paid for observation before reporting an aggregate-output incompatibility;
- one first ordinary observation is required to learn a new profile; and
- one ownership-ambiguous change requires the configured Gradle model and
  workflow-input observation before ambiguity can be proven safely.

The implementation therefore reports child Gradle duration and BuildOpt's
pre/post execution time directly. A single paired wall-time delta remains
descriptive and is never used as the fast-path acceptance gate.

## Required behavior

Repository-independent build-logic/global patterns and economic rejection from
an already verified central graph must decide before Gradle. Those six
observations must attach no output/impact observer, preserve the original
customer command and finish with no more than 5 seconds of BuildOpt wrapper
work each and 24 seconds in total. The two discovery-required observations must
declare that they instrumented Gradle; the POC must not suppress them merely to
make fallback timing look better.

Every observation must still execute one authoritative optimized-native build,
preserve the exact required output inventory and bytes, keep central and local
state fail-closed and report zero product-attributable failures. No threshold,
task, source revision or correctness boundary may change after timing begins.

## Boundaries

This is proof-of-concept evidence, not a production latency SLO. It adds no
repository-name branch, no production authority, no soak or design-partner
dependency, and no Test Optimization behavior.
