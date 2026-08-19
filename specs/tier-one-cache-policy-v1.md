# Tier 1 managed-cache policy v1

This specification materializes `A0-002` and the default-deny part of
`CACHE-007`/`COMPAT-001`. It constrains only a managed BuildOpt cache; it does
not configure a cache, grant shared access, or alter the neutral
`dev.buildopt` rendezvous.

## Activation boundary

The packaged restriction-only plugin is
`dev.buildopt.tier-one-policy`. The owner of the managed cache applies it only
after selecting that path. Applying the plugin can remove cache eligibility
but can never add it: Gradle's own `@CacheableTask`, `cacheIf`,
`doNotCacheIf`, enabled state, inputs, outputs, and implementation snapshot
remain authoritative. `A0-003` owns the first cache configuration and
`A0-006` owns authenticated policy delivery.

The separate `dev.buildopt` plugin remains neutral. Baseline, bypass,
unavailable-gateway, and ordinary customer-cache invocations therefore retain
their original behavior.

## Proven runtime adapter

The versioned adapter is enabled only on Linux AMD64 for the rows actually
executed by the repository:

- Gradle 8.14.2 on JDK 21, limited to the policy/private-L1 POC path;
- Gradle 8.14.3 on JDK 17 or 21;
- Gradle 9.6.1 on JDK 17, 21, or 25;
- Kotlin or Groovy DSL.

Every other Gradle/JDK/platform combination disables the managed cache for
that invocation. No row inherits another version's internal adapter.

The adapter inventories Gradle's complete project transform registry through
the exact 8.14.2/8.14.3/9.6.1 internal contract. The four POC transform entries
are GraalVM Native Build Tools 0.11.1
`org.graalvm.buildtools.gradle.tasks.scanner.JarAnalyzerTransform` and Kotlin
Gradle Plugin 2.2.0
`org.jetbrains.kotlin.gradle.internal.transforms.BuildToolsApiClasspathEntrySnapshotTransform`
plus
`org.jetbrains.kotlin.gradle.scripting.internal.DiscoverScriptExtensionsTransformAction`
from Kotlin Gradle Plugin 1.6.10 or 2.2.0. The 1.6.10 row is the exact
transform observed when Apache Beam's complete `classes` graph is configured;
the narrower Twitter workflow does not register it.
Each entry is bound to the exact provider, artifact name, public artifact
SHA-256, and source revision recorded in the machine-readable policy. Runtime
activation requires both the implementation name and the versioned artifact
name; Gradle's native implementation snapshot and input model remain
authoritative for its cache key, and BuildOpt's L1 remains private to the
repository/Wrapper scope.

Any other registered transform, unavailable inventory, linkage drift, or
provider failure disables the managed cache for the complete Gradle build. A
transform is never treated as a task and the implementation does not claim a
per-transform cache switch. The plugin also marks the selected tasks
Configuration-Cache-incompatible in this fallback so the next invocation must
repeat the inventory before Gradle can enable a cache; a stale serialized
decision is never accepted.

## Task allowlist

The sole v1 entry is a `JavaCompile` task created for a `SourceSet` by the core
`java` plugin from the same exact Gradle runtime. Qualification requires all
of the following:

1. the task name is a live `SourceSet.compileJavaTaskName`;
2. the runtime implementation is exactly
   `org.gradle.api.tasks.compile.JavaCompile_Decorated` with
   `JavaCompile` as its direct base;
3. the instance either has exactly one Gradle
   `org.gradle.api.internal.project.taskfactory.IncrementalTaskAction`, or the
   exact Error Prone 4.2.0 or 4.3.0 augmentation recorded in the machine-readable
   policy: one named wrapper before that built-in action plus the exact
   compiler and JVM argument-provider classes loaded from the versioned
   plugin artifact.

The Error Prone exception covers the two versions structurally observed in the
Mockito and Apache Beam POC subjects. Each row is bound to plugin id, version,
action order and display name, both provider classes, public artifact SHA-256,
and source revision. Other versions and renamed artifacts remain denied;
copying the action display name without the exact providers also remains
denied. Gradle's implementation snapshot, declared inputs, outputs, and native
cache key remain authoritative.

Every other task receives a named `doNotCacheIf`. This includes custom
`@CacheableTask` types, `Test` with no grant, a source-set task whose action
contract was modified, and a built-in registered outside its provider
contract. A later action mutation is checked when Gradle evaluates cache
eligibility, not only when the task is first configured.

## Executable evidence

[`tier-one-cache-policy-v1.json`](./tier-one-cache-policy-v1.json) is the
machine-readable closed allowlist. Run:

```bash
./dev/check-tier-one-policy
```

The checker validates the exact document and packaged plugin marker, then
executes both DSL fixtures on all twelve proven Gradle/JDK/DSL rows with isolated
TestKit homes and Configuration Cache. It proves source-set `compileJava` and
`compileTestJava` replay from cache while a custom cacheable task executes
again. All six Gradle 9.6.1 rows also execute `Test` twice and prove it cannot
replay without a grant. Gradle 8.14.3 retains strict warning failure instead
of suppressing its framework-autoload deprecation in the empty-test fixture.
Every row also proves that an added action using the allowlisted Error Prone
display name but lacking its exact providers still rejects the built-in, and
that one unknown artifact transform disables the otherwise allowed compile
task and intentionally prevents Configuration Cache reuse for the fail-closed
build.
The Mockito preflight separately proves the allowlisted GraalVM transform with
the real provider before any performance samples are accepted. The Apache Beam
preflight likewise inventories the real Kotlin 1.6.10 script-extension
transform before central-cache publication; neighboring and renamed Kotlin
artifacts remain denied by the packaged policy test.
The 8.14.2/JDK 21 rows exist solely to validate the public-repository POC path
discovered through Mockito; they do not promote that runtime into the broader
capability matrix.

`A0-G01` composes this default-deny matrix with the separate HTTP/backend fault
checker. `A0-G08` strengthens the `Test` branch with a dedicated no-grant
reason and proves root, actual `buildSrc`, and included-plugin isolation
against a usable authenticated remote cache. Positive signed
`TestCacheGrant` activation remains later Test Optimization integration work.
