# PatchBundle v1 vectors

Two synthetic positive bundles cover the only private-beta recipes:

- `valid/archive-reproducibility.json`
- `valid/custom-task-contract.json`

Their replacement blobs are exact UTF-8 files under `blobs/`. The checker
verifies file size/SHA-256, operation ordering, safe unique paths, exact
pre/postimage rules, blob references, validation/delivery windows, signature
binding, and the normative bundle digest:

```text
sha256(JCS({
  "manifest": <bundle without bundleDigest, signature, or blobs>,
  "blobs": [<blobRef, blobSha256, sizeBytes sorted by blobRef>]
}))
```

`invalid/` contains declarative mutations over the archive vector for absolute,
traversal, `.git`, NUL, delete, executable-mode, command, preimage, blob,
signature, digest, and duplicate-operation failures. Schema-layer and
semantic-layer rejections stay distinct.

These vectors do not claim the Java parser/applier or Git workflow complete.
Symlink/submodule escape, staged application, idempotent branch/PR recovery,
and content non-execution remain with `SPK-004`/F0-034 before
`PATCH-BUNDLE-001` can close.
