# Whole-build opportunity assessment

BO-03 · 14 September 2026 · Assessment complete; sustained value remains unproved.

The next useful comparison is still the short Elasticsearch Checkstyle screen.
It has measured opportunities in the development history, but little room for
overhead in the existing model. Micronaut needs evidence that its cache can
avoid work repeatedly in a real build workflow. The selected Spring result
does not justify more timing under the current criteria. Ktor and Apache Beam
remain candidates for the separate BO-04 review of Build Impact.

This assessment used existing timing records and inspected only the permitted
development source history. **No new Gradle build ran, no protected validation source
or timing was read, and no long replay was admitted.**

## Decisions by candidate

| Candidate | Work it can avoid | Current decision |
|---|---|---|
| Elasticsearch Checkstyle | Rechecking unchanged Java files when the source-code check needs to run. | Eligible for the already planned short comparison, after its correctness and measurement prerequisites. No sustained-saving claim. |
| Micronaut Python compilation | Running a task again when Gradle can restore its complete previous result. | Defer timing until the complete command and a recurring reason to restore cached outputs are established. |
| Spring architecture check | Repeating checks on compiled classes when their previous result can be restored. | Do not time this selected case again on the current evidence. Its recorded mean gain falls below today's one-second floor before costs. |
| Ktor Build Impact | Preparing and executing parts of the broad build beyond the selected WebJars module. | BO-04 review only. Required results, benefit beyond a direct module command, and lifetime remain unresolved. |
| Apache Beam Build Impact | Preparing and executing parts beyond the selected Twitter example compilation. | BO-04 review only, with the same unresolved questions assessed separately for Beam. |

An unknown frequency is not a measured zero. An eight-pair gain is not a
whole-history ceiling. These decisions preserve that distinction and do not
reopen retired searches for general caches or reusable build plans.

## Elasticsearch: an opportunity with little margin

Checkstyle checks Java source against coding rules. The supported correction
reuses checks for unchanged files while preserving the complete reports.
The relevant command is `:server:precommit`, which also runs other checks.
Its baseline already includes ordinary Gradle caching and the earlier
ForbiddenPatterns correction. Configuration Cache remains disabled because
this workflow failed its existing compatibility check.

Across all 20 development changes, Checkstyle performed checking work on only
three: 7, 19 and 20. Gradle avoided that work on the other 17. Those builds
remain in the denominator.

The retained dependency model asks what would happen if all Checkstyle actions
disappeared while every other task duration stayed fixed. Its estimated saving
is zero at change 7, 24.220 seconds at 19 and 5.629 seconds at 20. Other work
still determines the end of many builds. Adding the overlapping Checkstyle
task durations would greatly overstate the saving.

Using the **complete command duration**, the result is:

| Quantity over the 20 changes | Result |
|---|---:|
| Native command time | 516.979 s |
| Saving in the zero-action dependency model | 29.849 s |
| Average modeled saving per scheduled change | 1.492 s |
| Modeled reduction in complete-command time | 5.774% |
| Margin above both savings floors | 4.000 s total, or 0.200 s per build |

The earlier 5.945% calculation used the shorter internal Gradle build span,
502.069 seconds. BO-03 keeps that result and adds the complete-command
denominator; it does not replace the original observation.

The 0.200 seconds is **model headroom**, not an overhead measurement or an
allowance for a future replay. Changed files still need checking, full reports
need producing, and state needs hashing and saving. Application, validation
and maintenance also cost time. These may consume the margin. Conversely,
removing checking work can change competition for CPU and other resources,
which this fixed-duration model does not predict. It is not an absolute upper
bound on an actual comparison.

The previous implementation saved time on selected changes but lost 19.16%
over a separate 13-change development sequence. Supported V2 subsequently
passed its scoped correctness checks. BO-02 compared V2 with itself and found
no material difference under that control's rule; it did not compare V2 with
native Gradle. Neither result establishes recurring V2 savings.

There is enough evidence for the planned native-versus-V2 screen through
changes 17–20, including two changes where native Checkstyle did no checking.
There is not enough evidence for the protected 80-change replay. Preserve all
rows, the fixed CPU profile and applicable readiness requirements before
registering that screen. Required adoption and maintenance time remains
unmeasured for a usable delivery path.

Sources: [native task opportunities](../bv002/opportunity-analysis.json),
[dependency model and all 20 rows](../checkstyle-admission/residual-opportunity.json),
[earlier development comparisons](../bv006-screen-completion/README.md),
[supported V2 result](../bv006-checkstyle-finalization/README.md),
[identical-code control](../bo-02-measurement-control/README.md).

## Micronaut: establish when a cache restore would help

The patch makes the Python bytecode task cacheable and removes unnecessary
dependence on its checkout path. The task prepares generated Python resources;
the build script connects it to `processResources`. It reads Python resources
and a compiler classpath, including project dependencies.

The eight selected comparisons saved an average of 6.921 seconds. They also
record matching outputs and a successful cache restore in another checkout.
The portfolio does not retain a complete original launch command and state
reset policy sufficient to reconstruct those timings as a daily workflow.

In a persistent workspace, unchanged inputs and retained outputs can already
let Gradle skip the task. New inputs need a matching stored result before this
patch can help. A useful restoration therefore needs a reason, such as outputs
being absent while a valid cached result exists. A source edit alone does not
establish that condition. We must not clear outputs just to manufacture it.

