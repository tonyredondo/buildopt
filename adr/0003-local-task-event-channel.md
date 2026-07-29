# ADR 0003: Local task-event channel

- Status: accepted
- Date: 2026-07-29
- Items: `F0-019`, `GRADLE-CORR-001`

## Context

The Gradle plugin and optional JVM agent can produce events concurrently while
the Launcher and Local Verifying Cache Gateway own attempt lifecycle and pending
cache writes. The channel therefore needs a language-neutral wire contract that
preserves observed identifiers and fails closed when task ownership is not
unique.

`GRADLE-CORR-001` proved exact attribution for task-owned stores on the tested
Gradle 9.6.1 and 8.14.3 paths, but cold Kotlin DSL and accessor compilation also
produced remote stores with no task ancestor. Missing or ambiguous ancestry is
likewise possible. A task-only exact subset cannot be promoted to an exact
attempt capability: any such `UNATTRIBUTED` PUT makes correlation unavailable
and aborts the complete pending attempt.

## Decision

The normative v1 IDL is
[`contracts/proto/local-events/v1/task_events.proto`](../contracts/proto/local-events/v1/task_events.proto).
It uses Protobuf v3 with separate Go and Java package options. Buf 1.72.0 is
adopted with its `STANDARD` lint category and `FILE` breaking policy. The single
lint exception is `PACKAGE_DIRECTORY_MATCH`, because RFC section 29.2 fixes the
on-disk `local-events/` path while Protobuf packages use `local_events`.

The golden-lane transport is a bidirectional Unix domain `SOCK_STREAM`.
Messages use the conventional length-delimited form: an unsigned base-128
varint byte length followed by exactly that many serialized message bytes. A
frame is limited to 1 MiB. Producer-to-receiver frames contain `TaskEvent`;
receiver-to-producer frames contain `TaskEventAck`.

The authenticated rendezvous, token lifecycle, and socket ownership are owned
by `WS-004`. This ADR fixes the payload and framing consumed by that work; it
does not put long-lived credentials in task events or treat filesystem
permissions alone as the final authentication design.

## Protocol invariants

- The first event for each `producer_instance_id` is `ProducerHello` with a
  supported major/minor protocol version. Its sequence number is `1`; later
  frames increase strictly by one.
- `attempt_id` and `producer_instance_id` are non-empty and are repeated in the
  matching acknowledgement. A mismatched acknowledgement is a protocol
  violation.
- `CachePutObserved.exact` contains the observed native cache key, PUT outcome,
  one non-empty `task_execution_id`, and one completed task outcome. Task paths,
  timing, and thread names are never substitutes for that identifier.
- `put_operation_id` and `native_cache_key` are non-empty opaque strings and
  are preserved byte-for-byte; consumers do not normalize case or infer a key
  from another field.
- `CachePutObserved.unattributed` contains the supported attribution reason and
  a `WholeAttemptAbort` whose reason is
  `ATTEMPT_ABORT_REASON_UNATTRIBUTED_CACHE_PUT`. Receipt immediately discards
  every pending write for the attempt and returns
  `TASK_EVENT_ACK_STATUS_ATTEMPT_ABORTED`; the receiver does not wait for a
  later explanatory event.
- `CorrelationCapabilityDeclared` is attempt-wide and final. `EXACT` has no
  abort. `UNAVAILABLE` has a `WholeAttemptAbort`. Exact individual PUTs may
  coexist with an attempt-wide `UNAVAILABLE` declaration and never override it.
- An absent attribution arm, unspecified required enum, malformed or oversized
  frame, sequence gap, unsupported version, disconnect before the final
  capability declaration, or producer failure aborts the whole attempt. It
  never makes the baseline Gradle build fail and never authorizes publication.
- An accepted event is an observation, not a commit authorization. Stable cache
  visibility remains governed by the later attempt and `CommitDecision`
  contracts.

Proto3 cannot express all cross-field and stream-order requirements. Consumers
must apply these invariants after decoding and before acknowledging a frame as
accepted.

## Compatibility

Fields are additive within major version 1. Existing field numbers and enum
numbers are never reused. Unknown fields are tolerated at the Protobuf decoder
boundary, but an unsupported major version, unknown required semantic enum, or
unknown payload arm fails closed for optimization. The full N/N-1 generated
client and compatibility policy remains owned by `F0-022`.

## Rejected alternatives

- JSON over the socket: larger and less suitable for concurrent local event
  traffic, and it contradicts the RFC's Protobuf decision.
- Fixed-width frame lengths: they diverge from conventional delimited Protobuf
  streams without adding a useful local-channel property.
- Temporal, thread-name, or task-path correlation: the spike does not prove
  uniqueness and the RFC explicitly forbids these guesses.
- Per-task abort after an unattributed PUT: the gateway cannot prove which
  pending object is safe, so selective retention would turn missing evidence
  into authorization.
- Treating every task-owned event as an exact attempt: cold non-task stores
  demonstrate that this silently drops relevant PUTs.

## Consequences

`WS-003` can now implement the non-optimizing Gradle producer handshake and
`WS-004` can implement the authenticated rendezvous against one stable payload
contract. The initial Gradle combinations still declare correlation
`UNAVAILABLE`, so selective publication stays disabled even when individual
task-owned events are exact.

Generated clients are not checked in by this decision. `F0-022` owns generated
Go/Java clients and N/N-1 compatibility; `F0-005` owns generated-code drift
policy. The `F0-019` conformance peers deliberately use only standard-library
wire readers so this contract can prove cross-language bytes before those
generated artifacts exist.

## Validation

Run:

```bash
./dev/check-task-events-proto
```

The checker resolves exact locked `protoc` 35.1 and Buf 1.72.0 from the
repository-local tool root, compiles the same descriptor with both tools, runs
Buf lint, and uses locked Go 1.26.5 plus Temurin 21 with Java 17 bytecode.
Java-to-Go and Go-to-Java peers exchange framed messages over real Unix sockets
and cover exact attribution, `UNATTRIBUTED`, attempt-wide `UNAVAILABLE`,
whole-attempt abort, acknowledgements, malformed semantic combinations, and the
1 MiB frame bound.

## Sources

- [Protocol Buffers encoding](https://protobuf.dev/programming-guides/encoding/)
- [Protocol Buffers Java generated code](https://protobuf.dev/reference/java/java-generated/)
- [Buf v2 configuration](https://buf.build/docs/configuration/v2/buf-yaml/)
