# POC stability validation v1

This contract determines whether the realistic breadth classifications survive
removal of control/candidate carryover. It changes measurement isolation, not
the product, fixture, tasks, outputs, sample count, runner, or decision
thresholds.

Each batch runs the control and candidate in separate digest-pinned strict
containers. The containers receive different writable workspaces,
`GRADLE_USER_HOME` directories, daemon lifecycles, and installed prefixes. One
batch runs the complete control arm first; the other runs the complete candidate
arm first. Each workload cell performs the same unmeasured warm-up as the
original breadth contract and then keeps its private daemon and Gradle cache for
eight sequential mutations. Corresponding mutation indices are paired only
after both containers finish; no writable state or daemon crosses from one arm
to the other.

The two reports must be bound to the same commit and artifacts, use opposite arm
orders, preserve all correctness guardrails, and produce the same classification
for every Kotlin/Groovy change cell. A reproduced failure is valid evidence: it
identifies a product-value limitation rather than being relabelled or hidden by a
weaker threshold.

Validate checked evidence with:

```bash
./dev/check-poc-stability
```

This remains an owner-controlled POC experiment. It does not claim universal
savings, production readiness, an eight-hour soak, external validation, or any
Test Optimization behavior.
