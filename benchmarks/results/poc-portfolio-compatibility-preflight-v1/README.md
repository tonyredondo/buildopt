# Portfolio Compatibility Preflight Result

This proof-of-concept result validates a cheap, fail-closed decision between
the ordinary customer build and any additional native observation used only
for portfolio evaluation.

On the frozen Micronaut `assemble` subject, the ordinary optimized-native build
completed in **526.089 seconds**. The preflight then detected exact drift in
the Gradle Wrapper and output-contract bindings and returned
`NATIVE_RETAINED / PORTFOLIO_CONTEXT_DRIFT`.

The runner proved that no independent native workspace was cloned, no second
Gradle workflow was started and no timing pair was recorded. This avoided one
incompatible measurement-only native observation. The 526.089-second customer
build was required and is not reported as a saving; no performance percentage
is claimed.

The next experiment is preregistered on direct first-parent Micronaut revisions
`8e418f75dd7a3aa66bc94786101bc8a2005cb5e2` and
`4dc4299f8dd0faccc0c45c2f83a223b456dc0731`. They share the repository,
`assemble` workflow and Wrapper tree digest. Runtime preflight must still prove
an identical output contract before an independent observation or timing is
authorized.

Validate the compact evidence and implementation with:

```bash
./dev/check-portfolio-compatibility-preflight
```

The result adds no repository-name behavior, weakens no exact-output gate and
does not authorize production, soak testing, design-partner work or Test
Optimization.