The original task preimage matches all 21 permitted source snapshots, including
the anchor. Our selected source paths changed at one development transition,
11, in the Python tooling project. That is a source observation, not a cache
miss count: resolved classpaths, generated inputs and actual scheduling have
not been measured. Useful restore frequency and the effect on complete build
time remain unknown.

The recorded preparation charge was 600 seconds. Even if all 80 validation
builds saved the historical 6.921 seconds, their 553.690 seconds of gross saving
would not recover that charge. This is a sensitivity using the old campaign,
not a measurement of future installation cost. At that same saving on every
build, required preparation and maintenance together would need to stay at or
below 473.690 seconds just to leave one second net per validation build; a
longer complete command could impose a stricter 5% requirement.

Keep the candidate, but defer timing. Before a short comparison, establish the
complete requested command, normal workspace/cache policy, observed restoring
opportunities and the charged delivery work. Source compatibility alone is
insufficient.

Sources: [original paired results and costs](../../reviewed-native-patch-portfolio-v1/result.json),
[recovered patch](../bo-01-candidate-recovery/inventory.json),
[development source observations](./development-source-review.json).

## Spring: the selected result does not clear today's floor

This patch makes an architecture-check task cacheable. It checks compiled
classes against dependency rules; its plugin attaches the task to Gradle's
`check` command. It restores a whole result, rather than checking only changed
classes. The same distinction between an unchanged task and a useful cache
restore applies here.

The eight selected pairs saved an average of 0.9855 seconds. Even if that gain
occurred on every build, with free setup and no additional work, it would miss
the current one-second net floor. The earlier portfolio used a 0.5-second
floor, which explains its positive historical decision. That decision remains
intact; it is not acceptance under the new plan.

The 360-second recorded preparation charge would require 366 applicable builds
at the observed mean gain. The task preimage still matches all 21 development
snapshots. Selected Spring source/build-logic paths change in eight of the 20
transitions, but task execution and useful cache restore frequency are unknown.

This is a reason not to spend another timing allocation on this selected case.
It is not proof that every Spring workflow has a subsecond ceiling: the
observed mean is not an absolute maximum, and the full `check` command's
limiting work has not been measured. Reconsideration would need a separately
observed, larger whole-workflow opportunity under the admission rules, not a
reinterpretation of these eight pairs.

Sources: [original pairs and preparation charges](../../reviewed-native-patch-portfolio-v1/result.json),
[source observations](./development-source-review.json).

## Ktor and Beam: large selected savings, unresolved output scope

Ktor saved 30.979 seconds per selected comparison and Beam saved 40.123 seconds.
All eight pairs in each repository produced the compared required outputs.
Their recorded preparation charges were 784.031 and 1,096.653 seconds, with
estimated payback after 26 and 28 applicable builds respectively.

Those checks covered the Ktor WebJars JAR and Beam Twitter example classes.
They did not establish equality of every output of the original broad
`jvmJar` or `classes` command. The reduced entrypoints were
`:ktor-server-webjars:jvmJar` and `:examples:java:twitter:classes`. BO-04 must
establish what the workflow actually requires and what BuildOpt adds beyond
asking ordinary Gradle for that module directly.

Neither exact setup has a measured development lifetime or useful-plan
frequency. A separate Ktor Jetty study lost 1,551.814 seconds after learning
cost: a successful reuse was outweighed by the work following rejection and
the original preparation. That is relevant evidence about failure conditions,
but it is not a lifetime measurement of the WebJars case.

The gains are large enough to justify the **BO-04 evidence review**. They do not
yet justify timing. BO-04 must identify an unanswered question that can survive
repository changes without per-commit manual repair or weaker output checks.
If it cannot, reject the follow-up and retain the selected wins as historical
results. Generic plan reuse stays closed.

Sources: [Ktor and Beam comparisons](../../poc-magic-end-to-end-value-v2/README.md),
[negative Jetty lifetime](../../poc-profile-lifetime-v1/README.md),
[BO-04 admission requirements](../../../../docs/plans/buildopt-research-execution-plan-2026-09-14.md#bo-04-decide-what-remains-unanswered-in-build-impact).

## Accounting, verification and next step

The replay contract counts 80 validation builds. Development gains earn no
confirmation credit, and required discovery, verification, application and
maintenance work must be charged once. Historical campaign charges are retained
as a sensitivity; they are not silently set to zero or claimed to be measured
future adoption cost. Unknown values remain `null` in the assessment.

[Calculated results](./assessment.json) include both a zero-cost illustration
and the historical-charge illustration for each old paired experiment. Neither
is a forecast: they assume the selected mean durations continue, no cost on
inactive builds and no maintenance. Percentages are never pooled across cases.
These illustrations do not replace the complete replay, uncertainty,
correctness or slow-build criteria.

[Input hashes](./inputs.json) bind the retained records. The
[verification record](./verification.json) covers their integrity, arithmetic,
missing-row rejection and the current savings floors. The analysis preserves
all 20 native development rows and all 32 historical timing pairs across the
four selected Micronaut, Spring, Ktor and Beam experiments. Source inspection
verified only the Micronaut/Spring development snapshots, ordinals 0–20; it
did not build them or inspect protected source.

Reproduce the assessment without starting builds:

```bash
python3 benchmarks/results/buildopt-product-viability-v1/bo-03-opportunity-assessment/analyze.py --check
```

The next plan step is **BO-04**. It will decide whether the Ktor and Beam results
support a new, bounded Build Impact question. Checkstyle remains eligible for
BO-05 preparation after its applicable checks; the protected replay stays closed.
