# Micronaut Python VFS bytecode compile review

## Decision requested

Decide whether this exact correction is understandable and safe enough for a
controlled trial. Start the active-review timer before reading the evidence.

## Source binding

- Repository: `micronaut-projects/micronaut-core`
- Revision: `428ddeb3ad2acdabef2027cc06af3bf46865956a`
- Path: `buildSrc/src/main/groovy/io/micronaut/build/internal/python/PythonVfsBytecodeCompile.java`
- Preimage SHA-256: `2751e8434d1cdd6be5052abe4d3205662e4669d78fcbc9168e3c73a261c0adca`
- Postimage SHA-256: `2cfc774746985253205acb2f360df84c70a9f4ba985d2ef1f32e8ecbcfa464ed`

## Proposed diff

```diff
@@
  /**
   * Copies a GraalPy VFS resource tree and adds checked-hash Python bytecode caches to its file list.
   */
+@org.gradle.api.tasks.CacheableTask
 public abstract class PythonVfsBytecodeCompile extends DefaultTask {
@@
+    @org.gradle.api.tasks.PathSensitive(org.gradle.api.tasks.PathSensitivity.RELATIVE)
     @InputDirectory
     public abstract DirectoryProperty getSourceDirectory();
```

## Evidence

The relative-path semantic correction passed relocation, mutation,
cross-root-cache and exact-output checks. Eight balanced pairs were positive;
mean optimized-native time fell from 10,909.25 ms to 3,988.125 ms, saving
6,921.125 ms (63.44%). The correction is digest-bound and exactly reversible.

## Owner response

Record active review seconds, clarification count, concerns, and exactly one
decision: `ACCEPT_FOR_CONTROLLED_TRIAL` or `REJECT`.
