# Spring Messaging Fresh-Control Correctness v1 evidence

The new empty-state sequence naturally produced `OPTIMIZED_NATIVE`,
`OPTIMIZED_NATIVE`, then `INCREMENTAL_CANDIDATE`. Every request observed 11
compatible matches, exited zero and produced all 14,406 required outputs with
the same complete digest. No path was excluded or normalized, product failures
are zero and no timing sample exists.

This authorizes only a separately frozen Spring Messaging paired-value
contract. Run `./dev/check-spring-messaging-fresh-control-correctness` to
reconstruct raw hashes, modes, history, complete-output equality and the
authorization boundary.
