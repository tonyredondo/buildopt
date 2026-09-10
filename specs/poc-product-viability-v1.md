# Fixed historical replay, executable contract v2

This implements BV-005/BV-006 of the [viability tracker](../docs/plans/buildopt-product-viability-v1-tracker.md).
The [replay protocol](../docs/plans/buildopt-product-viability-v1-replay.md) remains
authoritative for the experiment and its gates. Instrument qualification is
separate from a public-repository value result.

The executable accepts `FIXED_NI`, persistent workspace mode `P`, and explicit
`QUALIFICATION`, `ENGINEERING`, or `CONFIRMATION` phases. It rejects unsupported
adaptive/ephemeral experiments. Confirmation requires the complete 101-revision
history, two fresh replications and the unchanged 21..100 validation horizon.
Fixtures and the engineering prefix cannot pass G3.

## Identity and inputs

Manifests and versioned receipts have explicit schema identifiers. Every typed
JSON object rejects missing, unknown and duplicate keys; costs and timestamps
are checked against their enclosing receipt version. Paths are absolute for bound external inputs and relative for owned
artifacts. A manifest binds the protocol, runner executable and source package,
Git common directory, each first-parent revision/tree/date/path digest, runtime,
workflow, patch preimages/postimages, output comparison policy and allocation.
`validate` is read-only. Execution creates new, exclusively owned roots; it
never attaches to an existing development workspace.

The baseline and candidate are separate exact file recipes. A recipe is atomic
with respect to admission: inspect every precondition before modifying any
file. Reverse only verified postimages. Historical advancement first verifies
all tracked bytes/modes and untracked paths, removes the owned deltas, performs
an ordinary guarded checkout on a fresh task-owned branch for that ordinal,
and applies the compatible baseline. Existing branches are never rewritten.
Candidate admission/application/finalization is customer work. Source inventory,
neutral checkout and independent output comparison are research work. Each
phase has a unique identifier and a monotonic interval; overlap cannot move
customer work outside its envelope or charge it twice.

Each arm owns its worktree, home, Gradle home, temporary files, native cache,
outputs and daemon service. Only declared, hash-bound acquisition layers may
initialize an arm. The allowlist excludes native task caches, project history,
BuildOpt state and previous outputs. State lineage points only to that arm's
previous completed observation in the same replication.

## Execution and recovery

Arms run sequentially, NI/IN alternating from NI at replication 1 ordinal 0;
replication 2 reverses the order. A Linux user systemd service owns each arm's
supervisor and all its descendants, including detached processes and Gradle
daemons. Parent identity includes PID and start time. Losing the driver stops
only these owned services. Time, free space, artifact growth and invocation
reservations are checked before work and while children run.

The complete customer request uses an external monotonic envelope; native child
time is nested within it. Immutable start/finish records and raw output are
retained even after failure. Gradle daemon command IDs, not JVM counts or task
callbacks, account for outer and nested invocations. An unfinished native
command still consumes its reservation and is retained as incomplete evidence.

`REPLICATION` daemon policy preserves the native daemons between requests. A
lost session cannot reconstruct daemon memory: recovery closes that replication
incomplete and retains its attempted candidate cost. `REQUEST` policy can
snapshot both quiescent arms before a pair. A declared infrastructure retry
restores both to that exact snapshot, archives failed state and starts a new
attempt. It never credits the completed half of an interrupted pair. Candidate
errors cannot take this retry path. An unavailable/invalid checkpoint closes
incomplete rather than manufacturing a clean restart.

Checkpoints are durable after verified pairs, including the required twenty
transition boundaries. A negative interim value never changes the horizon.
Every scheduled slot has a typed result, including all unrun slots.

## Outputs, accounting and independent checking

Raw files retain content, modes, symlink targets and absence, plus producer
identity. Output roots come from the complete workflow contract/captured graph.
Comparison transforms must be pinned and independently qualified before the
first candidate. No new mismatch admits a normalization. Raw files remain
available for the checker; output collection and checking are outside primary
timing and appear in the research ledger.

The checker reconstructs history, source/state lineage, order, containment,
reservations, raw artifact hashes, timing intervals, output equivalence,
classifications, phase accounting and metrics before comparing a claimed
result. The claim is tamper evidence against inconsistent retained records,
not authentication against an attacker rewriting every raw observation.

