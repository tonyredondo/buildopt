# PatchBundle application v1

This specification materializes `F0-034` and defines the only safe consumer of
the `PatchBundle v1` schema. The bundle is declarative data: the consumer never
executes it, invokes a hook from it, applies a fuzzy diff, deletes a path, or
changes an executable bit.

## Verification order

The customer-side patcher must complete these checks before changing a
worktree:

1. strictly decode the closed schema and reject duplicate JSON keys;
2. negotiate `buildopt-patch-bundle/v1`;
3. verify the configured Ed25519 trust root over exact JCS bytes;
4. verify expiration, repository, `actionId`, base revision/tree,
   `sourceStateDigest`, validation references, and bundle digest;
5. verify every replacement blob's exact size and SHA-256;
6. normalize no source bytes and accept only the two declared recipe/version
   combinations;
7. validate every target and blob path segment without following links.

An absolute, empty, NUL, repeated-separator, trailing-separator, `..`, or
`.git` path is invalid. A symlink at the target or any parent, a Git submodule
(`160000` gitlink), a nested repository boundary, or a resolved path outside
the repository is rejected. The patcher does not replace a symlink with a
regular file.

## Staged application

Application occurs in a private temporary worktree detached at the exact
`baseRevision`. The patcher verifies the base tree and source-state digest,
then processes operations in ascending contiguous order:

- `MODIFY` requires a regular mode-`100644` file whose exact preimage digest
  matches.
- `ADD` requires that no filesystem or index entry exists.
- replacement bytes are copied to a private temporary file, re-hashed, set to
  mode `100644`, and atomically moved inside the staging worktree;
- every resulting file must match its postimage digest.

Any failure discards the staging worktree and leaves the customer's checkout,
index, branches, and remotes unchanged. There is no three-way merge,
auto-rebase, fallback encoding, or partial apply.

Validation runs in the isolated candidate/control pipeline before persistence.
When applicable, the exact artifact checks and
`FULL_RELEVANT_VALIDATION` result in the signed bundle must still be current.

## Idempotency and delivery

The idempotency identity is `(repository, actionId, bundleDigest)`.
Persistence may create only a new `buildopt/<actionId>` branch and a draft pull
request. It never writes the default branch, modifies an unrelated/existing
branch, force-pushes, auto-merges, or marks a pull request ready.

An exact repeat returns the existing branch/PR. If the branch exists without a
PR, recovery verifies its commit tree and immutable action/bundle marker, then
creates only the missing draft PR. A different head, tree, marker, source
state, or bundle digest is a conflict and returns the action to `PROPOSED`;
the patcher never overwrites or rebases it.

## Executable plan

[`patch-bundle-v1.json`](./patch-bundle-v1.json) fixes the ordered phases and
15 golden/negative/idempotency/recovery cases consumed by
`SPK-004`.

Run the Phase 0 specification check:

```bash
./dev/check-patch-bundle-spec
```

This validates the phase/case completeness and composes the existing strict
bundle schema and cryptographic corpus. It does not claim the Java applier
complete; `SPK-004` must execute these cases against real Git worktrees.
