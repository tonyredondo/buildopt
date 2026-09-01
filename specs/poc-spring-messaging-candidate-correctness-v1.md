# Spring Messaging Candidate Correctness v1

`SPRING_MESSAGING_CANDIDATE_CORRECTNESS_V1` starts from empty BuildOpt state
and submits at most three ordinary `testClasses` requests for the exact Spring
Messaging subject confirmed by SMGC. The product must naturally produce
`OPTIMIZED_NATIVE`, `OPTIMIZED_NATIVE`, then `INCREMENTAL_CANDIDATE`; state
forgery and public-source writes are forbidden.

Every successful invocation must reproduce the SMGC 14,406-file required
output manifest byte for byte. The single candidate must equal both fresh
native controls, with zero product-attributable failures. The sequence is
untimed: observed durations are diagnostic only and cannot support a speedup or
value claim.

At most three Gradle starts, 30 minutes per invocation and 12 workers are
allowed. A pass authorizes only a separately frozen paired-value contract. Any
mode, output, history-admission or failure mismatch stops the route.
