# Generated control-plane client

This module compiles the checked-in Java 17 transport bindings generated from
the three OpenAPI v1 documents. The source is regenerated only through
`./dev/generate-code --artifact openapi-java-client-v1`; manual edits are
rejected by `./dev/check-generated-code`.

The client performs a single HTTPS attempt and deliberately does not hide
retry, deadline, idempotency, precondition, or response-validation policy from
its caller. `./dev/check-generated-clients` compiles it and runs the shared
N/N-1 compatibility corpus.
