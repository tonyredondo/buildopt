# PatchBundle applier fixture

`dev/check-patch-bundle-applier` generates one private Git repository per
`specs/patch-bundle-v1.json` case. The repositories contain the exact source
paths for both initial recipes and reuse the normative replacement blobs under
`contracts/jsonschema/testdata/patch-bundle.v1/blobs/`.

The Java runner creates real signed manifests at runtime, detached worktrees,
candidate commits, and action refs. Its local `DraftPullRequests` adapter
models only immutable draft-delivery state; it has no network or credentials.
Every case verifies that the customer checkout/index remains unchanged and no
private worktree leaks.
