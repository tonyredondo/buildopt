# Optional central cache and state contract

## Decision

The POC may expose one owner-operated HTTPS service to share Gradle cache
objects and BuildOpt learning between machines, but it must treat them as two
different planes. Gradle owns opaque cache entries. BuildOpt owns typed
portfolio, evidence and checkpoint state. They may share physical
content-addressed files, but never logical metadata, authorization, retention
or visibility.

This block defines the contract only. It does not add a remote listener,
credentials, synchronization or automatic remote selection.

## Namespace model

| Namespace | Logical identity | Mutation | Retention | Eviction/failure result |
|---|---|---|---|---|
| Gradle object | tenant, repository, trust domain, compatibility class, namespace generation, cache key | Existing pending attempt followed by atomic commit | 30-day stable byte-SLRU and quota | Ordinary cache miss |
| Portfolio | repository, `PORTFOLIO`, generation, compatibility digest | Immutable artifacts/manifest, then head CAS | Current plus 30 days after supersession | Revalidate another compatible source or use native |
| Evidence | repository, `EVIDENCE`, generation, exact bindings | Immutable artifacts/manifest, then head CAS | While referenced, then 30 days | Referencing portfolio becomes unusable |
| Checkpoint | repository, `CHECKPOINT`, generation, exact invocation bindings | Immutable artifacts/manifest, then head CAS | 24 hours | Restart learning or use native |

The repository scope and kind are always part of a state lookup. Identical
physical bytes in two namespaces may deduplicate on disk, but a metadata row in
one namespace never authorizes a read from another. A Gradle cache key cannot
address a BuildOpt state object, and a BuildOpt digest cannot become a Gradle
cache hit.

## Immutable publication and CAS

State publication has four ordered steps:

1. upload every artifact by SHA-256 into the exact repository/kind namespace;
2. upload a schema-valid immutable manifest that lists those artifacts;
3. verify every referenced artifact and evidence manifest; and
4. compare-and-swap the namespace head to exactly the next generation.

Generation one uses `If-None-Match: *`. Later generations use `If-Match` with
the SHA-256 ETag of the canonical current head. The next generation must equal
the current generation plus one. An exact retry with the same idempotency key
and request is safe; changed content under the same key is a conflict. A stale
head is `412`, a skipped generation is `409`, and an incomplete manifest is
`422`.

State document addresses and head ETags are SHA-256 over RFC 8785 JCS bytes.
The schemas deliberately restrict numeric fields to integers and all identity
strings to deterministic encodings, so the existing cross-language canonical
JSON corpus remains applicable. HTTP transfer whitespace never changes a state
identity.

Artifacts and manifests that exist without a successful head CAS are staged
garbage, not readable optimization authority. An interrupted, rejected or
partially uploaded calibration therefore cannot promote itself.

## State documents

[`CENTRAL_STATE_MANIFEST`](../contracts/jsonschema/central-state-manifest.v1.schema.json)
binds one immutable generation to its repository, compatibility and exact
origin, plus content-addressed artifacts and evidence references.
[`CENTRAL_STATE_HEAD`](../contracts/jsonschema/central-state-head.v1.schema.json)
is the sole mutable pointer. [`CENTRAL_STATE_CAS`](../contracts/jsonschema/central-state-cas.v1.schema.json)
is its idempotent preconditioned update.

Every state document says `selectionRequiresLocalRevalidation=true`,
`productionAuthorized=false` and `testOptimization=OUT_OF_SCOPE`. Central
storage preserves previously qualified inputs; it does not decide that a
profile applies to a new commit. The client must still validate repository,
Wrapper, Gradle, executable, graph, options, output contract and change-family
bindings before Gradle starts.

## HTTP surface

Gradle continues using `GET|PUT /cache/{cacheKey}` through the invocation-local
verifying gateway. BuildOpt state uses separate routes below
`/api/v1/repositories/{repositoryScope}/state/{kind}` for immutable objects,
immutable manifests and the CAS head. State responses use `Cache-Control:
no-store`; payload routes use complete SHA-256 verification before returning
bytes.

The next HTTPS/authentication block will map the independent `CACHE_READ`,
`CACHE_WRITE`, `STATE_READ` and `STATE_WRITE` capabilities to owner-issued POC
credentials. This contract fixes those capabilities without implementing or
pretending to validate the credential system early.

## Failure behavior

- A Gradle-cache miss or outage falls through to local cache and ordinary task
  execution.
- A state outage may use a previously verified local snapshot only after all
  exact bindings still match.
- Without such a snapshot, `buildopt optimize` runs optimized native Gradle.
- A state upload failure after Gradle starts is diagnostic and preserves the
  Gradle exit or signal status.
- Corruption, schema drift, namespace mismatch, stale CAS and incomplete state
  fail closed for optimization, not for the customer build.

Local `buildopt optimize` remains fully supported without the service.

## Executable evidence

Run:

```bash
./dev/check-central-storage-contract
```

The checker compiles all three Draft 2020-12 schemas and executes the
language-neutral vectors under
[`central-storage`](../contracts/test-vectors/central-storage/README.md). The
vectors cover Gradle eviction, namespace isolation, complete portfolio/evidence
publication, first-generation and update CAS, exact replay, stale and skipped
generations, idempotency conflicts, incomplete manifests, referenced-evidence
retention, checkpoint expiry, service outage and safe native fallback.

## POC boundary

This contract does not implement storage, HTTPS, credentials, client sync,
cross-commit applicability or a two-machine performance claim. It does not
require production HA, multi-tenancy, RBAC administration, KMS/HSM, backup
SLAs, soak or a design partner. Test Optimization remains outside this
product. The next block is `POC-CENTRAL-STATE-STORAGE-001`, which must implement
these exact state semantics on the existing content-addressed/SQLite
foundation.
