# BuildOpt Patcher

Customer-side Java artifact that validates and materializes a `PatchBundle` without executing bundle content.

`SPK-004` established the bounded Java 17 parser/applier and `C4-001` adds the production canonical signer, all without external
libraries. It:

- rejects malformed UTF-8, duplicate JSON keys, unknown fields, unsupported
  recipes, expired/untrusted JCS/Ed25519 envelopes, and inexact blobs;
- binds repository, action, full base revision/tree, recursive source-state
  inventory, ordered operations, preimages, postimages, and mode `100644`;
- validates every path segment without following symlinks or entering
  submodules/nested repositories;
- applies only exact `ADD`/`MODIFY` replacements in a private detached
  no-checkout worktree, bypasses checkout hooks/content filters, and creates
  an absent `buildopt/<actionId>` ref atomically;
- reuses an exact immutable commit/draft delivery, recovers a branch without a
  PR, and returns conflicts to `PROPOSED` without rebase or force-push.

The `DraftPullRequests` interface is owned by the caller. The spike uses an
in-memory adapter to prove recovery without credentials or remote mutation;
workflow orchestration remains in Go.

Run `./dev/check-patch-bundle-applier`.
