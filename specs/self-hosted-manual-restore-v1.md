# Self-hosted manual restore v1

This contract closes `A2-004` for the owner-operated single-node POC. Restore
requires an absent installed data root, a private same-deployment offline
snapshot, a stopped Shared writer, the prior still-verifiable signed authority
as recovery metadata, and a distinct current signed authority. It does not
overwrite or delete an existing data root.

`dev/manage-self-hosted restore` verifies both authorities with the packaged
server. Repository, policy, and namespace scope must be identical, while the
policy version, revocation epoch, L1 security generation, and namespace
generation must all increase. The snapshot is copied into a private sibling
stage and atomically renamed into place. Configuration is rebound to the new
authority; the original snapshot is never mutated. Failure removes only the
new marked target and restores the old installation manifest.

The executable fixture writes pending bytes under the old namespace, stops the
server, snapshots the private tree, and simulates loss of the installed data
root. Unrotated authority and a symlink-bearing snapshot both fail before
target creation. A rotated restore passes manager status, starts the packaged
server explicitly, reaches readiness `200`, rejects the old token with `401`,
and returns `404` for the old key under a new generation token. No restored
old-generation object becomes a hit.

Run:

```bash
./dev/check-self-hosted-manual-restore
```

This is not a backup product, HA, or an RPO/RTO claim. Retaining a current
signed authority recovery input and producing offline snapshots remain manual
operator responsibilities for this POC.
