# Elasticsearch correctness output contract, 2026-09-07

> **Research disposition: HISTORICAL evidence (2026-09-08).** Original
> findings, limits and outcomes are preserved below. Old recommendations and
> successor instructions are scoped to that campaign. Use the
> [research status register](../research-status.md) and
> [current tracker](../plans/buildopt-product-viability-v1-tracker.md) for new work.

The approved compiler/RAT and generation-provenance/Checkstyle contracts now
pass all twelve fresh C rows and all 24 public mutation rows. Independent replay
verifies C and M. This includes installed cross-root cache restoration, benign
input invalidation, matching native failures and exact input reversion.

The standalone T001 failure below is retained. The owner subsequently approved
the upstream composite entrypoint, and all 52 owner methods plus twelve nested
TestKit builds now pass independent C/M/T replay. The
[terminal installed finding](./buildopt-elasticsearch-installed-experiment-2026-09-08.md)
records the continuation and the final failed persistent-delivery prerequisite.
Value and chronology remain unrun; the output contract itself is verified.

The first campaign remains stopped at `STOP_CORRECTNESS_OUTPUT_CONTRACT`:
C001/C002 were verified and C003 built successfully but failed its original
output contract. Neither that decision nor historical native admission changes.
The second campaign's C008 rejection also remains unchanged.

This finding belongs to the
[installed experiment](../plans/installed-elasticsearch-native-correction-v1.md)
and its [tracker](../plans/installed-elasticsearch-native-correction-v1-tracker.md).

## Observed results

| Check | Result |
|---|---|
| Source | Elasticsearch B0 `16bd5bc5355ac7c6ad736f8a6f93281b24a05ab7`; exact reviewed annotation patch in N1/W1 |
| C001 | Unpatched full workflow succeeds; `forbiddenPatterns` executes |
| C002 | Native repetition succeeds; `forbiddenPatterns` is `UP-TO-DATE`; all outputs match C001 |
| C003 | Annotated full workflow succeeds; `forbiddenPatterns` executes; output comparison rejects the row |
| Inventories | 50,258 entries per capture; identical paths, types and modes |
| Raw equality | 50,055 entries, including all 41,843 class files, are identical |
| Accepted differences | 64 manifest-date outputs pass the existing exact archive/date/provenance checks |
| Unapproved differences | 138 compilation-state files and one license report |

The three Gradle processes exit zero. This is insufficient to claim correction:
the frozen contract permits only its existing manifest-date projection.
The checker therefore stops despite the successful builds. No performance
conclusion follows from these diagnostic runs.

## Compilation state

The graph declares 227 `previous-compilation-data.bin` outputs, each owned by one
`JavaCompile` task. C001/C003 have 226 files and the same one absent output.
Of the files, 138 differ bytewise.

A read-only Java diagnostic uses the retained Gradle 9.7.1
`PreviousCompilationData.Serializer` and requires full input consumption. The
decoded output snapshot, classpath snapshot, annotation-processing data and
compiler API data match in every changed file. The diagnostic compares all
fields and collection members, preserving ordered collections while treating
maps and sets according to their native unordered semantics.

For 137 files, the raw differences are entirely permutations in serialized
integer sets. The remaining file, under `x-pack/plugin/esql/compute`, also
requires comparing other unordered collections. Across the captures the
diagnostic observes 1,365 reordered integer sets. Changing a constant-set member
is rejected. No retained binary is rewritten. This establishes the cause of
this mismatch; it does not qualify or approve a replacement comparison rule.

## License report

`server/build/reports/licenseHeaders/rat.xml` contains the same 8,233 resources
and license results. Every other byte matches after replacing only:

- The source workspace prefix in each resource name.
- The root report timestamp.

C001 restores `:server:licenseHeaders` from cache. Its report is byte-identical
to the retained original native D001 report, including the old workspace path
and generation timestamp. C003 executes the producer in the new N1 workspace,
so it writes that workspace path and its current timestamp. No license result,
approval, resource identity or resource ordering changed.

## Approved continuation

The owner approved the exact proposal with "Aprobado". The accepted contract
compares all decoded compilation-state fields,
allowing only unordered map/set serialization order to vary, for the 227
graph-bound selectors. Paths, modes, absence, values and collection membership
remain checked. Malformed or trailing input, duplicate set members and unknown
decoded types must be rejected.

