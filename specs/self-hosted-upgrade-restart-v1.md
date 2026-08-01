# Self-hosted upgrade and restart v1

This contract closes `A2-003`. `dev/manage-self-hosted upgrade` takes a private nonblocking lock, then
revalidates the current signed installation, then delegates immutable version
publication and atomic selection to `dev/manage-deployment`. The running
process keeps its already-open old release. The generated unit and
self-hosted manifest are atomically rewritten to the new verified release for
the next explicit supervisor restart.

If composing the new unit or manifest fails after selection, the manager
selects the prior verified version again and restores both prior descriptor
files. Any unrecoverable composition remains fail-closed because `status`
rejects disagreement between the signed deployment and service manifest.

The configuration and persistent data root never change during upgrade. The
executable fixture places real bytes into a pending cache attempt, observes a
safe `404`, selects the new version while the old server remains ready, and
observes the same `404`. After graceful shutdown and startup of the packaged
new version, readiness must return to `200` and the same pending key must still
be `404`; partial bytes never become an HTTP hit.

Run:

```bash
./dev/check-self-hosted-upgrade-restart
```

Manual restore and revocation continuity remain `A2-004`.
