# Build Impact admission review

BO-04 · 14 September 2026 · Decision: **no new trial admitted for the reviewed
Ktor and Apache Beam continuations**.

The selected savings remain valid for the comparisons we ran. They do not yet
justify another Build Impact experiment. In both cases, the measured mechanism
chose a smaller set of ordinary Gradle tasks. Once that module is fixed
manually, we have not identified additional build work for BuildOpt to remove
beyond asking Gradle for those same tasks. Allowing the selection to change
automatically returns to the plan-reuse approach already studied and retired.

This closes the admission question in BO-04. It does not establish that Ktor,
Beam or build optimization in general cannot benefit from another mechanism.
The next step is BO-05 preparation for the Checkstyle comparison, including
its early BO-06 correctness and measurement prerequisites.

## What the two experiments actually compared

Both used the published BuildOpt v0.6.1 package, whose source revision was
`a354e3ee7fffc7abe83b8abe0a125956b159e01a`. BO-04 reviewed that revision, rather
than substituting the current implementation for the measured one.

| | Ktor | Apache Beam |
|---|---|---|
| Original request | `jvmJar` | `classes` |
| Selected request passed to Gradle | `:ktor-server-webjars:jvmJar` | `:examples:java:twitter:classes` |
| Result compared | The WebJars module JAR | Compiled classes of the Twitter example |
| Mean saving against the original request | 30.979 s, 79.82% | 40.123 s, 61.65% |
| Historical paired comparisons | Eight, all faster | Eight, all faster |
| Recorded preparation | 784.031 s | 1,096.653 s |
| Estimated payback | 26 applicable builds | 28 applicable builds |

All measured commands used `--max-workers=12 --build-cache --console=plain
--no-scan`. The [reconstruction](./assessment.json) records full argument lists,
the two original revision pairs and exact required-output paths. These commands
were reconstructed from existing records; BO-04 did not execute them.

The experiment code reset and cleaned both workspaces at the same target
revision and restored their native cache seed before every pair. It therefore
measured repeated selected builds with controlled cache state. It did not
advance these setups through a persistent development history. The original
results record zero later matching replays, so their payback figures remain
estimates.

