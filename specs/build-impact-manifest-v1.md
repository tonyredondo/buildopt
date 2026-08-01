# Build Impact manifest v1

This contract materializes `C3-001` and accepted decision `BIA-001`. A manifest
is customer-owned only when it is a regular, non-symlinked file committed in
the selected repository and explicitly bound to that repository, its pipeline
class, and a positive manifest version.

The manifest lists the complete original entrypoints and only complete
alternative entrypoint sets written by the repository owner. It also lists
required artifacts, explicitly owned checks, and global change paths. The
parser accepts no command lines, arguments, absolute/traversing paths, inferred
alternatives, unknown fields, or unknown-change policy other than `FULL_GRAPH`.

Build Optimization owns artifacts. Checks name either Build Optimization or
Test Optimization as their sole owner; this manifest never grants Build
Optimization permission to select or omit tests. A valid manifest is necessary
but insufficient for omission: the declared Gradle graph, shadow comparison,
and unchanged `BIA-002` promotion gate remain separate mandatory blocks.

Run:

```bash
./dev/check-build-impact-manifest
```

The checker validates the machine-readable contract, loads the checked-in
repository fixture through the production parser, and executes positive and
negative parser tests. It does not run a customer build or claim promotion.
