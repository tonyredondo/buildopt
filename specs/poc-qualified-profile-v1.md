# Qualified POC profile

This contract turns the mechanisms that survived the BuildOpt performance
roadmap into one reviewable POC invocation. It does not promote them to a
production policy or infer repository scope automatically.

The repository commits `buildopt-qualified-profile.json` and invokes:

```text
buildopt poc --changes-file PATH
```

The configuration binds the repository, pipeline class, Build Impact manifest,
declared graph, generated discovery state, Gradle execution options and exact
mechanism set. Only native Gradle cache, Build Impact and the standard `Jar`
adapter are part of the qualified profile. The adapter is enabled only when the
repository-owned impact alternative is selected.

Before Gradle starts, the CLI emits one
`buildopt.poc/qualified-profile-plan/v1` JSON object containing the selected or
full-graph mode, reason, entrypoints, fallback entrypoints, affected-project
set, omitted-project count, required output globs, preserved Test-owned check
IDs, enabled adapters and disabled mechanisms. This makes the POC decision
reviewable without treating it as production authorization.

Unknown or global changes, incomplete or drifted discovery, local bypass and
any other unqualified path retain the manifest's original entrypoints. The
standard `Jar` adapter is not active on that fallback path. `BUILDOPT_BYPASS=1`
therefore remains an immediate native/full-graph rollback.

The command deliberately masks ambient BuildOpt Safe Cache, Runtime Tuning,
Hot State, standard `Copy`, Shared/Edge and session integration settings. A
configuration that attempts to enable any of those mechanisms is rejected.
They can re-enter only through a separate preregistered experiment that beats
optimized native Gradle under the unchanged value and equivalence gates.

Run `./dev/check-poc-qualified-profile` for the executable contract.
