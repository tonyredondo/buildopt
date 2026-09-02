# Spring ArchitectureCheck review

## Decision requested

Decide whether this exact correction is understandable and safe enough for a
controlled trial. Start the active-review timer before reading the evidence.

## Source binding

- Repository: `spring-projects/spring-framework`
- Revision: `91eb42645e26a7ef9382b4a655bcefe5c8682fee`
- Path: `buildSrc/src/main/java/org/springframework/build/architecture/ArchitectureCheck.java`
- Preimage SHA-256: `22b20c6a01db4cc0ca93f871e8add20cbb105b8c20d9e6e8b3c55d705a3efd04`
- Postimage SHA-256: `dd75bc28223141ac121f6377f7ba95832fd250739861196f60120264dd4c2d1d`

## Proposed diff

```diff
@@
  * @author Andy Wilkinson
  * @author Scott Frederick
  */
+@org.gradle.api.tasks.CacheableTask
 public abstract class ArchitectureCheck extends DefaultTask {
```

## Evidence

Every file input already declares portable relative path normalization. The
marker-only correction passed cross-root cache restoration, exact-output and
exact-revert checks. Eight balanced pairs were positive; mean optimized-native
time fell from 2,788.5 ms to 1,803 ms, saving 985.5 ms (35.34%). The correction
is digest-bound and exactly reversible.

## Owner response

Record active review seconds, clarification count, concerns, and exactly one
decision: `ACCEPT_FOR_CONTROLLED_TRIAL` or `REJECT`.