For the single RAT report, compare exact bytes after projecting only verified
producer workspace prefixes and the root timestamp. Executed timestamps must
belong to the producer's execution; restored reports need byte-identical
provenance from a retained earlier capture or the exact seeded artifact. All
license findings and resource identities remain exact.

All other output rules, the existing 64-file manifest-date rule, failure
behavior, mutation cases, owner tests and numerical value gates remain intact.
The current failed campaign stays failed. Approval leads to local
qualification of these two rules, followed by a separately frozen fresh C/M/T
campaign with cumulative costs and starts. It would not authorize value rows
before correctness passes.

## Evidence and implementation status

Host-local evidence root:
`/home/tonyredondo/repos/github/tonyredondo/buildopt/.tools/state/eic-correctness-v1`.
The continuation locator remains
`/home/tonyredondo/repos/github/tonyredondo/buildopt/.tools/state/eic-qualified-correction-v2/task-state.json`.
These are local artifacts, not uploaded evidence.

The stopped receipt is `correctness-receipt.json`, SHA-256
`8ad7efb0994a3ae54dc8204816b270d9d1dd2b04def6f3379bef47a498dd965a`.
`acquisition/c003-output-audit.json` independently rehashes all retained outputs,
verifies the existing date rules and enumerates all 139 unapproved differences
with their task owners. `acquisition/compilation-semantics.tsv` and
`acquisition/rat-report-diagnostic.json` retain the diagnostic comparisons.
The complete proposed selector list and provenance are in the unchanged
`acquisition/output-contract-proposal.json`, whose historical status remains
`PROPOSED_NOT_ACCEPTED`. The separate accepted decision is recorded in
`.tools/state/eic-correctness-v2/owner-decision.json`, bound to proposal SHA-256
`afd73e4d204c6b634de1943379aa2e97bfc29182024551e20e9b9b257017b731`.

The new reader and [explicit contract](../../dev/installed-elasticsearch-runner/metadata-output-contract-v1.json)
have standalone qualification against 452 real files and negative cases for
duplicates, truncation, trailing bytes, changed hashes/constants, unknown types
and ordered values. Go checks cover exact RAT byte projection, changed license
results, producer/timestamp limits and missing cached provenance. Integrated
retained-data replay passes all 50,258 entries with exactly 139 approved metadata
and 64 original date differences, while the old comparator still rejects C003.
The fresh campaign subsequently verifies C001-C007, then stops at C008 as
described below. The original C003 receipt remains rejected.

The C runner now owns exact transitions, cache transfer, native process capture,
source checks, output retention and independent reconstruction. Public M and T
adapters are implemented and pass their local checks, including JUnit completeness
and fresh allocation boundaries, but their actual Gradle execution remains
unverified behind the failed C gate. The checker and mutation freezer both
reject the stopped receipt without starting another Gradle process.

The new Linux package builds. Two separately allocated local installed Gradle
fixtures verify graph collection through the real wrapper, native process
boundaries and preserved failure. Full launcher and runner tests, race checks
and the protocol checker pass. All three public services and the local backend
listener are closed. This stage consumed three public correctness starts and
two local qualification starts; cumulative totals are 13 public starts and 61
local Gradle qualification starts. Two BuildOpt tooling builds are recorded
separately. M, T and value add zero starts.

## Fresh campaign: installed cache proof and reused output stop

The new campaign root is `.tools/state/eic-correctness-v2`. It freezes the
approved metadata runtime separately from the previous exact/date comparator,
using the same verified Linux package and admitted cache seed. The new C/M/T
allocation carries the prior 13 public starts and reserves 52 fresh starts:
40 outer and 12 nested. No previous failed or unused slot is reused.

| Row | Native outcome | Correctness result |
|---|---|---|
| C001 | Success, original `forbiddenPatterns` executes | Verified |
| C002 | Success, original task `UP-TO-DATE` | Verified |
| C003 | Success, annotated task executes | Verified under the approved metadata contract |
| C004 | Success, annotated task `FROM-CACHE` | Verified |
| C005 | Success through the installed wrapper, `FROM-CACHE` | Verified; cross-root restoration and complete outputs |
| C006 | Success through the installed wrapper, `UP-TO-DATE` | Verified |
| C007 | Success after a benign input change, original task executes | Verified |
| C008 | Success after the same input change through the wrapper | Rejected by the unchanged date-producer rule |

