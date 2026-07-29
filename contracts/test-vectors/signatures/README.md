# Signature vectors

`ed25519.tsv` contains synthetic Ed25519 verification vectors over the exact
canonical payload bytes from the JCS corpus. The fixed public keys are
non-secret RFC test material; no deployment credential or private key is
stored.

The corpus covers a valid signed command, changed content, another key, and a
malformed signature. Unknown signed-command fields are rejected by the strict
JSON Schemas and receive an explicit cross-version vector in `F0-022`.

Run both the Go and Java 17 verifiers with:

```bash
./dev/check-contract-crypto
```
