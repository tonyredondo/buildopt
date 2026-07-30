# Gateway restart, rotation, and slot isolation v1

Status: implemented by `A0-G03`.

This contract composes the real detached managed-gateway lifecycle with
Gradle's public `HttpBuildCache` and Configuration Cache. It proves stable
restart, complete local-identity rotation, transient upstream authority, and
concurrent runner-slot isolation.

## Process and persistence

The real `buildopt __managed-gateway` process registers one current invocation
through a bounded same-UID Unix control connection. After an idle process
exit, it rebinds the persisted loopback address and reuses the complete local
identity: endpoint, local Basic credential, and
`gatewayConnectionGeneration`. If the address cannot be rebound, all three
fields rotate together before a new invocation becomes ready.

The private mode-`0600` `gateway-state.json` contains only that local
rendezvous identity. Upstream endpoint, Bearer credential, authority digest,
policy, and namespace exist only in the live registration context and are
removed when its control connection closes.

## Configuration Cache

The golden Gradle 9.6.1/JDK 21 TestKit executes both Kotlin and Groovy
fixtures:

1. a cold build stores through the authenticated local gateway;
2. the gateway stops and rebinds the same identity;
3. the next clean build reuses Configuration Cache and restores
   `compileJava` from the remote cache;
4. endpoint, local credential, and connection generation rotate together;
5. the next build still restores the remote hit but invalidates the stale
   Configuration Cache entry exactly once; and
6. the following build reuses the rotated entry.

Neither the local gateway credentials nor an upstream-credential marker
appear in Gradle output. The Java fixture models restart/rotation while using
the real settings plugin, `HttpBuildCache`, task policy, and Configuration
Cache. The Go half exercises the real gateway processes and persistent state.

## Concurrent slots

Two real gateway processes accept simultaneous live cache registrations with
different upstream endpoints, Bearer credentials, authority digests, and
namespace responses. Each correct local credential reaches only its own
binding; a credential from the other slot returns byte-free `401`. Neither
binding is serialized into either state file.

This closes `A0-G03`. It does not prove the complete spool fault matrix
(`A0-G04`), commit fault/recovery atomicity (`A0-G05`), the overhead and
no-grant Test gates (`A0-G06`/`A0-G08`), or the wider private-beta restart
gate (`A1-G04`). `MANAGED_SHARED_CACHE` remains unavailable.

Run:

```bash
./dev/check-gateway-rotation
```
