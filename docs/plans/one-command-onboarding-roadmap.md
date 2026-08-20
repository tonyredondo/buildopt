# One-command POC onboarding roadmap

## North star

BuildOpt's target onboarding experience is:

```text
install BuildOpt -> open a Gradle repository -> buildopt optimize build
```

That one command must discover the repository and Wrapper, establish an
optimized-native control, derive the exact Git change and declared Gradle
outputs, propose a smaller structural graph, calibrate it, decide whether it
has net value, materialize any qualifying profile, and replay it on later
matching builds. A user must not hand-author BuildOpt manifests, graphs,
changes files, evidence documents, or qualified-profile JSON.

The magic is orchestration, not hidden risk. BuildOpt may automatically use a
profile inside the explicitly invoked POC command only after proving
equivalence and net wall-time value. Ambiguous workflows, outputs, changes or
bindings retain optimized native Gradle. Autonomous production promotion,
soak, design-partner validation and Test Optimization remain outside this
roadmap.

## Customer promise

For a supported Gradle workflow, BuildOpt will answer four questions without
requiring BuildOpt expertise:

1. **Can this change request less Gradle work?**
2. **Does the smaller graph produce the same required result?**
3. **Is it materially faster after every BuildOpt cost is included?**
4. **Will the calibration repay before the profile is likely to become stale?**

The command returns one of three honest outcomes:

- `QUALIFIED_AND_USED`: a verified profile is selected and the measured net
  saving is reported;
- `LEARNING`: BuildOpt has not accumulated enough comparable evidence and the
  current build uses optimized native Gradle; or
- `NATIVE_RETAINED`: the workflow is unsupported, unsafe, drifted or lacks
  reproducible value, with the exact reason shown.

No cache hit, graph reduction or avoided-task count is itself a success. The
north star remains customer-visible wall time against optimized native Gradle
with equivalent required outputs and no additional product failure.

## Target experience

### First invocation

```bash
buildopt optimize build
```

For a conventional build, BuildOpt should:

1. locate `gradlew` or `gradlew.bat` and inspect the active JDK and Gradle
   environment;
2. derive the comparison base from immutable CI metadata or a local tracked
   branch, failing clearly when no unique base exists;
3. execute the requested native workflow once and inventory its non-empty,
   repository-contained Gradle-declared outputs;
4. discover the typed Gradle graph and map the exact change to project owners;
5. construct only complete candidates whose terminal task semantics are
   supported;
6. run isolated, order-balanced native/candidate calibration with exact output
   comparison and full-graph fallback;
7. calculate net saving, uncertainty, tail behavior and repeated-build
   break-even;
8. write machine-readable state and a short human report under `.buildopt/`;
9. use the candidate only if every correctness, value and payback gate passes;
   otherwise return the native result.

The initial command may take longer because it owns calibration. It must show
phase progress and estimated remaining work instead of appearing stuck. It
must be resumable only from exact digest-bound checkpoints.

### Subsequent invocation

```bash
buildopt optimize build
```

BuildOpt should validate the current repository, Wrapper, graph, options,
change, output contract and executable against its local portfolio. A matching
profile should produce a concise plan before Gradle starts:

```text
BuildOpt matched profile kotlin-jvm-resource/v1
Projects: 133 -> 9
Expected saving: 49.5 s/build
Calibration payback: 10 matching builds
Fallback: original `build` workflow
```

Unknown or global changes, option drift and stale generated state must select
the original workflow automatically. Profile lookup and decision overhead are
included in the observed end-to-end result.

### CI invocation

The GitHub Action and GitLab component should expose the same command with no
BuildOpt-internal inputs:

```yaml
- uses: tonyredondo/buildopt@<immutable-revision>
  with:
    command: optimize build
```

CI should derive immutable base/head revisions from the provider, restore only
digest-compatible calibration state, publish the human and JSON result, and
persist a qualifying profile portfolio as a normal cache/artifact. It must not
require a BuildOpt server, dashboard or operator-managed database for this POC
path.

Teams may optionally connect the same command to an owner-operated
`buildopt-server` that centralizes Gradle remote-cache objects and compatible
BuildOpt state across developer machines and CI providers. The local path
remains complete without that service. The separate
[centralized cache and state roadmap](./centralized-cache-and-state-roadmap.md)
defines namespaces, HTTPS/authentication, cross-commit applicability,
fallbacks and the completed two-machine/equal-opportunity value proofs.

## Generic architecture

The implementation remains repository-name independent. Repository-specific
facts are generated data, not product branches.

```text
CLI / CI metadata
      |
      v
Workflow and output preflight
      |
      v
Typed graph and change ownership
      |
      +---- incomplete / ambiguous / unsupported ----> optimized native Gradle
      |
      v
Candidate portfolio
      |
      v
Isolated calibration and equivalence
      |
      +---- unsafe / no value / poor payback --------> optimized native Gradle
      |
      v
Digest-bound qualified profile
      |
      v
Automatic POC replay with native fallback
```

