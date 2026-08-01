# Build Impact gate v1

This contract closes `C3-G01` and the owner-operated `MVP-C3` proof only
when all current-tree C3-001..005 checkers pass in one source-preserving
invocation.

The composite proves that only a strict repository/pipeline customer manifest
and its canonical current declared graph can authorize an omission. Global or
unknown work always uses the original full graph; Test Optimization work stays
separate; shadow/control divergence suspends; and active selection cannot occur
before the unchanged BIA-002 gate qualifies the exact current binding.

Run:

```bash
./dev/check-build-impact-gate
```

The checked-in operational observations are still honestly `INCONCLUSIVE`.
The active path is proven only by the deterministic threshold corpus and two
isolated offline builds of the owner-controlled synthetic repository, with
byte-identical required artifact and Test-owned marker.

This closes implementation and bounded owner validation for optional MVP-C3.
It does not derive authority from historical output, select tests, run the
deferred eight-hour soak, substitute for external validation, or claim
production promotion/readiness.
