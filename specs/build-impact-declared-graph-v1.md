# Build Impact declared graph v1

This contract closes `C3-002` by combining the customer-owned C3-001 manifest
with a strict Gradle-declared project/entrypoint graph. The graph is bound to
the same repository, pipeline class, canonical manifest digest, and a
versioned adapter. Historical observations never add an edge or authorize an
omission.

For localized paths, the engine maps changes to declared source owners and
adds every transitive reverse dependent. It considers only complete
alternative entrypoint sets already listed by the customer. A candidate must
reach every affected project, required artifact, and Build Optimization-owned
check. Test Optimization-owned checks are carried through unchanged and no
entrypoint containing Test tasks can be selected.

The actual executable entrypoints remain the original customer entrypoints in
this block. The alternative is a shadow prediction only. Invalid or incomplete
graphs, cycles, missing references, global/build-logic changes, unknown paths,
unknown relationships, unsafe paths, or insufficient candidate coverage all
fall back to `FULL_GRAPH`.

Run:

```bash
./dev/check-build-impact-declared-graph
```

The checker composes C3-001 and C3-002, loads the checked-in manifest/graph
pair through production parsers, and executes the conservative decision
matrix. It does not satisfy `BIA-002` or activate selection.
