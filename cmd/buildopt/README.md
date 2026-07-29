# `buildopt`

Go binary installed on CI runners and workstations.

Its boundary includes the launcher, neutral measurement envelope, Local Verifying Cache Gateway, and bypass to the original command.

## WS-001 passthrough

The initial executable surface is:

```bash
buildopt run -- <command> [args...]
```

The delimiter is mandatory. `buildopt` starts the command directly without a shell, so every token after `--` remains a distinct child-process argument. The child inherits the launcher's working directory, environment, standard input, standard output, and standard error. An ordinary child exit status, including a non-zero status, is returned unchanged.

The CLI uses these status conventions; once a child starts, its ordinary exit status always wins:

| Code | Meaning |
|---:|---|
| `0` | Help was printed, or the child completed successfully |
| `64` | Invalid CLI usage |
| `126` | The resolved command could not be executed |
| `127` | The command could not be found |
| `128 + signal` | The child terminated from an unhandled signal; for example, `130` for `SIGINT` and `143` for `SIGTERM` |

Run the real-binary integration suite with:

```bash
./dev/check-buildopt-cli
```

The suite covers empty, quoted, whitespace, wildcard, Unicode, newline, and literal `--` arguments; inherited cwd, environment, and standard streams; success and non-zero child statuses; usage; launch failures; process-group isolation; and signal forwarding through a child process tree.

## WS-002 process and signal contract

On the Linux x86-64 acceptance platform, the child becomes the leader of a process group distinct from `buildopt`. While that child is active, the launcher catches `SIGINT` and `SIGTERM` and forwards the same signal to the complete child group, including descendants that remain in it.

The launcher then waits for the child to finish and returns the child's ordinary exit status unchanged. It does not impose its own termination deadline or escalate to `SIGKILL`; the CI provider retains control of its grace period and final escalation. If the child does not handle the forwarded signal, the CLI returns the conventional `128 + signal` status without adding a launcher diagnostic to the child's standard error.

The integration fixture verifies an independent process group, a nested descendant, exact `SIGINT` and `SIGTERM` delivery, delayed cleanup during cancellation, preserved handled exit statuses, and conventional unhandled-signal status. Platform expansion remains deferred until its own compatibility fixtures.

Policy, gateway, plugin, observability export, token filtering, and optimization behavior remain inactive.
