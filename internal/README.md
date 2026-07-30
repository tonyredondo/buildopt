# Internal Go packages

Private implementation shared by `buildopt` and `buildopt-server`.

`launcher/` contains the dependency-free `WS-001` command passthrough, the
`WS-002` Linux process-group and signal contract, the `WS-003` plugin handshake,
and the neutral `WS-004` authenticated local rendezvous used by `cmd/buildopt`.
It forwards `SIGINT`/`SIGTERM` to the child group, preserves child status, owns
the private event socket and loopback readiness gateway, and consumes the
`F0-039` local bypass before creating either service or parsing server
configuration. The bypass uses the same process/signal contract and removes all
reserved launcher state from the child. Grace-period escalation remains with
the invoking CI environment. The gateway has no cache data or upstream route
yet. `A0-001` adds the opt-in managed runner-slot path: a current-user private
state root, exclusive invocation and gateway leases, a detached idle-bounded
process, UID-authenticated invocation registration, context-gated readiness,
restart-stable identity, and complete rotation when the endpoint cannot be
recovered. `A0-003` adds the launcher-owned native L1 lifecycle: opaque
tenant/repository/trust/compatibility scoping, generation-segmented private
directories, an exclusive child-lifetime lease, and local-cache disablement
for pending L2 writers. Authenticated generation authority, remote
population, and revoked-directory deletion remain with later A0 blocks.
Later policy, cache, and observation behavior must extend this boundary
without replacing contract sources.

`sessioningest/` contains the provisional `WS-005` gateway-to-server record,
strict authenticated HTTP transport, and concurrency-safe in-memory acceptance
store. Its optional `WS-006` handoff carries only predeclared tokenized context
and facts from an authenticated Gradle invocation.

`buildsession/` is the dependency-free producer for the normative
`BUILD_SESSION v1` schema and the atomic local-file exporter. It derives only
deterministic manifest/baseline digests, declares unobserved metrics
unavailable, publishes mode-`0600` immutable JSON, and leaves runtime schema
conformance to the isolated validator under `dev/schema-validator/`.

`neutralenvelope/` owns the strict `WS-009` observation and report contract. It
pairs externally timed native and optimization-off wrapper executions,
reconciles required-output digests, retains signed differences, and binds the
runner, metric catalog, envelope, launcher, server, and plugin inputs.

No type in this directory replaces the normative schemas, OpenAPI, or Protobuf definitions in `contracts/`.

`generated/openapi/` contains the checked-in Go transport binding derived from
the normative OpenAPI documents. It is regenerated through
`./dev/generate-code --artifact openapi-go-client-v1` and never edited
manually.
