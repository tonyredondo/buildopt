# Historical replay instrument

This is the local Linux AMD64 instrument for BV-005/BV-006. Follow the
[replay protocol](../../docs/plans/buildopt-product-viability-v1-replay.md),
[executable contract](../../specs/poc-product-viability-v1.md) and
[current tracker](../../docs/plans/buildopt-product-viability-v1-tracker.md).
It does not enable a product optimization or run the public confirmation by itself.

Owner capture precision has a separate, prospectively registered
[two-stage design](../../benchmarks/results/buildopt-product-viability-v1/bv006-attribution/precision-design.md).
The original small-fixture/owner test still declares 20 requests. The precision
test declares 80 pilot requests; its result can only size or stop one fresh
confirmation. Version-2 overhead records never imply qualification from the
pilot or point gates alone. The independent precision checker also requires
both registered block intervals inside the unchanged 100-ms arm envelope.
`precision_statistics.py` implements the fixed arithmetic; `--unit` checks
its rejection behavior. These qualification tests do not advance seed history.

## Build and validate

```bash
./dev/run --toolchain go -- go build -mod=readonly -o .tools/bin/history-replay github.com/tonyredondo/buildopt/dev/history-replay
.tools/bin/history-replay validate /absolute/path/to/manifest.json
.tools/bin/history-replay run /absolute/path/to/manifest.json
.tools/bin/history-replay check /absolute/path/to/owned-run
.tools/bin/history-replay resume /absolute/path/to/owned-run
```

Create the output parent directory before building if it is absent. `validate`
checks pins and exact first-parent history without modifying a worktree or
launching the workflow. `run` requires a new owned root. It creates private
worktrees on new branches, then executes the entire declared horizon unless a
safety, infrastructure or resource limit stops it. A negative interim saving
does not stop execution. The Git common directory is shared only for immutable
source objects and owned worktree registration; mutable build state is private.

`history COMMON_GIT ENDPOINT COUNT OUTPUT` writes the consecutive first-parent
revision table. Supply the frozen endpoint, not a moving branch name.
`COUNT` includes the anchor: 100 transitions require 101 revisions.
`Manifest`, `Revision`, `Patch` and `OutputPolicy` in [contract.go](contract.go)
define the strict JSON shape. Fields, empty collections and all schema versions
must be explicit. The machine protocol bytes have a compile-time SHA256 pin.
No credentials or ambient shell environment are imported into native workers.

The manifest binds the launcher/runtime, source package, baseline, candidate
preimages/postimages/prerequisites, generated paths, output rules, graph capture,
resource allocation and prior correctness/overhead/subject evidence. An external
adoption/maintenance/review/research cost uses a bound `ExternalCostRecord` with
monotonic boundaries and evidence. Confirmation requires an explicit external
adoption record for each replication; zero is a recorded assertion, not an
omitted field. Human review remains a separate cost category.

Qualification manifests may optionally bind `controlBaseline` to compare an
installed optimization against its revision. It replaces `baseline` only in
arm N; arm I still applies `baseline` followed by `candidate`. All source guards,
initial-state checks, historical advancement, recovery and offline verification
use the baseline for the actual arm. The override is rejected outside
`QUALIFICATION`; it cannot turn an optimized control into native product evidence.
Omitting it preserves prior manifest serialization and behavior. A present value
must be a complete, valid patch; null and partial definitions are rejected.

## Evidence and recovery

The owned run contains immutable attempts, states, phase boundaries, native
receipts, raw output bytes, daemon logs, snapshots and lifecycle records.
`attempts.jsonl`, `costs.jsonl`, `subjects.json`, `result.json` and `report.md`
are reconstructed exports; immutable export/result versions remain retained.
The checker verifies those exports against the individual raw records, including
source/patch lineage, complete output coverage, required maintenance phases,
monotonic durations, retry cost and nested native invocation IDs.

Native completion reaches the driver before research receipts are synchronized.
This keeps receipt I/O outside the customer request; the full replication
interval still accounts for recording and other research work. Diagnostic host
load, memory, affinity and service CPU/memory are retained. An undelegated
`io.stat` is explicitly `UNAVAILABLE`, not a measured zero. The final offline
checker/export command has its own research receipt outside the replication.

