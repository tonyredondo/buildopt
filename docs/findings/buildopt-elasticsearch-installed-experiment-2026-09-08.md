# Installed Elasticsearch experiment, 2026-09-08

> **Research disposition: HISTORICAL evidence (2026-09-08).** Original
> findings, limits and outcomes are preserved below. Old recommendations and
> successor instructions are scoped to that campaign. Use the
> [research status register](../research-status.md) and
> [current tracker](../plans/buildopt-product-viability-v1-tracker.md) for new work.

The experiment closes at `STOP_INSTALLED_PERSISTENT_UPLOAD_DEADLINE`.
The reviewed correction passes native admission, all twelve correctness rows,
24 mutation rows and 52 owner test methods, including twelve nested TestKit
builds. Persistent observation delivery fails the unchanged 100-ms prerequisite
on this host. The actual installed wrapper preserves successful native Gradle
execution and leaves the unacknowledged observation in its private queue.

This is a negative installed-admission result. V/L/H/O were not run; installed
savings, overhead, chronological persistence and repayment are unmeasured.
It does not establish that the annotation has no performance benefit. The
[plan](../plans/installed-elasticsearch-native-correction-v1.md) and
[tracker](../plans/installed-elasticsearch-native-correction-v1-tracker.md)
retain the complete history and earlier failed gates.

## Identity and evidence

| Input | Frozen value |
|---|---|
| BuildOpt checkout | `main` at `b76ded08c952ebb386576fafce4ae2d8fdcc09f1`, with the retained local implementation; no publication |
| Elasticsearch B0 | `16bd5bc5355ac7c6ad736f8a6f93281b24a05ab7` |
| Native runtime | Temurin 21.0.12+8, Gradle 9.7.1, Linux AMD64, CPU affinity 0-7, eight workers |
| Installed package SHA-256 | `ab139a092ce934060f56ed3a818b3ad9448e9c5c207e459e6859f0527284ece4` |
| Task preimage SHA-256 | `61fe2eaa06ff463c2a49cea656b147889060855acfa094288ebb8b31567e11b6` |
| Task postimage SHA-256 | `d6858f5ac43ad671496cf1e54e7af309cb98ebda4a9be53e61578df8baefa2d0` |
| Terminal decision SHA-256 | `fe800627b6e91807f927dae8d1f1dacd332e21b89458d0506dda425744fdf524` |

The host-local evidence root is
`/home/tonyredondo/repos/github/tonyredondo/buildopt/.tools/state/eic-correctness-v3`.
`experiment-decision.json` binds the proof files. The recovery locator is
`/home/tonyredondo/repos/github/tonyredondo/buildopt/.tools/state/eic-qualified-correction-v2/task-state.json`.
The three subject worktrees are `arms/N0`, `arms/N1` and `arms/W1` under the
evidence root, on branches `eic-correctness-v3-n0`, `eic-correctness-v3-n1` and
`eic-correctness-v3-w1`. Their shared Git directory is
`/home/tonyredondo/repos/github/tonyredondo/buildopt/.tools/state/eic-native-v1/repos/elasticsearch.git`.
N0 is unpatched; N1/W1 retain the exact annotation change, and W1 additionally
retains its installed bootstrap files. No upstream source reset was needed.

These private artifacts remain on this host. Hashes detect drift against retained
pins; they are not an external attestation or a published reproduction bundle.
Credentials remain in the task-private backend roots and are not in this report.

## Correctness and owner tests

| Proof | Result | Receipt SHA-256 |
|---|---|---|
| C | 12 rows; execution, repetition, cross-root cache restore, benign invalidation, matching tab failure and exact reversion | `34491dac0f86ba2b69d595dd34ac89608d0de5a54b66cd3b845e7da9231dd73c` |
| M | 24 starts; rules, excludes, relative rename, all-source removal, malformed UTF-8 and relative root | `9d6d72a796b61350294c759fd0ae1e0f77337adacd938e67c5936f6b4c920d64` |
| T | 20 unit and six functional methods per arm, with exactly twelve nested requests and no skipped or failed methods | `97dc11f966a5970127cdd1024982341164b2cb1db68db60ae9171714ac3d4a6e` |

The final independent owner checker replays its C/M parents and all four T
results in 602.100 seconds, exit 0. Its raw result and exit record are
`qualification/owner-continuation-check.json` and
`qualification/owner-continuation-check-exit.json`.
The approved output comparisons retain the raw 50,258-entry inventories,
exact generation origins for manifest dates and the qualified metadata rules.
The [output-contract finding](./buildopt-elasticsearch-correctness-output-contract-2026-09-07.md)
explains their limits. None of the historical C rejection receipts was promoted.

The owner accepted composite-entrypoint proposal
`0896a2f4643db749dffd47838923bbc4e859758b03d4ff365bd03e073e9dc9c7`.
The accepted commands use the upstream root build and unchanged test filters:

```text
./gradlew --no-daemon --console=plain --max-workers=8 :build-tools-internal:test --tests org.elasticsearch.gradle.internal.precommit.ForbiddenPatternsTaskTests
./gradlew --no-daemon --console=plain --max-workers=8 :build-tools-internal:integTest --tests org.elasticsearch.gradle.internal.precommit.ForbiddenPatternsPrecommitPluginFuncTest
```

