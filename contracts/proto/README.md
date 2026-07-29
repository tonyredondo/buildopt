# Protobuf contracts

Versioned Protobuf v3 IDLs for concurrent local component communication.

The first channel is the normative
[`local-events/v1/task_events.proto`](./local-events/v1/task_events.proto),
owned by `F0-019`. Its golden-lane transport is length-delimited Protobuf over a
Unix domain socket. Alternative transports must pass the same conformance
suite.

Buf 1.72.0 is adopted through the repository-root
[`buf.yaml`](../../buf.yaml) with `STANDARD` lint and `FILE` breaking rules.
Run the descriptor, lint, and Go/Java socket round trips from the repository
root:

```bash
./dev/check-task-events-proto
```

Generated clients are outputs of the normative IDL and must never become an
independently edited source of truth. Their materialization and N/N-1 suite
remain owned by `F0-022`.