Economics uses integer nanoseconds, conservative full candidate charges for
unpaired attempts, prefix excess/adoption cost, maintenance and every scheduled
validation slot. Quantiles use nearest rank. Bootstrap uses 10,000 resamples,
seed 20260908 and consecutive non-overlapping blocks of five and ten. Durable
payback requires a nonnegative suffix through the fixed final ordinal. Human
and research costs remain separate and visible. All G3 thresholds are those in
the parent protocol; no fixture or partial horizon can earn a positive gate.

## Qualification boundary

The bounded local check covers executable positive and adversarial cases. A
separate recorded Gradle fixture exercises native cache/up-to-date behavior,
multiple nested commands, detached process termination and instrumentation
overhead. Lean graph/outcome capture is symmetric; deep attribution traces are
diagnostic only. BV-006 must measure overhead and freeze the supported daemon
and capture policies on the actual engineering workflow before confirmation.
Previous command interfaces and closed research decisions stay unchanged.

## Evidence files and explicit external costs

The [runner reference](../dev/history-replay/README.md) lists the bounded
qualification commands. The JSON structures in `contract.go`, `records.go`,
`metrics.go` and `external_costs.go` are the decoder contract: required fields,
duplicate-key rejection and unknown-field rejection apply recursively. The
default unit command requires no user systemd manager; the explicitly selected
integration/native suites require the real platform and never skip missing
prerequisites. No additional Go module or system dependency is introduced.

Each run exports `manifest.json`, `subjects.json`, `attempts.jsonl`,
`costs.jsonl`, `checkpoint.json`, `result.json` and `report.md`. A run that stops
before its first complete pair has no checkpoint; its attempts and typed unrun
slots remain visible. Attempt/cost boundaries and raw files are immutable.
Result/export versions are retained; the root-level files are the latest view.
The checker reconstructs and compares the view, including its Markdown report.

`externalCosts` explicitly imports separately evidenced customer setup,
maintenance, human review or research intervals. IDs must be unique and each
record binds its supporting evidence. Customer-machine adoption is charged
before the prefix excess; later maintenance is charged to its declared ordinal.
Human time is separate. Confirmation requires an adoption record for each
replication, including an explicitly substantiated zero when appropriate.
Receipt synchronization and other research gaps are recovered from the complete
replication envelope. The offline checker/export command remains a separately
recorded research cost, with its scope disclosed in the report.

Only source/runtime-bound launchers execute. The running driver must have the
frozen executable hash. Resume verifies the sealed pair, both state bindings
and the remaining ordinal list before advancing exactly one edge. Missing
native lifecycle evidence retains an unresolved reservation and prevents a
positive decision even if later requests finish. Lost warm daemon memory and
a sealed pair without its completed checkpoint close incomplete.

A normalization selector cannot create an output root or change its producer.
A projector's frozen proof must contain different raw inputs that are
equivalent, a real semantic change with a different expected hash, and malformed
input that fails. The proof is executed before candidate use and independently
by the checker.

Version 2 introduces the explicit optional `outputs.owner` binding and records
owner comparison as a research phase. The Elasticsearch policy binds the exact
C5 output allowlist, lexical reader and full compiler-state reader/classpath.
It permits only the approved generated Checkstyle config, with exact producer,
content and mode checks. All other root-workflow outputs and graph contracts
remain required. The current capture contains exact raw bytes; normalization
is performed on a complete pair outside the customer request.

A reused normalized output must have an identical raw predecessor from an
earlier successful executed producer in the same manifest, replication, arm
and physical workspace. Native outcome and timestamp evidence must support
that origin. Other-arm, future, incomplete or unexplained origins are rejected.
Engineering/confirmation require bound negative/equivalence cases and actual
native reuse qualification, tied to all reader bytes and classpath order.
Actual owner capture overhead remains an independent prerequisite. These
requirements do not broaden C5's semantic rules or change the economic gates.

The live disk guard tolerates a descendant removed during its traversal, as
native temporary files can disappear. Missing owned roots and other I/O errors
still stop execution; immutable artifact inventories retain strict checks.
Its directory batches are cancellable when the child exits. Full size scans
retain the 100-ms polling cadence and run one at a time; independent driver,
free-space and process checks cannot be blocked by that traversal. Both observers
are joined, and real disk errors racing with child exit remain failures. Native completion
is stamped before joining those observers, with a complete postflight disk check
charged to research after the customer boundary. Actual owner overhead proof
is required before those timings can support confirmation.
Acquisition copying preserves exact modes even under restrictive umasks and
creates independent files. Shared dependency layer location does not determine
its admission: the allowlist examines the actual layer contents.
