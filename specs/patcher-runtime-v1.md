# Exact Java patcher runtime v1

This POC contract closes `C4-002` and the patcher conformance gate `C4-G07` by
promoting the existing Java 17 main-source verifier/applier from its Phase 0
capability proof into the C4 runtime baseline.

The public API accepts only the verifier's opaque `VerifiedPatchBundle`. It
materializes replacements in a private detached no-checkout Git worktree,
checks the exact base revision/tree/source-state digest, refuses symlink and
submodule boundaries, requires exact mode and preimage, applies ordered
`ADD|MODIFY` replacement blobs, verifies every postimage and the complete
staged path set, and only then creates a deterministic commit.

The only ref it can create is an absent `buildopt/<actionId>` head using atomic
compare-and-set. An exact retry reuses the immutable commit and matching draft;
a differing ref or PR returns `PROPOSED`. It never runs bundle content, hooks,
content filters, fuzzy patches, deletes, rebases, force pushes, writes the
default branch, marks a PR ready, or merges.

Run:

```bash
./dev/check-patcher-runtime
```

The checker composes schema/spec/signing conformance, executes all 15 cases in
real temporary Git repositories, checks Java 17 class files, and verifies that
the customer checkout and remote remain unchanged.