The generated portfolio may contain several profiles for one repository: for
example dependency-source, resource, packaging or multi-module source change
families. A Ktor profile is never copied into Spring. The same generic engine
creates independently bound profiles for both repositories.

## State model

Each workflow/change family moves through a small auditable state machine:

```text
UNSEEN -> DISCOVERED -> CALIBRATING -> QUALIFIED -> ACTIVE
                         |               |           |
                         v               v           v
                    NATIVE_RETAINED   STALE ------> RECALIBRATING
```

- `UNSEEN` and `DISCOVERED` always execute native Gradle.
- `CALIBRATING` may run additional isolated arms within the declared budget,
  but the native result remains authoritative.
- `QUALIFIED` requires output equivalence, positive net value, acceptable
  tails, zero additional failures, full fallback and worthwhile payback.
- `ACTIVE` means automatic selection only inside the owner-invoked POC command;
  it does not set `productionAuthorized=true`.
- `STALE` immediately disables the candidate. Recalibration never inherits
  evidence whose bindings changed.

## Implementation sequence

| Order | Tracker block | Deliverable | Acceptance criterion |
| ---: | --- | --- | --- |
| 0 | `POC-NEW-FAMILY-INSTALLED-PROFILE-REPLAY-001` | Replay Ktor's three reviewed profiles through the public package. | Clean checkouts select exact plans, option drift retains native and contemporary outputs equal native Gradle. |
| 1 | `POC-MAGIC-ONBOARDING-CONTRACT-001` | Define `buildopt optimize`, its state machine, authority, outputs, exit behavior and resumable state. | One normative CLI/JSON contract; no manual internal files in the supported path; production authority remains false. |
| 2 | `POC-MAGIC-AUTO-DISCOVERY-001` | Derive Wrapper, command, CI/local base, exact changes, Gradle output candidates and graph automatically. | Conventional packaging, classes, verification, distribution and build-owned test-preparation fixtures reach a proposal from command arguments alone; ambiguity retains native. |
| 3 | `POC-MAGIC-CALIBRATION-001` | Orchestrate preflight, propose, measure and evaluate behind one command with progress, budget and exact checkpoint reuse. | **Done:** a clean repository reaches a deterministic decision without invoking internal subcommands; exact evidence resumes and an insufficient pair budget makes no claim. |
| 4 | `POC-MAGIC-PROFILE-PORTFOLIO-001` | Classify observed changes into exact structural families and maintain multiple qualified profiles without repository-name rules. | **Done:** dependency, resource, leaf and mixed-source facts map to distinct logical families; independently qualified families coexist, the same family replaces only its exact binding, exact state resumes and tampering is rebuilt from valid evidence without granting selection. |
| 5 | `POC-MAGIC-AUTO-REPLAY-001` | Let `buildopt optimize` automatically use a qualifying portfolio entry and refresh stale entries. | **Done:** the first qualifying run creates the profile; a later exact run validates eleven bindings and selects it with no extra flag or calibration; drift disables it before Gradle starts and valid evidence may repair a corrupt artifact for the next run. |
| 6 | `POC-MAGIC-CI-ONBOARDING-001` | Add one-input GitHub/GitLab orchestration and portable calibration persistence. | **Done:** a clean consumer supplies `command: optimize build`; provider identity makes exact state independent of the checkout path, every other binding remains fail-closed, and both providers publish checksummed results without a service. |
| 7 | `POC-MAGIC-WOW-REPORT-001` | Present graph reduction, observed wall time, uncertainty, break-even, cumulative saving and fallback reasons. | **Done:** human output is understandable without the tracker; JSON recomputes every number, labels cumulative value as a projection, keeps useful lifetime unavailable without profile-specific cross-commit evidence, and never adds unrelated mechanism percentages. |
| 8 | `POC-MAGIC-END-TO-END-VALUE-001` | Validate the complete onboarding on fresh substantial public repositories. | Install-to-decision uses one command and zero manual BuildOpt files; at least two different Gradle families produce equivalent outputs and a net installed-path win, while a negative case retains native. |
| 8a | `POC-MAGIC-CALIBRATION-COST-001` | Reduce generic first-decision cost exposed by the partial public-repository matrix. | Reuse dependency snapshots and base preparation only under exact content bindings; remove at least the 94.1 seconds Beam needs to repay within 30 builds without reducing eight pairs, weakening output/fallback gates or adding repository-name rules. |

Blocks 1–8 are ordered. A later block cannot bypass a missing safety or value
contract from an earlier block. The installed Ktor replay is the immediate
foundation because it proves the current qualified-profile artifact through
the same public distribution that the one-command orchestrator will consume.

## Success scorecard

The final end-to-end block must report, per repository and workflow:

