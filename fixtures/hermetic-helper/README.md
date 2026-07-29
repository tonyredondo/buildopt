# Hermetic helper spike fixture

The checker builds `hermetic-fixture-producer`, copies it into a synthetic
task-specific workspace, and creates a closed manifest with:

- one read-only input;
- disjoint writable output and temporary directories;
- one exact producer command;
- explicit deny policies for network, clock, and randomness;
- stable task-execution and producer identities.

The helper must reject that otherwise valid producer before execution because
its coverage probe is incomplete. The checker confirms that no candidate
output exists, then runs the same producer directly as the baseline and
verifies real filesystem, process-tree/native-child, network, environment,
clock, and randomness markers. Writable-input and command-escape manifests are
negative fixtures in the Rust unit suite.
