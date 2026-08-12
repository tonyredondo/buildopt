# BuildOpt Generalization Audit

## Audit question

Can the retained BuildOpt POC mechanisms be applied to an arbitrary Gradle
repository without repository-name rules, while preserving native Gradle as
the safe and performance fallback?

The answer is **yes for the generic structural evaluation path, with explicit
repository-owned semantic inputs and review; not yet as zero-input automatic
activation**. This distinction is central. Generic implementation means that
BuildOpt uses the same discovery, measurement and decision algorithms for
every repository. It does not mean that BuildOpt can invent which command and
outputs represent success for an unknown customer build.

## What is generalized today

| Layer | Generic behavior | Repository-owned input | Evidence and boundary |
| --- | --- | --- | --- |
| Installation and launcher | Native packages and `buildopt gradle` locate the Wrapper, preserve argv/process behavior and support Linux, macOS and Windows. | The repository's Wrapper and requested Gradle command. | External Kotlin and Groovy pilots pass onboarding; platform CI covers native lifecycle. |
| Safe Cache / L1 | Cache scope is derived from repository, Wrapper and platform; unsafe or unavailable state falls back to native execution. | None beyond the repository and Wrapper already being executed. | At parity with a warm native Gradle cache, so it is a safety/onboarding feature rather than a retained accelerator claim. |
| Structural proposal | Typed Gradle discovery maps the original entrypoints and exact changed paths to project owners, constructs a smaller candidate and rejects incomplete or ambiguous graphs. | Repository identity, original entrypoints, exact Git change and required output globs. | The same code reproduced five clean-CI proposals and discovered the unseen Hibernate 29-to-1 candidate. No public-repository names appear in the customer execution decision. |
| Structural measurement | Independent source trees, Gradle homes and cache seeds compare optimized native Gradle with the candidate, bind source/tool/profile hashes, compare output bytes and prove full-graph fallback. | Accepted proposal and its declared output contract. | Spring, OpenTelemetry, Kafka, Micronaut, Groovy and Hibernate use the same installed measure/evaluate path. |
| Structural decision | Fixed minimum saving, reduction, uncertainty, repeatability, output, failure and fallback gates determine `REVIEW_STRUCTURAL_PROFILE` or native fallback. | Explicit human review remains required. | Hibernate version 5 qualifies at 5.88%, 8/8 reciprocal blocks; Spring remains native at 7/8 despite a positive mean. |
| Reviewed task optimization | Exact task adapters and Patch Autopilot can repair one understood cacheability contract with signed, reversible evidence. | A reviewed task type/recipe and its exact validation boundary. | Strong Kotlin/Groovy custom-task evidence exists, but this is not generalized to arbitrary task implementations. |
| Shared / Edge Cache | Implements Gradle remote-cache protocol, authenticated commit authority, locality and safe miss/failure behavior. | Operator endpoint, credentials and a workload/network profile. | Locality has bounded synthetic and Kafka evidence; it is not part of the uniform structural claim. |
| Evidence and CI | Review-only Action emits proposals from checked-in owner inputs and can replay them on clean runners without activating a profile. | A versioned owner input file. | Five-of-five clean-CI replay matched with zero graph drift and zero active profiles. |

Runtime Tuning, Hot State and the standard `Copy` adapter are intentionally not
generalized because end-to-end evidence was neutral, unstable or regressive.
The standard `Jar` adapter is retained only inside its exact qualified
OpenTelemetry composition. Generic availability is not the same as generic
value, so unqualified mechanisms remain disabled.

## Supported applicability

Any Gradle repository can run proposal discovery and receive either a
reviewable candidate or an explicit native verdict. That is different from
saying every Gradle workflow is currently optimizable. The retained structural
path supports conventional lifecycle entrypoints whose complete project and
task relationships can be discovered. It also supports multiple declared
entrypoints while preserving all of them as fallback.

The current POC deliberately rejects custom executable workflows, Gradle
`Test` execution, external included builds, incomplete or unknown
relationships, global/build-logic changes, ambiguous change ownership, no
graph reduction and output contracts that cannot be verified. These are safe
applicability boundaries, not repository allowlists. A future mechanism may
broaden a boundary only after independent correctness and wall-time evidence.

## Why the repository must own three inputs

BuildOpt can discover Gradle structure, but only the repository owner can
authoritatively define:

1. **the workflow**: for example `assemble`, `check` or a distribution task;
2. **the change**: exact paths between two immutable Git revisions; and
3. **the required outputs**: the artifacts whose equality makes the candidate
   semantically equivalent to the original workflow.

