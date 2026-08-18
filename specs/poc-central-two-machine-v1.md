# Central two-machine proof

This contract proves that the BuildOpt POC can move reusable Gradle state and
verified cache objects between two isolated build machines through one
owner-operated HTTPS service. It is a functional experiment, not a production
deployment design or a wall-time claim.

## Topology

The producer and consumer run in different Docker containers with independent
workspaces, Gradle User Homes and BuildOpt cache roots. They share no
filesystem volume. Their only persistent common component is the HTTPS server
state directory.

The producer is a trusted CI writer. It discovers and qualifies a source-only
Build Impact profile through `buildopt optimize`, publishes the immutable
state bundle, writes cache objects into a pending attempt and requires an
explicit signed owner commit before those objects become readable.

The clean consumer runs `buildopt connect` once and then uses the normal
`buildopt optimize` command. A verified central profile activates a loopback
Gradle cache gateway automatically. The gateway owns the central credential;
Gradle receives only invocation-local credentials and a read-only cache URL.

## Required proof

1. A producer publishes a qualified profile and at least one pending cache
   object.
2. The owner commits the exact pending object set.
3. The central service restarts with the same state directory.
4. A clean consumer selects `CENTRAL_PORTFOLIO`, restores at least one task
   `FROM-CACHE` and produces the same required JAR digest as the producer.
5. With the service stopped and local cache entries removed, the consumer
   retains the verified profile snapshot, treats central cache access as a
   miss, executes successfully and produces the same JAR digest.
6. Logs contain neither producer nor consumer central credentials.

The harness records phase durations only to establish that the path is
observable. It does not compare against native Gradle and therefore cannot
support a performance claim. That equal-opportunity comparison belongs to
`POC-CENTRAL-END-TO-END-VALUE-001`.

## POC boundary

This proof does not require a soak, a design partner, HA, RBAC, KMS integration
or backup automation. It never authorizes production use and keeps Test
Optimization out of scope.
