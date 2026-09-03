# Contract test vectors

Language-neutral fixtures consumed by the same Go and Java conformance suites.

| Namespace | Purpose | Owning item |
|---|---|---|
| `canonical-json/` | JCS, UTF-8, number, timestamp, and digest vectors | `F0-020` |
| `signatures/` | Signed-command and signature verification vectors | `F0-020` |
| `http-semantics/` | Stable errors, deadlines, retries, idempotency, and cancellation | `F0-021` |
| `state-machines/` | Valid, invalid, crash/retry, and recovery transitions | `F0-023` |
| `compatibility/` | N/N-1 and incompatible-major behavior | `F0-022` |
| `central-storage/` | Gradle/state namespace isolation, immutable publication, CAS, retention, outage, and fallback | `POC-CENTRAL-STORAGE-CONTRACT-001` |
| `wcncp/` | Synthetic valid/invalid observation, opportunity, proposal, validation, and owner-decision records | `WCNCP-000` |

`F0-020` materializes the canonical JSON and signature corpora and runs them
through independent Go and Java 17 consumers. `F0-021` materializes the common
HTTP failure-semantics catalog and audits all control-plane operations. The
compatibility and state-machine namespaces remain reserved for `F0-022` and
`F0-023`. The central-storage and WCNCP vectors are consumed by the isolated Go
Draft 2020-12 validator and define a POC contract, not a deployed service.

Fixtures are normative only when their owning item records the producer,
expected result, and conformance command.
