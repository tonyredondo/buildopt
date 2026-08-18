# Central Gradle cache gateway POC

## Decision

BuildOpt can now carry Gradle's native HTTP Build Cache protocol through the
existing invocation-local gateway to the owner-operated HTTPS service. Gradle
sees only a fresh loopback Basic credential. The gateway owns the upstream
bearer token and adds the exact repository namespace to every request plus the
pending attempt identity to writes.

This is a functional POC boundary, not production publication automation. A
trusted clean producer writes opaque Gradle objects into the existing pending
attempt. The owner-side harness then applies the existing signed
`CommitDecision`; objects remain unreadable before that decision. A separate
read-only consumer can retrieve only committed bytes.

## Request path

```text
Gradle HttpBuildCache
    | HTTP on 127.0.0.1, fresh Basic credential
    v
BuildOpt invocation-local verifying gateway
    | HTTPS, owner token, namespace and pending attempt
    v
buildopt-server central object plane
```

The gateway follows no redirects. It verifies the complete SHA-256 of a GET
response on private local spool storage before returning `200` to Gradle. A
remote miss, timeout or outage becomes an ordinary cache miss. A rejected PUT
is diagnostic only and cannot replace the Gradle process result.

## Executable proof

Run:

```bash
./dev/check-central-gradle-cache
```

The check uses Gradle 9.6.1 against the deterministic eight-object fixture. It
proves the following lifecycle:

1. A clean write-only producer executes all eight cacheable tasks and uploads
   at least eight exact pending objects through TLS.
2. The existing Ed25519 commit protocol makes the complete pending set visible.
3. An independent clean read-only project obtains at least eight
   `FROM-CACHE` task outcomes.
4. The producer and consumer output trees have the same SHA-256.
5. A write through the read-only gateway is rejected locally and never reaches
   the central service.
6. After the TLS server is stopped, another clean project executes all tasks
   normally, reports no remote hit and produces the same output SHA-256.

The integration also asserts that neither central bearer token appears in
Gradle output. Unit coverage fixes the upstream namespace/attempt headers and
their round trip through the persistent gateway registration.

## POC boundary and next block

This block proves protocol compatibility, containment, correctness and
fallback. It makes no wall-time claim: the later two-machine value block must
compare the installed central path against an equal native remote-cache
opportunity. The owner commit is intentionally explicit here; automatic
connection and orchestration belong to the state-sync/onboarding work rather
than this data-plane proof.

`POC-CENTRAL-STATE-SYNC-001` is next. It will add `buildopt connect` and exact
portfolio/evidence/checkpoint synchronization without making the service
mandatory for local optimization.
