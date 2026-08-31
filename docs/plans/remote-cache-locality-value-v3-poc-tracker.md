# Remote Cache Locality Value v3 POC Tracker

**Status:** stopped — controlled locality value gate failed<br>
**Current block:** none — terminal

| Block | Deliverable | State |
|---|---|---|
| `RCL3-001` | Fresh contract, five source-bound primary outputs, non-fail-fast correctness, controlled network and economics | `DONE` |
| `RCL3-002` | Production Edge harness and fixed-profile calibration | `DONE` |
| `RCL3-003` | Four fresh correctness builds for all five families | `DONE` |
| `RCL3-004` | Eight balanced direct/Edge pairs per eligible family | `DONE_STOP` |
| `RCL3-005` | Twenty-build cost/payback ledger per passing family | `NOT_RUN_FAILED_PREREQUISITE` |
| `RCL3-006` | Terminal controlled-mechanism and product-viability decision | `DONE` |

Execution preflight corrected the Micronaut task selector to
`:micronaut-core:jar` after the frozen `:core:jar` selector failed before any
producer build or evidence row. The source-bound output and every gate remain
unchanged.

The complete 24-pair result qualifies 0/3 eligible families. Groovy has 7/8
positive pairs and saves 1,238.5 ms on average, but reaches only 1.816% versus
the frozen 2% minimum. OpenTelemetry is negative on average. Spring reaches
6/8, 672.75 ms and 2.602%, but its corrected bootstrap lower 95% is
-605.125 ms. With the required 3/5 controlled-value breadth unavailable,
`RCL3-005` is not run and `RCL3-006` records the terminal stop.

V2 remains immutable. V3 does not fail fast across families and does not treat
a native output mismatch as a BuildOpt failure. It selects only primary binary
artifacts separately exposed by public build definitions. Every v3 build and
evidence row must be fresh.

The owner lab lacks a real customer remote origin, so timing uses the frozen
30-ms/100-MiB/s controlled envelope. A pass may qualify the mechanism and its
break-even within that envelope, but cannot claim real-path product viability.
