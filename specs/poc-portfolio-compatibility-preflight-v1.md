# Portfolio Compatibility Preflight Protocol

## Purpose

This proof-of-concept block prevents an incompatible producer-volatility
portfolio from triggering an unnecessary second native Gradle build. The
preflight compares the canonical repository, requested workflow, Gradle
Wrapper and owner-reviewed output-contract bindings after the ordinary
optimized-native invocation has produced its current output inventory and
before any independent native workspace is cloned or executed.

The first ordinary invocation is not preflight overhead: it is the customer
build and the only point at which the current Gradle output contract can be
observed without guessing. The avoided work is the additional independent
native observation used solely to evaluate portability.

## Decisions

`buildopt profile native-preflight` emits one strict JSON document:

- `COMPATIBLE / PORTFOLIO_CONTEXT_COMPATIBLE` authorizes only the independent
  diagnostic observation. It does not authorize output transport, profile
  selection, timing claims or production use.
- `NATIVE_RETAINED / PORTFOLIO_CONTEXT_DRIFT` lists every changed binding and
  prevents the independent native clone, Wrapper preparation and build from
  starting.

Malformed portfolio evidence, a learned context that does not bind the
portfolio, an invalid current context or a tampered decision is an error rather
than a compatibility result.

## Rejection fixture

The frozen Micronaut evaluation from the preceding experiment keeps canonical
repository and `assemble` workflow identity but changes both Wrapper and
output-contract bindings. This block repeats only its ordinary optimized-native
invocation. The expected terminal decision is immediate native retention with
zero independent native observations and no performance claim.

## Next experiment preregistration

The previous learning revision has no first-parent descendant with the same
Wrapper: its direct child `8e418f75dd7a3aa66bc94786101bc8a2005cb5e2`
updates Gradle. The next value experiment therefore starts a new learning
generation at that revision and evaluates its direct child
`4dc4299f8dd0faccc0c45c2f83a223b456dc0731`.

Both revisions use the same `assemble` workflow and identical Wrapper tree
digest `1d1bca219166747b5f7ecbe36d877e3972dfc2047dd6e6ef041867a6f789552c`.
The child changes only `gradle/libs.versions.toml`. This is sufficient to
preregister the window, not to claim full compatibility: the current output
contract must still match at runtime before an independent observation or any
timing starts.

## Boundaries

The implementation adds no repository-name behavior, threshold change,
production authority, soak or design-partner requirement. Test Optimization
remains out of scope, historical output bytes are never reused and exact
current-revision output verification remains mandatory.
