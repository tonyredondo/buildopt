# POC breadth validation v1

This contract asks whether the bounded `CONTINUE` result survives more realistic
change shapes without pretending that every build contains avoidable work.

The native control and BuildOpt candidate use the same Gradle, JDK, Build Cache,
Configuration Cache, parallelism, daemon policy, source mutations, and required
outputs. The candidate runs only installed public entry points. Static graph
generation, manifest review, installation, and warm-up remain outside measured
build time.

## Scenarios

| Scenario | Expected behavior | Decision threshold |
|---|---|---|
| No change | Execute the authorized full graph | No more than 2% mean regression |
| Leaf source change | Select the changed service and its dependencies | At least 500 ms and 2% saved, positive paired lower bound |
| Shared source change | Select every affected consumer while omitting an independent branch | At least 500 ms and 2% saved, positive paired lower bound |
| Build-logic change | Treat the change as global and execute the full graph | No more than 2% mean regression |

Every scenario runs eight alternating pairs in Kotlin and Groovy. Required
outputs must be byte-identical and product-attributable failures must remain
zero. Accelerator percentages are not added to parity results or to each other.

Passing broadens the POC claim only to these owner-controlled synthetic change
classes. Failing a parity cell blocks broadening and identifies launcher or
fallback overhead to remove. Failing an accelerator cell retains the narrower
`POC-VALUE-004` claim. Neither outcome starts production hardening, soak, design
partner work, or Test Optimization.

Validate checked evidence with:

```bash
./dev/check-poc-breadth
```
