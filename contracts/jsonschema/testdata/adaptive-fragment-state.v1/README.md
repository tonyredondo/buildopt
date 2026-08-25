# Adaptive fragment state vectors

The two valid bundles exercise the initial lifecycle and suspension followed by
mandatory shadow requalification. Each bundle contains one document for all
four `AF-002` record types plus the complete observation chain that links them.

[`invalid/vectors.json`](./invalid/vectors.json) applies seven deterministic
mutations to the initial valid bundle. They prove schema rejection, canonical
identity tamper detection, repository and generation isolation, impossible
lifecycle rejection and JCS digest drift. The mutations are data; the checker
must not rewrite a fixture after seeing a result.

Run:

```bash
./dev/check-adaptive-fragment-state
```
