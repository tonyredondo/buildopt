# Automatic Build Impact v1

`BIA-F4` adds conservative Gradle discovery without replacing the
repository-owned Build Impact policy manifest. `buildopt-impact generate`
loads that manifest, asks the configured Gradle build for its evaluated project
and dependency model, and emits two deterministic, reviewable files:

- a strict declared graph consumed by the existing `BIA-002` selection gate;
- a generated manifest binding the customer manifest, discovery snapshot,
  adapter, Gradle version, and canonical graph digests.

Only conventional artifact and test-preparation tasks are modeled
automatically. Unqualified task names retain Gradle's native selector semantics
across subprojects. Dependency declarations are captured while each project
owns its mutable configuration state, and cyclic project components share one
conservative source/dependency boundary. Repository-local included builds are
global-change paths; an included build outside the repository, a test-bearing
entrypoint, an unsupported task type, a missing project, malformed state,
generated-file drift, an unknown source path, or any failed discovery retains
the original full entrypoints. Build-owned artifacts and checks remain bound to
the repository manifest. Test Optimization-owned checks are preserved
separately and are never selected, omitted, reordered, or otherwise optimized
here.

Generate and review state with:

```bash
buildopt-impact generate \
  --repository . \
  --manifest buildopt-impact-manifest.json \
  --repository-id owner/repository \
  --pipeline-class pull-request \
  --graph buildopt-impact-graph.generated.json \
  --generated-manifest buildopt-impact.generated.json
```

CI uses the same command with `check`; it regenerates in isolation and fails if
either repository-visible file differs. Active selection still requires exact
current digests, explicit enablement, and independently qualified `BIA-002`
evidence.

Run the bounded proof with:

```bash
./dev/check-build-impact-automatic
```
