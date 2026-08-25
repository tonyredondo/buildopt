# Adaptive fragment state v1

Status: accepted POC state contract for `AF-002`.

Machine policy: [`poc-adaptive-fragment-state-v1.json`](./poc-adaptive-fragment-state-v1.json).

## Purpose

This contract makes the `AF-001` fragment model persistable without yet
building the compatibility index, learner, planner or central synchronization.
The four documents are immutable generations. Unknown versions and fields fail
closed; state migration is deliberately outside this POC block.

## Documents

| Record | Role | Identity and linking |
|---|---|---|
| `ADAPTIVE_FRAGMENT` | Current immutable lifecycle generation of one fragment revision. | Carries the canonical AF-001 family/revision, authority, bindings, state and generation. |
| `ADAPTIVE_FRAGMENT_OBSERVATION` | Append-only transition evidence. | Links exact repository, family, revision and contiguous generation plus the external evidence digest. |
| `ADAPTIVE_FRAGMENT_PORTFOLIO` | Repository-scoped snapshot of current fragment generations. | References one revision/generation/state per sorted unique family and links its predecessor by digest after generation one. |
| `ADAPTIVE_FRAGMENT_ECONOMIC_LEDGER` | Typed signed-value snapshot for the portfolio. | Links the exact portfolio generation and fragment generations. `AF-005` owns recurrence, decay, payback and activation formulas. |

The normative Draft 2020-12 schemas are:

- [`adaptive-fragment.v1.schema.json`](../contracts/jsonschema/adaptive-fragment.v1.schema.json)
- [`adaptive-fragment-observation.v1.schema.json`](../contracts/jsonschema/adaptive-fragment-observation.v1.schema.json)
- [`adaptive-fragment-portfolio.v1.schema.json`](../contracts/jsonschema/adaptive-fragment-portfolio.v1.schema.json)
- [`adaptive-fragment-economic-ledger.v1.schema.json`](../contracts/jsonschema/adaptive-fragment-economic-ledger.v1.schema.json)

## Canonical digest

Every individual document is canonicalized with RFC 8785 JCS and hashed with
SHA-256; the transport form is 64 lowercase hexadecimal characters. The digest
is stored by the containing reference or state manifest, not inside the hashed
document. This avoids a self-referential digest and lets local files and the
existing HTTPS state plane use the same bytes.

Semantic sets (`requires`, `conflictsWith`, portfolio families and ledger
families) are unique and lexicographically ordered before publication. Family
and revision identities retain the domain-separated length-prefixed algorithm
from `AF-001`; JCS does not redefine them.

## Cross-record invariants

JSON Schema owns closed shapes, enums, formats, bounds and kind-specific
authority/binding requirements. The Go conformance layer additionally proves:

1. AF-001 family and revision identities recompute exactly;
2. observations share repository/family/revision and advance generations
   contiguously;
3. only the declared lifecycle transitions occur, including mandatory shadow
   requalification after suspension;
4. fragment state/generation equals the last observation;
5. portfolio and ledger entries bind the same fragment generation;
6. ledger generation binds the exact portfolio generation;
7. repository scope never crosses documents; and
8. timestamps are UTC, monotonic and do not predate referenced state.

Missing, corrupt, unknown, cross-repository or generation-incompatible state
is unusable. Later runtime blocks must retain optimized native Gradle rather
than repair or reinterpret it implicitly.

## Vectors and boundaries

Two valid bundles cover initial activation and
`ACTIVE -> SUSPENDED -> SHADOW -> QUALIFIED -> ACTIVE`. Seven negative
mutations cover unknown schema version, identity tampering, unknown authority,
repository crossing, generation drift, impossible transition and canonical
digest mismatch.

Run:

```bash
./dev/check-adaptive-fragment-state
```

The accepted outcome is `TYPED_FRAGMENT_STATE_AVAILABLE`. No record activates a
fragment by itself. This block makes no performance, production, migration,
central-sync, soak, design-partner or Test Optimization claim.
