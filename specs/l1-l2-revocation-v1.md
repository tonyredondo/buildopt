# L2-to-L1 revocation and aborted-writer lifecycle v1

Status: implemented by `A0-G02`.

This contract composes the authenticated Shared authority and pending/commit
backend with Gradle's public `HttpBuildCache` and native
`DirectoryBuildCache`. It proves that a committed L2 hit may populate L1, an
authenticated generation advance prevents that L1 entry from surviving into
the next build, and an aborted writer leaves no reusable local or remote hit.

## Committed read and revocation

The backend sequence uses a signed current read/write authority. A PUT is
visible only as pending until an exact canonical Ed25519 `CommitDecision`
makes the verified object readable. Advancing policy, revocation, L1,
gateway, and namespace generations supersedes the old authenticated route.
The old route returns a byte-free `401`; the new generation returns a
byte-free `404` for the old key.

The golden Gradle 9.6.1/JDK 21 lane executes both Kotlin and Groovy fixtures:

1. a trusted writer uses `DISABLED_L2_WRITER`, publishes only pending L2
   bytes, and has no native L1 directory;
2. after explicit commit, a read-only invocation restores `compileJava` from
   L2 and lets Gradle populate generation 50 of native L1;
3. with remote reads unavailable, the next clean build restores from L1 and
   reuses Configuration Cache;
4. authenticated revocation advances every relevant generation before the
   next build, removes the stable fixture object, and selects an empty
   generation 51 directory;
5. that next build observes a remote miss, executes `compileJava`, and does
   not reuse the old Configuration Cache entry; and
6. a subsequent build restores from the warmed generation 51 L1.

## Aborted writer

After changing the source, another trusted writer again has no local cache
directory and publishes only pending bytes. Aborting the attempt removes its
pending set. A fresh read-only generation then observes no local hit, no
remote hit, and executes `compileJava`.

The Go half of the checker exercises the real storage, current-authority HTTP
handler, canonical decision verification, transactional commit, authenticated
generation supersession, abort, and byte-free miss. The Java TestKit uses a
stateful protocol fixture for commit, revocation, and abort while exercising
the real Gradle cache implementations and plugin lifecycle.

## Boundary

This closes `A0-G02`. It does not prove gateway process restart/rotation
(`A0-G03`), the complete spool fault matrix (`A0-G04`), the full production
commit atomicity/fault/recovery gate (`A0-G05`), physical deletion of old L1
directories (`A1-004`), or the later private-beta revocation gate (`A1-G04`).
`MANAGED_SHARED_CACHE` therefore remains unavailable.

Run:

```bash
./dev/check-l1-l2-revocation
```
