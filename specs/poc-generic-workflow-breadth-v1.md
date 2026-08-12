# Generic owner workflow breadth

## Question

Does the confirmed `.buildopt/profile.json` flow remain unchanged across
different Gradle-owned workflow families, and does it retain the original
workflow before timing when Gradle cannot provide safe task semantics?

This is a capability proof for the BuildOpt POC. It does not claim that any
candidate is faster, qualify a profile, or authorize automatic activation.

## Method

The deterministic two-project fixture changes one production source in
`service-a`. For each supported workflow, the installed CLI must:

1. execute the repository's original Gradle selector;
2. validate the owner-declared output;
3. require explicit confirmation to create `.buildopt/profile.json`;
4. derive the exact base-to-HEAD Git change;
5. propose only the corresponding `:service-a` task;
6. omit `service-b`;
7. execute the candidate after deleting all outputs; and
8. reproduce the owner-declared output byte for byte without executing a
   Gradle `Test` task.

The matrix uses public Gradle semantics rather than task-name rules:

| Family | Original selector | Candidate | Required output |
|---|---|---|---|
| Packaging | `jar` | `:service-a:jar` | `service-a/build/libs/**` |
| Verification | `verifyMetadata` implementing `VerificationTask` | `:service-a:verifyMetadata` | `service-a/build/reports/verification/**` |
| Distribution | `distZip` as an `AbstractArchiveTask` | `:service-a:distZip` | `service-a/build/distributions/**` |
| Test preparation | `testClasses` | `:service-a:testClasses` | `service-a/build/classes/java/test/**` |

`prepareReleaseNotes` deliberately has executable actions without a supported
public task contract. Its output contract must still validate, but proposal
generation must return `NATIVE_FULL_GRAPH / ORIGINAL_WORKFLOW_UNSUPPORTED`,
write no structural graph documents, and leave the native output present.

## Acceptance

```bash
./dev/check-generic-workflow-breadth /tmp/workflow-breadth.json
./dev/check-generic-workflow-breadth-result /tmp/workflow-breadth.json
```

The evidence is accepted only when all four supported cells return
`MEASURE_STRUCTURAL_CANDIDATE / COMPLETE_STRUCTURAL_REDUCTION`, omit exactly
`:service-b`, preserve their declared output bytes, and execute no tests. The
unsupported cell must retain the native workflow before timing.

The same checker runs in a read-only hosted GitHub fixture. The fixture, runner
and specification SHA-256 values are bound into the result.

## Boundaries

- All owner files remain review-required POC inputs.
- No timing or performance percentage is created.
- No profile is qualified or activated.
- No repository-name product rule is introduced.
- Test execution, selection, retry, sharding and prioritization remain outside
  Build Optimization.
- No production, soak or design-partner requirement is introduced.