| Metric | Required interpretation |
| --- | --- |
| Manual BuildOpt files | `0` for the supported path. Generated review/evidence files do not count as manual. |
| User commands | One repeated customer command after installation. |
| Time to first decision | Complete elapsed time from command start, including discovery and calibration. |
| Net build saving | Candidate wall time minus optimized-native wall time, including all BuildOpt overhead. |
| Break-even | `ceil(calibration cost / mean saving per matching build)`, reported per profile. |
| Expected useful lifetime | Estimated matching builds before source/Wrapper/graph/options drift, never assumed infinite. |
| Output correctness | Exact bytes by default; only reviewed bounded semantic equivalence may differ. |
| Repeatability and tails | Balanced observations, positive decision bound and non-regressive candidate p95. |
| Product failures | Zero additional failures; fallback failures remain visible. |
| Selection overhead | Measured on every replay and included in net wall time. |
| Native fallback | Proven for global, unknown, ambiguous, unsupported and drifted inputs. |

Activation requires both a measured net win and worthwhile economics. The
payback decision must compare break-even with the profile's evidence-based
expected matching lifetime; a fast profile that is likely to become stale
before repayment remains native. No universal percentage is averaged across
repositories or mechanisms.

## POC boundaries

This roadmap deliberately does not require:

- an eight-hour soak or design partner;
- production SLOs, high availability, RBAC, multi-tenancy or a hosted control
  plane;
- autonomous commits, pull requests or production promotion;
- Test selection, sharding, retries or any other Test Optimization behavior;
- enabling Runtime Tuning, Hot State, standard Copy or another retired
  mechanism; or
- claiming value from Safe Cache when native Gradle cache is already at
  parity.

It does require real public repositories, optimized-native controls, complete
installed-path measurements, equivalent required outputs, immutable evidence
and honest native decisions when the idea does not pay.

## Current end-to-end result and immediate next step

The terminal
[`poc-magic-end-to-end-value-v2`](../../benchmarks/results/poc-magic-end-to-end-value-v2/README.md)
bundle closes this roadmap's install-to-value gate with immutable public
`v0.6.1`. Fresh Ktor and Beam state requires no manual BuildOpt files,
qualifies at 79.82% and 61.65% lower wall time respectively, passes 8/8 pairs,
exact outputs, lower p95 and full fallback, and repays learning in 26 and 28
matching builds. A Ktor root build-logic change completes native Gradle and
retains it without calibration or a timing claim.

The earlier
[`poc-magic-end-to-end-value-v1`](../../benchmarks/results/poc-magic-end-to-end-value-v1/README.md)
matrix remains diagnostic history, and the calibration-cost result remains the
development evidence that made Beam economically eligible. Neither is
substituted for the published terminal capture.

All optional central storage, synchronization, two-machine and equal-opportunity
value blocks are complete. The subsequent Ktor
[profile-lifetime experiment](../../benchmarks/results/poc-profile-lifetime-v1/README.md)
observed one matching replay, one unrelated-owner fallback and one global
invalidation. Although the matching replay saved 112.198 seconds, the fallback
cost 220.761 seconds and the 1,443.324-second calibration did not repay.

Generic economic prequalification is now implemented and measured. It uses
direct graph ownership plus at most 64 first-parent commits and requires at
least eight analogous changes before a new eight-pair learning attempt. On the
public Ktor CORS change it rejects in 192.442 ms, performs no discovery or
calibration and reduces the observed fallback penalty from 220.761 to 13.896
seconds. The original Jetty qualification still needs 31 matching replays and
does not pay back in the observed window.

Automatic breadth transfer is now complete. The unchanged zero-manual-file
command ran on Spring, OpenTelemetry, Kafka, Micronaut and Groovy with one
exact binary. It produced zero product-attributable failures and retained
optimized native Gradle in all five cases. Spring saved 26.83% before learning
cost, OpenTelemetry saved 20.25% with 8/8 positive pairs, Kafka saved 13.42%,
and Groovy saved 2.81%; Micronaut stopped before timing because aggregate
`assemble` exposed 73 candidate entrypoints. None qualified economically:
the four timed candidates needed 103, 101, 190 and 710 matching builds to
repay synchronous calibration, beyond the unchanged 30-build gate.

The POC sequence attacks that observed economic blocker without weakening
correctness or adding repository-specific rules:

1. **Completed:** accumulate exact-bound control/candidate observations during ordinary
   `buildopt optimize` invocations instead of charging 16 extra builds before
   the first decision.
2. **Completed:** materialize verified unaffected outputs through Gradle-compatible cache or
   BuildOpt state before omitting their producers in a clean workspace.
3. **Next:** partition aggregate workflows from generic task, variant, ABI and output
   relationships rather than raising the task cap.
4. **Then:** repeat the same five repositories, revisions and commands with the same
   exact-output, fallback, statistical and 30-build payback gates.

The materialization proof rebuilt one changed JAR, restored two exact
unaffected JARs in a clean workspace and rejected corrupt state before
candidate execution. It closes a correctness gap only; aggregate partitioning
and the fresh five-repository rerun remain necessary for a wall-time claim.

The retained dataset and its interpretation are in
[`poc-automatic-breadth-transfer-v1`](../../benchmarks/results/poc-automatic-breadth-transfer-v1/README.md).
No step may add repository-name rules, average repository percentages, weaken
correctness gates or borrow another profile's lifetime.
