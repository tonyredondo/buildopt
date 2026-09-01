# Recurrent Root Candidate Correctness v1

Status: `RRCC-001` complete; `RRCC-002` is current.

This route reruns the exact public Groovy subject from FGRG from empty BuildOpt
state. It permits at most three ordinary requests and expects
`OPTIMIZED_NATIVE`, `OPTIMIZED_NATIVE`, `INCREMENTAL_CANDIDATE`. The candidate
must reproduce all 3,895 required output files byte for byte with zero product
failures. Timing and speedup claims are forbidden.
