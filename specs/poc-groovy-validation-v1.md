# Groovy value validation

`POC-GROOVY-001` tests one narrow product claim after the paired breadth
experiment reproduced a Groovy no-change regression: an uninstrumented
`buildopt gradle` invocation must remain within 2% of optimized native Gradle,
while Build Impact must still save at least 500 ms and 2% for a leaf change
with a positive paired lower confidence bound.

The generic breadth fixture remains byte-for-byte unchanged so its historical
evidence stays reproducible. This experiment applies
`dev/poc-groovy-workload.init.gradle` to both arms. It raises each existing
deterministic, non-cacheable `pocVerify` task from 5 million to 25 million
rounds. The original workload placed less than 500 ms of avoidable work on the
critical path, so it could not honestly test the unchanged 500 ms accelerator
floor. The profile changes neither requested/selected tasks nor required
outputs.

Two strict 4-CPU/16-GiB batches run both arms sequentially in one persistent
container, removing container identity as a timing variable while retaining
separate workspaces, installations, writable state, Gradle homes and daemons.
Pair order alternates within each batch and the second batch reverses the
starting order. Both reports must classify `NO_CHANGE` as
`PARITY_GUARDRAIL_MET` and `LEAF_CHANGE` as `THRESHOLD_MET`; all outputs,
execution shape, Configuration Cache behavior and zero-failure guardrails must
hold in every observation.

This is owner-controlled POC evidence. It is not a production, universal or
external-user claim, and it does not modify Test Optimization.
