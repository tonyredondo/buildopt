# Sticky wrapper contract fixtures

These files exercise the `SWL-001` byte and parsing contract. They are not a
customer wrapper and must never be packaged. The inert scripts deliberately
exit before performing any network or Gradle operation; `SWL-003` owns the
real bootstrap body.

`valid/` contains the four portable committed paths with canonical modes and
LF bytes. It is byte-identical to `SWL-002` generation for public release
`v0.6.1` plus the documented example server and project scope.
`invalid-cases.json` applies named mutations to the valid properties
or configuration and records the required rejection class. The Go fixture
tests independently parse the strict properties/configuration formats through
POSIX- and Windows-shaped algorithms, compare their values, validate argument
routing and reject every mutation.

Run the complete contract gate with:

```bash
./dev/check-sticky-wrapper-contract
```
