# Structural profile rebinding v1

## Purpose

This POC separates structural profile compatibility from Git revision
identity. A qualified profile may be considered on a later commit or a
different checkout only when the requested workflow still has the same
canonical structural fingerprint.

The fingerprint binds:

- repository scope;
- original and candidate entrypoints plus Gradle options;
- the Gradle Wrapper digest;
- the complete dependency closure of candidate and output-producing tasks;
- required and candidate outputs, their kind, unique owner and producers; and
- the change family and its resolved projects.

Commit IDs and absolute checkout paths are deliberately excluded. Evidence and
output revisions remain separate, mandatory safety bindings: the evidence
revision must be an ancestor, and materialized bytes remain tied to the native
revision that produced them. Structural compatibility never turns stale bytes
into current outputs.

## Decisions

1. Discovery emits the fingerprint only from complete typed Gradle task and
   output evidence.
2. A portfolio entry without the fingerprint is invalid and retains native
   Gradle.
3. Central replay checks the current repository, workflow, Wrapper and change
   family against the fingerprint before selecting the profile.
4. Producer-lineage or output-contract drift creates a different fingerprint;
   incomplete, ambiguous, missing or cyclic evidence is rejected.
5. Build-logic and materialized-producer changes still require an authoritative
   native refresh even when the broader structural identity remains compatible.

## POC boundary

This block proves a revision-independent compatibility identity and safe
cross-revision selection/refresh behavior. It makes no build-time claim, does
not authorize production selection, does not infer ABI compatibility and does
not require soak or design-partner evidence. Test Optimization remains outside
scope.

Validate the contract and committed evidence with:

```bash
./dev/check-structural-profile-rebinding
```
