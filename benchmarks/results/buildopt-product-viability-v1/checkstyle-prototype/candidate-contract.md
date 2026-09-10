# Content-aware Checkstyle prototype contract

This is a local, source-bound correction for the admitted Elasticsearch owner.
It implements BV-003 and is subject to the complete BV-004 correctness gate.
It is not a qualified replay runner, a measured speedup or an adaptive product.

## Exact boundary

The proof revision is `22d6425e9a44ec9e1aedc2785f4478b8019c851a`, engineering
ordinal 20. The [manifest](./candidate-manifest.json) binds five changed files,
their preimages/postimages, source prerequisites and both patch digests.
The prior ForbiddenPatterns cacheability delta is a separate native baseline
present in both worktrees. This candidate neither includes nor takes credit
for that delta.

The owner keeps the existing `Checkstyle` task type, lazy source collections,
exclusions, empty compiled-source classpath, checking classpath, dependency
graph, report locations, XML settings, failure behavior and heap settings.
Only `:server` opts in. Its three tasks receive a generated Gradle-only XML
configuration; the original Checkstyle XML and IDE resources remain unchanged.
The extra declared output is `server/build/checkstyle-content-aware.xml`.

The observed implementation uses Gradle 9.7.1, Checkstyle 13.11.0 and Temurin
21.0.12+8 on Linux x86_64. The runtime admission guard requires Gradle 9.7.1,
Java language version 21, the exact original owner XML, XML-only reporting,
empty compiled-source classpath, known engine/provider jar hashes and exact
active custom-rule class hashes. It additionally requires the adapter and its
state helper in the checking provider to match the loaded owner implementation.
The private state property and configuration directory must identify the exact
installed task state; replaced properties or an alternate directory use native
checking instead of trying to load an unresolved adapter configuration.
The Java executable and release-file hashes bind each accepted history epoch.
Only the listed environment was tested for incremental admission; Java 25 and
Checkstyle 10.24.0 were tested as native fallbacks.

## Native execution and reports

`ContentAwareChecker` implements Checkstyle's native `RootModule` interface
and delegates to an ordinary `Checker`. It does not fork the engine. Native
configuration, modules, messages, filters, parser, listeners and formatters
remain responsible for checking and diagnostics.

On an admitted invocation, the adapter hashes actual file bytes and absolute
paths. It sends changed or unproven files to `Checker`. Previously successful
files receive the same native empty file-start/file-finish report events in
the current native enumeration order. Every required XML file entry remains;
an empty diagnostic entry is not dropped to obtain a faster report.
Native exclusions remain exclusions. Excluded files never become reusable
successful checks.

No rule violation, warning, exception or incomplete request becomes accepted
history. Failures preserve the native complete or partial report and exception
path. A subsequent partial fix cannot reuse successes from the failed request.
Native `UP-TO-DATE`, `NO-SOURCE` and build-cache decisions occur before these
actions and remain native decisions. In particular, a no-source invocation can
retain the previous report, and a source-root move with identical relative
inputs can remain up to date; the candidate preserves both observed behaviors.

## History and invalidation

History is local to the worktree and task, under
`.gradle/buildopt-checkstyle/server/<task-name>`, and is registered as Gradle
local state. It contains content hashes of successful files and provenance,
not a second report/output cache. Gradle cache restoration clears this state;
the following edit must establish a fresh successful history.

The global identity includes the normalized state/report roots, Gradle and
Java identities, actual XML and all configuration properties, every file in
the configuration directory, ordered checking classpath paths and bytes, and
the history implementation class resources. Suppression changes, changed rule
bytecode, a changed source path, missing/tampered reports, a different root or
missing/corrupt state cannot silently reuse an old success.

The protocol has three records: `request`, `proposed` and `success`.

1. Before the worker starts, remove the previous committed success and stale
   intermediate records. A request can import previous successes only when
   its complete global identity and existing report digest match.
2. The checker verifies all input hashes again after successful native checking
   and report closure. Only then may it write a proposal for that request ID.
3. After Gradle has waited for the native worker, the final task action checks
   the request/proposal identity, recomputes global inputs and seals the actual
   report digest. It publishes the success record atomically.

Records use a bounded, versioned properties payload with a SHA-256 checksum
and atomic temporary-file replacement. Interrupted temporary files, corrupt
records and missing state are not accepted. Optional history I/O failure
returns to native checking. The native result does not become a success merely
because a metadata operation failed.

Unsupported rule/configuration, checking-provider or runtime combinations
bypass the adapter itself and use the original native `Checker`, retaining
caller-supplied native rules. An unsupported manual edit that corrupts the
generated adapter-root declaration is explicitly refused. This is a bounded
compatibility policy, not a claim of general automatic rule analysis.

## Application and removal

Use [apply-candidate.py](./apply-candidate.py) with an explicit Elasticsearch
worktree root. `--check` checks applicability; `--inverse` removes the patch.
The tool verifies patch digests, exact owned source bytes/modes and install
prerequisites, checks the complete Git patch, applies without the index and
verifies resulting bytes. A complete repeated application/removal is a no-op.
Mixed, drifted or unsupported owned source is refused before any patch write.
Unrelated files, the baseline correction, Git history and remotes are outside
its write set. Removing the correction does not require reverting unrelated
rule/configuration changes.

Removal restores the original plugin and removes the four introduced source
files. Native Gradle rebuilds its affected build-logic artifacts. The inverse
proof must execute ordinary native checking after removal, verify that the
removed implementation classes are absent from those jars, and verify safe
reapplication. Retained optional history is harmless to the original plugin;
it is not read by native Checkstyle. No global cache deletion or destructive
repository cleanup forms part of removal.

## Proof and economic limits

The diagnostic observer counts real `Checker.processFile` entries. Its Gradle
subclass only supplies worker observation options and is confined to small
correctness fixtures. Passing and failing XML were compared with the ordinary
unobserved native task. The actual full owner uses the unchanged native task
class. Cancellation tests pause an actual native worker and stop only its owned
systemd unit. Repeated-daemon and restarted-process behavior are separate tests.

All source transitions, commands, exits, observed processing, report digests,
intermediate state, failed probes and corrections remain bound to their actual
executable versions. A later compatibility guard does not silently replace an
earlier receipt. Instruction disassembly verifies that the per-file algorithm
and state codec are unchanged by formatting. Changed provider and state-binding
admission each have a failing reproduction, a successful first correction and
positive/negative revalidation. The final owner tests, full workflow, complete
output comparison, inverse/reapplication and five-file formatter check bind the
final source manifest. Earlier evidence is retained with its actual version;
[state-binding proof validity](./state-binding-proof-validity.json) records the
unchanged execution paths and the affected checks that were repeated.

Full-workflow comparison covers every declared root-build output and its task
owner. Exact equality is the default. Only independently qualified lexical
Checkstyle root mapping, RAT root/time mapping, the previously enumerated
manifest dates with provenance, and complete native compiler-state comparison
are permitted. The generated adapter configuration has an exact content rule.
No class output, finding, source entry or unknown metadata field is ignored.

Validation ordinals 21–100 remain untouched. BV-005 must qualify source
advancement, isolated arm state, accounting, output capture, fault detection
and restart before value measurement. BV-006 must then exercise the engineering
prefix and freeze the intervention. The original confirmation, replication,
payback, latency, 5% net-workflow and one-second-per-scheduled-transition gates
remain unchanged. The final product still requires the separate adaptive and
customer gates; this fixed correction supplies no proof for those claims.
