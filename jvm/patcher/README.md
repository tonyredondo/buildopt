# BuildOpt Patcher

The closed registry includes the two reviewed native cacheability corrections
qualified by RNPP. They bind exact Java source paths, preimage/postimage
SHA-256 values and insertion offsets, require a reviewed adapter, and use the
existing signed exact-revert path. Run
`./dev/check-reviewed-native-patch-delivery` for the controlled delivery proof.

Customer-side Java artifact that validates and materializes a `PatchBundle` without executing bundle content.

`SPK-004` established the bounded Java 17 parser/applier, `C4-001` added the
production canonical signer, `C4-002` promoted the exact verifier/applier as
the C4 runtime baseline, `PA-F2-001` centralized the exact versioned recipe
allowlist, `PA-F2-002` added Groovy archive reproducibility and root
build-cache properties recipes, and `C4-009` added fail-closed post-merge
classification plus exact signed registry-authorized revert bundles. The implementation
uses no external libraries. It:

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

The `DraftPullRequests` interface and workflow orchestration are caller-owned.
The spike uses an in-memory adapter to prove recovery without credentials or
remote mutation.

Post-merge evidence never claims percentage rollout: missing or inexact controls remain contextual, while a proven regression may create only a new signed inverse bundle and draft revert PR.

Run `./dev/check-patch-bundle-applier`.
