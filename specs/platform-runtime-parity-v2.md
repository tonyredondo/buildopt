# Full macOS and Windows runtime parity v2

`PLAT-F6` extends the portable per-invocation surface from PLAT-F5 to the
complete Build Optimization runtime. Native packages now include the launcher,
Build Impact, server, and Edge binaries; Windows additionally includes an SCM
service wrapper.

The persistent managed gateway, managed L1, signed local authority, and Gradle
bootstrap cache use the same state and fail-closed contracts on every supported
OS. Unix uses `flock`; Windows uses `LockFileEx`. Windows gateway control uses a
fresh 256-bit credential on a loopback-only endpoint because peer-UID Unix
sockets are unavailable. Data-plane credentials remain independent.

Shared and Edge storage accept only a proven local filesystem boundary: the
Linux allowlist, `MNT_LOCAL` plus same-device checks on macOS, and one local
Windows volume without reparse traversal. Atomic files and SQLite FULL
synchronous mode remain the durable boundary where Windows cannot flush a
directory handle.

Resource controls are reported rather than emulated. Linux may enforce cgroup
v2 profiles, macOS preserves complete process-group cancellation without a
hard resource limit, and Windows uses Job Objects. `buildopt doctor` exposes
these exact semantics as machine-readable JSON, so no platform silently claims
a stronger guarantee.

macOS ships launchd user-agent installation, while Windows ships an SCM-aware
wrapper and exact service install/remove scripts. Both service integrations
remain explicit because server and Edge require owner-provided private
configuration.

Run the cross-build and package inventory locally with:

```bash
./dev/check-platform-compatibility
```

The hosted native gate additionally executes the runtime and lifecycle on
macOS 15 ARM64 and Windows Server 2025 AMD64. Test Optimization remains a
separate product and is not implemented by this phase.