Source: [original comparisons and qualification records](../../poc-magic-end-to-end-value-v2/README.md),
[measurement loop](https://github.com/tonyredondo/buildopt/blob/a354e3ee7fffc7abe83b8abe0a125956b159e01a/internal/launcher/profile_measurement.go#L430-L525),
[workspace and cache reset](https://github.com/tonyredondo/buildopt/blob/a354e3ee7fffc7abe83b8abe0a125956b159e01a/internal/launcher/profile_measurement.go#L1008-L1038).

## What work disappeared

The task records show a substantial reduction, stable across all eight pairs
in each repository:

| Recorded task outcomes | Ktor, original → selected | Beam, original → selected |
|---|---:|---:|
| Total task records | 1,141 → 121 | 1,475 → 65 |
| Tasks that executed | 301 → 36 | 340 → 29 |
| Tasks restored from cache | 259 → 33 | 362 → 21 |

The original requests already benefited from Gradle's cache. Selecting fewer
tasks also avoided processing and restoring many results outside the selected
module. The records do not separate exactly how much wall time came from
configuration, execution, checking inputs or restoring outputs.

The reported project counts, 133 → 10 and 316 → 6, describe reach in the
selected plan. They do not prove that BuildOpt stopped Gradle from configuring
the other projects. The measured launcher assembled the selected task arguments
and passed them to the ordinary Gradle wrapper; that path did not rewrite the
project's settings to remove projects. The measurement environment removed
BuildOpt feature variables, and the selected path did not request an extra
cache adapter. The recorded mechanism was Build Impact alone.

Sources: [all task counts](./assessment.json),
[task argument selection](https://github.com/tonyredondo/buildopt/blob/a354e3ee7fffc7abe83b8abe0a125956b159e01a/internal/launcher/impact.go#L90-L150),
[Gradle dispatch](https://github.com/tonyredondo/buildopt/blob/a354e3ee7fffc7abe83b8abe0a125956b159e01a/internal/launcher/run.go#L178-L243),
[measurement environment](https://github.com/tonyredondo/buildopt/blob/a354e3ee7fffc7abe83b8abe0a125956b159e01a/internal/launcher/profile_measurement.go#L1344-L1358).

## Why a fixed module does not yet justify another trial

Gradle already accepts a specific subproject task, including its dependencies.
For a workflow that only needs WebJars or the Twitter example, the relevant
native comparison is the corresponding selected command above. This is
documented [Gradle command-line behavior](https://docs.gradle.org/current/userguide/command_line_interface.html#sec:executing_tasks_in_multi_project_builds).

The original experiments did not time that direct native command against
BuildOpt. BO-04 therefore makes no claim that their durations are identical,
or that BuildOpt has a measured zero saving against it. The narrower conclusion
comes from the implementation: the selector has not identified additional work
to omit once the same task scope is fixed on both sides. A longer replay would
not create that missing intervention.

If the workflow needs all results of the original broad command, the other
problem remains: equality of one JAR or one example's classes does not prove
that every required artifact, diagnostic and failure behavior was preserved.
Those wider guarantees were not part of the selected output checks. We cannot
silently narrow the requested result to carry the old percentage forward.

Helping someone choose a better Gradle command may still be useful. It is a
different claim from maintaining additional build-time savings over a correctly
scoped native command. This review does not assign the old percentages to that
unmeasured claim.

## What the later studies already answered

| Study | What it establishes | Consequence for a follow-up |
|---|---|---|
| Ktor Jetty lifetime | One matching replay saved time, but two fallbacks and learning cost left the window 1,551.814 s behind. | A selected win needs enough reuse and cheap rejection. This is not a lifetime result for WebJars. |
| Five-repository functional coverage | Only one of five repositories was net positive; a plan was selected on one of six eligible later builds. The route ended at `STOP_GENERIC_POC`. | Another repository or a longer sequence alone does not address selection failure. |
| Structural rebinding | Compatible source/checkout changes could retain a structural identity; workflow, dependency and output changes still rejected. This was correctness evidence, without a performance replay. | Removing commit IDs from plan identity has already been explored; it is not a new explanation for the failures. |
| Later Kafka/Groovy chronology | Kafka's qualified plan applied on none of three immediate descendants. Both measured repositories lost time after recorded costs; Spring was not run. | Qualification and portability do not establish lifetime value. The unrun Spring case remains unmeasured. |
| Installed profiles and adaptive fragments | 100 exact-output comparisons across five repositories applied no profile or fragment. Total delta was −368.623 s; 179.029 s was recorded BuildOpt work, with the rest a Gradle/runner residual. | Smaller reusable pieces and prior-only chronological learning have also been tested. The entire loss must not be attributed to BuildOpt overhead. |

Sources: [Jetty lifetime](../../poc-profile-lifetime-v1/README.md),
[functional-coverage decision](../../poc-functional-coverage-decision-v1/README.md),
[structural rebinding](../../poc-structural-profile-rebinding-v1/README.md),
[Kafka and Groovy descendants](../../three-class-chronological-value-v1/README.md),
[100-comparison attribution](../../current-longitudinal-attribution-v1.json),
[broader mechanism audit](../../../../docs/findings/buildopt-generalization-audit.md#why-graph-reduction-alone-is-insufficient).

## Admission decisions

| Proposed continuation | Decision and reason |
|---|---|
| Fix the module manually and replay it | Not admitted. The selected native command is already expressible directly; no additional avoidable work has been identified for this selector. |
| Keep choosing modules automatically as commits arrive | Remains retired. This repeats changed-project selection and saved-plan lifetime without resolving their observed limitations. |
| Omit projects while preserving every output of the broad command | Not admitted from these wins. It needs wider output guarantees; earlier producer/output and fragment work did not establish recurring value. No new mechanism is demonstrated here. |
| Avoid unnecessary configuration while keeping the same native task request | Potentially a different mechanism, but not admitted. The retained timings and project counts do not identify an avoidable configuration cost or a safe correction. It would need a new observed cause under the existing rules. |

None supplies both a distinct intervention and evidence of a material
whole-build opportunity. There is therefore no new timing protocol or allocation
from BO-04. This is an admission decision, not a new negative performance
measurement. The Checkstyle opportunity in BO-03 is unaffected.

## Verification and limits

BO-04 reconstructs the original commands, cache/reset policy, task counts and
output scope from retained records and the measured release source. All sixteen
selected timing pairs remain in the evidence. The original subject checkouts,
generated profiles, complete build logs and rebuilt artifacts have not been
recovered; current native buildability and phase-by-phase timing remain unknown.
Those gaps prevent new performance claims. They do not supply a distinct
intervention for the rejected follow-ups.

The existing checkers reproduce the selected wins, Jetty lifetime and terminal
functional-coverage decision. The [input bindings](./inputs.json) identify
thirteen retained data files and five source files at the measured revision.
The [verification record](./verification.json) includes the checks and limits.
No target build, protected source read or product implementation was performed.

To check the reconstruction, with the measured source revision available in Git:

```bash
python3 benchmarks/results/buildopt-product-viability-v1/bo-04-build-impact-review/analyze.py --check
```

Proceed to **BO-05 and its early BO-06 prerequisites** for the short native
Gradle versus supported Checkstyle comparison. Keep the original acceptance
criteria, all inactive and unfavorable builds, and the protected validation
history. BO-04 does not start that comparison.
