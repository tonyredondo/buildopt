# Internal Go packages

Private implementation shared by `buildopt` and `buildopt-server`.

`launcher/` contains the dependency-free `WS-001` command passthrough, the
`WS-002` Linux process-group and signal contract, the `WS-003` plugin handshake,
and the neutral `WS-004` authenticated local rendezvous used by `cmd/buildopt`.
It forwards `SIGINT`/`SIGTERM` to the child group, preserves child status, owns
the private event socket and loopback readiness gateway, and leaves
grace-period escalation to the invoking CI environment. The gateway has no
cache data or upstream route yet; later lifecycle, policy, cache, and
observation behavior must extend this boundary without replacing contract
sources.

No type in this directory replaces the normative schemas, OpenAPI, or Protobuf definitions in `contracts/`.
