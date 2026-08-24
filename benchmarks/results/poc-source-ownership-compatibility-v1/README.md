# Source ownership compatibility result

This bounded POC closes the one post-discovery ownership ambiguity left in the
frozen cross-commit breadth V2 window. Apache Groovy revision `1ff25776` changes
only `versions.properties` for the `classes` workflow. A fresh Gradle 8.14.3
observation is complete at 37 projects, 166 tasks and 33 reached projects, but
the changed path has neither a direct source owner nor a consuming task. Groovy
reads it through `Provider.fileContents` during configuration.

BuildOpt now reports that case explicitly as
`CONFIGURATION_INPUT_OWNERSHIP_UNPROVEN` and retains native Gradle. A generic
fixture separately proves that an unowned path consumed by a known task can be
attributed to that task's project; unknown consumers and ambiguous source roots
remain fail-closed. There is no repository, filename or extension rule.

The directed public replay ran `classes` in an independent source root in
29.710 seconds. It produced 3,890 required outputs with the same aggregate
SHA-256 as the diagnostic root, and recorded zero product failures. This is a
correctness and compatibility result, not a wall-time saving.

The attempted full lifetime replay is also retained rather than retried for a
favourable sample. Its target calibration averaged 29.623 seconds for optimized
native Gradle and 28.423 seconds for the structural candidate, a descriptive
1.200-second/4.05% mean saving with 6/8 positive pairs. The paired interval was
-0.262 to +2.625 seconds, so the unchanged value gate rejected qualification
and no descendant profile was published. The previous seven early-retention
observations remain checksum-bound and exercise code paths that this block did
not change; a complete fresh eight-revision runtime replay was not possible
after the target correctly failed qualification.

Validate the preregistration and terminal evidence with:

```bash
./dev/check-source-ownership-compatibility
```

This result authorizes no production behavior, weaker output gate, soak or
design-partner requirement, Test Optimization behavior, or claim that
configuration-time dependencies are safely selectable. A future experiment
would need a generic, complete configuration-input binding; task ownership
alone is insufficient.