These are not repository-specific branches in BuildOpt. They are the contract
against which a generic optimizer can prove correctness. Removing them would
make the POC easier to run only by making it capable of reporting a faster but
wrong build.

The Hibernate holdout demonstrated the remaining usability problem: its build
places JARs under `target/libs`, not Gradle's conventional `build/libs`. The
first attempt safely stopped before timing, but only after the user supplied a
wrong output glob. BuildOpt therefore still needs a generic output-contract
preflight that discovers candidate output directories, asks the owner to
confirm them and rejects missing, empty, overlapping or ambiguous ownership
before expensive measurement.

## Cross-repository evidence

The uniform structural-only method now has six public-repository results. The
percentages are not averaged because the workloads are different.

| Repository | Full -> selected projects | Direct wall-time result | Decision |
| --- | ---: | ---: | --- |
| Spring Framework | 27 -> 10 | 17.94% faster, 7/8 positive raw pairs | Native fallback under the frozen repeatability gate. |
| OpenTelemetry Java Instrumentation | 1,024 -> 34 | 14.43% faster, 8/8 | Reviewable candidate. |
| Apache Kafka | 64 -> 3 | 84.11% faster, 8/8 | Reviewable candidate. |
| Micronaut Core | 75 -> 22 | 41.74% faster, 8/8 | Reviewable candidate. |
| Apache Groovy | 37 -> 2 | 73.85% faster, 8/8 | Reviewable candidate. |
| Hibernate ORM holdout | 29 -> 1 | **5.88% faster, 8/8 reciprocal blocks** | Reviewable candidate after preregistered order correction. |

All accepted observations include BuildOpt overhead, preserve the declared
outputs byte for byte and exercise native full-graph fallback. Hibernate is
particularly important: it was selected after the method was frozen, failed
the first timing protocol, was investigated rather than discarded, and then
qualified only after a generic measurement correction removed execution-order
bias.

## What is implementation-generic versus evidence-specific

Public repository names remain in fixtures, runners and immutable result
files because evidence must identify what was tested. They do not appear in
the structural candidate-selection or qualification branches used by the
installed CLI. The product code reasons about:

- typed projects, included builds, dependencies and entrypoints;
- exact change ownership and global-change rules;
- complete versus unknown relationships;
- task categories and unsupported `Test` execution;
- required output patterns and byte manifests;
- repository, revision, Wrapper, graph, manifest and executable digests; and
- measured wall time, uncertainty, repeatability, failures and fallback.

Experiment scripts may prepare a fixed public revision or mutation. That is
test data, not product specialization. A new repository can use the same
`profile propose -> measure -> evaluate` pipeline without adding its name to
BuildOpt, provided its workflow is discoverable and its output contract is
declared.

## Remaining gaps before calling the POC broadly usable

1. **Output-contract preflight.** Discover and validate real Gradle outputs
   before warm-up or timing, then produce a review artifact rather than
   guessing conventions.
2. **Owner-input ergonomics.** Reduce the three semantic inputs to one small,
   documented, versioned file and actionable diagnostics; do not replace them
   with hidden inference.
3. **Workflow breadth.** Repeat the unchanged path on more packaging,
   verification, distribution and build-owned test-preparation workflows.
   Gradle `Test` optimization remains separate.
4. **Installed replay of qualified profiles.** Prove that a reviewed profile
   selected by `evaluate` produces the same value through the public package,
   not only the measurement harness, for more than the existing bounded cases.
5. **Generic task-contract research.** Add an adapter or patch recipe only when
   an exact task contract and end-to-end wall-time win transfer across
   repositories; never infer value from cacheability alone.
6. **Portfolio measurement.** When more than one mechanism qualifies for the
   same workload, measure the complete installed composition directly. Do not
   add isolated percentages.
7. **Native measurement parity.** Installation, launcher and service lifecycle
   are validated on Linux, macOS and Windows, but the current comparable
   structural wall-time matrix is Linux evidence. Run the same fail-closed
   qualification protocol natively before making macOS or Windows performance
   claims.

## POC conclusion

The retained idea is not “a faster Gradle cache.” It is a generic,
evidence-gated layer that can request less Gradle work for an exact change and
output contract, then decline the optimization when correctness or value does
not replicate. That idea now transfers across six materially different public
repositories and beats optimized native Gradle in five under their frozen
decision rules; Spring demonstrates that native fallback still governs a
positive but insufficiently repeatable result.

The next block should implement the output-contract preflight. It closes the
only generic onboarding failure observed by the blind holdout and makes the
current mechanism easier to evaluate on another arbitrary Gradle repository
without weakening correctness or pretending the POC is production-ready.
