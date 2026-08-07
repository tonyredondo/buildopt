# Gradle Optimization Plugin

Init/settings/project plugin and adapters built on public Gradle APIs.

`WS-003` packages the project plugin as ID `dev.buildopt`. A
Configuration-Cache-safe shared build service reads invocation-only launcher
context, connects to its Unix socket, sends the v1 `ProducerHello`, validates
the matching acknowledgement, and closes the channel. Registering the service
with Gradle's build-events listener registry realizes it once even when every
task is up-to-date or Configuration Cache is reused.

The plugin logs and disables its handshake when context is incomplete, the
receiver is missing, or the acknowledgement is invalid. These failures never
fail the baseline Gradle build. The plugin does not modify task inputs,
outputs, dependencies, cache policy, or execution, and it does not emit later
task-event payloads.

`A0-002` additionally packages `dev.buildopt.tier-one-policy` as a separate,
restriction-only plugin. A managed-cache owner may apply it to install named
`doNotCacheIf` guards. The current exact allowlist contains only core
source-set `JavaCompile` tasks with the expected decorated implementation and
single Gradle action. `Test`, custom tasks, modified instances, unsupported
runtimes, unavailable transform inventory, and every registered artifact
transform fail closed. Invocation-wide fallback disables Gradle's build cache
and prevents Configuration Cache reuse so the transform inventory is checked
again next time. This plugin never configures or enables a cache, so the
neutral `dev.buildopt` path above remains behavior-preserving.

`WS-004` adds the authenticated local rendezvous without placing secrets in
Gradle configuration. On every service realization, including Configuration
Cache reuse, the plugin reads fresh invocation context, authenticates an HTTP
readiness probe against the loopback gateway, verifies its connection
generation, and writes the separate event-channel authentication preface before
`ProducerHello`. Incomplete or rejected context remains fail-open for the
baseline build. The neutral plugin still changes no cache behavior.

`A0-003` packages `dev.buildopt.managed-l1` as a settings plugin. An init
script applies it in `beforeSettings`, and the invoker supplies Gradle's public
`--build-cache` flag. Complete launcher context configures one native
`DirectoryBuildCache` under the opaque scope/security-generation directory,
enables local load/store, and delegates seven-day retention to Gradle's native
cleanup. The plugin also applies `dev.buildopt.tier-one-policy` before each
project. Malformed context disables the managed cache; the distinct
`DISABLED_L2_WRITER` mode disables local load/store.

The Build Optimization POC also contains explicit, version-bound adapters for
unmodified standard `Jar` and `Copy` tasks that Gradle does not make cacheable
by default. The `Copy` adapter is restricted to Gradle 9.6.1, the exact
`Copy_Decorated -> Copy` hierarchy, one native `StandardTaskAction`, and a
destination strictly below the owning project's build directory. `Sync`,
custom actions, custom subclasses, external destinations, and every mismatch
retain native Gradle behavior. These adapters are experimental evidence
surfaces, not a general task-cache policy.

`A0-006` extends that settings plugin with the public `HttpBuildCache` API.
Only a complete launcher-owned authority/policy/configuration/gateway context
configures the loopback remote cache. Read-only mode sets `push=false`; an
authorized pending writer enables push while retaining the disabled native L1.
The gateway credential is local-only, and the authority, policy,
configuration, and connection generations remain Configuration Cache inputs.
Malformed context disables the remote cache.

`A0-G03` composes this client with the managed gateway lifecycle. A stable
gateway process restart retains the complete local identity and Configuration
Cache entry; complete endpoint/credential/generation rotation invalidates that
entry once. Concurrent runner slots retain distinct transient upstream
bindings, and no upstream credential is serialized for Gradle.

`A0-G08` gives every `Test` task an explicit no-grant `doNotCacheIf` guard.
The type-hierarchy rule applies independently in root, actual `buildSrc`, and
included plugin builds. A control first proves all three can store and restore
through the authenticated remote cache; the guarded path then executes all
three without a grant, reuses Configuration Cache, and makes zero remote
`GET`/`PUT` requests. Positive signed-grant activation remains a later Test
Optimization integration.

The plugin does not read Gradle's opaque cache files, parse signatures, receive
the Shared credential, delete revoked generations, or force build-cache
enablement by mutating `StartParameter` during settings evaluation. Signature,
anti-rollback, repository, expiration, and remote credential checks remain in
the launcher/gateway/Shared trust boundary.

Validate the packaged Java 17 artifact, gateway, and real Wrapper handshake
with:

```bash
./dev/run -- ./dev/check-jvm-release
./dev/check-local-gateway
./dev/check-gradle-plugin-handshake
./dev/check-tier-one-policy
./dev/check-tier-one-cache-conformance
./dev/check-test-cache-isolation
./dev/check-l1-l2-revocation
./dev/check-gateway-rotation
./dev/check-managed-l1
./dev/check-local-authority
```

Deep instrumentation and output-semantics changes do not belong in this module
without an explicit contract and gate.
