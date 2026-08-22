# Cross-commit value recovery v1

## Purpose

This proof-of-concept contract tests whether an already-qualified structural
profile can create attributable wall-time value on a later compatible public
commit. It follows the unchanged Apache Kafka window retained by qualified
lifetime v2. It does not add a repository-name rule, weaken exact-output
validation, or treat favorable native-arm noise as BuildOpt value.

## Product changes under test

The implementation must:

1. reject repository, Wrapper, workflow, build-logic, graph, and output drift
   before central synchronization or materialization;
2. refresh a qualified profile and its output pack on an authoritative native
   descendant when structural bindings remain compatible;
3. let a read-only consumer reuse that locally verified refreshed snapshot
   through the same materialization and binding validation as central state;
4. keep the qualification target, revalidated descendant, and materialized
   output revision distinct;
5. avoid attaching full-workflow output observation tasks after a selective
   profile has already been selected; and
6. retain optimized native Gradle on every uncertain or ineligible revision.

The fifth property is essential. Output observation is useful while learning,
refreshing, or falling back, but attaching its complete original workflow to a
selected replay reconstructs the work that graph reduction intentionally
removed.

## Frozen comparison

The before and after captures use:

- the same public Apache Kafka repository;
- qualifier revision
  `2e961afeff5cb27d60767a783edf20be00cc28e8`;
- the same six first-parent descendant revisions and alternating arm order;
- the same `:clients:testClasses` workflow and required-output inventory;
- exact output equality and public-ancestry checks from qualified lifetime v2;
- one installed BuildOpt executable per capture; and
- persistent but arm-isolated Gradle and BuildOpt state.

The before capture includes refreshed local replay but still attaches the
full-workflow observation graph. The after capture removes only that redundant
observation from a selected replay.

## Acceptance

The result is complete only when:

- both captures independently pass the qualified-lifetime v2 subject checker;
- the frozen repository, qualifier, six revisions, selected revision, workflow,
  and exact-output gates match;
- the before selected replay is regressive;
- the after selected replay uses a verified local profile and saves wall time;
- selected-replay savings are reported separately from native fallback deltas;
- at least one profile is selected;
- cumulative net value remains positive after qualification and publication
  cost; and
- all required outputs match with zero product-attributable failures.

The result may not claim that the selected mechanism caused fallback-arm
timing. The cumulative window is reported because it answers whether the POC
paid back; the selected replay is reported separately because it answers
whether cross-commit reuse itself accelerated the build.

## Reproduction

```bash
./dev/check-cross-commit-value-recovery
```

The long installed-package capture is produced by
`dev/run-qualified-lifetime-v2`. The checked bundle deliberately retains both
before and after raw subject evidence so the improvement cannot be reconstructed
from a favorable terminal summary alone.

## Boundaries

This is bounded POC evidence from one public repository window. It adds no
production authority, universal percentage, repository average, soak or design
partner requirement, or Test Optimization behavior.
