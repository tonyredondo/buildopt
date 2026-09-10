# Why the saved build-plan approach stopped

Build Impact can save a reduced plan for building only the parts needed by a
request. This study asked whether those plans helped often enough across five
repositories to justify using the approach more broadly. Only one repository
saved time after preparation was counted, and a plan was reused on only one
of six later builds where it might have applied. The approach did not meet
the requirements set before the experiment.

## What worked

All five repositories were observed, and all 27 requested-build observations
produced the required output. There were no extra builds run solely to collect
measurements and no failures attributed to BuildOpt. Plan selection used the
same rules across repositories.

Kafka provided the positive result. Its selected build was faster in all eight
comparisons, with a statistical interval that supported a saving and an
improvement in slower builds. It recovered its preparation time on the second
matching build and ended 82.527 seconds ahead after the recorded costs.

## What failed

| Question | Requirement fixed before measurement | Observed result |
| --- | --- | --- |
| Did enough repositories save time after costs? | At least three of five. | One of five. |
| Was a saved plan useful on later eligible builds? | At least half of those builds. | One of six, or 16.67%. |
| Was declining an optimization cheap enough? | Before Gradle started, median delay below 500 ms and p95 below 1,000 ms. | The only eligible observation took 4,098 ms. |

The last row contains one observation, so it does not describe a distribution
of delays. The reported median and p95 are both that same value. Decisions
made after Gradle had already started were outside this particular check.

Five of the eight criteria passed, but these three failures prevented a claim
that saved plans delivered general value. The recorded decision is
`STOP_GENERIC_POC`: stop developing this approach to selecting and reusing
plans. The selected Kafka and Spring savings, output checks and fallback code
remain useful evidence. Keeping that code does not reopen the stopped research.

The [decision data](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/poc-functional-coverage-decision-v1/summary.json)
and [original experiment rules](https://github.com/tonyredondo/buildopt/blob/main/specs/poc-functional-coverage-decision-v1.md)
provide the details. To recompute the decision from the repository root:

```bash
./dev/check-functional-coverage-decision
```
