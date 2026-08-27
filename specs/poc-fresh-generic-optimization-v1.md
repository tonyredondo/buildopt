# Fresh generic optimization POC v1

Status: active preregistered execution contract.

This contract resets BuildOpt performance evidence to zero. It tests the
current implementation only, starting with complete generic evidence producers
and ending—if the intermediate gates pass—with a fresh chronological campaign
against cache-equivalent optimized native Gradle.

The normative execution order, inputs, outputs, statuses and gates are defined
in the [active tracker](../docs/plans/fresh-generic-optimization-poc-tracker.md)
and the adjacent machine contract. Historical results remain immutable for
audit, but have `terminalAuthority=false` and `experimentInputAllowed=false`.

The experiment must never interpret unavailable producer input as absence of
optimization opportunity. Hosted CI checks schemas and reproducibility; it
does not decide wall-time thresholds. Performance evidence is generated on the
declared controlled runner from new checkouts and empty BuildOpt state.
