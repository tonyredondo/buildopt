# Spring Messaging Candidate Correctness v1 evidence

The exact empty-state sequence produced the required modes:
`OPTIMIZED_NATIVE`, `OPTIMIZED_NATIVE`, then `INCREMENTAL_CANDIDATE`. All three
requests exited zero, observed 11 compatible matches, produced 14,406 outputs
with the same fresh digest and recorded zero product-attributable failures.

The route nevertheless stops. The fresh sequence digest
`b2939eeb30a9c3f35e6c338c02f1ee69a5ce6aca008b88689a5b048c03ad87f9`
does not equal SMGC's frozen expected digest
`7de23e14d9987f0cc8661cb7de81b06f4df15cd3039ac8e947ace6f33904779f`.
The contract is not loosened after observation. The single candidate is not a
correctness pass, and no timing sample, speedup claim or paired-value successor
is authorized.

Run `./dev/check-spring-messaging-candidate-correctness` to reconstruct the raw
hashes, modes, history admission, output mismatch, failures and terminal stop.