Raw output retention uses at most eight concurrent file copies after native
completion. Every copy preserves independent bytes, modes and its file sync;
the output directory is synced before sealing the capture. Binding and
projection order remain deterministic, and a source that changes during
retention is rejected. The bounded copy sizing diagnostic is explicitly tagged
`replay_copy`; its fixed input manifest and outputs are research evidence.

Each arm's user systemd service owns its supervisor, native CLI, reused daemons
and detached descendants. The worker authenticates the exact driver PID/UID.
Driver death terminates those services. `REQUEST` policy can restore both
quiescent arms from a declared snapshot and preserve failed attempts/costs.
`REPLICATION` policy retains warm daemons but cannot recover their lost memory;
such a run remains incomplete. A sealed pair without a completed checkpoint
also closes incomplete, preventing duplicate credit. No repair overwrites
original failed attempt receipts. Running `resume` after a safety stop is refused.

Output roots come from the native graph and explicit rules. A glob is a
normalization selector within an existing producer's outputs. Projectors bind
their executable/arguments and independent equivalent, semantic-change and
malformed input cases. Raw files and metadata remain retained. A mismatch never
creates a new normalization.

Version 2 adds an explicit `outputs.owner` binding. An absent owner is the
explicit empty binding; generic rules retain their existing behavior. The
Elasticsearch owner binds the frozen C5 allowlist, lexical reader, complete
compiler-state reader and ordered classpath. It compares the complete root-build
task graph, Checkstyle source/report contract and captured outputs, including
the single approved additional generated config. It rejects additional generic
rules or projectors alongside that policy. The included builds remain in the
raw graph and native command accounting; they are not additional root-workflow
output producers.

Reuse of a normalized output requires an earlier successful executed producer
in the same arm, replication, manifest and physical root, with identical raw
bytes and producer identity. Only native `UP-TO-DATE`/`FROM-CACHE` outcomes can
use that origin. The current output remains retained independently. A changed
reader or classpath order invalidates the bound qualification. Engineering and
confirmation require explicit comparator cases and native reuse proof;
qualification runs may create that evidence. Pair comparison is recorded as
research cost and the offline checker repeats it. Owner integration and actual
workflow overhead are separate BV-006 prerequisites, not consequences of the
fixture pass. BV-005's immutable version-1 evidence and executable are retained.

The origin lookup indexes executed producers before opening an older capture.
It verifies the selected donor's bound graph/state and actual donated bytes;
it does not repeatedly read unrelated historical output files for every pair.
Every current output is still checked, and the independent checker verifies
each complete retained attempt.

Native completion is stamped immediately after waiting for the child. The live
disk sampler reads bounded directory batches and cancels at that boundary;
joining it cannot extend the recorded native duration. A complete postflight
disk check remains required outside customer timing. Actual owner qualification
must verify the resulting wrapper boundary and capture imbalance.

## Bounded qualification

```bash
./dev/check-history-replay --unit
./dev/check-history-replay --integration
./dev/check-history-replay --native
./dev/check-history-replay --overhead
```

`--unit` needs the pinned Go toolchain and Git. Integration additionally requires
a working user systemd manager and cgroup v2. Native/overhead checks require
explicit `BUILDOPT_REPLAY_GRADLE_HOME` and `BUILDOPT_REPLAY_JAVA_HOME`, pointing
to the installed Gradle 9.7.1 and JDK 21. Tests do not download toolchains.
Missing prerequisites fail rather than skip. Set `BUILDOPT_REPLAY_TEST_ROOT` to
a task-owned evidence directory to retain fixture roots; otherwise Go owns their
temporary directories. These variables do not change global `TMPDIR` or homes.

