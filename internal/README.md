# Internal Go packages

Private implementation shared by `buildopt` and `buildopt-server`.

`launcher/` contains the dependency-free `WS-001` command passthrough used by `cmd/buildopt`. Process groups and signals remain owned by `WS-002`; later lifecycle, policy, gateway, and observation behavior must extend this boundary without replacing contract sources.

No type in this directory replaces the normative schemas, OpenAPI, or Protobuf definitions in `contracts/`.