All eight native processes exit zero. The new stopped receipt is SHA-256
`3f41068969ff05bb3f6f974713d04cf198ce050daf97dca06fe63ff9ea426356`.
The checker reports `changed manifest requires an executed producer`.
Current N0/W1 worktrees retain the allocated benign input; they are not reset.

The C007/C008 audit rehashes all 50,258 output entries. Exactly 50,055 entries,
including all 41,843 class files, are byte-identical. The approved compiler/RAT
rules verify 138 differences: 137 compilation-state files and the RAT report.
The remaining 65 differences are:

- The same 64 date-bearing outputs already listed by the date contract. Every
  current producer is `UP-TO-DATE`. Each N0 file is byte-identical to its C001
  artifact, and each W1 file to its C005 artifact. Those original producers
  actually executed. Their original timestamp windows, exact archive content
  and current source-manifest propagation all verify.
- `server/build/reports/checkstyle/main.xml`, owned by `:server:checkstyleMain`.
  Both producers execute. The report version is 13.11.0, and all 4,960 file
  records and findings match. Replacing only the producer prefix in each
  `file name` attribute leaves every other raw XML byte identical. The unchanged
  seeded report also matches the original retained native D001 artifact exactly.

`acquisition/manifest-lineage-diagnostic.json` and
`acquisition/checkstyle-exact-projection.json` retain these diagnostic results.
The first broad lineage probe reached its 170-second test limit while rereading
unrelated origin files. The bounded replacement rehashes all current outputs
and verifies the specific origin artifacts and sealed raw diagnostics; it
passes in 78.17 seconds. Both probe records remain retained. Neither launches
Gradle or changes a comparison rule.

## Concrete next proposal: generation provenance and one Checkstyle report

The new proposal is `acquisition/reused-output-provenance-proposal.json`, SHA-256
`449e56c3924704129ae90d498004392fdfeb2ce51e02f11e6da578d15422a14d`.
The frozen proposal retains `PROPOSED_NOT_ACCEPTED`; a separate accepted decision
in `.tools/state/eic-correctness-v3/owner-decision.json` records the owner's
`Aprobado`. Approval changes only:

1. For the existing 64 date-bearing outputs, permit an earlier executed producer
   as the generation proof when the complete current raw file, path, type and
   mode equal that earlier retained artifact. Validate dates in that producer's
   actual window, preserve current source-manifest propagation and every existing
   archive/non-date check. Reject missing, future, cross-campaign or unexecuted
   origins and any changed cached bytes. Executed outputs still use their own
   invocation windows.
2. For the single graph-bound Checkstyle report, project only verified producer
   workspace prefixes in `file name` attributes. Keep resource identity/order,
   version, errors, messages, severity, line/column and all other bytes exact.
   Cached reports need byte-identical retained or specifically frozen seed
   provenance, following the already accepted RAT provenance pattern.

This required a separate decision because the existing date rule explicitly requires
an `EXECUTED` producer in both current arms and rejects changed cached dates;
the earlier approval also leaves reports other than RAT exact. The new
implementation passes local positive, tamper, malformed and negative tests and
the complete 50,258-entry retained replay in 158.836 seconds. It confirms 64
earlier executed date origins, 138 unchanged metadata projections and one
Checkstyle prefix-only difference. A fresh C/M/T campaign opens only after the
remaining runner qualification and separate freeze. The two failed
campaigns remain failed; value thresholds and all other rules stay unchanged.

The second campaign adds eight public starts and zero local Gradle starts:
cumulative totals are now 21 public and 61 local qualification starts. All eight
owned services and the backend listener are closed. C checking rejects the
stopped receipt, and mutation freezing rejects its C gate without creating a
mutation root. C009-C012 remain closed unstarted; M, T and value add zero starts.
A newly approved full C/M/T allocation would reserve 52 starts and carry the
cumulative public ceiling to 73, with every historical start still charged.

## Third-campaign C/M proof and standalone T failure