Every Go check has a 160-second test deadline and 175-second outer bound.
Integration cases run separately within those bounds, preserving every case
and its durable evidence even on a slower filesystem.
Integration starts at most 104 fixture workflows plus bounded projector helpers.
Native qualification starts 18 Gradle invocations, four nested; overhead starts
20, with four declared warmups and opposite balanced orders. Each fixture has
its own disk/free-space/start limits. Declare the enclosing phase allocation
before running these checks. The overhead pass applies only to the small fixture;
BV-006 must freeze the same lean/deep policy on the actual owner workflow.

Actual owner overhead uses `TestGradleOwnerSymmetricCaptureOverhead` with the
additional `replay_owner` build tag. Compile a dedicated test binary, bind that
binary and the complete owner manifest, then set the absolute
`BUILDOPT_REPLAY_OWNER_OVERHEAD_MANIFEST` path when invoking it. The manifest
requires a new owned root, the exact C5 candidate, one anchor, twenty starts
and a 900-second per-request limit. Declare a suitable whole-sequence deadline
before launch. The test freezes four warmups, four measured pairs per arm,
opposite style orders, the 10-ms wrapper p95 and 100-ms arm-imbalance gates.
It retains native receipts, source verification, samples and the result even
when the overhead gate fails. Run expensive qualification suites serially;
concurrent evidence synchronization can invalidate timing or test deadlines.

The existing EIC, longitudinal-campaign and other command contracts are retained.
This instrument does not reopen any retired research route.

## Explicit bounded observation

Manifest `buildopt.history-replay/manifest/v3` adds the required `observer`
binding (`path` and `sha256`) to a strict policy file. It extends the same frozen
workflow protocol; task, output, cost and result records retain version 2.
Version-2 manifests omit `observer` and preserve their existing behavior. A v2
manifest cannot enable observation, and v3 cannot silently omit it.

```json
{
  "schema": "buildopt.history-replay/observer-policy/v1",
  "samplerAffinity": "8",
  "heartbeatAffinity": "10",
  "sampleNS": 100000000,
  "maxGapNS": 500000000,
  "pendingRecords": 128,
  "pendingBytes": 16777216,
  "frameBytes": 524288,
  "totalBytes": 536870912,
  "drainNS": 5000000000,
  "maxRows": 10000
}
```

The two example observer CPUs must exist and be disjoint from each other and
the manifest's native CPU mask. Each observer mask contains exactly one CPU.
The worker supervisor shares the sampler CPU; only its native child inherits
the native mask. Launch uses the previously qualified locked-thread affinity
scope and restores the supervisor mask before continuing observation.
Queue records/bytes include the in-flight frame. Limits may be lowered; the
100-ms cadence and endpoint-inclusive 500-ms criterion cannot be weakened.
Payload bounds do not claim a bound on total runtime RSS.

`run` starts a separate sampler and its bounded writer/heartbeat before entering
the customer request. Native completion stops capture; drain and helper closure
are outside the customer boundary. An observer failure asks the existing worker
watchdog to cancel the native child, retaining native launch/completion evidence
before stopping its service. Driver death closes the observer lifetime pipe.
Missing final evidence remains incomplete and never receives a successful pair.

Each attempt retains `observation.json`, its launch policy, raw samples, exact
write acknowledgements, phase clocks, queue counts and helper identities/CPU.
`observer-start` and `observer-close` are explicit research phases. The existing
whole-replication research interval already covers overlap; CPU, drain and bytes
in `observations.jsonl` are diagnostic costs, not extra wall time to add again.
`check` verifies payload order/hashes/completeness, helper closure, costs and CPU
masks. Gradle coverage requires daemon snapshots, including both native endpoints;
fixture CLI coverage cannot substitute for a daemon. `resume` verifies completed
observation evidence before scheduling further work.

```bash
./dev/check-history-replay --observer
```

This bounded suite uses 21 non-Gradle fixture workflows, including deliberate
failures, a real writer process pause, checkpoint/resume and driver death. It
also exercises three affinity helper processes, actual CLI commands and rejection
of corrupted evidence. Set
`BUILDOPT_REPLAY_TEST_ROOT` to retain its evidence. Original owner S1/G0, the
100-ms imbalance gate and product value require separate actual-owner proof;
passing this suite does not qualify them or allocate a CPU-profile screen.
