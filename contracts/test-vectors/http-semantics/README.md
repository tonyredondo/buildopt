# HTTP failure-semantics vectors

`http-semantics.v1.json` is the common `F0-021` contract for BuildOpt and Test
Optimization control APIs. It fixes:

- stable error codes and their default HTTP status/retryability;
- the global 5-second maximum retry backoff;
- legal unknown-response recovery and fail-closed deadline outcomes;
- idempotency-key reuse semantics;
- accepted-mutation cancellation behavior; and
- executable deadline, retry, conflict, unknown-response, and cancellation
  scenarios.

The default retryability in the catalog is an upper bound. A concrete response
may be stricter, but it cannot make a non-retryable code retryable. Every retry
retains the original key, payload digest, state precondition, and deadline.

Validate the catalog, execute every fault case, and audit all operations in the
three OpenAPI documents with:

```bash
./dev/check-http-semantics
```

A timeout never manufactures a positive gate. An unknown stateful mutation is
resolved by reading durable state before another write. Cancellation after
acceptance does not reverse a durable mutation.
