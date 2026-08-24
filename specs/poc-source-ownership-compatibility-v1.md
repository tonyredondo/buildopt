# Source ownership compatibility

`POC-SOURCE-OWNERSHIP-COMPATIBILITY-001` investigates the only fallback in
the frozen cross-commit breadth V2 window that still ends as
`SOURCE_OWNERSHIP_AMBIGUOUS`. The subject is Apache Groovy revision
`1ff25776`, whose only changed repository path is `versions.properties` and
whose owner workflow is `classes`.

The baseline BuildOpt revision is `d2b68a9`. A read-only Gradle 8.14.3/JDK 21
observation established that the requested graph is complete, reaches 33 of
37 projects and contains 166 tasks. The changed path has no source-set owner
and zero direct consuming tasks. The repository reads it through
`Provider.fileContents` during configuration, so absence from `TaskInputs` is
not proof that the change is irrelevant. The stored qualification transported
3,823 outputs while the current workflow requires 3,890; the prior output
contract is therefore not identical either.

The implementation must distinguish three repository-independent cases:

1. A path with one direct source owner keeps that owner.
2. A path without a source owner but with consumers in the complete requested
   task graph may be attributed to those tasks' declared Gradle projects.
3. A change set composed only of unowned, unconsumed paths is configuration
   input scope that BuildOpt cannot prove from task ownership. It must remain
   native with `CONFIGURATION_INPUT_OWNERSHIP_UNPROVEN`.

Equal-specificity source ownership, unknown consuming tasks, incomplete input
evidence or invalid paths continue to fail closed. The Groovy subject is not
required to select a profile: an honest native-retained result is the expected
outcome unless the generic evidence proves otherwise. The other seven frozen
fallback decisions, exact required outputs, authoritative native execution and
zero-failure boundary must remain unchanged.

This is a bounded POC experiment. It adds no filename or repository identity
rule, no production authority, no soak or design-partner dependency, and no
Test Optimization behavior. The frozen contract is validated with:

```bash
./dev/check-source-ownership-compatibility --spec-only
```
