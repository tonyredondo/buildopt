# Checkstyle comparator qualification

15 September 2026 · BO-06 prerequisite complete; BO-06 remains partial

The current output comparator now has the qualification missing from the
previous replay attempt. It checks that the optimization preserves the files
and diagnostics produced by Elasticsearch's build. The runner, candidate
correction and comparison rules are unchanged.

This removes the known admission blocker. It does not establish a time saving:
no Elasticsearch build ran in this block. The next measurement still needs a
fresh control followed by the complete development sequence.

| Check | Result |
| --- | --- |
| Comparator behavior | All 33 tests passed, with no skips. They cover matching outputs, changed or missing diagnostics, invalid files, configuration changes and incorrect reuse of earlier results. |
| Retained Elasticsearch outputs | All six comparisons matched. These include earlier native builds, native versus corrected builds, and the identical-code control. Valid reuse from the same side was preserved. |
| Reader bindings | The Go check passed. A changed compiler-state reader cannot borrow the existing qualification. |
| Complete admission path | All 20 cases passed using the actual replay executable and real Elasticsearch inputs. Three valid configurations were accepted; 17 missing or invalid prerequisite cases were rejected. |
| Proposed live inputs | The control configuration is admitted. The development configuration correctly waits for a genuine passing control under the new identity. Both use the same frozen implementation and output policy. |

The positive development and confirmation admission tests use **synthetic
trial receipts**, confined to the test directory. They exercise the whole
validation path without running builds. They do not count as passing control,
development or confirmation experiments. The proposed live configurations
contain none of those synthetic receipts.

## What changed

The previous comparator qualification covered an older version. The current
version also recognizes a control where both sides contain the Checkstyle
correction. The new tests check that this allowance cannot hide an unexpected
file, a changed configuration or a different task producing the file.

Qualification was added in a new policy file. The measured policy and earlier
results remain intact. Because the runner includes the whole policy binding
in its measurement identity, the old control cannot qualify the new policy.
The admission tests verify that rejection. No compatibility exception was added.

The confirmation tests also checked preparation accounting: each repetition
must include evidence for the time spent applying the correction, even when
that measured time is zero. This remains a requirement for the later experiment.

## Next measurement

The [next measurement plan](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-comparator-qualification/next-measurement.md)
keeps the candidate and savings criteria unchanged. It proposes eight control
builds and, only after that control passes, 42 builds covering the anchor and
all 20 development changes. No protected validation changes are included.

The qualification used eight comparator JVMs, one focused Go test command and
no owner or fixture builds. All invoked processes returned. The allocation
closed within its limits. The earlier failed or incomplete attempts retain
their recorded outcomes and counts.

## Evidence

Run the portable audit from the repository root:

```sh
python3 -B benchmarks/results/buildopt-product-viability-v1/bo-06-comparator-qualification/verify-evidence.py
```

It checks 166 exported files, source archives, certificate bindings and the
reported test results. The archive includes the small comparator fixtures and
the exact test source. Repeating the six full comparisons requires the retained
Elasticsearch outputs and pinned runtimes at the original local paths; this
portable audit does not rerun them.

The [evidence index](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-comparator-qualification/evidence-manifest.json),
[comparator results](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-comparator-qualification/receipts/comparator-matrix.json),
[admission results](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-comparator-qualification/analysis/admission-matrix.json)
and [frozen inputs](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-comparator-qualification/analysis/measurement-freeze.json)
provide the detailed records.
