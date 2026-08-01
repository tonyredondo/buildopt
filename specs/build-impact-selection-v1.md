# Build Impact active selection v1

This contract closes `C3-005` with one active selection boundary.
`PlanSelection` accepts only parsed customer manifests and declared graphs,
recomputes their canonical digests, and reevaluates BIA-002 directly from the
current bound validation results. A caller-created promotion report is never
authority.

Selection is disabled by default. Explicit enablement plus a qualified current
sample can choose only an exact customer-manifest alternative already accepted
by the conservative graph engine. Local bypass, kill switch, digest or adapter
drift, insufficient or suspended evidence, malformed loaded state, global
changes, unknown paths/relationships, and insufficient coverage all return the
original customer entrypoints. Test Optimization-owned check IDs remain
separate and are never added to or removed by Build Impact selection.

Run:

```bash
./dev/check-build-impact-selection
```

The checker copies the repository-controlled three-project Gradle fixture into
two isolated workspaces. The full build runs `assemble`; the selected build
consumes the production plan and runs `:service-a:assemble` for a library-c
change. Both execute the Test-owned check separately. The proof requires an
identical service-a JAR and Test-owned marker, while service-b exists only in
the full build. Both builds are offline and leave the source tree unchanged.

This is bounded owner-operated POC evidence. It does not select tests, replace
the deferred soak or external validation, or claim production promotion.
