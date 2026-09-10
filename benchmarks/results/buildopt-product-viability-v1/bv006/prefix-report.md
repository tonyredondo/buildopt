# BV-006: owner readiness failed before candidate history replay

Decision: **INSUFFICIENT_READINESS**. BV-006 is **partial**. The fixed owner
overhead experiment completed, but its capture-imbalance gate failed. The
candidate engineering sequence and confirmation freeze remain blocked.

This is an instrument result. It neither rejects the C5 correction's existing
correctness proof nor establishes product savings. Seed validation ordinals
21..100 remain untouched.

## Actual owner measurement

The frozen workload is Elasticsearch `:server:precommit`, offline, with native
cache enabled, Configuration Cache disabled, eight workers and persistent
private daemons. The N arm uses the declared native baseline. The I arm adds
the exact five-file C5 candidate. Both start from independent fresh state at
`53a80bec683ad0b065ed9ebe6a57f984e6a91ed1`.

The allocation fixes 20 native starts: four warmups, then four plain/lean pairs
per arm in alternating order. Every native request exits successfully. All
20 command identities, source checks, final states and process closure are
verified. The [independent reconstruction](owner-overhead-reconstruction.json)
checks the raw sample/receipt bindings and recomputes the original metrics.

| Criterion | Measured | Required | Outcome |
|---|---:|---:|---|
| Wrapper p95, 16 measured requests | 1.352567 ms | At most 10 ms | Pass |
| N median extra native time with capture | 139.333503 ms | Used in paired imbalance | Observed |
| I median extra native time with capture | 728.016001 ms | Used in paired imbalance | Observed |
| Absolute difference between arm medians | 588.682498 ms | At most 100 ms | Fail |
| Independent native-exit timing bracket | At most 5.996661 ms late | Diagnostic proof of the repaired boundary | Verified for the fifth request |

Nearest-rank medians use the second sorted value of four, as fixed before
execution. The original values, in execution-pair order, are:

| Pair | N lean minus plain | I lean minus plain |
|---:|---:|---:|
| 0 | 118.080438 ms | 771.008410 ms |
| 1 | 139.333503 ms | 98.800940 ms |
| 2 | 766.073645 ms | 954.129575 ms |
| 3 | 381.518189 ms | 728.016001 ms |

The gate remains failed. No sample was removed, threshold relaxed or extra
native measurement added to seek a passing result. The measured difference
does not by itself establish a permanent collector bias: the observed ranges
overlap, and this allocation does not isolate all daemon variability.

Both styles use the same owned worker and resource guard. This experiment
measures incremental graph capture and wrapper boundaries; it does not prove
the absolute CPU cost of all observation against an unmonitored native build.

## What the existing data rules out

The [diagnosis](owner-overhead-diagnosis.json) verifies all eight measured lean
graphs have the same complete root contract after root-path normalization and
removal of the one approved candidate-only config. All 16 measured requests
have 1,256 `UP-TO-DATE`, 44 `NO-SOURCE` and one `EXECUTED` root task. The three
Checkstyle tasks are `UP-TO-DATE` throughout those measured requests. These
are repeated-build overhead measurements, not activated-mechanism savings.

Most paired timing variation is inside the Gradle daemon. The retained data
cannot separate graph materialization, Groovy/JSON serialization, JIT/GC and
other daemon scheduling well enough to assign a causal fix.

A bounded standalone Java diagnostic replays both actual 1,330-record files.
Eight outputs, containing 10,640 records in total, are byte-identical to their
inputs. Reopening the output channel for every record costs 14.36–31.56 ms;
keeping it open costs 7.85–9.25 ms. This isolates output syscalls and excludes
Groovy dispatch, object construction, serialization and native scheduling.
It does not support a file-open-only change as the main remedy for the
588.68-ms failed gate. That proposed shortcut is discarded for this recovery;
the production writer was left unchanged.

## Verified prerequisites and retained corrections

- The owner reader passes 20 equivalent/adversarial cases, including corrupt
  actual donor bytes. The complete retained C5 pair covers 50,020 exact,
  64 permitted date, three Checkstyle, one RAT and 137 full compiler-state
  comparisons, plus the exact generated config.
- Two actual native calibration pairs verify reuse; the later pair has 268
  proven earlier executed origins. The updated lookup completes that pair in
  12.830 seconds, compared with the previously observed 110.067 seconds.
  This is research processing cost, not product acceleration.
- The original calibration clock is excluded from value evidence. An external
  observer measured at least 5.234 seconds of delay after native CLI exit.
  The corrected worker stamps completion before joining its cancellable disk
  observer and retains a complete postflight resource check. The new actual
  owner observation bounds the delay below 6 ms for its fixed target request.
- Current unit/vet, all integration cases, race checks, 18 real native Gradle
  starts and the complete Go build pass. Integration cases retain their
  160-second individual limits after a combined group exceeded its deadline.
- Acquisition preserves exact modes and independent files; live traversal
  tolerates deleted temporary descendants while retaining strict root/error
  checks. Eight bounded durable evidence copies replace measured slow serial
  archival. Earlier setup errors, interrupted calibration and timeouts remain
  in the evidence; no failed attempt supplies positive proof.
- The [v5 source archive](owner-readiness-v5/source-manifest.json), reader proof,
  original actual-owner plan and all command versions are retained. One
  diagnostic-generated bytecode file was removed to restore the exact frozen
  source hash before native execution; no source bytes were changed.

## Unrun work and cohort limits

| Deliverable | Current result |
|---|---|
| Candidate engineering anchor and ordinals 1..20 | Not run; blocked by owner capture qualification |
| Engineering action/invalidation/startup/value analysis | Not measured; source-path classifications are metadata only |
| Confirmation protocol, allocation and fresh replication roots | Not frozen; an unqualified collector cannot supply G0 |
| 404 confirmation workflows | Not started; sizing and freezing remain after the engineering prefix |
| Six possible replication owners | [Inventory frozen](subjects.json); builds and opportunities unmeasured |
| H3 seed and possible replication cohorts | [Four registered, three explicitly incomplete](adaptive-cohort/README.md); no controller design or execution |
| G3/G4, installed delivery, adaptation and customer viability | Unproved |

The earlier N/native calibration uses only the anchor and first descendant;
it is not the requested C5 engineering sequence. Earlier C5/native discovery
exposure remains disclosed. No validation timing was inspected for tuning.

## Next action in the same BV-006 step

1. Freeze a diagnostic version that measures graph materialization, JSON
   serialization, callback writing and the relevant daemon phases separately.
   Preserve the original 20 samples and keep diagnostics out of primary
   performance claims. Declare its native/tool/time/disk allocation first.
2. Use that attribution to select one collector correction. If measurement
   variability instead limits resolution, preregister one sufficiently sized
   qualification design before further native starts. Do not accumulate
   repetitions until a point estimate happens to pass.
3. Requalify changed output, failure and lifecycle contracts, then complete a
   fresh owner qualification with the unchanged 10-ms/100-ms limits. A source
   or policy change receives a new immutable version; old proof is not relabeled.
4. Only after G0's instrumentation prerequisite passes, execute all engineering
   ordinals 0..20, retain every outcome, then size and freeze confirmation.

This continues H1/C5. It reopens no retired cache, fragment, worker-tuning,
CNC or installed-upload research route. All owned native services are closed;
there is no background replay or publication.
