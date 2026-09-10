# Historical replay instrument qualification

Date: 2026-09-08. Program: `BUILDOPT-VIABILITY-V1`. Step: BV-005. Actor: Codex.

The fixed N/I replay instrument is qualified on local Linux AMD64 fixtures.
It can advance exact Git history, preserve independent native and candidate
state, retain failed attempts and costs, and reject inconsistent positive
claims. This completes BV-005. It establishes no public-repository saving,
confirmation readiness, adaptive-controller behavior or product viability.

The next step is BV-006: integrate the exact C5 Elasticsearch output rules,
qualify capture overhead on that workflow, run paired engineering ordinals
0..20, and freeze the confirmation inputs. All 80 seed validation transitions
remain untouched. The frozen Checkstyle G2 proof and earlier negative decisions
retain their original bytes and scope.

## What was implemented

The [executable contract](../../../../specs/poc-product-viability-v1.md),
[protocol JSON](../../../../specs/poc-product-viability-v1.json),
[runner reference](../../../../dev/history-replay/README.md) and
[qualification entrypoint](../../../../dev/check-history-replay) define the tool.

- Strict required-field/version decoding, exact first-parent reconstruction,
  source/runtime/package predicates, guarded patch application and exact inverse.
- Separate arm worktrees, caches, homes, temporary directories and native
  daemons; alternating NI/IN order and fresh independent replications.
- Monotonic customer/native/research intervals, unique preparation and
  maintenance costs, retained interrupted attempts and nested Gradle command IDs.
- Actual process-tree ownership and driver-death handling. Cold-request recovery
  restores both pre-attempt states; lost warm JVM memory closes incomplete.
- Complete producer/output capture, independently qualified projectors and a
  checker reconstructing raw coverage, state, costs, metrics and exports.
- Fixed full-horizon arithmetic: native failures cannot become savings, native
  no-action rows receive zero mechanism credit, early negative rows stay in the
  experiment, and payback must persist through the final scheduled ordinal.

Only fixed N/I persistent-state mode is supported. Native cache is enabled,
Configuration Cache is disabled and lean graph/outcome capture is symmetric.
Deep attribution remains diagnostic. The adaptive N/F/A controller belongs to
BV-011 and cannot be enabled by changing a manifest field.

## Verification

[Qualification decisions](./qualification.json), [individual test events](./test-events.json),
[command receipts](./command-index.json), [exact source inputs](./source-manifest.json)
and [proof reuse boundaries](./proof-reuse.json) bind each claim to what ran.

| Contract | Actual proof |
|---|---|
| Schema/history/patch/metrics | Strict-decoder, ancestry, patch/inverse and metric goldens; final unit suite and vet pass |
| Build and retained commands | Complete `go build -mod=readonly ./...` passes; four selected existing EIC CLI/provenance/budget tests pass |
| Production CLI | Built binary validates, runs four real fixture workflows, checks raw evidence and refuses a completed resume; read-only commands preserve all evidence bytes |
| Chronology/isolation | Two fresh replications reverse initial order; shared/future state and source, patch, runtime and package drift stop before the next child |
| Failures and economics | Candidate mismatch, native failure, dependency unavailability and unrun slots stay distinct; external costs charge once; early negative/later positive horizon completes |
| Native execution | Actual Gradle EXECUTED -> UP-TO-DATE -> FROM-CACHE in both arms; persistent native no-action replay preserves zero action credit |
| Nested work | Two native TestKit requests per reused daemon are counted separately by command UUID |
| Crash/recovery | Real driver death and detached descendant termination; partial native receipts and failed pair costs retained; cold recovery, warm refusal and sealed-pair refusal exercised |
| Independent rejection | Forged duration/output/result, missing maintenance, hidden retry, malformed projection and incomplete coverage rejected |

Fifteen integration/native case families pass. The selected final core,
adversarial and recovery checks bind every production Go file to the final
source. The recovery command retains its one failed shutdown test; its other
cases pass. The assertion-only repair has a separate five-repetition success
record. It changes neither production termination nor the six-second deadline.
The [standalone CLI proof](./standalone-cli.json) uses executable SHA-256
`df9f1caf452387676277ab5cba48c52137bf6f0ccf954188424a95c39d2190cf`.

## Instrument overhead and charges

