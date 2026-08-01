# macOS and Windows compatibility v1

The supported cross-platform POC surface is deliberately narrower than the
Linux deployment surface. macOS and Windows ship the per-invocation
`buildopt` launcher and automatic `buildopt-impact` generator. Both retain the
authenticated loopback gateway, Gradle plugin handshake, exact child exit
status, conservative Build Impact fallbacks, and package lifecycle.

macOS isolates child trees with POSIX process groups. Windows assigns every
child to a Job Object with kill-on-close, so a cancelled or forcibly terminated
launcher cannot orphan descendants. Linux continues using its private Unix
socket and peer UID check; macOS and Windows use a random loopback TCP endpoint
plus the same fresh 256-bit event credential. The Java plugin validates the
canonical loopback endpoint before connecting.

Persistent managed gateway/L1/bootstrap services, server/Edge storage, the
Unix-only Rust hermetic helper, and native background-service installation are
not silently emulated. They fail explicitly and remain Linux-only. The native
packages therefore install only the two supported CLIs and keep a strict
receipt for idempotent upgrade and exact uninstall.

The native GitHub workflow runs on pinned `macos-15` and `windows-2025`
labels. Each runner builds the binaries and Java plugin, packages and installs
twice, executes a real Gradle handshake, checks generated Build Impact state,
proves process-tree cleanup, uninstalls, and uploads the package. This bounded
gate does not run the deferred soak and implements no Test Optimization logic.

Run the Linux-side inventory and cross-compilation gate with:

```bash
./dev/check-platform-compatibility
```
