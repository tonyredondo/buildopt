# Managed native L1 v1

This specification materializes `A0-003` and the local-cache portion of
`CACHE-002`. It configures Gradle's native `DirectoryBuildCache`; it does not
interpret that format, add a proprietary local LRU, configure L2, or claim an
authenticated revocation decision.

## Activation and ownership

The packaged settings plugin is `dev.buildopt.managed-l1`. It must be applied
from an init script in `beforeSettings`, before Gradle finalizes its user-home
cache cleanup configuration. The invoker also supplies the stable public
`--build-cache` flag. Mutating `StartParameter` during settings evaluation is
not an enablement contract because that mutation is not replayed when
Configuration Cache is reused.

The plugin reads only launcher-produced context through Gradle Providers,
configures one local `DirectoryBuildCache`, and applies
`dev.buildopt.tier-one-policy` before each project. A complete read/write
context enables local load/store. Missing or malformed context disables the
managed build cache. The separate neutral `dev.buildopt` plugin remains
behavior-preserving.

Gradle's native cache cleanup owns retention. The init-script application
configures build-cache entries unused for seven days; BuildOpt never reads the
cache's opaque files or runs a second eviction algorithm.

## Launcher lifecycle

The launcher accepts an absolute private state root plus bounded visible-ASCII
tenant, repository, trust-domain, and cache-compatibility identifiers. It
hashes those four length-prefixed UTF-8 values under the
`buildopt-managed-l1-scope-v1` domain. Raw identities do not enter the child
environment or on-disk path.

The native cache directory is:

```text
<stateRoot>/l1/scopes/<scopeDigest>/
  generation-<l1SecurityGeneration>/cache
```

Every launcher-owned directory is current-user-owned mode `0700`. A
mode-`0600` lock outside Gradle's opaque cache directory provides one
non-blocking exclusive writer for the exact scope and generation during the
complete Gradle child lifetime. A different generation has a distinct
directory and lease, so rotation occurs before Gradle starts and never mutates
an active cache. Old generations remain inaccessible but are not deleted by
this block.

Incomplete configuration, an unsafe root, a busy lease, or maintenance failure
emits one diagnostic and runs the child without managed-L1 context. Bypass
removes both launcher inputs and child context.

## Pending L2 writers

For A0/A1, an invocation authorized to write pending L2 objects receives
`DISABLED_L2_WRITER`. The launcher exposes no local directory and the settings
plugin disables local load and store while leaving the remote-cache owner free
to use Gradle's global `--build-cache` switch. This prevents an aborted pending
PUT from leaving a reusable local hit.

## Security boundary

The scope digest provides deterministic separation; it is not an authority
signature. `A0-006` supplies authenticated monotonic
`l1SecurityGeneration`, future/rollback rejection, and revocation-driven
generation rotation; physical deletion remains `A1-004`. `A0-004`/`A0-005`
own remote population and pending publication. This isolated block is composed
with those owners by `A0-G02`; it does not close that gate by itself.

## Executable evidence

Run:

```bash
./dev/check-managed-l1
```

The checker validates the exact machine contract and launcher race tests, then
executes Kotlin and Groovy fixtures on Gradle 8.14.3/9.6.1 with JDK 17/21. Each
row proves native replay in one generation, default-deny re-execution for a
custom cacheable task, one Configuration Cache invalidation on rotation, and a
hit after the new generation is warm. The golden row also proves invalid
context and L2-writer local disablement. A real launcher-to-Gradle sequence
binds the two L1 layers and verifies the opaque generated path. The independent
neutral handshake remains covered by `./dev/check-gradle-plugin-handshake`.
The cross-component L2-to-L1 revocation and aborted-writer lifecycle runs in
`./dev/check-l1-l2-revocation`.
