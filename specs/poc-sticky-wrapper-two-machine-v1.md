# Sticky Wrapper two-machine proof

This contract is the installed-customer proof for `SWL-014`. It answers one
narrow question: can a repository commit one `./buildoptw` command and use a
verified BuildOpt archive plus a central Gradle HTTP cache from a second clean
machine? It is a functional POC experiment, not a performance claim and not a
production deployment design.

## Customer path

The fixture commits `buildoptw`, `buildoptw.bat`, `.buildopt/wrapper.properties`
and `.buildopt/config.toml`. The producer and consumer then invoke only:

```text
./buildoptw --no-daemon --no-configuration-cache --console=plain assemble
```

The harness preloads the verified package into the wrapper cache using the
archive SHA-256. This models a release download without depending on a public
release created solely for the test. The real wrapper still validates the
archive manifest and refuses a corrupt cache entry.

## What is proved

1. The producer uses a separate workspace and central write capability to
   publish two cache objects. A read in the same pending generation records
   `OWNER_COMMIT_REQUIRED`; pending objects are not made visible early.
2. The owner commits the pending object set and restarts the HTTPS service with
   the same state directory.
3. The clean consumer uses the committed wrapper and read-only credential. It
   restores at least one task from the central cache and produces the exact
   producer JAR digest.
4. With the service stopped and local build/cache outputs removed, the same
   wrapper falls back to native Gradle and produces the exact digest again.
5. Credentials stay in the launcher boundary and are absent from Gradle output
   and harness logs. The existing connection/bootstrap suites cover live token
   revocation, wrapper-cache corruption and committed configuration drift.

The Build Impact profile is intentionally not required here. The two-commit
fixture has no qualifying recurrence history, so `buildopt optimize` remains
`NATIVE_RETAINED`; treating that as a failure would conflate profile learning
with the independent cache/transport proof.

## Acceptance and limits

The executable checker requires a verified bootstrap, a consumer cache hit,
identical producer/consumer/outage outputs, no credential leakage and native
success during outage. It records phase durations for the next fair wall-time
experiment but sets `wallTimeClaim=false`. No soak, design partner, HA/RBAC,
production authority or Test Optimization work is required.

Run:

```bash
./dev/check-sticky-wrapper-two-machine
```
