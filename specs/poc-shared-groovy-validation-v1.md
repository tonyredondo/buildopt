# Shared-source Groovy value validation

`POC-SHARED-GROOVY-001` tests the remaining reproduced Groovy value gap on the
same calibrated measurement boundary used by `POC-GROOVY-001`. It measures only
`SHARED_CHANGE`: optimized native Gradle runs the full five-project verification
graph, while Build Impact must run exactly the two affected project
verifications.

The generic fixture, requested tasks, required outputs, 25-million-round
deterministic verification profile, 500-ms/2%/positive-lower-bound threshold,
and correctness guardrails are unchanged. Two opposite-order batches run both
arms consecutively in one strict 4-CPU/16-GiB container. Workspaces,
installations, writable state, Gradle homes, and daemons remain separate.

A stable failure authorizes a narrowly attributed product experiment. A
classification mismatch authorizes no product change. Qualification requires
both batches to exceed the unchanged accelerator threshold with byte-identical
outputs, exact five-versus-two execution shape, Configuration Cache reuse, and
zero product-attributable failures.

This is owner-controlled POC evidence. It is not a production, universal,
external-user, or Test Optimization claim.
