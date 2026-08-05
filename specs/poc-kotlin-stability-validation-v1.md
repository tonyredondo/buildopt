# Kotlin classification stability validation

`POC-KOTLIN-STABILITY-001` resolves the two classifications that remained
order-sensitive after temporal pairing: `SHARED_CHANGE/KOTLIN` and
`BUILD_LOGIC_CHANGE/KOTLIN`.

The experiment reuses the frozen 25-million-round Kotlin verification profile
and the shared strict 4-CPU/16-GiB container. Control and candidate keep
separate workspaces, installations, writable state, Gradle homes, and daemons.
The fixture, requested tasks, outputs, sample count, pair ordering, and original
accelerator/parity thresholds remain unchanged.

Shared-source qualification requires exact five-versus-two verification,
Configuration Cache reuse, at least 500 ms and 2% mean savings, and a positive
paired lower bound. Global build-logic qualification requires five-versus-five
verification, Configuration Cache invalidation in both arms, and no regression
beyond 2%. Every pair must retain identical outputs and zero
product-attributable failures.

Only a negative classification reproduced in both starting orders may
authorize a narrow product experiment. A mismatch authorizes no product
change. This is bounded owner-controlled POC evidence, not a production,
universal, external-user, soak, or Test Optimization claim.
