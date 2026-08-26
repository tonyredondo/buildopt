# Neutral Gradle passthrough and bypass contract

Status: accepted POC contract for `SWL-004`.

This contract completes the neutral execution layer of the repository-committed
BuildOpt Wrapper. It proves that committing `buildoptw`/`buildoptw.bat` does not
change what Gradle receives or how the build process behaves. It does **not**
enable cache integration, observation, learning or an optimization profile.

The machine-readable companion is
[`poc-sticky-wrapper-passthrough-v1.json`](./poc-sticky-wrapper-passthrough-v1.json).

## Routing

Ordinary arguments belong to the repository's existing Gradle Wrapper:

```text
./buildoptw test --tests example.FastTest
              │
              └── verified buildopt run -- ./gradlew test --tests example.FastTest
```

The first argument alone selects a different route:

- `--buildopt` reserves wrapper management;
- `--gradle` is removed and forces every remaining argument to Gradle, including
  a literal leading `--buildopt`;
- no prefix means Gradle.

Neither implementation joins arguments into a command string. The verified
BuildOpt binary receives the Gradle Wrapper path and each Gradle argument as a
separate process argument.

## Process equivalence

The normal path delegates to the existing `buildopt run --` launcher. It
therefore preserves the caller's working directory and standard streams,
inherits ordinary environment variables, removes BuildOpt-private variables,
places the Gradle child in the platform's isolated process boundary, forwards
`SIGINT`/`SIGTERM` or the Windows cancellation event to its descendants, waits
for cleanup and returns the Gradle result.

The repository Gradle Wrapper remains authoritative. A missing wrapper returns
127; a present but non-executable POSIX wrapper returns 126.

## Pre-bootstrap bypass

`BUILDOPT_BYPASS=1` is evaluated before `.buildopt/wrapper.properties` is
opened, before the user cache is inspected and before network or product state
can start. The wrapper removes all BuildOpt-private environment variables and
replaces itself with the repository Gradle Wrapper. Gradle's result is
authoritative.

This means bypass still works when the committed BuildOpt configuration is
missing or invalid. Values other than the exact string `1` are not bypass.

## Bootstrap failure

For an ordinary Gradle invocation, a configuration, platform, network, archive
or integrity failure emits one controlled warning and executes the original
Gradle Wrapper directly. No unverified BuildOpt binary is ever run and the
Gradle result replaces the bootstrap error.

Management commands fail closed because there is no neutral Gradle operation
that can satisfy a management request.

## Evidence boundary

The executable fixture checks difficult arguments, cwd, stdin/stdout/stderr,
ordinary and private environment variables, exit 37, explicit Gradle routing,
unknown management, pre-bootstrap bypass, bootstrap fallback, a missing Gradle
Wrapper and a signalled descendant tree. Native platform CI repeats entrypoint
parity on macOS and Windows.

Passing this contract proves only neutral wrapper equivalence. It makes no
build-time claim and grants no authority to activate optimization or Test
Optimization behavior.
