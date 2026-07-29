# Local events v1

Namespace for the first version of plugin and agent events sent to the Launcher or Local Verifying Cache Gateway.

[`task_events.proto`](./task_events.proto) is the normative `F0-019` channel.
It preserves the exact `task_execution_id`, native cache key, completed task
outcome, and observed PUT outcome when one task owns the operation. An
`UnattributedCachePut` instead carries its attribution reason and an atomic
`WholeAttemptAbort`; the receiver immediately discards every pending write for
that attempt.

`CorrelationCapabilityDeclared` is final and attempt-wide. Exact individual
events do not promote an attempt that also contains a non-task, missing, or
ambiguous PUT. The current Gradle 9.6.1 and 8.14.3 spike result is therefore
`CORRELATION_CAPABILITY_UNAVAILABLE`.

The first frame is `ProducerHello`. Each later `TaskEvent` uses the next
sequence number, and each receiver response is a matching `TaskEventAck`.
Messages are varint-length-delimited, limited to 1 MiB, and exchanged over a
Unix domain socket in the golden lane. `WS-004` authenticates the connection
before the first frame with a fixed `BOA1` preface plus a fresh 256-bit token,
and the receiver also verifies the local peer user. The credential is never a
Protobuf field. Generated clients and N/N-1 policy remain owned by `F0-022`.

Run the exact locked-tool descriptor comparison, Buf lint, Java 17 compilation,
semantic negatives, and both Go/Java Unix-socket directions:

```bash
./dev/check-task-events-proto
```

See [ADR 0003](../../../../adr/0003-local-task-event-channel.md) for the
cross-field, stream-order, and fail-closed rules. Task ownership is never
inferred from timing, thread names, or task paths.