The owner-approved provenance runtime verifies all twelve fresh C rows in
`.tools/state/eic-correctness-v3`. Receipt SHA-256 is
`34491dac0f86ba2b69d595dd34ac89608d0de5a54b66cd3b845e7da9231dd73c`.
C008 compares 50,258 entries with 64 exact earlier executed date origins,
138 unchanged metadata projections and one Checkstyle prefix-only difference.
C009/C010 both produce the expected tab failure; C011 restores from cache and
C012 executes after reversion. All twelve cgroups and the local backend close.

Independent C replay passes in 664.843 seconds before public M opens. All 24 M
rows pass in 124.686 seconds, covering rules, excludes, relative rename,
all-source removal, malformed UTF-8 and relative root. Its receipt is
`9d6d72a796b61350294c759fd0ae1e0f77337adacd938e67c5936f6b4c920d64`.
T admission independently verifies M and C in 600.685 seconds. All 24 mutation
cgroups are absent. These timings are functional verification costs, not value.

The owner-test setup context previously began its 180-second limit before
that full parent replay. It now allows 1,800 seconds for preparation and replay;
the actual successful T freeze proves subsequent source checks still run.
Native T remains bounded at 2,700 seconds, 600 seconds per child and four outer
plus twelve nested starts. The original C/M runner is retained unchanged.

T001 starts Gradle but fails in settings resolution after 5.918 native seconds.
No unit method or nested TestKit build executes. The stopped owner receipt is
`dec48bd22e9bf1b53cc0e0ee18b5e07774046fd75e778e74fec652b40d151c11`.
The upstream standalone verification file has `verify-metadata=true` and an
empty `components` element. Gradle therefore rejects the plugin marker POM for
missing checksums. The unchanged main verification file disables metadata
verification and explicitly pins the Develocity implementation JAR; the cached
JAR matches that exact pin. Neither upstream verification file was changed.
The source, report, cached artifact and closure evidence are bound in
`qualification/owner-standalone-diagnosis.json` at SHA-256
`6d0d9e7f48f1b59a5d1a2df720412964a5abe9c5d93567bdb05b564e6c4b9d86`.
The complete owner checker independently revalidates C/M, then rejects the
incomplete T allocation for missing outer starts in 599.503 seconds. It adds no
Gradle start and cannot promote the failed receipt.

## Retained proposal: upstream composite owner-test invocation

The separately retained proposal is
`.tools/state/eic-correctness-v3/acquisition/owner-composite-invocation-proposal.json`,
SHA-256 `0896a2f4643db749dffd47838923bbc4e859758b03d4ff365bd03e073e9dc9c7`.
Its immutable proposal retains `PROPOSED_NOT_ACCEPTED`; the later acceptance is
separately bound by `owner-composite-decision.json`, SHA-256
`db00500b4be8687f6ef24762dac46b5e61382f0806f19ea5b059cf891e06b5a4`.
The following describes the proposal at that earlier checkpoint. At pinned B0,
`BUILDING.md:416` demonstrates the
composite `:build-tools-internal:integTest` entrypoint, and root settings include
the build under that exact name. The proposed commands are:

```text
./gradlew --no-daemon --console=plain --max-workers=8 :build-tools-internal:test --tests org.elasticsearch.gradle.internal.precommit.ForbiddenPatternsTaskTests
./gradlew --no-daemon --console=plain --max-workers=8 :build-tools-internal:integTest --tests org.elasticsearch.gradle.internal.precommit.ForbiddenPatternsPrecommitPluginFuncTest
```

An explicit new owner-test profile would bind the included-build task identities
in raw graphs and execution reports. Both source hashes, all 20 unit and six
functional methods per arm, exact JUnit results and twelve actual nested requests
remain required. Root settings and dependency-verification metadata stay as
upstream supplies them. There are no verification overrides, checksum generation,
forced reruns, test exclusions or Configuration Cache overrides.

The new evidence root would be `owner-tests-composite-v1` under the third
campaign, retaining the verified C/M parents and the failed standalone T receipt.
All old unstarted T slots remain closed. Current cumulative counts are 58 public
and 61 local Gradle starts. A complete new T requires four outer plus twelve
nested starts, carrying the public ceiling to 74: one more than the previous
73-start projection because the failed T001 remains charged. Whole-protocol
planning becomes 331 starts with all historical extras retained. Native T limits
stay unchanged. Local static checks bind the exact upstream sources and command
filters; the new profile is not yet implemented or runtime-qualified. This
prospective command and start-allocation change requires a separate owner decision.
