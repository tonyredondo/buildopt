# Neutral measurement envelope

`neutral-envelope` is the dependency-free `WS-009` measurement helper. It
executes one native or wrapper arm under an external monotonic clock, validates
the required deliverable, emits a private raw observation, combines complete
alternating pairs into a strict report, and validates existing reports.

It is a development/evidence binary, not a customer launcher. The executable
contract, boundaries, qualification rules, and commands are defined in
[`specs/walking-skeleton-overhead-v1.md`](../../specs/walking-skeleton-overhead-v1.md).
