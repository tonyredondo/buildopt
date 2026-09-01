# Spring Messaging Paired Value v1

`SPRING_MESSAGING_PAIRED_VALUE_V1` measures the exact SMFC-qualified Spring
Messaging class against optimized native Gradle. Starting from empty BuildOpt
state, it submits 17 ordinary `testClasses` requests: one learning baseline and
eight balanced native/candidate pairs. The runner may start Gradle at most 17
times, with 30 minutes per request and 12 workers.

Every arm must reproduce all 14,406 required outputs exactly with one digest
per fresh paired sequence and zero product-attributable failures. Qualification
requires 8/8 positive pairs, at least 500 ms and 2% mean saving, a positive
paired 95% lower bound, non-regressive candidate p95 and payback within five
compatible matches.

The run is the timing experiment; its full wrapper, learning, verification and
fallback costs remain charged. Public-source writes, state forgery, output
exclusions/normalization, threshold movement, production and Test Optimization
are forbidden. Failure of any gate stops the exact class without a value claim.

The one authorized campaign passed every frozen gate: 8/8 positive pairs,
7,802.5 ms / 19.441% mean saving, positive paired confidence, improved p95,
two-match payback, 14,406 exact outputs and zero product failures. The terminal
decision is `QUALIFY_SPRING_MESSAGING_VALUE`; generic breadth and production
remain unauthorized.
