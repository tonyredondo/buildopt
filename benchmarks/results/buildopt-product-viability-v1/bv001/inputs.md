# BV-001: Executable inputs and native build proof

Date: 2026-09-08. Program: `BUILDOPT-VIABILITY-V1`.
The build/history prerequisites below passed. Final step validation and the
transition to BV-002 are recorded in the
[execution tracker](../../../../docs/plans/buildopt-product-viability-v1-tracker.md).
This report establishes a usable experimental foundation; it reports no saving.

## Source and execution identity

| Input | Verified value |
|---|---|
| BuildOpt checkout | `/home/tonyredondo/repos/github/tonyredondo/buildopt` |
| BuildOpt branch / HEAD | `main` / `b76ded08c952ebb386576fafce4ae2d8fdcc09f1` |
| Actual working state | 194 pre-existing modified/untracked files; the committed SHA alone does not identify the source tested |
| Source record | [Entry manifest](./source-manifest.json); all non-document inputs remained byte-identical through the checks |
| Environment | Linux x86_64; local execution, no publication or customer contact |
| Program state locator | `.tools/state/buildopt-product-viability-v1/task-state.json` under the BuildOpt checkout |
| Subject Git directory | `.tools/state/eic-native-v1/repos/elasticsearch.git` under the BuildOpt checkout |
| New subject worktree | `.tools/state/buildopt-product-viability-v1/worktrees/elasticsearch-native` under the BuildOpt checkout |
| Subject branch | `bv1-elasticsearch-native-prefix`; a new branch sharing the existing Elasticsearch Git directory |
| Native workflow | `:server:precommit`, with its complete dependency graph |
| Source after the successful native build | Clean; no tracked source edit, removal or staged change |

The entry source manifest predates five progress-only Markdown edits. The
preservation check explicitly identifies those edits; no code, executable
specification or historical result was changed. The old Elasticsearch worktrees
were not used as new execution targets. Fetching history extended the shared
object store while preserving their pinned source revisions.

## Frozen seed history

[The history manifest](./seed-history.json) contains 101 unique commits and
verifies every adjacent first-parent edge: one anchor and 100 transitions.
The shared repository remains shallow outside this complete required window.
Its original single-commit history was insufficient and was extended using the
preselected endpoint, without selecting by candidate timings.

| Boundary | Revision | Commit time |
|---|---|---|
| Anchor, ordinal 0 | `53a80bec683ad0b065ed9ebe6a57f984e6a91ed1` | 2026-08-31 14:12:17 UTC |
| Engineering prefix ends, ordinal 20 | `22d6425e9a44ec9e1aedc2785f4478b8019c851a` | 2026-08-31 23:16:21 UTC |
| Locked validation ends, ordinal 100 | `16bd5bc5355ac7c6ad736f8a6f93281b24a05ab7` | 2026-09-02 12:09:50 UTC |

The actual span is **45.959 hours**. Follow ancestry order even when displayed
timestamps are non-monotonic. Ordinals 1-20 are engineering inputs; 21-100
remain unconsumed validation outcomes. No replay or candidate timing has run.

The wrapper, daemon criteria, minimum compiler/runtime versions, task source
and registration source each have one blob identity throughout this window.
Three engineering transitions touch `server/`: ordinals 7, 19 and 20. These
are source facts, not measurements of task execution or opportunity frequency.
This window cannot establish maintenance across months or toolchain changes.

The [workflow/source inventory](./source-workflow-inventory.json) binds the
native task and its registration, owner build instructions and the historical
Buildkite precommit command. CI invokes root `precommit`; the selected study
uses the declared server workflow. Its results cannot be presented as a
percentage for the entire root CI pipeline.

## Toolchains and dependency preparation

[Toolchain bindings](./toolchains.json) record the repository lock, installation
manifests and actual executable hashes. The repository-supported `dev/run`
version probes resolved Go 1.26.5, Temurin 21.0.12+8 and Temurin 25.0.3+9.
The native build uses Temurin 21.0.12+8 and the source-pinned Gradle 9.7.1
wrapper; BuildOpt's patcher build uses its pinned Gradle 9.6.1 wrapper.

Elasticsearch's AGENTS snapshot mentions JDK 25, but explicitly defers to its
authoritative build documents. At this revision, CONTRIBUTING, daemon criteria
and minimum compiler/runtime files require JDK 21. The native launcher uses
that verified requirement. Multi-release compilation additionally needs other
JDK installations; the initial offline failure exposed this dependency.

