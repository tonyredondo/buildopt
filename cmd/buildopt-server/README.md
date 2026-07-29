# `buildopt-server`

Go modular monolith for the private beta.

It will host the Shared Cache, Policy API, experiment/evidence state, and export. Internal boundaries must follow versioned contracts, but they will not be split prematurely into microservices. The first ingest flow will be implemented in `WS-005`.
