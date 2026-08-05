# Leaf-source Kotlin value validation

`POC-LEAF-KOTLIN-001` tests the last failure reproduced by both temporal pairing
batches. It measures only `LEAF_CHANGE/KOTLIN`: optimized native Gradle runs
the full five-project verification graph, while Build Impact must run exactly
the one affected project verification.

The generic fixture, requested tasks, required outputs, 500-ms/2%/positive-
lower-bound threshold, and correctness guardrails remain unchanged. A separate
init script raises only the existing deterministic non-cacheable verification
tasks to 25 million rounds in both arms, giving the unchanged 500-ms threshold
enough avoidable critical-path work for an honest test.

Two opposite-order batches run both arms consecutively in one strict
4-CPU/16-GiB container. Workspaces, installations, writable state, Gradle homes,
and daemons remain separate. A stable failure authorizes a narrowly attributed
product experiment; a mismatch authorizes no product change. Qualification
requires identical outputs, exact five-versus-one execution, Configuration
Cache reuse, and zero product-attributable failures in every pair.

This is owner-controlled POC evidence, not a production, universal,
external-user, or Test Optimization claim.
