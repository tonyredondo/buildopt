# Contract test vectors

Language-neutral fixtures consumed by the same Go and Java conformance suites.

| Namespace | Purpose | Owning item |
|---|---|---|
| `canonical-json/` | JCS, UTF-8, number, timestamp, and digest vectors | `F0-020` |
| `signatures/` | Signed-command and signature verification vectors | `F0-020` |
| `state-machines/` | Valid, invalid, crash/retry, and recovery transitions | `F0-023` |
| `compatibility/` | N/N-1 and incompatible-major behavior | `F0-022` |

`F0-020` materializes the canonical JSON and signature corpora and runs them
through independent Go and Java 17 consumers. The compatibility and
state-machine namespaces remain reserved for `F0-022` and `F0-023`.

Fixtures are normative only when their owning item records the producer,
expected result, and conformance command.
