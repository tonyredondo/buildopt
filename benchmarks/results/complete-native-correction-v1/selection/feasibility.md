# CNC-001: GraphQL Java source feasibility

Date: 2026-09-05. BuildOpt inspection baseline:
`ca5fb5d8c10ac581de7478cc1dec52269da67e24`, plus the uncommitted planning
documents. This is a source investigation, not a tested candidate.

Subsequent scope decision: the owner accepted the local non-CI recommendation.
The [CNC-002 preparation record](../contract/local-scope-and-budget-review.md)
records that acceptance and the new proof-budget question. The CNC-001 decision
below is preserved as the result before that acceptance.

## Decision

**Advancement: `INCOMPLETE_EXPERIMENT_INPUT`.** All five historical blockers
have identifiable owners and plausible native replacement mechanisms. However,
the selected local command and the owner-CI workflow are not equivalent inputs.
The plan cannot yet freeze a behavior-preserving exact-output experiment while
leaving that distinction implicit.

**Recommendation, not an adopted scope change:** first qualify an explicitly
local, non-CI `assemble` workflow, using the native branch-based version path
and Corretto 25 to align the main JDK vendor with the owner workflow. Preserve
the selected command's existing behavior; do not set `RELEASE_VERSION`, freeze
the clock, or disable CI in an actual CI workflow to manufacture equivalence.
Label this a directed developer-workflow study, not owner-CI qualification.

If the owner instead requires CI first, the timestamp/version contract needs
an explicit semantic decision before implementation. No universal impossibility
of Configuration Cache is claimed: the present exact-output comparison and
unchanged timestamp semantics are the conflict. Moving version generation to
execution or changing version policy would be a different, unproved correction.

CNC-001 records a verified source-analysis/refusal result; CNC-002 cannot advance
until the workflow/environment choice is resolved. No public Gradle process,
candidate, patch, new timing, or product failure was produced.

## Source identity and access limitations

The subject is `graphql-java/graphql-java` at
`f2d8c9126f898c084b176631b7346bc6fbec296a`. Its historical Git archive digest is
`83e80bbaa2b3e3308dd35e0b7120399f02149507674d958b0e2e4cc81718d051`.
The prior temporary checkout recorded in the retained root log is absent.
No checkout/worktree was recreated and no filesystem-wide recovery was used.

Instead, exact-commit public files were read without execution. The first three
file hashes below match the retained source/subject bindings. The GitHub
recursive tree response was not truncated and contained no `.bnd` file. This
checks source ownership and configuration inputs, not the full Git archive:
**the historical archive hash was not recomputed**. A later implementation
still needs a registered Git-preserving worktree and an independent archive
verification before source mutation or a Gradle start.

