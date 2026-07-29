# Protobuf contracts

Versioned Protobuf v3 IDLs for concurrent local component communication.

The first channel is `local-events/v1/task_events.proto`, owned by `F0-019`. Its golden-lane transport is length-delimited Protobuf over a Unix domain socket. Alternative transports must pass the same conformance suite.

Generated clients are outputs of the normative IDL and must never become an independently edited source of truth.
