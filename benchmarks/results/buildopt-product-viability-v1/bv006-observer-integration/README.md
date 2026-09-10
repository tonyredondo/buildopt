# BV-006: observation consumed by the chronological replay CLI

**Decision: R4_NON_GRADLE_CONSUMER_VERIFIED.** R4.1–R4.3 are verified within
the non-Gradle scope. The bounded observer is now used by the real `run`, `check`
and `resume` commands in `dev/history-replay`. This is a local implementation
change, following the [sealed R2/R3 prototype](../bv006-buffered-observer/README.md).
R4.4/R4.5, S1/G0 and product value remain unqualified. CPU screen: **0/36**.

## What changed

An explicit v3 manifest binds the observer policy. Legacy v2 replay remains
compatible; workflow protocol and result schema remain v2. The policy fixes
100-ms sampling, the original 500-ms coverage bound including native endpoints,
bounded pending records/bytes, total output and drain time. It assigns separate
native, sampler and heartbeat CPU masks. The supervisor shares the observer CPU;
native children inherit their own mask and the supervisor restores its mask.

Each observed request owns a sampler, asynchronous writer and external heartbeat.
Sampling starts before the customer envelope; capture stops at native completion
and persistence drains afterward. Receipts include all artifacts, queue counts,
write acknowledgements, clocks, helper identities and CPU costs. Startup/closure
are explicit research phases; the complete replication wall includes concurrent
observation. No overlapping wall interval is added twice.

Failed observation cancels the request while preserving native launch/finish
receipts. Missing or altered completed observation prevents resume before another
workflow starts. Failed and interrupted attempts remain in exports and costs.
For Gradle, CLI snapshots cannot substitute for actual daemon snapshots.

## Verification

| Proof | Observed outcome |
|---|---|
| Actual `validate/run/check`, chronological fixture history | Four workflows with complete receipts/exports; five malformed, implicit or weakened policy variants rejected |
| Controlled 2,104-ms writer pause | Two workflows; nine pending records persisted completely, maximum sampling gap 100.609036 ms in the paused request |
| Writer/collector failures | Eight workflows: write error, short write, writer exit, drain deadline, pending record/byte limits, total byte limit, missing cgroup; each stops replay and retains its native launch |
| Checkpoint/resume | Four workflows, no duplicate starts; altered completed observation rejected before resume |
| Driver death | Three workflows; helpers and detached descendant close, result remains incomplete and warm state cannot be invented |
| Legacy v2 and owner-reader bindings under race | Six workflows; old replay/checker behavior and C5 reader binding preserved |
| Paused writer under race | Two workflows, complete persistence and separate customer/drain boundaries; no reported race |
| Affinity scope under race | Five cases pass: native inheritance, launch failure, set failure, restore failure with child reaping/thread disposal, cancellation |
| C5 output comparisons | Seventeen equivalent/changed/malformed/causal-reuse cases pass without JVM. Three unchanged metadata/JVM cases reuse prior proof; they were not rerun |
| Release checker counterexamples | Six altered artifact sets rejected: missing phase, reordered acknowledgements, hidden pending data, corrupt payload, omitted writer CPU, absent native coverage; hashes rebound where relevant |
| Build and quality | Pinned Go 1.26.5 release/race compilation, unit checks and vet pass; task-relative source delta inspected |

The final consuming set contains **29 fixture workflows: 23 observed and six
legacy**. Six top-level observed tests passed in the last full observer suite.
That command exited 1 because the imported affinity test incorrectly required
the main OS thread to disappear. Go parks that thread instead. A focused
diagnostic identified it, and all five corrected affinity cases then passed
under race. Only that test changed afterward; runtime and other test evidence
remain bound to their actual compiled source. The aggregate failed log is retained.

The paused non-race request took 605.028157 ms inside the customer envelope;
observer closure completed 1174.869043 ms later. Endpoint-inclusive native
coverage gap was 100.317705 ms. These establish timing boundaries in a fixture,
not a product speedup. Race binary timings are correctness evidence only.