| Inspected file | SHA-256 | Binding |
| --- | --- | --- |
| [build.gradle](https://github.com/graphql-java/graphql-java/blob/f2d8c9126f898c084b176631b7346bc6fbec296a/build.gradle) | `27cfbb5dc699619d3398eb651b9e6cb4dda1d6fd47f87591849d01284841bbbc` | 42,229 bytes; matches the historical source binding. |
| [Owner workflow](https://github.com/graphql-java/graphql-java/blob/f2d8c9126f898c084b176631b7346bc6fbec296a/.github/workflows/pull_request.yml) | `05478c6b84875e6be640c96fc4fda2c4a4383f7a11d777b03710be3430e5ee97` | 11,266 bytes; matches the cohort manifest. |
| [Wrapper properties](https://github.com/graphql-java/graphql-java/blob/f2d8c9126f898c084b176631b7346bc6fbec296a/gradle/wrapper/gradle-wrapper.properties) | `1d126b2f29d1a87825e57ae7a7a1b1cc06f8dcbc89dd614dc6a5b493244ba2cc` | 252 bytes; selects Gradle 9.6.1; matches the cohort manifest. |
| [settings.gradle](https://github.com/graphql-java/graphql-java/blob/f2d8c9126f898c084b176631b7346bc6fbec296a/settings.gradle) | `b2df82d2e3c57a8ae2b916389aeeb50fad13fac8d9a9c5f0d2411bda0d812034` | Includes `performance-results-page`; newly inspected source binding. |
| [Bnd BundleTaskExtension.java](https://github.com/bndtools/bnd/blob/47e504d7881ba466703c55a8dca7b0578561582d/gradle-plugins/biz.aQute.bnd.gradle/src/main/java/aQute/bnd/gradle/BundleTaskExtension.java) | `5d9532d8ee2b82580748f225b0114378d1cbe646d21c4abf513177bf15f520c0` | Bnd 7.1.0 tag resolves to this exact commit; 20,657 bytes. |
| [BndBuilderPlugin.java](https://github.com/bndtools/bnd/blob/47e504d7881ba466703c55a8dca7b0578561582d/gradle-plugins/biz.aQute.bnd.gradle/src/main/java/aQute/bnd/gradle/BndBuilderPlugin.java) | `9172a430ae90a78160b53c80d6331e791da3542f63ae9c4ec81be13dfcc708cb` | Registers the `bundle` extension and its `buildBundle` action on the regular JAR. |

Also inspected at the subject commit: `AGENTS.md`, `gradle.properties`, and
`performance-results-page/build.gradle`. The subject prohibits new dependencies
and uses Spock for tests; no subject code was changed. The included Java
subproject requests Java 21, in addition to the root Java 25 toolchain. Its
HTML-generation task is not an excuse to omit its normal assemble outputs.

The tree API response is 572,136 bytes with SHA-256
`e56998d6c07044b5d2485854c4f544c5fdcaab976dc89c32df612d6b6f472cf0`.
This is a response digest, not a Git archive digest or a binary-plugin digest.
The installed Bnd binary has not been newly acquired or hash-verified.

## Historical diagnostic binding

Both retained reports were decompressed and their report-data JSON parsed
without executing the embedded page scripts. Each has exactly five entries
with a `problem` field. Their stack summaries agree on all five owners below.

| Retained report | Compressed SHA-256 | Decompressed HTML SHA-256 |
| --- | --- | --- |
| [Strict 1](../../wrapper-coordinated-native-corrections-v1/wcncp-e009/diagnostics/graphql-java/configuration-cache-strict-1/configuration-cache-reports/report-001.html.gz) | `56438ea0e0aa6d63a2c110b9853253f13b402feced242a20c09aa6f40a5529fb` | `d25e4b70f8e033d1357c5ef83a0771a2d3f0a6bea3df873dd0379878e7b884fe` |
| [Strict 2](../../wrapper-coordinated-native-corrections-v1/wcncp-e009/diagnostics/graphql-java/configuration-cache-strict-2/configuration-cache-reports/report-001.html.gz) | `f873198f5ea9c732ad27a9cf7325594aade926cf8370ca4259c14ecdd2bdd9aa` | `eb361c4b9dd0e7f8e143646cb39d23f2c1be828b26fb4ec7773fefa2fe444973` |

The `:jar` exception is specifically
`BundleTaskExtension$BuildAction.lambda$execute$0`, line 407, not an unidentified
closure in GraphQL's build script. Task name alone did not establish ownership.
These are historical diagnostics, not two fresh CNC runs or an exhaustive
report for a different environment or task set.

## Five-blocker ledger

All subject spans below refer to the exact `build.gradle` digest above. Bnd
spans refer to the exact `BundleTaskExtension.java` digest above.

| ID / declaration and phase | Consumers and original behavior | Native replacement hypothesis | Required proof before admission |
| --- | --- | --- | --- |
| B1: `getDevelopmentVersion`, 140-173; external call at 144; configuration | Executes Git worktree detection with `projectDir` as `-C`; waits for stdout/stderr; anything other than trimmed `true` selects the timestamped no-Git version. Does not explicitly reject a nonzero process exit. `version` assignment at 180 consumes the result. | A tracked `ValueSource`/`ExecOperations` result preserving command, streams, launch failure, trimming and fallback; no inference from a `.git` directory alone. | Git directory and Git-file worktrees; no repository; absent Git executable; nonzero/empty/malformed output; changed working directory; interruption; no dropped stderr/changed fallback. |
| B2: same method; external call at 169; configuration | In the non-CI Git branch, obtains the branch name, trims it and replaces slash/backslash with hyphens. Produces a branch-based snapshot; detached HEAD gives a HEAD-based snapshot. CI takes a different call at 156 and returns timestamp plus short commit hash. | Track the consumed Git result and environment, including `CI` and `RELEASE_VERSION`; preserve the existing branch-specific computation. | Branch rename, detached HEAD, new commit, CI transition, release override, empty/nonzero output. Preserve the CI branch's empty-hash diagnostic and exception. No untracked clock capture or false CI equivalence. |
| B3: first `buildFinished`, 186-197; registered during configuration, consumes build result at completion | Prints success or the build failure and the selected version. No deletion, file mutation, or cleanup is present in this callback. | Isolated Gradle dataflow action consuming build-work result and explicit version. | Success, configuration failure, task failure, cancellation, reuse, exactly-once printing and relative ordering. Build-work result coverage must not be assumed identical to all `BuildResult` cases. |
| B4: second `buildFinished`, 938-968, helper 970-975; state populated at 751-788 | Prints test totals, failure identities, top 20 slow classes and top 50 slow tests. `afterTest` counts all results, collects failures, accumulates elapsed time and includes only tests slower than 500 ms in ranked maps. No file cleanup is present. | Execution-owned shared service/test listener plus ordered final reporting; use primitive/string records, not captured `TestDescriptor` or script collections. | Zero tests in `assemble`; multiple test tasks; failures/skips; timing cutoff and ranking; service reset on each reused build; cancellation; exact first-summary/second-summary ordering. Do not delete summaries or invent equivalent task-completion-only test data. |
| B5: Bnd `BuildAction.execute`, 393-408, actual access at 407; JAR execution | The default `properties` map contains a sentinel under `project`; the action replaces it with `Task.project`. Bnd uses the map for instruction evaluation and constructs the bundle manifest. | Configure the existing public `bundle.properties` API with only necessary explicit values. Bnd's own 143-180 documentation prescribes this for Configuration Cache. No plugin upgrade, action removal or third-party source patch is required by this hypothesis. | Inventory Bnd instructions and macros; preserve all needed scalar values; compare complete JAR bytes and manifests; cover each realized Bnd extension, including shadow-related configuration; never remove bundle generation. |

The process proposals use the frozen Gradle version's
[external-information APIs](https://docs.gradle.org/9.6.1/userguide/configuration_cache_requirements.html).
The lifecycle proposal uses
[dataflow actions](https://docs.gradle.org/9.6.1/userguide/dataflow_actions.html),
an incubating public feature, and the
[TestListener API](https://docs.gradle.org/9.6.1/javadoc/org/gradle/api/tasks/testing/AbstractTestTask.html).
These are source-backed hypotheses, not fixture proof. In particular, lifecycle
ordering and configuration-failure coverage remain meaningful risks.

The version-pinned [FlowProviders API](https://docs.gradle.org/9.6.1/javadoc/org/gradle/api/flow/FlowProviders.html)
explicitly makes the result available after a configuration failure prevents
execution. This strengthens the proposed first-listener replacement; it does
not prove partial-registration, cancellation, ordering, or shared-service
behavior in this build. Those cases remain in the fixture matrix.

## Version, workflow, JDK and outputs

The previous manifest records `assemble`, but the actual owner workflow's
`check` matrix entry runs `assemble` followed by a separate
`check -x test -x testng` invocation. It also has independent Java test, jcstress
and javadoc jobs. A successful local `assemble` would not prove that whole CI.

GitHub Actions sets [CI to true](https://docs.github.com/en/actions/reference/workflows-and-actions/variables).
With no release override, the inspected script's CI path generates a timestamp
to the second plus a short Git hash. That value flows into archive names and
Bnd's default bundle version. A later native/native run may therefore produce
different required paths and bytes even with identical source. Freezing this
configuration value on reuse changes behavior; tracking it without changing
its semantics can invalidate configuration on each changed timestamp.

In the local Git path the timestamp is computed but not used in the returned
branch-based version. The historical
[output inventory](../../wrapper-coordinated-native-corrections-v1/observations/graphql-java/outputs/invocation-1.json)
contains both the main JAR and a `jcstress` JAR with HEAD-based snapshot names.
It supports the environment distinction, not fresh output equality or proof
that the current subject's complete output inventory has only two files.

The main artifact producer chain is `shadowJar` -> `extractWithoutGuava` ->
`cleanShadedClassAnnotations` -> `buildFinalJar`. The regular JAR and shaded
intermediate stay under `build/intermediates`; `buildFinalJar` owns the final
unclassified JAR and replaces the API/runtime outgoing artifacts. Bnd's
manifest is carried through cleanup into that final JAR. Clearing a Bnd
property map cannot be justified just because the regular JAR is intermediate.

The owner already supplies `verifyShadedClassAnnotations` and
`compileShadedJarConsumer` as focused artifact checks. The proof matrix must
cover them, the other requested artifacts, and the listener tests. It must
not substitute the lone final JAR for the complete requested inventory or
pretend that an `assemble` invocation also ran the owner's following check.

JDK resolution: the workflow declares Corretto 25; root source requests Java
25, while the included subproject requests Java 21. Historical controlled
capture used Temurin 25. These are distinct environment bindings. Recommend
Corretto 25 for both native arms and an explicitly pinned Java 21 subproject
toolchain, but **no execution vendor/version/package choice is frozen yet**.
No JDK was installed or launched. The next contract must choose and hash the
actual runtimes rather than inheriting whichever vendor is available locally.

## Span fingerprints

Ranges are inclusive, one-based lines. Hashes cover original UTF-8 bytes with
their original line endings, including each selected final newline.

| Source / span | SHA-256 |
| --- | --- |
| Subject 140-173 | `02c563e01c795bc1dbe4f4087cd88e8d14cec9b88dddb103e82ea7c87323300e` |
| Subject 144 | `f3f00a9b11758c5356f5dac6817929646653f157936ae794a39eae452689f8b5` |
| Subject 169 | `c9f6ba16c19f2200760fa3f55fae45fb4444b36319ffed55fdb2d7a0610f4823` |
| Subject 186-197 | `4ba1d2d651280f4eaebe6ba7e7956bc81e4012b63a229c394f5e6956d6927645` |
| Subject 751-788 | `5f4405ce448bbbddd2b1403a687dd99e6f6c995056b183f7bde41fbe9e6b49c0` |
| Subject 938-980 | `104afb47711254e5e886d3e9fcb9bf245f6434703cc0dfafaa21e88999593f72` |
| Subject 468-495 | `0ebff2808ad0f53f8e8d121754df3364bcfeb88a866ce2a2e36b14a9c28246d2` |
| Bnd 143-180 | `0257d387c9f2ebce07d02f81fd45631158660296b4b63df4bee700110b97cf6f` |
| Bnd 393-408 | `718118dc57c9464100722d199024df69f77bdcd41a2fb03cbe948bb8e29b92f9` |

## What can happen next

Resolve the local-versus-CI scope and runtime binding first. For an explicitly
local study, CNC-002 can then allocate the complete proof matrix and CNC-003
can validate the lifecycle and Bnd hypotheses in fixtures. Discovery of an
additional unsupported blocker still stops public candidate timing.

This analysis does not spend any of the proposed 40 Gradle-start slots or
claim the earlier configuration time as attainable saving. Source-inspection
and documentation effort is research cost, not zero-cost optimization. The
initial setup was not continuously timed; no exact human-cost or phase-payback
figure is claimed. No experiment worktree, cache, runtime or watcher exists.
