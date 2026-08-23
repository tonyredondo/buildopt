# Cross-Revision Producer Volatility Portfolio Evidence

This experiment tested whether producer volatility learned from authoritative
native Micronaut builds could be applied to a later public revision without
weakening exact-output safety. The terminal result is a safe rejection, not a
performance result.

## Result

- The diagnostic learning revision compared **11,187** outputs from two
  independent native roots. Five Kotlin task-state files differed.
- Producer-atomic quarantine excluded **476** outputs from five Kotlin
  producers and left **10,711** outputs exactly transportable.
- The portfolio stored only those producer task paths and evidence digests. It
  stored no historical output bytes or reusable historical output hashes.
- The later public revision retained the same canonical repository identity
  and workflow, but both its Gradle Wrapper and output contract changed.
- The evaluation therefore returned `NATIVE_RETAINED` with reason
  `PORTFOLIO_CONTEXT_DRIFT`, named both changed bindings and stopped before all
  eight timing pairs. It records **zero performance claim**.
- The current-revision native comparison independently observed two volatile
  JAR producers among 186 outputs. This differs from both the five Kotlin
  producers learned in the fresh diagnostic pair and the earlier one-JAR
  incident. Native volatility is therefore observation-dependent; one pair is
  a safe input to a bounded portfolio, not a universal producer list.

## Decision

The portfolio mechanism is fail-closed and checkout-path independent, but
cross-revision replay value is not proven for this public window. Reusing a
portfolio after Wrapper or output-contract drift would make the evidence
non-comparable, so BuildOpt correctly retained optimized native Gradle.

The next bounded work is a compatibility preflight that makes this decision
before paying for the second native observation. A later performance
experiment must preregister a revision whose repository, workflow, Wrapper and
output-contract bindings all match. No percentage from the two native builds
in this experiment may be attributed to BuildOpt.

This remains POC evidence. It introduces no repository-name product rule,
production authority, soak/design-partner requirement or Test Optimization
behavior.

Validate the checked-in evidence with:

```bash
./dev/check-cross-revision-volatility-portfolio
```
