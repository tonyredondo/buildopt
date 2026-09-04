# Bounded opportunity evidence

This directory contains fresh source-only classification and the 16 selected
native diagnostic starts used by the WCNCP-009 breadth reconstruction. The
selection stays within the two-diagnostics-per-family and 20-start campaign
limits. Excluded runner and infrastructure attempts are named separately in
`diagnostic-selection.json` and are not product failures.

The independent checker reconstructs all ten source rows, capture bindings,
Configuration Cache problem counts, recurrent signatures, requested-workflow
reachability, source hashes and family totals. It concludes that seven families
are conclusive while GraphQL Java, Apache Groovy and Test Retry still require
controlled critical-path materiality. Therefore the current decision is
`INCOMPLETE_EXPERIMENT_INPUT`, not an insufficient-breadth stop.

No public source was modified. No candidate build or timing sample ran, and no
speedup, value or product-failure claim is present.

```bash
./dev/check-wcncp-opportunity-breadth /absolute/source-root
./dev/check-wcncp-opportunity-breadth-negatives
```
