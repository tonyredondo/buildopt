# Cross-commit breadth replication

This bundle tests whether the positive Kafka cross-commit replay transfers
unchanged to two additional public repository families. It does not. The
terminal decision is `DO_NOT_BROADEN_CROSS_COMMIT_VALUE_CLAIM`.

| Subject | Frozen result | Attributable lifetime value |
| --- | --- | ---: |
| Spring root `classes` | 27→21 projects; candidate 22.883 s versus 22.677 s control; 1/8 positive pairs | None; calibration rejected at **-206 ms / -0.91%** |
| OpenTelemetry JMX `testClasses` family | Ordinary workflow succeeds, but one root `CHANGELOG.md` path has no Gradle project owner | None; discovery fails closed before calibration |
| Spring JMS root `classes` | 27→10 projects; candidate 20.064 s versus 22.654 s control; 8/8 positive pairs | None; **+2.590 s / 11.43%** calibration is rejected before replay because 14 native class outputs differ across roots |

The Spring JMS native mismatch contains 13 AspectJ classes whose embedded
source path changes with the isolated checkout and one Java class whose
compiler-generated local-variable name changes. `javap -p -c -s` shows matching
code and signatures, but that diagnostic does not grant semantic-equivalence
authority. BuildOpt therefore transports none of those outputs and claims no
timing value.

The original Spring preregistration remains unchanged even though the public
target also touched `:framework-docs`. The mismatch was discovered before
timing and is retained in the raw result rather than retroactively rewriting
the contract. Spring JMS is a separately committed addendum selected by graph
topology before its timings were observed.

## Interpretation

This is not a product failure: all customer Gradle invocations succeed and the
uncertain subjects retain native execution. It is a failed breadth claim. The
current positive cross-commit value remains bounded to the checked Kafka
workflow.

The evidence identifies three generic next hypotheses:

1. derive mixed-path relevance from observed Gradle workflow inputs so an
   unconsumed root document does not obscure owned source changes;
2. quarantine native-volatile producers from transported packs, rebuild them
   locally and keep exact-byte checks for every output BuildOpt reuses; and
3. preregister descendant windows containing a profile refresh followed by a
   structurally compatible omitted-owner change.

None of these may use repository names, filename-extension allowlists or
post-timing window selection.

## Validate

```bash
./dev/check-cross-commit-breadth-replication
```

The checker reconstructs `summary.json` from all three raw results, validates
the frozen negative classifications and rejects a broadened claim. Normal CI
does not rerun the long public builds.

## Boundaries

- 12-CPU Linux development-host evidence with a common 12-worker cap;
- public first-parent commits and fresh Gradle processes;
- zero production authorization and zero repository-specific product rules;
- no repository percentages averaged or mechanism percentages added;
- no soak or design-partner requirement; and
- Test Optimization remains out of scope.

Raw captures are retained beside the summary. The positive Kafka recovery
evidence remains separate and unchanged.
