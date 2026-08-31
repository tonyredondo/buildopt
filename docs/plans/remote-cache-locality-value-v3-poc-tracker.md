# Remote Cache Locality Value v3 POC Tracker

**Status:** active — corrected five-family protocol frozen<br>
**Current block:** `RCL3-003`

| Block | Deliverable | State |
|---|---|---|
| `RCL3-001` | Fresh contract, five source-bound primary outputs, non-fail-fast correctness, controlled network and economics | `DONE` |
| `RCL3-002` | Production Edge harness and fixed-profile calibration | `DONE` |
| `RCL3-003` | Four fresh correctness builds for all five families | `TODO` |
| `RCL3-004` | Eight balanced direct/Edge pairs per eligible family | `WAITING` |
| `RCL3-005` | Twenty-build cost/payback ledger per passing family | `WAITING` |
| `RCL3-006` | Terminal controlled-mechanism and product-viability decision | `WAITING` |

V2 remains immutable. V3 does not fail fast across families and does not treat
a native output mismatch as a BuildOpt failure. It selects only primary binary
artifacts separately exposed by public build definitions. Every v3 build and
evidence row must be fresh.

The owner lab lacks a real customer remote origin, so timing uses the frozen
30-ms/100-MiB/s controlled envelope. A pass may qualify the mechanism and its
break-even within that envelope, but cannot claim real-path product viability.
