# Central storage contract vectors

These language-neutral vectors define the POC boundary between Gradle cache
objects and BuildOpt portfolios, evidence and checkpoints. They are not server
fixtures and do not authorize a production deployment.

The catalog contains schema-valid manifests, deliberately invalid manifests
and stateful operation sequences. Each scenario starts with empty storage and
declares the expected result of every operation and the final visible heads.
The conformance runner computes canonical manifest/head digests rather than
trusting a digest copied into the vector.

Covered behavior:

- Gradle object hit, eviction and ordinary miss without optimization authority;
- immutable artifact and manifest upload before visibility;
- generation-one creation and exact next-generation CAS;
- idempotent replay, changed-payload conflict and stale/skipped generation;
- repository/kind namespace isolation even for identical physical bytes;
- portfolio dependence on an exact evidence manifest;
- referenced-evidence retention and short-lived checkpoint expiry; and
- compatible local-snapshot fallback or optimized native Gradle on outage.

Run:

```bash
./dev/check-central-storage-contract
```

The owning item is `POC-CENTRAL-STORAGE-CONTRACT-001`.
