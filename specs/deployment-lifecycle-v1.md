# Deployment lifecycle v1

This contract closes `DEPLOY-001` for the Linux AMD64 private-beta topology.
It consumes Release Bundle v1 with an externally pinned Cosign public key and
does not execute payloads until the complete signed bundle has passed
`dev/verify-release`.

## Layout and activation

`dev/manage-deployment` owns one explicit mode-`0700` deployment root.
Verified versions are retained side by side beneath `versions/`; each version
contains the original six-file signed bundle, the exact extracted payload,
payload checksums, and a trust/source manifest. No installed version is
modified in place.

One private JSON manifest selects the current and previous versions. Upgrade
and rollback publish that manifest with a same-directory atomic rename after
the target bundle and payload have been independently revalidated. A running
process continues using its already-open version; the next supervised start
uses the new selected root.

Persistent data lives in a separate marked mode-`0700` root. The deployment
manifest records its exact canonical path. Default uninstall removes only the
verified deployment root and preserves data. `--purge-data` is an explicit
separate choice and removes only the marked recorded root. Both paths refuse
to proceed while the Shared writer lock is held by `buildopt-server`.

## Executable lifecycle

The fixture creates two independently signed, reproducible bundles from one
clean source revision. It then exercises:

1. install version 1 and start the real packaged modular monolith;
2. deliver and export a session through the real packaged launcher;
3. idempotently request version 1 again, then upgrade to version 2;
4. restart version 2 against the same Shared/export data and add a session;
5. roll back to the stored, reverified version 1;
6. reject uninstall while the server owns the writer lock;
7. stop cleanly and uninstall while preserving all persistent data;
8. reinstall version 2 at the same deployment/data paths and reopen the data;
9. stop and explicitly purge both deployment and persistent data.

Every selection is checked through `status`; tampered installed payload,
wrong trust root, unsafe/unmarked roots, and invalid command shapes fail
closed. The repository working tree remains unchanged.

Run:

```bash
./dev/check-deployment-lifecycle
```

No GitHub release is published by this gate. Public publication, the first
per-pilot deployment, online revocation, and the operational fault/soak
profile remain `A1-001` and `OPS-001/A1`.
