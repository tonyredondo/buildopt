# Native-retention fast-path result

This directory closes the preregistered native-retention fast-path experiment
on the eight fallback revisions from cross-commit breadth V2. The candidate
uses generic change classification and verified economic evidence to decide as
early as possible, but it still executes the complete customer Gradle command
as the authoritative fallback.

## Result

The preregistered hypothesis is **rejected**, while the generic fast-path
behavior is proven. The contract predicted six pre-Gradle decisions and two
post-discovery decisions. The implementation produced seven and one:

| Repository | Revision | Decision | Output observer | Direct wrapper overhead |
|---|---|---|---:|---:|
| OpenTelemetry | `7fd551e2` | Pre-Gradle economic | No | 2,878 ms |
| OpenTelemetry | `2d3f0004` | Pre-Gradle economic | No | 2,386 ms |
| OpenTelemetry | `fecf3fa2` | Pre-Gradle compatibility | No | 474 ms |
| Ktor | `c237e886` | Pre-Gradle economic | No | 2,390 ms |
| Ktor | `835d7f9f` | Pre-Gradle compatibility | No | 300 ms |
| Groovy | `be211c1b` | Pre-Gradle compatibility | No | 311 ms |
| Groovy | `18ed80f5` | Pre-Gradle compatibility | No | 281 ms |
| Groovy | `1ff25776` | Post-Gradle discovery | Yes | 2,621 ms |

The seven early decisions total **9,020 ms** of directly measured BuildOpt
pre/post execution time; the maximum is **2,878 ms**. Both are below the frozen
24-second aggregate and 5-second per-observation limits. All eight native
fallbacks execute `OPTIMIZED_NATIVE`, preserve their exact required output
inventories and bytes, and report zero product failures.

The single mismatch is explainable and safe. Groovy `be211c1b` modifies five
subproject `build.gradle` files. The repository-independent
`**/build.gradle(.kts)` classifier therefore recognizes a global build-logic
change before Gradle and retains the full graph without attaching the output
observer. The preregistration had conservatively classified it as
`ORDINARY_OBSERVATIONS_PENDING`. The original expectation remains unchanged in
the specification and the machine result records the mismatch explicitly.

The selected Ktor regression control also remains intact: `eb60b722` selects
the remote qualified profile, preserves exact outputs and changes 223,523 ms
to 97,298 ms, saving **126,225 ms / 56.47%**. The fast path therefore did not
disable the positive selective path.

## Interpretation

Single control/candidate wall-time deltas for native fallbacks range widely
because Gradle itself dominates each build. They are retained in
`summary.json` as descriptive observations only. The causal acceptance basis
for this block is BuildOpt's direct pre/post execution measurement, not those
single-pair deltas.

The product conclusion is narrower than the rejected hypothesis: generic
early retention is cheap and safe for every compatibility/economic case seen
here, and one supposedly discovery-dependent build was also resolved earlier
than predicted. The remaining ownership-ambiguous Groovy change still needs
the configured Gradle model and must not be optimized away without a new
generic ownership proof.

This is bounded POC evidence. It adds no repository-specific branch,
production authority, soak or design-partner requirement, weaker output gate,
or Test Optimization behavior.

## Files and validation

- [`summary.json`](./summary.json) is the recomputable terminal result.
- [`source-v2/raw/summary.json`](./source-v2/raw/summary.json) and the adjacent
  compact subject captures preserve the frozen breadth V2 source evidence.
- The original preregistration remains
  [`specs/poc-native-retention-fast-path-v1.json`](../../../specs/poc-native-retention-fast-path-v1.json).

Validate the checked evidence without cloning the public repositories:

```bash
./dev/check-native-retention-fast-path
```

Capture a new complete frozen run with:

```bash
./dev/run-native-retention-fast-path /new/absolute/evidence/directory
```
