# Internal Go packages

Private implementation shared by `buildopt` and `buildopt-server`.

`launcher/` contains the dependency-free `WS-001` command passthrough and the `WS-002` Linux process-group and signal contract used by `cmd/buildopt`. It forwards `SIGINT`/`SIGTERM` to the child group, preserves child status, and leaves grace-period escalation to the invoking CI environment. Later lifecycle, policy, gateway, and observation behavior must extend this boundary without replacing contract sources.

No type in this directory replaces the normative schemas, OpenAPI, or Protobuf definitions in `contracts/`.