The [frozen overhead plan and samples](./overhead/plan.json) contain 20 actual
Gradle commands: four declared warmups and four measured plain/lean pairs per
arm, with both arm and style order balanced. The
[independent reconstruction](./overhead-reconstruction.json) checks raw native
receipts, hashes, monotonic intervals and arithmetic.

| Measurement | Observed | Prospectively declared limit |
|---|---:|---:|
| N median lean-capture extra | 23.645480 ms | Descriptive |
| I median lean-capture extra | 12.670142 ms | Descriptive |
| Difference between arm medians | 10.975338 ms | 100 ms |
| External wrapper p95 | 1.486684 ms | 10 ms |

These qualify the small no-action fixture. Individual paired deltas include
negative values from noise. Neither the medians nor the small imbalance estimate
can substitute for overhead qualification on Elasticsearch. The native timing,
IPC and resource-capture implementation is unchanged since measurement apart
from explicit platform build constraints; later admission/checker changes were
qualified separately on final source.

[Native command reconstruction](./native-start-reconstruction.json) verifies
50 distinct Gradle starts in this phase, including eight nested TestKit starts
and the 20 overhead starts. There are two standalone javac preparations and four
TestKit helper JVMs. All are research costs. The enclosing
[allocation](./allocation.json) is eight hours, 60 Gradle starts, 600 declared
qualification workflow/tool reservations, 20 GiB new state and 40 GiB free.
Reservation counts include failed planned attempts; they are not an operating
system process census. No public owner workflow or validation row ran in BV-005.

## Failures, corrections and limits

The [correction ledger](./corrections.json) preserves every failed qualification
command and analysis correction. Optional `io.stat` was unavailable on the
user-delegated cgroup; it now records UNAVAILABLE explicitly. A shutdown test
stopped waiting at an empty group before checking the remaining process identity;
both conditions now share the original deadline. Five subsequent actual driver
kills pass, with interrupted candidate cost retained. The failed run did not
capture the transient kernel process state, so no zombie-state diagnosis is claimed.

A standalone diagnostic checked a previously completed but expired manifest;
its resume correctly refused on elapsed allocation. The first fresh-root
correction accidentally retained its absolute deadline and ran zero workflows.
A separately declared fresh fixture then proved completed-horizon refusal.
The expired evidence was never reset. One failed setup correction is charged;
no unresolved correction or hidden rerun remains. The offline nesting audit
also corrected its assumption that TestKit client cwd identifies nested work.

The actual Elasticsearch output adapters, including the one exact generated
config, are still BV-006 integration work. Generic fixture normalization does
not broaden C5's owner contract. This tool is qualified on Linux AMD64 with
user systemd and cgroup v2; other platforms and Configuration Cache are unproved.
External G0/G2 evidence bindings preserve supplied artifact identity; they do
not independently establish the truth or applicability of every prior claim.
Final offline checking/export time is a separately recorded research cost,
outside the complete replication envelope.

## Recovery and retained artifacts

The local raw fixture archive, `raw-fixtures.tar.zst`, retains all 79 task-owned
fixture roots, three standalone CLI runs and the production executable. It
includes actual raw outputs, receipts, daemon logs, snapshots, synthetic Git
objects and test binaries. Some negative fixtures deliberately contain forged
records: their test rejection evidence determines validity, not those modified
summaries. The earliest Go-owned temporary fixtures were cleaned by Go; their
completed command logs remain retained and are not the final proof source.
The archive is not included in Git; see the [publication record](../publication.md)
for its size and checksum.
The [source package](./source-package.tar.zst) preserves the exact new sources,
protocol, check command and existing module/toolchain manifests.

Archives preserve evidence across machines; their manifests and synthetic
worktree metadata still contain the observed absolute host paths. They are not
silently relocatable execution state. Inspect archives in a separate directory
and reproduce with fresh owned roots and the pinned toolchains. Do not restore
old warm daemons, rewrite observed paths in evidence, or reuse consumed state
as a fresh replication. No toolchain, credential or unrelated user directory
is copied into these archives.

The active recovery record remains
`/home/tonyredondo/repos/github/tonyredondo/buildopt/.tools/state/buildopt-product-viability-v1/task-state.json`.
The [execution tracker](../../../../docs/plans/buildopt-product-viability-v1-tracker.md)
contains BV-006 criteria and the later value, replication and adaptive gates.
All local source changes remain unstaged; no commit, push, upload, customer
contact or background watcher was created.
