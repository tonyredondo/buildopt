# Spring Messaging Fresh-Control Correctness v1

`SPRING_MESSAGING_FRESH_CONTROL_CORRECTNESS_V1` corrects SMCC's comparator,
not its exactness requirement. SMCC proved that two unrelated AspectJ test
classes vary across separate native builds while both fresh native controls
and the candidate inside one sequence have the same complete 14,406-file
digest. An older native digest is therefore not a valid current control.

This route starts from empty BuildOpt state and permits at most three ordinary
requests: `OPTIMIZED_NATIVE`, `OPTIMIZED_NATIVE`, then
`INCREMENTAL_CANDIDATE`. All 14,406 required outputs must be present and the
complete digest must be identical across both fresh controls and the candidate.
No path may be excluded, normalized or ignored. Public-source writes, state
forgery, timing samples and speedup claims are forbidden.

A pass authorizes only a separately frozen paired-value contract. Any mode,
history, count, digest or failure mismatch stops the route.
