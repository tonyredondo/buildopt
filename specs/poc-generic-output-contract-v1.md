# Generic Gradle output-contract preflight

## Purpose

`POC-GENERIC-OUTPUT-CONTRACT-001` closes the adoption failure exposed by the
blind Hibernate holdout. A repository owner could previously supply a
syntactically valid output glob that did not describe the repository's real
Gradle outputs. The measurement collector rejected the empty output set, but
only after proposal generation and dependency preparation had begun.

The preflight executes the owner-declared Gradle workflow once before
structural discovery, records the repository-contained output roots declared
by the tasks in that exact graph, and checks the declared globs against
non-empty regular files. Each matched file must have one most-specific Gradle
project owner. The result is review evidence; it never activates a profile.

## Decisions

The v1 artifact has three review outcomes:

- `VALIDATED_REQUIRED_OUTPUTS`: every declared glob matched at least one
  regular file and every matched file had unambiguous project ownership;
- `REVIEW_REQUIRED_OUTPUTS`: Gradle exposed non-empty candidates, but the owner
  supplied no declaration to validate; and
- `NATIVE_FULL_GRAPH`: the workflow produced no repository outputs, a declared
  glob was empty, or ownership was ambiguous.

An invalid Gradle invocation, malformed snapshot, tracked-file mutation, or
revision change remains a command error rather than a successful review
decision. Files outside the repository, symlinks and empty roots are never
suggested as required outputs.

`profile propose` embeds this preflight and writes
`buildopt-output-contract.json`. It cannot write candidate graph documents or
hand off to `profile measure` unless the decision is
`VALIDATED_REQUIRED_OUTPUTS`. `profile outputs` exposes the same operation for
review before an owner prepares a change-bound proposal.

## Frozen public validation

The checked JSON preregistration repeats the original Hibernate v1 declaration
without correcting it after the fact:

- public revision `2b448a59d332326f0cd0691c868425124d55cbb5`;
- owner workflow `assemble`;
- declared output `hibernate-core/build/libs/**`; and
- expected review result `NATIVE_FULL_GRAPH / REQUIRED_OUTPUTS_EMPTY`.

The accepted artifact must show a Gradle-owned candidate below
`hibernate-core/target/libs/`, stop before warm-up or timing, and write no
qualified profile. This validates the generic guard that would have prevented
the failed blind-holdout attempt; it does not create another performance
percentage.

## Conformance

```bash
./dev/check-generic-output-contract
./dev/check-generic-output-contract-evidence
```

The first command covers discovery, successful validation, an empty declared
glob and cross-project ownership ambiguity. The second binds the frozen
Hibernate observation to the preregistration.
