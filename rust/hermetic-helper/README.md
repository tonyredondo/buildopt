# Linux Hermetic Helper

Experimental optional Rust helper for C1.

It applies the OS boundary for a supported task-specific producer. Rust is not a source of hermeticity by itself, and this component is not required for A1/B. Its viability is decided through `SPK-003`.

`SPK-003` closes as `UNAVAILABLE`. The dependency-free Rust package performs
a real namespace/kernel probe and strictly validates a dedicated task producer
manifest. It refuses to execute the candidate because clock via vDSO,
`getrandom`, environment, cgroup delegation, and kernel-policy installation
are not completely mediated. The refusal discards the candidate, aborts
pending publication, and leaves the uninstrumented baseline authoritative.

Run `./dev/check-hermetic-helper-spike`. The helper does not claim that Rust,
namespace availability, or a whole-Gradle sandbox establishes hermeticity.
