# Generic workflow breadth fixture

This two-project Gradle build exercises the same repository-owned BuildOpt
input across four build-owned workflow families:

- reproducible JAR packaging;
- a task implementing Gradle's public `VerificationTask` contract;
- reproducible application distribution archives;
- test compilation and preparation without executing tests.

`prepareReleaseNotes` is intentionally an arbitrary task with actions. Its
output can be confirmed, but its execution relationship is not covered by a
public Gradle task contract. BuildOpt must therefore retain the native full
workflow before any timing or activation.

The fixture contains no repository-name selection rule and no performance
workload. `dev/check-generic-workflow-breadth` copies it into a temporary Git
repository, creates the owner inputs through the installed CLI, checks the
proposals, and compares every supported candidate output byte for byte with
the original workflow.