See [raw reconstruction](./analysis/result.json), [release checker rejection
proof](./analysis/checker-proof.json), [design and acceptance](./implementation.md)
and [source delta](./analysis/source-diff.json). The supported command is
`./dev/check-history-replay --observer`; its fixture cap is 21 workflows plus
three affinity helper children, with zero Gradle/JVM starts.

## Retained failures and accounting

All initial failures remain in this bundle: unknown observer parsing; two imports
left during source extraction; early supervisor cancellation losing six native
receipts; missing pre-resume validation and Gradle-daemon coverage; supervisor
sharing native CPUs; the affinity m0 assertion; and a counterexample script that
restored bytes but recreated a 0600 file as 0644. Each affected path was corrected
and verified. The latter script now restores both bytes and modes; the release
checker correctly rejected the original mode drift.

| Charge | Phase total |
|---|---:|
| Fixture workflow reservations/charges | 81 |
| Native fixture starts independently observed | 76 |
| Additional starts unresolved after early cancellation | 5 |
| Sampler / writer / heartbeat processes | 75 / 75 / 75 |
| Affinity helper children | 9 |
| Compiler commands, including failed/reproduction builds | 14 |
| New Gradle / JVM helper starts | 0 / 0 |

Of the six missing early native receipts, one launch is recoverable from a
complete raw sample. The other five remain unknown; they are not counted as zero
and all reservations remain charged. Every final workflow has its native PID
receipt. The independent audit verifies 425 retained process identities are gone
and all fixture units are unloaded. No watcher remains active.

The 90-minute phase has a 4-GiB artifact ceiling and 40-GiB minimum free disk;
its amendments and command receipts retain the prospective limits. Recorded
command wall is about 398 seconds; the first three commands lack complete wall
receipts, so their configured 180-second bounds are charged conservatively for
the 1,200-second proof-time check. No claim of exact whole-phase command CPU or
exact total fixture starts is made.

Program Gradle totals remain **646 reservations / 547 actual starts**, including
the same 29 older nested starts. The older possible metadata JVM remains unresolved.
All 80 held-out history transitions are untouched. The historical 2,104-ms pause
cause and the prior cold daemon endpoint failures of 504/603 ms remain unresolved.

## Recovery and next work

The [next-step tracker](./next-step.md) starts with actual-owner cold coverage and
then symmetric cost qualification. The existing overhead test directly invokes
the worker and does not automatically consume this new observer; its old success
cannot qualify the changed instrument. No native allocation is active.

Workspace: `/home/tonyredondo/repos/github/tonyredondo/buildopt`, local `main`,
HEAD `b76ded08c952ebb386576fafce4ae2d8fdcc09f1`, shared Git at that path plus `.git`.
This checkout contains pre-existing dirty work. Actual runtime proof source is
`00692f6f30878c36d60888b296d3482dd8d03e30f1c53f8850ff4a2e937d69c6`;
final source is `268e22edf821bc799c52574421e0977cb9427847227bb4429d1caa132bdadcde`.
Only the affinity test differs. `analysis/result.json` records binary hashes and
distinguishes checkout snapshots from the actual precompiled race binary source.

Raw state: `.tools/state/buildopt-product-viability-v1/bv006-observer-integration`.
Durable locator: `/home/tonyredondo/repos/github/tonyredondo/buildopt/.tools/state/buildopt-product-viability-v1/task-state.json`,
key `engineeringPrefix.observerIntegration`. The [origin map](./origin-map.json)
preserves absolute paths and byte hashes; recorded bindings are original paths,
not silently rewritten portable claims. [Evidence manifest](./evidence-manifest.json)
and [external seal audit](../bv006-observer-integration-seal-audit.json) cover this
bundle. No commit, push, publication, native worktree change or host change.
