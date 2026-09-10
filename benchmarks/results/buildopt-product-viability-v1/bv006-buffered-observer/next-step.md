# R4: qualify the replay consumer before native measurement

R2/R3 are [verified within their controlled scope](./README.md). R4 is pending
integration; R5 and CPU-screen S1 remain blocked on that proof and actual-owner
observation/readiness. The current zero-native allocation is closed.

| Step | Status | Work and checks | Required outcome |
|---|---|---|---|
| R4.1 | pending | Verify this seal, exact source/binaries, dirty checkout and prior C5 rules. Inspect the chronological CLI, sampler call site and output/cost reader. Register bounded non-Gradle work before execution | One concrete source/consumer target and command matrix; no accidental reuse of the old synchronous write boundary |
| R4.2 | pending | Integrate bounded persistence and independent disk-write acknowledgements into the actual consumer. Keep complete raw output, accepted/persisted/pending data, clock boundaries and all helper CPU/drain/storage charges | The consuming CLI accepts only complete successful evidence and retains explicit failed/incomplete attempts |
| R4.3 | pending | Exercise normal/delayed writes, saturation, partial output, cancellation, timeout, driver death and restart through that consumer. Check frozen C5 equivalent/changed/malformed outputs and original disk/ownership/affinity behavior | Source-bound non-Gradle integration proof plus an independent checker that rejects missing, reordered and corrupted evidence; no surviving owned process |
| R4.4 | blocked | After R4.3, declare the smallest useful actual-owner allocation. Include warmups, failures and helpers; freeze command, output/comparator, masks, timing boundaries and cost capture before launch | Every original observation criterion is measured, including native start to first and last daemon snapshot; no endpoint exclusion or relaxed 500-ms limit |
| R4.5 | blocked | Evaluate unchanged S1/G0 and 100-ms imbalance requirements from that owner evidence. Charge collector/writer CPU, drain and output costs symmetrically | Explicit qualified or rejected readiness; a successful build alone is insufficient |
| R5 | blocked | Only after the prerequisite passes, execute the existing 8/4/2-CPU screen under its own allocation | At most 12 starts per profile / 36 total, including warmups and failures; exploratory result and no automatic expansion |

Do not repeat the 80-request pilot per profile or consume the 80 held-out history
transitions to tune this instrument. The controlled result does not explain the
old pause and cannot promote prior failed gates. If consuming proof exposes a
defect, retain it and reopen affected proof under the existing correction limits.

The current sampler/writer lives in test-only diagnostic source. R4.1 must determine
the smallest integration delta; merely compiling that copy does not fulfill R4.
No new product optimization, host change, publication or unrelated cleanup is
needed for the next non-Gradle step. Existing local implementation authorization
continues; this tracker adds no repeated approval requirement.

Resume from `/home/tonyredondo/repos/github/tonyredondo/buildopt` on local `main`,
shared Git `/home/tonyredondo/repos/github/tonyredondo/buildopt/.git`. Verify HEAD
`b76ded08c952ebb386576fafce4ae2d8fdcc09f1` and actual dirty-source identities.
State locator: `/home/tonyredondo/repos/github/tonyredondo/buildopt/.tools/state/buildopt-product-viability-v1/task-state.json`,
key `engineeringPrefix.bufferedObserver`; no active watcher or native allocation.
