# R4 actual replay consumer integration

Target: `/home/tonyredondo/repos/github/tonyredondo/buildopt/dev/history-replay`,
native Linux x86_64, local main b76ded08c952ebb386576fafce4ae2d8fdcc09f1,
shared Git `/home/tonyredondo/repos/github/tonyredondo/buildopt/.git`.
State locator: `../task-state.json`, `engineeringPrefix.observerIntegration`.
Existing dirty source is preserved under `inputs/source-before`; all 20,638
entry files are hashed. Prior R2/R3 seal and 417 artifacts verified on entry.

R4.1/R4.2/R4.3 verified; R4.4/R4.5/S1/G0 blocked; no native allocation.
The sampler was exclusively test-only and not consumed by the real command.

Implement an explicit v3 manifest extension with an immutable observer policy.
Keep v2 manifests, protocol bytes and results compatible. Bind observer limits,
100-ms cadence, 500-ms criterion and separate CPU masks; no ambient environment
can enable observation or weaken policy. Run the sampler as an owned child of
the driver, with the already verified writer and heartbeat as its descendants.
Start observation before the customer envelope; close capture on native completion
and join/drain afterward. Preserve original disk, worker and output contracts.
Failure prevents a usable attempt; parent death closes the helper lifetime pipe.

Require complete raw samples, ordered acknowledgements, clocks, queue counts,
helper identities/CPU and final closure in the offline checker. Research wall
already covers the replication interval; do not double count overlap. Explicit
startup/closure cost phases and an observation export expose helper CPU and bytes.
Missing or altered evidence cannot fall back to unobserved success. Recovery
retains interrupted artifacts and prior costs.

Proof matrix: CLI validate/run/check, chronological fixture replay, corrupt output
and missing final evidence, controlled writer delay/error/short-write/saturation,
cancellation/timeout/driver death and resume, existing C5 comparison/reuse tests,
unchanged disk/ownership checks, normal/race compilation, unit checks and vet.
Use actual user systemd units and temporary Git fixtures; zero Gradle/JVM and
no seed-history transitions. Limits are prospectively recorded in allocation.json.

## Implementation and acceptance

| Part | Status | Implemented behavior and proof |
|---|---|---|
| R4.1 target and contract | verified | Prior 417-file seal and entry hashes; explicit v3 observer binding, unchanged v2 workflow protocol/result bytes; parser accepts v3 and rejects missing/weakened/implicit policy |
| R4.2 real consumer | verified | `run`, `check` and `resume` consume observer evidence; one owned sampler with bounded writer and external heartbeat per request. Supervisor and sampler use observer CPU; child inherits native mask; restoration checked |
| R4.3 runtime paths | verified | 21 final observed fixture workflows, six legacy v2 workflows under race and two delayed-writer workflows under race; five affinity cases pass after correcting the m0 test assumption |
| R4.3 output and failure paths | verified | Eight explicit writer/collection failures; driver death retains incomplete result; checkpoint resume preserves costs and refuses altered completed evidence. Six rebound/corrupted artifact sets rejected by release CLI |
| R4.3 C5 boundary | verified within unchanged scope | Seventeen actual lexical/output/reuse cases plus Go owner-reader binding rejection; three metadata/JVM cases not rerun. Entry hash audit binds unchanged reader code, qualification and classpath to prior proof |
| R4.3 compiler and quality | verified | Pinned Go 1.26.5 normal/race compilation, unit checks and vet. Task-relative source delta inspected; exact source snapshots and binary hashes retained |
| R4.4/R4.5 actual owner | blocked | No Gradle/JVM start in this phase. Cold daemon coverage and symmetric cost still need native evidence; the old overhead test calls `workers[arm].invoke` directly and does not consume this new observation path |

The final runtime source digest is
`00692f6f30878c36d60888b296d3482dd8d03e30f1c53f8850ff4a2e937d69c6`.
Final checkout digest is
`268e22edf821bc799c52574421e0977cb9427847227bb4429d1caa132bdadcde`.
Their only difference is `observer_affinity_test.go`: Go parks m0 permanently,
so its presence in `/proc` does not establish runtime reuse. A three-case
diagnostic observed the parked main thread and removal of two secondary threads.
The final test forces a secondary thread and preserves the original disappearance
and child kill/reap assertions. All five affinity cases then pass under race.
Runtime code and all other tests are unchanged; unchanged compiler/consumer proof
is reused rather than rerunning 21 fixture workflows for a test-only correction.

The release checker additionally rejects missing phases, reordered acknowledgements,
hidden pending data, changed payloads, omitted writer CPU and missing native
coverage, even when artifact hashes are rebound. Every deliberate mutation was
restored byte-for-byte and mode-for-mode. The first checker script restored bytes
but recreated a removed 0600 file with mode 0644; the checker rejected this too.
That failed script and its diagnosis are retained; v2 restores both and passes.

An earlier cancellation implementation killed the supervisor before native launch
and finish receipts were saved. Six missing receipts remain historical evidence;
one native identity is recovered from complete raw samples, five starts remain
unresolved. All 81 workflow reservations are charged, 76 starts are independently
observed, five are unknown. Every final failure has a native launch receipt. No
failed attempt is promoted to successful evidence and no replacement is free.

Measured proof: 29 final workflows, 75 sampler/writer/heartbeat helpers each over
the whole phase, nine affinity helper children, 14 compiler commands. The paused
non-race request persisted all nine accepted rows with a 100.609036-ms maximum
sampling interval; native endpoint-inclusive coverage gap was 100.317705 ms.
Customer wall was 605.028157 ms; observer closure occurred 1174.869043 ms later.
The latter is retained research wall, not removed from the study accounting.
Race timing is correctness evidence only. Prior synchronous timings use a
different fixture and are not a fresh product speedup comparison.

Reproduce the supported consuming suite with `./dev/check-history-replay --observer`.
Use `analysis/result.json`, `analysis/checker-proof.json`, source-specific command
receipts and raw attempts for this execution. A successful full final aggregate
suite is not claimed: all observed cases passed in that run, its sole affinity
assertion failed, and the affected affinity cases passed in the subsequent run.