Only dependency modules and Gradle distributions were copied from the retained
acquisition area into a new private Gradle home. After the missing-compiler
failure, the retained Gradle-managed JDK archives/installations were copied
and pinned separately. Both `java -version` and `javac -version` succeeded for
22.0.2, 25.0.4.1, 26.0.2 and 27. See the
[compiler preparation receipt](./receipts/prepare-native-compiler-toolchains.json)
and [actual version probes](./receipts/subject-compiler-version-probes.json).

No project outputs, task history, task-cache entries, daemon registry, compiled
Gradle scripts or BuildOpt state were borrowed from the old experiment.
The successful retry retained its own partial compilation from the failed
prerequisite attempt. It is preparation, not a fresh-state timing comparison.
Dependency-copy receipts report zero network bytes for those copies; Git-fetch
wire bytes were not measured and must not be reported as zero.

Native commands use private Gradle, Java user, Maven, XDG and temporary paths.
The global TMPDIR is unchanged. Java options are applied after `dev/run` has
validated the launcher/compiler version strings. CI mode is false and scans
are disabled. These runs complete offline. A temporary systemd unit bounds each
Gradle process tree; the final native unit is inactive and no longer loaded.

## Observed checks and retained failures

Every command below has its actual argument vector, working directory, exit,
elapsed time and log digest in the linked receipt. The compact
[evidence manifest](./evidence-manifest.json) binds portable files and the
host-local operation traces. Native trace time is diagnostic, not a value pair.

| Check | Observed result |
|---|---|
| [Build all Go command packages](./receipts/buildopt-go-build.json) | `go build -mod=readonly ./cmd/...` exited 0 |
| [Core prerequisite tests](./receipts/buildopt-core-tests.json) | Fresh tests for durable catalog, normalization-aware patching and output equivalence all passed |
| [Longitudinal verifier tests](./receipts/longitudinal-verifier-tests.json) | Fresh local tests passed; no historical campaign rerun |
| [Initial JVM check](./receipts/buildopt-patcher-build-tests.json) | Build succeeded using up-to-date tasks; it did not execute the spike again |
| [Fresh JVM proof](./receipts/buildopt-patcher-fresh-proof.json) | Forced actual compilation and spike execution; all 15 real-Git parser/applier cases passed, with no remote mutation |
| [First native reservation](./receipts/seed-anchor-precommit.json) | Rejected before Gradle: inherited JAVA_TOOL_OPTIONS changed the version-probe output. Command ordering corrected; one reservation remains charged |
| [First actual native build](./receipts/seed-anchor-precommit-v2.json) | Failed at `:libs:cli-terminal:compileMain22Java` because the private offline home lacked JDK 22; failure and partial state retained |
| [Compiler-ready native build](./receipts/seed-anchor-precommit-v3.json) | Exit 0; `:server:precommit` completed 1,045 actionable tasks: 988 executed, three from cache, 54 up-to-date; source remained clean |

The marker contains `done`, and the graph includes the native forbidden-pattern
check. A marker alone is not the future candidate's output contract. The
1045-task build and pinned graph establish the usable workflow; BV-003/BV-004
must separately define and prove complete candidate semantics and owner tests.

## Allocation and next work

The [allocation checkpoint](./allocation.json) retains the initial six-hour /
30-start ceiling shared by BV-001 and BV-002. At this checkpoint, five attempts
are charged and four Gradle invocations actually ran: two BuildOpt tooling
invocations and two native Elasticsearch invocations. No nested TestKit build
ran. The pre-Gradle rejection is retained in the conservative charged count.
The state footprint was 5,692,815,624 bytes, below the 120-GiB ceiling, with
more than the required 40 GiB free. Initial static orientation was unmetered;
allocation timing began before downloads, builds and worktree creation.

BV-002 must now qualify the existing cacheability correction as part of the
native baseline, choose supported daemon/configuration-cache policy, and
observe consecutive engineering-prefix changes. A native cause and plausible
whole-workflow value must pass G1 before any incremental candidate is written.
The historical C007/C008 task/action spans and their actual operation ancestry
were [reconstructed](./prior-causal-diagnostics.json); they remain selection
evidence and do not pass G1 by themselves.

[Six additional repository endpoints](./replication-endpoints.json) are fixed
in the planned order: Groovy, Kafka, Spring Framework, OpenTelemetry Java,
Micronaut Core and Hibernate ORM. Their 101-commit histories and causal
admission are still pending. No replication subject has been selected on value.
