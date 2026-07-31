# PatchBundle signing v1

This POC contract closes `C4-001`. The control-plane signer and customer-side
verifier share the closed PatchBundle v1 digest and Ed25519 payload definition;
the signer does not become customer-side authority.

The production signer accepts only a complete unsigned top-level manifest with
neither `bundleDigest` nor `signature`. It rejects malformed/duplicate-key JSON,
unknown or incomplete top-level fields, invalid identity constants, oversized
input, invalid key IDs, and non-Ed25519 keys. It computes the digest over JCS of
the manifest plus sorted blob inventory, binds that digest to contract version
and key ID, and returns immutable canonical bytes.

The customer-side verifier independently recalculates the digest and validates
the pinned public key, signature, repository/action binding, validity window,
blob bytes, recipes, operations, paths, and delivery restrictions. Every real
Git acceptance case now obtains its bundle from the production signer before
verification and application.

Run:

```bash
./dev/check-patch-bundle-signing
```

The checker proves deterministic signatures, defensive returned bytes,
incomplete-manifest rejection, Java 17 packaging, shared JCS/Ed25519 vectors,
and all 15 existing verifier/applier cases.