The original standalone T001 failed during settings verification before tests.
Composite T001 then passed all twenty methods, but the reader rejected its
incomplete `whenReady` graph. Its raw native operation trace contains six task
plans and all 36 executed tasks; the notification graph contains only 25.
The corrected reader joins the complete native plans to execution identities
and checks every dependency. It changes neither instrumentation nor Gradle
commands, source, dependencies or test selection.

The new continuation binds and independently verifies that retained successful
T001, then captures only T002-T004 in 132.432 seconds. Exactly three new outer
and twelve nested starts are charged. The original incomplete composite receipt
`90d92b3b3f521d876cadc3f6c319d1b1a103711549827fdee3b69f0e12ee964d`
remains unchanged. Positive and malformed/duplicate/missing-task tests, retained
T001 replay, source/boot/chronology checks and the full runner race suite pass.

## Persistent delivery fails before value admission

The plan requires acknowledged W1 uploads for V/H. Its 100-ms deadline still
preserves the native result when the backend is unavailable. A fresh Btrfs
backend probe attempted twenty distinct canonical observation uploads with that
deadline and then reopened storage. Results in
`persistent-transport-v1/results.json` show **0/20 acknowledged and 0/20 stored
observations after reopen**. All calls reach their deadline, approximately
100.09-101.10 ms. Passing the capture test means the failure was recorded and
checked; it does not mean the delivery prerequisite passed.

A separate zero-Gradle diagnostic uses task-private Go overlays to time the
existing storage operations. Product source files and durability behavior are
unchanged. In its first request, file synchronization costs 29.920 ms and three
directory synchronizations cost 33.402, 33.237 and 33.303 ms. The client expires
at 100.160 ms; the server finishes with HTTP 500 at 131.274 ms. These sequential
durability barriers alone exceed the deadline in the observed request.

The second diagnostic request has an explicitly diagnostic 2-second allowance.
It is acknowledged in 122.260 ms, including a 64.050-ms file synchronization,
33.162-ms observation transaction commit and 22.432-ms audit write. It confirms
that the path can acknowledge the record with more time; it cannot qualify the
unchanged 100-ms production gate. Raw timing logs, overlays, the allocation and
the two responses remain in `transport-diagnosis-v1`.

The final installed check runs exactly two real B0 `help` invocations through
direct annotated Gradle and the existing installed package:

| Row | Path | Native result | Delivery |
|---|---|---|---|
| UI001 | N1 `./gradlew` | Exit 0; 17.150 s outer duration | Direct control |
| UI002 | W1 `./buildoptw` | Exit 0; 25.421 s outer duration | One posted fact, HTTP 500, one queued observation, no acknowledgement |

These are functional help probes, not a paired overhead estimate. They preserve
the frozen flags, CPU affinity, runtime, source and installed endpoint, using a
fresh private backend and outbox. Actual native supervision places the upload
after the child exits. The outer wrapper exits 100.018 ms after the server
receives the post. The queued fact exactly matches that post and has no
prospective or controlled-value authority. Native success is preserved.

The capture helper initially returned exit 1 because its final checker compared
privacy-redacted arguments to raw arguments. Both native runs and their source
checks had already succeeded. The corrected retained-data replay verifies the
product's redaction contract and the digest of the original workflow, exits 0,
and starts no additional Gradle process. The original helper failure and its
exact compiled source are retained; no successful capture result was fabricated.
See `qualification/installed-delivery-checker-diagnosis.json`,
`qualification/installed-delivery-replay.json` and their logs/exit records.

## Verification, costs and next work

The final protocol gate and vet pass in 16.237 seconds. The runner, `buildopt`
and `buildoptw` build successfully with the repository's pinned Go toolchain.
The affected backend and actual retained installed replay pass under the race
detector. Earlier in this continuation, the complete runner race suite passed
in 53.974 seconds. The package bytes, all 146 product qualification inputs and
all ten qualified provenance inputs remain unchanged. No additional dependency,
global temporary-directory change or storage substitution was introduced.

There are **76 cumulative public Gradle starts and 61 local qualification
starts**. Public counts include the failed standalone T request, all sixteen
successful composite outer/nested requests and the two final installed help
probes. Transport diagnostics add zero Gradle starts. The phase carries forward
39,301 charged seconds; `qualification/experiment-closeout.json` records its
final elapsed charge, bounded disk use, source/check bindings and service closure.
All owned native services and the final backend listener are closed. The
worktrees and raw evidence remain retained; nothing is staged or published.

The immediate next engineering problem is durable observation-write latency.
A separate correction must preserve durability and the 100-ms contract, then
pass fresh persistent delivery and actual installed replay before value work
can resume. Dynamic post-child output-manifest binding and the remaining
V/L/H/O adapters are still incomplete and must be qualified together with their
consumers. This closed result supplies no installed saving, overhead threshold,
ordinary-revision persistence, repayment or production-readiness claim.
