# Kotlin measurement-boundary decision

`POC-KOTLIN-BOUNDARY-001` is the terminal stop-or-replicate decision for the
two Kotlin cells that remained order-sensitive after temporal pairing and the
calibrated stability rerun.

The decision uses only the already recorded `E-163` and `E-167` reports. It
does not collect another sample, change the workload, move the original
500-ms/2%/positive-bound accelerator threshold, relax the 2% parity
guardrail, discard a signed pair, or tune product code.

Four batches per cell are available. Shared-source alternates
`PASS/FAIL/PASS/FAIL`; its latest failed batch still has 8/8 positive pairs,
27.2% mean savings, and a positive interval, but misses the absolute floor by
31.875 ms. Global build-logic alternates `FAIL/PASS/PASS/FAIL`, executes all
five verifications in both arms, and has no stable product-attributable
regression. Every batch preserves required outputs and records zero product
failures.

There is therefore no remaining correctness defect, reproduced negative
classification, or new causal hypothesis for a third unchanged experiment to
test. Repeating until the noisy classification lands on the preferred side of
a fixed boundary would be optional stopping. The selected terminal decision is
`STOP_RETAIN_BOUNDED_CLAIM`: retain only the previously qualified synthetic
claims, keep these two cells blocked from generalization, and authorize no
product change or further replication inside the current POC.

This is an owner-controlled POC boundary, not a production, universal,
external-user, soak, or Test Optimization claim.
