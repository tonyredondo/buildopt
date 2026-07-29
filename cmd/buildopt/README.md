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

Run the real-binary integration suite with:

```bash
./dev/check-buildopt-cli
```

The suite covers empty, quoted, whitespace, wildcard, Unicode, newline, and literal `--` arguments; inherited cwd, environment, and standard streams; success and non-zero child statuses; usage; and launch failures.

`WS-001` does not claim process-group or signal semantics. `WS-002` owns process-group creation, `SIGINT`/`SIGTERM` forwarding, grace periods, and signal fixtures. Policy, gateway, plugin, observability export, token filtering, and optimization behavior also remain inactive.
