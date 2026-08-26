# Sticky-wrapper Gradle HTTP cache POC

## Decision

The committed wrapper can automatically use the existing central Gradle HTTP
cache connection when its private owner-issued credential is valid. Gradle
continues to use its native cache key, object format and `FROM-CACHE` behavior;
BuildOpt supplies only the invocation-local verifying gateway and the
credential-bound namespace. The default sticky-wrapper connection is
read-only. A trusted producer remains an explicit owner-operated operation and
is never created merely by running a developer checkout or pull request.

This is a functional POC contract. It proves cross-machine cache reuse and
safe fallback, not a wall-time advantage or production service readiness.

## Request path

```text
committed buildoptw
        |
        | verify/cache BuildOpt distribution and read private token
        v
BuildOpt launcher
        |
        | fresh loopback Basic credential
        v
invocation-local verifying gateway
        |
        | HTTPS bearer, project namespace and cache capability
        v
owner-operated central Gradle object plane
```

The wrapper never writes the central token to the repository or passes it to
Gradle. A missing, expired, revoked or mismatched credential retains native
Gradle without consuming a central object. Redirects and unverified responses
are rejected by the existing gateway contract.

## Central policy boundary

The managed local L1 remains protected by BuildOpt's fail-closed Tier 1 policy.
For the central HTTP cache, Gradle itself already verifies the cache key and
the complete downloaded entry before making it usable. The sticky central path
therefore selects `GRADLE_NATIVE` policy and does not silently deny a
customer's arbitrary `@CacheableTask` implementation. This is a scope decision,
not a trust shortcut: the gateway still enforces the owner-issued namespace,
read-only capability and response checksum before returning bytes to Gradle.

`buildopt gradle` also adds `--build-cache` when the user did not specify a
cache flag. `--no-build-cache` remains an explicit opt-out and prevents central
cache setup. All other Gradle arguments, streams, working directory, signals
and exit status retain the normal wrapper contract.

## Executable proof

Run:

```bash
./dev/check-sticky-wrapper-cache
```

The integration uses the existing eight-task Gradle fixture and performs this
lifecycle:

1. an explicit write-only producer publishes opaque objects to one pending
   attempt;
2. the existing signed commit protocol makes the complete set readable;
3. a clean checkout configured only with sticky-wrapper read capabilities runs
   through `buildopt run -- <gradlew>` and obtains at least eight `FROM-CACHE`
   outcomes;
4. producer and consumer output trees have identical SHA-256 values and the
   read-only consumer sends no PUT;
5. a corrupted response is normalized to a cache miss before any bytes reach
   Gradle; and
6. stopping the central server causes a successful native rebuild with no
   `FROM-CACHE` outcomes and the same output SHA-256.

The race-enabled test also checks that swapping the gateway's central HTTP
transport while Gradle starts cannot race with concurrent cache requests.

## POC boundary and next block

This block proves that the sticky wrapper can reach the central Gradle data
plane without changing Gradle's cache protocol, and that ordinary native
execution remains the recovery path. It deliberately makes no performance
claim; timing must compare this path with an equal native remote-cache
opportunity in a separate experiment. Decision/state learning and trusted
writer automation remain deferred to later POC blocks.
